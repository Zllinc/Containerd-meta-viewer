package formatters

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/containerd/meta-viewer/internal/database"
)

// TableFormatter formats output as tables
type TableFormatter struct {
	writer *tabwriter.Writer
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter() *TableFormatter {
	return &TableFormatter{
		writer: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
	}
}

// FormatBuckets formats bucket information as a table
func (f *TableFormatter) FormatBuckets(buckets []database.BucketInfo) error {
	fmt.Fprintln(f.writer, "NAME\tKEYS")
	for _, bucket := range buckets {
		fmt.Fprintf(f.writer, "%s\t%d\n", bucket.Name, bucket.KeyCount)
	}
	return f.writer.Flush()
}

// FormatSnapshots formats snapshot information as a table
func (f *TableFormatter) FormatSnapshots(snapshots []database.SnapshotInfo) error {
	fmt.Fprintln(f.writer, "ID\tKEY\tKIND\tPARENT\tCONTENT_ID\tPATH\tINODES\tSIZE\tCREATED")
	for _, snapshot := range snapshots {
		created := snapshot.CreatedAt.Format("2006-01-02 15:04:05")
		parent := snapshot.Parent
		if parent == "" {
			parent = "-"
		}
		contentID := snapshot.ContentID
		if contentID == "" {
			contentID = "-"
		}
		path := snapshot.Path
		if path == "" {
			path = "-"
		}

		fmt.Fprintf(f.writer, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
			snapshot.ID,
			truncateString(snapshot.Key, 12),
			database.SnapshotKindString(snapshot.Kind),
			parent,
			truncateString(contentID, 12),
			truncateString(path, 20),
			snapshot.Inodes,
			snapshot.Size,
			created)
	}
	return f.writer.Flush()
}

// FormatSnapshot formats a single snapshot as detailed information
func (f *TableFormatter) FormatSnapshot(snapshot *database.SnapshotInfo) error {
	fmt.Printf("Snapshot Information:\n")
	fmt.Printf("====================\n")
	fmt.Printf("ID:       %d\n", snapshot.ID)
	fmt.Printf("Key:      %s\n", snapshot.Key)
	fmt.Printf("Kind:     %s\n", database.SnapshotKindString(snapshot.Kind))
	fmt.Printf("Parent:   %s\n", snapshot.Parent)
	fmt.Printf("Created:  %s\n", snapshot.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:  %s\n", snapshot.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Inodes:   %d\n", snapshot.Inodes)
	fmt.Printf("Size:     %d bytes\n", snapshot.Size)

	if snapshot.ContentID != "" {
		fmt.Printf("ContentID: %s\n", snapshot.ContentID)
	}

	if snapshot.Path != "" {
		fmt.Printf("Path:      %s\n", snapshot.Path)
	}

	if len(snapshot.Labels) > 0 {
		fmt.Printf("\nLabels:\n")
		for k, v := range snapshot.Labels {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}

	return nil
}

// FormatDevboxStorage formats devbox storage information as a table
func (f *TableFormatter) FormatDevboxStorage(storage []database.DevboxStorageInfo) error {
	fmt.Fprintln(f.writer, "CONTENT_ID\tLV_NAME\tPATH\tSTATUS\tSNAPSHOT_KEY")
	for _, item := range storage {
		lvName := item.LvName
		if lvName == "" {
			lvName = "-"
		}
		path := item.Path
		if path == "" {
			path = "-"
		}
		status := item.Status
		if status == "" {
			status = "unknown"
		}
		snapshotKey := item.SnapshotKey
		if snapshotKey == "" {
			snapshotKey = "-"
		}

		fmt.Fprintf(f.writer, "%s\t%s\t%s\t%s\t%s\n",
			truncateString(item.ContentID, 12),
			lvName,
			truncateString(path, 30),
			status,
			truncateString(snapshotKey, 40))
	}
	return f.writer.Flush()
}

// FormatDevboxStorageItem formats a single devbox storage item as detailed information
func (f *TableFormatter) FormatDevboxStorageItem(item *database.DevboxStorageInfo) error {
	fmt.Printf("Devbox Storage Information:\n")
	fmt.Printf("==========================\n")
	fmt.Printf("ContentID:   %s\n", item.ContentID)
	fmt.Printf("LV Name:     %s\n", item.LvName)
	fmt.Printf("Path:        %s\n", item.Path)
	fmt.Printf("Status:      %s\n", item.Status)
	if item.SnapshotKey != "" {
		fmt.Printf("Snapshot Key: %s\n", item.SnapshotKey)
	}
	return nil
}

// FormatLVMMap formats LVM mapping information as a table
func (f *TableFormatter) FormatLVMMap(storage []database.DevboxStorageInfo) error {
	fmt.Fprintln(f.writer, "LV_NAME\tPATH")
	for _, item := range storage {
		if item.LvName != "" && item.Path != "" {
			fmt.Fprintf(f.writer, "%s\t%s\n",
				item.LvName,
				truncateString(item.Path, 50))
		}
	}
	return f.writer.Flush()
}

// TruncateString truncates a string to the specified length
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		if maxLen <= 1 {
			return s[:1]
		}
		return s[:maxLen-1] + "."
	}
	return s[:maxLen-3] + "..."
}

// truncateString truncates a string to the specified length (internal function)
func truncateString(s string, maxLen int) string {
	return TruncateString(s, maxLen)
}
