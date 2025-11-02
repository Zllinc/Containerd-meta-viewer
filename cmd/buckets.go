package cmd

import (
	"fmt"

	"github.com/containerd/meta-viewer/internal/database"
	"github.com/containerd/meta-viewer/internal/formatters"
	"github.com/spf13/cobra"
)

// bucketsCmd represents the buckets command
var bucketsCmd = &cobra.Command{
	Use:   "buckets",
	Short: "List all top-level buckets in the database",
	Long: `List all top-level buckets in the containerd metadata database.
This command shows the bucket structure and key counts for each bucket.`,
	RunE: runBuckets,
}

func runBuckets(cmd *cobra.Command, args []string) error {
	// Create database reader
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	// Get buckets
	buckets, err := reader.ListBuckets()
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}

	// Format output
	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatBuckets(buckets)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatBuckets(buckets)
	}
}

func init() {
	rootCmd.AddCommand(bucketsCmd)
}