package prompt

import (
	"bytes"
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

// FieldContext provides context for a schema field
type FieldContext struct {
	Name     string
	Type     string
	Required bool
}

// TemplateContext provides context for template rendering
type TemplateContext struct {
	Type     TemplateType
	Language string

	// API context
	Path   string
	Method string

	// Schema context
	SchemaName string
	Fields     []FieldContext

	// Field context
	FieldName string
	FieldType string
	Required  bool

	// Parameter context
	ParamName string

	// Response context
	ResponseCode string
}

// Template represents a prompt template with system and user prompts
type Template struct {
	System string
	User   string
}

// Render renders the template with the given context
func (t *Template) Render(ctx TemplateContext) (system, user string, err error) {
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
	t, err := template.New("prompt").Parse(tmpl)
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
				System: `You are an API documentation expert. Generate concise, clear descriptions.
Respond in {{.Language}} language.
Output format: JSON with "summary" and "description" fields.`,
				User: `API Endpoint: {{.Path}}
HTTP Method: {{.Method}}

Generate the summary (one line) and description (1-3 sentences) for this API.`,
			},
			TemplateTypeSchema: {
				System: `You are an API documentation expert. Generate concise field descriptions.
Respond in {{.Language}} language.
Output format: JSON mapping field names to descriptions.`,
				User: `Schema: {{.SchemaName}}
Fields:
{{range .Fields}}- {{.Name}} ({{.Type}}, {{if .Required}}required{{else}}optional{{end}})
{{end}}

Generate a description for each field.`,
			},
			TemplateTypeParam: {
				System: `You are an API documentation expert. Generate concise parameter descriptions.
Respond in {{.Language}} language.
Output format: JSON with "description" field.`,
				User: `Parameter: {{.ParamName}}
Type: {{.FieldType}}
Required: {{.Required}}

Generate a brief description for this parameter.`,
			},
			TemplateTypeResponse: {
				System: `You are an API documentation expert. Generate concise response descriptions.
Respond in {{.Language}} language.
Output format: JSON with "description" field.`,
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
func (m *TemplateManager) Set(ttype TemplateType, tmpl *Template) {
	m.templates[ttype] = tmpl
}
