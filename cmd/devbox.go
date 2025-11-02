package cmd

import (
	"fmt"

	"github.com/containerd/meta-viewer/internal/database"
	"github.com/containerd/meta-viewer/internal/formatters"
	"github.com/spf13/cobra"
)

// devboxCmd represents the devbox command
var devboxCmd = &cobra.Command{
	Use:   "devbox",
	Short: "Manage and inspect devbox-specific storage information",
	Long: `View and manage devbox-specific storage metadata including LVM
volume mappings, mount paths, and storage status. This command provides
access to the devbox_storage_path bucket and related metadata.`,
}

// devboxListCmd represents the devbox list command
var devboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devbox storage entries",
	Long: `List all devbox storage entries from the devbox_storage_path bucket.
This shows content IDs, LVM volume names, mount paths, and status information.`,
	RunE: runDevboxList,
}

// devboxGetCmd represents the devbox get command
var devboxGetCmd = &cobra.Command{
	Use:   "get [content-id]",
	Short: "Get detailed information about a specific devbox storage entry",
	Long: `Get detailed information about a specific devbox storage entry by content ID.
This shows all available metadata including LVM volume name, mount path, and status.`,
	Args: cobra.ExactArgs(1),
	RunE: runDevboxGet,
}

// devboxLvmMapCmd represents the devbox lvm-map command
var devboxLvmMapCmd = &cobra.Command{
	Use:   "lvm-map",
	Short: "Show LVM volume to mount path mappings",
	Long: `Display a mapping of LVM volume names to their corresponding mount paths.
This is useful for understanding which LVM volumes are mounted where.`,
	RunE: runDevboxLvmMap,
}

func runDevboxList(cmd *cobra.Command, args []string) error {
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	storage, err := reader.ListDevboxStorage()
	if err != nil {
		return fmt.Errorf("failed to list devbox storage: %w", err)
	}

	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatDevboxStorage(storage)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatDevboxStorage(storage)
	}
}

func runDevboxGet(cmd *cobra.Command, args []string) error {
	contentID := args[0]

	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	storage, err := reader.GetDevboxStorage(contentID)
	if err != nil {
		return fmt.Errorf("failed to get devbox storage %s: %w", contentID, err)
	}

	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatDevboxStorageItem(storage)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatDevboxStorageItem(storage)
	}
}

func runDevboxLvmMap(cmd *cobra.Command, args []string) error {
	reader, err := database.NewMetaReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to create database reader: %w", err)
	}
	defer reader.Close()

	storage, err := reader.ListDevboxStorage()
	if err != nil {
		return fmt.Errorf("failed to list devbox storage: %w", err)
	}

	if output == "json" {
		formatter := formatters.NewJSONFormatter(verbose)
		return formatter.FormatLVMMap(storage)
	} else {
		formatter := formatters.NewTableFormatter()
		return formatter.FormatLVMMap(storage)
	}
}

func init() {
	rootCmd.AddCommand(devboxCmd)
	devboxCmd.AddCommand(devboxListCmd)
	devboxCmd.AddCommand(devboxGetCmd)
	devboxCmd.AddCommand(devboxLvmMapCmd)
}