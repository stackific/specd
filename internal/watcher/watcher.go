// Package watcher monitors the specd workspace for file changes and syncs
// them to SQLite. It debounces events (200ms per path), detects CLI-owned
// writes via content hash, and handles user edits, new files, and deletions.
package watcher

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stackific/specd/internal/frontmatter"
	"github.com/stackific/specd/internal/hash"
	"github.com/stackific/specd/internal/workspace"
)

var (
	specDirPattern = regexp.MustCompile(`^SPEC-(\d+)-`)
	taskFilePattern = regexp.MustCompile(`^TASK-(\d+)-.+\.md$`)
)

// Watcher monitors specd/ for file changes and syncs to SQLite.
type Watcher struct {
	w       *workspace.Workspace
	fsw     *fsnotify.Watcher
	done    chan struct{}
	stopped chan struct{}
	logger  *log.Logger

	// Debounce state: per-path timers.
	mu     sync.Mutex
	timers map[string]*time.Timer
}

// New creates a new file watcher for the given workspace.
func New(w *workspace.Workspace, logger *log.Logger) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}

	if logger == nil {
		logger = log.Default()
	}

	return &Watcher{
		w:       w,
		fsw:     fsw,
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
		logger:  logger,
		timers:  make(map[string]*time.Timer),
	}, nil
}

// Start begins watching the workspace. It adds watches on specd/specs/ and
// specd/kb/ directories (and all spec subdirectories). Call Stop to shut down.
func (wt *Watcher) Start() error {
	// Watch specd/specs/ and all spec subdirectories.
	specsDir := wt.w.SpecsDir()
	if err := wt.addDirRecursive(specsDir); err != nil {
		return fmt.Errorf("watch specs dir: %w", err)
	}

	// Watch specd/kb/.
	kbDir := wt.w.KBDir()
	if _, err := os.Stat(kbDir); err == nil {
		if err := wt.fsw.Add(kbDir); err != nil {
			return fmt.Errorf("watch kb dir: %w", err)
		}
	}

	go wt.loop()
	return nil
}

// Stop shuts down the watcher and waits for the event loop to finish.
func (wt *Watcher) Stop() error {
	close(wt.done)
	<-wt.stopped
	return wt.fsw.Close()
}

// addDirRecursive adds a directory and all subdirectories to the watcher.
func (wt *Watcher) addDirRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if info.IsDir() {
			return wt.fsw.Add(path)
		}
		return nil
	})
}

// loop is the main event loop, processing fsnotify events with debouncing.
func (wt *Watcher) loop() {
	defer close(wt.stopped)

	for {
		select {
		case <-wt.done:
			// Flush remaining timers.
			wt.mu.Lock()
			for _, t := range wt.timers {
				t.Stop()
			}
			wt.mu.Unlock()
			return

		case event, ok := <-wt.fsw.Events:
			if !ok {
				return
			}
			wt.debounce(event)

		case err, ok := <-wt.fsw.Errors:
			if !ok {
				return
			}
			wt.logger.Printf("watcher error: %v", err)
		}
	}
}

// debounce batches events within 200ms per path before processing.
func (wt *Watcher) debounce(event fsnotify.Event) {
	path := event.Name
	op := event.Op

	wt.mu.Lock()
	defer wt.mu.Unlock()

	// Cancel any pending timer for this path.
	if t, ok := wt.timers[path]; ok {
		t.Stop()
	}

	wt.timers[path] = time.AfterFunc(200*time.Millisecond, func() {
		wt.mu.Lock()
		delete(wt.timers, path)
		wt.mu.Unlock()

		wt.handleEvent(path, op)
	})
}

// handleEvent processes a single file event after debouncing.
func (wt *Watcher) handleEvent(absPath string, op fsnotify.Op) {
	// Only process files under specd/.
	rel, err := filepath.Rel(wt.w.Root, absPath)
	if err != nil {
		return
	}
	if !strings.HasPrefix(rel, "specd"+string(filepath.Separator)) {
		return
	}

	// If a new directory appeared under specs/, add it to the watcher.
	if op&fsnotify.Create != 0 {
		info, err := os.Stat(absPath)
		if err == nil && info.IsDir() {
			wt.fsw.Add(absPath)
			return
		}
	}

	// Handle deletion.
	if op&fsnotify.Remove != 0 || op&fsnotify.Rename != 0 {
		wt.handleDelete(rel)
		return
	}

	// Handle create or write.
	if op&fsnotify.Create != 0 || op&fsnotify.Write != 0 {
		wt.handleWriteOrCreate(rel, absPath)
	}
}

