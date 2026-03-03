# Spec Forge M1: 项目脚手架 实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 搭建 Spec Forge CLI 项目的基础脚手架，包括 Go 模块初始化、CLI 命令结构、配置加载。

**Architecture:** 使用 cobra 构建 CLI，viper 管理配置，采用 internal/pkg 目录结构分离内部和公开代码。

**Tech Stack:** Go 1.26, spf13/cobra, spf13/viper, gopkg.in/yaml.v3

---

## 前置条件

- Go 1.26 已安装
- golangci-lint v2.9.0 已安装
- 工作目录: `/Users/spencercjh/codes/spec-forge`

---

## Task 1: 初始化 Go 模块

**Files:**
- Create: `go.mod`
- Create: `go.sum`

**Step 1: 初始化 Go 模块**

Run: `go mod init github.com/spencercjh/spec-forge`
Expected: 创建 go.mod 文件

**Step 2: 验证 go.mod 内容**

```go
module github.com/spencercjh/spec-forge

go 1.26
```

**Step 3: Commit**

```bash
git init
git add go.mod
git commit -m "chore: initialize go module"
```

---

## Task 2: 创建项目目录结构

**Files:**
- Create: `cmd/spec-forge/main.go`
- Create: `internal/cmd/root.go`
- Create: `internal/config/config.go`
- Create: `pkg/openapi/spec.go` (占位)

**Step 1: 创建目录结构**

Run:
```bash
mkdir -p cmd/spec-forge
mkdir -p internal/cmd
mkdir -p internal/config
mkdir -p internal/extractor/spring
mkdir -p internal/enricher/llm
mkdir -p internal/publisher/local
mkdir -p pkg/openapi
mkdir -p configs
```

**Step 2: 创建空的 .gitkeep 文件（占位）**

Run:
```bash
touch internal/extractor/spring/.gitkeep
touch internal/enricher/llm/.gitkeep
touch internal/publisher/local/.gitkeep
```

**Step 3: Commit**

```bash
git add .
git commit -m "chore: create project directory structure"
```

---

## Task 3: 添加核心依赖

**Files:**
- Modify: `go.mod`
- Create: `go.sum`

**Step 1: 添加 cobra 和 viper 依赖**

Run:
```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get gopkg.in/yaml.v3
```

**Step 2: 验证依赖已添加**

检查 go.mod 包含:
```
require (
    github.com/spf13/cobra v1.x.x
    github.com/spf13/viper v1.x.x
    gopkg.in/yaml.v3 v3.x.x
)
```

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add cobra and viper dependencies"
```

---

## Task 4: 实现 root 命令

**Files:**
- Create: `internal/cmd/root.go`
- Create: `cmd/spec-forge/main.go`

**Step 1: 创建 root 命令**

创建 `internal/cmd/root.go`:

```go
// Package cmd contains all CLI commands for spec-forge.
package cmd

import (
	"fmt"
	"os"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "spec-forge",
	Short: "Generate OpenAPI specifications from source code",
	Long: `Spec Forge is a CLI tool that automatically generates OpenAPI specifications
from your source code. It supports multiple frameworks and uses AI to enhance
API descriptions.

Core workflow: Source Code → Extract → Enrich → Publish`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .spec-forge.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		panic(err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".spec-forge")
	}

	viper.SetEnvPrefix("SPEC_FORGE")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}

	// Load configuration
	config.Load()
}
```

**Step 2: 创建 main 入口**

创建 `cmd/spec-forge/main.go`:

```go
// Package main is the entry point for spec-forge CLI.
package main

import "github.com/spencercjh/spec-forge/internal/cmd"

