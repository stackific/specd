package main

import (
	"embed"
	"io/fs"
	"log"

	"github.com/stackific/specd/cmd"
)

// skillsFS embeds the skills/ directory at compile time so that skill
// files (Agent Skills Standard format) ship inside the binary.
//
//go:embed skills
var skillsFS embed.FS

// uiFS embeds the built SPA at compile time. The `all:` prefix is required
// so dotfiles (e.g. frontend/dist/.gitkeep, used to keep the directory
// present in clean clones before the first `pnpm build`) are included.
// Without it, go:embed silently skips files starting with `_` or `.`.
//
//go:embed all:frontend/dist
var uiFS embed.FS

func main() {
	// Hand the embedded filesystems to the cmd package before running.
	cmd.SetSkillsFS(skillsFS)

	// Strip the frontend/dist prefix so SPA handlers see root-relative paths
	// (e.g. "index.html", "assets/foo.js"). When the SPA has not yet been
	// built, the resulting FS will only contain .gitkeep — cmd.hasSPA()
	// detects this and `specd serve` will require --spa-proxy.
	uFS, err := fs.Sub(uiFS, "frontend/dist")
	if err != nil {
		log.Fatalf("failed to create ui sub-filesystem: %v", err)
	}
	cmd.SetUIFS(uFS)

	cmd.Execute()
}
