package cmd

import (
	"fmt"
	"strings"

	"github.com/containerd/meta-viewer/internal/database"
	"github.com/spf13/cobra"
)

var (
	containerdMetaDBPath string
	ctrdNamespace        string
	ctrdSnapshotter      string
)

// containerdCmd represents the containerd metadata command group
var containerdCmd = &cobra.Command{
	Use:   "containerd",
	Short: "Inspect containerd's core metadata database",
	Long: `Inspect containerd's core metadata database (meta.db).

This is different from the devbox snapshotter's metadata.db.
The containerd core database is located at:
  /var/lib/containerd/io.containerd.metadata.v1.bolt/meta.db

Use these commands when you need to investigate issues at the containerd
core level, such as snapshot parent-child relationships that may not be
visible in the devbox snapshotter's database.`,
}

// containerdBucketsCmd lists bucket structure
var containerdBucketsCmd = &cobra.Command{
	Use:   "buckets",
	Short: "List bucket structure in containerd meta.db",
	RunE:  runContainerdBuckets,
}

// containerdNamespacesCmd lists namespaces
var containerdNamespacesCmd = &cobra.Command{
	Use:   "namespaces",
	Short: "List all namespaces in containerd",
	RunE:  runContainerdNamespaces,
}

// containerdSnapshotsCmd lists snapshots
var containerdSnapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "List snapshots from containerd's core metadata",
	Long: `List snapshots stored in containerd's core metadata database.

This shows the snapshot information as containerd sees it, which may differ
from what the devbox snapshotter's database shows.

Required flags:
  --namespace    The containerd namespace (e.g., k8s.io, default)
  --snapshotter  The snapshotter name (e.g., devbox, overlayfs)`,
	RunE: runContainerdSnapshots,
}

// containerdGetCmd gets a specific snapshot
var containerdGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get detailed info about a specific snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runContainerdGet,
}

// containerdChildrenCmd finds children of a snapshot
var containerdChildrenCmd = &cobra.Command{
	Use:   "children [key]",
	Short: "Find all children of a snapshot in the 'children' bucket",
	Long: `Find all children references stored in the 'children' bucket for a snapshot.

This directly reads the children bucket in containerd's core metadata,
which is where containerd tracks parent-child relationships.

When ctr says "cannot remove snapshot with child", it's checking this bucket.`,
	Args: cobra.ExactArgs(1),
	RunE: runContainerdChildren,
}

// containerdGhostCmd finds ghost children in containerd metadata
var containerdGhostCmd = &cobra.Command{
	Use:   "ghost",
	Short: "Find ghost children in containerd's core metadata",
	Long: `Find children references in the 'children' bucket that point to non-existent snapshots.

These ghost references prevent the parent snapshot from being deleted via ctr,
even though the child snapshot no longer exists.`,
	RunE: runContainerdGhost,
}

// containerdDumpCmd dumps bucket contents
var containerdDumpCmd = &cobra.Command{
	Use:   "dump [bucket-path...]",
	Short: "Dump raw contents of a bucket path",
	Long: `Dump the raw contents of a bucket for debugging.

Example:
  containerd-meta-viewer containerd dump v1 k8s.io snapshots devbox`,
	Args: cobra.MinimumNArgs(1),
	RunE: runContainerdDump,
}

func runContainerdBuckets(cmd *cobra.Command, args []string) error {
	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}
	defer reader.Close()

	structure, err := reader.ListBucketStructure()
	if err != nil {
		return fmt.Errorf("failed to list bucket structure: %w", err)
	}

	fmt.Println("Containerd meta.db bucket structure:")
	fmt.Println()
	for name, subs := range structure {
		fmt.Printf("  %s/\n", name)
		for _, sub := range subs {
			fmt.Printf("    %s/\n", sub)
		}
	}

	return nil
}

func runContainerdNamespaces(cmd *cobra.Command, args []string) error {
	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}
	defer reader.Close()

	namespaces, err := reader.ListNamespaces()
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	fmt.Println("Namespaces in containerd:")
	for _, ns := range namespaces {
		fmt.Printf("  %s\n", ns)
	}

	return nil
}

