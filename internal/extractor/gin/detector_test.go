package gin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Error("expected non-nil detector")
	}
}

func TestDetector_parseGinVersion(t *testing.T) {
	tests := []struct {
		name     string
		goMod    string
		expected string
	}{
		{
			name: "has gin dependency",
			goMod: `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`,
			expected: "v1.9.1",
		},
		{
			name: "has gin - different version",
			goMod: `module test

go 1.21

require github.com/gin-gonic/gin v1.9.0
`,
			expected: "v1.9.0",
		},
		{
			name: "no gin - other framework",
			goMod: `module test

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			goModPath := filepath.Join(dir, "go.mod")
			os.WriteFile(goModPath, []byte(tt.goMod), 0o644)

			d := NewDetector()
			version, err := d.parseGinVersion(goModPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if version != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, version)
			}
		})
	}
}

func TestDetector_findMainFiles(t *testing.T) {
	dir := t.TempDir()

	// Create main.go
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}"), 0o644)

	// Create router.go with route registration
	os.WriteFile(filepath.Join(dir, "router.go"), []byte("package main\n\nfunc setupRouter() {}"), 0o644)

	// Create non-main file
	os.WriteFile(filepath.Join(dir, "utils.go"), []byte("package main\n\nfunc helper() {}"), 0o644)

	// Create vendor directory (should be excluded)
	os.MkdirAll(filepath.Join(dir, "vendor", "test"), 0o755)
	os.WriteFile(filepath.Join(dir, "vendor", "test", "main.go"), []byte("package main"), 0o644)

	d := NewDetector()
	files, err := d.findMainFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find main.go and router.go
	if len(files) < 1 {
		t.Errorf("expected at least 1 main file, got %d", len(files))
	}

	// Check main.go is found
	foundMain := false
	for _, f := range files {
		if filepath.Base(f) == "main.go" {
			foundMain = true
			break
		}
	}
	if !foundMain {
		t.Error("expected main.go to be found")
	}
}

func TestDetector_Detect(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) string
		wantErr     bool
		wantVersion string
		wantHasGin  bool
	}{
		{
			name: "valid gin project",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
				os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}"), 0o644)
				return dir
			},
			wantErr:     false,
			wantVersion: "v1.9.1",
			wantHasGin:  true,
		},
		{
			name: "missing go.mod",
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr: true,
		},
		{
			name: "no gin dependency",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/zeromicro/go-zero v1.6.0
`
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
				return dir
			},
			wantErr: true,
		},
		{
			name: "no go files",
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				goMod := `module test

go 1.21

require github.com/gin-gonic/gin v1.9.1
`
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
				return dir
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			d := NewDetector()
			info, err := d.Detect(dir)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if info.Framework != FrameworkName {
				t.Errorf("expected framework %q, got %q", FrameworkName, info.Framework)
			}
			ginInfo, ok := info.FrameworkData.(*Info)
			if !ok {
				t.Fatal("expected FrameworkData to be *gin.Info")
			}
			if ginInfo.GinVersion != tt.wantVersion {
				t.Errorf("expected version %q, got %q", tt.wantVersion, ginInfo.GinVersion)
			}
		})
	}
}
