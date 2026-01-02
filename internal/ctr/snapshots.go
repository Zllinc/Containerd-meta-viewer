package ctr

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// SnapshotInfo represents snapshot information from ctr command
type SnapshotInfo struct {
	Key    string
	Parent string
	Kind   string // Active, Committed, View
}

// ListContainerdSnapshots retrieves the list of snapshot keys from containerd
// using the ctr command. It uses the specified namespace and snapshotter.
func ListContainerdSnapshots(namespace, snapshotter string) ([]string, error) {
	snapshots, err := ListContainerdSnapshotsDetailed(namespace, snapshotter)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(snapshots))
	for i, s := range snapshots {
		keys[i] = s.Key
	}
	return keys, nil
}

// ListContainerdSnapshotsDetailed retrieves detailed snapshot info from containerd
// using the ctr command, including parent and kind information.
func ListContainerdSnapshotsDetailed(namespace, snapshotter string) ([]SnapshotInfo, error) {
	args := []string{}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, "snapshots")
	if snapshotter != "" {
		args = append(args, "--snapshotter", snapshotter)
	}
	args = append(args, "ls")

	cmd := exec.Command("ctr", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run ctr snapshots ls: %w, stderr: %s", err, stderr.String())
	}

	var snapshots []SnapshotInfo
	scanner := bufio.NewScanner(&stdout)
	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if first {
			// Skip header line
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			snapshots = append(snapshots, SnapshotInfo{
				Key:    fields[0],
				Parent: fields[1],
				Kind:   fields[2],
			})
		} else if len(fields) == 2 {
			// No parent case
			snapshots = append(snapshots, SnapshotInfo{
				Key:    fields[0],
				Parent: "",
				Kind:   fields[1],
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse ctr output: %w", err)
	}

	return snapshots, nil
}

// ExtractSnapshotID extracts the snapshot ID (sha256:xxx) from a database key.
// Database keys have format like "k8s.io/12/sha256:xxx" or "default/sha256:xxx"
// We need to extract the "sha256:xxx" part for comparison with ctr output.
func ExtractSnapshotID(dbKey string) string {
	// Find the position of "sha256:" in the key
	idx := strings.Index(dbKey, "sha256:")
	if idx >= 0 {
		return dbKey[idx:]
	}
	// If no sha256: prefix, return the part after the last "/"
	lastSlash := strings.LastIndex(dbKey, "/")
	if lastSlash >= 0 && lastSlash < len(dbKey)-1 {
		return dbKey[lastSlash+1:]
	}
	return dbKey
}

// FindOrphanSnapshots compares database snapshots with containerd snapshots
// and returns the keys that exist in database but not in containerd.
// It handles the format difference: DB keys have namespace prefix (k8s.io/12/sha256:xxx)
// while ctr returns just the snapshot ID (sha256:xxx).
func FindOrphanSnapshots(dbKeys, containerdKeys []string) []string {
	containerdSet := make(map[string]struct{}, len(containerdKeys))
	for _, k := range containerdKeys {
		containerdSet[k] = struct{}{}
	}

	var orphans []string
	for _, k := range dbKeys {
		// Extract the sha256 part from DB key for comparison
		snapshotID := ExtractSnapshotID(k)
		if _, exists := containerdSet[snapshotID]; !exists {
			orphans = append(orphans, k) // Return original DB key for cleanup operations
		}
	}
	return orphans
}

// FindUnusedSnapshots finds snapshots that are not referenced as parent by any other
// snapshot and are not active (not being used by a container).
// Returns snapshot keys that can be safely removed.
func FindUnusedSnapshots(snapshots []SnapshotInfo) []string {
	// Build a set of all parents (snapshots that are referenced by others)
	parentSet := make(map[string]struct{})
	for _, s := range snapshots {
		if s.Parent != "" {
			parentSet[s.Parent] = struct{}{}
		}
	}

	// Find unused snapshots:
	// 1. Not in parentSet (no one references it as parent)
	// 2. Kind is not "Active" (no container is using it)
	var unused []string
	for _, s := range snapshots {
		_, isParent := parentSet[s.Key]
		isActive := strings.EqualFold(s.Kind, "Active")

		if !isParent && !isActive {
			unused = append(unused, s.Key)
		}
	}
	return unused
}

// SafeUnusedInfo contains detailed check results for a snapshot
type SafeUnusedInfo struct {
	Key             string   `json:"key"`
	Kind            string   `json:"kind"`
	IsActive        bool     `json:"is_active"`
	IsParent        bool     `json:"is_parent"`
	HasMounts       bool     `json:"has_mounts"`
	MountPaths      []string `json:"mount_paths,omitempty"`
	UsedByContainer bool     `json:"used_by_container"`
	ContainerIDs    []string `json:"container_ids,omitempty"`
	Safe            bool     `json:"safe"`
	Reason          string   `json:"reason,omitempty"`
}

// CheckSnapshotMounts checks if a snapshot has any active mounts
func CheckSnapshotMounts(namespace, snapshotter, key string) (bool, []string, error) {
	args := []string{}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, "snapshots")
	if snapshotter != "" {
		args = append(args, "--snapshotter", snapshotter)
	}
	args = append(args, "mounts", "/tmp/dummy-target", key)

	cmd := exec.Command("ctr", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If the command fails, assume no mounts (snapshot might not exist or no mounts)
		return false, nil, nil
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return false, nil, nil
	}

	// Filter out help messages - real mount output starts with "mount" command
	// ctr snapshots mounts outputs something like: mount -t overlay overlay -o ...
	var mountPaths []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Only consider lines that look like actual mount commands
		if line != "" && strings.HasPrefix(line, "mount ") {
			mountPaths = append(mountPaths, line)
		}
	}

	return len(mountPaths) > 0, mountPaths, nil
}

