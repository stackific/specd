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

// templatesFS embeds the HTML templates (layouts, partials, pages) at
// compile time for server-side rendering.
//
//go:embed templates
var templatesFS embed.FS

// staticFS embeds static assets (vendor JS, CSS, fonts, images) at
// compile time so the Web UI ships with no external files needed.
//
//go:embed static
var staticFS embed.FS

func main() {
	// Hand the embedded filesystems to the cmd package before running.
	cmd.SetSkillsFS(skillsFS)

	// Strip embed prefixes so template/static handlers see root-relative paths.
	tFS, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		log.Fatalf("failed to create templates sub-filesystem: %v", err)
	}
	cmd.SetTemplateFS(tFS)

	sFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatalf("failed to create static sub-filesystem: %v", err)
	}
	cmd.SetStaticFS(sFS)

	cmd.Execute()
}
