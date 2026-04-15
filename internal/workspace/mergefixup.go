// Package workspace — mergefixup.go detects and repairs ID collisions and
// inconsistencies that can occur after a git merge. It scans for duplicate
// spec/task/KB IDs and renumbers colliding entries.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// MergeFixupResult holds the report from a merge-fixup operation.
type MergeFixupResult struct {
	DuplicateSpecs []DuplicateID `json:"duplicate_specs,omitempty"`
	DuplicateTasks []DuplicateID `json:"duplicate_tasks,omitempty"`
	DuplicateKB    []DuplicateID `json:"duplicate_kb,omitempty"`
	Renumbered     []Renumbered  `json:"renumbered,omitempty"`
}

// DuplicateID describes a collision between two items sharing the same ID.
type DuplicateID struct {
	ID    string   `json:"id"`
	Paths []string `json:"paths"`
}

// Renumbered describes an item that was assigned a new ID during fixup.
type Renumbered struct {
	OldID   string `json:"old_id"`
	NewID   string `json:"new_id"`
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

// MergeFixup scans the workspace for ID collisions and repairs them.
// This is a destructive operation — it renames directories and files
// and updates frontmatter references.
func (w *Workspace) MergeFixup() (*MergeFixupResult, error) {
	var result *MergeFixupResult

	err := w.WithLock(func() error {
		result = &MergeFixupResult{}

		// Scan for duplicate spec IDs.
		specDups, err := w.findDuplicateSpecs()
		if err != nil {
			return fmt.Errorf("scan spec duplicates: %w", err)
		}
		result.DuplicateSpecs = specDups

		// Scan for duplicate task IDs.
		taskDups, err := w.findDuplicateTasks()
		if err != nil {
			return fmt.Errorf("scan task duplicates: %w", err)
		}
		result.DuplicateTasks = taskDups

		// Scan for duplicate KB IDs.
		kbDups, err := w.findDuplicateKB()
		if err != nil {
			return fmt.Errorf("scan KB duplicates: %w", err)
		}
		result.DuplicateKB = kbDups

		// Fix duplicates by renumbering the second occurrence.
		for _, dup := range specDups {
			if len(dup.Paths) < 2 {
				continue
			}
			// Keep the first, renumber the rest.
			for _, path := range dup.Paths[1:] {
				renumbered, err := w.renumberSpec(dup.ID, path)
				if err != nil {
					return fmt.Errorf("renumber spec %s at %s: %w", dup.ID, path, err)
				}
				result.Renumbered = append(result.Renumbered, *renumbered)
			}
		}

		for _, dup := range taskDups {
			if len(dup.Paths) < 2 {
				continue
			}
			for _, path := range dup.Paths[1:] {
				renumbered, err := w.renumberTask(dup.ID, path)
				if err != nil {
					return fmt.Errorf("renumber task %s at %s: %w", dup.ID, path, err)
				}
				result.Renumbered = append(result.Renumbered, *renumbered)
			}
		}

		for _, dup := range kbDups {
			if len(dup.Paths) < 2 {
				continue
			}
			for _, path := range dup.Paths[1:] {
				renumbered, err := w.renumberKB(dup.ID, path)
				if err != nil {
					return fmt.Errorf("renumber KB %s at %s: %w", dup.ID, path, err)
				}
				result.Renumbered = append(result.Renumbered, *renumbered)
			}
		}

		// After renumbering, rebuild the database to ensure consistency.
		if len(result.Renumbered) > 0 {
			_, err := w.Rebuild(true)
			if err != nil {
				return fmt.Errorf("rebuild after fixup: %w", err)
			}
		}

		return nil
	})

	return result, err
}

// findDuplicateSpecs scans the specs directory for duplicate SPEC-N IDs.
func (w *Workspace) findDuplicateSpecs() ([]DuplicateID, error) {
	specsDir := w.SpecsDir()
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Map spec ID number -> list of directory paths.
	seen := map[int][]string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		match := specIDPattern.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		num, _ := strconv.Atoi(match[1])
		seen[num] = append(seen[num], filepath.Join("specd", "specs", entry.Name()))
	}

	var dups []DuplicateID
	for num, paths := range seen {
		if len(paths) > 1 {
			dups = append(dups, DuplicateID{
				ID:    fmt.Sprintf("SPEC-%d", num),
				Paths: paths,
			})
		}
	}
	return dups, nil
}

