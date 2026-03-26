//go:build e2e

package helpers

import "testing"

// AcquireGozeroLock acquires the global go-zero generation lock.
// This uses file-based locking (flock) to work across OS processes,
// which is necessary when running tests in multiple packages (go test ./...).
// The lock prevents concurrent access to the shared gozero-demo project directory,
// where goctl writes intermediate files (e.g., openapi.swagger.json).
// Call this at the start of any test that runs go-zero generation.
// The lock is automatically released when the test completes.
//
// On unsupported platforms (e.g., Windows), the test is skipped with a warning
// because cross-process locking is not available and file conflicts would occur.
//
// Example:
//
//	func TestGozeroGeneration(t *testing.T) {
//		helpers.AcquireGozeroLock(t)
//		// ... test code that runs goctl ...
//	}
func AcquireGozeroLock(t testing.TB) {
	AcquireFileLock(t, "gozero")
}
