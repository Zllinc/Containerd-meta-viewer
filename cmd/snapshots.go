package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/containerd/meta-viewer/internal/ctr"
	"github.com/containerd/meta-viewer/internal/database"
	"github.com/containerd/meta-viewer/internal/formatters"
	"github.com/spf13/cobra"
)

var (
	searchContentID   string
	searchPath        string
	ctrNamespace      string
	ctrSnapshotter    string
	orphanExportFile  string
	unusedExportFile  string
	unusedNamespace   string
	unusedSnapshotter string
	ghostDryRun       bool
	ghostNamespace    string
)

// snapshotsCmd represents the snapshots command
var snapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "Manage and inspect devbox snapshots",
	Long: `View and search devbox snapshots stored in the metadata database.
This command provides access to snapshot information including parent
relationships, usage statistics, and devbox-specific metadata.`,
}

// snapshotsListCmd represents the snapshots list command
var snapshotsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all snapshots",
	Long: `List all snapshots in the devbox metadata database.
This shows basic information about each snapshot including ID, kind,
parent, content ID, and usage statistics.`,
	RunE: runSnapshotsList,
}

// snapshotsGetCmd represents the snapshots get command
var snapshotsGetCmd = &cobra.Command{
	Use:   "get [snapshot-key]",
	Short: "Get detailed information about a specific snapshot",
	Long: `Get detailed information about a specific snapshot by its key.
This shows all available metadata including labels and timestamps.`,
	Args: cobra.ExactArgs(1),
	RunE: runSnapshotsGet,
}

// snapshotsSearchCmd represents the snapshots search command
var snapshotsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search snapshots by content ID or path",
	Long: `Search snapshots by content ID or mount path.
You can specify one or both search criteria to filter snapshots.`,
	RunE: runSnapshotsSearch,
}

func runSnapshotsList(cmd *cobra.Command, args []string) error {
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	snapshots, err := reader.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatSnapshots(snapshots)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatSnapshots(snapshots)
	}
}

func runSnapshotsGet(cmd *cobra.Command, args []string) error {
	snapshotKey := args[0]

	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	snapshot, err := reader.GetSnapshot(snapshotKey)
	if err != nil {
		return fmt.Errorf("failed to get snapshot %s: %w", snapshotKey, err)
	}

	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatSnapshot(snapshot)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatSnapshot(snapshot)
	}
}

func runSnapshotsSearch(cmd *cobra.Command, args []string) error {
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	snapshots, err := reader.SearchSnapshots(searchContentID, searchPath)
	if err != nil {
		return fmt.Errorf("failed to search snapshots: %w", err)
	}

	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatSnapshots(snapshots)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatSnapshots(snapshots)
	}
}

// snapshotsChildrenCmd represents the snapshots children command
var snapshotsChildrenCmd = &cobra.Command{
	Use:   "children [snapshot-id]",
	Short: "Show all children references for a snapshot",
	Long: `Show all children references in the "parents" bucket for a specific snapshot ID.

This helps diagnose why a snapshot cannot be deleted (error: "cannot remove snapshot with child").
It shows both existing children and ghost children (where the child snapshot no longer exists).`,
	Args: cobra.ExactArgs(1),
	RunE: runSnapshotsChildren,
}

func runSnapshotsChildren(cmd *cobra.Command, args []string) error {
	// Parse snapshot ID
	var parentID uint64
	if _, err := fmt.Sscanf(args[0], "%d", &parentID); err != nil {
		return fmt.Errorf("invalid snapshot ID: %s (must be a number)", args[0])
	}

	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	children, err := reader.FindChildrenByParentID(parentID)
	if err != nil {
		return fmt.Errorf("failed to find children: %w", err)
	}

	if len(children) == 0 {
		fmt.Printf("No children references found for snapshot ID %d in parents bucket.\n", parentID)
		return nil
	}

	fmt.Printf("Found %d children reference(s) for snapshot ID %d:\n\n", len(children), parentID)
	for _, c := range children {
		status := "EXISTS"
		if !c.ChildExists {
			status = "GHOST (snapshot not found)"
		}
		fmt.Printf("  Child ID %d: %s [%s]\n", c.ChildID, c.ChildKey, status)
	}

	return nil
}

var parentsDumpFilter uint64

