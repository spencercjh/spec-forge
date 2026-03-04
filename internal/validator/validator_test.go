package validator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidator_Validate_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.json")

	validSpec := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {}
	}`

	if err := os.WriteFile(specFile, []byte(validSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	validator := NewValidator()
	result, err := validator.Validate(context.Background(), specFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidator_Validate_ValidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.yaml")

	validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths: {}
`

	if err := os.WriteFile(specFile, []byte(validSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	validator := NewValidator()
	result, err := validator.Validate(context.Background(), specFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
	}
}

func TestValidator_Validate_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.json")

	invalidSpec := `{ invalid json }`

	if err := os.WriteFile(specFile, []byte(invalidSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	validator := NewValidator()
	result, err := validator.Validate(context.Background(), specFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for malformed JSON")
	}

	if len(result.Errors) == 0 {
		t.Error("expected error messages")
	}
}

func TestValidator_Validate_MissingRequiredFields(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.json")

	// Missing required "info" field
	invalidSpec := `{
		"openapi": "3.0.0",
		"paths": {}
	}`

	if err := os.WriteFile(specFile, []byte(invalidSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	validator := NewValidator()
	result, err := validator.Validate(context.Background(), specFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for missing required fields")
	}
}

func TestValidator_Validate_FileNotFound(t *testing.T) {
	validator := NewValidator()
	_, err := validator.Validate(context.Background(), "/nonexistent/path/openapi.json")

	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestValidator_Validate_ComplexSpec(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.json")

	complexSpec := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Sample API",
			"version": "1.0.0",
			"description": "A sample API"
		},
		"paths": {
			"/users": {
				"get": {
					"summary": "List users",
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"application/json": {
									"schema": {
										"type": "array",
										"items": {
											"$ref": "#/components/schemas/User"
										}
									}
								}
							}
						}
					}
				}
			}
		},
		"components": {
			"schemas": {
				"User": {
					"type": "object",
					"properties": {
						"id": {
							"type": "integer"
						},
						"name": {
							"type": "string"
						}
					}
				}
			}
		}
	}`

	if err := os.WriteFile(specFile, []byte(complexSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	validator := NewValidator()
	result, err := validator.Validate(context.Background(), specFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Errorf("expected valid result for complex spec, got errors: %v", result.Errors)
	}
}

func TestValidator_Validate_InvalidReference(t *testing.T) {
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "openapi.json")

	// Invalid $ref to non-existent schema
	invalidSpec := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {
			"/users": {
				"get": {
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"application/json": {
									"schema": {
										"$ref": "#/components/schemas/NonExistent"
									}
								}
							}
						}
					}
				}
			}
		}
	}`

	if err := os.WriteFile(specFile, []byte(invalidSpec), 0o644); err != nil {
		t.Fatal(err)
	}

	validator := NewValidator()
	result, err := validator.Validate(context.Background(), specFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Valid {
		t.Error("expected invalid result for broken $ref")
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatValidationError(tt.err)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
