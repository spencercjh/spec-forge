//go:build e2e

package helpers

import (
	"sync"
)

// springGenMu is a cross-package mutex to serialize Spring Boot generation tests.
// Spring Boot applications bind to port 8080 by default, and running multiple
// generate commands concurrently causes port binding conflicts and flaky CI failures.
// All tests that invoke Spring Boot generation (via spec-forge generate or directly
// calling Maven/Gradle with springdoc) should acquire this lock before execution.
var springGenMu sync.Mutex

// AcquireSpringLock acquires the global Spring generation lock.
// Call this at the start of any test that runs Spring Boot generation.
// Always defer the release call to ensure cleanup on test failure.
//
// Example:
//
//	func TestSpringGeneration(t *testing.T) {
//		helpers.AcquireSpringLock(t)
//		// ... test code that runs Spring Boot ...
//	}
func AcquireSpringLock(t interface{ Cleanup(func()) }) {
	springGenMu.Lock()
	t.Cleanup(springGenMu.Unlock)
}

// AcquirePackageSpringLock acquires the global Spring generation lock for the
// lifetime of the package (until all tests in the package complete).
// This is useful for packages that need to ensure no concurrent Spring Boot
// tests run from other packages. Use this in an init() function.
//
// Example:
//
//	func init() {
//		helpers.AcquirePackageSpringLock()
//	}
func AcquirePackageSpringLock() {
	springGenMu.Lock()
	// Note: This lock is released when the process exits
	// For fine-grained control, prefer AcquireSpringLock(t)
}
