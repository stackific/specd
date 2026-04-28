package cmd

import "io/fs"

// uiFS holds the embedded built SPA (frontend/dist). Set by main.go.
// May be empty before the first `pnpm build` — callers must check via
// hasSPA() to decide whether the static-embed serve path is available.
var uiFS fs.FS

// SetUIFS injects the embedded SPA filesystem (frontend/dist) into the cmd
// package. Pass nil or an empty FS to disable the static-embed serve path
// (`specd serve` will then require --spa-proxy).
func SetUIFS(fsys fs.FS) {
	uiFS = fsys
}

// hasSPA reports whether the embedded UI filesystem contains a usable
// index.html — i.e. the SPA has been built and embedded into the binary.
func hasSPA() bool {
	if uiFS == nil {
		return false
	}
	_, err := fs.Stat(uiFS, "index.html")
	return err == nil
}