func runContainerdSnapshots(cmd *cobra.Command, args []string) error {
	if ctrdNamespace == "" || ctrdSnapshotter == "" {
		return fmt.Errorf("--namespace and --snapshotter are required")
	}

	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}
	defer reader.Close()

	snapshots, err := reader.ListSnapshots(ctrdNamespace, ctrdSnapshotter)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		fmt.Printf("No snapshots found for namespace=%s, snapshotter=%s\n", ctrdNamespace, ctrdSnapshotter)
		return nil
	}

	fmt.Printf("Found %d snapshot(s) in containerd metadata:\n\n", len(snapshots))
	fmt.Printf("%-10s %-50s %s\n", "CHILDREN", "KEY", "PARENT")
	fmt.Println(strings.Repeat("-", 100))

	for _, s := range snapshots {
		parent := s.Parent
		if parent == "" {
			parent = "(none)"
		}
		// Truncate key if too long
		key := s.Key
		if len(key) > 48 {
			key = key[:45] + "..."
		}
		fmt.Printf("%-10d %-50s %s\n", s.ChildrenCount, key, parent)
	}

	return nil
}

func runContainerdGet(cmd *cobra.Command, args []string) error {
	if ctrdNamespace == "" || ctrdSnapshotter == "" {
		return fmt.Errorf("--namespace and --snapshotter are required")
	}

	key := args[0]

	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}
	defer reader.Close()

	info, err := reader.GetSnapshotInfo(ctrdNamespace, ctrdSnapshotter, key)
	if err != nil {
		return fmt.Errorf("failed to get snapshot: %w", err)
	}

	fmt.Println("Snapshot info from containerd metadata:")
	fmt.Printf("  Namespace:      %s\n", info.Namespace)
	fmt.Printf("  Snapshotter:    %s\n", info.Snapshotter)
	fmt.Printf("  Key:            %s\n", info.Key)
	fmt.Printf("  Name (internal):%s\n", info.Name)
	fmt.Printf("  Parent:         %s\n", info.Parent)
	fmt.Printf("  Children count: %d\n", info.ChildrenCount)
	fmt.Printf("  Created:        %s\n", info.CreatedAt)
	fmt.Printf("  Updated:        %s\n", info.UpdatedAt)

	return nil
}

func runContainerdChildren(cmd *cobra.Command, args []string) error {
	if ctrdNamespace == "" || ctrdSnapshotter == "" {
		return fmt.Errorf("--namespace and --snapshotter are required")
	}

	parentKey := args[0]

	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}
	defer reader.Close()

	// Use the children bucket directly (this is how containerd tracks children)
	children, err := reader.ListChildrenFromBucket(ctrdNamespace, ctrdSnapshotter, parentKey)
	if err != nil {
		return fmt.Errorf("failed to find children: %w", err)
	}

	if len(children) == 0 {
		fmt.Printf("No children found in 'children' bucket for snapshot: %s\n", parentKey)
		fmt.Println("\nThis means containerd's core metadata has no children for this snapshot.")
		fmt.Println("If ctr still says 'cannot remove snapshot with child', check:")
		fmt.Println("  1. The devbox snapshotter's metadata.db (use 'snapshots children' command)")
		fmt.Println("  2. Try a different snapshot key format")
		return nil
	}

	fmt.Printf("Found %d children in 'children' bucket for snapshot %s:\n\n", len(children), parentKey)
	for _, c := range children {
		fmt.Printf("  - %s\n", c)
	}

	fmt.Println("\nThese children references in containerd's core metadata are preventing")
	fmt.Println("the snapshot from being deleted.")

	// Also check if these children actually exist
	existing, ghosts, err := reader.ListChildrenWithExistence(ctrdNamespace, ctrdSnapshotter, parentKey)
	if err == nil && len(ghosts) > 0 {
		fmt.Printf("\n⚠️  WARNING: %d of these children are GHOSTS (don't actually exist):\n", len(ghosts))
		for _, g := range ghosts {
			fmt.Printf("  - %s [GHOST]\n", g)
		}
		fmt.Printf("\nExisting children: %d\n", len(existing))
		fmt.Println("\nUse 'containerd ghost' to find all ghost children across all snapshots.")
	}

	return nil
}

func runContainerdGhost(cmd *cobra.Command, args []string) error {
	if ctrdNamespace == "" || ctrdSnapshotter == "" {
		return fmt.Errorf("--namespace and --snapshotter are required")
	}

	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}
	defer reader.Close()

	ghosts, err := reader.FindAllGhostChildren(ctrdNamespace, ctrdSnapshotter)
	if err != nil {
		return fmt.Errorf("failed to find ghost children: %w", err)
	}

	if len(ghosts) == 0 {
		fmt.Println("No ghost children found in containerd's core metadata.")
		fmt.Println("\nAll children references point to existing snapshots.")
		return nil
	}

	fmt.Printf("Found %d ghost child reference(s) in containerd's core metadata:\n\n", len(ghosts))

	// Group by parent
	parentGroups := make(map[string][]string)
	for _, g := range ghosts {
		parentGroups[g.ParentKey] = append(parentGroups[g.ParentKey], g.ChildKey)
	}

	for parent, children := range parentGroups {
		fmt.Printf("Parent: %s\n", parent)
		for _, child := range children {
			fmt.Printf("  - %s [GHOST]\n", child)
		}
		fmt.Println()
	}

	fmt.Println("These ghost references are preventing the parent snapshots from being deleted.")
	fmt.Println("\nUse 'containerd ghost-cleanup' to remove these stale children references.")

	return nil
}

