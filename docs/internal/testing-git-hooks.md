# Testing Git Hooks

This guide walks through each lefthook hook, shows how to deliberately trigger a failure, and how to fix it so the commit or push goes through.

## Prerequisites

Make sure hooks are installed:

```sh
task hooks:install
```

Verify:

```sh
ls .git/hooks/pre-commit .git/hooks/commit-msg .git/hooks/pre-push
```

All three files should exist and contain lefthook shims.

### Required tools

Tools installed via Homebrew (automatically on PATH):
- `golangci-lint`, `gitleaks`

Tools installed via `go install` (live in `~/go/bin/`):
- `gofumpt`, `goimports`, `gci`, `govulncheck`, `deadcode`, `conform`

The hooks handle PATH setup internally — each command that needs `~/go/bin/` tools exports the path before running. You only need `~/go/bin` on your shell PATH if you want to run these tools directly from the terminal.

---

## Pre-commit Hooks

Three commands run in parallel every time you `git commit`. They operate on staged `.go` files.

| Command | What it does | Blocks commit? |
|---------|-------------|----------------|
| `format` | gofumpt + goimports + gci (auto-fix, re-stage) | Rarely (self-healing) |
| `golangci-lint` | Lint with --fix | Yes |
| `gitleaks` | Secret scanning on staged changes | Yes |

### 1. format (gofumpt + goimports + gci)

All three formatters run sequentially in a single hook, then re-stage the fixed files. This avoids git index lock conflicts from parallel `git add` calls.

**Trigger failure — bad formatting:**

```go
// cmd/test_hook.go
package cmd

import "fmt"

func badFormat()    {
fmt.Println(    "hello"   )
}
```

```sh
git add cmd/test_hook.go
git commit -m "test: check formatters"
```

**What happens:** gofumpt fixes the spacing, goimports verifies imports, gci reorders import groups, then `git add` re-stages. The commit succeeds with the corrected file.

**Undo / pass:** Nothing to do — self-healing. To fix manually:

```sh
gofumpt -w cmd/test_hook.go
goimports -w cmd/test_hook.go
gci write --section standard --section default --section "prefix(github.com/stackific/specd)" cmd/test_hook.go
```

---

**Trigger failure — wrong import order:**

```go
// cmd/test_hook.go
package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
	"os"
)

func testGCI(_ *cobra.Command) {
	fmt.Fprintln(os.Stderr, "wrong import order")
}
```

The correct order is: stdlib, then third-party, then local (`github.com/stackific/specd/...`).

**What happens:** gci rewrites the imports and re-stages. Self-healing.

---

### 2. golangci-lint (linting)

This is the one that **blocks commits** — it does not always auto-fix.

**Trigger failure — unchecked error (errcheck):**

```go
// cmd/test_hook.go
package cmd

import "os"

func testLint() {
	os.Remove("/tmp/fakefile")
}
```

```sh
git add cmd/test_hook.go
git commit -m "test: check errcheck"
```

**What happens:** The commit is rejected:

```
cmd/test_hook.go:7:12: Error return value of `os.Remove` is not checked (errcheck)
```

**Undo / pass:** Handle the error.

```go
func testLint() {
	_ = os.Remove("/tmp/fakefile")
}
```

Or properly:

```go
func testLint() error {
	return os.Remove("/tmp/fakefile")
}
```

Then re-stage and commit.

---

**Trigger failure — unused parameter (revive):**

```go
// cmd/test_hook.go
package cmd

func testUnused(x int) {
	return
}
```

**What happens:** Rejected with:

```
unused-parameter: parameter 'x' seems to be unused (revive)
```

**Undo / pass:** Rename to `_`:

```go
func testUnused(_ int) {
	return
}
```

---

**Trigger failure — exported without doc comment (revive):**

```go
// cmd/test_hook.go
package cmd

func TestExported() {}
```

**What happens:** Rejected with:

```
exported: exported function TestExported should have comment or be unexported (revive)
```

**Undo / pass:** Add a doc comment or unexport:

```go
// TestExported does something.
func TestExported() {}
```

---

**Trigger failure — security issue (gosec):**

```go
// cmd/test_hook.go
package cmd

import (
	"crypto/md5"
	"fmt"
)

func testGosec() {
	h := md5.New()
	fmt.Println(h)
}
```

**What happens:** Rejected with:

```
Use of weak cryptographic primitive (gosec: G401)
```

**Undo / pass:** Use a strong hash:

```go
import "crypto/sha256"

func testGosec() {
	h := sha256.New()
	fmt.Println(h)
}
```

