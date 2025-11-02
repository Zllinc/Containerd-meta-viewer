package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	defaultDBPath = "/var/lib/containerd/io.containerd.snapshotter.v1.devbox/metadata.db"
)

var (
	dbPath   string
	output   string
	verbose  bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "containerd-meta-viewer",
	Short: "A CLI tool to inspect containerd snapshotter metadata",
	Long: `containerd-meta-viewer is a command-line tool for inspecting the metadata
stored by containerd snapshotters. It allows you to view snapshots,
storage information, and LVM mappings stored in the bolt database.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Use default database path if not provided
		if dbPath == "" {
			dbPath = defaultDBPath
			if verbose {
				fmt.Fprintf(os.Stderr, "Using default database path: %s\n", dbPath)
			}
		}

		// Validate output format
		if output != "table" && output != "json" {
			fmt.Fprintf(os.Stderr, "Error: invalid output format '%s'. Use 'table' or 'json'\n", output)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dbPath, "db-path", "p", "", "Path to the containerd metadata.db file (default: /var/lib/containerd/io.containerd.snapshotter.v1.devbox/metadata.db)")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format (table|json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
}