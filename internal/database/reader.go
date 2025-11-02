package database

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/containerd/containerd/metadata/boltutil"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/meta-viewer/internal/utils"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketKeyStorageVersion = []byte("v1")
	bucketKeySnapshot       = []byte("snapshots")
	bucketKeyParents        = []byte("parents")
	bucketKeyID             = []byte("id")
	bucketKeyParent         = []byte("parent")
	bucketKeyKind           = []byte("kind")
	bucketKeyInodes         = []byte("inodes")
	bucketKeySize           = []byte("size")
	DevboxKeyContentID      = []byte("content_id")
	DevboxKeyPath           = []byte("path")
	DevboxStoragePathBucket = []byte("devbox_storage_path")
	DevboxKeyLvName         = []byte("lv_name")
	DevboxKeyStatus         = []byte("status")
	DevboxStatusActive      = []byte("active")
	DevboxStatusRemoved     = []byte("removed")
)

// MetaReader handles reading metadata from devbox snapshotter bolt database
type MetaReader struct {
	db       *bolt.DB
	tempPath string // Path to temporary copy if database was copied
}

// NewMetaReader creates a new MetaReader instance
// If the database is locked by another process, it will automatically copy
// the database file to a temporary location and read from the copy.
func NewMetaReader(dbPath string) (*MetaReader, error) {
	// First, try to open in ReadOnly mode with a short timeout
	opts := &bolt.Options{
		ReadOnly: true,
		Timeout:  1 * time.Second, // Short timeout to quickly detect lock
	}
	db, err := bolt.Open(dbPath, 0400, opts)

	var tempPath string

	// If we got a timeout (database is locked), try copying the file
	if err != nil && err.Error() == "timeout" {
		// Copy database to temporary file for reading
		tempFile, err := os.CreateTemp("", "containerd-meta-viewer-*.db")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary file for database copy: %w", err)
		}
		tempPath = tempFile.Name()
		tempFile.Close()

		// Copy the database file
		if err := copyFile(dbPath, tempPath); err != nil {
			os.Remove(tempPath)
			return nil, fmt.Errorf("failed to copy database file for reading: %w", err)
		}

		// Try to open the copy in ReadOnly mode
		opts := &bolt.Options{
			ReadOnly: true,
		}
		db, err = bolt.Open(tempPath, 0400, opts)
		if err != nil {
			os.Remove(tempPath)
			return nil, fmt.Errorf("failed to open copied database: %w", err)
		}
	} else if err != nil {
		// Other errors (file not found, permission denied, etc.)
		return nil, fmt.Errorf("failed to open bolt database: %w", err)
	}

	return &MetaReader{db: db, tempPath: tempPath}, nil
}

// Close closes the database connection and cleans up temporary files
func (r *MetaReader) Close() error {
	var err error
	if r.db != nil {
		err = r.db.Close()
	}

	// Clean up temporary copy if it was created
	if r.tempPath != "" {
		if removeErr := os.Remove(r.tempPath); removeErr != nil && err == nil {
			err = removeErr
		}
	}

	return err
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Ensure the file is synced to disk
	return destFile.Sync()
}

// ListBuckets returns all top-level buckets in the database
func (r *MetaReader) ListBuckets() ([]BucketInfo, error) {
	var buckets []BucketInfo

	err := r.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			info := BucketInfo{
				Name:     string(name),
				KeyCount: b.Stats().KeyN,
			}
			buckets = append(buckets, info)
			return nil
		})
	})

	return buckets, err
}

// ListSnapshots returns all snapshots in the database
func (r *MetaReader) ListSnapshots() ([]SnapshotInfo, error) {
	var snapshots []SnapshotInfo

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(bucketKeyStorageVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		snapshotsBkt := v1Bkt.Bucket(bucketKeySnapshot)
		if snapshotsBkt == nil {
			return fmt.Errorf("snapshots bucket not found")
		}

		return snapshotsBkt.ForEach(func(k, v []byte) error {
			if v != nil { // skip non-buckets
				return nil
			}

			sbkt := snapshotsBkt.Bucket(k)
			info, err := r.readSnapshotInfo(string(k), sbkt)
			if err != nil {
				return fmt.Errorf("failed to read snapshot %s: %w", string(k), err)
			}

			snapshots = append(snapshots, info)
			return nil
		})
	})

	return snapshots, err
}

// GetSnapshot returns a specific snapshot by key
func (r *MetaReader) GetSnapshot(key string) (*SnapshotInfo, error) {
	var info *SnapshotInfo

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(bucketKeyStorageVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		snapshotsBkt := v1Bkt.Bucket(bucketKeySnapshot)
		if snapshotsBkt == nil {
			return fmt.Errorf("snapshots bucket not found")
		}

		sbkt := snapshotsBkt.Bucket([]byte(key))
		if sbkt == nil {
			return fmt.Errorf("snapshot %s not found", key)
		}

		snapshotInfo, err := r.readSnapshotInfo(key, sbkt)
		if err != nil {
			return err
		}

		info = &snapshotInfo
		return nil
	})

	return info, err
}

