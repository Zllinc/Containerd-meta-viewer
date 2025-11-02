package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/containerd/containerd/metadata/boltutil"
	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/meta-viewer/internal/utils"
	bolt "go.etcd.io/bbolt"
)

// setupTestDB creates a temporary test database with sample data
func setupTestDB(t *testing.T) string {
	// Create temporary database file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create sample data
	err = db.Update(func(tx *bolt.Tx) error {
		// Create v1 bucket
		v1Bkt, err := tx.CreateBucket(bucketKeyStorageVersion)
		if err != nil {
			return err
		}

		// Create snapshots bucket
		snapshotsBkt, err := v1Bkt.CreateBucket(bucketKeySnapshot)
		if err != nil {
			return err
		}

		// Create parents bucket
		_, err = v1Bkt.CreateBucket(bucketKeyParents)
		if err != nil {
			return err
		}

		// Create devbox storage bucket
		devboxBkt, err := v1Bkt.CreateBucket(DevboxStoragePathBucket)
		if err != nil {
			return err
		}

		// Create sample snapshots
		err = createTestSnapshot(snapshotsBkt, "snapshot-1", 1, snapshots.KindActive, "", "content-123", "/mount/path/1")
		if err != nil {
			return err
		}

		err = createTestSnapshot(snapshotsBkt, "snapshot-2", 2, snapshots.KindCommitted, "snapshot-1", "content-456", "/mount/path/2")
		if err != nil {
			return err
		}

		// Create sample devbox storage entries
		err = createTestDevboxStorage(devboxBkt, "content-123", "lv-volume-1", "/mount/path/1", "active")
		if err != nil {
			return err
		}

		err = createTestDevboxStorage(devboxBkt, "content-456", "lv-volume-2", "/mount/path/2", "active")
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to setup test data: %v", err)
	}

	return dbPath
}

// createTestSnapshot creates a test snapshot in the database
func createTestSnapshot(bkt *bolt.Bucket, key string, id uint64, kind snapshots.Kind, parent, contentID, path string) error {
	snapBkt, err := bkt.CreateBucket([]byte(key))
	if err != nil {
		return err
	}

	// Write basic snapshot data
	idBytes := make([]byte, 8)
	n := utils.EncodeID(idBytes, id)
	if err := snapBkt.Put(bucketKeyID, idBytes[:n]); err != nil {
		return err
	}

	if err := snapBkt.Put(bucketKeyKind, []byte{byte(kind)}); err != nil {
		return err
	}

	if parent != "" {
		if err := snapBkt.Put(bucketKeyParent, []byte(parent)); err != nil {
			return err
		}
	}

	// Write timestamps
	now := time.Now()
	if err := boltutil.WriteTimestamps(snapBkt, now, now); err != nil {
		return err
	}

	// Write labels
	labels := map[string]string{
		"test-label": "test-value",
	}
	if err := boltutil.WriteLabels(snapBkt, labels); err != nil {
		return err
	}

	// Write usage data
	inodesBytes := make([]byte, 8)
	n = utils.EncodeSize(inodesBytes, 1000)
	if err := snapBkt.Put(bucketKeyInodes, inodesBytes[:n]); err != nil {
		return err
	}

	sizeBytes := make([]byte, 8)
	n = utils.EncodeSize(sizeBytes, 2048)
	if err := snapBkt.Put(bucketKeySize, sizeBytes[:n]); err != nil {
		return err
	}

	// Write devbox specific data
	if contentID != "" {
		if err := snapBkt.Put(DevboxKeyContentID, []byte(contentID)); err != nil {
			return err
		}
	}

	if path != "" {
		if err := snapBkt.Put(DevboxKeyPath, []byte(path)); err != nil {
			return err
		}
	}

	return nil
}

// createTestDevboxStorage creates a test devbox storage entry
func createTestDevboxStorage(bkt *bolt.Bucket, contentID, lvName, path, status string) error {
	contentBkt, err := bkt.CreateBucket([]byte(contentID))
	if err != nil {
		return err
	}

	if lvName != "" {
		if err := contentBkt.Put(DevboxKeyLvName, []byte(lvName)); err != nil {
			return err
		}
	}

	if path != "" {
		if err := contentBkt.Put(DevboxKeyPath, []byte(path)); err != nil {
			return err
		}
	}

	if status != "" {
		if err := contentBkt.Put(DevboxKeyStatus, []byte(status)); err != nil {
			return err
		}
	}

	return nil
}

func TestNewMetaReader(t *testing.T) {
	t.Run("valid database", func(t *testing.T) {
		dbPath := setupTestDB(t)
		reader, err := NewMetaReader(dbPath)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		defer reader.Close()

		if reader.db == nil {
			t.Error("Expected database to be initialized")
		}
	})

	t.Run("non-existent database", func(t *testing.T) {
		_, err := NewMetaReader("/non/existent/path")
		if err == nil {
			t.Error("Expected error for non-existent database")
		}
	})
}

func TestMetaReader_ListBuckets(t *testing.T) {
	dbPath := setupTestDB(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	buckets, err := reader.ListBuckets()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have at least the v1 bucket
	found := false
	for _, bucket := range buckets {
		if bucket.Name == "v1" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find v1 bucket")
	}
}

func TestMetaReader_ListSnapshots(t *testing.T) {
	dbPath := setupTestDB(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	snapshotList, err := reader.ListSnapshots()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(snapshotList) != 2 {
		t.Errorf("Expected 2 snapshots, got %d", len(snapshotList))
	}

	// Check first snapshot
	if snapshotList[0].Key != "snapshot-1" {
		t.Errorf("Expected first snapshot key = 'snapshot-1', got %s", snapshotList[0].Key)
	}

	if snapshotList[0].ID != 1 {
		t.Errorf("Expected first snapshot ID = 1, got %d", snapshotList[0].ID)
	}

	if snapshotList[0].Kind != snapshots.KindActive {
		t.Errorf("Expected first snapshot kind = KindActive, got %v", snapshotList[0].Kind)
	}

	if snapshotList[0].ContentID != "content-123" {
		t.Errorf("Expected first snapshot contentID = 'content-123', got %s", snapshotList[0].ContentID)
	}

	if snapshotList[0].Path != "/mount/path/1" {
		t.Errorf("Expected first snapshot path = '/mount/path/1', got %s", snapshotList[0].Path)
	}
}

func TestMetaReader_GetSnapshot(t *testing.T) {
	dbPath := setupTestDB(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	t.Run("existing snapshot", func(t *testing.T) {
		snapshot, err := reader.GetSnapshot("snapshot-1")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if snapshot.Key != "snapshot-1" {
			t.Errorf("Expected key = 'snapshot-1', got %s", snapshot.Key)
		}

		if snapshot.ID != 1 {
			t.Errorf("Expected ID = 1, got %d", snapshot.ID)
		}

		if snapshot.ContentID != "content-123" {
			t.Errorf("Expected contentID = 'content-123', got %s", snapshot.ContentID)
		}
	})

	t.Run("non-existent snapshot", func(t *testing.T) {
		_, err := reader.GetSnapshot("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent snapshot")
		}
	})
}

func TestMetaReader_ListDevboxStorage(t *testing.T) {
	dbPath := setupTestDB(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	storage, err := reader.ListDevboxStorage()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(storage) != 2 {
		t.Errorf("Expected 2 storage entries, got %d", len(storage))
	}

	// Check first storage entry
	if storage[0].ContentID != "content-123" {
		t.Errorf("Expected first storage contentID = 'content-123', got %s", storage[0].ContentID)
	}

	if storage[0].LvName != "lv-volume-1" {
		t.Errorf("Expected first storage lvName = 'lv-volume-1', got %s", storage[0].LvName)
	}

	if storage[0].Path != "/mount/path/1" {
		t.Errorf("Expected first storage path = '/mount/path/1', got %s", storage[0].Path)
	}

	if storage[0].Status != "active" {
		t.Errorf("Expected first storage status = 'active', got %s", storage[0].Status)
	}
}

func TestMetaReader_GetDevboxStorage(t *testing.T) {
	dbPath := setupTestDB(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	t.Run("existing storage", func(t *testing.T) {
		storage, err := reader.GetDevboxStorage("content-123")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if storage.ContentID != "content-123" {
			t.Errorf("Expected contentID = 'content-123', got %s", storage.ContentID)
		}

		if storage.LvName != "lv-volume-1" {
			t.Errorf("Expected lvName = 'lv-volume-1', got %s", storage.LvName)
		}

		if storage.Path != "/mount/path/1" {
			t.Errorf("Expected path = '/mount/path/1', got %s", storage.Path)
		}

		if storage.Status != "active" {
			t.Errorf("Expected status = 'active', got %s", storage.Status)
		}
	})

	t.Run("non-existent storage", func(t *testing.T) {
		_, err := reader.GetDevboxStorage("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent storage")
		}
	})
}

func TestMetaReader_SearchSnapshots(t *testing.T) {
	dbPath := setupTestDB(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	t.Run("search by content ID", func(t *testing.T) {
		results, err := reader.SearchSnapshots("content-123", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if results[0].ContentID != "content-123" {
			t.Errorf("Expected contentID = 'content-123', got %s", results[0].ContentID)
		}
	})

	t.Run("search by path", func(t *testing.T) {
		results, err := reader.SearchSnapshots("", "/mount/path/1")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}

		if results[0].Path != "/mount/path/1" {
			t.Errorf("Expected path = '/mount/path/1', got %s", results[0].Path)
		}
	})

	t.Run("search by both criteria", func(t *testing.T) {
		results, err := reader.SearchSnapshots("content-123", "/mount/path/1")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("search with no matches", func(t *testing.T) {
		results, err := reader.SearchSnapshots("non-existent", "")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})
}

func TestMetaReader_Close(t *testing.T) {
	dbPath := setupTestDB(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	// Close should not return an error
	if err := reader.Close(); err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}

	// Test that database operations fail after close
	_, err = reader.ListBuckets()
	if err == nil {
		t.Error("Expected error when using closed reader")
	}
}

// Integration test with empty database
func TestMetaReader_EmptyDatabase(t *testing.T) {
	// Create empty database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create empty database: %v", err)
	}
	db.Close()

	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	// List operations on empty database should succeed but return empty results
	buckets, err := reader.ListBuckets()
	if err != nil {
		t.Fatalf("Expected no error listing buckets, got %v", err)
	}

	// Should have no buckets in empty database
	if len(buckets) != 0 {
		t.Errorf("Expected 0 buckets in empty database, got %d", len(buckets))
	}

	// Listing snapshots should return an error since v1 bucket doesn't exist
	_, err = reader.ListSnapshots()
	if err == nil {
		t.Error("Expected error when listing snapshots from empty database")
	}

	// Listing devbox storage should also return an error since v1 bucket doesn't exist
	_, err = reader.ListDevboxStorage()
	if err == nil {
		t.Error("Expected error when listing devbox storage from empty database")
	}
}