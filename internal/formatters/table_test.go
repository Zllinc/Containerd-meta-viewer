package formatters

import (
	"testing"
	"time"

	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/meta-viewer/internal/database"
)

func TestNewTableFormatter(t *testing.T) {
	formatter := NewTableFormatter()
	if formatter == nil {
		t.Error("Expected formatter to be created")
	}
	if formatter.writer == nil {
		t.Error("Expected writer to be initialized")
	}
}

func TestTableFormatter_DataValidation(t *testing.T) {
	// Test bucket data validation
	t.Run("bucket data", func(t *testing.T) {
		buckets := []database.BucketInfo{
			{Name: "v1", KeyCount: 3},
			{Name: "test", KeyCount: 10},
		}

		if len(buckets) != 2 {
			t.Errorf("Expected 2 buckets, got %d", len(buckets))
		}

		// Verify bucket names
		expectedNames := []string{"v1", "test"}
		for i, bucket := range buckets {
			if bucket.Name != expectedNames[i] {
				t.Errorf("Expected bucket %d name = %s, got %s", i, expectedNames[i], bucket.Name)
			}
		}

		// Verify key counts
		expectedCounts := []int{3, 10}
		for i, bucket := range buckets {
			if bucket.KeyCount != expectedCounts[i] {
				t.Errorf("Expected bucket %d key count = %d, got %d", i, expectedCounts[i], bucket.KeyCount)
			}
		}
	})

	// Test snapshot data validation
	t.Run("snapshot data", func(t *testing.T) {
		now := time.Now()
		snapshotList := []database.SnapshotInfo{
			{
				Key:       "snapshot-1",
				ID:        1,
				Kind:      snapshots.KindActive,
				Parent:    "",
				CreatedAt: now,
				Inodes:    1000,
				Size:      2048,
				ContentID: "content-123",
				Path:      "/test/path/1",
			},
			{
				Key:       "snapshot-2",
				ID:        2,
				Kind:      snapshots.KindCommitted,
				Parent:    "snapshot-1",
				CreatedAt: now,
				Inodes:    1500,
				Size:      3072,
				ContentID: "content-456",
				Path:      "/test/path/2",
			},
		}

		if len(snapshotList) != 2 {
			t.Errorf("Expected 2 snapshots, got %d", len(snapshotList))
		}

		// Verify first snapshot
		if snapshotList[0].Key != "snapshot-1" {
			t.Errorf("Expected first snapshot key = snapshot-1, got %s", snapshotList[0].Key)
		}
		if snapshotList[0].ID != 1 {
			t.Errorf("Expected first snapshot ID = 1, got %d", snapshotList[0].ID)
		}
		if snapshotList[0].Kind != snapshots.KindActive {
			t.Errorf("Expected first snapshot kind = KindActive, got %v", snapshotList[0].Kind)
		}
		if snapshotList[0].ContentID != "content-123" {
			t.Errorf("Expected first snapshot content ID = content-123, got %s", snapshotList[0].ContentID)
		}

		// Verify second snapshot
		if snapshotList[1].Key != "snapshot-2" {
			t.Errorf("Expected second snapshot key = snapshot-2, got %s", snapshotList[1].Key)
		}
		if snapshotList[1].ID != 2 {
			t.Errorf("Expected second snapshot ID = 2, got %d", snapshotList[1].ID)
		}
		if snapshotList[1].Kind != snapshots.KindCommitted {
			t.Errorf("Expected second snapshot kind = KindCommitted, got %v", snapshotList[1].Kind)
		}
		if snapshotList[1].Parent != "snapshot-1" {
			t.Errorf("Expected second snapshot parent = snapshot-1, got %s", snapshotList[1].Parent)
		}
	})

	// Test devbox storage data validation
	t.Run("devbox storage data", func(t *testing.T) {
		storage := []database.DevboxStorageInfo{
			{
				ContentID: "content-123",
				LvName:    "lv-test-1",
				Path:      "/mount/path/1",
				Status:    "active",
			},
			{
				ContentID: "content-456",
				LvName:    "", // Empty LV name
				Path:      "/mount/path/2",
				Status:    "removed",
			},
		}

		if len(storage) != 2 {
			t.Errorf("Expected 2 storage entries, got %d", len(storage))
		}

		// Verify first storage entry
		if storage[0].ContentID != "content-123" {
			t.Errorf("Expected first storage content ID = content-123, got %s", storage[0].ContentID)
		}
		if storage[0].LvName != "lv-test-1" {
			t.Errorf("Expected first storage LV name = lv-test-1, got %s", storage[0].LvName)
		}
		if storage[0].Path != "/mount/path/1" {
			t.Errorf("Expected first storage path = /mount/path/1, got %s", storage[0].Path)
		}
		if storage[0].Status != "active" {
			t.Errorf("Expected first storage status = active, got %s", storage[0].Status)
		}

		// Verify second storage entry
		if storage[1].ContentID != "content-456" {
			t.Errorf("Expected second storage content ID = content-456, got %s", storage[1].ContentID)
		}
		if storage[1].LvName != "" {
			t.Errorf("Expected second storage LV name to be empty, got %s", storage[1].LvName)
		}
		if storage[1].Status != "removed" {
			t.Errorf("Expected second storage status = removed, got %s", storage[1].Status)
		}
	})
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "exact length",
			input:    "exact",
			maxLen:   5,
			expected: "exact",
		},
		{
			name:     "long string",
			input:    "this is a very long string",
			maxLen:   10,
			expected: "this is...",
		},
		{
			name:     "very short max length",
			input:    "testing",
			maxLen:   3,
			expected: "te.",
		},
		{
			name:     "max length 1",
			input:    "test",
			maxLen:   1,
			expected: "t",
		},
		{
			name:     "max length 2",
			input:    "test",
			maxLen:   2,
			expected: "t.",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "max length 0",
			input:    "test",
			maxLen:   0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestSnapshotKindString(t *testing.T) {
	tests := []struct {
		name     string
		kind     snapshots.Kind
		expected string
	}{
		{
			name:     "active snapshot",
			kind:     snapshots.KindActive,
			expected: "active",
		},
		{
			name:     "view snapshot",
			kind:     snapshots.KindView,
			expected: "view",
		},
		{
			name:     "committed snapshot",
			kind:     snapshots.KindCommitted,
			expected: "committed",
		},
		{
			name:     "unknown snapshot kind",
			kind:     snapshots.Kind(99),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := database.SnapshotKindString(tt.kind)
			if result != tt.expected {
				t.Errorf("SnapshotKindString(%v) = %s, expected %s", tt.kind, result, tt.expected)
			}
		})
	}
}