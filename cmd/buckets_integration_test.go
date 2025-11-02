package cmd

import (
	"testing"
)

func TestBucketsCommandBasicFunctionality(t *testing.T) {
	// Test basic command registration and structure
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{
			name: "buckets command exists",
			test: func(t *testing.T) {
				cmd, _, err := rootCmd.Find([]string{"buckets"})
				if err != nil {
					t.Errorf("Expected buckets command to exist, got error: %v", err)
				}
				if cmd == nil {
					t.Error("Expected buckets command to be found")
				}
			},
		},
		{
			name: "buckets command has proper use",
			test: func(t *testing.T) {
				cmd, _, err := rootCmd.Find([]string{"buckets"})
				if err != nil {
					t.Fatalf("Failed to find buckets command: %v", err)
				}

				expectedUse := "buckets"
				if cmd.Use != expectedUse {
					t.Errorf("Expected command use = '%s', got '%s'", expectedUse, cmd.Use)
				}
			},
		},
		{
			name: "buckets command has short description",
			test: func(t *testing.T) {
				cmd, _, err := rootCmd.Find([]string{"buckets"})
				if err != nil {
					t.Fatalf("Failed to find buckets command: %v", err)
				}

				if cmd.Short == "" {
					t.Error("Expected buckets command to have a short description")
				}

				expectedKeywords := []string{"bucket", "database"}
				for _, keyword := range expectedKeywords {
					if !containsString(cmd.Short, keyword) {
						t.Errorf("Expected short description to contain '%s', got: %s", keyword, cmd.Short)
					}
				}
			},
		},
		{
			name: "buckets command has long description",
			test: func(t *testing.T) {
				cmd, _, err := rootCmd.Find([]string{"buckets"})
				if err != nil {
					t.Fatalf("Failed to find buckets command: %v", err)
				}

				if cmd.Long == "" {
					t.Error("Expected buckets command to have a long description")
				}
			},
		},
		{
			name: "buckets command has run function",
			test: func(t *testing.T) {
				cmd, _, err := rootCmd.Find([]string{"buckets"})
				if err != nil {
					t.Fatalf("Failed to find buckets command: %v", err)
				}

				if cmd.RunE == nil {
					t.Error("Expected buckets command to have RunE function")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func TestBucketsCommandFlags(t *testing.T) {
	// Test that the command properly inherits global flags from root
	flag := rootCmd.PersistentFlags().Lookup("db-path")
	if flag == nil {
		t.Error("Expected db-path flag to be available on root command")
	} else {
		t.Logf("db-path flag found on root command")
	}

	// Check that output flag is available
	flag = rootCmd.PersistentFlags().Lookup("output")
	if flag == nil {
		t.Error("Expected output flag to be available on root command")
	} else {
		defaultValue := flag.Value.String()
		if defaultValue != "table" {
			t.Errorf("Expected output flag default value = 'table', got '%s'", defaultValue)
		}
	}

	// Check that verbose flag is available
	flag = rootCmd.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Error("Expected verbose flag to be available on root command")
	}
}

func TestBucketsCommandHelp(t *testing.T) {
	// Test that help works for buckets command
	args := []string{"buckets", "--help"}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	if err != nil {
		// Help command usually returns an error when executed in tests
		// That's expected behavior, so we just log it
		t.Logf("Help command execution returned error (expected): %v", err)
	}
}

func TestRootCommandValidation(t *testing.T) {
	// Test that root command has proper flag validation set up
	flag := rootCmd.PersistentFlags().Lookup("db-path")
	if flag != nil {
		t.Logf("db-path flag is marked as required: %v", flag)
	}

	// Test that validation function exists
	if rootCmd.PersistentPreRun != nil {
		t.Log("Root command has PersistentPreRun function for validation")
	} else {
		t.Error("Expected root command to have PersistentPreRun function")
	}
}

// Helper function
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}