// snapshotsParentsCmd represents the snapshots parents-dump command
var snapshotsParentsCmd = &cobra.Command{
	Use:   "parents-dump [parent-id]",
	Short: "Dump all entries in the parents bucket",
	Long: `Dump all parent-child relationships stored in the parents bucket for debugging.
Optionally filter by a specific parent ID.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSnapshotsParents,
}

func runSnapshotsParents(cmd *cobra.Command, args []string) error {
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	// Parse optional filter by parent ID
	var filterParentID uint64
	if len(args) > 0 {
		parsed, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid parent ID: %s", args[0])
		}
		filterParentID = parsed
	}

	allChildren, err := reader.DumpAllParentLinks()
	if err != nil {
		return fmt.Errorf("failed to dump parents bucket: %w", err)
	}

	// Filter if parent ID specified
	if filterParentID > 0 {
		var filtered []database.ChildInfo
		for _, c := range allChildren {
			if c.ParentID == filterParentID {
				filtered = append(filtered, c)
			}
		}
		allChildren = filtered
	}

	if len(allChildren) == 0 {
		if filterParentID > 0 {
			fmt.Printf("No parent-child links found for parent ID %d.\n", filterParentID)
		} else {
			fmt.Println("Parents bucket is empty.")
		}
		return nil
	}

	fmt.Printf("Found %d parent-child link(s) in parents bucket:\n\n", len(allChildren))

	for _, c := range allChildren {
		exists := "EXISTS"
		if !c.ChildExists {
			exists = "GHOST"
		}
		fmt.Printf("ParentID: %d -> ChildID: %d [%s]\n", c.ParentID, c.ChildID, exists)
		fmt.Printf("  ChildKey: %s\n", c.ChildKey)
	}

	return nil
}

// snapshotsGhostCmd represents the snapshots ghost command
var snapshotsGhostCmd = &cobra.Command{
	Use:   "ghost",
	Short: "Find ghost child references in the database",
	Long: `Find parent links in the database that point to non-existent child snapshots.

This can happen when a child snapshot is deleted but the parent link in the 
"parents" bucket is not cleaned up properly (a bug in the snapshotter).

These ghost references prevent the parent snapshot from being deleted via ctr,
even though the child snapshot no longer exists. The error message would be:
"cannot remove snapshot with child: failed precondition"

Use 'snapshots ghost-cleanup' to remove these stale parent links.`,
	RunE: runSnapshotsGhost,
}

func runSnapshotsGhost(cmd *cobra.Command, args []string) error {
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	ghosts, err := reader.FindGhostChildren()
	if err != nil {
		return fmt.Errorf("failed to find ghost children: %w", err)
	}

	if len(ghosts) == 0 {
		fmt.Println("No ghost child references found.")
		return nil
	}

	fmt.Printf("Found %d ghost child reference(s):\n\n", len(ghosts))

	// Group by parent for better readability
	parentGroups := make(map[uint64][]database.GhostChildInfo)
	for _, g := range ghosts {
		parentGroups[g.ParentID] = append(parentGroups[g.ParentID], g)
	}

	for parentID, children := range parentGroups {
		parentKey := children[0].ParentKey
		if parentKey == "" {
			parentKey = "(unknown)"
		}
		fmt.Printf("Parent ID %d (%s) has %d ghost child(ren):\n", parentID, parentKey, len(children))
		for _, g := range children {
			fmt.Printf("  - Child: %s (ID: %d)\n", g.ChildKey, g.ChildID)
		}
		fmt.Println()
	}

	fmt.Println("These parent snapshots cannot be deleted via ctr because they have")
	fmt.Println("stale child references. Use 'snapshots ghost-cleanup' to fix this.")

	return nil
}

// snapshotsGhostCleanupCmd represents the snapshots ghost-cleanup command
var snapshotsGhostCleanupCmd = &cobra.Command{
	Use:   "ghost-cleanup",
	Short: "Remove ghost child references from the database",
	Long: `Remove stale parent-child links from the database.

These are links where the child snapshot no longer exists but the link remains
in the "parents" bucket. This prevents the parent snapshot from being deleted.

WARNING: Make sure containerd is stopped or the database is not locked before 
running this command.`,
	RunE: runSnapshotsGhostCleanup,
}

func runSnapshotsGhostCleanup(cmd *cobra.Command, args []string) error {
	// First, find all ghost children
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}

	allGhosts, err := reader.FindGhostChildren()
	reader.Close() // Close reader before writing

	if err != nil {
		return fmt.Errorf("failed to find ghost children: %w", err)
	}

	// Filter by namespace if specified
	var ghosts []database.GhostChildInfo
	if ghostNamespace != "" {
		nsPrefix := ghostNamespace + "/"
		for _, g := range allGhosts {
			if strings.HasPrefix(g.ChildKey, nsPrefix) {
				ghosts = append(ghosts, g)
			}
		}
	} else {
		ghosts = allGhosts
	}

	if len(ghosts) == 0 {
		if ghostNamespace != "" {
			fmt.Printf("No ghost child references found for namespace %s.\n", ghostNamespace)
		} else {
			fmt.Println("No ghost child references found. Nothing to clean up.")
		}
		return nil
	}

	fmt.Printf("Found %d ghost child reference(s) to remove.\n\n", len(ghosts))

	// Show what would be deleted
	for _, g := range ghosts {
		parentKey := g.ParentKey
		if parentKey == "" {
			parentKey = "(unknown)"
		}
		fmt.Printf("  Parent ID %d (%s) -> Child: %s (ID: %d)\n", g.ParentID, parentKey, g.ChildKey, g.ChildID)
	}

	if ghostDryRun {
		fmt.Println("\n[DRY-RUN] No changes made. Remove --dry-run to actually clean up.")
		return nil
	}

	fmt.Println()

	// Remove ghost children
	removed, failed, err := database.RemoveGhostChildren(dbPath, ghosts)
	if err != nil {
		return fmt.Errorf("failed to remove ghost children: %w", err)
	}

	fmt.Printf("Cleanup complete: %d removed, %d failed\n", removed, failed)
	return nil
}

// snapshotsOrphanCmd represents the snapshots orphan command
var snapshotsOrphanCmd = &cobra.Command{
	Use:   "orphan",
	Short: "Find orphan snapshots that exist in database but not in containerd",
	Long: `Compare snapshots in the metadata database with the live containerd
snapshots (via ctr command) and list any orphaned entries that exist
in the database but are no longer present in containerd.`,
	RunE: runSnapshotsOrphan,
}

func runSnapshotsOrphan(cmd *cobra.Command, args []string) error {
	// Get snapshots from database
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	dbSnapshots, err := reader.ListSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list database snapshots: %w", err)
	}

	// Filter database snapshots by namespace prefix and extract keys
	nsPrefix := ctrNamespace + "/"
	var dbKeys []string
	var filteredSnapshots []database.SnapshotInfo
	for _, s := range dbSnapshots {
		if strings.HasPrefix(s.Key, nsPrefix) {
			dbKeys = append(dbKeys, s.Key)
			filteredSnapshots = append(filteredSnapshots, s)
		}
	}

	if len(dbKeys) == 0 {
		fmt.Printf("No snapshots found in database for namespace %s\n", ctrNamespace)
		return nil
	}

	// Get snapshots from containerd via ctr
	containerdKeys, err := ctr.ListContainerdSnapshots(ctrNamespace, ctrSnapshotter)
	if err != nil {
		return fmt.Errorf("failed to list containerd snapshots: %w", err)
	}

	// Find orphans
	orphans := ctr.FindOrphanSnapshots(dbKeys, containerdKeys)

	if len(orphans) == 0 {
		fmt.Println("No orphan snapshots found.")
		return nil
	}

	fmt.Printf("Found %d orphan snapshot(s) in database but not in containerd:\n\n", len(orphans))

	// Filter database snapshots to only orphans for detailed output
	var orphanSnapshots []database.SnapshotInfo
	orphanSet := make(map[string]struct{}, len(orphans))
	for _, k := range orphans {
		orphanSet[k] = struct{}{}
	}
	for _, s := range filteredSnapshots {
		if _, exists := orphanSet[s.Key]; exists {
			orphanSnapshots = append(orphanSnapshots, s)
		}
	}

	// Export to file if specified
	if orphanExportFile != "" {
		if err := exportOrphansToFile(orphanExportFile, orphans); err != nil {
			return fmt.Errorf("failed to export orphans to file: %w", err)
		}
		fmt.Printf("Orphan snapshot keys exported to: %s\n", orphanExportFile)
	}

	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatSnapshots(orphanSnapshots)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatSnapshots(orphanSnapshots)
	}
}

func exportOrphansToFile(filename string, keys []string) error {
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal orphan keys: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}

// snapshotsCleanupCmd represents the snapshots cleanup command
var snapshotsCleanupCmd = &cobra.Command{
	Use:   "cleanup [orphan-file]",
	Short: "Clean up orphan snapshots from database",
	Long: `Remove orphan snapshots from the metadata database.
This command reads orphan snapshot keys from a JSON file (generated by 'snapshots orphan --export')
and removes them from the database following the same logic as containerd's Remove function.

WARNING: Make sure containerd is stopped or the database is not locked before running this command.`,
	Args: cobra.ExactArgs(1),
	RunE: runSnapshotsCleanup,
}

func runSnapshotsCleanup(cmd *cobra.Command, args []string) error {
	orphanFile := args[0]

	// Read orphan keys from file
	data, err := os.ReadFile(orphanFile)
	if err != nil {
		return fmt.Errorf("failed to read orphan file %s: %w", orphanFile, err)
	}

	var orphanKeys []string
	if err := json.Unmarshal(data, &orphanKeys); err != nil {
		return fmt.Errorf("failed to parse orphan file: %w", err)
	}

	if len(orphanKeys) == 0 {
		fmt.Println("No orphan keys found in file.")
		return nil
	}

	fmt.Printf("Found %d orphan snapshot(s) to clean up.\n", len(orphanKeys))

	// Remove each orphan snapshot
	var removed, failed int
	for _, key := range orphanKeys {
		if err := database.RemoveOrphanSnapshot(dbPath, key); err != nil {
			fmt.Printf("  [FAILED] %s: %v\n", key, err)
			failed++
		} else {
			fmt.Printf("  [OK] %s\n", key)
			removed++
		}
	}

	fmt.Printf("\nCleanup complete: %d removed, %d failed\n", removed, failed)
	return nil
}

// snapshotsUnusedCmd represents the snapshots unused command
var snapshotsUnusedCmd = &cobra.Command{
	Use:   "unused",
	Short: "Find unused snapshots (not a parent and not active)",
	Long: `Find snapshots that are not being used by any container and are not 
a parent of any other snapshot. These are "leaf" snapshots that can potentially 
be safely removed.

This command queries containerd via ctr to get the real-time parent relationships,
ensuring accurate detection of unused snapshots.

A snapshot is considered unused if:
1. It is not the parent of any other snapshot (checked via ctr)
2. Its kind is "Committed" (not "Active" - active means container is using it)`,
	RunE: runSnapshotsUnused,
}

func runSnapshotsUnused(cmd *cobra.Command, args []string) error {
	if unusedNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}

	// Get snapshots from containerd via ctr (this has real-time parent info)
	ctrSnapshots, err := ctr.ListContainerdSnapshotsDetailed(unusedNamespace, unusedSnapshotter)
	if err != nil {
		return fmt.Errorf("failed to list containerd snapshots: %w", err)
	}

	if len(ctrSnapshots) == 0 {
		fmt.Printf("No snapshots found in containerd for namespace %s\n", unusedNamespace)
		return nil
	}

	// Find unused snapshots using ctr's real-time data
	unusedKeys := ctr.FindUnusedSnapshots(ctrSnapshots)

	if len(unusedKeys) == 0 {
		fmt.Println("No unused snapshots found.")
		return nil
	}

	fmt.Printf("Found %d unused snapshot(s) (not a parent, not active):\n\n", len(unusedKeys))

	// Export to file if specified
	if unusedExportFile != "" {
		if err := exportOrphansToFile(unusedExportFile, unusedKeys); err != nil {
			return fmt.Errorf("failed to export unused snapshots to file: %w", err)
		}
		fmt.Printf("Unused snapshot keys exported to: %s\n", unusedExportFile)
	}

	// Display the unused snapshots
	for _, key := range unusedKeys {
		fmt.Println(key)
	}

	return nil
}

var (
	safeUnusedNamespace   string
	safeUnusedSnapshotter string
	safeUnusedExportFile  string
	safeUnusedOnlySafe    bool
)

// snapshotsSafeUnusedCmd represents the snapshots safe-unused command
var snapshotsSafeUnusedCmd = &cobra.Command{
	Use:   "safe-unused",
	Short: "Find unused snapshots with multi-level safety checks",
	Long: `Find snapshots that are safe to delete with comprehensive verification.

This command performs 4 levels of checks:
1. Kind check - Not "Active" (container rootfs)
2. Parent check - Not referenced as parent by any other snapshot
3. Container check - Not used by any running container (via ctr containers)
4. Mount check - Not mounted on the system (via ctr mounts + system mount)

Only snapshots passing ALL checks are marked as "safe".

Use --only-safe to show only safe-to-delete snapshots.`,
	RunE: runSnapshotsSafeUnused,
}

func runSnapshotsSafeUnused(cmd *cobra.Command, args []string) error {
	if safeUnusedNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}

	fmt.Println("Performing multi-level safety checks...")
	fmt.Println()

	results, err := ctr.FindSafeUnusedSnapshots(safeUnusedNamespace, safeUnusedSnapshotter)
	if err != nil {
		return fmt.Errorf("failed to check snapshots: %w", err)
	}

	if len(results) == 0 {
		fmt.Printf("No snapshots found in namespace %s\n", safeUnusedNamespace)
		return nil
	}

	// Count safe vs unsafe
	var safeCount, unsafeCount int
	var safeKeys []string
	for _, r := range results {
		if r.Safe {
			safeCount++
			safeKeys = append(safeKeys, r.Key)
		} else {
			unsafeCount++
		}
	}

	// Display results
	fmt.Printf("Total snapshots: %d | Safe to delete: %d | In use: %d\n\n", len(results), safeCount, unsafeCount)

	if safeUnusedOnlySafe {
		// Only show safe snapshots
		if safeCount == 0 {
			fmt.Println("No snapshots are safe to delete.")
			return nil
		}
		fmt.Println("Safe to delete (passed all checks):")
		for _, key := range safeKeys {
			fmt.Printf("  ✅ %s\n", key)
		}
	} else {
		// Show all with status
		fmt.Println("Snapshot status:")
		for _, r := range results {
			if r.Safe {
				fmt.Printf("  ✅ SAFE   %s\n", r.Key)
			} else {
				fmt.Printf("  ❌ IN-USE %s\n", r.Key)
				fmt.Printf("           Reason: %s\n", r.Reason)
			}
		}
	}

	// Export if specified
	if safeUnusedExportFile != "" && len(safeKeys) > 0 {
		if err := exportOrphansToFile(safeUnusedExportFile, safeKeys); err != nil {
			return fmt.Errorf("failed to export safe snapshots: %w", err)
		}
		fmt.Printf("\nSafe snapshot keys exported to: %s\n", safeUnusedExportFile)
	}

	// Summary
	fmt.Println()
	if safeUnusedExportFile != "" {
		fmt.Printf("To delete safe snapshots:\n")
		fmt.Printf("  for key in $(cat %s | jq -r '.[]'); do\n", safeUnusedExportFile)
		fmt.Printf("    ctr -n %s snapshots --snapshotter %s rm \"$key\"\n", safeUnusedNamespace, safeUnusedSnapshotter)
		fmt.Printf("  done\n")
		fmt.Println()
		fmt.Printf("Or use the safe-cleanup command:\n")
		fmt.Printf("  containerd-meta-viewer snapshots safe-cleanup --namespace %s --file %s --dry-run\n", safeUnusedNamespace, safeUnusedExportFile)
	} else {
		fmt.Printf("To delete these snapshots, export them first:\n")
		fmt.Printf("  containerd-meta-viewer snapshots safe-unused --namespace %s --export /tmp/safe-snapshots.json --only-safe\n", safeUnusedNamespace)
		fmt.Println()
		fmt.Printf("Then run cleanup:\n")
		fmt.Printf("  containerd-meta-viewer snapshots safe-cleanup --namespace %s --file /tmp/safe-snapshots.json --dry-run\n", safeUnusedNamespace)
	}

	return nil
}

var (
	safeCleanupNamespace   string
	safeCleanupSnapshotter string
	safeCleanupDryRun      bool
	safeCleanupFile        string
)

// snapshotsSafeCleanupCmd represents the snapshots safe-cleanup command
var snapshotsSafeCleanupCmd = &cobra.Command{
	Use:   "safe-cleanup",
	Short: "Clean up snapshots that pass all safety checks",
	Long: `Delete snapshots that have been verified as safe to remove.

This command either:
1. Reads snapshot keys from a JSON file (generated by 'safe-unused --export')
2. Or performs live safety checks and deletes safe snapshots

Before deletion, each snapshot is re-verified with 4-level safety checks:
1. Kind check - Not "Active"
2. Parent check - Not referenced as parent
3. Container check - Not used by any container
4. Mount check - Not mounted

Use --dry-run to preview what would be deleted without making changes.`,
	RunE: runSnapshotsSafeCleanup,
}

func runSnapshotsSafeCleanup(cmd *cobra.Command, args []string) error {
	if safeCleanupNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}

	var keysToDelete []string

	if safeCleanupFile != "" {
		// Read from file
		data, err := os.ReadFile(safeCleanupFile)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", safeCleanupFile, err)
		}
		if err := json.Unmarshal(data, &keysToDelete); err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}
		fmt.Printf("Loaded %d snapshot key(s) from %s\n\n", len(keysToDelete), safeCleanupFile)
	} else {
		// Live check
		fmt.Println("Performing live safety checks...")
		results, err := ctr.FindSafeUnusedSnapshots(safeCleanupNamespace, safeCleanupSnapshotter)
		if err != nil {
			return fmt.Errorf("failed to check snapshots: %w", err)
		}
		for _, r := range results {
			if r.Safe {
				keysToDelete = append(keysToDelete, r.Key)
			}
		}
		fmt.Printf("Found %d safe snapshot(s) to delete\n\n", len(keysToDelete))
	}

	if len(keysToDelete) == 0 {
		fmt.Println("No snapshots to delete.")
		return nil
	}

	// Re-verify each snapshot before deletion
	fmt.Println("Re-verifying snapshots before deletion...")
	fmt.Println()

	var verified []string
	var skipped []string
	for _, key := range keysToDelete {
		// Check 1: Is it Active?
		snapshots, err := ctr.ListContainerdSnapshotsDetailed(safeCleanupNamespace, safeCleanupSnapshotter)
		if err != nil {
			fmt.Printf("  ⚠️  %s - cannot verify, skipping\n", key)
			skipped = append(skipped, key)
			continue
		}

		// Find this snapshot
		var found *ctr.SnapshotInfo
		parentSet := make(map[string]struct{})
		for i, s := range snapshots {
			if s.Parent != "" {
				parentSet[s.Parent] = struct{}{}
			}
			if s.Key == key {
				found = &snapshots[i]
			}
		}

		if found == nil {
			fmt.Printf("  ⚠️  %s - not found, skipping\n", key)
			skipped = append(skipped, key)
			continue
		}

		// Check Active
		if strings.EqualFold(found.Kind, "Active") {
			fmt.Printf("  ❌ %s - now Active, skipping\n", key)
			skipped = append(skipped, key)
			continue
		}

		// Check Parent
		if _, isParent := parentSet[key]; isParent {
			fmt.Printf("  ❌ %s - now has children, skipping\n", key)
			skipped = append(skipped, key)
			continue
		}

		// Check mounts
		hasMounts, _, _ := ctr.CheckSnapshotMounts(safeCleanupNamespace, safeCleanupSnapshotter, key)
		hasSysMounts, _, _ := ctr.CheckSystemMounts(key)
		if hasMounts || hasSysMounts {
			fmt.Printf("  ❌ %s - now has mounts, skipping\n", key)
			skipped = append(skipped, key)
			continue
		}

		fmt.Printf("  ✅ %s - verified safe\n", key)
		verified = append(verified, key)
	}

	fmt.Println()
	fmt.Printf("Verified: %d | Skipped: %d\n\n", len(verified), len(skipped))

	if len(verified) == 0 {
		fmt.Println("No snapshots passed re-verification.")
		return nil
	}

	if safeCleanupDryRun {
		fmt.Println("[DRY-RUN] Would delete the following snapshots:")
		for _, key := range verified {
			fmt.Printf("  - %s\n", key)
		}
		fmt.Println("\nRemove --dry-run to actually delete.")
		return nil
	}

	// Actually delete
	fmt.Println("Deleting snapshots...")
	var deleted, failed int
	for _, key := range verified {
		if err := deleteSnapshot(safeCleanupNamespace, safeCleanupSnapshotter, key); err != nil {
			fmt.Printf("  ❌ FAILED %s: %v\n", key, err)
			failed++
		} else {
			fmt.Printf("  ✅ DELETED %s\n", key)
			deleted++
		}
	}

	fmt.Printf("\nCleanup complete: %d deleted, %d failed\n", deleted, failed)
	return nil
}

func deleteSnapshot(namespace, snapshotter, key string) error {
	args := []string{}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	args = append(args, "snapshots")
	if snapshotter != "" {
		args = append(args, "--snapshotter", snapshotter)
	}
	args = append(args, "rm", key)

	cmd := exec.Command("ctr", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

var (
	depsNamespace   string
	depsSnapshotter string
	depsGroupBy     string
	depsMinCount    int
)

// snapshotsDepsCmd represents the snapshots deps command
var snapshotsDepsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Analyze snapshot dependencies by containers",
	Long: `Analyze which containers depend on each snapshot (directly or indirectly).

This helps understand:
- Which base layers are shared by many containers
- Which snapshots are unused (0 containers)
- The dependency tree depth

Examples:
  # Show all snapshots with container dependencies
  containerd-meta-viewer snapshots deps --namespace k8s.io

  # Group by container count
  containerd-meta-viewer snapshots deps --namespace k8s.io --group-by count

  # Show only snapshots used by at least 5 containers
  containerd-meta-viewer snapshots deps --namespace k8s.io --min-count 5
`,
	RunE: runSnapshotsDeps,
}

func runSnapshotsDeps(cmd *cobra.Command, args []string) error {
	if depsNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}

	fmt.Println("Analyzing snapshot dependencies...")
	deps, err := ctr.AnalyzeSnapshotDependencies(depsNamespace, depsSnapshotter)
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Calculate statistics
	var (
		totalSnapshots    = len(deps)
		unusedCount       = 0
		directUsedCount   = 0
		indirectOnlyCount = 0
		maxContainers     = 0
		countDistribution = make(map[int]int) // count -> number of snapshots with that count
	)

	for _, d := range deps {
		countDistribution[d.TotalCount]++
		if d.TotalCount == 0 {
			unusedCount++
		} else if d.DirectCount > 0 {
			directUsedCount++
		} else {
			indirectOnlyCount++
		}
		if d.TotalCount > maxContainers {
			maxContainers = d.TotalCount
		}
	}

	// Print summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Printf("Total snapshots: %d\n", totalSnapshots)
	fmt.Printf("  Unused (0 containers):       %d\n", unusedCount)
	fmt.Printf("  Direct container usage:      %d\n", directUsedCount)
	fmt.Printf("  Indirect only (base layers): %d\n", indirectOnlyCount)
	fmt.Printf("  Max containers per snapshot: %d\n", maxContainers)

	// Print distribution
	fmt.Println()
	fmt.Println("=== Distribution by Container Count ===")
	fmt.Printf("%-20s %s\n", "Container Count", "Snapshot Count")
	fmt.Printf("%-20s %s\n", strings.Repeat("-", 15), strings.Repeat("-", 15))

	// Sort keys for display
	var counts []int
	for c := range countDistribution {
		counts = append(counts, c)
	}
	sort.Ints(counts)

	for _, c := range counts {
		if depsMinCount > 0 && c < depsMinCount {
			continue
		}
		bar := strings.Repeat("█", min(countDistribution[c]/10+1, 50))
		fmt.Printf("%-20d %-10d %s\n", c, countDistribution[c], bar)
	}

	// Detail output based on groupBy
	if depsGroupBy == "count" {
		fmt.Println()
		fmt.Println("=== Snapshots Grouped by Container Count ===")

		// Group snapshots by count
		grouped := make(map[int][]ctr.SnapshotDependency)
		for _, d := range deps {
			if depsMinCount > 0 && d.TotalCount < depsMinCount {
				continue
			}
			grouped[d.TotalCount] = append(grouped[d.TotalCount], d)
		}

		// Print in reverse order (most used first)
		for i := len(counts) - 1; i >= 0; i-- {
			c := counts[i]
			if depsMinCount > 0 && c < depsMinCount {
				continue
			}
			snapshots := grouped[c]
			if len(snapshots) == 0 {
				continue
			}

			fmt.Printf("\n--- %d containers (%d snapshots) ---\n", c, len(snapshots))
			for _, d := range snapshots {
				shortKey := d.Key
				if len(shortKey) > 20 {
					shortKey = shortKey[:20] + "..."
				}
				fmt.Printf("  %s (depth=%d, direct=%d)\n", shortKey, d.Depth, d.DirectCount)
			}
		}
	} else if depsGroupBy == "depth" {
		fmt.Println()
		fmt.Println("=== Snapshots Grouped by Depth ===")

		// Group snapshots by depth
		grouped := make(map[int][]ctr.SnapshotDependency)
		maxDepth := 0
		for _, d := range deps {
			grouped[d.Depth] = append(grouped[d.Depth], d)
			if d.Depth > maxDepth {
				maxDepth = d.Depth
			}
		}

		for depth := 0; depth <= maxDepth; depth++ {
			snapshots := grouped[depth]
			if len(snapshots) == 0 {
				continue
			}

			// Calculate stats for this depth
			var totalContainers, avgContainers int
			for _, d := range snapshots {
				totalContainers += d.TotalCount
			}
			if len(snapshots) > 0 {
				avgContainers = totalContainers / len(snapshots)
			}

			fmt.Printf("\nDepth %d: %d snapshots (avg %d containers each)\n", depth, len(snapshots), avgContainers)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(snapshotsCmd)
	snapshotsCmd.AddCommand(snapshotsListCmd)
	snapshotsCmd.AddCommand(snapshotsGetCmd)
	snapshotsCmd.AddCommand(snapshotsSearchCmd)
	snapshotsCmd.AddCommand(snapshotsOrphanCmd)
	snapshotsCmd.AddCommand(snapshotsCleanupCmd)
	snapshotsCmd.AddCommand(snapshotsUnusedCmd)
	snapshotsCmd.AddCommand(snapshotsSafeUnusedCmd)
	snapshotsCmd.AddCommand(snapshotsSafeCleanupCmd)
	snapshotsCmd.AddCommand(snapshotsGhostCmd)
	snapshotsCmd.AddCommand(snapshotsGhostCleanupCmd)
	snapshotsCmd.AddCommand(snapshotsChildrenCmd)
	snapshotsCmd.AddCommand(snapshotsParentsCmd)
	snapshotsCmd.AddCommand(snapshotsDepsCmd)

	// Add flags to search command
	snapshotsSearchCmd.Flags().StringVar(&searchContentID, "content-id", "", "Search by content ID")
	snapshotsSearchCmd.Flags().StringVar(&searchPath, "path", "", "Search by mount path")

	// Add flags to orphan command
	snapshotsOrphanCmd.Flags().StringVar(&ctrNamespace, "namespace", "k8s.io", "Containerd namespace for ctr command")
	snapshotsOrphanCmd.Flags().StringVar(&ctrSnapshotter, "snapshotter", "devbox", "Snapshotter name for ctr command")
	snapshotsOrphanCmd.Flags().StringVar(&orphanExportFile, "export", "", "Export orphan snapshot keys to a JSON file")

	// Add flags to unused command
	snapshotsUnusedCmd.Flags().StringVar(&unusedNamespace, "namespace", "", "Containerd namespace (e.g., default, k8s.io)")
	snapshotsUnusedCmd.Flags().StringVar(&unusedSnapshotter, "snapshotter", "devbox", "Snapshotter name for ctr command")
	snapshotsUnusedCmd.Flags().StringVar(&unusedExportFile, "export", "", "Export unused snapshot keys to a JSON file")

	// Add flags to safe-unused command
	snapshotsSafeUnusedCmd.Flags().StringVar(&safeUnusedNamespace, "namespace", "", "Containerd namespace (e.g., default, k8s.io)")
	snapshotsSafeUnusedCmd.Flags().StringVar(&safeUnusedSnapshotter, "snapshotter", "devbox", "Snapshotter name")
	snapshotsSafeUnusedCmd.Flags().StringVar(&safeUnusedExportFile, "export", "", "Export safe snapshot keys to a JSON file")
	snapshotsSafeUnusedCmd.Flags().BoolVar(&safeUnusedOnlySafe, "only-safe", false, "Only show snapshots that are safe to delete")

	// Add flags to safe-cleanup command
	snapshotsSafeCleanupCmd.Flags().StringVar(&safeCleanupNamespace, "namespace", "", "Containerd namespace (e.g., default, k8s.io)")
	snapshotsSafeCleanupCmd.Flags().StringVar(&safeCleanupSnapshotter, "snapshotter", "devbox", "Snapshotter name")
	snapshotsSafeCleanupCmd.Flags().StringVar(&safeCleanupFile, "file", "", "Read snapshot keys from JSON file (from safe-unused --export)")
	snapshotsSafeCleanupCmd.Flags().BoolVar(&safeCleanupDryRun, "dry-run", false, "Show what would be deleted without making changes")

	// Add flags to ghost-cleanup command
	snapshotsGhostCleanupCmd.Flags().BoolVar(&ghostDryRun, "dry-run", false, "Show what would be deleted without making changes")
	snapshotsGhostCleanupCmd.Flags().StringVar(&ghostNamespace, "namespace", "", "Filter by namespace (e.g., k8s.io, default)")

	// Add flags to deps command
	snapshotsDepsCmd.Flags().StringVar(&depsNamespace, "namespace", "", "Containerd namespace (e.g., k8s.io)")
	snapshotsDepsCmd.Flags().StringVar(&depsSnapshotter, "snapshotter", "devbox", "Snapshotter name")
	snapshotsDepsCmd.Flags().StringVar(&depsGroupBy, "group-by", "", "Group output by: count, depth")
	snapshotsDepsCmd.Flags().IntVar(&depsMinCount, "min-count", 0, "Only show snapshots used by at least N containers")
}
