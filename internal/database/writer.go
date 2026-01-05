package database

import (
	"encoding/binary"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// ClearDevboxSnapshotKey removes the snapshot key reference for a content ID
// under the devbox_storage_path bucket.
func ClearDevboxSnapshotKey(dbPath, contentID string) error {
	if contentID == "" {
		return fmt.Errorf("content ID cannot be empty")
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return fmt.Errorf("failed to open bolt database for writing: %w", err)
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
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

		if err := contentBkt.Delete(DevboxKeySnapshotKey); err != nil {
			return fmt.Errorf("failed to clear snapshot key for %s: %w", contentID, err)
		}

		return nil
	})
}

// RemoveOrphanSnapshot removes an orphan snapshot from the database.
// This follows the same logic as containerd's devbox Remove function:
// 1. RemoveDevbox: clears devbox content references
// 2. Remove: deletes the snapshot bucket and parent links
func RemoveOrphanSnapshot(dbPath, key string) error {
	if key == "" {
		return fmt.Errorf("snapshot key cannot be empty")
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return fmt.Errorf("failed to open bolt database for writing: %w", err)
	}
	defer db.Close()

	return db.Update(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(bucketKeyStorageVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		// Step 1: RemoveDevbox - clean up devbox content references
		snapshotsBkt := v1Bkt.Bucket(bucketKeySnapshot)
		if snapshotsBkt == nil {
			return fmt.Errorf("snapshots bucket not found")
		}

		sbkt := snapshotsBkt.Bucket([]byte(key))
		if sbkt == nil {
			return fmt.Errorf("snapshot %s not found", key)
		}

		// Get content ID from snapshot
		contentID := sbkt.Get(DevboxKeyContentID)

		// If contentID exists, clean up devbox_storage_path bucket
		if len(contentID) > 0 {
			devboxBkt := v1Bkt.Bucket(DevboxStoragePathBucket)
			if devboxBkt != nil {
				contentBkt := devboxBkt.Bucket(contentID)
				if contentBkt != nil {
					status := contentBkt.Get(DevboxKeyStatus)
					if status != nil && string(status) == string(DevboxStatusRemoved) {
						// Remove the bucket if it is already marked as removed
						devboxBkt.DeleteBucket(contentID)
					} else {
						// Clear snapshot_key if it matches current key
						snapshotKey := contentBkt.Get(DevboxKeySnapshotKey)
						if string(snapshotKey) == key {
							contentBkt.Delete(DevboxKeySnapshotKey)
						}
					}
				}
			}
		}

		// Step 2: Remove - delete snapshot and parent links
		// Read snapshot info first
		id := readSnapshotID(sbkt)
		parent := sbkt.Get(bucketKeyParent)

		// Handle parent links
		parentsBkt := v1Bkt.Bucket(bucketKeyParents)
		if parentsBkt != nil {
			// Check if this snapshot has children (cannot remove if it does)
			// For orphan cleanup, we skip this check since orphans shouldn't have valid children

			// Delete parent link if parent exists
			if len(parent) > 0 {
				parentSbkt := snapshotsBkt.Bucket(parent)
				if parentSbkt != nil {
					parentID := readSnapshotID(parentSbkt)
					parentsBkt.Delete(parentKey(parentID, id))
				}
			}
		}

		// Delete the snapshot bucket
		if err := snapshotsBkt.DeleteBucket([]byte(key)); err != nil {
			return fmt.Errorf("failed to delete snapshot %s: %w", key, err)
		}

		return nil
	})
}

// readSnapshotID reads the ID from a snapshot bucket
func readSnapshotID(bkt *bolt.Bucket) uint64 {
	idData := bkt.Get(bucketKeyID)
	if idData == nil {
		return 0
	}
	id, _ := binary.Uvarint(idData)
	return id
}

// parentKey returns a composite key of the parent and child identifiers
func parentKey(parent, child uint64) []byte {
	b := make([]byte, binary.MaxVarintLen64*2+1)
	i := binary.PutUvarint(b, parent)
	j := binary.PutUvarint(b[i+1:], child)
	return b[0 : i+j+1]
}

// RemoveGhostChildren removes stale parent-child links from the parents bucket.
// These are links where the child snapshot no longer exists but the link remains.
func RemoveGhostChildren(dbPath string, ghosts []GhostChildInfo) (removed int, failed int, err error) {
	if len(ghosts) == 0 {
		return 0, 0, nil
	}

	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to open bolt database for writing: %w", err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(bucketKeyStorageVersion)
		if v1Bkt == nil {
			return fmt.Errorf("v1 bucket not found")
		}

		parentsBkt := v1Bkt.Bucket(bucketKeyParents)
		if parentsBkt == nil {
			return fmt.Errorf("parents bucket not found")
		}

		for _, g := range ghosts {
			key := parentKey(g.ParentID, g.ChildID)
			if err := parentsBkt.Delete(key); err != nil {
				failed++
			} else {
				removed++
			}
		}

		return nil
	})

	return removed, failed, err
}
