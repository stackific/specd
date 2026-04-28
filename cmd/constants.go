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
	DefaultDir     = "specd"  // default name for the specd project directory
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
	TopSearchResults   = 5   // max related specs/KB chunks returned by new-spec
	ChunkPreviewLength = 200 // max characters shown in KB chunk previews

	// Pagination defaults.
	DefaultPageSize = 20 // default results per page for list commands

	// Serve tunables.
	DefaultServePort     = 8000 // starting port for specd serve
	MaxPortAttempts      = 100  // max ports to try before giving up
	MaxSettingsFormBytes = 4096 // upper bound on settings POST bodies

	// BM25 column weights: title matches are most important, body least.
	// FTS columns are ordered (title, summary, body) in specs_fts and tasks_fts.
	BM25WeightTitle   = 10.0
	BM25WeightSummary = 5.0
	BM25WeightBody    = 1.0

	// Search kind identifiers — used in search queries and trigram table.
	KindSpec = "spec"
	KindTask = "task"
	KindKB   = "kb"
	KindAll  = "all"

	// Meta table keys for ID counters and UI settings.
	MetaNextSpecID   = "next_spec_id"
	MetaNextTaskID   = "next_task_id"
	MetaNextKBID     = "next_kb_id"
	MetaDefaultRoute = "default_route"

	// Default route for the Web UI root redirect.
	DefaultRoute = "/docs/tutorial"

	// Theme seed color for Material Design 3 dynamic color generation.
	ThemeSeedColor = "#1c4bea"

	// API prefix for REST endpoints.
	APIPrefix = "/api/"

	// ID prefixes for each content type.
	IDPrefixSpec = "SPEC-"
	IDPrefixTask = "TASK-"
	IDPrefixKB   = "KB-"

	// Directory conventions inside the specd project folder.
	SpecsSubdir = "specs" // subdirectory for spec markdown files
	KBSubdir    = "kb"    // subdirectory for KB document files

	// Logging.
	LogFile    = "specd.log"      // log filename inside InstallDir (~/.specd/)
	MaxLogSize = 10 * 1024 * 1024 // 10 MB — truncate log file when exceeded
)

// DefaultSpecTypes are the built-in spec types offered during init.
var DefaultSpecTypes = []string{"Business", "Functional", "Non-functional"}

// RequiredTaskStages are always included and cannot be deselected.
var RequiredTaskStages = []string{"Backlog", "Todo", "In progress", "Done"}

// StartpageChoice is a selectable startpage option exposed on /settings.
type StartpageChoice struct {
	Label string
	Route string
}

// StartpageChoices lists the routes a user may pick as the Web UI startpage
// (the page `/` redirects to). Order is the order shown in the UI.
var StartpageChoices = []StartpageChoice{
	{Label: "Tutorial", Route: "/docs/tutorial"},
	{Label: "Tasks", Route: "/tasks"},
	{Label: "Specs", Route: "/specs"},
	{Label: "KB", Route: "/kb"},
	{Label: "Search", Route: "/search"},
}

// IsValidStartpageRoute reports whether route is one of the allowed
// startpage routes in StartpageChoices.
func IsValidStartpageRoute(route string) bool {
	for _, c := range StartpageChoices {
		if c.Route == route {
			return true
		}
	}
	return false
}

// OptionalTaskStages can be toggled on or off during init (all on by default).
var OptionalTaskStages = []string{"Blocked", "Pending Verification", "Cancelled", "Wont Fix"}
