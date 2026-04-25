package cmd

import (
	"io/fs"
)

// permissionsForLevel returns directory and file permissions for provider
// skill directories (.claude/, .agents/, .gemini/). These are always
// world-readable: repo-level ones are committed to VCS, and user-level
// ones must be readable by the AI tools that consume them.
//
// The level parameter is accepted (but unused) to keep the call sites
// self-documenting about which install level they're operating on.
func permissionsForLevel(_ string) (dirPerm, filePerm fs.FileMode) {
	return 0o755, 0o644
}

// Provider describes an AI coding tool and where its skill files live.
type Provider struct {
	Name           string // display name shown in prompts (e.g. "Claude")
	Dir            string // top-level config directory (e.g. ".claude")
	CommandsSubdir string // subdirectory for skills within Dir (always "skills")
}

// Providers is the list of supported AI coding tool providers.
// All three follow the Agent Skills Standard: <Dir>/skills/<name>/SKILL.md.
var Providers = []Provider{
	{Name: ProviderClaude, Dir: ClaudeDir, CommandsSubdir: ClaudeSkillsSubdir},
	{Name: ProviderCodex, Dir: CodexDir, CommandsSubdir: CodexSkillsSubdir},
	{Name: ProviderGemini, Dir: GeminiDir, CommandsSubdir: GeminiSkillsSubdir},
}
