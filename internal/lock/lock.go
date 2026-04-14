package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/flock"
)

const (
	Timeout    = 5 * time.Second
	retryDelay = 50 * time.Millisecond
)

// Acquire takes an exclusive flock on the given path with a 5-second timeout.
// Returns an unlock function. The caller must call unlock when done.
func Acquire(lockPath string) (unlock func(), err error) {
	fl := flock.New(lockPath)

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	locked, err := fl.TryLockContext(ctx, retryDelay)
	if err != nil {
		return nil, fmt.Errorf("acquire lock %s: %w", lockPath, err)
	}
	if !locked {
		return nil, fmt.Errorf("could not acquire lock %s within %s", lockPath, Timeout)
	}

	return func() {
		fl.Unlock()
	}, nil
}
