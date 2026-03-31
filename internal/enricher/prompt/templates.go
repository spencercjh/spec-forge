package prompt

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// TemplateType defines the type of enrichment template
type TemplateType string

const (
	TemplateTypeAPI      TemplateType = "api"
	TemplateTypeSchema   TemplateType = "schema"
	TemplateTypeParam    TemplateType = "param"
	TemplateTypeResponse TemplateType = "response"
)

// FieldContext provides specctx for a schema field
type FieldContext struct {
	Name                string
	Type                string
	Required            bool
	Format              string   // e.g., "email", "date-time", "uuid"
	Enum                []string // allowed values, e.g., ["active", "inactive"]
	Constraints         string   // human-readable: "min: 0, max: 100, pattern: ^[a-z]+$"
	ExistingDescription string   // existing description from the spec, if any
}

// ParamFieldContext provides specctx for a parameter in a group.
type ParamFieldContext struct {
	Name                string
	Type                string
	ParamIn             string // path, query, header, cookie
	Required            bool
	Format              string   // e.g., "int32", "uuid"
	Enum                []string // allowed values
	Constraints         string   // human-readable validation rules
	ExistingDescription string   // existing description from the spec, if any
}

// TemplateContext provides specctx for template rendering
type TemplateContext struct {
	Type     TemplateType
	Language string

	// API specctx
	Path                string
	Method              string
	Tags                []string
	ExistingSummary     string
	ExistingDescription string

	// Schema specctx
	SchemaName string
	Fields     []FieldContext

	// Field specctx
	FieldName string
	FieldType string
	Required  bool

	// Parameter specctx
	ParamName   string
	ParamIn     string // path, query, header, cookie
	ParamFields []ParamFieldContext

	// Response specctx
	ResponseCode string
}

// Template represents a prompt template with system and user prompts
type Template struct {
	System string
	User   string
}

// Render renders the template with the given specctx
func (t *Template) Render(ctx TemplateContext) (system, user string, err error) { //nolint:gocritic // copying specctx is acceptable here
	if t.System != "" {
		system, err = renderString(t.System, ctx)
		if err != nil {
			return "", "", err
		}
	}

	user, err = renderString(t.User, ctx)
	if err != nil {
		return "", "", err
	}

	return system, user, nil
}

