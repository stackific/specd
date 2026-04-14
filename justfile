app    := "specd"
webdir := "web"

# Run all tests
test:
    go test ./...

# Build CLI binary (current OS/arch)
build:
    go build -o {{app}} ./cmd/specd/

# Development: Astro dev server + Go with livereload (air)
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    trap 'kill 0' EXIT
    (cd {{webdir}} && pnpm dev) &
    sleep 2
    air &
    wait

# Build Astro static site
web:
    cd {{webdir}} && pnpm install --frozen-lockfile && pnpm build

# Cross-compile for macOS, Linux, Windows
build-all: web
    mkdir -p dist
    GOOS=darwin  GOARCH=arm64 go build -o dist/{{app}}-darwin-arm64       ./cmd/specd/
    GOOS=darwin  GOARCH=amd64 go build -o dist/{{app}}-darwin-amd64       ./cmd/specd/
    GOOS=linux   GOARCH=amd64 go build -o dist/{{app}}-linux-amd64        ./cmd/specd/
    GOOS=linux   GOARCH=arm64 go build -o dist/{{app}}-linux-arm64        ./cmd/specd/
    GOOS=windows GOARCH=amd64 go build -o dist/{{app}}-windows-amd64.exe  ./cmd/specd/

# Remove build artifacts
clean:
    rm -rf {{app}} dist/
