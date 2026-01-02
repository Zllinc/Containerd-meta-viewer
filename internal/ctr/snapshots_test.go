package ctr

import (
	"testing"
)

func TestFindOrphanSnapshots(t *testing.T) {
	tests := []struct {
		name           string
		dbKeys         []string
		containerdKeys []string
		expected       []string
	}{
		{
			name:           "no orphans",
			dbKeys:         []string{"snap1", "snap2"},
			containerdKeys: []string{"snap1", "snap2", "snap3"},
			expected:       nil,
		},
		{
			name:           "some orphans",
			dbKeys:         []string{"snap1", "snap2", "snap3"},
			containerdKeys: []string{"snap1"},
			expected:       []string{"snap2", "snap3"},
		},
		{
			name:           "all orphans",
			dbKeys:         []string{"snap1", "snap2"},
			containerdKeys: []string{},
			expected:       []string{"snap1", "snap2"},
		},
		{
			name:           "empty db",
			dbKeys:         []string{},
			containerdKeys: []string{"snap1", "snap2"},
			expected:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindOrphanSnapshots(tt.dbKeys, tt.containerdKeys)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d orphans, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("expected orphan[%d] = %s, got %s", i, tt.expected[i], v)
				}
			}
		})
	}
}
