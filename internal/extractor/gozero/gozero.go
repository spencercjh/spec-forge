// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import "github.com/spencercjh/spec-forge/internal/extractor"

const (
	// BuildToolGoModules represents Go modules build tool.
	BuildToolGoModules extractor.BuildTool = "gomodules"
)

// Default version constants (convention over configuration).
const (
	DefaultGoctlVersion = "1.7.0"
)

// goctl constants used across the package.
const (
	GoctlPackage = "github.com/zeromicro/go-zero/tools/goctl"
)

// ProjectInfo contains detected information about a go-zero project.
type ProjectInfo struct {
	BuildTool     extractor.BuildTool // Always "gomodules" for go-zero
	BuildFilePath string              // Path to go.mod
	ModuleName    string              // Module name from go.mod
	GoVersion     string              // Go version from go.mod
	HasGoZeroDeps bool                // Whether go-zero dependencies exist
	GoZeroVersion string              // go-zero version if detected
	APIFiles      []string            // List of .api file paths
	HasGoctl      bool                // Whether goctl is available
}
