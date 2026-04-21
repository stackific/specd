package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Version is injected at build time via:
//
//	-ldflags "-X github.com/stackific/specd/cmd.Version=v0.1.0"
//
// Defaults to "dev" for local builds.
var Version = "dev"

// versionCmd prints the current version, company, and homepage.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of specd",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("%s %s\n", ProductName, Version)
		fmt.Printf("%s %s\n", Copyright, CompanyName)
		fmt.Println(Homepage)
	},
}

// cacheEntry is the JSON structure persisted in ~/.specd/update-check.json.
type cacheEntry struct {
	Latest    string `json:"latest"`     // latest release tag (e.g. "v0.2.0")
	CheckedAt int64  `json:"checked_at"` // unix timestamp of last check
}

// CheckForUpdate prints a warning to stderr if a newer version is available.
// It runs after every command via PersistentPostRun on the root command.
// Skipped entirely for local dev builds (Version == "dev").
func CheckForUpdate() {
	if Version == "dev" {
		return
	}

	// Try the cache first to avoid hitting the API on every invocation.
	latest, ok := getCachedLatest()
	if !ok {
		// Cache miss or expired — fetch from GitHub.
		latest = fetchLatestVersion()
		if latest == "" {
			return // network error or API issue — silently skip
		}
		writeCache(latest)
	}

	// Normalize both versions for comparison (strip leading "v").
	current := strings.TrimPrefix(Version, "v")
	latest = strings.TrimPrefix(latest, "v")

	if latest != current {
		fmt.Fprintf(os.Stderr, "\nA new version of %s is available: v%s (current: v%s)\n", ProductName, latest, current)
		fmt.Fprintf(os.Stderr, "Upgrade: curl -sSL %s | sh\n\n", InstallURL)
	}
}

// cachePath returns the absolute path to the update-check cache file.
func cachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, InstallDir, CacheFile)
}

// getCachedLatest reads the cached latest version if the cache exists
// and has not expired (CheckInterval). Returns ("", false) on miss.
func getCachedLatest() (string, bool) {
	p := cachePath()
	if p == "" {
		return "", false
	}

	data, err := os.ReadFile(p) //nolint:gosec // path built from UserHomeDir + hardcoded suffix
	if err != nil {
		return "", false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}

	// Expired — treat as cache miss.
	if time.Since(time.Unix(entry.CheckedAt, 0)) > CheckInterval {
		return "", false
	}

	return entry.Latest, true
}

// writeCache persists the latest version and current timestamp to disk.
// Errors are silently ignored — update checks are best-effort.
func writeCache(latest string) {
	p := cachePath()
	if p == "" {
		return
	}

	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return
	}

	data, err := json.Marshal(cacheEntry{
		Latest:    latest,
		CheckedAt: time.Now().Unix(),
	})
	if err != nil {
		return
	}

	_ = os.WriteFile(p, data, 0o600)
}

// fetchLatestVersion queries the GitHub Releases API for the latest tag.
// Returns "" on any error (network, non-200, parse failure).
func fetchLatestVersion() string {
	client := &http.Client{Timeout: HTTPTimeout}

	resp, err := client.Get(ReleaseAPI)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	// We only need the tag_name field from the release JSON.
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}

	return release.TagName
}
