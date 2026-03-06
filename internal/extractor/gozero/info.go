// Package gozero provides go-zero framework specific extraction functionality.
package gozero

// Info contains go-zero framework specific information.
type Info struct {
	GoVersion     string   // Go version
	GoModule      string   // Go module path
	ModuleName    string   // Go module name
	HasGoZeroDeps bool     // Whether go-zero dependencies exist
	GoZeroVersion string   // go-zero version if any
	HasGoctl      bool     // Whether goctl is available
	APIFiles      []string // List of .api file paths
	MainPackage   string   // Main package path
}
