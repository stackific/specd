// Package cli — serve.go registers the serve command for starting the
// embedded HTTP server that hosts the read-write web UI.
package cli

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	specd "github.com/stackific/specd"
	"github.com/stackific/specd/internal/web"
)

func init() {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web UI server",
		Long:  "Starts an HTTP server on localhost serving the embedded read-write web UI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			srv, err := web.NewServer(specd.TemplateFS, specd.DistFS, specd.AssetsFS)
			if err != nil {
				return fmt.Errorf("create web server: %w", err)
			}

			addr := fmt.Sprintf(":%d", port)
			log.Printf("specd web UI listening on http://localhost%s", addr)
			return http.ListenAndServe(addr, srv)
		},
	}

	cmd.Flags().IntVar(&port, "port", 7823, "listen port")
	rootCmd.AddCommand(cmd)
}
