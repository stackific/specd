// spa_static.go implements the production-mode SPA serve path. When the
// binary is built with a populated frontend/dist embed and `specd serve`
// runs without --spa-proxy, this handler serves the SPA directly from the
// embedded filesystem. Asset requests resolve to real files; every other
// path returns the cached index.html so client-side routing can take over.
package cmd

import (
	"fmt"
	"io/fs"
	"net/http"
)

// makeSPAStatic returns an http.Handler that serves the embedded SPA from
// fsys. index.html is read once at handler-creation time and cached; asset
// requests are delegated to http.FileServer over the same filesystem.
//
// The caller must gate on hasSPA() — passing an FS without index.html
// panics, since that indicates a programmer error (the proxy/legacy paths
// should have been chosen instead).
func makeSPAStatic(fsys fs.FS) http.Handler {
	indexBytes, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		panic(fmt.Sprintf("makeSPAStatic: reading index.html: %v", err))
	}
	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isAssetRequest(r) {
			fileServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	})
}
