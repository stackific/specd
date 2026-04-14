package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/stackific/specd/internal/db"
)

// InitOptions configures workspace initialization.
type InitOptions struct {
	Force bool
}

// Init creates a new specd workspace at the given path.
func Init(root string, opts InitOptions) (*Workspace, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	dotDir := filepath.Join(abs, ".specd")
	if !opts.Force {
		if _, err := os.Stat(dotDir); err == nil {
			return nil, fmt.Errorf("workspace already initialized at %s (use --force to reinitialize)", abs)
		}
	}

	// Create directory structure.
	dirs := []string{
		filepath.Join(abs, "specd", "specs"),
		filepath.Join(abs, "specd", "kb"),
		dotDir,
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("create %s: %w", d, err)
		}
	}

	// Initialize database.
	dbPath := filepath.Join(dotDir, "cache.db")
	d, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	// Seed user.name from git config if available.
	if name := gitUserName(); name != "" {
		d.SetMeta("user_name", name)
	}

	w := &Workspace{Root: abs, DB: d}

	// Write index.md and log.md.
	if err := w.writeInitialIndex(); err != nil {
		d.Close()
		return nil, err
	}
	if err := w.writeInitialLog(); err != nil {
		d.Close()
		return nil, err
	}

	// Add .specd/ to .gitignore.
	if err := w.ensureGitignore(); err != nil {
		d.Close()
		return nil, err
	}

	return w, nil
}

// writeInitialIndex creates specd/specs/index.md if it doesn't exist.
func (w *Workspace) writeInitialIndex() error {
	path := w.IndexPath()
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	content := "# Specs Index\n\n<!-- Auto-maintained by specd. Do not edit. -->\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// writeInitialLog creates specd/specs/log.md if it doesn't exist.
func (w *Workspace) writeInitialLog() error {
	path := w.LogPath()
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	content := "# Specs Log\n\n<!-- Auto-maintained by specd. Do not edit. -->\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// ensureGitignore appends ".specd/" to .gitignore if not already present.
func (w *Workspace) ensureGitignore() error {
	path := filepath.Join(w.Root, ".gitignore")
	entry := ".specd/"

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if strings.Contains(string(data), entry) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline before entry if file doesn't end with one.
	if len(data) > 0 && data[len(data)-1] != '\n' {
		f.WriteString("\n")
	}
	_, err = f.WriteString(entry + "\n")
	return err
}

// gitUserName reads the current user's name from git config, or returns ""
// if git is not available or no name is configured.
func gitUserName() string {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