var containerdGhostCleanupDryRun bool

var containerdGhostCleanupCmd = &cobra.Command{
	Use:   "ghost-cleanup",
	Short: "Remove ghost children from containerd's core metadata",
	Long: `Remove stale children references from containerd's meta.db.

These are entries in the 'children' bucket that point to snapshots that no longer exist.
These ghost references prevent parent snapshots from being deleted.

WARNING: containerd MUST be stopped before running this command!
The database cannot be modified while containerd is running.`,
	RunE: runContainerdGhostCleanup,
}

func runContainerdGhostCleanup(cmd *cobra.Command, args []string) error {
	if ctrdNamespace == "" {
		return fmt.Errorf("--namespace is required")
	}

	// First, find all ghost children using read-only reader
	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}

	ghosts, err := reader.FindAllGhostChildren(ctrdNamespace, ctrdSnapshotter)
	reader.Close() // Close before writing

	if err != nil {
		return fmt.Errorf("failed to find ghost children: %w", err)
	}

	if len(ghosts) == 0 {
		fmt.Println("No ghost children found. Nothing to clean up.")
		return nil
	}

	fmt.Printf("Found %d ghost child reference(s) to remove:\n\n", len(ghosts))

	// Group and display
	parentGroups := make(map[string][]string)
	for _, g := range ghosts {
		parentGroups[g.ParentKey] = append(parentGroups[g.ParentKey], g.ChildKey)
	}

	for parent, children := range parentGroups {
		fmt.Printf("Parent: %s\n", parent)
		for _, child := range children {
			fmt.Printf("  - %s [GHOST]\n", child)
		}
	}

	if containerdGhostCleanupDryRun {
		fmt.Println("\n[DRY-RUN] No changes made. Remove --dry-run to actually clean up.")
		return nil
	}

	fmt.Println()

	// Remove ghost children
	removed, failed, err := database.RemoveGhostChildrenFromContainerd(containerdMetaDBPath, ghosts)
	if err != nil {
		return fmt.Errorf("failed to remove ghost children: %w", err)
	}

	fmt.Printf("Cleanup complete: %d removed, %d failed\n", removed, failed)
	return nil
}

func runContainerdDump(cmd *cobra.Command, args []string) error {
	reader, err := database.NewContainerdMetaReader(containerdMetaDBPath)
	if err != nil {
		return fmt.Errorf("failed to open containerd meta.db: %w", err)
	}
	defer reader.Close()

	contents, err := reader.DumpBucketContents(args)
	if err != nil {
		return fmt.Errorf("failed to dump bucket: %w", err)
	}

	fmt.Printf("Contents of bucket path: %s\n\n", strings.Join(args, " -> "))
	for k, v := range contents {
		fmt.Printf("  %s: %v\n", k, v)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(containerdCmd)
	containerdCmd.AddCommand(containerdBucketsCmd)
	containerdCmd.AddCommand(containerdNamespacesCmd)
	containerdCmd.AddCommand(containerdSnapshotsCmd)
	containerdCmd.AddCommand(containerdGetCmd)
	containerdCmd.AddCommand(containerdChildrenCmd)
	containerdCmd.AddCommand(containerdGhostCmd)
	containerdCmd.AddCommand(containerdGhostCleanupCmd)
	containerdCmd.AddCommand(containerdDumpCmd)

	// Add persistent flags to containerd command
	containerdCmd.PersistentFlags().StringVar(&containerdMetaDBPath, "meta-db", database.DefaultContainerdMetaDBPath, "Path to containerd's meta.db")
	containerdCmd.PersistentFlags().StringVar(&ctrdNamespace, "namespace", "", "Containerd namespace (e.g., k8s.io, default)")
	containerdCmd.PersistentFlags().StringVar(&ctrdSnapshotter, "snapshotter", "devbox", "Snapshotter name (e.g., devbox, overlayfs)")

	// Add flags to ghost-cleanup command
	containerdGhostCleanupCmd.Flags().BoolVar(&containerdGhostCleanupDryRun, "dry-run", false, "Show what would be deleted without making changes")
}
