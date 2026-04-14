//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web/dist
var distFS embed.FS

func newHandler() http.Handler {
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
