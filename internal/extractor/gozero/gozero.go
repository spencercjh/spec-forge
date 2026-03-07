// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import "github.com/spencercjh/spec-forge/internal/extractor"

const (
	// BuildToolGoModules represents Go modules build tool.
	BuildToolGoModules extractor.BuildTool = "gomodules"
)

// goctl constants used across the package.
const (
	goZeroModulePath = "github.com/zeromicro/go-zero"
)
