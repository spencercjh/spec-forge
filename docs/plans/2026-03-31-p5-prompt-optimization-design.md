# P5 Prompt Optimization Design

> **Status:** Implemented
> **Date:** 2026-03-31
> **Parent Issue:** #40 (Phase 4 - LangchainGo Features)

## Overview

Improve LLM enrichment output quality by enriching context passing, rewriting built-in prompt templates, and adding custom prompt file support.

## Current State

| Feature           | Status        | Detail                                                   |
|-------------------|---------------|----------------------------------------------------------|
| Context passing   | Minimal       | Only name, type, required passed to templates            |
| System prompts    | Generic       | All 4 types use "You are an API documentation expert"    |
| Few-shot examples | None          | Templates contain no input/output examples               |
| Constraints/enums | Ignored       | OpenAPI format, enum, min/max, pattern not passed to LLM |
| API tags          | Ignored       | Operation tags not included in context                   |
| Custom prompts    | Not supported | No way to override built-in templates                    |

## Design

### Optimization 1: Enriched Context Passing

**Problem:** Templates receive only `Name`, `Type`, `Required` for fields/params. The OpenAPI spec contains much richer metadata that would help LLMs generate more precise descriptions.

**Solution:** Pass additional spec metadata to templates:

**FieldContext additions:**
- `Format` — e.g., `"email"`, `"date-time"`, `"uuid"`
- `Enum` — allowed values, e.g., `["active", "inactive"]`
- `Constraints` — human-readable string: `"min: 0, max: 100, pattern: ^[a-z]+$"`
- `ExistingDescription` — existing description from spec (useful in `--force` mode)

**ParamFieldContext additions:** Same as FieldContext.

**TemplateContext additions (API-specific):**
- `Tags` — operation tags from the spec
- `ExistingSummary` / `ExistingDescription` — existing partial documentation

**Example impact on Schema prompt:**

```
Before:
- email (string, required)

After:
- email (string, required, format: email, maxLength: 255)
- role (string, optional, enum: [admin, user, guest])
```

### Optimization 2: Improved Built-in Prompts

**Problem:** Generic system prompts produce generic descriptions. No examples, no quality guidelines, no output constraints.

**Solution:** Type-specific system prompts with:

1. **Role definition** — Different expert roles per type (API writer, data modeler, parameter documenter)
2. **Quality guidelines** — Specific rules per type (e.g., "Summary starts with a verb", "Avoid repeating field name")
3. **Few-shot examples** — Input/output pairs showing expected quality
4. **Explicit output format** — JSON schema with constraints

**API Template (before):**
```
System: You are an API documentation expert. Generate concise, clear descriptions.
Respond in {{.Language}} language.
Output format: JSON with "summary" and "description" fields.

User: API Endpoint: {{.Path}}
HTTP Method: {{.Method}}

Generate the summary (one line) and description (1-3 sentences) for this API.
```

**API Template (after):**
```
System: You are an expert OpenAPI documentation writer specializing in REST API descriptions.
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
  {"summary": "Create a new user", "description": "Registers a new user account..."}

User: API Endpoint: {{.Method}} {{.Path}}
{{- if .Tags}}
Tags: {{join .Tags ", "}}
{{- end}}
...
```

### Optimization 3: Custom Prompt File Support

**Problem:** Users cannot customize prompts for their domain without modifying source code.

**Solution:** Add `customPrompts` section to `.spec-forge.yaml`:

```yaml
enrich:
  customPrompts:
    api:
      system: "You are a Chinese API documentation writer..."
      user: "API: {{.Method}} {{.Path}}\n用中文描述这个接口。"
    schema:
      system: "You are a data model expert..."
```

**Implementation:** Config loads custom prompts via Viper, passes through `enricher.Config.CustomPrompts`, and applies via `TemplateManager.Set()`.

## Architecture

```
                    ┌─────────────────────┐
                    │   .spec-forge.yaml   │
                    │   customPrompts:     │
                    │     api/system/user  │
                    └──────────┬──────────┘
                               │
                               ▼
┌──────────────┐    ┌─────────────────────┐    ┌──────────────┐
│  OpenAPI     │───▶│  Collection Layer   │───▶│  Template    │
│  Spec        │    │  (enricher.go)      │    │  Manager     │
│              │    │  - format           │    │              │
│  - format    │    │  - enum             │    │  Built-in    │
│  - enum      │    │  - constraints      │    │  + Custom    │
│  - min/max   │    │  - tags             │    │  overrides   │
│  - pattern   │    │  - existing desc    │    │              │
│  - tags      │    └─────────────────────┘    └──────┬───────┘
└──────────────┘                                      │
                                                      ▼
                                            ┌──────────────────┐
                                            │  LLM Provider    │
                                            │  (OpenAI/etc.)   │
                                            └──────────────────┘
```

## Key Decisions

1. **Backward compatible** — Output format stays the same (`{"summary": "...", "description": "..."}` for API, `{"field": "desc"}` for schema/param). No response parsing changes.

2. **Constraint helper reuse** — `enricher.go` reuses `processor.BuildConstraintsString` and `processor.BuildEnumStrings` to keep constraint/enum formatting logic centralized in the `processor` package, avoiding duplicated helpers.

3. **ExistingDescription in templates** — Only visible in `--force` mode (fields with existing descriptions are skipped otherwise). When force is on, the LLM can improve or translate existing descriptions.

4. **Template FuncMap** — Added `join` function (maps to `strings.Join`) for rendering enum/tag lists. Registered in `renderString` via `template.FuncMap`.

5. **Config key mapping** — Custom prompt keys (`"api"`, `"schema"`, `"param"`, `"response"`) directly match `TemplateType` string constants.
