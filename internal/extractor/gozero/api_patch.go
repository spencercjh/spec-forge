// Package gozero provides go-zero framework specific extraction functionality.
package gozero

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/spencercjh/spec-forge/internal/executor"
)

// minGoctlVersionForPatch is the minimum goctl version that fixes issue #5425.
// goctl 1.9.2+ has fixed the multi-hyphen prefix parsing issue.
const minGoctlVersionForPatch = "1.9.2"

// versionCheckTimeout is the timeout for goctl version check.
const versionCheckTimeout = 5 * time.Second

// APIFilePatcher patches .api files to work around goctl parser bugs.
// See: https://github.com/zeromicro/go-zero/issues/5425
type APIFilePatcher struct {
	// patchedFiles maps original file paths to temporary patched file paths
	patchedFiles map[string]string
	// skipPatch indicates whether patching should be skipped (goctl version >= 1.9.2)
	skipPatch bool
	// exec is the command executor
	exec executor.Interface
}

// NewAPIFilePatcher creates a new APIFilePatcher.
func NewAPIFilePatcher() *APIFilePatcher {
	return NewAPIFilePatcherWithExecutor(executor.NewExecutor())
}

// NewAPIFilePatcherWithExecutor creates a new APIFilePatcher with a custom executor.
// This is primarily used for testing.
func NewAPIFilePatcherWithExecutor(exec executor.Interface) *APIFilePatcher {
	patcher := &APIFilePatcher{
		patchedFiles: make(map[string]string),
		skipPatch:    false,
		exec:         exec,
	}

	// Check goctl version to determine if patching is needed
	if version, err := patcher.getGoctlVersion(); err == nil {
		// Use semantic version comparison (add "v" prefix for semver.Compare)
		patcher.skipPatch = semver.Compare("v"+version, "v"+minGoctlVersionForPatch) >= 0
		if patcher.skipPatch {
			slog.Debug("goctl version >= 1.9.2, skipping API file patching (issue #5425 fixed upstream)")
		}
	} else {
		slog.Debug("failed to get goctl version, will apply patching", "error", err)
	}

	return patcher
}

// getGoctlVersion returns the goctl version string (e.g., "1.9.2").
func (p *APIFilePatcher) getGoctlVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()

	opts := &executor.ExecuteOptions{
		Command: "goctl",
		Args:    []string{"--version"},
	}

	result, err := p.exec.Execute(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to get goctl version: %w", err)
	}

	// Parse version from output
	// goctl --version output: "goctl version 1.9.2 darwin/arm64"
	fields := strings.Fields(result.Stdout)
	if len(fields) < 3 {
		return "", fmt.Errorf("unexpected goctl version output: %s", result.Stdout)
	}

	// fields[0] = "goctl", fields[1] = "version", fields[2] = "1.9.2"
	return fields[2], nil
}

// PatchAPIFiles scans and patches .api files to fix known issues.
// Only patches files if goctl version < 1.9.2 (issue #5425 fixed in goctl 1.9.2+).
// Returns the path to the patched file (may be same as original if no patching needed).
func (p *APIFilePatcher) PatchAPIFiles(apiFiles []string) (map[string]string, error) {
	slog.Debug("checking .api files for patching", "count", len(apiFiles))
	result := make(map[string]string)

	patchedCount := 0

	// Skip patching entirely if goctl version >= 1.9.2
	if p.skipPatch {
		slog.Debug("skipping API file patching (goctl version >= 1.9.2)")
		for _, apiFile := range apiFiles {
			result[apiFile] = apiFile
		}
		return result, nil
	}

	for _, apiFile := range apiFiles {
		needsPatch, err := p.checkNeedsPatch(apiFile)
		if err != nil {
			slog.Error("failed to check API file", "file", apiFile, "error", err)
			return nil, fmt.Errorf("failed to check %s: %w", apiFile, err)
		}

		if !needsPatch {
			result[apiFile] = apiFile
			continue
		}

		slog.Info("patching API file for goctl compatibility", "file", apiFile, "issue", "#5425")

		// Create temporary patched file
		patchedPath, err := p.createPatchedFile(apiFile)
		if err != nil {
			slog.Error("failed to patch API file", "file", apiFile, "error", err)
			return nil, fmt.Errorf("failed to patch %s: %w", apiFile, err)
		}

		result[apiFile] = patchedPath
		p.patchedFiles[apiFile] = patchedPath
		patchedCount++
	}

	if patchedCount > 0 {
		slog.Info("patched API files for goctl compatibility", "count", patchedCount)
	} else {
		slog.Debug("no API files needed patching")
	}

	return result, nil
}

