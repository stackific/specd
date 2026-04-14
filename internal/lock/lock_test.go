package lock

import (
	"path/filepath"
	"testing"
)

func TestAcquireAndRelease(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "test.lock")

	unlock, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Release the lock.
	unlock()

	// Should be able to acquire again.
	unlock2, err := Acquire(lockPath)
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	unlock2()
}
