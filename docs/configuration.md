# Configuration

Complete reference for `.spec-forge.yaml` configuration.

## Configuration File Location

Spec Forge looks for `.spec-forge.yaml` in:

1. Current working directory
2. Project directory (when using `spec-forge generate ./project`)

## Configuration Priority

Values are resolved in this order (highest to lowest):

1. **CLI flags** — `--provider`, `--model`, etc.
2. **Environment variables** — `OPENAI_API_KEY`, `LLM_API_KEY`
3. **Config file** — `.spec-forge.yaml`
4. **Defaults**

---

## Full Example

```yaml
# Enrichment settings
enrich:
  enabled: true
  provider: openai              # openai, anthropic, ollama, custom
  model: gpt-4o
  baseUrl: ""                   # For custom providers
  apiKeyEnv: OPENAI_API_KEY     # Environment variable name
  apiKey: ""                    # Direct API key (not recommended)
  language: zh                  # Output language
  timeout: 60s                  # Request timeout
  force: false                  # Enrich even if description exists
  concurrency: 5                # Concurrent requests (non-streaming)
  customPrompts:                # Override built-in prompts
    api:
      system: "You are an API documentation expert..."
      user: "API: {{.Method}} {{.Path}}"
    schema:
      system: "You are a data model expert..."
    param:
      system: "You are a parameter documentation expert..."
    response:
      system: "You are a response schema expert..."

# Output settings
output:
  dir: ./openapi                # Output directory
  format: yaml                  # yaml or json
  filename: openapi             # Base filename (without extension)

# Publishing settings
readme:
  slug: my-api                  # API identifier in ReadMe
  branch: stable                # Version/branch name

# General settings
verbose: false                  # Verbose logging
skipValidation: false           # Skip OpenAPI validation
```

---

## Enrichment Settings

### `enrich.enabled`

Enable AI enrichment during generation.

```yaml
enrich:
  enabled: true
```

CLI equivalent: `--enrich`

### `enrich.provider`

LLM provider to use.

| Value       | Description                  |
|-------------|------------------------------|
| `openai`    | OpenAI API                   |
| `anthropic` | Anthropic Claude API         |
| `ollama`    | Local Ollama instance        |
| `custom`    | Custom OpenAI-compatible API |

```yaml
enrich:
  provider: openai
```

CLI equivalent: `--provider`

### `enrich.model`

Model name for the provider.

```yaml
# OpenAI
enrich:
  model: gpt-4o

# Anthropic
enrich:
  model: claude-3-sonnet-20240229

# Ollama
enrich:
  model: llama3

# Custom (DeepSeek example)
enrich:
  model: deepseek-chat
```

CLI equivalent: `--model`

### `enrich.baseUrl`

Base URL for custom providers.

```yaml
enrich:
  provider: custom
  baseUrl: https://api.deepseek.com/v1
```

CLI equivalent: `--custom-base-url`

### `enrich.apiKeyEnv`

Environment variable containing the API key.

```yaml
enrich:
  apiKeyEnv: LLM_API_KEY
```

Default: `LLM_API_KEY` for custom, `OPENAI_API_KEY` for OpenAI, etc.

### `enrich.language`

Language for generated descriptions.

```yaml
enrich:
  language: zh
```

Supported: `zh`, `en`, `ja`, `ko`, `de`, `fr`, `es`, `ru`, `pt`, `it`, `ar`

CLI equivalent: `--language`

### `enrich.timeout`

Timeout for LLM requests.

```yaml
enrich:
  timeout: 120s
```

Default: `60s`

### `enrich.force`

Enrich fields even if they already have descriptions.

```yaml
enrich:
  force: true
```

CLI equivalent: `--force`

### `enrich.concurrency`

Number of concurrent LLM requests (only when streaming is disabled).

```yaml
enrich:
  concurrency: 10
```

Default: `5`

---

## Output Settings

### `output.dir`

Output directory for generated specs.

```yaml
output:
  dir: ./api-specs
```

Default: Current directory

CLI equivalent: `--output` (accepts file or directory)

### `output.format`

Output format.

```yaml
output:
  format: json
```

Values: `yaml`, `json`

