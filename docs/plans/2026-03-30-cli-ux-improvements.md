# CLI UX Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add shell completion, colored output, and progress bar to spec-forge CLI.

**Architecture:** Three independent features added incrementally. `fatih/color` for terminal coloring, `schollz/progressbar` for batch progress, Cobra native completion for shell support. All output goes to stderr; stdout reserved for spec content.

**Tech Stack:** Go 1.26, Cobra v1.10.2, fatih/color, schollz/progressbar/v3

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `cmd/completion.go` | Shell completion subcommand |
| Create | `internal/cli/output.go` | Colored output helpers |
| Create | `internal/cli/output_test.go` | Tests for colored output |
| Modify | `cmd/root.go` | Register completion cmd in `NewRootCommand()`, colored error output |
| Modify | `cmd/enrich.go` | ValidArgsFunction for flags, colored status output |
| Modify | `cmd/generate.go` | ValidArgsFunction for flags, colored status output |
| Modify | `cmd/publish.go` | ValidArgsFunction for flags, colored status output |
| Modify | `cmd/spring.go` | Colored status output |
| Modify | `internal/enricher/processor/concurrent.go` | Progress bar in batch processing loops |
| Modify | `internal/enricher/enricher.go` | Pass progress bar through to processor |
| Modify | `go.mod` / `go.sum` | Add fatih/color, schollz/progressbar/v3 |

---

### Task 1: Add Dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Install fatih/color**

Run:
```bash
cd /home/caijh/codes/open-source/spec-forge/.claude/worktrees/cli-ux
go get github.com/fatih/color@latest
```

- [ ] **Step 2: Install schollz/progressbar**

Run:
```bash
go get github.com/schollz/progressbar/v3@latest
```

- [ ] **Step 3: Tidy dependencies**

Run:
```bash
go mod tidy
```

- [ ] **Step 4: Verify build**

Run:
```bash
make build
```
Expected: Build succeeds with no errors.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -S -s -m "build: add fatih/color and schollz/progressbar dependencies for CLI UX"
```

---

### Task 2: Create Colored Output Package

**Files:**
- Create: `internal/cli/output.go`
- Create: `internal/cli/output_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/cli/output_test.go`:

```go
package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorEnabled(t *testing.T) {
	// When NO_COLOR is not set, color should be enabled (assuming terminal)
	t.Run("color enabled by default", func(t *testing.T) {
		// Save and restore NO_COLOR
		orig := os.Getenv("NO_COLOR")
		os.Unsetenv("NO_COLOR")
		defer os.Setenv("NO_COLOR", orig)

		// Re-init color state
		initColorState()
		assert.True(t, ColorEnabled())
	})

	t.Run("color disabled with NO_COLOR", func(t *testing.T) {
		os.Setenv("NO_COLOR", "1")
		defer os.Unsetenv("NO_COLOR")

		initColorState()
		assert.False(t, ColorEnabled())
	})
}

func TestStatusFunctions(t *testing.T) {
	// Force color on for testing
	origNoColor := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", origNoColor)
	initColorState()

	t.Run("Successf contains checkmark", func(t *testing.T) {
		var buf bytes.Buffer
		Successf(&buf, "test message")
		assert.Contains(t, buf.String(), "test message")
		assert.Contains(t, buf.String(), "✅")
	})

	t.Run("Skipf contains skip mark", func(t *testing.T) {
		var buf bytes.Buffer
		Skipf(&buf, "skipped")
		assert.Contains(t, buf.String(), "skipped")
		assert.Contains(t, buf.String(), "⏭️")
	})

	t.Run("Errorf contains error marker", func(t *testing.T) {
		var buf bytes.Buffer
		Errorf(&buf, "something failed")
		assert.Contains(t, buf.String(), "something failed")
	})

	t.Run("Hintf contains hint", func(t *testing.T) {
		var buf bytes.Buffer
		Hintf(&buf, "check your config")
		assert.Contains(t, buf.String(), "check your config")
	})

	t.Run("Statusf formats with args", func(t *testing.T) {
		var buf bytes.Buffer
		Statusf(&buf, "found %d items", 5)
		assert.Contains(t, buf.String(), "found 5 items")
	})
}

