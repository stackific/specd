package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GlobalConfig holds user-level settings persisted at ~/.specd/config.json.
type GlobalConfig struct {
	Username string `json:"username"` // the user's chosen display name
}

// ProjectConfig is the marker file (.specd.json) written to a project root
// when `specd init` is run. It records project-level specd settings.
type ProjectConfig struct {
	Dir              string        `json:"dir"`                // name of the specd project directory
	Username         string        `json:"username,omitempty"` // project-specific username override (empty = use global)
	SpecTypes        []string      `json:"spec_types"`         // slugs of enabled spec types
	TaskStages       []string      `json:"task_stages"`        // slugs of enabled task stages
	TopSearchResults int           `json:"top_search_results"` // max related items returned by search
	SearchWeights    SearchWeights `json:"search_weights"`     // BM25 column weights for ranking
}

// SearchWeights holds BM25 column weights for full-text search ranking.
// Higher values mean matches in that column are more important.
type SearchWeights struct {
	Title   float64 `json:"title"`   // weight for title column matches
	Summary float64 `json:"summary"` // weight for summary column matches
	Body    float64 `json:"body"`    // weight for body/text column matches
}

// globalConfigPath returns the absolute path to ~/.specd/config.json.
// Returns "" if the home directory cannot be determined.
func globalConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, InstallDir, ConfigFile)
}

// LoadGlobalConfig reads the global config from disk.
// Returns an empty config (not an error) if the file does not exist yet.
func LoadGlobalConfig() (*GlobalConfig, error) {
	p := globalConfigPath()
	if p == "" {
		return &GlobalConfig{}, nil
	}

	data, err := os.ReadFile(p) //nolint:gosec // path built from UserHomeDir + hardcoded suffix
	if err != nil {
		if os.IsNotExist(err) {
			return &GlobalConfig{}, nil // first run — no config yet
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg GlobalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// SaveGlobalConfig writes the global config to ~/.specd/config.json,
// creating the directory if it doesn't exist.
func SaveGlobalConfig(cfg *GlobalConfig) error {
	p := globalConfigPath()
	if p == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	// Ensure ~/.specd/ exists.
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	// 0o600: owner read/write only — config may contain user identity.
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// LoadProjectConfig reads the .specd.json marker from the given directory.
// Returns (nil, nil) if the marker does not exist (project not initialized).
func LoadProjectConfig(dir string) (*ProjectConfig, error) {
	p := filepath.Join(dir, ProjectMarker)

	data, err := os.ReadFile(p) //nolint:gosec // path from caller
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not initialized — not an error
		}
		return nil, fmt.Errorf("reading project config: %w", err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing project config: %w", err)
	}

	return &cfg, nil
}

// SaveProjectConfig writes the .specd.json marker into the given directory.
func SaveProjectConfig(dir string, cfg *ProjectConfig) error {
	p := filepath.Join(dir, ProjectMarker)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling project config: %w", err)
	}

	if err := os.WriteFile(p, data, 0o644); err != nil { //nolint:gosec // project marker is committed to VCS, must be world-readable
		return fmt.Errorf("writing project config: %w", err)
	}

	return nil
}