func renderString(tmpl string, data any) (string, error) {
	t, err := template.New("prompt").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// TemplateManager manages built-in and custom templates
type TemplateManager struct {
	templates map[TemplateType]*Template
}

// NewTemplateManager creates a new template manager with built-in templates
func NewTemplateManager() *TemplateManager {
	return &TemplateManager{
		templates: map[TemplateType]*Template{
			TemplateTypeAPI: {
				System: `You are an expert OpenAPI documentation writer specializing in REST API descriptions.
Your task is to write clear, concise, and informative API summaries and descriptions.

Guidelines:
- Summary: A single line (max 80 chars) starting with a verb (e.g., "List", "Create", "Delete")
- Description: 1-3 sentences explaining what the endpoint does, when to use it, and notable behavior
- Be specific: mention resource names, ID formats, and key constraints
- Avoid generic phrases like "This API is used for..."

Respond in {{.Language}} language.
Output MUST be valid JSON: {"summary": "...", "description": "..."}

Example input:
  POST /users
Example output:
  {"summary": "Create a new user", "description": "Registers a new user account in the system. The request body must include a valid email address and password. Returns the created user object with a generated ID."}`,
				User: `API Endpoint: {{.Method}} {{.Path}}
{{- if .Tags}}
Tags: {{join .Tags ", "}}
{{- end}}
{{- if .ExistingSummary}}
Existing summary: {{.ExistingSummary}}
{{- end}}
{{- if .ExistingDescription}}
Existing description: {{.ExistingDescription}}
{{- end}}

Generate the summary (one line) and description (1-3 sentences) for this API.`,
			},
			TemplateTypeSchema: {
				System: `You are an expert OpenAPI data model documenter.
Your task is to write concise, precise field descriptions for API data schemas.

Guidelines:
- Each description should be 1-2 sentences explaining what the field represents
- Mention constraints (format, range, pattern) when relevant to clarify the field's purpose
- For enum fields, briefly explain what the values represent if the field name alone isn't clear
- Avoid repeating the field name in the description
- Be specific about units, formats, and expected values

Respond in {{.Language}} language.
Output MUST be valid JSON mapping field names to descriptions: {"field1": "description1", "field2": "description2", ...}

Example input:
  Schema: User
  Fields:
  - email (string, required, format: email, maxLength: 255)
  - role (string, optional, enum: [admin, user, guest])
Example output:
  {"email": "The user's primary email address used for authentication and notifications", "role": "The user's permission level determining access to system features"}`,
				User: `Schema: {{.SchemaName}}
Fields:
{{range .Fields}}- {{.Name}} ({{.Type}}, {{if .Required}}required{{else}}optional{{end}}{{if .Format}}, format: {{.Format}}{{end}}{{if .Constraints}}, {{.Constraints}}{{end}}{{if .Enum}}, enum: [{{join .Enum ", "}}]{{end}}{{if .ExistingDescription}}, existing: "{{.ExistingDescription}}"{{end}})
{{end}}
Generate a description for each field.`,
			},
			TemplateTypeParam: {
				System: `You are an expert API parameter documenter.
Your task is to write concise, precise parameter descriptions for REST API endpoints.

Guidelines:
- Each description should be 1-2 sentences explaining what the parameter controls
- Mention the parameter location context (path, query, header) when it affects behavior
- For enum parameters, briefly describe what the allowed values represent
- Include the unit or format when relevant (e.g., "page number starting from 1")
- Avoid generic descriptions like "the X parameter"

Respond in {{.Language}} language.
Output MUST be valid JSON mapping parameter names to descriptions: {"param1": "description1", ...}

Example input:
  API: GET /users
  Parameters:
  - page (integer, in: query, optional)
  - status (string, in: query, optional, enum: [active, inactive])
Example output:
  {"page": "Page number for pagination, starting from 1. Defaults to 1 if not specified.", "status": "Filter users by account status. Use 'active' for current users or 'inactive' for deactivated accounts."}`,
				User: `API: {{.Method}} {{.Path}}
Parameters:
{{range .ParamFields}}- {{.Name}} ({{.Type}}, in: {{.ParamIn}}, {{if .Required}}required{{else}}optional{{end}}{{if .Format}}, format: {{.Format}}{{end}}{{if .Constraints}}, {{.Constraints}}{{end}}{{if .Enum}}, enum: [{{join .Enum ", "}}]{{end}}{{if .ExistingDescription}}, existing: "{{.ExistingDescription}}"{{end}})
{{end}}
Generate a description for each parameter.`,
			},
			TemplateTypeResponse: {
				System: `You are an expert API response documenter.
Your task is to write brief, informative response descriptions for REST API endpoints.

Guidelines:
- Describe what the response represents and when it is returned
- For error responses (4xx, 5xx), mention common causes
- For success responses (2xx), mention what data is returned
- Keep descriptions to 1-2 sentences

Respond in {{.Language}} language.
Output MUST be valid JSON: {"description": "..."}

Example input:
  API: GET /users/{id}
  Response Code: 404
Example output:
  {"description": "The requested user was not found. Verify the user ID is correct and the user has not been deleted."}`,
				User: `API: {{.Method}} {{.Path}}
Response Code: {{.ResponseCode}}

Generate a brief description for this response.`,
			},
		},
	}
}

// Get retrieves a template by type
func (m *TemplateManager) Get(ttype TemplateType) (*Template, error) {
	tmpl, ok := m.templates[ttype]
	if !ok {
		return nil, ErrTemplateNotFound
	}
	return tmpl, nil
}

// Set sets a custom template
func (m *TemplateManager) Set(ttype TemplateType, tmpl *Template) error {
	// Validate templates can be parsed
	if _, err := template.New("system").Funcs(template.FuncMap{"join": strings.Join}).Parse(tmpl.System); err != nil {
		return fmt.Errorf("invalid system prompt template for %q: %w", ttype, err)
	}
	if _, err := template.New("user").Funcs(template.FuncMap{"join": strings.Join}).Parse(tmpl.User); err != nil {
		return fmt.Errorf("invalid user prompt template for %q: %w", ttype, err)
	}
	m.templates[ttype] = tmpl
	return nil
}
