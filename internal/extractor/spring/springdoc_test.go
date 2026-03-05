package spring

import (
	"testing"

	"github.com/spencercjh/spec-forge/internal/extractor"
)

func TestBuildToolConstants(t *testing.T) {
	tests := []struct {
		name     string
		tool     extractor.BuildTool
		expected string
	}{
		{"maven", BuildToolMaven, "maven"},
		{"gradle", BuildToolGradle, "gradle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.tool) != tt.expected {
				t.Errorf("BuildTool %s = %s, want %s", tt.name, tt.tool, tt.expected)
			}
		})
	}
}
