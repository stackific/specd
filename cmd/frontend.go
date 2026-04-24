package cmd

import "io/fs"

// frontendFS holds the embedded frontend assets (ui/dist).
// Set by main.go before command execution.
var frontendFS fs.FS

// SetFrontendFS injects the embedded frontend filesystem into the cmd package.
func SetFrontendFS(fsys fs.FS) {
	frontendFS = fsys
}
