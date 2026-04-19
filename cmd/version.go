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

// Version is set at build time via ldflags.
var Version = "dev"

const (
	repoAPI       = "https://api.github.com/repos/stackific/specd/releases/latest"
	checkInterval = 24 * time.Hour
	httpTimeout   = 3 * time.Second
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of specd",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("specd " + Version)
	},
}

type cacheEntry struct {
	Latest    string `json:"latest"`
	CheckedAt int64  `json:"checked_at"`
}

// CheckForUpdate prints a warning if a newer version is available.
// It caches the result to avoid hitting the API on every run.
func CheckForUpdate() {
	if Version == "dev" {
		return
	}

	latest, ok := getCachedLatest()
	if !ok {
		latest = fetchLatestVersion()
		if latest == "" {
			return
		}
		writeCache(latest)
	}

	current := strings.TrimPrefix(Version, "v")
	latest = strings.TrimPrefix(latest, "v")

	if latest != current {
		fmt.Fprintf(os.Stderr, "\nA new version of specd is available: v%s (current: v%s)\n", latest, current)
		fmt.Fprintf(os.Stderr, "Upgrade: curl -sSL https://stackific.com/specd/install.sh | sh\n\n")
	}
}

func cachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".specd", "update-check.json")
}

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

	if time.Since(time.Unix(entry.CheckedAt, 0)) > checkInterval {
		return "", false
	}

	return entry.Latest, true
}

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

func fetchLatestVersion() string {
	client := &http.Client{Timeout: httpTimeout}

	resp, err := client.Get(repoAPI)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return ""
	}

	return release.TagName
}
