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

// frontendDistFS embeds the built frontend assets (ui/dist) at compile time
// so the Web UI ships inside the binary with no external files needed.
//
//go:embed ui/dist
var frontendDistFS embed.FS

func main() {
	// Hand the embedded filesystems to the cmd package before running.
	cmd.SetSkillsFS(skillsFS)

	// Strip the "ui/dist" prefix so the SPA handler sees paths like /index.html.
	distFS, err := fs.Sub(frontendDistFS, "ui/dist")
	if err != nil {
		log.Fatalf("failed to create frontend sub-filesystem: %v", err)
	}
	cmd.SetFrontendFS(distFS)

	cmd.Execute()
}
