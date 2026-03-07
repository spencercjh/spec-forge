// Package grpcprotoc provides gRPC-protoc framework extraction functionality.
package grpcprotoc

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

// ErrBufProjectDetected is returned when a buf.yaml is found in the project.
var ErrBufProjectDetected = errors.New(
	"buf.yaml detected: this is a buf-managed project. " +
		"spec-forge currently only supports native protoc projects for gRPC. " +
		"Please use 'buf generate' with protoc-gen-connect-openapi, " +
		"then use 'spec-forge enrich' on the generated OpenAPI spec")

// ErrNotProtocProject is returned when the project is not a valid protoc project.
type ErrNotProtocProject struct {
	Reason string
}

func (e *ErrNotProtocProject) Error() string {
	return "not a protoc project: " + e.Reason
}

// Detector detects gRPC-protoc project information.
type Detector struct{}

// NewDetector creates a new Detector instance.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect analyzes a gRPC-protoc project and returns its information.
func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error) {
	slog.Debug("starting gRPC-protoc project detection", "path", projectPath)

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		slog.Error("failed to resolve project path", "path", projectPath, "error", err)
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}
	slog.Debug("resolved absolute path", "path", absPath)

	// Check if path exists
	if _, statErr := os.Stat(absPath); statErr != nil {
		if os.IsNotExist(statErr) {
			slog.Error("project path does not exist", "path", absPath)
			return nil, fmt.Errorf("project path does not exist: %s", absPath)
		}
		slog.Error("failed to check project path", "path", absPath, "error", statErr)
		return nil, fmt.Errorf("failed to check project path: %w", statErr)
	}

	// Check for buf.yaml - reject buf-managed projects
	bufYamlPath := filepath.Join(absPath, "buf.yaml")
	if _, statErr := os.Stat(bufYamlPath); statErr == nil {
		slog.Warn("buf.yaml detected, rejecting buf-managed project", "path", bufYamlPath)
		return nil, ErrBufProjectDetected
	}

	// Find all .proto files
	protoFiles, err := d.findProtoFiles(absPath)
	if err != nil {
		slog.Error("failed to find .proto files", "path", absPath, "error", err)
		return nil, fmt.Errorf("failed to find .proto files: %w", err)
	}
	slog.Debug("found .proto files", "count", len(protoFiles), "files", protoFiles)

	// Reject if no .proto files found
	if len(protoFiles) == 0 {
		slog.Warn("no .proto files found in project", "path", absPath)
		return nil, &ErrNotProtocProject{Reason: "no .proto files found in project"}
	}
	slog.Info("found .proto files", "count", len(protoFiles))

	// Detect import paths
	importPaths := d.detectImportPaths(absPath, protoFiles)
	slog.Debug("detected import paths", "paths", importPaths)

	// Check for google.api.http annotations
	hasGoogleAPI := d.checkGoogleAPIAnnotations(protoFiles)
	if hasGoogleAPI {
		slog.Info("detected google.api.http annotations in project")
	}

	// Find proto files with service definitions (main entry points)
	serviceProtoFiles := d.findServiceProtoFiles(protoFiles)
	slog.Debug("found service proto files", "count", len(serviceProtoFiles), "files", serviceProtoFiles)

	// Build info
	grpcInfo := &Info{
		ProtoFiles:        protoFiles,
		ServiceProtoFiles: serviceProtoFiles,
		ProtoRoot:         absPath,
		HasGoogleAPI:      hasGoogleAPI,
		HasBuf:            false,
		ImportPaths:       importPaths,
	}

	info := &extractor.ProjectInfo{
		Framework:     FrameworkName,
		BuildTool:     BuildToolProtoc,
		BuildFilePath: absPath,
		FrameworkData: grpcInfo,
	}

	slog.Info("gRPC-protoc project detection completed successfully",
		"protoFiles", len(protoFiles),
		"hasGoogleAPI", hasGoogleAPI,
		"importPaths", len(importPaths))

	return info, nil
}

// findProtoFiles walks the project directory and finds all .proto files.
func (d *Detector) findProtoFiles(projectPath string) ([]string, error) {
	var protoFiles []string

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

		// Collect .proto files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".proto") {
			protoFiles = append(protoFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return protoFiles, nil
}

// detectImportPaths identifies directories that should be used as import paths for protoc.
func (d *Detector) detectImportPaths(projectPath string, protoFiles []string) []string {
	// Common proto directory names to check
	knownProtoDirs := []string{"proto", "third_party", "protos"}

	// Use a map to avoid duplicates
	pathSet := make(map[string]bool)
	var importPaths []string

	// Always include project root
	importPaths = append(importPaths, projectPath)
	pathSet[projectPath] = true

	// Check for known proto directories
	for _, dirName := range knownProtoDirs {
		dirPath := filepath.Join(projectPath, dirName)
		if _, err := os.Stat(dirPath); err == nil {
			if !pathSet[dirPath] {
				importPaths = append(importPaths, dirPath)
				pathSet[dirPath] = true
			}
		}
	}

	// Also include directories containing proto files
	for _, protoFile := range protoFiles {
		dirPath := filepath.Dir(protoFile)
		if !pathSet[dirPath] {
			importPaths = append(importPaths, dirPath)
			pathSet[dirPath] = true
		}
	}

	return importPaths
}

// checkGoogleAPIAnnotations scans proto files for google/api/annotations.proto imports.
func (d *Detector) checkGoogleAPIAnnotations(protoFiles []string) bool {
	return slices.ContainsFunc(protoFiles, d.hasGoogleAPIImport)
}

// hasGoogleAPIImport checks if a proto file imports google/api/annotations.proto.
func (d *Detector) hasGoogleAPIImport(protoFile string) bool {
	file, err := os.Open(protoFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Check for import "google/api/annotations.proto"
		if strings.Contains(line, "google/api/annotations.proto") {
			return true
		}
		// Also check for import 'google/api/annotations.proto' (single quotes)
		if strings.Contains(line, "google/api/annotations.proto") {
			return true
		}
	}

	return false
}

// findServiceProtoFiles identifies proto files that contain service definitions.
// These are the main entry points that should be passed to protoc.
func (d *Detector) findServiceProtoFiles(protoFiles []string) []string {
	var serviceFiles []string
	for _, protoFile := range protoFiles {
		if d.hasServiceDefinition(protoFile) {
			serviceFiles = append(serviceFiles, protoFile)
		}
	}
	return serviceFiles
}

// hasServiceDefinition checks if a proto file contains a service definition.
func (d *Detector) hasServiceDefinition(protoFile string) bool {
	file, err := os.Open(protoFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Check for service keyword (not inside a comment)
		if strings.HasPrefix(line, "service ") || strings.HasPrefix(line, "service\t") {
			return true
		}
	}

	return false
}
