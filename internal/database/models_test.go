package database

import (
	"testing"
	"time"

	"github.com/containerd/containerd/snapshots"
)

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
			result := SnapshotKindString(tt.kind)
			if result != tt.expected {
				t.Errorf("SnapshotKindString() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSnapshotInfo(t *testing.T) {
	// Test SnapshotInfo struct creation and field access
	t.Run("create snapshot info", func(t *testing.T) {
		now := time.Now()
		labels := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		info := SnapshotInfo{
			Key:       "test-snapshot-key",
			ID:        123,
			Kind:      snapshots.KindActive,
			Parent:    "parent-snapshot",
			CreatedAt: now,
			UpdatedAt: now,
			Labels:    labels,
			Inodes:    1000,
			Size:      2048,
			ContentID: "test-content-id",
			Path:      "/test/path",
		}

		if info.Key != "test-snapshot-key" {
			t.Errorf("Expected Key = 'test-snapshot-key', got %s", info.Key)
		}

		if info.ID != 123 {
			t.Errorf("Expected ID = 123, got %d", info.ID)
		}

		if info.Kind != snapshots.KindActive {
			t.Errorf("Expected Kind = KindActive, got %v", info.Kind)
		}

		if info.Parent != "parent-snapshot" {
			t.Errorf("Expected Parent = 'parent-snapshot', got %s", info.Parent)
		}

		if info.ContentID != "test-content-id" {
			t.Errorf("Expected ContentID = 'test-content-id', got %s", info.ContentID)
		}

		if info.Path != "/test/path" {
			t.Errorf("Expected Path = '/test/path', got %s", info.Path)
		}

		if info.Inodes != 1000 {
			t.Errorf("Expected Inodes = 1000, got %d", info.Inodes)
		}

		if info.Size != 2048 {
			t.Errorf("Expected Size = 2048, got %d", info.Size)
		}

		if len(info.Labels) != 2 {
			t.Errorf("Expected 2 labels, got %d", len(info.Labels))
		}

		if info.Labels["key1"] != "value1" {
			t.Errorf("Expected Labels['key1'] = 'value1', got %s", info.Labels["key1"])
		}
	})
}

func TestDevboxStorageInfo(t *testing.T) {
	t.Run("create devbox storage info", func(t *testing.T) {
		info := DevboxStorageInfo{
			ContentID: "test-content-id",
			LvName:    "lv-test-volume",
			Path:      "/test/mount/path",
			Status:    "active",
		}

		if info.ContentID != "test-content-id" {
			t.Errorf("Expected ContentID = 'test-content-id', got %s", info.ContentID)
		}

		if info.LvName != "lv-test-volume" {
			t.Errorf("Expected LvName = 'lv-test-volume', got %s", info.LvName)
		}

		if info.Path != "/test/mount/path" {
			t.Errorf("Expected Path = '/test/mount/path', got %s", info.Path)
		}

		if info.Status != "active" {
			t.Errorf("Expected Status = 'active', got %s", info.Status)
		}
	})
}

func TestBucketInfo(t *testing.T) {
	t.Run("create bucket info", func(t *testing.T) {
		info := BucketInfo{
			Name:     "test-bucket",
			KeyCount: 42,
		}

		if info.Name != "test-bucket" {
			t.Errorf("Expected Name = 'test-bucket', got %s", info.Name)
		}

		if info.KeyCount != 42 {
			t.Errorf("Expected KeyCount = 42, got %d", info.KeyCount)
		}
	})
}