// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spencercjh/spec-forge/internal/extractor"
	"golang.org/x/mod/modfile"
)

// Detector detects go-zero project information.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a go-zero project and returns its information.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check for go.mod
	goModPath := filepath.Join(absPath, "go.mod")
	if _, statErr := os.Stat(goModPath); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, fmt.Errorf("no go.mod found in %s", absPath)
		}
		return nil, fmt.Errorf("failed to check go.mod: %w", statErr)
	}

	goZeroInfo := &Info{}

	// Parse go.mod
	if parseErr := d.parseGoMod(goModPath, goZeroInfo); parseErr != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", parseErr)
	}

	// Find .api files
	apiFiles, err := d.findAPIFiles(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find .api files: %w", err)
	}
	goZeroInfo.APIFiles = apiFiles

	// Check if goctl is available
	goZeroInfo.HasGoctl = d.checkGoctl()

	info := &extractor.ProjectInfo{
		Framework:     FrameworkGoZero,
		BuildTool:     BuildToolGoModules,
		BuildFilePath: goModPath,
		FrameworkData: goZeroInfo,
	}

	return info, nil
}

// parseGoMod parses the go.mod file and extracts project information.
func (d *Detector) parseGoMod(goModPath string, info *Info) error {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	f, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod: %w", err)
	}

	// Extract module name
	if f.Module != nil {
		info.ModuleName = f.Module.Mod.Path
	}

	// Extract Go version
	if f.Go != nil {
		info.GoVersion = f.Go.Version
	}

	// Check for go-zero dependency
	for _, req := range f.Require {
		if req != nil && strings.Contains(req.Mod.Path, "go-zero") {
			info.HasGoZeroDeps = true
			info.GoZeroVersion = req.Mod.Version
			break
		}
	}

	return nil
}

// findAPIFiles walks the project directory and finds all .api files.
func (d *Detector) findAPIFiles(projectPath string) ([]string, error) {
	var apiFiles []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor directories
		if info.IsDir() && strings.Contains(path, "vendor") {
			return filepath.SkipDir
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Collect .api files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".api") {
			apiFiles = append(apiFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return apiFiles, nil
}

// checkGoctl checks if goctl command is available in PATH.
func (d *Detector) checkGoctl() bool {
	_, err := os.Stat("goctl")
	if err == nil {
		return true
	}

	// Check in PATH
	path, err := exec.LookPath("goctl")
	return err == nil && path != ""
}
