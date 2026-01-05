package database

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/containerd/containerd/metadata/boltutil"
	bolt "go.etcd.io/bbolt"
)

const (
	// Default path for containerd's core metadata database
	DefaultContainerdMetaDBPath = "/var/lib/containerd/io.containerd.metadata.v1.bolt/meta.db"
)

// Bucket keys matching containerd's metadata/buckets.go
var (
	ctrdBucketKeyVersion   = []byte("v1")
	ctrdBucketKeySnapshots = []byte("snapshots")
	ctrdBucketKeyName      = []byte("name")
	ctrdBucketKeyParent    = []byte("parent")
	ctrdBucketKeyChildren  = []byte("children")
)

// ContainerdMetaReader handles reading metadata from containerd's core bolt database
type ContainerdMetaReader struct {
	db       *bolt.DB
	tempPath string
}

// ContainerdSnapshotInfo represents snapshot info from containerd's core metadata
type ContainerdSnapshotInfo struct {
	Namespace     string `json:"namespace"`
	Snapshotter   string `json:"snapshotter"`
	Key           string `json:"key"`
	Name          string `json:"name,omitempty"` // Internal name (namespace/id/key)
	Parent        string `json:"parent,omitempty"`
	ChildrenCount int    `json:"children_count"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

// ContainerdChildLink represents a parent-child link in containerd's metadata
type ContainerdChildLink struct {
	Namespace   string `json:"namespace"`
	Snapshotter string `json:"snapshotter"`
	ParentKey   string `json:"parent_key"`
	ChildKey    string `json:"child_key"`
	ChildExists bool   `json:"child_exists"`
}

// NewContainerdMetaReader creates a new reader for containerd's core metadata
func NewContainerdMetaReader(dbPath string) (*ContainerdMetaReader, error) {
	if dbPath == "" {
		dbPath = DefaultContainerdMetaDBPath
	}

	// Try to open in ReadOnly mode
	opts := &bolt.Options{
		ReadOnly: true,
		Timeout:  1 * time.Second,
	}
	db, err := bolt.Open(dbPath, 0400, opts)
	if err != nil {
		// If locked, copy the database
		tempPath, copyErr := copyDatabaseFile(dbPath)
		if copyErr != nil {
			return nil, fmt.Errorf("database is locked and copy failed: %w (original: %v)", copyErr, err)
		}

		db, err = bolt.Open(tempPath, 0400, &bolt.Options{ReadOnly: true})
		if err != nil {
			os.Remove(tempPath)
			return nil, fmt.Errorf("failed to open copied database: %w", err)
		}

		return &ContainerdMetaReader{db: db, tempPath: tempPath}, nil
	}

	return &ContainerdMetaReader{db: db, tempPath: ""}, nil
}

// Close closes the database and cleans up temporary files
func (r *ContainerdMetaReader) Close() error {
	err := r.db.Close()
	if r.tempPath != "" {
		os.Remove(r.tempPath)
	}
	return err
}

// copyDatabaseFile copies a database file to a temp location
func copyDatabaseFile(srcPath string) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp("", "containerd-meta-*.db")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, src); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// ListBucketStructure returns the top-level bucket structure for debugging
func (r *ContainerdMetaReader) ListBucketStructure() (map[string][]string, error) {
	structure := make(map[string][]string)

	err := r.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			topLevel := string(name)
			var subBuckets []string

			b.ForEach(func(k, v []byte) error {
				if v == nil { // It's a sub-bucket
					subBuckets = append(subBuckets, string(k))
				}
				return nil
			})

			structure[topLevel] = subBuckets
			return nil
		})
	})

	return structure, err
}

// ListNamespaces returns all namespaces in the containerd metadata
func (r *ContainerdMetaReader) ListNamespaces() ([]string, error) {
	var namespaces []string

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket([]byte("v1"))
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		return v1Bkt.ForEach(func(k, v []byte) error {
			if v == nil { // It's a bucket (namespace)
				namespaces = append(namespaces, string(k))
			}
			return nil
		})
	})

	return namespaces, err
}

// GetSnapshotInfo gets detailed info about a specific snapshot
func (r *ContainerdMetaReader) GetSnapshotInfo(namespace, snapshotter, key string) (*ContainerdSnapshotInfo, error) {
	var info *ContainerdSnapshotInfo

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(ctrdBucketKeyVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		nsBkt := v1Bkt.Bucket([]byte(namespace))
		if nsBkt == nil {
			return fmt.Errorf("namespace %s not found", namespace)
		}

		snapshotsBkt := nsBkt.Bucket(ctrdBucketKeySnapshots)
		if snapshotsBkt == nil {
			return fmt.Errorf("snapshots bucket not found in namespace %s", namespace)
		}

		ssrBkt := snapshotsBkt.Bucket([]byte(snapshotter))
		if ssrBkt == nil {
			return fmt.Errorf("snapshotter %s not found", snapshotter)
		}

		keyBkt := ssrBkt.Bucket([]byte(key))
		if keyBkt == nil {
			return fmt.Errorf("snapshot %s not found", key)
		}

		info = &ContainerdSnapshotInfo{
			Namespace:   namespace,
			Snapshotter: snapshotter,
			Key:         key,
		}

		// Read name (internal key)
		if name := keyBkt.Get(ctrdBucketKeyName); name != nil {
			info.Name = string(name)
		}

		// Read parent
		if parent := keyBkt.Get(ctrdBucketKeyParent); parent != nil {
			info.Parent = string(parent)
		}

		// Read timestamps
		var created, updated time.Time
		if err := boltutil.ReadTimestamps(keyBkt, &created, &updated); err == nil {
			if !created.IsZero() {
				info.CreatedAt = created.Format(time.RFC3339)
			}
			if !updated.IsZero() {
				info.UpdatedAt = updated.Format(time.RFC3339)
			}
		}

		// Count children
		childrenBkt := keyBkt.Bucket(ctrdBucketKeyChildren)
		if childrenBkt != nil {
			childrenBkt.ForEach(func(ck, cv []byte) error {
				info.ChildrenCount++
				return nil
			})
		}

		return nil
	})

	return info, err
}

// ListSnapshots lists all snapshots for a namespace and snapshotter
func (r *ContainerdMetaReader) ListSnapshots(namespace, snapshotter string) ([]ContainerdSnapshotInfo, error) {
	var snapshots []ContainerdSnapshotInfo

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(ctrdBucketKeyVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		nsBkt := v1Bkt.Bucket([]byte(namespace))
		if nsBkt == nil {
			return fmt.Errorf("namespace %s not found", namespace)
		}

		snapshotsBkt := nsBkt.Bucket(ctrdBucketKeySnapshots)
		if snapshotsBkt == nil {
			return nil // No snapshots bucket is OK
		}

		ssrBkt := snapshotsBkt.Bucket([]byte(snapshotter))
		if ssrBkt == nil {
			return nil // No snapshotter bucket is OK
		}

		return ssrBkt.ForEach(func(k, v []byte) error {
			if v != nil {
				return nil // Skip non-buckets
			}

			keyBkt := ssrBkt.Bucket(k)
			if keyBkt == nil {
				return nil
			}

			info := ContainerdSnapshotInfo{
				Namespace:   namespace,
				Snapshotter: snapshotter,
				Key:         string(k),
			}

			// Read name (the internal key used by snapshotter)
			if name := keyBkt.Get(ctrdBucketKeyName); name != nil {
				info.Name = string(name)
			}

			// Read parent
			if parent := keyBkt.Get(ctrdBucketKeyParent); parent != nil {
				info.Parent = string(parent)
			}

			// Read timestamps using boltutil
			var created, updated time.Time
			if err := boltutil.ReadTimestamps(keyBkt, &created, &updated); err == nil {
				if !created.IsZero() {
					info.CreatedAt = created.Format(time.RFC3339)
				}
				if !updated.IsZero() {
					info.UpdatedAt = updated.Format(time.RFC3339)
				}
			}

			// Count children
			childrenBkt := keyBkt.Bucket(ctrdBucketKeyChildren)
			if childrenBkt != nil {
				childrenBkt.ForEach(func(ck, cv []byte) error {
					info.ChildrenCount++
					return nil
				})
			}

			snapshots = append(snapshots, info)
			return nil
		})
	})

	return snapshots, err
}

// DumpBucketContents dumps raw bucket contents for debugging
func (r *ContainerdMetaReader) DumpBucketContents(path []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	err := r.db.View(func(tx *bolt.Tx) error {
		var bkt *bolt.Bucket

		// Navigate to the target bucket
		for i, p := range path {
			if i == 0 {
				bkt = tx.Bucket([]byte(p))
			} else {
				if bkt == nil {
					return fmt.Errorf("bucket %s not found", path[i-1])
				}
				bkt = bkt.Bucket([]byte(p))
			}
			if bkt == nil {
				return fmt.Errorf("bucket %s not found", p)
			}
		}

		// Dump contents
		return bkt.ForEach(func(k, v []byte) error {
			key := string(k)
			if v == nil {
				result[key] = "(bucket)"
			} else {
				// Try to interpret the value
				if len(v) <= 8 {
					// Could be a number
					num, n := binary.Uvarint(v)
					if n > 0 {
						result[key] = fmt.Sprintf("%d (or raw: %x)", num, v)
					} else {
						result[key] = fmt.Sprintf("raw: %x", v)
					}
				} else {
					// Likely a string
					result[key] = string(v)
				}
			}
			return nil
		})
	})

	return result, err
}

// ListChildrenFromBucket reads the children bucket for a specific snapshot
// This is the actual children tracking in containerd's core metadata
func (r *ContainerdMetaReader) ListChildrenFromBucket(namespace, snapshotter, snapshotKey string) ([]string, error) {
	var children []string

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket([]byte("v1"))
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		nsBkt := v1Bkt.Bucket([]byte(namespace))
		if nsBkt == nil {
			return fmt.Errorf("namespace %s not found", namespace)
		}

		snapshotsBkt := nsBkt.Bucket([]byte("snapshots"))
		if snapshotsBkt == nil {
			return fmt.Errorf("snapshots bucket not found")
		}

		ssrBkt := snapshotsBkt.Bucket([]byte(snapshotter))
		if ssrBkt == nil {
			return fmt.Errorf("snapshotter %s not found", snapshotter)
		}

		keyBkt := ssrBkt.Bucket([]byte(snapshotKey))
		if keyBkt == nil {
			return fmt.Errorf("snapshot %s not found", snapshotKey)
		}

		childrenBkt := keyBkt.Bucket([]byte("children"))
		if childrenBkt == nil {
			// No children bucket means no children
			return nil
		}

		// Children bucket stores: childKey -> timestamp or empty
		return childrenBkt.ForEach(func(k, v []byte) error {
			children = append(children, string(k))
			return nil
		})
	})

	return children, err
}

// GetSnapshotName reads the "name" field from a snapshot bucket
// This contains the full key as used internally
func (r *ContainerdMetaReader) GetSnapshotName(namespace, snapshotter, snapshotKey string) (string, error) {
	var name string

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket([]byte("v1"))
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		nsBkt := v1Bkt.Bucket([]byte(namespace))
		if nsBkt == nil {
			return fmt.Errorf("namespace %s not found", namespace)
		}

		snapshotsBkt := nsBkt.Bucket([]byte("snapshots"))
		if snapshotsBkt == nil {
			return fmt.Errorf("snapshots bucket not found")
		}

		ssrBkt := snapshotsBkt.Bucket([]byte(snapshotter))
		if ssrBkt == nil {
			return fmt.Errorf("snapshotter %s not found", snapshotter)
		}

		keyBkt := ssrBkt.Bucket([]byte(snapshotKey))
		if keyBkt == nil {
			return fmt.Errorf("snapshot %s not found", snapshotKey)
		}

		if nameData := keyBkt.Get([]byte("name")); nameData != nil {
			name = string(nameData)
		}

		return nil
	})

	return name, err
}

// FindAllChildrenLinks finds all parent->children relationships in containerd metadata
func (r *ContainerdMetaReader) FindAllChildrenLinks(namespace, snapshotter string) (map[string][]string, error) {
	result := make(map[string][]string)

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket([]byte("v1"))
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		nsBkt := v1Bkt.Bucket([]byte(namespace))
		if nsBkt == nil {
			return fmt.Errorf("namespace %s not found", namespace)
		}

		snapshotsBkt := nsBkt.Bucket([]byte("snapshots"))
		if snapshotsBkt == nil {
			return nil
		}

		ssrBkt := snapshotsBkt.Bucket([]byte(snapshotter))
		if ssrBkt == nil {
			return nil
		}

		// Iterate through all snapshots
		return ssrBkt.ForEach(func(k, v []byte) error {
			if v != nil {
				return nil // Skip non-buckets
			}

			keyBkt := ssrBkt.Bucket(k)
			if keyBkt == nil {
				return nil
			}

			childrenBkt := keyBkt.Bucket([]byte("children"))
			if childrenBkt == nil {
				return nil
			}

			parentKey := string(k)
			var children []string

			childrenBkt.ForEach(func(ck, cv []byte) error {
				children = append(children, string(ck))
				return nil
			})

			if len(children) > 0 {
				result[parentKey] = children
			}

			return nil
		})
	})

	return result, err
}

// ContainerdGhostChild represents a ghost child reference in containerd's metadata
type ContainerdGhostChild struct {
	Namespace   string `json:"namespace"`
	Snapshotter string `json:"snapshotter"`
	ParentKey   string `json:"parent_key"`
	ChildKey    string `json:"child_key"`
}

// ListChildrenWithExistence checks if children in the children bucket actually exist
func (r *ContainerdMetaReader) ListChildrenWithExistence(namespace, snapshotter, snapshotKey string) (existing []string, ghosts []string, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(ctrdBucketKeyVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		nsBkt := v1Bkt.Bucket([]byte(namespace))
		if nsBkt == nil {
			return fmt.Errorf("namespace %s not found", namespace)
		}

		snapshotsBkt := nsBkt.Bucket(ctrdBucketKeySnapshots)
		if snapshotsBkt == nil {
			return fmt.Errorf("snapshots bucket not found")
		}

		ssrBkt := snapshotsBkt.Bucket([]byte(snapshotter))
		if ssrBkt == nil {
			return fmt.Errorf("snapshotter %s not found", snapshotter)
		}

		keyBkt := ssrBkt.Bucket([]byte(snapshotKey))
		if keyBkt == nil {
			return fmt.Errorf("snapshot %s not found", snapshotKey)
		}

		childrenBkt := keyBkt.Bucket(ctrdBucketKeyChildren)
		if childrenBkt == nil {
			return nil
		}

		// Check each child
		return childrenBkt.ForEach(func(k, v []byte) error {
			childKey := string(k)
			// Check if child exists in snapshotter bucket
			if ssrBkt.Bucket([]byte(childKey)) != nil {
				existing = append(existing, childKey)
			} else {
				ghosts = append(ghosts, childKey)
			}
			return nil
		})
	})

	return existing, ghosts, err
}

// FindAllGhostChildren finds all ghost children across all snapshots
func (r *ContainerdMetaReader) FindAllGhostChildren(namespace, snapshotter string) ([]ContainerdGhostChild, error) {
	var ghosts []ContainerdGhostChild

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(ctrdBucketKeyVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		nsBkt := v1Bkt.Bucket([]byte(namespace))
		if nsBkt == nil {
			return fmt.Errorf("namespace %s not found", namespace)
		}

		snapshotsBkt := nsBkt.Bucket(ctrdBucketKeySnapshots)
		if snapshotsBkt == nil {
			return nil
		}

		ssrBkt := snapshotsBkt.Bucket([]byte(snapshotter))
		if ssrBkt == nil {
			return nil
		}

		// Iterate through all snapshots
		return ssrBkt.ForEach(func(k, v []byte) error {
			if v != nil {
				return nil // Skip non-buckets
			}

			keyBkt := ssrBkt.Bucket(k)
			if keyBkt == nil {
				return nil
			}

			childrenBkt := keyBkt.Bucket(ctrdBucketKeyChildren)
			if childrenBkt == nil {
				return nil
			}

			parentKey := string(k)

			// Check each child
			childrenBkt.ForEach(func(ck, cv []byte) error {
				childKey := string(ck)
				// Check if child exists
				if ssrBkt.Bucket([]byte(childKey)) == nil {
					ghosts = append(ghosts, ContainerdGhostChild{
						Namespace:   namespace,
						Snapshotter: snapshotter,
						ParentKey:   parentKey,
						ChildKey:    childKey,
					})
				}
				return nil
			})

			return nil
		})
	})

	return ghosts, err
}

// RemoveGhostChildrenFromContainerd removes ghost children entries from containerd's meta.db
// WARNING: containerd must be stopped before calling this function
func RemoveGhostChildrenFromContainerd(dbPath string, ghosts []ContainerdGhostChild) (removed, failed int, err error) {
	if len(ghosts) == 0 {
		return 0, 0, nil
	}

	// Open database in read-write mode
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{
		Timeout: 3 * time.Second,
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open database for writing (is containerd running?): %w", err)
	}
	defer db.Close()

	// Group ghosts by namespace/snapshotter/parent for efficient batch processing
	type ghostKey struct {
		ns, ssr, parent string
	}
	grouped := make(map[ghostKey][]string)
	for _, g := range ghosts {
		key := ghostKey{g.Namespace, g.Snapshotter, g.ParentKey}
		grouped[key] = append(grouped[key], g.ChildKey)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(ctrdBucketKeyVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		for key, children := range grouped {
			nsBkt := v1Bkt.Bucket([]byte(key.ns))
			if nsBkt == nil {
				failed += len(children)
				continue
			}

			snapshotsBkt := nsBkt.Bucket(ctrdBucketKeySnapshots)
			if snapshotsBkt == nil {
				failed += len(children)
				continue
			}

			ssrBkt := snapshotsBkt.Bucket([]byte(key.ssr))
			if ssrBkt == nil {
				failed += len(children)
				continue
			}

			parentBkt := ssrBkt.Bucket([]byte(key.parent))
			if parentBkt == nil {
				failed += len(children)
				continue
			}

			childrenBkt := parentBkt.Bucket(ctrdBucketKeyChildren)
			if childrenBkt == nil {
				failed += len(children)
				continue
			}

			for _, childKey := range children {
				if err := childrenBkt.Delete([]byte(childKey)); err != nil {
					failed++
				} else {
					removed++
				}
			}
		}

		return nil
	})

	return removed, failed, err
}
