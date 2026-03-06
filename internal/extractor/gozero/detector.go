// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Detector detects go-zero project information.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a go-zero project and returns its information.
func (d *Detector) Detect(projectPath string) (*ProjectInfo, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check for go.mod
	goModPath := filepath.Join(absPath, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no go.mod found in %s", absPath)
		}
		return nil, fmt.Errorf("failed to check go.mod: %w", err)
	}

	info := &ProjectInfo{
		BuildTool:     BuildToolGoModules,
		BuildFilePath: goModPath,
	}

	// Parse go.mod
	if err := d.parseGoMod(goModPath, info); err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	// Find .api files
	apiFiles, err := d.findAPIFiles(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find .api files: %w", err)
	}
	info.APIFiles = apiFiles

	// Check if goctl is available
	info.HasGoctl = d.checkGoctl()

	return info, nil
}

// parseGoMod parses the go.mod file and extracts project information.
func (d *Detector) parseGoMod(goModPath string, info *ProjectInfo) error {
	file, err := os.Open(goModPath)
	if err != nil {
		return fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inRequireBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Parse module name
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && fields[0] == "module" {
			info.ModuleName = fields[1]
			continue
		}

		// Parse go version
		if len(fields) >= 2 && fields[0] == "go" {
			info.GoVersion = fields[1]
			continue
		}

		// Detect require block start
		if strings.HasPrefix(trimmed, "require (") {
			inRequireBlock = true
			continue
		}

		// Detect require block end
		if inRequireBlock && trimmed == ")" {
			inRequireBlock = false
			continue
		}

		// Parse require statements (inside or outside block)
		d.parseRequireLine(trimmed, inRequireBlock, info)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading go.mod: %w", err)
	}

	return nil
}

// parseRequireLine parses a single require line and extracts go-zero dependency info.
func (d *Detector) parseRequireLine(line string, inRequireBlock bool, info *ProjectInfo) {
	fields := strings.Fields(line)

	// Handle single-line require: require github.com/zeromicro/go-zero v1.6.0
	if len(fields) >= 3 && fields[0] == "require" && !inRequireBlock {
		if strings.Contains(fields[1], "go-zero") {
			info.HasGoZeroDeps = true
			info.GoZeroVersion = fields[2]
		}
		return
	}

	// Handle require block entries: github.com/zeromicro/go-zero v1.6.0
	if inRequireBlock && len(fields) >= 2 {
		if strings.Contains(fields[0], "go-zero") {
			info.HasGoZeroDeps = true
			if len(fields) >= 2 {
				info.GoZeroVersion = fields[1]
			}
		}
	}
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