func main() {
	cmd.Execute()
}
```

**Step 3: 验证编译**

Run: `go build -o bin/spec-forge ./cmd/spec-forge`
Expected: 编译成功，生成 bin/spec-forge

**Step 4: 测试运行**

Run: `./bin/spec-forge --help`
Expected: 显示帮助信息

Run: `./bin/spec-forge version`
Expected: 显示版本 "0.1.0"

**Step 5: Commit**

```bash
git add internal/cmd/root.go cmd/spec-forge/main.go
git commit -m "feat: add root command and CLI entry point"
```

---

## Task 5: 实现配置加载

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: 写配置结构测试**

创建 `internal/config/config_test.go`:

```go
package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Output.Dir != "./openapi" {
		t.Errorf("expected default output dir ./openapi, got %s", cfg.Output.Dir)
	}
	if cfg.Output.Format != "yaml" {
		t.Errorf("expected default format yaml, got %s", cfg.Output.Format)
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test ./internal/config/...`
Expected: FAIL - config.go not exists

**Step 3: 实现配置结构**

创建 `internal/config/config.go`:

```go
// Package config handles configuration loading and management.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the complete configuration for spec-forge.
type Config struct {
	Enrich  EnrichConfig  `mapstructure:"enrich"`
	Output  OutputConfig  `mapstructure:"output"`
	Extract ExtractConfig `mapstructure:"extract"`
}

// EnrichConfig contains LLM enrichment settings.
type EnrichConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"apiKey"`
}

// OutputConfig contains output settings.
type OutputConfig struct {
	Dir    string `mapstructure:"dir"`
	Format string `mapstructure:"format"`
}

// ExtractConfig contains extraction settings.
type ExtractConfig struct {
	Strict bool `mapstructure:"strict"`
}

// global is the global configuration instance.
var global *Config

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Enrich: EnrichConfig{
			Enabled: true,
		},
		Output: OutputConfig{
			Dir:    "./openapi",
			Format: "yaml",
		},
		Extract: ExtractConfig{
			Strict: false,
		},
	}
}

// Load loads configuration from viper and returns the global config.
func Load() *Config {
	cfg := Default()

	// Unmarshal from viper
	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Printf("warning: failed to unmarshal config: %v\n", err)
	}

	// Override with environment variables
	if apiKey := viper.GetString("llm_api_key"); apiKey != "" {
		cfg.Enrich.APIKey = apiKey
	}

	// Override with flags
	if dir := viper.GetString("output"); dir != "" {
		cfg.Output.Dir = dir
	}
	if format := viper.GetString("format"); format != "" {
		cfg.Output.Format = format
	}

	global = cfg
	return cfg
}

// Get returns the global configuration.
func Get() *Config {
	if global == nil {
		return Load()
	}
	return global
}
```

**Step 4: 运行测试验证通过**

Run: `go test ./internal/config/... -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add configuration loading with viper"
```

---

## Task 6: 添加 generate 命令

**Files:**
- Create: `internal/cmd/generate.go`
- Create: `internal/cmd/generate_test.go`

**Step 1: 写 generate 命令测试**

创建 `internal/cmd/generate_test.go`:

```go
package cmd

import (
	"testing"
)

func TestGenerateCommandExists(t *testing.T) {
	cmd := generateCmd
	if cmd == nil {
		t.Fatal("generateCmd should not be nil")
	}
	if cmd.Use != "generate" {
		t.Errorf("expected Use 'generate', got %s", cmd.Use)
	}
}
```

**Step 2: 运行测试验证失败**

Run: `go test ./internal/cmd/...`
Expected: FAIL - generate.go not exists

**Step 3: 实现 generate 命令**

创建 `internal/cmd/generate.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// generateCmd represents the generate command.
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate OpenAPI spec from source code",
	Long: `Generate OpenAPI specification by running the complete pipeline:
extract → enrich → publish

This is the main command that orchestrates the entire workflow.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("generate called - implementation coming soon")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
```

**Step 4: 运行测试验证通过**

Run: `go test ./internal/cmd/... -v`
Expected: PASS

**Step 5: 验证命令可用**

Run: `go build -o bin/spec-forge ./cmd/spec-forge && ./bin/spec-forge generate --help`
Expected: 显示 generate 命令帮助

**Step 6: Commit**

```bash
git add internal/cmd/generate.go internal/cmd/generate_test.go
git commit -m "feat: add generate command skeleton"
```

---

## Task 7: 添加 extract 命令

**Files:**
- Create: `internal/cmd/extract.go`

**Step 1: 实现 extract 命令**

创建 `internal/cmd/extract.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var extractStrict bool

// extractCmd represents the extract command.
var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract OpenAPI spec from source code",
	Long:  `Extract OpenAPI specification from the source code using framework-specific extractors.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("extract called - implementation coming soon")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)

	extractCmd.Flags().BoolVar(&extractStrict, "strict", false, "fail if validation errors occur")
}
```

**Step 2: 验证命令可用**

Run: `go build -o bin/spec-forge ./cmd/spec-forge && ./bin/spec-forge extract --help`
Expected: 显示 extract 命令帮助，包含 --strict flag

**Step 3: Commit**

```bash
git add internal/cmd/extract.go
git commit -m "feat: add extract command skeleton"
```

---

## Task 8: 添加 enrich 命令

**Files:**
- Create: `internal/cmd/enrich.go`

**Step 1: 实现 enrich 命令**

创建 `internal/cmd/enrich.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	llmProvider string
	llmModel    string
	noEnrich    bool
)

// enrichCmd represents the enrich command.
var enrichCmd = &cobra.Command{
	Use:   "enrich",
	Short: "Enrich OpenAPI spec with AI-generated descriptions",
	Long:  `Enrich OpenAPI specification by using LLM to generate missing descriptions for APIs and fields.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("enrich called - implementation coming soon")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(enrichCmd)

	enrichCmd.Flags().StringVar(&llmProvider, "llm-provider", "", "LLM provider (openai, anthropic, ollama, zhipu)")
	enrichCmd.Flags().StringVar(&llmModel, "llm-model", "", "LLM model name")
	enrichCmd.Flags().BoolVar(&noEnrich, "no-enrich", false, "skip AI enrichment")
}
```

**Step 2: 验证命令可用**

Run: `go build -o bin/spec-forge ./cmd/spec-forge && ./bin/spec-forge enrich --help`
Expected: 显示 enrich 命令帮助，包含所有 flags

**Step 3: Commit**

```bash
git add internal/cmd/enrich.go
git commit -m "feat: add enrich command skeleton"
```

---

## Task 9: 添加 publish 命令

**Files:**
- Create: `internal/cmd/publish.go`

**Step 1: 实现 publish 命令**

创建 `internal/cmd/publish.go`:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// publishCmd represents the publish command.
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish OpenAPI spec to target platform",
	Long:  `Publish OpenAPI specification to local files or external platforms.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("publish called - implementation coming soon")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(publishCmd)
}
```

**Step 2: 验证命令可用**

Run: `go build -o bin/spec-forge ./cmd/spec-forge && ./bin/spec-forge publish --help`
Expected: 显示 publish 命令帮助

**Step 3: Commit**

```bash
git add internal/cmd/publish.go
git commit -m "feat: add publish command skeleton"
```

---

## Task 10: 添加 spring 父命令和 detect 子命令

**Files:**
- Create: `internal/cmd/spring.go`
- Create: `internal/cmd/spring/detect.go`

**Step 1: 实现 spring 父命令**

创建 `internal/cmd/spring.go`:

```go
package cmd

