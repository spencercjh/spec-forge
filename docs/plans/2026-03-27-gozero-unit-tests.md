# Go-Zero Unit Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add unit tests for `findMainAPIFile` and `patchSchema` functions to improve go-zero extractor test coverage from 55% toward 80%.

**Architecture:** Pure logic functions require no mocking - test file selection priority rules and recursive schema patching paths directly.

**Tech Stack:** Go 1.22+, stretchr/testify/assert, kin-openapi/openapi2, kin-openapi/openapi3

---

## File Structure

**Files to modify:**
- `internal/extractor/gozero/generator_test.go` - Add `findMainAPIFile` tests
- `internal/extractor/gozero/swagger_patch_test.go` - Add direct `patchSchema` tests

---

### Task 1: Add Unit Tests for `findMainAPIFile`

**Files:**
- Modify: `internal/extractor/gozero/generator_test.go`

**Function signature (generator.go:255-288):**
```go
func (g *Generator) findMainAPIFile(workDir string, info *Info, patchedFiles map[string]string) string
```

**Test scenarios to cover:**

- [ ] **Step 1: Write the failing tests**

Add to `internal/extractor/gozero/generator_test.go`:

```go
package gozero_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/spencercjh/spec-forge/internal/extractor/gozero"
)

func TestGenerator_FindMainAPIFile(t *testing.T) {
	g := gozero.NewGenerator()

	t.Run("returns empty string when no API files", func(t *testing.T) {
		info := &gozero.Info{APIFiles: []string{}}
		result := g.FindMainAPIFile("/work", info, nil)
		assert.Equal(t, "", result)
	})

	t.Run("prefers api directory file", func(t *testing.T) {
		workDir := "/project"
		info := &gozero.Info{
			APIFiles: []string{
				"/project/internal/handler.api",
				"/project/api/service.api",
				"/project/other.api",
			},
		}
		result := g.FindMainAPIFile(workDir, info, nil)
		assert.Equal(t, "/project/api/service.api", result)
	})

	t.Run("falls back to first file when no api directory", func(t *testing.T) {
		workDir := "/project"
		info := &gozero.Info{
			APIFiles: []string{
				"/project/internal/handler.api",
				"/project/cmd/api.api",
			},
		}
		result := g.FindMainAPIFile(workDir, info, nil)
		assert.Equal(t, "/project/internal/handler.api", result)
	})

	t.Run("returns patched path when available", func(t *testing.T) {
		workDir := "/project"
		info := &gozero.Info{
			APIFiles: []string{"/project/api/service.api"},
		}
		patchedFiles := map[string]string{
			"/project/api/service.api": "/tmp/service.api.patched",
		}
		result := g.FindMainAPIFile(workDir, info, patchedFiles)
		assert.Equal(t, "/tmp/service.api.patched", result)
	})

	t.Run("returns original path when not in patched map", func(t *testing.T) {
		workDir := "/project"
		info := &gozero.Info{
			APIFiles: []string{"/project/api/service.api"},
		}
		patchedFiles := map[string]string{
			"/project/other.api": "/tmp/other.api.patched",
		}
		result := g.FindMainAPIFile(workDir, info, patchedFiles)
		assert.Equal(t, "/project/api/service.api", result)
	})

	t.Run("handles nil patchedFiles map", func(t *testing.T) {
		workDir := "/project"
		info := &gozero.Info{
			APIFiles: []string{"/project/api/service.api"},
		}
		result := g.FindMainAPIFile(workDir, info, nil)
		assert.Equal(t, "/project/api/service.api", result)
	})

	t.Run("handles empty patchedFiles map", func(t *testing.T) {
		workDir := "/project"
		info := &gozero.Info{
			APIFiles: []string{"/project/api/service.api"},
		}
		result := g.FindMainAPIFile(workDir, info, map[string]string{})
		assert.Equal(t, "/project/api/service.api", result)
	})

	t.Run("handles api subdirectory", func(t *testing.T) {
		workDir := "/project"
		info := &gozero.Info{
			APIFiles: []string{
				"/project/api/v1/service.api",
				"/project/other.api",
			},
		}
		result := g.FindMainAPIFile(workDir, info, nil)
		assert.Equal(t, "/project/api/v1/service.api", result)
	})

	t.Run("handles cross-platform paths with backslashes", func(t *testing.T) {
		workDir := "C:\\project"
		info := &gozero.Info{
			APIFiles: []string{
				"C:\\project\\api\\service.api",
				"C:\\project\\other.api",
			},
		}
		// filepath.Rel should handle this on Windows; on Unix this may fail but won't crash
		result := g.FindMainAPIFile(workDir, info, nil)
		// On Unix, filepath.Rel may fail, falling back to first file
		// The test verifies no panic occurs
		assert.NotPanics(t, func() {
			g.FindMainAPIFile(workDir, info, nil)
		})
	})
}
```