func TestNoColorMode(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	initColorState()

	t.Run("Successf no ANSI codes", func(t *testing.T) {
		var buf bytes.Buffer
		Successf(&buf, "test")
		assert.NotContains(t, buf.String(), "\x1b[")
		assert.Contains(t, buf.String(), "✅")
	})

	t.Run("Errorf no ANSI codes", func(t *testing.T) {
		var buf bytes.Buffer
		Errorf(&buf, "test")
		assert.NotContains(t, buf.String(), "\x1b[")
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./internal/cli/... -run TestColor`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write implementation**

Create `internal/cli/output.go`:

```go
// Package cli provides terminal output helpers for spec-forge CLI.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

var (
	colorEnabled bool

	// Pre-configured color printers (no-op when color is disabled)
	green  = color.New(color.FgGreen).FprintfFunc()
	red    = color.New(color.FgRed, color.Bold).FprintfFunc()
	cyan   = color.New(color.FgCyan).FprintfFunc()
	yellow = color.New(color.FgYellow).FprintfFunc()
	dim    = color.New(color.Faint).FprintfFunc()
)

func init() {
	initColorState()
}

// initColorState reads environment and configures color output.
func initColorState() {
	colorEnabled = os.Getenv("NO_COLOR") == ""
	color.NoColor = !colorEnabled
}

// ColorEnabled reports whether colored output is active.
func ColorEnabled() bool {
	return colorEnabled
}

// Successf prints a green success message with checkmark prefix.
func Successf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	green(w, "✅ %s\n", msg)
}

// Skipf prints a dim skip message with skip prefix.
func Skipf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	dim(w, "⏭️  %s\n", msg)
}

// Errorf prints a red error message.
func Errorf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	red(w, "❌ %s\n", msg)
}

// Hintf prints a cyan hint prefix with yellow hint content.
func Hintf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	cyan(w, "💡 Hint: ")
	yellow(w, "%s\n", msg)
}

// Statusf prints a neutral status message.
func Statusf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format+"\n", args...)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v ./internal/cli/...`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/output.go internal/cli/output_test.go
git commit -S -s -m "feat(cli): add colored output package with success/error/hint helpers"
```

---

### Task 3: Add Completion Subcommand

**Files:**
- Create: `cmd/completion.go`
- Modify: `cmd/root.go:64-91` (add completion to `NewRootCommand()`)

- [ ] **Step 1: Create completion command**

Create `cmd/completion.go`:

```go
package cmd

import (
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for spec-forge.

To load completions:

Bash:
  source <(spec-forge completion bash)

  # To load completions for each session, execute once:
  # Linux:
  spec-forge completion bash > /etc/bash_completion.d/spec-forge
  # macOS:
  spec-forge completion bash > $(brew --prefix)/etc/bash_completion.d/spec-forge

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Add the following to your ~/.zshrc:
  autoload -Uz compinit
  compinit

  # Then load completions:
  spec-forge completion zsh > "${fpath[1]}/_spec-forge"

  # You will need to start a new shell for this setup to take effect.

Fish:
  spec-forge completion fish | source

  # To load completions for each session, execute once:
  spec-forge completion fish > ~/.config/fish/completions/spec-forge.fish

PowerShell:
  spec-forge completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  spec-forge completion powershell > spec-forge.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return nil
		}
	},
}
```

Note: The `os` import will be resolved with `"os"` added to the import block.

- [ ] **Step 2: Register completion in init() and NewRootCommand()**

Add to `cmd/root.go`:

In `init()` function, add after `rootCmd.AddCommand(enrichCmd)` (line ~88 equivalent — there is no enrichCmd added in init, but spring/publish/etc are):
```go
rootCmd.AddCommand(completionCmd)
```

In `NewRootCommand()` function (line ~86-90), add:
```go
c.AddCommand(newCompletionCmd())
```

Create `newCompletionCmd()` factory in `cmd/completion.go`:

```go
// newCompletionCmd creates a new completion command instance for testing.
func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion script",
		Long:                  completionCmd.Long,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE:                  completionCmd.RunE,
	}
}
```

- [ ] **Step 3: Verify completion works**

Run:
```bash
go build -o ./build/spec-forge . && ./build/spec-forge completion bash | head -5
```
Expected: Outputs bash completion script (starts with `# bash completion` or similar).

Run:
```bash
./build/spec-forge completion zsh | head -5
```
Expected: Outputs zsh completion script.

- [ ] **Step 4: Commit**

```bash
git add cmd/completion.go cmd/root.go
git commit -S -s -m "feat(cli): add shell completion subcommand (bash/zsh/fish/powershell)"
```

---

### Task 4: Add ValidArgsFunction for Enum Flags

