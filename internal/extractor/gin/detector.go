package gin

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
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
		return nil, forgeerrors.DetectError("failed to resolve path", err)
	}

	// Check for go.mod
	goModPath := filepath.Join(absPath, GoModFile)
	if _, statErr := os.Stat(goModPath); statErr != nil {
		slog.Warn("No go.mod found", "path", absPath)
		return nil, forgeerrors.DetectError("no go.mod found in "+absPath, nil)
	}

	// Parse go.mod for Gin dependency
	ginVersion, err := d.parseGinVersion(goModPath)
	if err != nil {
		slog.Error("Failed to parse go.mod", "path", goModPath, "error", err)
		return nil, forgeerrors.DetectError("failed to parse go.mod", err)
	}
	if ginVersion == "" {
		slog.Warn("No gin dependency found in go.mod", "path", goModPath)
		return nil, forgeerrors.DetectError("no gin dependency found in go.mod", nil)
	}

	// Extract module name
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, forgeerrors.DetectError("failed to read go.mod", err)
	}
	modFile, err := modfile.Parse("go.mod", content, nil)
	if err != nil {
		return nil, forgeerrors.DetectError("failed to parse go.mod", err)
	}
	moduleName := modFile.Module.Mod.Path

	// Find Go files
	mainFiles, err := d.findMainFiles(absPath)
	if err != nil {
		return nil, forgeerrors.DetectError("failed to find main files", err)
	}
	if len(mainFiles) == 0 {
		return nil, forgeerrors.DetectError("no .go files found in "+absPath, nil)
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
		Framework:     ExtractorName,
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

		// Skip vendor and hidden directories (but not the root itself when path is ".")
		if info.IsDir() {
			if path != projectPath &&
				(info.Name() == "vendor" || strings.HasPrefix(info.Name(), ".")) {
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
