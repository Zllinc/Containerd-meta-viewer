package database

import (
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestClearDevboxSnapshotKey(t *testing.T) {
	dbPath := setupDevboxTestDatabase(t, true)

	if err := ClearDevboxSnapshotKey(dbPath, "content-123"); err != nil {
		t.Fatalf("ClearDevboxSnapshotKey() returned error: %v", err)
	}

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("failed to reopen db: %v", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		v1Bkt := tx.Bucket(bucketKeyStorageVersion)
		if v1Bkt == nil {
			t.Fatal("v1 bucket missing")
		}
		devboxBkt := v1Bkt.Bucket(DevboxStoragePathBucket)
		if devboxBkt == nil {
			t.Fatal("devbox bucket missing")
		}
		contentBkt := devboxBkt.Bucket([]byte("content-123"))
		if contentBkt == nil {
			t.Fatal("content bucket missing")
		}
		if val := contentBkt.Get(DevboxKeySnapshotKey); val != nil {
			t.Fatalf("expected snapshot key to be cleared, got %q", string(val))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("validation view failed: %v", err)
	}
}

func TestClearDevboxSnapshotKeyMissingContent(t *testing.T) {
	dbPath := setupDevboxTestDatabase(t, false)

	err := ClearDevboxSnapshotKey(dbPath, "content-123")
	if err == nil {
		t.Fatal("expected error when content bucket missing")
	}
}

func setupDevboxTestDatabase(t *testing.T, withSnapshotKey bool) string {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "devbox.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		v1Bkt, err := tx.CreateBucket(bucketKeyStorageVersion)
		if err != nil {
			return err
		}

		devboxBkt, err := v1Bkt.CreateBucket(DevboxStoragePathBucket)
		if err != nil {
			return err
		}

		if withSnapshotKey {
			contentBkt, err := devboxBkt.CreateBucket([]byte("content-123"))
			if err != nil {
				return err
			}
			if err := contentBkt.Put(DevboxKeySnapshotKey, []byte("sha256:test")); err != nil {
				return err
			}
		}
		return nil
	})
	db.Close()
	if err != nil {
		t.Fatalf("failed to seed db: %v", err)
	}

	return dbPath
}