Default: `yaml`

### `output.filename`

Base filename (without extension).

```yaml
output:
  filename: my-api
```

Default: `openapi`

---

## Publishing Settings

### `readme.slug`

API identifier in ReadMe.com.

```yaml
readme:
  slug: my-awesome-api
```

CLI equivalent: `--readme-slug`

### `readme.branch`

Version or branch name in ReadMe.com.

```yaml
readme:
  branch: v1.0
```

Default: `stable`

CLI equivalent: `--readme-branch`

---

## Custom Prompts

Override built-in prompts for each enrichment type.

### Structure

```yaml
enrich:
  customPrompts:
    api:
      system: "System prompt for API descriptions"
      user: "User prompt template for API"
    schema:
      system: "System prompt for schema fields"
      user: "User prompt template for fields"
    param:
      system: "System prompt for parameters"
      user: "User prompt template for params"
    response:
      system: "System prompt for responses"
      user: "User prompt template for responses"
```

Each type supports:
- `system` — System prompt that sets the AI's role and behavior
- `user` — User prompt template with variables (see below)

### Available Types

- `api` — API operation descriptions (summary + description)
- `schema` — Schema field descriptions
- `param` — Parameter descriptions
- `response` — Response schema descriptions

### Template Variables

**API template:**
- `{{.Method}}` — HTTP method
- `{{.Path}}` — API path
- `{{.Tags}}` — Operation tags
- `{{.ExistingSummary}}` — Current summary (if any)
- `{{.ExistingDescription}}` — Current description (if any)

**Schema/Param/Response templates:**
- `{{.Name}}` — Field/parameter name
- `{{.Type}}` — Data type
- `{{.Required}}` — Whether required
- `{{.Format}}` — Format (email, uuid, etc.)
- `{{.Enum}}` — Allowed values
- `{{.Constraints}}` — Min/max/pattern constraints

### Example

```yaml
enrich:
  customPrompts:
    api:
      system: |
        You are a Chinese API documentation expert.
        Write concise, accurate descriptions in Chinese.
      user: |
        API: {{.Method}} {{.Path}}
        {{if .Tags}}Tags: {{join .Tags ", "}}{{end}}

        Generate a summary (max 20 characters) and description (2-3 sentences) for this API.

        Output JSON: {"summary": "...", "description": "..."}

    schema:
      system: "You are a data modeling expert."
      user: |
        Field: {{.Name}} ({{.Type}})
        {{if .Required}}Required{{else}}Optional{{end}}
        {{if .Format}}Format: {{.Format}}{{end}}
        {{if .Enum}}Allowed values: {{join .Enum ", "}}{{end}}

        Generate a concise description.
```

---

## Provider-Specific Examples

### OpenAI

```yaml
enrich:
  enabled: true
  provider: openai
  model: gpt-4o
  apiKeyEnv: OPENAI_API_KEY
  language: en
```

### Anthropic

```yaml
enrich:
  enabled: true
  provider: anthropic
  model: claude-3-sonnet-20240229
  apiKeyEnv: ANTHROPIC_API_KEY
  language: en
```

### Ollama (Local)

```yaml
enrich:
  enabled: true
  provider: ollama
  model: llama3
  baseUrl: http://localhost:11434
```

### DeepSeek

```yaml
enrich:
  enabled: true
  provider: custom
  model: deepseek-chat
  baseUrl: https://api.deepseek.com/v1
  apiKeyEnv: LLM_API_KEY
  language: zh
```

---

## Complete Real-World Example

```yaml
# .spec-forge.yaml for a Chinese Spring Boot project using DeepSeek

enrich:
  enabled: true
  provider: custom
  model: deepseek-chat
  baseUrl: https://api.deepseek.com/v1
  apiKeyEnv: DEEPSEEK_API_KEY
  language: zh
  timeout: 90s
  customPrompts:
    api:
      system: "你是一个专业的中文API文档编写专家。"

output:
  dir: ./docs/api
  format: yaml

readme:
  slug: my-service-api
  branch: stable
```

Usage:

```bash
export DEEPSEEK_API_KEY="sk-xxx"
spec-forge generate ./my-spring-boot-project
```
