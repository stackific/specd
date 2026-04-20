package cmd

import "testing"

// TestToSlug verifies slug conversion for various input formats.
func TestToSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Business", "business"},
		{"Non-functional", "nonfunctional"},
		{"Pending Verification", "pending_verification"},
		{"In progress", "in_progress"},
		{"Wont Fix", "wont_fix"},
		{"  spaces  ", "spaces"},
		{"UPPER CASE", "upper_case"},
		{"already_slug", "already_slug"},
		{"multiple   spaces", "multiple_spaces"},
		{"Hello World 123", "hello_world_123"},
		{"special!@#chars", "specialchars"},
	}

	for _, tt := range tests {
		got := ToSlug(tt.input)
		if got != tt.want {
			t.Errorf("ToSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestToDashSlug verifies dash-separated slug conversion for content identifiers.
func TestToDashSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"User Authentication", "user-authentication"},
		{"Session Management", "session-management"},
		{"OAuth2 Login Flow", "oauth2-login-flow"},
		{"multiple   spaces", "multiple-spaces"},
		{"already-slug", "already-slug"},
		{"underscore_slug", "underscore-slug"},
		{"special!@#chars", "specialchars"},
		{"  leading trailing  ", "leading-trailing"},
	}

	for _, tt := range tests {
		got := ToDashSlug(tt.input)
		if got != tt.want {
			t.Errorf("ToDashSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestFromSlug verifies slug-to-display-text conversion.
func TestFromSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"business", "Business"},
		{"pending_verification", "Pending Verification"},
		{"in_progress", "In Progress"},
		{"wont_fix", "Wont Fix"},
		{"nonfunctional", "Nonfunctional"},
		{"hello_world_123", "Hello World 123"},
	}

	for _, tt := range tests {
		got := FromSlug(tt.input)
		if got != tt.want {
			t.Errorf("FromSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