// handleDelete processes a file deletion by moving data to trash.
func (wt *Watcher) handleDelete(relPath string) {
	// Determine what was deleted.
	if strings.HasPrefix(relPath, filepath.Join("specd", "specs")) {
		wt.handleSpecOrTaskDelete(relPath)
	} else if strings.HasPrefix(relPath, filepath.Join("specd", "kb")) {
		wt.handleKBDelete(relPath)
	}
}

// handleSpecOrTaskDelete handles deletion of spec or task files.
func (wt *Watcher) handleSpecOrTaskDelete(relPath string) {
	base := filepath.Base(relPath)

	// Skip auto-maintained files.
	if base == "index.md" || base == "log.md" {
		return
	}

	// Check if it's a task file.
	if base != "spec.md" && taskFilePattern.MatchString(base) {
		// Task file deleted.
		taskID := wt.taskIDFromPath(relPath)
		if taskID == "" {
			return
		}
		wt.deleteTaskByWatcher(taskID)
		return
	}

	// Check if spec.md was deleted.
	if base == "spec.md" {
		specID := wt.specIDFromPath(relPath)
		if specID == "" {
			return
		}
		wt.deleteSpecByWatcher(specID)
	}
}

// handleKBDelete handles deletion of KB files.
func (wt *Watcher) handleKBDelete(relPath string) {
	base := filepath.Base(relPath)
	// Skip clean HTML sidecars.
	if strings.HasSuffix(base, ".clean.html") {
		return
	}

	// Find the KB doc by path.
	var kbID string
	err := wt.w.DB.QueryRow("SELECT id FROM kb_docs WHERE path = ?", relPath).Scan(&kbID)
	if err != nil {
		return // not tracked
	}

	wt.deleteKBByWatcher(kbID)
}

// handleWriteOrCreate processes a new or modified file.
func (wt *Watcher) handleWriteOrCreate(relPath, absPath string) {
	// Read file content.
	data, err := os.ReadFile(absPath)
	if err != nil {
		return
	}

	contentHash := hash.String(string(data))

	if strings.HasPrefix(relPath, filepath.Join("specd", "specs")) {
		wt.handleSpecOrTaskWrite(relPath, data, contentHash)
	} else if strings.HasPrefix(relPath, filepath.Join("specd", "kb")) {
		wt.handleKBWrite(relPath, data, contentHash)
	}
}

// handleSpecOrTaskWrite handles writes to spec or task files.
func (wt *Watcher) handleSpecOrTaskWrite(relPath string, data []byte, contentHash string) {
	base := filepath.Base(relPath)

	// Skip auto-maintained files.
	if base == "index.md" || base == "log.md" {
		return
	}

	if base == "spec.md" {
		wt.syncSpec(relPath, data, contentHash)
	} else if taskFilePattern.MatchString(base) {
		wt.syncTask(relPath, data, contentHash)
	} else {
		// Non-canonical file — reject.
		wt.rejectFile(relPath, "non-canonical filename in specs directory")
	}
}

// handleKBWrite handles writes to KB files — for now just log.
// KB files should only be created via `kb add`; external changes are noted.
func (wt *Watcher) handleKBWrite(relPath string, data []byte, contentHash string) {
	base := filepath.Base(relPath)
	if strings.HasSuffix(base, ".clean.html") {
		return // ignore sidecar writes
	}

	// Check if this KB doc exists.
	var storedHash string
	err := wt.w.DB.QueryRow("SELECT content_hash FROM kb_docs WHERE path = ?", relPath).Scan(&storedHash)
	if err == sql.ErrNoRows {
		// New file — reject unless it matches canonical pattern.
		wt.rejectFile(relPath, "KB file created outside specd CLI")
		return
	}
	if err != nil {
		return
	}

	// If hash matches, this is CLI's own write — skip.
	if storedHash == contentHash {
		return
	}

	// Content changed outside CLI. Update the hash but log a warning.
	now := time.Now().UTC().Format(time.RFC3339)
	wt.w.DB.Exec("UPDATE kb_docs SET content_hash = ?, added_at = ? WHERE path = ?",
		contentHash, now, relPath)
	wt.logger.Printf("watcher: KB doc at %s changed outside CLI (hash updated)", relPath)
}

