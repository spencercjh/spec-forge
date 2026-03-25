// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	forgeerrors "github.com/spencercjh/spec-forge/internal/errors"
	"github.com/spencercjh/spec-forge/internal/extractor"
)

// ErrNotGoZeroProject is returned when the project is not a go-zero project.
type ErrNotGoZeroProject struct {
	Reason string
}

func (e *ErrNotGoZeroProject) Error() string {
	return "not a go-zero project: " + e.Reason
}

// Detector detects go-zero project information.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a go-zero project and returns its information.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	slog.Debug("starting go-zero project detection", "path", projectPath)

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		slog.Error("failed to resolve project path", "path", projectPath, "error", err)
		return nil, forgeerrors.DetectError("failed to resolve path", err)
	}
	slog.Debug("resolved absolute path", "path", absPath)

	// Check for go.mod
	goModPath := filepath.Join(absPath, "go.mod")
	if _, statErr := os.Stat(goModPath); statErr != nil {
		if os.IsNotExist(statErr) {
			slog.Warn("no go.mod found", "path", absPath)
			return nil, forgeerrors.DetectError("no go.mod found in "+absPath, nil)
		}
		slog.Error("failed to check go.mod", "path", goModPath, "error", statErr)
		return nil, forgeerrors.DetectError("failed to check go.mod", statErr)
	}
	slog.Debug("found go.mod", "path", goModPath)

	goZeroInfo := &Info{}

	// Parse go.mod
	if parseErr := d.parseGoMod(goModPath, goZeroInfo); parseErr != nil {
		slog.Error("failed to parse go.mod", "path", goModPath, "error", parseErr)
		return nil, forgeerrors.DetectError("failed to parse go.mod", parseErr)
	}
	slog.Debug("parsed go.mod successfully",
		"module", goZeroInfo.ModuleName,
		"goVersion", goZeroInfo.GoVersion,
		"hasGoZeroDeps", goZeroInfo.HasGoZeroDeps,
		"goZeroVersion", goZeroInfo.GoZeroVersion)

	// Reject if no go-zero dependency found
	if !goZeroInfo.HasGoZeroDeps {
		slog.Warn("no go-zero dependency found", "path", goModPath)
		return nil, &ErrNotGoZeroProject{Reason: "no go-zero dependency found in go.mod"}
	}
	slog.Info("detected go-zero dependency", "version", goZeroInfo.GoZeroVersion)

	// Find .api files
	apiFiles, err := d.findAPIFiles(absPath)
	if err != nil {
		slog.Error("failed to find .api files", "path", absPath, "error", err)
		return nil, forgeerrors.DetectError("failed to find .api files", err)
	}
	slog.Debug("found .api files", "count", len(apiFiles), "files", apiFiles)

	// Reject if no .api files found
	if len(apiFiles) == 0 {
		slog.Warn("no .api files found in project", "path", absPath)
		return nil, &ErrNotGoZeroProject{Reason: "no .api files found in project"}
	}
	slog.Info("found .api files", "count", len(apiFiles))

	goZeroInfo.APIFiles = apiFiles

	// Check if goctl is available
	goZeroInfo.HasGoctl = d.checkGoctl()
	if goZeroInfo.HasGoctl {
		slog.Info("goctl is available")
	} else {
		slog.Warn("goctl not found in PATH", "hint", "install with: go install github.com/zeromicro/go-zero/tools/goctl@latest")
	}

	info := &extractor.ProjectInfo{
		Framework:     FrameworkGoZero,
		BuildTool:     BuildToolGoModules,
		BuildFilePath: goModPath,
		FrameworkData: goZeroInfo,
	}

	slog.Info("go-zero project detection completed successfully",
		"module", goZeroInfo.ModuleName,
		"apiFiles", len(apiFiles),
		"goctlAvailable", goZeroInfo.HasGoctl)

	return info, nil
}

// parseGoMod parses the go.mod file and extracts project information.
func (d *Detector) parseGoMod(goModPath string, info *Info) error {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	f, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return err
	}

	// Extract module name
	if f.Module != nil {
		info.ModuleName = f.Module.Mod.Path
	}

	// Extract Go version
	if f.Go != nil {
		info.GoVersion = f.Go.Version
	}

	// Check for go-zero dependency (exact match to avoid false positives)
	for _, req := range f.Require {
		if req != nil && req.Mod.Path == goZeroModulePath {
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
		if info.IsDir() && info.Name() == "vendor" {
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
	_, err := exec.LookPath("goctl")
	return err == nil
}