// findDuplicateTasks scans all spec directories for duplicate TASK-N filenames.
func (w *Workspace) findDuplicateTasks() ([]DuplicateID, error) {
	specsDir := w.SpecsDir()
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	seen := map[int][]string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		specDir := filepath.Join(specsDir, entry.Name())
		files, err := os.ReadDir(specDir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || f.Name() == "spec.md" || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			match := taskIDPattern.FindStringSubmatch(f.Name())
			if match == nil {
				continue
			}
			num, _ := strconv.Atoi(match[1])
			path := filepath.Join("specd", "specs", entry.Name(), f.Name())
			seen[num] = append(seen[num], path)
		}
	}

	var dups []DuplicateID
	for num, paths := range seen {
		if len(paths) > 1 {
			dups = append(dups, DuplicateID{
				ID:    fmt.Sprintf("TASK-%d", num),
				Paths: paths,
			})
		}
	}
	return dups, nil
}

// findDuplicateKB scans the kb directory for duplicate KB-N filenames.
func (w *Workspace) findDuplicateKB() ([]DuplicateID, error) {
	kbDir := w.KBDir()
	entries, err := os.ReadDir(kbDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	seen := map[int][]string{}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".clean.html") {
			continue
		}
		match := kbIDPattern.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		num, _ := strconv.Atoi(match[1])
		path := filepath.Join("specd", "kb", entry.Name())
		seen[num] = append(seen[num], path)
	}

	var dups []DuplicateID
	for num, paths := range seen {
		if len(paths) > 1 {
			dups = append(dups, DuplicateID{
				ID:    fmt.Sprintf("KB-%d", num),
				Paths: paths,
			})
		}
	}
	return dups, nil
}

// renumberSpec renames a spec directory to use a new ID number.
func (w *Workspace) renumberSpec(oldID, oldRelDir string) (*Renumbered, error) {
	newNum, err := w.DB.NextID("spec")
	if err != nil {
		return nil, err
	}
	newID := fmt.Sprintf("SPEC-%d", newNum)

	oldDirName := filepath.Base(oldRelDir)
	newDirName := newID + oldDirName[len(oldID):]
	newRelDir := filepath.Join("specd", "specs", newDirName)

	oldAbs := filepath.Join(w.Root, oldRelDir)
	newAbs := filepath.Join(w.Root, newRelDir)

	if err := os.Rename(oldAbs, newAbs); err != nil {
		return nil, fmt.Errorf("rename directory: %w", err)
	}

	return &Renumbered{
		OldID:   oldID,
		NewID:   newID,
		OldPath: oldRelDir,
		NewPath: newRelDir,
	}, nil
}

// renumberTask renames a task file to use a new ID number.
func (w *Workspace) renumberTask(oldID, oldRelPath string) (*Renumbered, error) {
	newNum, err := w.DB.NextID("task")
	if err != nil {
		return nil, err
	}
	newID := fmt.Sprintf("TASK-%d", newNum)

	oldFileName := filepath.Base(oldRelPath)
	newFileName := newID + oldFileName[len(oldID):]
	dir := filepath.Dir(oldRelPath)
	newRelPath := filepath.Join(dir, newFileName)

	oldAbs := filepath.Join(w.Root, oldRelPath)
	newAbs := filepath.Join(w.Root, newRelPath)

	if err := os.Rename(oldAbs, newAbs); err != nil {
		return nil, fmt.Errorf("rename task file: %w", err)
	}

	return &Renumbered{
		OldID:   oldID,
		NewID:   newID,
		OldPath: oldRelPath,
		NewPath: newRelPath,
	}, nil
}

// renumberKB renames a KB file to use a new ID number.
func (w *Workspace) renumberKB(oldID, oldRelPath string) (*Renumbered, error) {
	newNum, err := w.DB.NextID("kb")
	if err != nil {
		return nil, err
	}
	newID := fmt.Sprintf("KB-%d", newNum)

	oldFileName := filepath.Base(oldRelPath)
	newFileName := newID + oldFileName[len(oldID):]
	newRelPath := filepath.Join("specd", "kb", newFileName)

	oldAbs := filepath.Join(w.Root, oldRelPath)
	newAbs := filepath.Join(w.Root, newRelPath)

	if err := os.Rename(oldAbs, newAbs); err != nil {
		return nil, fmt.Errorf("rename KB file: %w", err)
	}

	// Also rename clean HTML sidecar if it exists.
	if strings.HasSuffix(oldFileName, ".html") {
		oldClean := strings.TrimSuffix(oldAbs, ".html") + ".clean.html"
		newClean := strings.TrimSuffix(newAbs, ".html") + ".clean.html"
		if _, err := os.Stat(oldClean); err == nil {
			os.Rename(oldClean, newClean)
		}
	}

	return &Renumbered{
		OldID:   oldID,
		NewID:   newID,
		OldPath: oldRelPath,
		NewPath: newRelPath,
	}, nil
}
