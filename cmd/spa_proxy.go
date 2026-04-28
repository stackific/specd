// spa_proxy.go implements a reverse-proxy handler used by `specd serve` to
// forward client traffic to a separately-running Vite dev server during
// local development. /api/* requests are served by the Go process directly;
// every other path goes through the proxy. Non-asset paths (e.g. /welcome,
// /specs/SPEC-1) are rewritten to "/" so Vite returns index.html — the SPA
// shell — and TanStack Router handles routing on the client.
package cmd

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

// makeSPAProxy returns an http.Handler that reverse-proxies requests to the
// given upstream (e.g. "http://127.0.0.1:5173"). Asset requests pass through
// unchanged; everything else is rewritten to "/" so the SPA shell is served.
// Websockets/SSE work transparently — httputil.ReverseProxy hijacks on
// Upgrade, which is what Vite HMR relies on.
func makeSPAProxy(upstream string) (http.Handler, error) {
	target, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("parsing spa-proxy target %q: %w", upstream, err)
	}
	// Use the modern Rewrite hook (Go 1.20+) instead of the deprecated
	// Director. Rewrite gets a *httputil.ProxyRequest that exposes both the
	// inbound (.In) and outbound (.Out) requests, and SetURL handles host
	// rewriting + scheme/path joining the way Director did.
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			if !isAssetRequest(pr.In) {
				// SPA fallback: serve index.html for any non-asset path.
				pr.Out.URL.Path = "/"
				pr.Out.URL.RawPath = ""
			}
			pr.SetURL(target)
			// Preserve the upstream Host so Vite generates URLs that
			// resolve back through our user-facing port.
			pr.Out.Host = target.Host
		},
	}
	return proxy, nil
}

// isAssetRequest reports whether the request should pass through to the
// upstream untouched (true) versus be rewritten to "/" for SPA fallback
// (false).
func isAssetRequest(r *http.Request) bool {
	p := r.URL.Path
	// Vite internal paths always pass through.
	switch {
	case strings.HasPrefix(p, "/@vite/"),
		strings.HasPrefix(p, "/@id/"),
		strings.HasPrefix(p, "/@fs/"),
		strings.HasPrefix(p, "/node_modules/"),
		strings.HasPrefix(p, "/src/"):
		return true
	}
	// Anything with an extension on the last path segment is an asset.
	return path.Ext(p) != ""
}
