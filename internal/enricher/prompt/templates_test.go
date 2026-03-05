package prompt

import (
	"testing"
)

func TestTemplateType_Constants(t *testing.T) {
	tests := []struct {
		name     string
		ttype    TemplateType
		expected string
	}{
		{"api", TemplateTypeAPI, "api"},
		{"schema", TemplateTypeSchema, "schema"},
		{"param", TemplateTypeParam, "param"},
		{"response", TemplateTypeResponse, "response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ttype) != tt.expected {
				t.Errorf("TemplateType = %q, want %q", tt.ttype, tt.expected)
			}
		})
	}
}

func TestTemplateContext_Fields(t *testing.T) {
	ctx := TemplateContext{
		Type:         TemplateTypeAPI,
		Language:     "en",
		Path:         "GET /users/{id}",
		Method:       "GET",
		SchemaName:   "User",
		FieldName:    "userId",
		FieldType:    "string",
		ParamName:    "id",
		ResponseCode: "200",
		Required:     true,
	}

	if ctx.Type != TemplateTypeAPI {
		t.Errorf("Type = %v, want %v", ctx.Type, TemplateTypeAPI)
	}
	if ctx.Language != "en" {
		t.Errorf("Language = %q, want %q", ctx.Language, "en")
	}
	if ctx.Path != "GET /users/{id}" {
		t.Errorf("Path = %q, want %q", ctx.Path, "GET /users/{id}")
	}
}

func TestTemplate_Render(t *testing.T) {
	tmpl := &Template{
		System: "You are an API expert. Respond in {{.Language}}.",
		User:   "API: {{.Method}} {{.Path}}",
	}

	ctx := TemplateContext{
		Language: "en",
		Method:   "GET",
		Path:     "/users/{id}",
	}

	system, user, err := tmpl.Render(ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	expectedSystem := "You are an API expert. Respond in en."
	if system != expectedSystem {
		t.Errorf("System = %q, want %q", system, expectedSystem)
	}

	expectedUser := "API: GET /users/{id}"
	if user != expectedUser {
		t.Errorf("User = %q, want %q", user, expectedUser)
	}
}

func TestTemplate_RenderWithFields(t *testing.T) {
	tmpl := &Template{
		User: `Schema: {{.SchemaName}}
{{range .Fields}}- {{.Name}} ({{.Type}})
{{end}}`,
	}

	ctx := TemplateContext{
		SchemaName: "User",
		Fields: []FieldContext{
			{Name: "id", Type: "integer"},
			{Name: "name", Type: "string"},
		},
	}

	_, user, err := tmpl.Render(ctx)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Should contain schema name and field names
	if !containsAll(user, "User", "id", "name", "integer", "string") {
		t.Errorf("User prompt missing expected content: %s", user)
	}
}

func TestTemplate_RenderInvalidTemplate(t *testing.T) {
	tmpl := &Template{
		User: "{{.InvalidField}}",
	}

	ctx := TemplateContext{Language: "en"}

	_, _, err := tmpl.Render(ctx)
	if err == nil {
		t.Fatal("expected error for invalid template field")
	}
}

func TestTemplateManager_Get(t *testing.T) {
	mgr := NewTemplateManager()

	// Test getting API template
	tmpl, err := mgr.Get(TemplateTypeAPI)
	if err != nil {
		t.Fatalf("Get(API) error = %v", err)
	}
	if tmpl == nil {
		t.Fatal("Get(API) returned nil")
	}
	if tmpl.System == "" {
		t.Error("API template should have system prompt")
	}
	if tmpl.User == "" {
		t.Error("API template should have user prompt")
	}
}

func TestTemplateManager_GetAllTypes(t *testing.T) {
	mgr := NewTemplateManager()

	types := []TemplateType{TemplateTypeAPI, TemplateTypeSchema, TemplateTypeParam, TemplateTypeResponse}

	for _, tt := range types {
		t.Run(string(tt), func(t *testing.T) {
			tmpl, err := mgr.Get(tt)
			if err != nil {
				t.Fatalf("Get(%s) error = %v", tt, err)
			}
			if tmpl.System == "" {
				t.Errorf("%s template should have system prompt", tt)
			}
		})
	}
}

// Helper function
func containsAll(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if !contains(s, substr) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
