app    := "specd"
webdir := "web"

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

# Build production binary (current OS/arch)
build: web
    go build -o {{app}} .

# Cross-compile for macOS, Linux, Windows
build-all: web
    mkdir -p dist
    GOOS=darwin  GOARCH=arm64 go build -o dist/{{app}}-darwin-arm64       .
    GOOS=darwin  GOARCH=amd64 go build -o dist/{{app}}-darwin-amd64       .
    GOOS=linux   GOARCH=amd64 go build -o dist/{{app}}-linux-amd64        .
    GOOS=linux   GOARCH=arm64 go build -o dist/{{app}}-linux-arm64        .
    GOOS=windows GOARCH=amd64 go build -o dist/{{app}}-windows-amd64.exe  .

# Remove build artifacts
clean:
    rm -rf {{app}} dist/
