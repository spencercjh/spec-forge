//go:build e2e

package helpers

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
)

// springGenMu is an in-process mutex to serialize tests within the same process.
var springGenMu sync.Mutex

// lockFile is the file descriptor for the cross-process lock file.
var lockFile *os.File

// AcquireSpringLock acquires the global Spring generation lock.
// This uses file-based locking (flock) to work across OS processes,
// which is necessary when running tests in multiple packages (go test ./...).
// Call this at the start of any test that runs Spring Boot generation.
// The lock is automatically released when the test completes.
//
// On unsupported platforms (e.g., Windows), the test is skipped with a warning
// because cross-process locking is not available and port conflicts would occur.
//
// Example:
//
//	func TestSpringGeneration(t *testing.T) {
//		helpers.AcquireSpringLock(t)
//		// ... test code that runs Spring Boot ...
//	}
func AcquireSpringLock(t testing.TB) {
	// First acquire in-process lock for tests within the same process
	springGenMu.Lock()
	t.Cleanup(springGenMu.Unlock)

	// Then acquire file-based lock for cross-process synchronization
	acquireFileLock(t)
}

// acquireFileLock acquires a file-based lock using flock.
// The lock file is created in the system temp directory.
// On unsupported platforms, the test is skipped.
func acquireFileLock(t testing.TB) {
	// Check if platform supports flock (Unix-only)
	if runtime.GOOS == "windows" {
		t.Skip("Spring Boot tests skipped on Windows: cross-process file locking not supported")
	}

	if lockFile == nil {
		// Create lock file in system temp directory
		tmpDir := os.TempDir()
		if tmpDir == "" {
			tmpDir = "/tmp"
		}
		lockPath := filepath.Join(tmpDir, "spec-forge-spring-test.lock")

		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			t.Fatalf("failed to create spring lock file: %v", err)
		}
		lockFile = f
	}

	// Acquire exclusive lock (blocks until available)
	err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX)
	if err != nil {
		t.Skipf("Spring Boot test skipped: failed to acquire cross-process lock: %v", err)
	}

	// Release lock when test completes
	t.Cleanup(func() {
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	})
}
