# CLI UX Improvements Design

**Date:** 2026-03-30
**Status:** Approved
**Scope:** Completion subcommand, error message coloring, enricher progress bar

## Overview

Improve spec-forge CLI user experience through three targeted enhancements:
1. Shell completion subcommand (Bash/Zsh/Fish/PowerShell)
2. Colored error and status output
3. Enricher batch processing progress bar

**Technical choice:** Lightweight dependency approach — `fatih/color` for coloring + `schollz/progressbar` for progress. No TUI framework.

---

## 1. Completion Subcommand

### Interface

```
spec-forge completion bash       # Output bash completion script
spec-forge completion zsh        # Output zsh completion script
spec-forge completion fish       # Output fish completion script
spec-forge completion powershell # Output PowerShell completion script
```

### Implementation

- New file: `cmd/completion.go`
- Use Cobra's native `rootCmd.GenBashCompletion()`, `GenZshCompletion()`, `GenFishCompletion()`, `GenPowerShellCompletion()` APIs
- Standard Cobra completion command pattern with `DisableFlagsInUseLine: true` and `Hidden: false`

### Dynamic Completions (ValidArgsFunction)

Register `ValidArgsFunction` on enum-type flags for shell completion:

| Flag | Command | Valid Values |
|------|---------|-------------|
| `--provider` | enrich | openai, anthropic, ollama, custom |
| `--publish-target` | generate, publish | readme |
| `-o, --output` | generate, enrich, publish | yaml, json |
| `--language` | generate, enrich | en, zh |
| `-t, --to` | publish | readme |

### Files Changed

- **New:** `cmd/completion.go`
- **Modified:** `cmd/enrich.go`, `cmd/publish.go`, `cmd/generate.go` (add `ValidArgsFunction` to flag definitions)

---

## 2. Colored Error and Status Output

### Color Scheme

| Element | Style | Example |
|---------|-------|---------|
| Error category prefix | Red + Bold | **[DETECT]** |
| Error message | Default | no supported framework detected |
| Hint marker | Cyan | 💡 Hint: |
| Hint content | Yellow | Verify the project structure... |
| Success status | Green | ✅ OpenAPI spec validated |
| Skip status | Dim (gray) | ⏭️ Validation skipped |
| Progress info | Default | Enriching OpenAPI spec... |

### Implementation

- New package: `internal/cli/output.go`
- Core functions:
  - `Errorf(format, args...)` — prints red error prefix + message
  - `Successf(format, args...)` — prints green success message
  - `Hintf(format, args...)` — prints cyan hint prefix + yellow content
  - `Skipf(format, args...)` — prints dim skip message
- Auto-detection:
  - Respect `NO_COLOR` environment variable (https://no-color.org/) — mere presence disables color
  - Otherwise rely on `fatih/color` defaults for terminal detection and color enable/disable behavior
  - Initialize `color.NoColor` in `internal/cli/initColorState()` instead of performing explicit `IsTerminal` checks

### Output Architecture

```
User-facing status messages → internal/cli (colored)
Debug/diagnostic logs      → slog (plain text, -v only)
Spec file content           → stdout (no color ever)
```

### Files Changed

- **New:** `internal/cli/output.go`
- **Modified:** `cmd/generate.go`, `cmd/enrich.go`, `cmd/publish.go`, `cmd/root.go` (replace slog status calls with colored output)

---

## 3. Enricher Progress Bar

### Display

```
Enriching OpenAPI spec...  ████████████████░░░░ 80% | 12/15 batches | 0 failed
```

### Implementation

- Use `schollz/progressbar/v3`
- Create in `internal/enricher/processor/concurrent.go`:
  - `progressbar.NewOptions(total, ...option)` with total = `len(batches)`
  - Options: `progressbar.OptionSetWriter(os.Stderr)`, `progressbar.OptionShowCount()`, custom description with failed count
  - `bar.Add(1)` after each batch completes
  - `bar.Finish()` at end

### Streaming Compatibility

| Mode | Behavior |
|------|----------|
| Streaming (default) | Progress bar on stderr; streaming text prints above it via `bar.Describe()` or interleaved |
| Concurrent (`--no-stream`) | Thread-safe `bar.Add(1)` from multiple goroutines |
| Non-TTY / piped | Progress bar writes to stderr (schollz/progressbar handles non-TTY rendering); colors disabled via fatih/color TTY detection or `NO_COLOR` |

### Progress Bar Options

```go
progressbar.NewOptions(len(batches),
    progressbar.OptionSetWriter(os.Stderr),
    progressbar.OptionShowCount(),
    progressbar.OptionSetDescription("Enriching OpenAPI spec..."),
    progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
)
```

### Files Changed

- **Modified:** `internal/enricher/processor/concurrent.go`

---

## Dependencies

```
github.com/fatih/color v1.19.0    # Terminal color output
github.com/schollz/progressbar/v3 v3.19.0  # Progress bar
```

Both are well-maintained, widely-used, zero-TUI-framework dependencies.

## Out of Scope

- Full TUI interface (Bubbletea/Lipgloss)
- Multi-line formatted error panels
- Generate pipeline progress (only enricher phase)
- Interactive prompts or spinners
- Configuration file for color theme
