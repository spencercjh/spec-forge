# AI Enrichment

Guide to using LLM providers and customizing prompts in Spec Forge.

## Overview

AI Enrichment adds human-readable descriptions to your OpenAPI spec:

- **API operations** — Summary and description for each endpoint
- **Schema fields** — Descriptions for data model properties
- **Parameters** — Descriptions for query/path/header parameters
- **Responses** — Descriptions for response schemas

## How It Works

1. Spec Forge extracts items needing descriptions from your OpenAPI spec
2. Groups items into batches for efficient processing
3. Sends structured prompts to LLM
4. Parses responses and updates the spec

**Important:** Spec Forge reads your **code structure** for accuracy. AI only generates the human-readable text, ensuring specs never drift from reality.

---

## Supported Providers

### OpenAI

```yaml
enrich:
  provider: openai
  model: gpt-4o
  apiKeyEnv: OPENAI_API_KEY
```

Recommended models:
- `gpt-4o` — Best quality
- `gpt-4o-mini` — Faster, cheaper

### Anthropic

```yaml
enrich:
  provider: anthropic
  model: claude-3-sonnet-20240229
  apiKeyEnv: ANTHROPIC_API_KEY
```

### Ollama (Local)

```yaml
enrich:
  provider: ollama
  model: llama3
  baseUrl: http://localhost:11434
```

Requirements:
- Ollama running locally
- Model pulled: `ollama pull llama3`

### Custom (OpenAI-Compatible)

```yaml
enrich:
  provider: custom
  model: deepseek-chat
  baseUrl: https://api.deepseek.com/v1
  apiKeyEnv: LLM_API_KEY
```

Works with any OpenAI-compatible API: DeepSeek, Mistral, Groq, etc.

---

## Usage

### During Generation

AI enrichment runs automatically during `spec-forge generate` when `enrich.provider` and `enrich.model` are configured in `.spec-forge.yaml`:

```bash
export OPENAI_API_KEY="sk-xxx"
spec-forge generate ./project --language zh
```

To disable enrichment during generation:

```bash
spec-forge generate ./project --language zh --skip-enrich
```

### Standalone Enrichment

```bash
# Enrich existing spec
spec-forge enrich ./openapi.json --provider openai --model gpt-4o

# With custom provider
LLM_API_KEY="sk-xxx" spec-forge enrich ./openapi.json \
    --provider custom \
    --custom-base-url https://api.deepseek.com/v1 \
    --model deepseek-chat \
    --language zh
```

---

## Streaming vs Concurrent

### Streaming (Default)

Shows real-time progress with batch type prefixes:

```
[api] Processing batch 1/3...
[schema] Processing batch 2/3...
[param] Processing batch 3/3...
```

Best for: Interactive use, seeing progress

### Concurrent (`--no-stream`)

Processes batches in parallel for faster completion:

```bash
spec-forge enrich ./openapi.json --no-stream
```

Best for: CI/CD, large specs, speed-critical scenarios

---

## Languages

Use `--language` or `enrich.language` config:

| Code | Language             |
|------|----------------------|
| `zh` | Chinese (Simplified) |
| `en` | English              |
| `ja` | Japanese             |
| `ko` | Korean               |
| `de` | German               |
| `fr` | French               |
| `es` | Spanish              |
| `ru` | Russian              |
| `pt` | Portuguese           |
| `it` | Italian              |
| `ar` | Arabic               |

Example:

```bash
spec-forge generate ./project --language ja
```

---

## Custom Prompts

Override built-in prompts via `.spec-forge.yaml`:

```yaml
enrich:
  customPrompts:
    api:
      system: "You are an expert API documentation writer."
      user: |
        API: {{.Method}} {{.Path}}
        {{if .Tags}}Tags: {{join .Tags ", "}}{{end}}

        Generate a concise summary (max 80 chars) and description.
        Output JSON: {"summary": "...", "description": "..."}

    schema:
      system: "You are a data modeling expert."
      user: |
        Field: {{.Name}} ({{.Type}})
        {{if .Required}}Required{{else}}Optional{{end}}
        {{if .Format}}Format: {{.Format}}{{end}}

        Generate a clear, concise description.
```

