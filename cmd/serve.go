// serve.go implements `specd serve`. Starts an HTTP server with a barebones
// HTMX Web UI. Scans for available ports starting from DefaultServePort (8000),
// prints progress, and opens the browser on success.
package cmd

import (
	"fmt"
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
	mux.HandleFunc("/", handleIndex)

	slog.Info("serve", "port", port)
	fmt.Printf("specd Web UI running at %s\n", url)

	// Try to open the browser.
	if err := openBrowser(url); err != nil {
		slog.Debug("could not open browser", "error", err)
	}

	// Start the server (blocks until interrupted).
	if err := http.ListenAndServe(addr, mux); err != nil { //nolint:gosec // local dev server, no TLS needed
		return fmt.Errorf("server error: %w", err)
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

// handleIndex serves the main Web UI page.
func handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, indexHTML)
}

// indexHTML is the barebones HTMX page served at /.
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>specd</title>
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body { font-family: system-ui, -apple-system, sans-serif; max-width: 960px; margin: 0 auto; padding: 2rem; color: #1a1a1a; background: #fafafa; }
    h1 { font-size: 1.5rem; margin-bottom: 1rem; }
    p { color: #666; }
  </style>
</head>
<body>
  <h1>specd</h1>
  <p>Web UI — coming soon.</p>
</body>
</html>
`
