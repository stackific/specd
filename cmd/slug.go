package cmd

import (
	"regexp"
	"strings"
)

// nonAlphanumeric matches anything that isn't a letter, digit, space, or underscore.
var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9 _]+`)

// multiUnderscore collapses consecutive underscores into one.
var multiUnderscore = regexp.MustCompile(`_+`)

// ToSlug converts a display string like "Pending Verification" into a
// lowercase, underscore-separated slug like "pending_verification".
func ToSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphanumeric.ReplaceAllString(s, "")  // strip non-alphanumeric
	s = strings.ReplaceAll(s, " ", "_")          // spaces to underscores
	s = multiUnderscore.ReplaceAllString(s, "_") // collapse runs of underscores
	s = strings.Trim(s, "_")                     // trim leading/trailing underscores
	return s
}

// FromSlug converts a slug like "pending_verification" into title-cased
// display text like "Pending Verification".
func FromSlug(s string) string {
	words := strings.Split(s, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
