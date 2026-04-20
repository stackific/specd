package main

import (
	"embed"

	"github.com/stackific/specd/cmd"
)

// skillsFS embeds the skills/ directory at compile time so that skill
// files (Agent Skills Standard format) ship inside the binary.
//
//go:embed skills
var skillsFS embed.FS

func main() {
	// Hand the embedded filesystem to the cmd package before running.
	cmd.SetSkillsFS(skillsFS)
	cmd.Execute()
}