// CheckSystemMounts checks if any path related to snapshot is mounted using system mount command
func CheckSystemMounts(snapshotKey string) (bool, []string, error) {
	cmd := exec.Command("mount")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, nil, fmt.Errorf("failed to run mount: %w", err)
	}

	var matchedMounts []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if the mount line contains the snapshot key (any part of it)
		if strings.Contains(line, snapshotKey) {
			matchedMounts = append(matchedMounts, line)
		}
	}

	return len(matchedMounts) > 0, matchedMounts, nil
}

// ListContainers lists all containers in the namespace
func ListContainers(namespace string) ([]string, error) {
	args := []string{}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, "containers", "ls", "-q")

	cmd := exec.Command("ctr", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list containers: %w, stderr: %s", err, stderr.String())
	}

	var containers []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		id := strings.TrimSpace(scanner.Text())
		if id != "" {
			containers = append(containers, id)
		}
	}
	return containers, nil
}

// GetContainerSnapshot gets the snapshot key used by a container
func GetContainerSnapshot(namespace, containerID string) (string, error) {
	args := []string{}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, "containers", "info", containerID)

	cmd := exec.Command("ctr", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get container info: %w", err)
	}

	// Parse output to find SnapshotKey
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "SnapshotKey:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "SnapshotKey:")), nil
		}
	}
	return "", nil
}

// FindSafeUnusedSnapshots performs multi-level checks to find truly safe-to-delete snapshots
func FindSafeUnusedSnapshots(namespace, snapshotter string) ([]SafeUnusedInfo, error) {
	// Step 1: Get all snapshots
	snapshots, err := ListContainerdSnapshotsDetailed(namespace, snapshotter)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Build parent set
	parentSet := make(map[string]struct{})
	for _, s := range snapshots {
		if s.Parent != "" {
			parentSet[s.Parent] = struct{}{}
		}
	}

	// Step 2: Get all containers and their snapshots
	containers, _ := ListContainers(namespace)
	containerSnapshots := make(map[string][]string) // snapshot -> container IDs
	for _, cid := range containers {
		snapKey, err := GetContainerSnapshot(namespace, cid)
		if err == nil && snapKey != "" {
			containerSnapshots[snapKey] = append(containerSnapshots[snapKey], cid)
		}
	}

	// Step 3: Check each snapshot
	var results []SafeUnusedInfo
	for _, s := range snapshots {
		info := SafeUnusedInfo{
			Key:      s.Key,
			Kind:     s.Kind,
			IsActive: strings.EqualFold(s.Kind, "Active"),
		}

		// Check 1: Is it a parent?
		_, info.IsParent = parentSet[s.Key]

		// Check 2: Used by container?
		if cids, ok := containerSnapshots[s.Key]; ok && len(cids) > 0 {
			info.UsedByContainer = true
			info.ContainerIDs = cids
		}

		// Check 3: Has mounts via ctr?
		hasCtrMounts, ctrMounts, _ := CheckSnapshotMounts(namespace, snapshotter, s.Key)

		// Check 4: Has system mounts?
		hasSysMounts, sysMounts, _ := CheckSystemMounts(s.Key)

		info.HasMounts = hasCtrMounts || hasSysMounts
		info.MountPaths = append(ctrMounts, sysMounts...)

		// Determine if safe
		if info.IsActive {
			info.Safe = false
			info.Reason = "Active snapshot (in use by container)"
		} else if info.IsParent {
			info.Safe = false
			info.Reason = "Has children snapshots"
		} else if info.UsedByContainer {
			info.Safe = false
			info.Reason = fmt.Sprintf("Used by container(s): %v", info.ContainerIDs)
		} else if info.HasMounts {
			info.Safe = false
			info.Reason = fmt.Sprintf("Has active mounts: %v", info.MountPaths)
		} else {
			info.Safe = true
			info.Reason = "Passed all checks"
		}

		results = append(results, info)
	}

	return results, nil
}

