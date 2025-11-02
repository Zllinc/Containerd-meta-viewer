package cmd

import (
	"fmt"

	"github.com/containerd/meta-viewer/internal/database"
	"github.com/containerd/meta-viewer/internal/formatters"
	"github.com/spf13/cobra"
)

var (
	searchContentID string
	searchPath      string
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

func init() {
	rootCmd.AddCommand(snapshotsCmd)
	snapshotsCmd.AddCommand(snapshotsListCmd)
	snapshotsCmd.AddCommand(snapshotsGetCmd)
	snapshotsCmd.AddCommand(snapshotsSearchCmd)

	// Add flags to search command
	snapshotsSearchCmd.Flags().StringVar(&searchContentID, "content-id", "", "Search by content ID")
	snapshotsSearchCmd.Flags().StringVar(&searchPath, "path", "", "Search by mount path")
}