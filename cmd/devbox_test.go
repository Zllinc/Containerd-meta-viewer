package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDevboxCmd(t *testing.T) {
	// Find devbox command
	var devboxCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "devbox" {
			devboxCmd = cmd
			break
		}
	}

	if devboxCmd == nil {
		t.Fatal("Expected devbox command to be registered")
	}

	// Test command properties
	if devboxCmd.Use != "devbox" {
		t.Errorf("Expected devbox command use = 'devbox', got %s", devboxCmd.Use)
	}

	if devboxCmd.Short == "" {
		t.Error("Expected devbox command to have a short description")
	}

	if devboxCmd.Long == "" {
		t.Error("Expected devbox command to have a long description")
	}
}

func TestDevboxSubcommands(t *testing.T) {
	// Find devbox command
	var devboxCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "devbox" {
			devboxCmd = cmd
			break
		}
	}

	if devboxCmd == nil {
		t.Fatal("Expected devbox command to be registered")
	}

	// Test expected subcommands
	expectedSubcommands := []string{
		"list",
		"get",
		"lvm-map",
	}

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range devboxCmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected devbox subcommand %s to be registered", expected)
		}
	}
}

func TestDevboxListCmd(t *testing.T) {
	// Find devbox command
	var devboxCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "devbox" {
			devboxCmd = cmd
			break
		}
	}

	if devboxCmd == nil {
		t.Fatal("Expected devbox command to be registered")
	}

	// Find list subcommand
	var listCmd *cobra.Command
	for _, cmd := range devboxCmd.Commands() {
		if cmd.Name() == "list" {
			listCmd = cmd
			break
		}
	}

	if listCmd == nil {
		t.Fatal("Expected devbox list subcommand to be registered")
	}

	// Test command properties
	if listCmd.Use != "list" {
		t.Errorf("Expected devbox list command use = 'list', got %s", listCmd.Use)
	}

	if listCmd.RunE == nil {
		t.Error("Expected devbox list command to have RunE function")
	}
}

func TestDevboxGetCmd(t *testing.T) {
	// Find devbox command
	var devboxCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "devbox" {
			devboxCmd = cmd
			break
		}
	}

	if devboxCmd == nil {
		t.Fatal("Expected devbox command to be registered")
	}

	// Find get subcommand
	var getCmd *cobra.Command
	for _, cmd := range devboxCmd.Commands() {
		if cmd.Name() == "get" {
			getCmd = cmd
			break
		}
	}

	if getCmd == nil {
		t.Fatal("Expected devbox get subcommand to be registered")
	}

	// Test command properties
	if getCmd.Use != "get [content-id]" {
		t.Errorf("Expected devbox get command use = 'get [content-id]', got %s", getCmd.Use)
	}

	// Should expect exactly one argument
	if getCmd.Args != nil {
		// Test argument validation
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
		t.Error("Expected devbox get command to have RunE function")
	}
}

func TestDevboxLvmMapCmd(t *testing.T) {
	// Find devbox command
	var devboxCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "devbox" {
			devboxCmd = cmd
			break
		}
	}

	if devboxCmd == nil {
		t.Fatal("Expected devbox command to be registered")
	}

	// Find lvm-map subcommand
	var lvmMapCmd *cobra.Command
	for _, cmd := range devboxCmd.Commands() {
		if cmd.Name() == "lvm-map" {
			lvmMapCmd = cmd
			break
		}
	}

	if lvmMapCmd == nil {
		t.Fatal("Expected devbox lvm-map subcommand to be registered")
	}

	// Test command properties
	if lvmMapCmd.Use != "lvm-map" {
		t.Errorf("Expected devbox lvm-map command use = 'lvm-map', got %s", lvmMapCmd.Use)
	}

	if lvmMapCmd.RunE == nil {
		t.Error("Expected devbox lvm-map command to have RunE function")
	}
}

func TestDevboxCmdRegistration(t *testing.T) {
	// Test that devbox command is properly registered under root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "devbox" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected devbox command to be registered under root command")
	}
}