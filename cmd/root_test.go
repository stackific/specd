package cmd

import "testing"

// TestCommandNameResolution verifies that commandName correctly walks up
// the command tree to find the top-level subcommand name.
func TestCommandNameResolution(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"version"}, "version"},
		{[]string{"skills", "install"}, "skills"}, // subcommand should resolve to parent
		{[]string{"init"}, "init"},
	}

	for _, tt := range tests {
		// Find the cobra.Command that matches the given args.
		c, _, err := rootCmd.Find(tt.args)
		if err != nil {
			t.Fatalf("Find(%v): %v", tt.args, err)
		}
		got := commandName(c)
		if got != tt.want {
			t.Errorf("commandName(%v) = %q, want %q", tt.args, got, tt.want)
		}
	}
}

// TestExemptCommands confirms that the expected set of commands are
// exempt from the project initialization guard.
func TestExemptCommands(t *testing.T) {
	exempt := []string{"init", "version", "skills", "help"}
	for _, name := range exempt {
		if !exemptCommands[name] {
			t.Errorf("expected %q to be exempt", name)
		}
	}
}
