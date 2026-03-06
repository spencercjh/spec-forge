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