// ListDevboxStorage returns all devbox storage entries
func (r *MetaReader) ListDevboxStorage() ([]DevboxStorageInfo, error) {
	var storage []DevboxStorageInfo

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(bucketKeyStorageVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		devboxBkt := v1Bkt.Bucket(DevboxStoragePathBucket)
		if devboxBkt == nil {
			// Devbox bucket might not exist, return empty list
			return nil
		}

		return devboxBkt.ForEach(func(k, v []byte) error {
			if v != nil { // skip non-buckets
				return nil
			}

			contentBkt := devboxBkt.Bucket(k)
			info, err := r.readDevboxStorageInfo(string(k), contentBkt)
			if err != nil {
				return fmt.Errorf("failed to read devbox storage %s: %w", string(k), err)
			}

			storage = append(storage, info)
			return nil
		})
	})

	return storage, err
}

// GetDevboxStorage returns a specific devbox storage entry by content ID
func (r *MetaReader) GetDevboxStorage(contentID string) (*DevboxStorageInfo, error) {
	var info *DevboxStorageInfo

	err := r.db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(bucketKeyStorageVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		devboxBkt := v1Bkt.Bucket(DevboxStoragePathBucket)
		if devboxBkt == nil {
			return fmt.Errorf("devbox storage bucket not found")
		}

		contentBkt := devboxBkt.Bucket([]byte(contentID))
		if contentBkt == nil {
			return fmt.Errorf("devbox storage %s not found", contentID)
		}

		storageInfo, err := r.readDevboxStorageInfo(contentID, contentBkt)
		if err != nil {
			return err
		}

		info = &storageInfo
		return nil
	})

	return info, err
}

// SearchSnapshots searches snapshots by content ID or path
func (r *MetaReader) SearchSnapshots(contentID, path string) ([]SnapshotInfo, error) {
	var results []SnapshotInfo

	snapshots, err := r.ListSnapshots()
	if err != nil {
		return nil, err
	}

	for _, snapshot := range snapshots {
		match := true

		if contentID != "" && snapshot.ContentID != contentID {
			match = false
		}

		if path != "" && snapshot.Path != path {
			match = false
		}

		if match {
			results = append(results, snapshot)
		}
	}

	return results, nil
}

// readSnapshotInfo reads snapshot information from a bucket
func (r *MetaReader) readSnapshotInfo(key string, bkt *bolt.Bucket) (SnapshotInfo, error) {
	var info SnapshotInfo
	info.Key = key

	// Read basic fields
	if idData := bkt.Get(bucketKeyID); idData != nil {
		info.ID = utils.ReadID(idData)
	}

	if kindData := bkt.Get(bucketKeyKind); len(kindData) == 1 {
		info.Kind = snapshots.Kind(kindData[0])
	}

	if parentData := bkt.Get(bucketKeyParent); parentData != nil {
		info.Parent = string(parentData)
	}

	// Read timestamps
	if err := boltutil.ReadTimestamps(bkt, &info.CreatedAt, &info.UpdatedAt); err != nil {
		return info, fmt.Errorf("failed to read timestamps: %w", err)
	}

	// Read labels
	labels, err := boltutil.ReadLabels(bkt)
	if err != nil {
		return info, fmt.Errorf("failed to read labels: %w", err)
	}
	info.Labels = labels

	// Read usage information
	if inodesData := bkt.Get(bucketKeyInodes); inodesData != nil {
		info.Inodes = utils.ReadInodes(inodesData)
	}

	if sizeData := bkt.Get(bucketKeySize); sizeData != nil {
		info.Size = utils.ReadSize(sizeData)
	}

	// Read devbox specific fields
	if contentIDData := bkt.Get(DevboxKeyContentID); contentIDData != nil {
		info.ContentID = string(contentIDData)
	}

	if pathData := bkt.Get(DevboxKeyPath); pathData != nil {
		info.Path = string(pathData)
	}

	return info, nil
}

// readDevboxStorageInfo reads devbox storage information from a bucket
func (r *MetaReader) readDevboxStorageInfo(contentID string, bkt *bolt.Bucket) (DevboxStorageInfo, error) {
	var info DevboxStorageInfo
	info.ContentID = contentID

	if lvNameData := bkt.Get(DevboxKeyLvName); lvNameData != nil {
		info.LvName = string(lvNameData)
	}

	if pathData := bkt.Get(DevboxKeyPath); pathData != nil {
		info.Path = string(pathData)
	}

	if statusData := bkt.Get(DevboxKeyStatus); statusData != nil {
		info.Status = string(statusData)
	}

	return info, nil
}
