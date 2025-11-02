package database

import (
	"os"
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func TestMetaReader_ListBucketsComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func(*testing.T) string
		expectedBuckets []string
		expectError    bool
	}{
		{
			name: "standard database with v1 structure",
			setupFunc: func(t *testing.T) string {
				return setupTestDB(t)
			},
			expectedBuckets: []string{"v1"},
			expectError:    false,
		},
		{
			name: "empty database",
			setupFunc: func(t *testing.T) string {
				return setupEmptyDatabase(t)
			},
			expectedBuckets: []string{},
			expectError:    false,
		},
		{
			name: "database with multiple top-level buckets",
			setupFunc: func(t *testing.T) string {
				return setupMultiBucketDatabase(t)
			},
			expectedBuckets: []string{"v1", "metadata", "config", "temp"},
			expectError:    false,
		},
		{
			name: "database with nested buckets",
			setupFunc: func(t *testing.T) string {
				return setupNestedBucketDatabase(t)
			},
			expectedBuckets: []string{"v1", "nested"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setupFunc(t)
			reader, err := NewMetaReader(dbPath)
			if err != nil {
				t.Fatalf("Failed to create reader: %v", err)
			}
			defer reader.Close()

			buckets, err := reader.ListBuckets()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify expected buckets are present
			foundBuckets := make(map[string]bool)
			for _, bucket := range buckets {
				foundBuckets[bucket.Name] = true
				t.Logf("Found bucket: %s with %d keys", bucket.Name, bucket.KeyCount)
			}

			for _, expectedBucket := range tt.expectedBuckets {
				if !foundBuckets[expectedBucket] {
					t.Errorf("Expected to find bucket '%s' but it was not found", expectedBucket)
				}
			}

			// Verify bucket info completeness
			for _, bucket := range buckets {
				if bucket.Name == "" {
					t.Error("Bucket name should not be empty")
				}
				if bucket.KeyCount < 0 {
					t.Errorf("Bucket key count should not be negative, got %d", bucket.KeyCount)
				}
			}
		})
	}
}

func TestMetaReader_ListBucketsErrorCases(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(*testing.T) string
		expectErr bool
		errMsg    string
	}{
		{
			name: "non-existent database",
			setupFunc: func(t *testing.T) string {
				return "/non/existent/path"
			},
			expectErr: true,
			errMsg:    "failed to open bolt database",
		},
		{
			name: "corrupted database",
			setupFunc: func(t *testing.T) string {
				return setupCorruptedDatabase(t)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setupFunc(t)
			reader, err := NewMetaReader(dbPath)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error creating reader: %v", err)
			}
			defer reader.Close()

			_, err = reader.ListBuckets()
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			}
		})
	}
}

func TestMetaReader_ListBucketsWithLargeDataset(t *testing.T) {
	dbPath := setupLargeTestDatabase(t)
	reader, err := NewMetaReader(dbPath)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}
	defer reader.Close()

	buckets, err := reader.ListBuckets()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should handle large datasets without issues
	if len(buckets) == 0 {
		t.Error("Expected to find buckets in large dataset")
	}

	// Verify performance doesn't degrade significantly
	// This is more of a smoke test than a performance test
	t.Logf("Successfully listed %d buckets from large dataset", len(buckets))
}

func TestMetaReader_ListBucketsConcurrentAccess(t *testing.T) {
	dbPath := setupTestDB(t)

	// Test concurrent reads
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			reader, err := NewMetaReader(dbPath)
			if err != nil {
				results <- err
				return
			}
			defer reader.Close()

			_, err = reader.ListBuckets()
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent access failed: %v", err)
		}
	}
}

// Helper functions for test database setup

func setupEmptyDatabase(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create empty database: %v", err)
	}
	defer db.Close()

	return dbPath
}

func setupMultiBucketDatabase(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "multi.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create multi-bucket database: %v", err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		// Create multiple top-level buckets
		buckets := []string{"v1", "metadata", "config", "temp"}

		for _, bucketName := range buckets {
			bkt, err := tx.CreateBucket([]byte(bucketName))
			if err != nil {
				return err
			}

			// Add some keys to each bucket
			for i := 0; i < 5; i++ {
				key := []byte("key" + string(rune('0'+i)))
				value := []byte("value" + string(rune('0'+i)))
				if err := bkt.Put(key, value); err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to setup multi-bucket database: %v", err)
	}

	return dbPath
}

func setupNestedBucketDatabase(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "nested.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create nested database: %v", err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		// Create v1 bucket with nested structure
		v1Bkt, err := tx.CreateBucket([]byte("v1"))
		if err != nil {
			return err
		}

		// Create nested bucket
		nestedBkt, err := tx.CreateBucket([]byte("nested"))
		if err != nil {
			return err
		}

		// Add nested buckets
		_, err = nestedBkt.CreateBucket([]byte("level1"))
		if err != nil {
			return err
		}

		// Add some data to v1
		if err := v1Bkt.Put([]byte("key1"), []byte("value1")); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to setup nested database: %v", err)
	}

	return dbPath
}

func setupCorruptedDatabase(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "corrupted.db")

	// Create a file with invalid content
	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create corrupted database file: %v", err)
	}
	defer file.Close()

	// Write some invalid data
	_, err = file.WriteString("invalid bolt database content")
	if err != nil {
		t.Fatalf("Failed to write corrupted data: %v", err)
	}

	return dbPath
}

func setupLargeTestDatabase(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "large.db")

	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create large database: %v", err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		// Create many buckets with lots of data
		for i := 0; i < 100; i++ {
			bucketName := []byte("bucket" + string(rune('0'+i%10)))
			bkt, err := tx.CreateBucketIfNotExists(bucketName)
			if err != nil {
				return err
			}

			// Add many keys
			for j := 0; j < 1000; j++ {
				key := []byte("key" + string(rune('0'+j%10)) + string(rune('0'+i)))
				value := []byte("value" + string(rune('0'+j%10)) + string(rune('0'+i)))
				if err := bkt.Put(key, value); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Failed to setup large database: %v", err)
	}

	return dbPath
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}