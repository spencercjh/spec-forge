package gozero

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func TestPatchSwagger_ArrayItemsMissing(t *testing.T) {
	tests := []struct {
		name     string
		schema   *openapi2.Schema
		wantType string
	}{
		{
			name: "array without items gets object type",
			schema: &openapi2.Schema{
				Type: &openapi3.Types{"array"},
			},
			wantType: "object",
		},
		{
			name: "array with existing items is unchanged",
			schema: &openapi2.Schema{
				Type: &openapi3.Types{"array"},
				Items: &openapi2.SchemaRef{
					Value: &openapi2.Schema{Type: &openapi3.Types{"string"}},
				},
			},
			wantType: "string",
		},
		{
			name: "non-array type is unchanged",
			schema: &openapi2.Schema{
				Type: &openapi3.Types{"object"},
			},
			wantType: "",
		},
		{
			name: "nested array in properties gets patched",
			schema: &openapi2.Schema{
				Type: &openapi3.Types{"object"},
				Properties: map[string]*openapi2.SchemaRef{
					"items": {
						Value: &openapi2.Schema{
							Type: &openapi3.Types{"array"},
						},
					},
				},
			},
			wantType: "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &openapi2.T{
				Definitions: map[string]*openapi2.SchemaRef{
					"TestSchema": {Value: tt.schema},
				},
			}

			PatchSwagger(doc)

			if tt.schema.Type != nil && tt.schema.Type.Is("array") {
				if tt.schema.Items != nil {
					assert.Equal(t, tt.wantType, tt.schema.Items.Value.Type.Slice()[0])
				}
			}

			// Check nested property
			if prop, ok := tt.schema.Properties["items"]; ok {
				assert.NotNil(t, prop.Value.Items)
				assert.Equal(t, "object", prop.Value.Items.Value.Type.Slice()[0])
			}
		})
	}
}

func TestPatchSwagger_SkipDashParameters(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		params         openapi2.Parameters
		wantParamCount int
		wantNames      []string
	}{
		{
			name: "parameters with dash name are removed",
			path: "/test/{id}",
			params: openapi2.Parameters{
				{Name: "id", In: "path"},
				{Name: "-", In: "query"},
				{Name: "name", In: "query"},
			},
			wantParamCount: 2,
			wantNames:      []string{"id", "name"},
		},
		{
			name: "all dash parameters are removed",
			path: "/test",
			params: openapi2.Parameters{
				{Name: "-", In: "query"},
				{Name: "-", In: "path"},
			},
			wantParamCount: 0,
			wantNames:      []string{},
		},
		{
			name: "normal parameters are kept",
			path: "/test/{id}",
			params: openapi2.Parameters{
				{Name: "id", In: "path"},
				{Name: "limit", In: "query"},
			},
			wantParamCount: 2,
			wantNames:      []string{"id", "limit"},
		},
		{
			name:           "empty parameters stay empty",
			path:           "/test",
			params:         openapi2.Parameters{},
			wantParamCount: 0,
			wantNames:      []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &openapi2.T{
				Paths: map[string]*openapi2.PathItem{
				tt.path: {
						Get: &openapi2.Operation{
							Parameters: tt.params,
						},
					},
				},
			}

			PatchSwagger(doc)

			gotParams := doc.Paths[tt.path].Get.Parameters
			assert.Len(t, gotParams, tt.wantParamCount)

			for i, name := range tt.wantNames {
				if i < len(gotParams) {
					assert.Equal(t, name, gotParams[i].Name)
				}
			}
		})
	}
}

func TestPatchSwagger_RemoveInvalidPathParams(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		params         openapi2.Parameters
		wantParamCount int
		wantNames      []string
	}{
		{
			name:           "valid path params are kept",
			path:           "/users/{id}",
			params:         openapi2.Parameters{{Name: "id", In: "path"}},
			wantParamCount: 1,
			wantNames:      []string{"id"},
		},
		{
			name: "invalid path params are removed",
			path: "/users/{id}",
			params: openapi2.Parameters{
				{Name: "id", In: "path"},
				{Name: "invalid", In: "path"},
			},
			wantParamCount: 1,
			wantNames:      []string{"id"},
		},
		{
			name: "query params are not affected",
			path: "/users/{id}",
			params: openapi2.Parameters{
				{Name: "id", In: "path"},
				{Name: "limit", In: "query"},
			},
			wantParamCount: 2,
			wantNames:      []string{"id", "limit"},
		},
		{
			name:           "multiple path params",
			path:           "/users/{userId}/posts/{postId}",
			params:         openapi2.Parameters{{Name: "userId", In: "path"}, {Name: "postId", In: "path"}},
			wantParamCount: 2,
			wantNames:      []string{"userId", "postId"},
		},
		{
			name: "extra path params are removed",
			path: "/users/{userId}/posts/{postId}",
			params: openapi2.Parameters{
				{Name: "userId", In: "path"},
				{Name: "postId", In: "path"},
				{Name: "extra", In: "path"},
			},
			wantParamCount: 2,
			wantNames:      []string{"userId", "postId"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &openapi2.T{
				Paths: map[string]*openapi2.PathItem{
					tt.path: {
						Get: &openapi2.Operation{
							Parameters: tt.params,
						},
					},
				},
			}

			PatchSwagger(doc)

			gotParams := doc.Paths[tt.path].Get.Parameters
			assert.Len(t, gotParams, tt.wantParamCount)

			for i, name := range tt.wantNames {
				if i < len(gotParams) {
					assert.Equal(t, name, gotParams[i].Name)
				}
			}
		})
	}
}