### Template Variables

**API template:**
- `{{.Method}}` — HTTP method (GET, POST, etc.)
- `{{.Path}}` — API path
- `{{.Tags}}` — List of operation tags
- `{{.ExistingSummary}}` — Current summary (if force mode)
- `{{.ExistingDescription}}` — Current description (if force mode)

**Schema/Param/Response templates:**
- `{{.Name}}` — Field/parameter name
- `{{.Type}}` — Data type (string, integer, etc.)
- `{{.Required}}` — true/false
- `{{.Format}}` — Format modifier (email, uuid, date-time)
- `{{.Enum}}` — List of allowed values
- `{{.Constraints}}` — Human-readable constraints (min, max, pattern)
- `{{.ExistingDescription}}` — Current description (if force mode)

### Helper Functions

- `{{join list separator}}` — Join list with separator

---

## Force Mode

Enrich fields even if they already have descriptions:

```bash
spec-forge enrich ./openapi.json --force
```

Useful for:
- Translating existing descriptions
- Improving low-quality descriptions
- Re-processing with different provider/model

---

## Token Optimization

Spec Forge automatically optimizes prompts to reduce token usage:

1. **Batches related items** — Groups similar APIs/schemas together
2. **Deduplicates schemas** — References instead of inline definitions
3. **Prioritizes key fields** — Focuses on important schema properties

Enable verbose mode to see batching:

```bash
spec-forge enrich ./openapi.json -v
```

---

## Troubleshooting

### "API key not found"

Check environment variable:

```bash
echo $OPENAI_API_KEY
```

Or set in config:

```yaml
enrich:
  apiKeyEnv: MY_API_KEY
```

### Slow processing

Use `--no-stream` for concurrent processing:

```bash
spec-forge enrich ./openapi.json --no-stream
```

Or increase concurrency:

```yaml
enrich:
  concurrency: 10
```

### Poor quality descriptions

1. Try a better model (gpt-4o instead of gpt-3.5-turbo)
2. Customize prompts for your domain
3. Use force mode to re-process

### Rate limiting

- Reduce concurrency: `enrich.concurrency: 3`
- Add delays between batches (not directly supported, use smaller specs)
- Consider upgrading your API plan

---

## Cost Estimation

Approximate costs for AI enrichment (varies by model, spec complexity, and description length):

Pricing basis (per 1M tokens):
- OpenAI GPT-4o: $2.50 input / $10.00 output
- DeepSeek: $0.14 input / $0.28 output

| Spec Size        | ~Tokens | OpenAI GPT-4o | DeepSeek |
|------------------|---------|---------------|----------|
| Small (10 APIs)  | 5K      | ~$0.03        | ~$0.001  |
| Medium (50 APIs) | 25K     | ~$0.15        | ~$0.005  |
| Large (200 APIs) | 100K    | ~$0.60        | ~$0.02   |

**Notes:**
- Costs are typically negligible for development workflows
- Actual costs depend on prompt complexity and generated description length
- Use `--no-stream` with `concurrency` setting to process faster (same cost)
- Consider using local Ollama for zero-cost enrichment during development

---

## Examples

### DeepSeek (Chinese)

```yaml
# .spec-forge.yaml
enrich:
  provider: custom
  model: deepseek-chat
  baseUrl: https://api.deepseek.com/v1
  apiKeyEnv: DEEPSEEK_API_KEY
  language: zh
```

```bash
export DEEPSEEK_API_KEY="sk-xxx"
spec-forge generate ./project
```

### Ollama (Local, Free)

```bash
# Start Ollama
ollama serve

# Pull model
ollama pull llama3

# Generate
spec-forge generate ./project \
    --provider ollama \
    --model llama3 \
    --custom-base-url http://localhost:11434
```

### Mixing Providers

Generate without enrichment, then enrich separately:

```bash
# Generate spec
spec-forge generate ./project

# Enrich with specific provider
spec-forge enrich ./openapi.yaml \
    --provider anthropic \
    --model claude-3-opus-20240229
```