- [ ] **Step 2: Export the function for testing**

The function `findMainAPIFile` is unexported. We need to either:
1. Export it as `FindMainAPIFile` (preferred for testing), or
2. Test it indirectly through the public interface

Modify `internal/extractor/gozero/generator.go` to export the function:

```go
// FindMainAPIFile finds the main API file to use for swagger generation.
// Returns the patched file path if available, otherwise returns the original.
// This method is exported for testing purposes.
func (g *Generator) FindMainAPIFile(workDir string, info *Info, patchedFiles map[string]string) string {
	return g.findMainAPIFile(workDir, info, patchedFiles)
}
```

Or alternatively, change the function name directly:

```go
// FindMainAPIFile finds the main API file to use for swagger generation.
// Returns the patched file path if available, otherwise returns the original.
func (g *Generator) FindMainAPIFile(workDir string, info *Info, patchedFiles map[string]string) string {
	if len(info.APIFiles) == 0 {
		return ""
	}

	var selectedFile string

	// Prefer API files in the api/ directory
	for _, apiFile := range info.APIFiles {
		relPath, err := filepath.Rel(workDir, apiFile)
		if err != nil {
			continue
		}
		// Use filepath.ToSlash for cross-platform consistency, then check for "api/" prefix
		normalizedPath := filepath.ToSlash(relPath)
		if strings.HasPrefix(normalizedPath, "api/") {
			selectedFile = apiFile
			break
		}
	}

	// Fallback to the first API file found
	if selectedFile == "" {
		selectedFile = info.APIFiles[0]
	}

	// Return patched path if available
	if patchedPath, ok := patchedFiles[selectedFile]; ok {
		return patchedPath
	}
	return selectedFile
}
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `go test -v ./internal/extractor/gozero/... -run TestGenerator_FindMainAPIFile`

Expected: All tests pass

- [ ] **Step 4: Commit**

```bash
git add internal/extractor/gozero/generator.go internal/extractor/gozero/generator_test.go
git commit -S -s -m "test(gozero): add unit tests for FindMainAPIFile function"
```

---

### Task 2: Add Direct Unit Tests for `patchSchema`

**Files:**
- Modify: `internal/extractor/gozero/swagger_patch_test.go`

**Function signature (swagger_patch.go:102-144):**
```go
func patchSchema(schema *openapi2.Schema)
```

**Note:** The `patchSchema` function is tested indirectly through `patchSwagger` tests. We should add direct tests for better coverage of edge cases and recursive paths.

**Test scenarios to cover:**

- [ ] **Step 1: Write the failing tests**

Add to `internal/extractor/gozero/swagger_patch_test.go`:

```go
func TestPatchSchema_NilSchema(t *testing.T) {
	// Should not panic
	patchSchema(nil)
}

func TestPatchSchema_ArrayWithoutItems(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"array"},
	}
	patchSchema(schema)

	assert.NotNil(t, schema.Items)
	assert.Equal(t, "object", schema.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_ArrayWithExistingItems(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"array"},
		Items: &openapi2.SchemaRef{
			Value: &openapi2.Schema{Type: &openapi3.Types{"string"}},
		},
	}
	patchSchema(schema)

	// Items should remain unchanged
	assert.Equal(t, "string", schema.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_NonArraySchema(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
	}
	patchSchema(schema)

	// Should not add items to non-array schema
	assert.Nil(t, schema.Items)
}