---

### 3. gitleaks (secret scanning)

**Trigger failure:** Stage a file containing something that looks like a secret.

```sh
echo 'GITHUB_TOKEN=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef12' > secrets.txt
git add secrets.txt
git commit -m "test: check gitleaks"
```

**What happens:** The commit is rejected:

```
Finding: GITHUB_TOKEN=ghp_ABCDEF...
RuleID:  github-pat
```

**Undo / pass:** Remove the secret and unstage:

```sh
rm secrets.txt
git reset HEAD secrets.txt
```

If the file is legitimate (e.g., test fixtures with fake tokens), add it to `.gitleaks.toml`:

```toml
[allowlist]
paths = [
  "go\\.sum",
  "\\.gitleaks\\.toml",
  "testdata/.*",
]
```

---

## Commit-msg Hook

Runs after you write your commit message. Enforced by **conform**.

### Conventional commit format

**Trigger failure — wrong format:**

```sh
git add .
git commit -m "added new feature"
```

**What happens:** Rejected:

```
conventional commit format is not valid
```

**Undo / pass:** Use conventional commit format:

```sh
git commit -m "feat: add new feature"
```

---

**Trigger failure — uppercase first letter:**

```sh
git commit -m "feat: Add new feature"
```

**What happens:** Rejected — header case must be lower.

**Undo / pass:**

```sh
git commit -m "feat: add new feature"
```

---

**Trigger failure — non-imperative mood:**

```sh
git commit -m "feat: added new feature"
```

**What happens:** May be rejected — conform checks for imperative mood ("add" not "added").

**Undo / pass:**

```sh
git commit -m "feat: add new feature"
```

---

**Trigger failure — message too long:**

```sh
git commit -m "feat: this is an extremely long commit message that goes well beyond the seventy two character limit set in conform"
```

**What happens:** Rejected — header length exceeds 72 characters.

**Undo / pass:** Shorten the header. Put details in the body:

```sh
git commit -m "feat: add user authentication" -m "Implements OAuth2 flow with refresh token support."
```

---

**Valid commit message examples:**

```sh
git commit -m "feat: add user authentication"
git commit -m "fix(cli): handle empty input gracefully"
git commit -m "docs: update API reference"
git commit -m "refactor: extract validation logic"
git commit -m "test: add edge cases for parser"
git commit -m "chore: update dependencies"
```

Allowed types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.

Scopes are optional and freeform: `feat(cli):`, `fix(parser):`, etc.

---

## Pre-push Hooks

These run in parallel when you `git push`. They are slower checks that gate what reaches the remote.

### 1. go test

**Trigger failure:** Add a failing test.

```go
// cmd/root_test.go
package cmd

import "testing"

func TestFail(t *testing.T) {
	t.Fatal("intentional failure")
}
```

```sh
git add cmd/root_test.go
git commit -m "test: add failing test"
git push
```

**What happens:** Push is rejected:

```
--- FAIL: TestFail (0.00s)
    root_test.go:6: intentional failure
FAIL
```

**Undo / pass:** Fix the test:

```go
func TestFail(t *testing.T) {
	if 1+1 != 2 {
		t.Fatal("math is broken")
	}
}
```

Then amend and push:

```sh
git add cmd/root_test.go
git commit --amend --no-edit
git push
```

---

### 2. govulncheck

**Trigger failure:** This one is hard to trigger deliberately — it flags known CVEs in your dependency tree that your code actually calls. You would need to pin an old, vulnerable version of a dependency.

In practice, if govulncheck blocks your push, it means a dependency has a known vulnerability.

**Undo / pass:** Update the vulnerable dependency:

```sh
go get -u <module>@latest
go mod tidy
```

Then commit and push again.

---

## Bypassing Hooks (escape hatch)

For rare cases where you need to skip hooks (e.g., WIP commits on a feature branch):

```sh
# Skip pre-commit and commit-msg
git commit --no-verify -m "wip: checkpoint"

# Skip pre-push
git push --no-verify
```

Use sparingly. CI should catch anything hooks would have caught.

---

## Running Hooks Manually

You don't need to commit to test hooks. Run them directly:

```sh
# Run all pre-commit checks
lefthook run pre-commit

# Run all pre-push checks
lefthook run pre-push

# Or use task to run individual checks
task fmt:check    # formatting
task lint         # linting
task sec          # all security
task check        # everything
```

---

## Cleanup

After testing, remove any throwaway files:

```sh
rm -f cmd/test_hook.go cmd/root_test.go secrets.txt
git checkout -- .
```
