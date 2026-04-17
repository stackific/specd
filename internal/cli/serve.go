// Package cli — serve.go registers the serve command for starting the
// embedded HTTP server that hosts the read-write web UI. The watcher is
// started automatically alongside the server to sync file changes.
package cli

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	specd "github.com/stackific/specd"
	"github.com/stackific/specd/internal/watcher"
	"github.com/stackific/specd/internal/web"
)

func init() {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web UI server",
		Long:  "Starts an HTTP server on localhost serving the embedded read-write web UI. The file watcher is started automatically.",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := openWorkspace()
			if err != nil {
				return err
			}
			defer w.Close()

			// Start the file watcher.
			logger := log.New(os.Stderr, "", log.LstdFlags)
			wt, err := watcher.New(w, logger)
			if err != nil {
				return fmt.Errorf("create watcher: %w", err)
			}
			if err := wt.Start(); err != nil {
				return fmt.Errorf("start watcher: %w", err)
			}
			defer wt.Stop()

			srv, err := web.NewServer(w, specd.TemplateFS, specd.DistFS, specd.AssetsFS)
			if err != nil {
				return fmt.Errorf("create web server: %w", err)
			}

			addr := fmt.Sprintf(":%d", port)
			log.Printf("specd web UI listening on http://localhost%s (watcher active)", addr)
			return http.ListenAndServe(addr, srv)
		},
	}

	cmd.Flags().IntVar(&port, "port", 7823, "listen port")
	rootCmd.AddCommand(cmd)
}