func TestPatchSchema_NestedProperties(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		Properties: map[string]*openapi2.SchemaRef{
			"items": {
				Value: &openapi2.Schema{
					Type: &openapi3.Types{"array"}, // Missing items
				},
			},
		},
	}
	patchSchema(schema)

	// Nested property should be patched
	assert.NotNil(t, schema.Properties["items"].Value.Items)
	assert.Equal(t, "object", schema.Properties["items"].Value.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_DeeplyNestedProperties(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		Properties: map[string]*openapi2.SchemaRef{
			"level1": {
				Value: &openapi2.Schema{
					Type: &openapi3.Types{"object"},
					Properties: map[string]*openapi2.SchemaRef{
						"level2": {
							Value: &openapi2.Schema{
								Type: &openapi3.Types{"array"}, // Missing items
							},
						},
					},
				},
			},
		},
	}
	patchSchema(schema)

	// Deeply nested property should be patched
	nested := schema.Properties["level1"].Value.Properties["level2"].Value
	assert.NotNil(t, nested.Items)
	assert.Equal(t, "object", nested.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_NilPropertyRef(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		Properties: map[string]*openapi2.SchemaRef{
			"nilProp": nil,
		},
	}
	// Should not panic
	patchSchema(schema)
}

func TestPatchSchema_ItemsRecursion(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"array"},
		Items: &openapi2.SchemaRef{
			Value: &openapi2.Schema{
				Type: &openapi3.Types{"array"}, // Nested array without items
			},
		},
	}
	patchSchema(schema)

	// Nested items should be patched
	assert.NotNil(t, schema.Items.Value.Items)
	assert.Equal(t, "object", schema.Items.Value.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_AllOfRecursion(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		AllOf: []*openapi2.SchemaRef{
			{
				Value: &openapi2.Schema{
					Type: &openapi3.Types{"array"}, // Missing items
				},
			},
		},
	}
	patchSchema(schema)

	// AllOf schema should be patched
	assert.NotNil(t, schema.AllOf[0].Value.Items)
	assert.Equal(t, "object", schema.AllOf[0].Value.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_AllOfNilRef(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		AllOf: []*openapi2.SchemaRef{
			nil,
			{Value: &openapi2.Schema{Type: &openapi3.Types{"string"}}},
		},
	}
	// Should not panic
	patchSchema(schema)
}

func TestPatchSchema_NotRecursion(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		Not: &openapi2.SchemaRef{
			Value: &openapi2.Schema{
				Type: &openapi3.Types{"array"}, // Missing items
			},
		},
	}
	patchSchema(schema)

	// Not schema should be patched
	assert.NotNil(t, schema.Not.Value.Items)
	assert.Equal(t, "object", schema.Not.Value.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_NotNilRef(t *testing.T) {
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		Not:  nil,
	}
	// Should not panic
	patchSchema(schema)
}

func TestPatchSchema_ComplexNestedStructure(t *testing.T) {
	// Test a complex schema with multiple levels of nesting
	schema := &openapi2.Schema{
		Type: &openapi3.Types{"object"},
		Properties: map[string]*openapi2.SchemaRef{
			"data": {
				Value: &openapi2.Schema{
					Type: &openapi3.Types{"array"},
					Items: &openapi2.SchemaRef{
						Value: &openapi2.Schema{
							Type: &openapi3.Types{"object"},
							Properties: map[string]*openapi2.SchemaRef{
								"tags": {
									Value: &openapi2.Schema{
										Type: &openapi3.Types{"array"}, // Missing items
									},
								},
							},
						},
					},
				},
			},
		},
	}
	patchSchema(schema)

	// Deeply nested tags array should be patched
	tagsSchema := schema.Properties["data"].Value.Items.Value.Properties["tags"].Value
	assert.NotNil(t, tagsSchema.Items)
	assert.Equal(t, "object", tagsSchema.Items.Value.Type.Slice()[0])
}

func TestPatchSchema_EmptySchema(t *testing.T) {
	schema := &openapi2.Schema{}
	// Should not panic
	patchSchema(schema)
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test -v ./internal/extractor/gozero/... -run TestPatchSchema`

Expected: All tests pass

- [ ] **Step 3: Commit**

```bash
git add internal/extractor/gozero/swagger_patch_test.go
git commit -S -s -m "test(gozero): add direct unit tests for patchSchema function"
```

---

### Task 3: Verify Coverage Improvement

- [ ] **Step 1: Run coverage report**

Run: `go test -coverprofile=coverage.out ./internal/extractor/gozero/... && go tool cover -func=coverage.out`

Expected: Coverage should increase from baseline 55.0%

- [ ] **Step 2: Verify specific function coverage**

Run: `go test -coverprofile=coverage.out ./internal/extractor/gozero/... && go tool cover -func=coverage.out | grep -E "(findMainAPIFile|patchSchema)"`

Expected: Both functions should show high coverage (ideally 80%+)

- [ ] **Step 4: Commit coverage verification (optional)**

```bash
git add docs/superpowers/plans/2026-03-27-gozero-unit-tests.md
git commit -S -s -m "docs: add go-zero unit test implementation plan"
```

---

## Self-Review Checklist

**1. Spec coverage:**
- [x] `findMainAPIFile` - Tests cover: empty list, api/ directory preference, fallback, patched files, nil/empty maps, subdirectory, cross-platform
- [x] `patchSchema` - Tests cover: nil schema, array with/without items, non-array, nested properties, deeply nested, nil refs, items recursion, allOf recursion, not recursion, complex structure

**2. Placeholder scan:**
- [x] No TBD, TODO, or "implement later" placeholders
- [x] All test code is complete
- [x] All commands are exact

**3. Type consistency:**
- [x] `Info` struct with `APIFiles []string` - consistent
- [x] `openapi2.Schema` types - consistent
- [x] Function names match between tasks

---

## Execution Options

Plan complete and saved to `docs/superpowers/plans/2026-03-27-gozero-unit-tests.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
