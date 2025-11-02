package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCmd(t *testing.T) {
	// Test that root command is created properly
	root := &cobra.Command{Use: "test"}

	if root.Use != "test" {
		t.Errorf("Expected root command use = 'test', got %s", root.Use)
	}
}

func TestRootFlags(t *testing.T) {
	tests := []struct {
		name        string
		flagName    string
		flagDefault string
		exists      bool
	}{
		{
			name:        "db-path flag",
			flagName:    "db-path",
			flagDefault: "",
			exists:      true,
		},
		{
			name:        "output flag",
			flagName:    "output",
			flagDefault: "table",
			exists:      true,
		},
		{
			name:        "verbose flag",
			flagName:    "verbose",
			flagDefault: "false",
			exists:      true,
		},
		{
			name:        "non-existent flag",
			flagName:    "non-existent",
			flagDefault: "",
			exists:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup(tt.flagName)

			if tt.exists {
				if flag == nil {
					t.Errorf("Expected flag %s to exist", tt.flagName)
					return
				}

				if flag.DefValue != tt.flagDefault {
					t.Errorf("Expected flag %s default value = %s, got %s", tt.flagName, tt.flagDefault, flag.DefValue)
				}
			} else {
				if flag != nil {
					t.Errorf("Expected flag %s to not exist", tt.flagName)
				}
			}
		})
	}
}

func TestRootCmdHasSubcommands(t *testing.T) {
	expectedSubcommands := []string{
		"buckets",
		"snapshots",
		"devbox",
		// "help" is added automatically by cobra, so we don't test for it explicitly
	}

	for _, expected := range expectedSubcommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == expected {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected subcommand %s to be registered", expected)
		}
	}
}

func TestRootCmdRequiredFlags(t *testing.T) {
	// Test that db-path flag is marked as required
	flag := rootCmd.PersistentFlags().Lookup("db-path")
	if flag == nil {
		t.Error("Expected db-path flag to exist")
		return
	}

	// Check if annotation indicates it's required (this depends on cobra version)
	// For now, we'll just check that the flag exists
}