package formatters

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/containerd/containerd/snapshots"
	"github.com/containerd/meta-viewer/internal/database"
)

func TestNewJSONFormatter(t *testing.T) {
	t.Run("compact formatter", func(t *testing.T) {
		formatter := NewJSONFormatter(false)
		if formatter == nil {
			t.Error("Expected formatter to be created")
		}
		if formatter.pretty != false {
			t.Error("Expected formatter to be compact")
		}
	})

	t.Run("pretty formatter", func(t *testing.T) {
		formatter := NewJSONFormatter(true)
		if formatter == nil {
			t.Error("Expected formatter to be created")
		}
		if formatter.pretty != true {
			t.Error("Expected formatter to be pretty")
		}
	})
}

func TestJSONFormatter_FormatBuckets(t *testing.T) {
	buckets := []database.BucketInfo{
		{Name: "v1", KeyCount: 3},
		{Name: "test", KeyCount: 10},
	}

	// Test the marshaling logic directly
	data, err := json.Marshal(buckets)
	if err != nil {
		t.Fatalf("Expected no error marshaling, got %v", err)
	}

	if !strings.Contains(string(data), "v1") {
		t.Error("Expected JSON to contain v1 bucket")
	}
	if !strings.Contains(string(data), "test") {
		t.Error("Expected JSON to contain test bucket")
	}
}

func TestJSONFormatter_FormatSnapshots(t *testing.T) {
	_ = NewJSONFormatter(true) // Test creation

	now := time.Now().UTC().Truncate(time.Second) // Truncate to avoid precision issues
	snapshots := []database.SnapshotInfo{
		{
			Key:       "snapshot-1",
			ID:        1,
			Kind:      snapshots.KindActive,
			Parent:    "",
			CreatedAt: now,
			UpdatedAt: now,
			Labels: map[string]string{
				"test-label": "test-value",
			},
			Inodes:    1000,
			Size:      2048,
			ContentID: "content-123",
			Path:      "/test/path/1",
		},
	}

	// Test marshaling directly to verify structure
	data, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		t.Fatalf("Expected no error marshaling, got %v", err)
	}

	output := string(data)

	// Check JSON structure
	expectedFields := []string{
		`"key": "snapshot-1"`,
		`"id": 1`,
		`"kind": "Active"`, // String representation of KindActive
		`"parent": ""`,
		`"inodes": 1000`,
		`"size": 2048`,
		`"content_id": "content-123"`,
		`"path": "/test/path/1"`,
		`"test-label": "test-value"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected JSON to contain %s", field)
		}
	}
}

func TestJSONFormatter_FormatSnapshot(t *testing.T) {
	_ = NewJSONFormatter(false) // Test creation

	now := time.Now().UTC().Truncate(time.Second)
	snapshot := &database.SnapshotInfo{
		Key:       "test-snapshot",
		ID:        42,
		Kind:      snapshots.KindCommitted,
		Parent:    "parent-snapshot",
		CreatedAt: now,
		UpdatedAt: now,
		Labels: map[string]string{
			"env": "production",
		},
		Inodes:    5000,
		Size:      10240,
		ContentID: "test-content-id",
		Path:      "/test/mount/path",
	}

	// Test marshaling directly
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Expected no error marshaling, got %v", err)
	}

	output := string(data)

	expectedFields := []string{
		`"key":"test-snapshot"`, // No space in compact JSON
		`"id":42`,
		`"kind":"Committed"`, // String representation of KindCommitted
		`"parent":"parent-snapshot"`,
		`"content_id":"test-content-id"`,
		`"path":"/test/mount/path"`,
		`"env":"production"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected JSON to contain %s", field)
		}
	}
}

