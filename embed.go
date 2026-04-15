// Package specd provides embedded filesystem variables for the web UI.
// Embed directives must be in a package at the module root so that
// relative paths to templates/, styles/dist/, and assets/ resolve correctly.
package specd

import "embed"

// TemplateFS holds Go HTML templates for server-side rendering.
//
//go:embed all:templates
var TemplateFS embed.FS

// DistFS holds the Vite-built CSS bundle, fonts, and static assets
// from styles/dist/ (favicons, logos, vendor CSS, etc.).
//
//go:embed all:styles/dist
var DistFS embed.FS

// AssetsFS holds vendored JS libraries (htmx, BeerCSS runtime, app.js).
//
//go:embed all:assets
var AssetsFS embed.FS
