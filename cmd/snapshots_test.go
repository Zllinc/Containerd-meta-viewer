package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSnapshotsCmd(t *testing.T) {
	// Find snapshots command
	var snapshotsCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "snapshots" {
			snapshotsCmd = cmd
			break
		}
	}

	if snapshotsCmd == nil {
		t.Fatal("Expected snapshots command to be registered")
	}

	// Test command properties
	if snapshotsCmd.Use != "snapshots" {
		t.Errorf("Expected snapshots command use = 'snapshots', got %s", snapshotsCmd.Use)
	}

	if snapshotsCmd.Short == "" {
		t.Error("Expected snapshots command to have a short description")
	}

	if snapshotsCmd.Long == "" {
		t.Error("Expected snapshots command to have a long description")
	}
}

func TestSnapshotsSubcommands(t *testing.T) {
	// Find snapshots command
	var snapshotsCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "snapshots" {
			snapshotsCmd = cmd
			break
		}
	}

	if snapshotsCmd == nil {
		t.Fatal("Expected snapshots command to be registered")
	}

	// Test expected subcommands
	expectedSubcommands := []string{
		"list",
		"get",
		"search",
	}

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range snapshotsCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected snapshots subcommand %s to be registered", expected)
		}
	}
}

func TestSnapshotsListCmd(t *testing.T) {
	// Find snapshots command
	var snapshotsCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "snapshots" {
			snapshotsCmd = cmd
			break
		}
	}

	if snapshotsCmd == nil {
		t.Fatal("Expected snapshots command to be registered")
	}

	// Find list subcommand
	var listCmd *cobra.Command
	for _, cmd := range snapshotsCmd.Commands() {
		if cmd.Name() == "list" {
			listCmd = cmd
			break
		}
	}

	if listCmd == nil {
		t.Fatal("Expected snapshots list subcommand to be registered")
	}

	// Test command properties
	if listCmd.Use != "list" {
		t.Errorf("Expected snapshots list command use = 'list', got %s", listCmd.Use)
	}

	if listCmd.RunE == nil {
		t.Error("Expected snapshots list command to have RunE function")
	}
}

func TestSnapshotsGetCmd(t *testing.T) {
	// Find snapshots command
	var snapshotsCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "snapshots" {
			snapshotsCmd = cmd
			break
		}
	}

	if snapshotsCmd == nil {
		t.Fatal("Expected snapshots command to be registered")
	}

	// Find get subcommand
	var getCmd *cobra.Command
	for _, cmd := range snapshotsCmd.Commands() {
		if cmd.Name() == "get" {
			getCmd = cmd
			break
		}
	}

	if getCmd == nil {
		t.Fatal("Expected snapshots get subcommand to be registered")
	}

	// Test command properties
	if getCmd.Use != "get [snapshot-key]" {
		t.Errorf("Expected snapshots get command use = 'get [snapshot-key]', got %s", getCmd.Use)
	}

	// Should expect exactly one argument
	if getCmd.Args != nil {
		// Test argument validation by checking the Args function
		err := getCmd.Args(getCmd, []string{})
		if err == nil {
			t.Error("Expected get command to require an argument")
		}

		err = getCmd.Args(getCmd, []string{"arg1", "arg2"})
		if err == nil {
			t.Error("Expected get command to reject multiple arguments")
		}

		err = getCmd.Args(getCmd, []string{"arg1"})
		if err != nil {
			t.Errorf("Expected get command to accept single argument, got error: %v", err)
		}
	}

	if getCmd.RunE == nil {
		t.Error("Expected snapshots get command to have RunE function")
	}
}

func TestSnapshotsSearchCmd(t *testing.T) {
	// Find snapshots command
	var snapshotsCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "snapshots" {
			snapshotsCmd = cmd
			break
		}
	}

	if snapshotsCmd == nil {
		t.Fatal("Expected snapshots command to be registered")
	}

	// Find search subcommand
	var searchCmd *cobra.Command
	for _, cmd := range snapshotsCmd.Commands() {
		if cmd.Name() == "search" {
			searchCmd = cmd
			break
		}
	}

	if searchCmd == nil {
		t.Fatal("Expected snapshots search subcommand to be registered")
	}

	// Test command properties
	if searchCmd.Use != "search" {
		t.Errorf("Expected snapshots search command use = 'search', got %s", searchCmd.Use)
	}

	// Test that search command has the expected flags
	contentIDFlag := searchCmd.Flags().Lookup("content-id")
	if contentIDFlag == nil {
		t.Error("Expected search command to have content-id flag")
	}

	pathFlag := searchCmd.Flags().Lookup("path")
	if pathFlag == nil {
		t.Error("Expected search command to have path flag")
	}

	if searchCmd.RunE == nil {
		t.Error("Expected snapshots search command to have RunE function")
	}
}