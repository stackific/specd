// serve.go implements `specd serve`. Starts an HTTP server serving the
// embedded Svelte SPA and REST API routes. Scans for available ports
// starting from DefaultServePort (8000), prints progress, and opens the
// browser on success.
package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/spf13/cobra"
)

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
	rootCmd.AddCommand(serveCmd)
}

func runServe(c *cobra.Command, _ []string) error {
	startPort, _ := c.Flags().GetInt("port")

	// Find an available port starting from startPort.
	port, err := findAvailablePort(startPort)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	// Set up routes.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/meta/default-route", handleGetDefaultRoute)
	mux.Handle("/", spaHandler(frontendFS))

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

// spaHandler returns an http.Handler that serves the SPA from the given
// filesystem. The root path "/" redirects to the default route stored in the
// database. Requests for existing files are served directly. All other
// paths fall back to index.html for client-side routing.
func spaHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Root path: redirect to the configured default route.
		if r.URL.Path == "/" {
			route := readDefaultRoute()
			http.Redirect(w, r, route, http.StatusTemporaryRedirect)
			return
		}

		// Try to serve the requested file from the embedded FS.
		if f, err := assets.Open(r.URL.Path[1:]); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html directly for client-side routing.
		serveIndexHTML(w, assets)
	})
}

// serveIndexHTML reads index.html from the embedded FS and writes it directly.
// This avoids http.FileServer's redirect from /index.html → /.
func serveIndexHTML(w http.ResponseWriter, assets fs.FS) {
	data, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
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

// handleGetDefaultRoute returns the configured default route as JSON.
func handleGetDefaultRoute(w http.ResponseWriter, _ *http.Request) {
	route := readDefaultRoute()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"default_route": route})
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
