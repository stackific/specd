// Package web — handlers.go defines HTTP handler functions for the
// embedded web UI pages.
package web

import "net/http"

// handleIndex renders the home page.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "index", PageData{})
}

// handleAbout renders the about page.
func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "about", PageData{Title: "About"})
}
