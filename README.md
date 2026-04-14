# specd

A Go server that serves the Astro frontend as a single deployable binary.

## Prerequisites

- [Go](https://go.dev/dl/) 1.26+
- [Node.js](https://nodejs.org/) 22+
- [pnpm](https://pnpm.io/)
- [just](https://github.com/casey/just) (`brew install just`)
- [air](https://github.com/air-verse/air) (`go install github.com/air-verse/air@latest`)

## Development

```sh
just dev
```

Starts the Astro dev server (port 4321) and the Go server (port 8080) with livereload via [air](https://github.com/air-verse/air). Astro's HMR handles frontend changes, air rebuilds and restarts Go on backend changes. Browse `http://localhost:8080`.

## Production Build

```sh
just build
```

Builds the Astro site, then compiles a single Go binary (`specd`) with all static assets embedded. Deploy it anywhere with no dependencies:

```sh
./specd                  # serves on :8080
./specd -addr :3000      # custom port
```

## Cross-Compile

```sh
just build-all
```

Outputs binaries to `dist/`:

- `specd-darwin-arm64` / `specd-darwin-amd64`
- `specd-linux-arm64` / `specd-linux-amd64`
- `specd-windows-amd64.exe`

## Project Structure

```
main.go           Entry point
static_prod.go    Embeds web/dist/client/ into the binary
static_dev.go     Reverse proxies to Astro dev server
justfile          Task runner
web/              Astro frontend (pnpm)
```
