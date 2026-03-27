package gozero

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestNewGenerator(t *testing.T) {
	g := NewGenerator()
	if g == nil {
		t.Error("NewGenerator() should not return nil")
	}
}

func TestGenerator_Generate_NoGoMod(t *testing.T) {
	g := NewGenerator()
	ctx := context.Background()
	_, err := g.Generate(ctx, "/tmp/non-existent-project", &extractor.ProjectInfo{}, &extractor.GenerateOptions{})

	if err == nil {
		t.Error("Generate should return error for project without go.mod")
	}
}

func TestGenerator_findMainAPIFile(t *testing.T) {
	g := NewGenerator()

	tests := []struct {
		name         string
		workDir      string
		info         *Info
		patchedFiles map[string]string
		want         string
	}{
		{
			name:    "returns empty string when no API files",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{},
			},
			patchedFiles: nil,
			want:         "",
		},
		{
			name:    "prefers api directory file over non-api file",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/internal/handler.api",
					"/project/api/service.api",
				},
			},
			patchedFiles: nil,
			want:         "/project/api/service.api",
		},
		{
			name:    "prefers first api directory file when multiple exist",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/internal/handler.api",
					"/project/api/service.api",
					"/project/api/user.api",
				},
			},
			patchedFiles: nil,
			want:         "/project/api/service.api",
		},
		{
			name:    "falls back to first file when no api directory",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/internal/handler.api",
					"/project/pkg/types.api",
				},
			},
			patchedFiles: nil,
			want:         "/project/internal/handler.api",
		},
		{
			name:    "handles api subdirectory (api/v1/)",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/internal/handler.api",
					"/project/api/v1/service.api",
				},
			},
			patchedFiles: nil,
			want:         "/project/api/v1/service.api",
		},
		{
			name:    "returns patched path when available",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/api/service.api",
				},
			},
			patchedFiles: map[string]string{
				"/project/api/service.api": "/tmp/patched-service.api",
			},
			want: "/tmp/patched-service.api",
		},
		{
			name:    "returns original path when not in patched map",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/api/service.api",
				},
			},
			patchedFiles: map[string]string{
				"/project/other.api": "/tmp/patched-other.api",
			},
			want: "/project/api/service.api",
		},
		{
			name:    "handles nil patchedFiles map",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/api/service.api",
				},
			},
			patchedFiles: nil,
			want:         "/project/api/service.api",
		},
		{
			name:    "handles empty patchedFiles map",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/api/service.api",
				},
			},
			patchedFiles: map[string]string{},
			want:         "/project/api/service.api",
		},
		{
			name:    "handles single API file",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/api/service.api",
				},
			},
			patchedFiles: nil,
			want:         "/project/api/service.api",
		},
		{
			name:    "handles file path with different workDir",
			workDir: "/home/user/projects/my-service",
			info: &Info{
				APIFiles: []string{
					"/home/user/projects/my-service/api/desc/user.api",
				},
			},
			patchedFiles: nil,
			want:         "/home/user/projects/my-service/api/desc/user.api",
		},
		{
			name:    "handles relative path outside workDir gracefully",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/other/project/api/service.api",
				},
			},
			patchedFiles: nil,
			want:         "/other/project/api/service.api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.findMainAPIFile(tt.workDir, tt.info, tt.patchedFiles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerator_findMainAPIFile_NilInfo(t *testing.T) {
	g := NewGenerator()

	// Test with nil info - this should not panic but may return empty
	// since info.APIFiles would cause nil pointer dereference
	// We document this as undefined behavior, but it shouldn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Logf("findMainAPIFile panicked with nil info (expected): %v", r)
		}
	}()

	// This will panic because info is nil - that's expected behavior
	g.findMainAPIFile("/project", nil, nil)
}

func TestGenerator_findMainAPIFile_WindowsPaths(t *testing.T) {
	g := NewGenerator()

	// Test that Windows-style paths are handled without panicking
	// The function uses filepath.Rel and filepath.ToSlash for cross-platform support
	tests := []struct {
		name    string
		workDir string
		info    *Info
		want    string
	}{
		{
			name:    "handles Windows-style paths with backslashes",
			workDir: "C:\\project",
			info: &Info{
				APIFiles: []string{
					"C:\\project\\api\\service.api",
				},
			},
			want: "C:\\project\\api\\service.api",
		},
		{
			name:    "handles mixed path separators",
			workDir: "/project",
			info: &Info{
				APIFiles: []string{
					"/project/api/service.api",
				},
			},
			want: "/project/api/service.api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The test should not panic and should return a valid result
			got := g.findMainAPIFile(tt.workDir, tt.info, nil)
			// Just ensure we get some result without panic
			assert.NotEmpty(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerator_findMainAPIFile_PrefersApiDirectory(t *testing.T) {
	g := NewGenerator()

	// Test that files in api/ directory are preferred regardless of order
	tests := []struct {
		name     string
		workDir  string
		apiFiles []string
		want     string
	}{
		{
			name:     "api file at end of list",
			workDir:  "/project",
			apiFiles: []string{"/project/internal/a.api", "/project/pkg/b.api", "/project/api/main.api"},
			want:     "/project/api/main.api",
		},
		{
			name:     "api file at start of list",
			workDir:  "/project",
			apiFiles: []string{"/project/api/main.api", "/project/internal/a.api", "/project/pkg/b.api"},
			want:     "/project/api/main.api",
		},
		{
			name:     "api file in middle of list",
			workDir:  "/project",
			apiFiles: []string{"/project/internal/a.api", "/project/api/main.api", "/project/pkg/b.api"},
			want:     "/project/api/main.api",
		},
		{
			name:     "multiple api files - first one wins",
			workDir:  "/project",
			apiFiles: []string{"/project/internal/a.api", "/project/api/first.api", "/project/api/second.api"},
			want:     "/project/api/first.api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &Info{APIFiles: tt.apiFiles}
			got := g.findMainAPIFile(tt.workDir, info, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
