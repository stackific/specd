package hash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestString(t *testing.T) {
	h := String("hello")
	// Well-known SHA-256 of "hello".
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h != want {
		t.Errorf("String(\"hello\") = %s, want %s", h, want)
	}
}

func TestBytes(t *testing.T) {
	h := Bytes([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h != want {
		t.Errorf("Bytes = %s, want %s", h, want)
	}
}

func TestFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.txt")
	os.WriteFile(p, []byte("hello"), 0o644)

	h, err := File(p)
	if err != nil {
		t.Fatalf("File: %v", err)
	}

	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if h != want {
		t.Errorf("File = %s, want %s", h, want)
	}
}

func TestFileNotFound(t *testing.T) {
	_, err := File("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
