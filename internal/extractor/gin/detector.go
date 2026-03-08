package gin

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

const (
	GinModule = "github.com/gin-gonic/gin"
	GoModFile = "go.mod"
)

// Detector detects Gin projects.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a project and returns info if it's a Gin project.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	slog.Debug("Detecting Gin project", "path", projectPath)

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		slog.Error("Failed to resolve path", "path", projectPath, "error", err)
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check for go.mod
	goModPath := filepath.Join(absPath, GoModFile)
	if _, statErr := os.Stat(goModPath); statErr != nil {
		slog.Warn("No go.mod found", "path", absPath)
		return nil, fmt.Errorf("no go.mod found in %s", absPath)
	}

	// Parse go.mod for Gin dependency
	ginVersion, err := d.parseGinVersion(goModPath)
	if err != nil {
		slog.Error("Failed to parse go.mod", "path", goModPath, "error", err)
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}
	if ginVersion == "" {
		slog.Warn("No gin dependency found in go.mod", "path", goModPath)
		return nil, errors.New("no gin dependency found in go.mod")
	}

	// Extract module name
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}
	modFile, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}
	moduleName := modFile.Module.Mod.Path

	// Find Go files
	mainFiles, err := d.findMainFiles(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find main files: %w", err)
	}
	if len(mainFiles) == 0 {
		return nil, fmt.Errorf("no .go files found in %s", absPath)
	}

	slog.Info("Detected Gin project", "module", moduleName, "version", ginVersion, "files", len(mainFiles))

	// Create Gin-specific info
	ginInfo := &Info{
		GoVersion:  "", // Will be filled if needed
		ModuleName: moduleName,
		GinVersion: ginVersion,
		HasGin:     true,
		MainFiles:  mainFiles,
	}

	return &extractor.ProjectInfo{
		Framework:     FrameworkName,
		BuildTool:     "gomodules",
		BuildFilePath: goModPath,
		FrameworkData: ginInfo,
	}, nil
}

// parseGinVersion parses go.mod and returns the gin version if present.
func (d *Detector) parseGinVersion(goModPath string) (string, error) {
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	modFile, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return "", err
	}

	for _, req := range modFile.Require {
		if req.Mod.Path == GinModule {
			return req.Mod.Version, nil
		}
	}

	return "", nil
}

// findMainFiles finds all Go files in the project (excluding vendor and tests).
func (d *Detector) findMainFiles(projectPath string) ([]string, error) {
	var mainFiles []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			if info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		mainFiles = append(mainFiles, path)
		return nil
	})

	return mainFiles, err
}