// syncSpec syncs a spec file change to SQLite.
func (wt *Watcher) syncSpec(relPath string, data []byte, contentHash string) {
	specID := wt.specIDFromPath(relPath)
	if specID == "" {
		wt.rejectFile(relPath, "cannot determine spec ID from path")
		return
	}

	// Check if the spec exists in DB.
	var storedHash string
	err := wt.w.DB.QueryRow("SELECT content_hash FROM specs WHERE id = ?", specID).Scan(&storedHash)
	if err == sql.ErrNoRows {
		// Spec not in DB — reject (only rebuild ingests unregistered files).
		wt.rejectFile(relPath, "spec not registered in database")
		return
	}
	if err != nil {
		return
	}

	// CLI's own write — skip.
	if storedHash == contentHash {
		return
	}

	// User edited the file. Parse and sync user-editable fields.
	doc, err := frontmatter.Parse(string(data))
	if err != nil {
		wt.logger.Printf("watcher: unparseable frontmatter in %s: %v", relPath, err)
		return
	}

	fm, err := frontmatter.DecodeSpec(doc.RawFrontmatter)
	if err != nil {
		wt.logger.Printf("watcher: decode spec frontmatter %s: %v", relPath, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userName, _ := wt.w.DB.GetMeta("user_name")

	// Update user-editable fields from markdown. SQLite wins for system fields.
	err = wt.w.WithLock(func() error {
		_, err := wt.w.DB.Exec(`UPDATE specs SET title=?, type=?, summary=?, body=?,
			updated_by=?, content_hash=?, updated_at=? WHERE id=?`,
			fm.Title, fm.Type, fm.Summary, doc.Body, userName, contentHash, now, specID)
		return err
	})

	if err != nil {
		wt.logger.Printf("watcher: sync spec %s: %v", specID, err)
	} else {
		wt.logger.Printf("watcher: synced spec %s from file edit", specID)
	}
}

// syncTask syncs a task file change to SQLite.
func (wt *Watcher) syncTask(relPath string, data []byte, contentHash string) {
	taskID := wt.taskIDFromPath(relPath)
	if taskID == "" {
		wt.rejectFile(relPath, "cannot determine task ID from path")
		return
	}

	// Check if the task exists in DB.
	var storedHash string
	err := wt.w.DB.QueryRow("SELECT content_hash FROM tasks WHERE id = ?", taskID).Scan(&storedHash)
	if err == sql.ErrNoRows {
		wt.rejectFile(relPath, "task not registered in database")
		return
	}
	if err != nil {
		return
	}

	// CLI's own write — skip.
	if storedHash == contentHash {
		return
	}

	// User edited the file. Parse and sync.
	doc, err := frontmatter.Parse(string(data))
	if err != nil {
		wt.logger.Printf("watcher: unparseable frontmatter in %s: %v", relPath, err)
		return
	}

	fm, err := frontmatter.DecodeTask(doc.RawFrontmatter)
	if err != nil {
		wt.logger.Printf("watcher: decode task frontmatter %s: %v", relPath, err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userName, _ := wt.w.DB.GetMeta("user_name")

	err = wt.w.WithLock(func() error {
		// Update user-editable fields.
		_, err := wt.w.DB.Exec(`UPDATE tasks SET title=?, status=?, summary=?, body=?,
			updated_by=?, content_hash=?, updated_at=? WHERE id=?`,
			fm.Title, fm.Status, fm.Summary, doc.Body, userName, contentHash, now, taskID)
		if err != nil {
			return err
		}

		// Re-sync acceptance criteria.
		criteria := frontmatter.ParseCriteria(doc.Body)
		wt.w.DB.Exec("DELETE FROM task_criteria WHERE task_id = ?", taskID)
		for i, c := range criteria {
			checked := 0
			if c.Checked {
				checked = 1
			}
			wt.w.DB.Exec(`INSERT INTO task_criteria (task_id, position, text, checked)
				VALUES (?, ?, ?, ?)`, taskID, i+1, c.Text, checked)
		}

		return nil
	})

	if err != nil {
		wt.logger.Printf("watcher: sync task %s: %v", taskID, err)
	} else {
		wt.logger.Printf("watcher: synced task %s from file edit", taskID)
	}
}

// deleteSpecByWatcher soft-deletes a spec when its file is removed externally.
func (wt *Watcher) deleteSpecByWatcher(specID string) {
	err := wt.w.WithLock(func() error {
		now := time.Now().UTC().Format(time.RFC3339)
		metadata := fmt.Sprintf(`{"id":"%s"}`, specID)

		tx, err := wt.w.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec(`INSERT INTO trash (kind, original_id, original_path, content, metadata, deleted_at, deleted_by)
			VALUES ('spec', ?, ?, '', ?, ?, 'watcher')`,
			specID, "", metadata, now)
		if err != nil {
			return err
		}

		// Delete citations (not FK-cascaded).
		tx.Exec("DELETE FROM citations WHERE from_kind = 'spec' AND from_id = ?", specID)

		_, err = tx.Exec("DELETE FROM specs WHERE id = ?", specID)
		if err != nil {
			return err
		}

		return tx.Commit()
	})

	if err != nil {
		wt.logger.Printf("watcher: delete spec %s: %v", specID, err)
	} else {
		wt.logger.Printf("watcher: spec %s deleted (file removed)", specID)
	}
}

// deleteTaskByWatcher soft-deletes a task when its file is removed externally.
func (wt *Watcher) deleteTaskByWatcher(taskID string) {
	err := wt.w.WithLock(func() error {
		now := time.Now().UTC().Format(time.RFC3339)
		metadata := fmt.Sprintf(`{"id":"%s"}`, taskID)

		tx, err := wt.w.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec(`INSERT INTO trash (kind, original_id, original_path, content, metadata, deleted_at, deleted_by)
			VALUES ('task', ?, ?, '', ?, ?, 'watcher')`,
			taskID, "", metadata, now)
		if err != nil {
			return err
		}

		tx.Exec("DELETE FROM citations WHERE from_kind = 'task' AND from_id = ?", taskID)

		_, err = tx.Exec("DELETE FROM tasks WHERE id = ?", taskID)
		if err != nil {
			return err
		}

		return tx.Commit()
	})

	if err != nil {
		wt.logger.Printf("watcher: delete task %s: %v", taskID, err)
	} else {
		wt.logger.Printf("watcher: task %s deleted (file removed)", taskID)
	}
}

// deleteKBByWatcher soft-deletes a KB doc when its file is removed externally.
func (wt *Watcher) deleteKBByWatcher(kbID string) {
	err := wt.w.WithLock(func() error {
		now := time.Now().UTC().Format(time.RFC3339)
		metadata := fmt.Sprintf(`{"id":"%s"}`, kbID)

		tx, err := wt.w.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec(`INSERT INTO trash (kind, original_id, original_path, content, metadata, deleted_at, deleted_by)
			VALUES ('kb', ?, ?, '', ?, ?, 'watcher')`,
			kbID, "", metadata, now)
		if err != nil {
			return err
		}

		_, err = tx.Exec("DELETE FROM kb_docs WHERE id = ?", kbID)
		if err != nil {
			return err
		}

		return tx.Commit()
	})

	if err != nil {
		wt.logger.Printf("watcher: delete KB %s: %v", kbID, err)
	} else {
		wt.logger.Printf("watcher: KB %s deleted (file removed)", kbID)
	}
}

// rejectFile records a non-canonical file in the rejected_files table.
func (wt *Watcher) rejectFile(relPath, reason string) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := wt.w.DB.Exec(`INSERT OR IGNORE INTO rejected_files (path, detected_at, reason)
		VALUES (?, ?, ?)`, relPath, now, reason)
	if err != nil {
		wt.logger.Printf("watcher: reject file %s: %v", relPath, err)
	} else {
		wt.logger.Printf("watcher: rejected %s (%s)", relPath, reason)
	}
}

// specIDFromPath extracts the spec ID from a relative path like
// "specd/specs/SPEC-42-oauth/spec.md".
func (wt *Watcher) specIDFromPath(relPath string) string {
	parts := strings.Split(relPath, string(filepath.Separator))
	// Expected: specd/specs/SPEC-N-slug/spec.md
	if len(parts) < 4 {
		return ""
	}
	dirName := parts[2] // SPEC-N-slug
	match := specDirPattern.FindStringSubmatch(dirName)
	if match == nil {
		return ""
	}
	return "SPEC-" + match[1]
}

// taskIDFromPath extracts the task ID from a relative path like
// "specd/specs/SPEC-1-auth/TASK-5-design.md".
func (wt *Watcher) taskIDFromPath(relPath string) string {
	base := filepath.Base(relPath)
	match := taskFilePattern.FindStringSubmatch(base)
	if match == nil {
		return ""
	}
	return "TASK-" + match[1]
}
