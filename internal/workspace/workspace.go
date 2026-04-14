// Package workspace provides domain-level operations for a specd workspace.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/stackific/specd/internal/db"
	"github.com/stackific/specd/internal/lock"
)

// Workspace represents an open specd workspace.
type Workspace struct {
	Root string // absolute path to workspace root
	DB   *db.DB
}

// SpecdDir returns the path to the specd/ vault directory (source of truth).
func (w *Workspace) SpecdDir() string { return filepath.Join(w.Root, "specd") }

// SpecsDir returns the path to specd/specs/.
func (w *Workspace) SpecsDir() string { return filepath.Join(w.Root, "specd", "specs") }

// KBDir returns the path to specd/kb/ where knowledge base files are stored.
func (w *Workspace) KBDir() string { return filepath.Join(w.Root, "specd", "kb") }

// DotDir returns the path to .specd/ (SQLite cache, lockfile; gitignored).
func (w *Workspace) DotDir() string { return filepath.Join(w.Root, ".specd") }

// DBPath returns the path to the SQLite cache database.
func (w *Workspace) DBPath() string { return filepath.Join(w.Root, ".specd", "cache.db") }

// LockPath returns the path to the flock lockfile for single-writer enforcement.
func (w *Workspace) LockPath() string { return filepath.Join(w.Root, ".specd", "lock") }

// IndexPath returns the path to the auto-maintained specs index.
func (w *Workspace) IndexPath() string { return filepath.Join(w.Root, "specd", "specs", "index.md") }

// LogPath returns the path to the auto-maintained specs changelog.
func (w *Workspace) LogPath() string { return filepath.Join(w.Root, "specd", "specs", "log.md") }

// AgentsPath returns the path to the AGENTS.md file at the workspace root.
func (w *Workspace) AgentsPath() string { return filepath.Join(w.Root, "AGENTS.md") }

// Open opens an existing workspace at the given root.
func Open(root string) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Verify it looks like a specd workspace.
	if _, err := os.Stat(filepath.Join(abs, ".specd", "cache.db")); err != nil {
		return nil, fmt.Errorf("not a specd workspace (missing .specd/cache.db): %s", abs)
	}

	d, err := db.Open(filepath.Join(abs, ".specd", "cache.db"))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	return &Workspace{Root: abs, DB: d}, nil
}

// Close closes the workspace database.
func (w *Workspace) Close() error {
	return w.DB.Close()
}

// WithLock acquires the workspace lock, runs fn, then releases.
func (w *Workspace) WithLock(fn func() error) error {
	unlock, err := lock.Acquire(w.LockPath())
	if err != nil {
		return err
	}
	defer unlock()
	return fn()
}

// FindRoot walks up from dir looking for a .specd directory.
// Returns the workspace root or an error if not found.
func FindRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(abs, ".specd")); err == nil {
			return abs, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", fmt.Errorf("no specd workspace found (looked up from %s)", dir)
		}
		abs = parent
	}
}