import (
	"github.com/spf13/cobra"
)

// springCmd represents the spring command group.
var springCmd = &cobra.Command{
	Use:   "spring",
	Short: "Spring framework specific commands",
	Long:  `Commands for working with Spring (Java) projects.`,
}

func init() {
	rootCmd.AddCommand(springCmd)
}
```

**Step 2: 实现 spring detect 子命令**

创建 `internal/cmd/spring/detect.go`:

```go
package spring

import (
	"fmt"

	"github.com/spencercjh/spec-forge/internal/config"
	"github.com/spf13/cobra"
)

// detectCmd represents the spring detect command.
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect Spring project information",
	Long:  `Analyze the current directory to detect Spring project type, build tool, and dependencies.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		fmt.Printf("Detecting Spring project...\n")
		fmt.Printf("Config: %+v\n", cfg)
		fmt.Println("detect called - implementation coming soon")
		return nil
	},
}

func init() {
	// springCmd is added in spring.go
}

// GetDetectCmd returns the detect command for registration.
func GetDetectCmd() *cobra.Command {
	return detectCmd
}
```

**Step 3: 在 spring.go 中注册 detect 子命令**

修改 `internal/cmd/spring.go`:

```go
package cmd

import (
	"github.com/spencercjh/spec-forge/internal/cmd/spring"
	"github.com/spf13/cobra"
)

// springCmd represents the spring command group.
var springCmd = &cobra.Command{
	Use:   "spring",
	Short: "Spring framework specific commands",
	Long:  `Commands for working with Spring (Java) projects.`,
}

func init() {
	rootCmd.AddCommand(springCmd)
	springCmd.AddCommand(spring.GetDetectCmd())
}
```

