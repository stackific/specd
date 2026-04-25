package cmd

import "io/fs"

// templateFS holds the embedded HTML templates (layouts, partials, pages).
// Set by main.go before command execution.
var templateFS fs.FS

// staticFS holds the embedded static assets (vendor JS, CSS, fonts, images).
// Set by main.go before command execution.
var staticFS fs.FS

// SetTemplateFS injects the embedded template filesystem into the cmd package.
func SetTemplateFS(fsys fs.FS) {
	templateFS = fsys
}

// SetStaticFS injects the embedded static asset filesystem into the cmd package.
func SetStaticFS(fsys fs.FS) {
	staticFS = fsys
}
