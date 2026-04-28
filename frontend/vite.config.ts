import { defineConfig } from "vite"
import viteReact from "@vitejs/plugin-react"
import viteTsConfigPaths from "vite-tsconfig-paths"
import tailwindcss from "@tailwindcss/vite"
import { tanstackRouter } from "@tanstack/router-plugin/vite"

// Pure CSR build. TanStack Router file-based routing only — no Start, no
// Nitro server bundle. Output is a static dist/ that can be served by any
// static file server (or embedded in the Go binary in production).
//
// Vite serves the SPA on :5173 in dev. The Go binary (specd serve) listens
// on :8000 and owns the JSON API at /api/*. When Vite is hit directly,
// proxy /api/* to Go so both ports work for development.
export default defineConfig({
  server: {
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8000",
        changeOrigin: true,
      },
    },
  },
  plugins: [
    viteTsConfigPaths({ projects: ["./tsconfig.json"] }),
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    viteReact(),
    tailwindcss(),
  ],
})