**Files:**
- Modify: `cmd/enrich.go:290-299` (newEnrichCmd flag registration)
- Modify: `cmd/publish.go:130-146` (newPublishCmd flag registration)
- Modify: `cmd/generate.go:288-329` (newGenerateCmd flag registration)

- [ ] **Step 1: Add ValidArgsFunction to enrich command flags**

In `cmd/enrich.go`, in `newEnrichCmd()`, after flag registration (after line 299), add:

```go
	// Shell completion for enum flags
	c.RegisterFlagCompletionFunc("provider", cobra.FixedCompletions(
		[]string{"openai", "anthropic", "ollama", "custom"}, cobra.ShellCompDirectiveNoFileComp,
	))
	c.RegisterFlagCompletionFunc("language", cobra.FixedCompletions(
		[]string{"en", "zh"}, cobra.ShellCompDirectiveNoFileComp,
	))
	c.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(
		[]string{"yaml", "json"}, cobra.ShellCompDirectiveNoFileComp,
	))
```

Also add same completions to `init()` for the global `enrichCmd` (after line 330).

- [ ] **Step 2: Add ValidArgsFunction to publish command flags**

In `cmd/publish.go`, in `newPublishCmd()`, after flag registration (after line 146), add:

```go
	c.RegisterFlagCompletionFunc("format", cobra.FixedCompletions(
		[]string{"yaml", "json"}, cobra.ShellCompDirectiveNoFileComp,
	))
	c.RegisterFlagCompletionFunc("to", cobra.FixedCompletions(
		[]string{"readme"}, cobra.ShellCompDirectiveNoFileComp,
	))
```

Also add same completions to `init()` for the global `publishCmd` (after line 180).

- [ ] **Step 3: Add ValidArgsFunction to generate command flags**

In `cmd/generate.go`, in `newGenerateCmd()`, after flag registration (after line 327), add:

```go
	c.RegisterFlagCompletionFunc("output", cobra.FixedCompletions(
		[]string{"yaml", "json"}, cobra.ShellCompDirectiveNoFileComp,
	))
	c.RegisterFlagCompletionFunc("language", cobra.FixedCompletions(
		[]string{"en", "zh"}, cobra.ShellCompDirectiveNoFileComp,
	))
	c.RegisterFlagCompletionFunc("publish-target", cobra.FixedCompletions(
		[]string{"readme"}, cobra.ShellCompDirectiveNoFileComp,
	))
```

Also add same completions to `init()` for the global `generateCmd` (after line 374).

- [ ] **Step 4: Verify completions work**

Run:
```bash
go build -o ./build/spec-forge . && ./build/spec-forge enrich --provider <TAB>
```
Expected: Completes with "openai", "anthropic", "ollama", "custom".

Alternatively verify the script content:
```bash
./build/spec-forge completion bash | grep -A2 "provider"
```
Expected: Contains provider completion values.

- [ ] **Step 5: Commit**

```bash
git add cmd/enrich.go cmd/publish.go cmd/generate.go
git commit -S -s -m "feat(cli): add shell completion for enum flags (provider, language, format, target)"
```

---

### Task 5: Apply Colored Output to Commands

**Files:**
- Modify: `cmd/root.go:40-48` (printHintAndExit)
- Modify: `cmd/generate.go:56-284` (status messages)
- Modify: `cmd/enrich.go:108-204` (status messages)
- Modify: `cmd/publish.go:54-112` (status messages)
- Modify: `cmd/spring.go:63-166` (status messages)

- [ ] **Step 1: Update root.go error output**

Add import for `"github.com/spencercjh/spec-forge/internal/cli"` to `cmd/root.go`.

Replace `printHintAndExit` function (lines 40-48):

```go
func printHintAndExit(err error) {
	if fe, ok := errors.AsType[*forgeerrors.Error](err); ok {
		cli.Errorf(os.Stderr, fe.Error())
		hint := fe.Hint()
		if hint != "" {
			cli.Hintf(os.Stderr, hint)
		}
		os.Exit(exitCodeForCode(fe.Code))
	}
	cli.Errorf(os.Stderr, err.Error())
	os.Exit(1)
}
```

- [ ] **Step 2: Update generate.go status output**

Add import `"github.com/spencercjh/spec-forge/internal/cli"` to `cmd/generate.go`.

Replace status slog calls with colored output. Key changes in `runGenerate()`:

