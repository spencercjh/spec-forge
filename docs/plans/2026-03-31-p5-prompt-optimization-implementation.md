# P5 Prompt Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve LLM enrichment quality by enriching context passing, rewriting built-in prompts with few-shot examples, and adding custom prompt file support.

**Architecture:** Three-layer improvement: (1) pass more OpenAPI spec metadata (format, enum, constraints, tags) to templates, (2) rewrite templates with type-specific system prompts and examples, (3) allow users to override prompts via `.spec-forge.yaml`. Output format stays the same — no response parsing changes needed.

**Tech Stack:** Go 1.26, text/template, Viper config, kin-openapi

---

## File Structure

| File                                         | Responsibility                                              |
|----------------------------------------------|-------------------------------------------------------------|
| `internal/enricher/prompt/templates.go`      | Context types, built-in templates, FuncMap, TemplateManager |
| `internal/enricher/prompt/templates_test.go` | Template rendering tests                                    |
| `internal/enricher/processor/schema.go`      | Schema field collection with enriched metadata              |
| `internal/enricher/processor/processor.go`   | FieldElement/ParamFieldItem types, conversion helpers       |
| `internal/enricher/enricher.go`              | Parameter collection, API context, custom prompt wiring     |
| `internal/enricher/config.go`                | CustomPrompts field on enricher Config                      |
| `internal/config/config.go`                  | CustomPrompts on EnrichConfig                               |
| `cmd/enrich.go`                              | Wire custom prompts from config to enricher                 |
| `cmd/generate.go`                            | Wire custom prompts in generate pipeline                    |
| `.spec-forge.example.yaml`                   | Document new config option                                  |

---

### Task 1: Enrich FieldContext, ParamFieldContext, and TemplateContext types

**Files:**
- Modify: `internal/enricher/prompt/templates.go`
- Modify: `internal/enricher/prompt/templates_test.go`

- [x] Add enriched fields to `FieldContext` (Format, Enum, Constraints, ExistingDescription)
- [x] Add enriched fields to `ParamFieldContext` (Format, Enum, Constraints, ExistingDescription)
- [x] Add Tags, ExistingSummary, ExistingDescription to `TemplateContext`
- [x] Add `join` func to `renderString` FuncMap
- [x] Add tests for enriched field rendering
- [x] Commit: `feat(enricher): add enriched context fields to FieldContext, ParamFieldContext, and TemplateContext`

---

### Task 2: Populate enriched context in schema field collection

**Files:**
- Modify: `internal/enricher/processor/processor.go`
- Modify: `internal/enricher/processor/schema.go`
- Test: `internal/enricher/processor/schema_test.go`

- [x] Add enriched fields to `FieldElement` and `ParamFieldItem`
- [x] Update `convertFieldElements` and `convertParamFieldItems` to propagate enriched fields
- [x] Add `buildConstraintsString` and `buildEnumStrings` helpers to schema.go
- [x] Update `CollectSchemaFields` to populate Format, Enum, Constraints, ExistingDescription
- [x] Add `TestCollectSchemaFields_EnrichedContext`
- [x] Commit: `feat(enricher): populate enriched context (format, enum, constraints) in schema field collection`

---

### Task 3: Populate enriched context in parameter and API collection

**Files:**
- Modify: `internal/enricher/enricher.go`
- Modify: `internal/enricher/enricher_test.go`

- [x] Add `buildParamConstraintsString` and `buildParamEnumStrings` helpers
- [x] Update `collectParameterGroups` to extract format, enum, constraints from param schemas
- [x] Update `collectElements` to pass Tags, ExistingSummary, ExistingDescription for API operations
- [x] Add `TestEnricher_CollectParameters_EnrichedContext` and `TestEnricher_CollectElements_APITags`
- [x] Commit: `feat(enricher): populate enriched context (tags, format, enum, constraints) in parameter and API collection`

---

### Task 4: Rewrite built-in prompt templates

**Files:**
- Modify: `internal/enricher/prompt/templates.go`
- Modify: `internal/enricher/prompt/templates_test.go`

- [x] Replace all 4 templates with type-specific system prompts, few-shot examples, quality guidelines
- [x] API template: verb-led summaries, specificity guidelines, tags/existing desc support
- [x] Schema template: constraint-aware descriptions, enum explanation guidance
- [x] Param template: location context, enum guidance, format hints
- [x] Response template: error cause guidance, success content hints
- [x] Add `TestNewTemplateManager_RendersAllTypesWithEnrichedContext` and `TestNewTemplateManager_APITemplateUsesTags`
- [x] Commit: `feat(enricher): rewrite built-in prompts with type-specific system prompts, few-shot examples, and enriched context`

---

### Task 5: Add custom prompt config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/enricher/config.go`
- Modify: `.spec-forge.example.yaml`

- [x] Add `CustomPrompts map[string]CustomPromptCfg` to `config.EnrichConfig`
- [x] Add `CustomPromptConfig` type and `CustomPrompts` field to `enricher.Config`
- [x] Update `.spec-forge.example.yaml` with commented customPrompts section

---

### Task 6: Wire custom prompts through enricher pipeline

**Files:**
- Modify: `internal/enricher/enricher.go`
- Modify: `cmd/enrich.go`
- Modify: `cmd/generate.go`

- [x] Apply custom prompts via `TemplateManager.Set()` in `Enrich()` method
- [x] Map custom prompts from config in `cmd/enrich.go`
- [x] Map custom prompts from config in `cmd/generate.go`
- [x] Commit: `feat(enricher): add custom prompt configuration and wire through enricher pipeline`

---

### Task 7: Integration test and verification

**Files:**
- Modify: `internal/enricher/enricher_test.go`

- [x] Add `TestEnricher_CustomPrompts` with `trackingMockProvider`
- [x] Fix lint issues (perfsprint, gocritic rangeValCopy)
- [x] `make fmt`, `make lint` (0 issues), `make test` (all pass)
- [x] Commit: `chore: fix lint issues and add custom prompts integration test`

---

## Verification

```bash
# Build
go build -o ./build/spec-forge .

# Test with real LLM
LLM_API_KEY="your-key" ./build/spec-forge enrich \
    ./integration-tests/maven-springboot-openapi-demo/target/openapi.json \
    --provider custom --model deepseek-chat \
    --custom-base-url https://api.deepseek.com/v1 \
    --language zh -v

# Test custom prompts via config
# Add to .spec-forge.yaml:
#   enrich:
#     customPrompts:
#       api:
#         system: "You are a Chinese API writer..."
#         user: "Endpoint: {{.Method}} {{.Path}}\nWrite summary+description in JSON."
```

Expected: Descriptions leverage format/enum/constraint context for more specific output.
