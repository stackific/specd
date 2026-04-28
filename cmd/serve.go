// serve.go implements `specd serve`. Starts an HTTP server that exposes the
// JSON API under /api/* and serves the embedded TanStack Router SPA from
// frontend/dist (or, in dev, reverse-proxies non-API traffic to the Vite dev
// server when --spa-proxy is set). Scans for an available port starting from
// DefaultServePort (8000), prints progress, and opens the browser on success.
package cmd

import (
	"database/sql"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	// Ensure correct MIME types for embedded static SPA assets.
	_ = mime.AddExtensionType(".css", "text/css")
	_ = mime.AddExtensionType(".js", "text/javascript")
	_ = mime.AddExtensionType(".woff2", "font/woff2")
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

// serveCmd implements `specd serve`.
// It starts an HTTP server with the Web UI, scanning for an available port.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the specd Web UI",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().Int("port", DefaultServePort, "starting port number")
	serveCmd.Flags().Bool("no-open", false, "do not open the browser on start")
	serveCmd.Flags().String("dir", "", "directory to serve from (defaults to current directory)")
	serveCmd.Flags().String("spa-proxy", "", "if set (e.g. http://localhost:5173), proxy non-/api/* paths to that URL (Vite dev server)")
	rootCmd.AddCommand(serveCmd)
}

func runServe(c *cobra.Command, _ []string) error {
	dir, _ := c.Flags().GetString("dir")
	if dir != "" {
		if err := os.Chdir(dir); err != nil {
			return fmt.Errorf("changing to directory %q: %w", dir, err)
		}
	}

	startPort, _ := c.Flags().GetInt("port")
	spaProxyTarget, _ := c.Flags().GetString("spa-proxy")

	// Find an available port starting from startPort.
	port, err := findAvailablePort(startPort)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	// Set up routes.
	mux := http.NewServeMux()

	// JSON-only API endpoints backing the SPA. Registered first so the
	// catch-all fallback below cannot accidentally shadow them.
	RegisterAPI(mux)

	switch {
	case spaProxyTarget != "":
		proxy, err := makeSPAProxy(spaProxyTarget)
		if err != nil {
			return err
		}
		mux.Handle("/", proxy)
		slog.Info("serve.spa_proxy", "target", spaProxyTarget)
	case hasSPA():
		// Production: serve the SPA from the embedded frontend/dist.
		// /api/* is already registered above and wins over the catch-all.
		mux.Handle("/", makeSPAStatic(uiFS))
		slog.Info("serve.spa_static")
	default:
		return fmt.Errorf("no SPA available: build frontend (`task ui:build`) or run with --spa-proxy")
	}

	slog.Info("serve", "port", port)

	noOpen, _ := c.Flags().GetBool("no-open")
	if !noOpen {
		fmt.Printf("specd Web UI running at %s\n", url)
		if err := openBrowser(url); err != nil {
			slog.Debug("could not open browser", "error", err)
		}
	}

	// Start the server (blocks until interrupted).
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // local dev server, no TLS needed
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// readDefaultRoute queries the database for the configured default route.
// Returns DefaultRoute on any error (DB not initialized, missing key, etc.).
func readDefaultRoute() string {
	db, _, err := OpenProjectDB()
	if err != nil {
		return DefaultRoute
	}
	defer func() { _ = db.Close() }()

	var route string
	err = db.QueryRow("SELECT value FROM meta WHERE key = ?", MetaDefaultRoute).Scan(&route)
	if err != nil {
		return DefaultRoute
	}
	return route
}

// lookupStartpageRoute returns the canonical route from StartpageChoices
// matching the given input. The second return reports whether a match was
// found.
func lookupStartpageRoute(route string) (string, bool) {
	for _, c := range StartpageChoices {
		if c.Route == route {
			return c.Route, true
		}
	}
	return "", false
}

// ReadMeta reads a single value from the meta table by key.
// Returns sql.ErrNoRows if the key does not exist.
func ReadMeta(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM meta WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("reading meta %s: %w", key, err)
	}
	return value, nil
}

// WriteMeta upserts a key/value pair in the meta table.
func WriteMeta(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		`INSERT INTO meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("writing meta %s: %w", key, err)
	}
	return nil
}

// findAvailablePort tries ports starting from startPort, returning the first
// available one. Prints progress to the terminal as it scans.
func findAvailablePort(startPort int) (int, error) {
	for i := 0; i < MaxPortAttempts; i++ {
		port := startPort + i
		ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			fmt.Printf("Port %d in use, trying %d...\n", port, port+1)
			continue
		}
		_ = ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available port found in range %d–%d", startPort, startPort+MaxPortAttempts-1)
}

// openBrowser opens the given URL in the user's default browser.
func openBrowser(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start() //nolint:gosec // url is constructed internally, not from user input
	case "linux":
		return exec.Command("xdg-open", rawURL).Start() //nolint:gosec // url is constructed internally
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start() //nolint:gosec // url is constructed internally
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