**Step 4: 验证命令可用**

Run: `go build -o bin/spec-forge ./cmd/spec-forge && ./bin/spec-forge spring --help`
Expected: 显示 spring 命令组，包含 detect

**Step 5: Commit**

```bash
git add internal/cmd/spring.go internal/cmd/spring/
git commit -m "feat: add spring command group and detect subcommand"
```

---

## Task 11: 添加 spring patch 子命令

**Files:**
- Create: `internal/cmd/spring/patch.go`

**Step 1: 实现 spring patch 命令**

创建 `internal/cmd/spring/patch.go`:

```go
package spring

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	patchDryRun bool
	patchForce  bool
)

// patchCmd represents the spring patch command.
var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: "Add springdoc dependencies to Spring project",
	Long: `Add springdoc-openapi dependencies to the Spring project's build file.
Supports both Maven (pom.xml) and Gradle (build.gradle) projects.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if patchDryRun {
			fmt.Println("Dry run mode - showing changes without modifying files")
		}
		fmt.Println("patch called - implementation coming soon")
		return nil
	},
}

func init() {
	patchCmd.Flags().BoolVar(&patchDryRun, "dry-run", false, "show changes without modifying files")
	patchCmd.Flags().BoolVar(&patchForce, "force", false, "force overwrite existing dependencies")
}

// GetPatchCmd returns the patch command for registration.
func GetPatchCmd() *cobra.Command {
	return patchCmd
}
```

**Step 2: 在 spring.go 中注册 patch 子命令**

修改 `internal/cmd/spring.go`:

```go
package cmd

import (
	"github.com/spencercjh/spec-forge/internal/cmd/spring"
	"github.com/spf13/cobra"
)

// springCmd represents the spring command group.
var springCmd = &cobra.Command{
	Use:   "spring",
	Short: "Spring framework specific commands",
	Long:  `Commands for working with Spring (Java) projects.`,
}

func init() {
	rootCmd.AddCommand(springCmd)
	springCmd.AddCommand(spring.GetDetectCmd())
	springCmd.AddCommand(spring.GetPatchCmd())
}
```

**Step 3: 验证命令可用**

Run: `go build -o bin/spec-forge ./cmd/spec-forge && ./bin/spec-forge spring patch --help`
Expected: 显示 spring patch 命令帮助，包含 --dry-run 和 --force flags

**Step 4: Commit**

```bash
git add internal/cmd/spring.go internal/cmd/spring/patch.go
git commit -m "feat: add spring patch command skeleton"
```

---

## Task 12: 添加 Makefile

**Files:**
- Create: `Makefile`

**Step 1: 创建 Makefile**

创建 `Makefile`:

```makefile
.PHONY: build clean test lint fmt help

BINARY_NAME=spec-forge
MAIN_PATH=./cmd/spec-forge

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

clean: ## Clean build artifacts
	rm -rf bin/

test: ## Run tests
	go test -v ./...

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	goimports -w .

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

all: fmt lint test build ## Run all checks and build
```

**Step 2: 验证 Makefile 可用**

Run: `make build`
Expected: 编译成功

Run: `make test`
Expected: 所有测试通过

**Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile with build, test, lint targets"
```

---