```go
// After detection (line ~90)
cli.Statusf(os.Stderr, "Detected %s project (tool: %s, build: %s)",
	extractorImpl.Name(), info.BuildTool, info.BuildFilePath)

// After patch (lines ~118-126)
if patchResult.DependencyAdded {
	cli.Successf(os.Stderr, "Dependencies added temporarily")
}
if patchResult.PluginAdded {
	cli.Successf(os.Stderr, "Plugin added temporarily")
}
if patchResult.SpringBootConfigured {
	cli.Successf(os.Stderr, "spring-boot-maven-plugin configured with start/stop goals")
}

// After generation (line ~167)
cli.Statusf(os.Stderr, "OpenAPI spec generated: %s (%s)", genResult.SpecFilePath, genResult.Format)

// After validation (lines ~181-191)
if !valResult.Valid {
	cli.Errorf(os.Stderr, "OpenAPI spec validation failed")
	for _, validationErr := range valResult.Errors {
		cli.Errorf(os.Stderr, "  - %s", validationErr)
	}
	return errors.New("generated OpenAPI spec is invalid")
}
cli.Successf(os.Stderr, "OpenAPI spec validated")
} else {
	cli.Skipf(os.Stderr, "Validation skipped")
}

// Enrichment skipped (line ~201)
cli.Skipf(os.Stderr, "Enrichment skipped")

// Spec saved (lines ~224-229)
cli.Successf(os.Stderr, "Spec saved: %s", targetPath)
// or
cli.Successf(os.Stderr, "Spec saved: %s", genResult.SpecFilePath)

// Published (line ~270)
cli.Successf(os.Stderr, "Spec published to %s", pub.Name())

// Complete (line ~282)
cli.Successf(os.Stderr, "Generation complete")
```

Keep `slog.Debug*` calls for verbose logging — only replace `slog.Info*` status messages.

- [ ] **Step 3: Update enrich.go status output**

Add import `"github.com/spencercjh/spec-forge/internal/cli"` to `cmd/enrich.go`.

Replace in `runEnrich()`:

```go
// Line ~108 (start)
cli.Statusf(os.Stderr, "Enriching OpenAPI spec (provider: %s, model: %s, language: %s)",
	prov, model, lang)

// Line ~169 (partial enrichment)
cli.Statusf(os.Stderr, "Partial enrichment: %d/%d batches succeeded",
	partialErr.TotalBatches-partialErr.FailedBatches, partialErr.TotalBatches)

// Line ~204 (complete)
cli.Successf(os.Stderr, "Enrichment complete: %s", outputFile)
```

Replace in `enrichGeneratedSpec()` in `cmd/generate.go`:

```go
// Line ~378
cli.Statusf(os.Stderr, "Enriching OpenAPI spec with AI descriptions...")

// Line ~461
cli.Successf(os.Stderr, "OpenAPI spec enriched: %s", specFilePath)
```

- [ ] **Step 4: Update publish.go status output**

Add import `"github.com/spencercjh/spec-forge/internal/cli"` to `cmd/publish.go`.

Replace in `runPublish()`:

```go
// Line ~54
cli.Statusf(os.Stderr, "Publishing spec to %s", target)

// Line ~62
cli.Statusf(os.Stderr, "Using publisher: %s", pub.Name())

// Lines ~103-110
cli.Successf(os.Stderr, "Published successfully (%d bytes, %s)", result.BytesWritten, result.Format)
if result.Message != "" {
	cli.Statusf(os.Stderr, "%s", result.Message)
}
```

- [ ] **Step 5: Update spring.go status output**

Add import `"github.com/spencercjh/spec-forge/internal/cli"` to `cmd/spring.go`.

Replace `printProjectInfo()` slog calls:

```go
func printProjectInfo(_ context.Context, info *extractor.ProjectInfo) {
	cli.Statusf(os.Stderr, "Spring Project Detection Results")
	cli.Statusf(os.Stderr, "Build Tool: %s", info.BuildTool)
	cli.Statusf(os.Stderr, "Build File: %s", info.BuildFilePath)

	// ... (keep same logic, replace slog.InfoContext with cli.Statusf/Successf)
	if springInfo.HasSpringdocDeps {
		cli.Successf(os.Stderr, "springdoc Dependency: present (v%s)", springInfo.SpringdocVersion)
	} else {
		cli.Statusf(os.Stderr, "springdoc Dependency: not found")
	}
	// etc.
}
```

Replace `runSpringPatch()` status calls similarly.

- [ ] **Step 6: Run tests**

Run:
```bash
make test
```
Expected: All existing tests pass.

