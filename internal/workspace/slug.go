// Package workspace — slug.go provides URL-safe slug generation from titles.
package workspace

import (
	"fmt"
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// h1Pattern matches markdown heading 1 (# Title) at the start of a line.
var h1Pattern = regexp.MustCompile(`(?m)^#\s+`)

// ErrBodyHasH1 is returned when a body contains a heading 1.
var ErrBodyHasH1 = fmt.Errorf("do not use heading 1 (# Title) in the body — the title must come from the frontmatter title field")

// validateNoH1 returns ErrBodyHasH1 if the body contains a markdown H1 heading.
func validateNoH1(body string) error {
	if h1Pattern.MatchString(body) {
		return ErrBodyHasH1
	}
	return nil
}

// Slugify converts a title into a URL/path-safe slug.
func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	return s
}
