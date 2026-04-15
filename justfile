app    := "specd"
webdir := "styles"

# Run all tests
test:
    go test -v -count=1 ./... 2>&1 | grep -E '(^=== RUN|^--- |^FAIL|^ok |^\?)' | tee /dev/stderr | grep -c '^--- PASS' | xargs -I{} echo "{} tests passed"

# Build CSS with Vite (LightningCSS + PurgeCSS)
web:
    cd {{webdir}} && pnpm install --frozen-lockfile && pnpm build

# Build CLI binary (current OS/arch)
build: web
    go build -o {{app}} ./cmd/specd/

# Development: Vite CSS watch + Air Go hot-reload
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    trap 'kill 0' EXIT
    (cd {{webdir}} && pnpm dev) &
    air &
    wait

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
    rm -rf {{app}} dist/ {{webdir}}/dist/