// SnapshotDependency represents a snapshot's dependency information
type SnapshotDependency struct {
	Key              string   `json:"key"`
	Kind             string   `json:"kind"`
	Parent           string   `json:"parent,omitempty"`
	DirectContainers []string `json:"direct_containers,omitempty"` // Containers using this snapshot directly
	AllContainers    []string `json:"all_containers,omitempty"`    // All containers (direct + indirect via children)
	DirectCount      int      `json:"direct_count"`
	TotalCount       int      `json:"total_count"`
	Depth            int      `json:"depth"` // Distance from root (0 = root layer)
}

// AnalyzeSnapshotDependencies analyzes which containers depend on each snapshot
func AnalyzeSnapshotDependencies(namespace, snapshotter string) ([]SnapshotDependency, error) {
	// Step 1: Get all snapshots
	snapshots, err := ListContainerdSnapshotsDetailed(namespace, snapshotter)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Build snapshot map for quick lookup
	snapshotMap := make(map[string]*SnapshotInfo)
	for i := range snapshots {
		snapshotMap[snapshots[i].Key] = &snapshots[i]
	}

	// Step 2: Get all containers and their snapshots
	containers, err := ListContainers(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Map container -> snapshot
	containerSnapshots := make(map[string]string)
	for _, c := range containers {
		snap, err := GetContainerSnapshot(namespace, c)
		if err == nil && snap != "" {
			containerSnapshots[c] = snap
		}
	}

	// Step 3: For each container, trace its snapshot ancestry chain
	// and mark all ancestors as "indirectly" used by this container
	snapshotContainers := make(map[string]map[string]bool) // snapshot -> set of containers
	directContainers := make(map[string]map[string]bool)   // snapshot -> set of direct containers

	for container, snap := range containerSnapshots {
		// Mark direct usage
		if directContainers[snap] == nil {
			directContainers[snap] = make(map[string]bool)
		}
		directContainers[snap][container] = true

		// Trace ancestry and mark indirect usage
		current := snap
		for current != "" {
			if snapshotContainers[current] == nil {
				snapshotContainers[current] = make(map[string]bool)
			}
			snapshotContainers[current][container] = true

			// Move to parent
			if s, exists := snapshotMap[current]; exists && s.Parent != "" {
				current = s.Parent
			} else {
				break
			}
		}
	}

	// Step 4: Calculate depth for each snapshot (distance from root)
	depthMap := make(map[string]int)
	for _, s := range snapshots {
		depth := 0
		current := s.Key
		for {
			if parent, exists := snapshotMap[current]; exists && parent.Parent != "" {
				depth++
				current = parent.Parent
			} else {
				break
			}
		}
		depthMap[s.Key] = depth
	}

	// Step 5: Build results
	results := make([]SnapshotDependency, 0, len(snapshots))
	for _, s := range snapshots {
		dep := SnapshotDependency{
			Key:    s.Key,
			Kind:   s.Kind,
			Parent: s.Parent,
			Depth:  depthMap[s.Key],
		}

		// Direct containers
		if dc, exists := directContainers[s.Key]; exists {
			for c := range dc {
				dep.DirectContainers = append(dep.DirectContainers, c)
			}
			dep.DirectCount = len(dep.DirectContainers)
		}

		// All containers (direct + indirect)
		if ac, exists := snapshotContainers[s.Key]; exists {
			for c := range ac {
				dep.AllContainers = append(dep.AllContainers, c)
			}
			dep.TotalCount = len(dep.AllContainers)
		}

		results = append(results, dep)
	}

	return results, nil
}
