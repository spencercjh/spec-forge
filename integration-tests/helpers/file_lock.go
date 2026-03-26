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

// fileLocks stores file descriptors for cross-process lock files, keyed by lock name.
var fileLocks = struct {
	mu    sync.Mutex
	files map[string]*os.File
}{
	files: make(map[string]*os.File),
}

// AcquireFileLock acquires a cross-process file lock for the given name.
// This is useful for serializing tests that share resources (e.g., project directories,
// ports) across multiple test packages when running `go test ./...`.
//
// The lock uses flock (Unix-only) for cross-process synchronization and an in-process
// mutex for serialization within the same process. The lock is automatically released
// when the test completes.
//
// On unsupported platforms (e.g., Windows), the test is skipped with a warning
// because cross-process locking is not available.
//
// Example:
//
//	func TestMyGeneration(t *testing.T) {
//		helpers.AcquireFileLock(t, "my-resource")
//		// ... test code that needs serialization ...
//	}
func AcquireFileLock(t testing.TB, name string) {
	// In-process mutex map for tests within the same process
	processMu := getProcessMutex(name)
	processMu.Lock()
	t.Cleanup(processMu.Unlock)

	// File-based lock for cross-process synchronization
	acquireFileLock(t, name)
}

// processMutexes stores in-process mutexes, keyed by lock name.
var processMutexes = struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}{
	locks: make(map[string]*sync.Mutex),
}

// getProcessMutex returns (or creates) a mutex for the given lock name.
func getProcessMutex(name string) *sync.Mutex {
	processMutexes.mu.Lock()
	defer processMutexes.mu.Unlock()

	if mu, ok := processMutexes.locks[name]; ok {
		return mu
	}

	mu := &sync.Mutex{}
	processMutexes.locks[name] = mu
	return mu
}

// acquireFileLock acquires a file-based lock using flock.
// The lock file is created in the system temp directory.
// On unsupported platforms, the test is skipped.
func acquireFileLock(t testing.TB, name string) {
	// Check if platform supports flock (Unix-only)
	if runtime.GOOS == "windows" {
		t.Skip("tests skipped on Windows: cross-process file locking not supported")
	}

	fileLocks.mu.Lock()
	defer fileLocks.mu.Unlock()

	// Check if we already have a file descriptor for this lock
	lockFile, exists := fileLocks.files[name]
	if !exists {
		// Create lock file in system temp directory
		tmpDir := os.TempDir()
		if tmpDir == "" {
			tmpDir = "/tmp"
		}
		lockPath := filepath.Join(tmpDir, "spec-forge-"+name+".lock")

		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			t.Fatalf("failed to create lock file %s: %v", lockPath, err)
		}
		lockFile = f
		fileLocks.files[name] = lockFile
	}

	// Acquire exclusive lock (blocks until available)
	err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX)
	if err != nil {
		t.Skipf("test skipped: failed to acquire cross-process lock: %v", err)
	}

	// Release lock when test completes
	t.Cleanup(func() {
		syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)
	})
}
