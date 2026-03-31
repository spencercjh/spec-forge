# Publishing

Guide to publishing OpenAPI specs to documentation platforms.

## Supported Publishers

| Publisher                        | Status     | Authentication |
|----------------------------------|------------|----------------|
| Local File                       | ✅ Ready    | None           |
| [ReadMe.com](https://readme.com) | ✅ Ready    | API key        |
| Apifox                           | 🚧 Planned | —              |
| Postman                          | 🚧 Planned | —              |

---

## Local File (Default)

The default output is a local file.

```bash
# Default output
spec-forge generate ./project

# Custom output path (format auto-detected from extension)
spec-forge generate ./project --output ./docs/api.yaml
spec-forge generate ./project --output ./api.json

# Custom output directory (uses default filename)
spec-forge generate ./project --output-dir ./docs
```

---

## ReadMe.com

Publish directly to [ReadMe.com](https://readme.com) using the `rdme` CLI.

### Prerequisites

Install `rdme`:

```bash
npm install -g rdme
```

Get your API key from ReadMe.com dashboard.

### Standalone Publish

```bash
export README_API_KEY="rdme_xxx"

spec-forge publish ./openapi.json \
    --to readme \
    --readme-slug my-api
```

Options:
- `--readme-slug` — API identifier in ReadMe (required)
- `--readme-branch` — Version/branch name (default: `stable`)
- `--overwrite` — Overwrite existing spec (default: false)

### Full Pipeline

Publish as part of generation:

```bash
export LLM_API_KEY="sk-xxx"
export README_API_KEY="rdme_xxx"

spec-forge generate ./project \
    --language zh \
    --publish-target readme \
    --readme-slug my-api \
    -v
```

This runs: Detect → Patch → Generate → Validate → Enrich → Publish

### Safety: Overwrite Protection

By default, Spec Forge refuses to overwrite existing specs in ReadMe:

```
Error: Spec already exists. Use --overwrite to replace.
```

Use `--overwrite` in CI/CD pipelines:

```bash
spec-forge publish ./openapi.json \
    --to readme \
    --readme-slug my-api \
    --overwrite
```

### Configuration

Set defaults in `.spec-forge.yaml`:

```yaml
readme:
  slug: my-service-api
  branch: v1.0
```

Then publish with less typing:

```bash
README_API_KEY="rdme_xxx" spec-forge publish ./openapi.json --to readme
```

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Publish API Docs

on:
  push:
    branches: [main]

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.26'

      - name: Install Spec Forge
        run: go install github.com/spencercjh/spec-forge@latest

      - name: Generate and Publish
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          README_API_KEY: ${{ secrets.README_API_KEY }}
        run: |
          spec-forge generate ./ \
            --language en \
            --publish-target readme \
            --readme-slug ${{ secrets.README_SLUG }} \
            --publish-overwrite
```

### GitLab CI

```yaml
publish-api:
  image: golang:1.26
  stage: deploy
  script:
    - go install github.com/spencercjh/spec-forge@latest
    - spec-forge generate ./ \
        --publish-target readme \
        --readme-slug $README_SLUG \
        --publish-overwrite
  only:
    - main
```

---

## Security Best Practices

### Never Commit API Keys

❌ Bad:

```yaml
# .spec-forge.yaml
enrich:
  apiKey: "sk-xxx"  # Never do this!
```

✅ Good:

```yaml
# .spec-forge.yaml - No API keys here!
enrich:
  enabled: true
  provider: openai
  model: gpt-4o
```

```bash
export OPENAI_API_KEY="sk-xxx"
spec-forge generate ./
```

### Use Environment Variables

Keep API keys in environment variables, never in config files:

```yaml
# .spec-forge.yaml - Safe: no API keys
enrich:
  enabled: true
  provider: openai
  model: gpt-4o
```

```bash
# Set environment variable
export OPENAI_API_KEY="sk-xxx"

# Run spec-forge
spec-forge generate ./
```

**Note:** For OpenAI and Anthropic, API keys are always read from `OPENAI_API_KEY` and `ANTHROPIC_API_KEY` respectively. For custom providers, use `LLM_API_KEY` or set `apiKeyEnv` in config.

### CI/CD Secrets

Store in your CI system's secret management:

- GitHub Actions: `Settings → Secrets and variables → Actions`
- GitLab CI: `Settings → CI/CD → Variables`
- CircleCI: `Project Settings → Environment Variables`

---

## Troubleshooting

### "rdme not found"

Install the ReadMe CLI:

```bash
npm install -g rdme
```

### "Authentication failed"

Check your API key:

```bash
echo $README_API_KEY
```

Verify in ReadMe.com dashboard under API Keys.

### "Spec already exists"

Either:
1. Use `--publish-overwrite` to replace
2. Change `--readme-slug` to a new identifier
3. Change `--readme-branch` to a different version

### "Slug not found"

The slug must match an existing API in your ReadMe project. Create it first in the ReadMe dashboard.

---

## Future Publishers

### Apifox (Planned)

```bash
# Coming soon
spec-forge publish ./openapi.json --to apifox
```

### Postman (Planned)

```bash
# Coming soon
spec-forge publish ./openapi.json --to postman
```

### Custom Webhook (Planned)

```bash
# Coming soon
spec-forge publish ./openapi.json \
    --to webhook \
    --webhook-url https://api.example.com/specs
```

---

## Comparison

| Feature       | Local File             | ReadMe.com        |
|---------------|------------------------|-------------------|
| Versioning    | Manual (filenames)     | Built-in branches |
| Hosting       | Self-managed           | Hosted            |
| Collaboration | Git-based              | Built-in comments |
| Try-it-now    | Requires separate tool | Built-in          |
| Cost          | Free                   | Paid tiers        |
| Custom domain | Your setup             | Supported         |

Choose based on your team's needs. Many teams use both:
- **Local files** for development and PR reviews
- **ReadMe.com** for public API documentation
