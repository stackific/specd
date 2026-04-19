# Release & Distribution Setup Guide

End-to-end guide for configuring specd releases and user installation.

## 1. GitHub Actions Release Workflow

The workflow lives at `.github/workflows/release.yml`. It is triggered manually via `workflow_dispatch`.

### How to trigger a release

1. Go to **Actions** tab in the GitHub repo
2. Select **Release** workflow
3. Click **Run workflow**
4. Enter the version (e.g. `0.1.0` — no `v` prefix, the workflow adds it)
5. Click **Run workflow**

The workflow will:
- Validate the version format (semver)
- Run tests
- Cross-compile for all 6 platform targets (linux/darwin/windows x amd64/arm64)
- Generate SHA-256 checksums
- Create a GitHub Release with tag `v{version}` and attach all binaries + checksums

### Prerequisites

- The repo must have **Actions enabled**
- The default `GITHUB_TOKEN` needs `contents: write` permission (already set in the workflow)
- No additional secrets are needed

### Adding version to the binary (optional enhancement)

The workflow injects version via ldflags:

```
-X github.com/stackific/specd/cmd.Version=${VERSION}
```

To use this, add to `cmd/root.go`:

```go
// Version is set at build time via ldflags.
var Version = "dev"
```

And add a `version` subcommand or use `rootCmd.Version = Version`.

## 2. Install Scripts

### Unix (macOS / Linux)

File: `scripts/install.sh`

The user runs:

```sh
curl -sSL https://stackific.com/specd/install.sh | sh
```

What it does:
1. Detects OS (linux/darwin) and architecture (amd64/arm64)
2. Fetches the latest release version from GitHub API
3. Downloads the correct binary
4. Installs to `~/.specd/bin/specd`
5. Prints instructions to add `~/.specd/bin` to PATH (detects bash/zsh/fish)
6. Tells user to reload their terminal

### Windows (PowerShell)

File: `scripts/install.ps1`

The user runs:

```powershell
irm https://stackific.com/specd/install.ps1 | iex
```

What it does:
1. Detects architecture (amd64/arm64)
2. Fetches the latest release version from GitHub API
3. Downloads the correct `.exe` binary
4. Installs to `%USERPROFILE%\.specd\bin\specd.exe`
5. Asks the user if they want to add to PATH (modifies User environment variable)
6. Tells user to open a new terminal

## 3. Hosting the Install Scripts

The install scripts reference `https://stackific.com/specd/install.sh` and `install.ps1`. You need to serve these files. Options:

### Option A: Redirect from stackific.com (recommended)

Set up URL redirects on your domain:

```
https://stackific.com/specd/install.sh  -> https://raw.githubusercontent.com/stackific/specd/main/scripts/install.sh
https://stackific.com/specd/install.ps1 -> https://raw.githubusercontent.com/stackific/specd/main/scripts/install.ps1
```

This can be done via:
- **Cloudflare Workers / Pages**: redirect rule or serve the file
- **Netlify**: `_redirects` file
- **Nginx/Caddy**: proxy or redirect rule
- **Vercel**: `vercel.json` rewrite

Example Cloudflare Worker:

```js
export default {
  async fetch(request) {
    const url = new URL(request.url);
    if (url.pathname === "/specd/install.sh") {
      return fetch("https://raw.githubusercontent.com/stackific/specd/main/scripts/install.sh");
    }
    if (url.pathname === "/specd/install.ps1") {
      return fetch("https://raw.githubusercontent.com/stackific/specd/main/scripts/install.ps1");
    }
    return new Response("Not found", { status: 404 });
  }
};
```

### Option B: GitHub raw URL directly

If you don't want a custom domain, users can install with:

```sh
curl -sSL https://raw.githubusercontent.com/stackific/specd/main/scripts/install.sh | sh
```

```powershell
irm https://raw.githubusercontent.com/stackific/specd/main/scripts/install.ps1 | iex
```

## 4. Version Command

Add version support to the CLI so users can verify their install:

In `cmd/root.go`, add:

```go
var Version = "dev"

func init() {
    rootCmd.AddCommand(&cobra.Command{
        Use:   "version",
        Short: "Print the version of specd",
        Run: func(_ *cobra.Command, _ []string) {
            fmt.Println("specd " + Version)
        },
    })
}
```

The release workflow already passes `-X github.com/stackific/specd/cmd.Version=v{version}` via ldflags.

## 5. Updating

For now, users re-run the install script to update. The script always fetches the latest release and overwrites the existing binary.

Future enhancement: add a `specd update` command that does this automatically.

## 6. Uninstalling

Users remove specd by:

1. Deleting `~/.specd/` (or `%USERPROFILE%\.specd\` on Windows)
2. Removing the PATH entry from their shell config

## 7. Checklist Before First Release

- [ ] Push all code to `main` on GitHub
- [ ] Verify Actions are enabled on the repo
- [ ] Set up install script hosting (Option A or B above)
- [ ] Run the Release workflow with version `0.1.0`
- [ ] Test the install script on macOS, Linux, and Windows
- [ ] Verify `specd version` prints the correct version
