package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestBucketsCmd(t *testing.T) {
	// Find buckets command
	var bucketsCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "buckets" {
			bucketsCmd = cmd
			break
		}
	}

	if bucketsCmd == nil {
		t.Fatal("Expected buckets command to be registered")
	}

	// Test command properties
	if bucketsCmd.Use != "buckets" {
		t.Errorf("Expected buckets command use = 'buckets', got %s", bucketsCmd.Use)
	}

	if bucketsCmd.Short == "" {
		t.Error("Expected buckets command to have a short description")
	}

	if bucketsCmd.Long == "" {
		t.Error("Expected buckets command to have a long description")
	}

	// Test that it has a RunE function
	if bucketsCmd.RunE == nil {
		t.Error("Expected buckets command to have RunE function")
	}
}

func TestBucketsCmdRegistration(t *testing.T) {
	// Test that buckets command is properly registered under root
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "buckets" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected buckets command to be registered under root command")
	}
}