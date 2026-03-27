//go:build e2e

package e2e_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/spencercjh/spec-forge/cmd"
)

// e2eConfig represents the E2E test configuration structure
type e2eConfig struct {
	Enrich struct {
		Enabled   bool   `yaml:"enabled"`
		Provider  string `yaml:"provider"`
		Model     string `yaml:"model"`
		BaseURL   string `yaml:"baseUrl"`
		APIKeyEnv string `yaml:"apiKeyEnv"`
		Language  string `yaml:"language"`
		Timeout   string `yaml:"timeout"`
	} `yaml:"enrich"`
}

// loadE2EConfig loads the E2E test configuration from a local config file.
// It checks for .spec-forge.e2e.local.yaml in the project root.
// Returns nil if the config file doesn't exist or is invalid.
func loadE2EConfig(t *testing.T) *e2eConfig {
	t.Helper()

	// Look for the local E2E config file
	configPaths := []string{
		".spec-forge.e2e.local.yaml", // Preferred: dedicated E2E config
		".spec-forge.local.yaml",     // Alternative: local config
	}

	for _, configPath := range configPaths {
		// Check in current directory and parent directories
		for i := 0; i < 3; i++ {
			path := configPath
			if i > 0 {
				path = filepath.Join(strings.Repeat("../", i), configPath)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			var cfg e2eConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				t.Logf("Failed to parse %s: %v", path, err)
				continue
			}

			// Validate config has required fields
			if cfg.Enrich.Provider == "" || cfg.Enrich.Model == "" {
				t.Logf("Config %s missing provider or model", path)
				continue
			}

			// Check if API key is available
			apiKeyEnv := cfg.Enrich.APIKeyEnv
			if apiKeyEnv == "" {
				apiKeyEnv = "LLM_API_KEY"
			}
			if os.Getenv(apiKeyEnv) == "" {
				t.Logf("Config %s found but %s not set", path, apiKeyEnv)
				continue
			}

			t.Logf("Using E2E config from %s (provider=%s, model=%s)",
				path, cfg.Enrich.Provider, cfg.Enrich.Model)

			return &cfg
		}
	}

	return nil
}

// skipIfNoConfig skips the test if no valid E2E config is found.
func skipIfNoConfig(t *testing.T) *e2eConfig {
	t.Helper()

	cfg := loadE2EConfig(t)
	if cfg == nil {
		t.Skip("Skipping: no valid E2E config found. " +
			"Create .spec-forge.e2e.local.yaml with LLM settings and set the API key env var.")
	}

	return cfg
}

// TestE2E_Enrich_Help tests the enrich command help output.
func TestE2E_Enrich_Help(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"enrich", "--help"})

	err := rootCmd.Execute()
	require.NoError(t, err, "command failed: %s", stderr.String())

	output := stdout.String()

	// Verify help contains expected content
	assert.Contains(t, output, "enrich", "Expected 'enrich' in help output")
	assert.Contains(t, output, "--provider", "Expected '--provider' flag in help")
	assert.Contains(t, output, "--model", "Expected '--model' flag in help")
	assert.Contains(t, output, "--language", "Expected '--language' flag in help")
	assert.Contains(t, output, "--no-stream", "Expected '--no-stream' flag in help (P4.1 feature)")

	t.Log("Enrich help test passed!")
}

// TestE2E_Enrich_MissingArgs tests error handling when spec file is not provided.
func TestE2E_Enrich_MissingArgs(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"enrich"}) // No spec file

	err := rootCmd.Execute()

	// Should fail with error
	require.Error(t, err, "Expected error for missing spec file argument")
	t.Logf("Got expected error: %v", err)
}

// TestE2E_Enrich_NonExistentFile tests error handling for non-existent spec file.
func TestE2E_Enrich_NonExistentFile(t *testing.T) {
	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"enrich", "/nonexistent/spec.yaml"})

	err := rootCmd.Execute()

	// Should fail because file doesn't exist
	require.Error(t, err, "Expected error for non-existent spec file")
	t.Logf("Got expected error: %v", err)
}

