package database

import (
	"time"

	"github.com/containerd/containerd/snapshots"
)

// SnapshotInfo represents metadata for a containerd snapshot
type SnapshotInfo struct {
	Key       string            `json:"key"`
	ID        uint64            `json:"id"`
	Kind      snapshots.Kind    `json:"kind"`
	Parent    string            `json:"parent"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Labels    map[string]string `json:"labels,omitempty"`

	// Usage information
	Inodes int64 `json:"inodes"`
	Size   int64 `json:"size"`

	// Devbox specific fields
	ContentID string `json:"content_id,omitempty"`
	Path      string `json:"path,omitempty"`
}

// DevboxStorageInfo represents devbox-specific storage metadata
type DevboxStorageInfo struct {
	ContentID string `json:"content_id"`
	LvName    string `json:"lv_name"`
	Path      string `json:"path"`
	Status    string `json:"status"`
}

// BucketInfo represents basic information about a bolt bucket
type BucketInfo struct {
	Name     string `json:"name"`
	KeyCount int    `json:"key_count"`
}

// SnapshotKindString converts snapshot kind to human readable string
func SnapshotKindString(kind snapshots.Kind) string {
	switch kind {
	case snapshots.KindActive:
		return "active"
	case snapshots.KindView:
		return "view"
	case snapshots.KindCommitted:
		return "committed"
	default:
		return "unknown"
	}
}