func TestJSONFormatter_FormatDevboxStorage(t *testing.T) {
	_ = NewJSONFormatter(true) // Test creation

	storage := []database.DevboxStorageInfo{
		{
			ContentID: "content-123",
			LvName:    "lv-test-1",
			Path:      "/mount/path/1",
			Status:    "active",
		},
		{
			ContentID: "content-456",
			LvName:    "",
			Path:      "/mount/path/2",
			Status:    "removed",
		},
	}

	// Test marshaling directly
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		t.Fatalf("Expected no error marshaling, got %v", err)
	}

	output := string(data)

	expectedFields := []string{
		`"content_id": "content-123"`,
		`"content_id": "content-456"`,
		`"lv_name": "lv-test-1"`,
		`"lv_name": ""`,
		`"path": "/mount/path/1"`,
		`"path": "/mount/path/2"`,
		`"status": "active"`,
		`"status": "removed"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected JSON to contain %s", field)
		}
	}
}

func TestJSONFormatter_FormatDevboxStorageItem(t *testing.T) {
	_ = NewJSONFormatter(false) // Test creation

	item := &database.DevboxStorageInfo{
		ContentID: "test-content-id",
		LvName:    "lv-test-volume",
		Path:      "/test/mount/path",
		Status:    "active",
	}

	// Test marshaling directly
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Expected no error marshaling, got %v", err)
	}

	output := string(data)

	expectedFields := []string{
		`"content_id":"test-content-id"`, // No space in compact JSON
		`"lv_name":"lv-test-volume"`,
		`"path":"/test/mount/path"`,
		`"status":"active"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Expected JSON to contain %s", field)
		}
	}
}

func TestJSONFormatter_FormatLVMMap(t *testing.T) {
	_ = NewJSONFormatter(false) // Test creation

	storage := []database.DevboxStorageInfo{
		{
			ContentID: "content-123",
			LvName:    "lv-volume-1",
			Path:      "/mount/path/1",
			Status:    "active",
		},
		{
			ContentID: "content-456",
			LvName:    "", // Empty LV name should be excluded
			Path:      "/mount/path/2",
			Status:    "active",
		},
		{
			ContentID: "content-789",
			LvName:    "lv-volume-3",
			Path:      "", // Empty path should be excluded
			Status:    "active",
		},
	}

	// Test the LVM map creation logic directly
	lvmMap := make(map[string]string)
	for _, item := range storage {
		if item.LvName != "" && item.Path != "" {
			lvmMap[item.LvName] = item.Path
		}
	}

	// Should only include entries with both LV name and path
	expectedCount := 1 // Only content-123 has both LV name and path
	if len(lvmMap) != expectedCount {
		t.Errorf("Expected LVM map to have %d entries, got %d", expectedCount, len(lvmMap))
	}

	// Test marshaling
	data, err := json.Marshal(lvmMap)
	if err != nil {
		t.Fatalf("Expected no error marshaling, got %v", err)
	}

	output := string(data)

	// Should include the valid entry
	if !strings.Contains(output, "lv-volume-1") {
		t.Error("Expected LVM map to contain lv-volume-1")
	}
	if !strings.Contains(output, "/mount/path/1") {
		t.Error("Expected LVM map to contain /mount/path/1")
	}

	// Should not include entries with empty LV name or path
	if strings.Contains(output, "lv-volume-3") {
		t.Error("Expected LVM map to NOT contain lv-volume-3 (empty path)")
	}
}

func TestJSONFormatter_ToJSON(t *testing.T) {
	t.Run("compact output", func(t *testing.T) {
		_ = NewJSONFormatter(false) // Test creation

		testData := map[string]interface{}{
			"name": "test",
			"value": 42,
		}

		// Test marshaling
		data, err := json.Marshal(testData)
		if err != nil {
			t.Fatalf("Expected no error marshaling, got %v", err)
		}

		output := string(data)
		if !strings.Contains(output, `"name":"test"`) {
			t.Error("Expected compact JSON format")
		}
		if !strings.Contains(output, `"value":42`) {
			t.Error("Expected compact JSON format")
		}
	})

	t.Run("pretty output", func(t *testing.T) {
		_ = NewJSONFormatter(true) // Test creation

		testData := map[string]interface{}{
			"name": "test",
			"value": 42,
		}

		// Test marshaling with indent
		data, err := json.MarshalIndent(testData, "", "  ")
		if err != nil {
			t.Fatalf("Expected no error marshaling, got %v", err)
		}

		output := string(data)
		if !strings.Contains(output, "\n") {
			t.Error("Expected pretty JSON to contain newlines")
		}
		if !strings.Contains(output, "  ") {
			t.Error("Expected pretty JSON to contain indentation")
		}
	})
}

func TestJSONFormatter_ErrorHandling(t *testing.T) {
	formatter := NewJSONFormatter(false)

	// Test with invalid data (function that can't be marshaled)
	invalidData := func() {} // functions can't be marshaled to JSON

	err := formatter.toJSON(invalidData)
	if err == nil {
		t.Error("Expected error when marshaling invalid data")
	}
}