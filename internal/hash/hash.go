// Package hash provides SHA-256 content hashing for change detection.
// The watcher and CLI use content hashes to determine whether a file
// has been modified since the last known state in SQLite.
package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// File returns the hex-encoded SHA-256 hash of the file at path.
func File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Bytes returns the hex-encoded SHA-256 hash of the given bytes.
func Bytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// String returns the hex-encoded SHA-256 hash of the given string.
func String(s string) string {
	return Bytes([]byte(s))
}
