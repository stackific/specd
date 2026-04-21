// Package cmd implements all CLI commands for specd.
package cmd

import "time"

// All magic strings and tunables live here — single source of truth.
const (
	// Company and product identity.
	CompanyName = "Stackific Inc."
	ProductName = "specd"
	Homepage    = "https://stackific.com/specd"
	Copyright   = "All rights reserved."

	// GitHub repository coordinates used for releases and update checks.
	GitHubOwner = "stackific"
	GitHubRepo  = "specd"
	RepoURL     = "https://github.com/" + GitHubOwner + "/" + GitHubRepo
	ReleaseAPI  = "https://api.github.com/repos/" + GitHubOwner + "/" + GitHubRepo + "/releases/latest"

	// Paths and filenames under the user's home directory (~/.specd/).
	InstallDir    = ".specd"                 // top-level directory for all specd user data
	InstallURL    = Homepage + "/install.sh" // URL printed in upgrade prompts
	BinaryName    = "specd"                  // name of the compiled binary
	CacheFile     = "update-check.json"      // cached latest-version response
	ConfigFile    = "config.json"            // global user config (username, etc.)
	ProjectMarker = ".specd.json"            // per-project init marker written to the project root

	// Skills directories.
	SkillsDir      = "skills" // subdirectory under InstallDir for canonical skills
	DefaultFolder  = "specd"  // default name for the specd project folder
	EmbedSkillsDir = "skills" // directory name inside the embedded filesystem

	// Provider display names shown in interactive prompts.
	ProviderClaude = "Claude"
	ProviderCodex  = "OpenAI Codex"
	ProviderGemini = "Gemini"

	// Provider top-level config directories (relative to home or repo root).
	ClaudeDir = ".claude"
	CodexDir  = ".agents"
	GeminiDir = ".gemini"

	// Subdirectory within each provider's config dir where skills are placed.
	// All three follow the Agent Skills Standard: <dir>/skills/<name>/SKILL.md.
	ClaudeSkillsSubdir = "skills"
	CodexSkillsSubdir  = "skills"
	GeminiSkillsSubdir = "skills"

	// Update-check tunables.
	CheckInterval = 24 * time.Hour  // how long a cached version check stays valid
	HTTPTimeout   = 3 * time.Second // max time to wait for the GitHub API

	// Search tunables.
	TopSearchResults = 5 // max related specs/KB chunks returned by new-spec

	// Directory conventions inside the specd project folder.
	SpecsSubdir = "specs" // subdirectory for spec markdown files
)

// DefaultSpecTypes are the built-in spec types offered during init.
var DefaultSpecTypes = []string{"Business", "Functional", "Non-functional"}

// RequiredTaskStages are always included and cannot be deselected.
var RequiredTaskStages = []string{"Backlog", "Todo", "In progress", "Done"}

// OptionalTaskStages can be toggled on or off during init (all on by default).
var OptionalTaskStages = []string{"Blocked", "Pending Verification", "Cancelled", "Wont Fix"}
