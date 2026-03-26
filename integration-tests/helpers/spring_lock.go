//go:build e2e

package helpers

import "testing"

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
	AcquireFileLock(t, "spring")
}
