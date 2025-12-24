package database

import (
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