// TestE2E_Enrich_NoStreamFlag tests that --no-stream flag is accepted.
func TestE2E_Enrich_NoStreamFlag(t *testing.T) {
	cfg := skipIfNoConfig(t)

	// Create a minimal test spec
	specContent := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths:
  /health:
    get:
      summary: ""
      operationId: healthCheck
      responses:
        "200":
          description: ""
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")
	require.NoError(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	args := []string{
		"enrich",
		specFile,
		"--provider", cfg.Enrich.Provider,
		"--model", cfg.Enrich.Model,
		"--language", cfg.Enrich.Language,
		"--no-stream", // Disable streaming
	}
	if cfg.Enrich.BaseURL != "" {
		args = append(args, "--custom-base-url", cfg.Enrich.BaseURL)
	}
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	require.NoError(t, err, "enrich command failed: %s", stderr.String())

	// Verify output doesn't contain streaming prefixes (since --no-stream)
	output := stdout.String()
	assert.NotContains(t, output, "[api]", "Should not have streaming prefixes with --no-stream")
	assert.NotContains(t, output, "[schema]", "Should not have streaming prefixes with --no-stream")

	t.Log("Enrich with --no-stream succeeded!")
}

// TestE2E_Enrich_WithStreaming tests real LLM enrichment with streaming enabled (default).
// This test requires a valid E2E config with LLM settings.
func TestE2E_Enrich_WithStreaming(t *testing.T) {
	cfg := skipIfNoConfig(t)

	// Create a test spec with empty descriptions that need enrichment
	specContent := `openapi: "3.0.0"
info:
  title: User Management API
  version: "1.0"
paths:
  /users:
    get:
      summary: ""
      description: ""
      operationId: listUsers
      parameters:
        - name: page
          in: query
          schema:
            type: integer
          description: ""
        - name: pageSize
          in: query
          schema:
            type: integer
          description: ""
      responses:
        "200":
          description: ""
  /users/{id}:
    get:
      summary: ""
      description: ""
      operationId: getUserById
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
          description: ""
      responses:
        "200":
          description: ""
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          description: ""
        name:
          type: string
          description: ""
        email:
          type: string
          description: ""
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")
	require.NoError(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	args := []string{
		"enrich",
		specFile,
		"--provider", cfg.Enrich.Provider,
		"--model", cfg.Enrich.Model,
		"--language", cfg.Enrich.Language,
		"-v", // Verbose output
	}
	if cfg.Enrich.BaseURL != "" {
		args = append(args, "--custom-base-url", cfg.Enrich.BaseURL)
	}
	rootCmd.SetArgs(args)

	start := time.Now()
	err := rootCmd.Execute()
	duration := time.Since(start)

	require.NoError(t, err, "enrich command failed: %s", stderr.String())

	// Verify streaming output contains expected prefixes
	output := stdout.String()
	t.Logf("Enrich output (took %v):\n%s", duration, output)

	// Check for streaming prefixes (indicates streaming is working)
	streamingPrefixes := []string{"[api]", "[schema]", "[param]"}
	foundAnyPrefix := false
	for _, prefix := range streamingPrefixes {
		if strings.Contains(output, prefix) {
			foundAnyPrefix = true
			t.Logf("Found streaming prefix: %s", prefix)
		}
	}
	assert.True(t, foundAnyPrefix, "Expected at least one streaming prefix in output")

	// Verify the spec file was enriched
	enrichedData, err := os.ReadFile(specFile)
	require.NoError(t, err, "Failed to read enriched spec")

	enrichedContent := string(enrichedData)
	t.Logf("Enriched spec:\n%s", enrichedContent)

	// Check that descriptions were added (not empty anymore)
	assert.Contains(t, enrichedContent, "description:", "Spec should have descriptions after enrichment")

	// Verify API operations got descriptions
	assert.NotContains(t, enrichedContent, "summary: \"\"", "Summary should not be empty after enrichment")

	t.Log("Enrich with streaming succeeded!")
}

// TestE2E_Enrich_WithLocalConfig tests using a local .spec-forge.yaml config file.
// This verifies the config file loading mechanism works in E2E scenarios.
func TestE2E_Enrich_WithLocalConfig(t *testing.T) {
	cfg := skipIfNoConfig(t)

	// Create a test spec
	specContent := `openapi: "3.0.0"
info:
  title: Config Test API
  version: "1.0"
paths:
  /test:
    get:
      summary: ""
      operationId: testEndpoint
      responses:
        "200":
          description: ""
`
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "test-spec.yaml")
	require.NoError(t, os.WriteFile(specFile, []byte(specContent), 0o644))

	// Create a local config file in the same directory
	localConfig := "enrich:\n"
	if cfg.Enrich.Provider != "" {
		localConfig += "  provider: " + cfg.Enrich.Provider + "\n"
	}
	if cfg.Enrich.Model != "" {
		localConfig += "  model: " + cfg.Enrich.Model + "\n"
	}
	if cfg.Enrich.BaseURL != "" {
		localConfig += "  baseUrl: " + cfg.Enrich.BaseURL + "\n"
	}
	if cfg.Enrich.APIKeyEnv != "" {
		localConfig += "  apiKeyEnv: " + cfg.Enrich.APIKeyEnv + "\n"
	}
	if cfg.Enrich.Language != "" {
		localConfig += "  language: " + cfg.Enrich.Language + "\n"
	}

	configFile := filepath.Join(tmpDir, ".spec-forge.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(localConfig), 0o644))

	// Change to the temp directory so the config is picked up
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalDir) }()

	rootCmd := cmd.NewRootCommand()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	// Don't specify provider/model - should use config file
	rootCmd.SetArgs([]string{"enrich", "test-spec.yaml", "-v"})

	err = rootCmd.Execute()
	require.NoError(t, err, "enrich with local config failed: %s", stderr.String())

	t.Log("Enrich with local config succeeded!")
}