func TestPatchSwagger_AllPatchesTogether(t *testing.T) {
	// Test all three patches working together
	params := openapi2.Parameters{
		{Name: "userId", In: "path"},
		{Name: "invalid", In: "path"}, // Should be removed (#5428)
		{Name: "-", In: "query"},      // Should be removed (#5427)
		{Name: "limit", In: "query"},
	}

	doc := &openapi2.T{
		Paths: map[string]*openapi2.PathItem{
			"/users/{userId}": {
				Get: &openapi2.Operation{
					Parameters: params,
					Responses: map[string]*openapi2.Response{
						"200": {
							Schema: &openapi2.SchemaRef{
								Value: &openapi2.Schema{
									Type: &openapi3.Types{"array"}, // Missing items - should be patched (#5426)
								},
							},
						},
					},
				},
			},
		},
		Definitions: map[string]*openapi2.SchemaRef{
			"UserList": {
				Value: &openapi2.Schema{
					Type: &openapi3.Types{"array"}, // Missing items - should be patched (#5426)
				},
			},
		},
	}

	PatchSwagger(doc)

	// Check parameters
	gotParams := doc.Paths["/users/{userId}"].Get.Parameters
	assert.Len(t, gotParams, 2)
	assert.Equal(t, "userId", gotParams[0].Name)
	assert.Equal(t, "limit", gotParams[1].Name)

	// Check response schema
	respSchema := doc.Paths["/users/{userId}"].Get.Responses["200"].Schema.Value
	assert.NotNil(t, respSchema.Items)
	assert.Equal(t, "object", respSchema.Items.Value.Type.Slice()[0])

	// Check definition schema
	defSchema := doc.Definitions["UserList"].Value
	assert.NotNil(t, defSchema.Items)
	assert.Equal(t, "object", defSchema.Items.Value.Type.Slice()[0])
}

func TestPatchSwagger_NilDoc(t *testing.T) {
	// Should not panic
	PatchSwagger(nil)
}

func TestPatchSwagger_EmptyDoc(t *testing.T) {
	// Should not panic
	doc := &openapi2.T{}
	PatchSwagger(doc)
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "single param",
			path: "/users/{id}",
			want: []string{"id"},
		},
		{
			name: "multiple params",
			path: "/users/{userId}/posts/{postId}",
			want: []string{"userId", "postId"},
		},
		{
			name: "no params",
			path: "/users",
			want: []string{},
		},
		{
			name: "empty path",
			path: "",
			want: []string{},
		},
		{
			name: "param at start",
			path: "{version}/api/users",
			want: []string{"version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathParams(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		value string
		want  bool
	}{
		{
			name:  "value exists",
			slice: []string{"a", "b", "c"},
			value: "b",
			want:  true,
		},
		{
			name:  "value does not exist",
			slice: []string{"a", "b", "c"},
			value: "d",
			want:  false,
		},
		{
			name:  "empty slice",
			slice: []string{},
			value: "a",
			want:  false,
		},
		{
			name:  "nil slice",
			slice: nil,
			value: "a",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.slice, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPatchSwagger_NestedArrayItems(t *testing.T) {
	// Test deeply nested array items
	doc := &openapi2.T{
		Definitions: map[string]*openapi2.SchemaRef{
			"NestedArray": {
				Value: &openapi2.Schema{
					Type: &openapi3.Types{"object"},
					Properties: map[string]*openapi2.SchemaRef{
						"level1": {
							Value: &openapi2.Schema{
								Type: &openapi3.Types{"array"},
								Items: &openapi2.SchemaRef{
									Value: &openapi2.Schema{
										Type: &openapi3.Types{"array"}, // Nested array without items
									},
								},
							},
						},
					},
				},
			},
		},
	}

	PatchSwagger(doc)

	// Check that nested array items are patched
	nestedItems := doc.Definitions["NestedArray"].Value.Properties["level1"].Value.Items.Value.Items
	assert.NotNil(t, nestedItems)
	assert.Equal(t, "object", nestedItems.Value.Type.Slice()[0])
}

func TestPatchSwagger_AllOperations(t *testing.T) {
	// Test that all HTTP methods are patched
	params := openapi2.Parameters{
		{Name: "-", In: "query"}, // Should be removed
	}

	doc := &openapi2.T{
		Paths: map[string]*openapi2.PathItem{
			"/test": {
				Get:     &openapi2.Operation{Parameters: params},
				Post:    &openapi2.Operation{Parameters: openapi2.Parameters{{Name: "-", In: "query"}}},
				Put:     &openapi2.Operation{Parameters: openapi2.Parameters{{Name: "-", In: "query"}}},
				Delete:  &openapi2.Operation{Parameters: openapi2.Parameters{{Name: "-", In: "query"}}},
				Patch:   &openapi2.Operation{Parameters: openapi2.Parameters{{Name: "-", In: "query"}}},
				Head:    &openapi2.Operation{Parameters: openapi2.Parameters{{Name: "-", In: "query"}}},
				Options: &openapi2.Operation{Parameters: openapi2.Parameters{{Name: "-", In: "query"}}},
			},
		},
	}

	PatchSwagger(doc)

	// All operations should have empty parameters
	assert.Empty(t, doc.Paths["/test"].Get.Parameters)
	assert.Empty(t, doc.Paths["/test"].Post.Parameters)
	assert.Empty(t, doc.Paths["/test"].Put.Parameters)
	assert.Empty(t, doc.Paths["/test"].Delete.Parameters)
	assert.Empty(t, doc.Paths["/test"].Patch.Parameters)
	assert.Empty(t, doc.Paths["/test"].Head.Parameters)
	assert.Empty(t, doc.Paths["/test"].Options.Parameters)
}