## Task 13: 添加 golangci-lint 配置

**Files:**
- Create: `.golangci.yml`

**Step 1: 创建 golangci-lint 配置**

创建 `.golangci.yml`:

```yaml
run:
  timeout: 5m
  go: "1.26"

linters:
  enable:
    - gofmt
    - goimports
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - revive

linters-settings:
  gofmt:
    simplify: true
  goimports:
    local-prefixes: github.com/spencercjh/spec-forge
  revive:
    rules:
      - name: exported
        arguments:
          - checkPrivateReceivers
          - sayRepetitiveInsteadOfRepeats

issues:
  exclude-rules:
    - path: _test\.go
      linters:
        - errcheck
```

**Step 2: 验证 lint 通过**

Run: `make lint`
Expected: 无错误

**Step 3: Commit**

```bash
git add .golangci.yml
git commit -m "chore: add golangci-lint configuration"
```

---

## Task 14: 添加示例配置文件

**Files:**
- Create: `configs/.spec-forge.yaml`
- Create: `.spec-forge.yaml` (根目录，用于开发测试)

**Step 1: 创建示例配置**

创建 `configs/.spec-forge.yaml`:

```yaml
# Spec Forge Configuration Example
# Copy this file to your project root and customize as needed.

# LLM enrichment settings
enrich:
  enabled: true
  provider: openai  # openai, anthropic, ollama, zhipu
  model: gpt-4
  # apiKey should be set via LLM_API_KEY environment variable

# Output settings
output:
  dir: ./openapi
  format: yaml  # yaml or json

# Extract settings
extract:
  strict: false
```

**Step 2: 创建开发测试用配置**

创建 `.spec-forge.yaml`:

```yaml
enrich:
  enabled: false  # disabled for development
```

**Step 3: Commit**

```bash
git add configs/.spec-forge.yaml .spec-forge.yaml
git commit -m "docs: add example configuration file"
```

---

## Task 15: 添加 .gitignore

**Files:**
- Create: `.gitignore`

**Step 1: 创建 .gitignore**

创建 `.gitignore`:

```gitignore
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of go coverage
*.out
coverage.html

# Dependency directories
vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store

# Local config with secrets
.spec-forge.local.yaml

# Generated OpenAPI specs (during development)
openapi/
```

**Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore"
```

---

## Task 16: 最终验证

**Step 1: 运行所有测试**

Run: `make test`
Expected: 所有测试通过

**Step 2: 运行 lint**

Run: `make lint`
Expected: 无错误

**Step 3: 运行完整构建**

Run: `make all`
Expected: 编译成功

**Step 4: 验证所有命令**

Run:
```bash
./bin/spec-forge --help
./bin/spec-forge version
./bin/spec-forge generate --help
./bin/spec-forge extract --help
./bin/spec-forge enrich --help
./bin/spec-forge publish --help
./bin/spec-forge spring --help
./bin/spec-forge spring detect --help
./bin/spec-forge spring patch --help
```
Expected: 所有命令显示正确帮助信息

---

## 里程碑 M1 完成检查清单

- [ ] Go 模块已初始化 (go.mod, go.sum)
- [ ] 目录结构已创建
- [ ] 依赖已添加 (cobra, viper, yaml.v3)
- [ ] CLI 命令可用:
  - [ ] root (version, help)
  - [ ] generate
  - [ ] extract
  - [ ] enrich
  - [ ] publish
  - [ ] spring detect
  - [ ] spring patch
- [ ] 配置加载可用
- [ ] 测试通过
- [ ] Lint 通过
- [ ] Makefile 可用
- [ ] 示例配置文件存在

---

## 后续里程碑

M1 完成后，将创建以下里程碑的详细计划：

- **M2:** Spring 检测和 Patch (Detector, Patcher)
- **M3:** Extractor (Generator, Validator)
- **M4:** Enricher (langchaingo 集成)
- **M5:** Publisher (本地文件输出)
- **M6:** 集成测试和文档

**注意：** M2 开始需要你提供 Spring demo 项目用于集成测试。