// Cleanup removes temporary patched files.
func (p *APIFilePatcher) Cleanup() {
	if len(p.patchedFiles) == 0 {
		return
	}
	slog.Debug("cleaning up patched API files", "count", len(p.patchedFiles))
	for _, patchedPath := range p.patchedFiles {
		if patchedPath != "" {
			_ = os.Remove(patchedPath)
		}
	}
	p.patchedFiles = make(map[string]string)
}

// checkNeedsPatch checks if the .api file needs patching.
// Detects #5425: prefix values with multi-hyphen without quotes.
func (p *APIFilePatcher) checkNeedsPatch(apiFile string) (bool, error) {
	content, err := os.ReadFile(apiFile)
	if err != nil {
		return false, err
	}

	// Pattern: prefix: some-value-with-hyphens (without quotes)
	// This regex matches prefix: followed by unquoted text containing hyphens
	re := regexp.MustCompile(`(?m)^\s*prefix:\s*([^"\s]\S*-\S*[^"\s]*)`)

	return re.Match(content), nil
}

// createPatchedFile creates a patched version of the .api file.
// Fixes #5425 by wrapping unquoted multi-hyphen prefix values in quotes.
func (p *APIFilePatcher) createPatchedFile(apiFile string) (string, error) {
	file, err := os.Open(apiFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create temp file in same directory
	dir := filepath.Dir(apiFile)
	tempFile, err := os.CreateTemp(dir, "*.api.patched")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Regex to find unquoted prefix values with hyphens
	re := regexp.MustCompile(`^(\s*prefix:\s*)([^"\s]\S*-\S*[^"\s]*)`)

	scanner := bufio.NewScanner(file)
	writer := bufio.NewWriter(tempFile)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check if this line needs patching
		if matches := re.FindStringSubmatch(line); matches != nil {
			// Wrap the value in quotes
			patchedLine := re.ReplaceAllString(line, `${1}"${2}"`)
			line = patchedLine
		}

		if _, err := writer.WriteString(line + "\n"); err != nil {
			return "", err
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if err := writer.Flush(); err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

// GetPatchedPath returns the patched path for an original API file.
func (p *APIFilePatcher) GetPatchedPath(originalPath string) string {
	if patched, ok := p.patchedFiles[originalPath]; ok {
		return patched
	}
	return originalPath
}

// HasPatchedFiles returns true if any files were patched.
func (p *APIFilePatcher) HasPatchedFiles() bool {
	return len(p.patchedFiles) > 0
}

// ValidateAPIFile checks for issues that cannot be automatically patched.
// Returns an error with helpful message if manual intervention is needed.
func ValidateAPIFile(apiFile string) error {
	content, err := os.ReadFile(apiFile)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// Check for complex #5425 cases that might not be patchable
	// Look for prefix lines with multiple hyphens but without quotes
	lines := strings.Split(contentStr, "\n")
	for i, line := range lines {
		// Skip already quoted values
		if strings.Contains(line, `prefix: "`) {
			continue
		}

		// Check for unquoted prefix with multiple hyphens
		if strings.Contains(line, "prefix:") && strings.Contains(line, "-") {
			// Extract the value after prefix
			_, after, found := strings.Cut(line, "prefix:")
			if !found {
				continue
			}
			value := strings.TrimSpace(after)

			// If value is not quoted and contains hyphens, it might cause issues
			if value != "" && value[0] != '"' && strings.Count(value, "-") > 1 {
				return fmt.Errorf("API file %s line %d: unquoted prefix value with multiple hyphens detected: %s\n"+
					"This is a known goctl bug (#5425). Please wrap the value in quotes:\n"+
					"  prefix: \"%s\"", apiFile, i+1, value, value)
			}
		}
	}

	return nil
}
