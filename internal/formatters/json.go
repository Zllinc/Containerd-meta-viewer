package formatters

import (
	"encoding/json"
	"fmt"

	"github.com/containerd/meta-viewer/internal/database"
)

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	pretty bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(pretty bool) *JSONFormatter {
	return &JSONFormatter{pretty: pretty}
}

// FormatBuckets formats bucket information as JSON
func (f *JSONFormatter) FormatBuckets(buckets []database.BucketInfo) error {
	return f.toJSON(buckets)
}

// FormatSnapshots formats snapshot information as JSON
func (f *JSONFormatter) FormatSnapshots(snapshots []database.SnapshotInfo) error {
	return f.toJSON(snapshots)
}

// FormatSnapshot formats a single snapshot as JSON
func (f *JSONFormatter) FormatSnapshot(snapshot *database.SnapshotInfo) error {
	return f.toJSON(snapshot)
}

// FormatDevboxStorage formats devbox storage information as JSON
func (f *JSONFormatter) FormatDevboxStorage(storage []database.DevboxStorageInfo) error {
	return f.toJSON(storage)
}

// FormatDevboxStorageItem formats a single devbox storage item as JSON
func (f *JSONFormatter) FormatDevboxStorageItem(item *database.DevboxStorageInfo) error {
	return f.toJSON(item)
}

// FormatLVMMap formats LVM mapping information as JSON
func (f *JSONFormatter) FormatLVMMap(storage []database.DevboxStorageInfo) error {
	// Filter and create LVM map
	lvmMap := make(map[string]string)
	for _, item := range storage {
		if item.LvName != "" && item.Path != "" {
			lvmMap[item.LvName] = item.Path
		}
	}
	return f.toJSON(lvmMap)
}

// toJSON marshals data to JSON with optional pretty printing
func (f *JSONFormatter) toJSON(data interface{}) error {
	var output []byte
	var err error

	if f.pretty {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(output))
	return nil
}