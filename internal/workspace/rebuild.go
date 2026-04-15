// Package workspace — rebuild.go implements the rebuild command which wipes
// the SQLite cache and re-parses the entire workspace from markdown files.
// After rebuild, counters are set to max(existing) + 1 to prevent ID collisions.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stackific/specd/internal/db"
	"github.com/stackific/specd/internal/frontmatter"
	"github.com/stackific/specd/internal/hash"
)

// RebuildResult holds the summary of a rebuild operation.
type RebuildResult struct {
	Specs         int      `json:"specs"`
	Tasks         int      `json:"tasks"`
	KBDocs        int      `json:"kb_docs"`
	KBChunks      int      `json:"kb_chunks"`
	RejectedFiles []string `json:"rejected_files,omitempty"`
}

// Rebuild wipes the SQLite cache and re-parses the workspace from disk.
// Preserves user_name from meta. Re-indexes FTS and trigram via triggers.
func (w *Workspace) Rebuild(force bool) (*RebuildResult, error) {
	var result *RebuildResult

	err := w.WithLock(func() error {
		// Save user_name before wiping.
		userName, _ := w.DB.GetMeta("user_name")

		// Close existing DB.
		dbPath := w.DBPath()
		if err := w.DB.Close(); err != nil {
			return fmt.Errorf("close db: %w", err)
		}

		// Remove old database.
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove db: %w", err)
		}

		// Re-create database.
		newDB, err := db.Open(dbPath)
		if err != nil {
			return fmt.Errorf("reopen db: %w", err)
		}
		w.DB = newDB

		// Restore user_name.
		if userName != "" {
			w.DB.SetMeta("user_name", userName)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		result = &RebuildResult{}

		var maxSpecID, maxTaskID, maxKBID int

		// Scan specs.
		specsDir := w.SpecsDir()
		entries, err := os.ReadDir(specsDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read specs dir: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			match := specIDPattern.FindStringSubmatch(entry.Name())
			if match == nil {
				result.RejectedFiles = append(result.RejectedFiles, filepath.Join("specd", "specs", entry.Name()))
				w.DB.Exec(`INSERT OR IGNORE INTO rejected_files (path, detected_at, reason)
					VALUES (?, ?, ?)`,
					filepath.Join("specd", "specs", entry.Name()), now, "non-canonical spec directory name")
				continue
			}

			specNum, _ := strconv.Atoi(match[1])
			if specNum > maxSpecID {
				maxSpecID = specNum
			}

			specID := fmt.Sprintf("SPEC-%d", specNum)
			specPath := filepath.Join("specd", "specs", entry.Name(), "spec.md")
			absSpecPath := filepath.Join(w.Root, specPath)

			data, err := os.ReadFile(absSpecPath)
			if err != nil {
				result.RejectedFiles = append(result.RejectedFiles, specPath)
				continue
			}

			doc, err := frontmatter.Parse(string(data))
			if err != nil {
				result.RejectedFiles = append(result.RejectedFiles, specPath)
				w.DB.Exec(`INSERT OR IGNORE INTO rejected_files (path, detected_at, reason)
					VALUES (?, ?, ?)`, specPath, now, "unparseable frontmatter")
				continue
			}

			fm, err := frontmatter.DecodeSpec(doc.RawFrontmatter)
			if err != nil {
				result.RejectedFiles = append(result.RejectedFiles, specPath)
				continue
			}

			contentHash := hash.String(string(data))
			slug := Slugify(fm.Title)

			_, err = w.DB.Exec(`INSERT INTO specs (id, slug, title, type, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				specID, slug, fm.Title, fm.Type, fm.Summary, doc.Body,
				specPath, result.Specs, userName, userName, contentHash, now, now)
			if err != nil {
				result.RejectedFiles = append(result.RejectedFiles, specPath)
				continue
			}

			result.Specs++

			// Rebuild spec links from frontmatter.
			for _, linkedID := range fm.LinkedSpecs {
				w.DB.Exec("INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)",
					specID, linkedID)
				w.DB.Exec("INSERT OR IGNORE INTO spec_links (from_spec, to_spec) VALUES (?, ?)",
					linkedID, specID)
			}

			// Scan for task files in the spec directory.
			specDir := filepath.Join(specsDir, entry.Name())
			taskFiles, err := os.ReadDir(specDir)
			if err != nil {
				continue
			}

			for _, tf := range taskFiles {
				if tf.IsDir() || !strings.HasSuffix(tf.Name(), ".md") || tf.Name() == "spec.md" {
					continue
				}

				taskMatch := taskIDPattern.FindStringSubmatch(tf.Name())
				if taskMatch == nil {
					result.RejectedFiles = append(result.RejectedFiles, filepath.Join("specd", "specs", entry.Name(), tf.Name()))
					w.DB.Exec(`INSERT OR IGNORE INTO rejected_files (path, detected_at, reason)
						VALUES (?, ?, ?)`,
						filepath.Join("specd", "specs", entry.Name(), tf.Name()), now, "non-canonical task filename")
					continue
				}

				taskNum, _ := strconv.Atoi(taskMatch[1])
				if taskNum > maxTaskID {
					maxTaskID = taskNum
				}

				taskID := fmt.Sprintf("TASK-%d", taskNum)
				taskPath := filepath.Join("specd", "specs", entry.Name(), tf.Name())
				absTaskPath := filepath.Join(w.Root, taskPath)

				taskData, err := os.ReadFile(absTaskPath)
				if err != nil {
					result.RejectedFiles = append(result.RejectedFiles, taskPath)
					continue
				}

				taskDoc, err := frontmatter.Parse(string(taskData))
				if err != nil {
					result.RejectedFiles = append(result.RejectedFiles, taskPath)
					w.DB.Exec(`INSERT OR IGNORE INTO rejected_files (path, detected_at, reason)
						VALUES (?, ?, ?)`, taskPath, now, "unparseable frontmatter")
					continue
				}

				taskFM, err := frontmatter.DecodeTask(taskDoc.RawFrontmatter)
				if err != nil {
					result.RejectedFiles = append(result.RejectedFiles, taskPath)
					continue
				}

				taskHash := hash.String(string(taskData))
				taskSlug := Slugify(taskFM.Title)

				_, err = w.DB.Exec(`INSERT INTO tasks (id, slug, spec_id, title, status, summary, body, path, position, created_by, updated_by, content_hash, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					taskID, taskSlug, specID, taskFM.Title, taskFM.Status, taskFM.Summary, taskDoc.Body,
					taskPath, result.Tasks, userName, userName, taskHash, now, now)
				if err != nil {
					result.RejectedFiles = append(result.RejectedFiles, taskPath)
					continue
				}

				result.Tasks++

				// Parse and insert acceptance criteria.
				criteria := frontmatter.ParseCriteria(taskDoc.Body)
				for i, c := range criteria {
					checked := 0
					if c.Checked {
						checked = 1
					}
					w.DB.Exec(`INSERT INTO task_criteria (task_id, position, text, checked)
						VALUES (?, ?, ?, ?)`, taskID, i+1, c.Text, checked)
				}

				// Rebuild task links.
				for _, linkedID := range taskFM.LinkedTasks {
					w.DB.Exec("INSERT OR IGNORE INTO task_links (from_task, to_task) VALUES (?, ?)",
						taskID, linkedID)
					w.DB.Exec("INSERT OR IGNORE INTO task_links (from_task, to_task) VALUES (?, ?)",
						linkedID, taskID)
				}

				// Rebuild task dependencies.
				for _, depID := range taskFM.DependsOn {
					w.DB.Exec("INSERT OR IGNORE INTO task_dependencies (blocker_task, blocked_task) VALUES (?, ?)",
						depID, taskID)
				}

				// Rebuild task citations.
				for _, cite := range taskFM.Cites {
					for _, pos := range cite.Chunks {
						w.DB.Exec(`INSERT OR IGNORE INTO citations (from_kind, from_id, kb_doc_id, chunk_position, created_at)
							VALUES ('task', ?, ?, ?, ?)`, taskID, cite.KB, pos, now)
					}
				}
			}

			// Rebuild spec citations.
			for _, cite := range fm.Cites {
				for _, pos := range cite.Chunks {
					w.DB.Exec(`INSERT OR IGNORE INTO citations (from_kind, from_id, kb_doc_id, chunk_position, created_at)
						VALUES ('spec', ?, ?, ?, ?)`, specID, cite.KB, pos, now)
				}
			}
		}

		// Scan KB documents.
		kbDir := w.KBDir()
		kbEntries, err := os.ReadDir(kbDir)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read kb dir: %w", err)
		}

		for _, entry := range kbEntries {
			if entry.IsDir() {
				continue
			}

			// Skip clean HTML sidecars.
			if strings.HasSuffix(entry.Name(), ".clean.html") {
				continue
			}

			match := kbIDPattern.FindStringSubmatch(entry.Name())
			if match == nil {
				result.RejectedFiles = append(result.RejectedFiles, filepath.Join("specd", "kb", entry.Name()))
				w.DB.Exec(`INSERT OR IGNORE INTO rejected_files (path, detected_at, reason)
					VALUES (?, ?, ?)`,
					filepath.Join("specd", "kb", entry.Name()), now, "non-canonical KB filename")
				continue
			}

			kbNum, _ := strconv.Atoi(match[1])
			if kbNum > maxKBID {
				maxKBID = kbNum
			}

			kbID := fmt.Sprintf("KB-%d", kbNum)
			kbPath := filepath.Join("specd", "kb", entry.Name())
			absKBPath := filepath.Join(w.Root, kbPath)

			data, err := os.ReadFile(absKBPath)
			if err != nil {
				result.RejectedFiles = append(result.RejectedFiles, kbPath)
				continue
			}

			srcType := detectSourceType(entry.Name())
			title := titleFromFilename(entry.Name())
			// Strip the KB-N- prefix from the title.
			if idx := strings.Index(title, " "); idx > 0 && strings.HasPrefix(title, "KB") {
				title = strings.TrimSpace(title[idx:])
			}
			contentHash := hash.Bytes(data)
			slug := Slugify(title)

			// Check for clean HTML sidecar.
			var cleanPath *string
			if srcType == "html" {
				cleanName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) + ".clean.html"
				cleanRel := filepath.Join("specd", "kb", cleanName)
				if _, err := os.Stat(filepath.Join(w.Root, cleanRel)); err == nil {
					cleanPath = &cleanRel
				}
			}

			// Chunk the document.
			var chunks []Chunk
			var pageCount *int

			switch srcType {
			case "md":
				chunks = ChunkMarkdown(string(data))
			case "txt":
				chunks = ChunkPlainText(string(data))
			case "html":
				var chunkErr error
				chunks, chunkErr = ChunkHTML(string(data))
				if chunkErr != nil {
					chunks = ChunkPlainText(string(data))
				}
			case "pdf":
				var pc int
				var chunkErr error
				chunks, pc, chunkErr = ChunkPDF(absKBPath)
				if chunkErr != nil {
					result.RejectedFiles = append(result.RejectedFiles, kbPath)
					continue
				}
				pageCount = &pc
			}

			_, err = w.DB.Exec(`INSERT INTO kb_docs (id, slug, title, source_type, path, clean_path, page_count, content_hash, added_at, added_by)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				kbID, slug, title, srcType, kbPath, nilString(cleanPath),
				pageCount, contentHash, now, userName)
			if err != nil {
				result.RejectedFiles = append(result.RejectedFiles, kbPath)
				continue
			}

			for _, c := range chunks {
				_, err := w.DB.Exec(`INSERT INTO kb_chunks (doc_id, position, text, char_start, char_end, page)
					VALUES (?, ?, ?, ?, ?, ?)`,
					kbID, c.Position, c.Text, c.CharStart, c.CharEnd, c.Page)
				if err != nil {
					continue
				}
				result.KBChunks++
			}

			result.KBDocs++
		}

		// Set counters to max(existing) + 1.
		w.DB.SetMeta("next_spec_id", fmt.Sprintf("%d", maxSpecID+1))
		w.DB.SetMeta("next_task_id", fmt.Sprintf("%d", maxTaskID+1))
		w.DB.SetMeta("next_kb_id", fmt.Sprintf("%d", maxKBID+1))

		// Update tidy timestamp.
		w.DB.SetMeta("last_tidy_at", now)

		// Rebuild TF-IDF connections.
		if result.KBChunks > 1 {
			w.KBRebuildConnections(defaultConnectionThreshold, defaultConnectionTopK)
		}

		return nil
	})

	return result, err
}