- [ ] **Step 7: Commit**

```bash
git add cmd/root.go cmd/generate.go cmd/enrich.go cmd/publish.go cmd/spring.go
git commit -S -s -m "feat(cli): apply colored output to all command status messages"
```

---

### Task 6: Add Progress Bar to Enricher Batch Processing

**Files:**
- Modify: `internal/enricher/processor/concurrent.go`

- [ ] **Step 1: Add progress bar to sequential processing**

In `internal/enricher/processor/concurrent.go`, add import `"github.com/schollz/progressbar/v3"` and `"os"`.

In `processSequential()`, add progress bar creation before the loop:

```go
func (p *ConcurrentProcessor) processSequential(ctx context.Context, batches []*Batch) (*provider.TokenUsage, error) {
	var (
		totalUsage   provider.TokenUsage
		failedCount  int
		failedErrors []error
	)

	bar := progressbar.NewOptions(len(batches),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Enriching OpenAPI spec..."),
		progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
	)

	for i, batch := range batches {
		usage, err := p.batchProcessor.ProcessBatch(ctx, batch)
		if usage != nil {
			totalUsage.Add(usage)
		}
		if err != nil {
			failedCount++
			failedErrors = append(failedErrors, err)
			bar.Describe(fmt.Sprintf("Enriching OpenAPI spec... | %d failed", failedCount))
			slog.Warn("batch processing failed",
				"batch_index", i,
				"batch_type", batch.Type,
				"error", err)
		}
		_ = bar.Add(1)
	}

	_ = bar.Finish()

	if failedCount > 0 {
		return &totalUsage, &PartialEnrichmentError{
			TotalBatches:  len(batches),
			FailedBatches: failedCount,
			Errors:        failedErrors,
		}
	}
	return &totalUsage, nil
}
```

Note: Requires adding `"fmt"` to imports.

- [ ] **Step 2: Add progress bar to concurrent processing**

In `processConcurrent()`, add progress bar:

```go
func (p *ConcurrentProcessor) processConcurrent(ctx context.Context, batches []*Batch) (*provider.TokenUsage, error) {
	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		totalUsage   provider.TokenUsage
		failedCount  int
		failedErrors []error
	)

	bar := progressbar.NewOptions(len(batches),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Enriching OpenAPI spec..."),
		progressbar.OptionOnCompletion(func() { fmt.Fprint(os.Stderr, "\n") }),
	)

	semaphore := make(chan struct{}, p.concurrency)

	for i, batch := range batches {
		wg.Add(1)

		go func(idx int, b *Batch) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			usage, err := p.batchProcessor.ProcessBatch(ctx, b)
			mu.Lock()
			if usage != nil {
				totalUsage.Add(usage)
			}
			if err != nil {
				failedCount++
				failedErrors = append(failedErrors, err)
				bar.Describe(fmt.Sprintf("Enriching OpenAPI spec... | %d failed", failedCount))
				slog.Warn("batch processing failed",
					"batch_index", idx,
					"batch_type", b.Type,
					"error", err)
			}
			mu.Unlock()
			_ = bar.Add(1)
		}(i, batch)
	}

	wg.Wait()
	_ = bar.Finish()

	if failedCount > 0 {
		return &totalUsage, &PartialEnrichmentError{
			TotalBatches:  len(batches),
			FailedBatches: failedCount,
			Errors:        failedErrors,
		}
	}

	return &totalUsage, nil
}
```

- [ ] **Step 3: Run tests**

Run:
```bash
make test
```
Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/enricher/processor/concurrent.go
git commit -S -s -m "feat(enricher): add progress bar for batch processing"
```

---

### Task 7: Final Verification

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run:
```bash
make test
```
Expected: All tests pass.

- [ ] **Step 2: Run linter**

Run:
```bash
make lint
```
Expected: No new lint errors.

- [ ] **Step 3: Format code**

Run:
```bash
make fmt
```

- [ ] **Step 4: Build binary**

Run:
```bash
make build
```
Expected: Build succeeds.

- [ ] **Step 5: Smoke test completion**

Run:
```bash
./build/spec-forge completion bash | head -10
./build/spec-forge completion zsh | head -10
./build/spec-forge --help
```
Expected: Completion scripts output correctly, help shows completion subcommand.

- [ ] **Step 6: Commit any formatting fixes**

```bash
git add -A
git diff --cached --quiet || git commit -S -s -m "style: format code for CLI UX improvements"
```
