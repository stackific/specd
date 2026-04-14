//go:build dev

package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func newHandler() http.Handler {
	target, _ := url.Parse("http://localhost:4321")
	return httputil.NewSingleHostReverseProxy(target)
}
