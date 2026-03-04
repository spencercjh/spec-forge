# M4: Enricher 设计文档

> **日期：** 2026-03-04
> **状态：** 已批准
> **里程碑：** M4

---

## 概述

M4 实现使用 LLM 增强 OpenAPI Spec 的描述信息。

**核心功能：**
- 全面增强：API 操作、Schema 字段、参数、响应等所有缺失描述的元素
- 分组批量调用：平衡 token 限制和调用次数
- 多 Provider 支持：OpenAI、Anthropic、Ollama、自定义 OpenAI 兼容服务

---

## 1. 整体架构

```
internal/enricher/
├── enricher.go           # 接口定义和主入口
├── config.go             # 配置结构
├── errors.go             # 错误类型定义
├── prompt/
│   ├── templates.go      # 内置 prompt 模板
│   └── loader.go         # 自定义模板加载
├── provider/
│   ├── provider.go       # Provider 接口
│   ├── factory.go        # 工厂函数
│   └── openai_compatible.go  # 通用 OpenAI 兼容 Provider
├── processor/
│   ├── processor.go      # 分组逻辑
│   ├── batch.go          # 批量调用
│   └── concurrent.go     # 并发控制
└── *_test.go
```

**数据流：**
```
OpenAPI Spec → 分组器 → 批处理队列 → 并发调用 LLM → 合并结果 → 增强 Spec
                  ↓
              应用 Prompt 模板
```

---

## 2. 核心类型和接口

### 2.1 配置结构

```go
// internal/enricher/config.go

// Config Enricher 配置
type Config struct {
    // Provider 类型: "openai", "anthropic", "ollama", "custom"
    Provider string

    // 通用配置
    Model       string
    Language    string        // 输出语言，默认 "en"
    Concurrency int           // 并发数，默认 3
    MaxRetries  int           // 最大重试次数，默认 2
    Timeout     time.Duration // 单次调用超时，默认 30s

    // 自定义 Provider 配置（当 Provider == "custom" 时使用）
    CustomBaseURL   string            // API 地址
    CustomAPIKeyEnv string            // API Key 环境变量名，默认 "LLM_API_KEY"
    CustomHeaders   map[string]string // 额外 Headers

    // 高级配置
    PromptTemplateDir string // 自定义模板目录
}

var DefaultConfig = Config{
    Language:    "en",
    Concurrency: 3,
    MaxRetries:  2,
    Timeout:     30 * time.Second,
}
```

### 2.2 Provider 接口

```go
// internal/enricher/provider/provider.go

// Provider LLM 提供商接口
type Provider interface {
    Generate(ctx context.Context, prompt string) (string, error)
    Name() string
}
```

### 2.3 通用 OpenAI 兼容 Provider

```go
// internal/enricher/provider/openai_compatible.go

// OpenAICompatibleConfig OpenAI 兼容服务配置
type OpenAICompatibleConfig struct {
    BaseURL      string            // API 地址
    APIKey       string            // API 密钥
    Model        string            // 模型名称
    ExtraHeaders map[string]string // 额外 Headers
}

// OpenAICompatibleProvider 通用的 OpenAI 兼容 Provider
type OpenAICompatibleProvider struct {
    config OpenAICompatibleConfig
    client *http.Client
}

func NewOpenAICompatibleProvider(config OpenAICompatibleConfig) *OpenAICompatibleProvider
func (p *OpenAICompatibleProvider) Generate(ctx context.Context, prompt string) (string, error)
func (p *OpenAICompatibleProvider) Name() string
```

### 2.4 预设 Provider 工厂

```go
// internal/enricher/provider/factory.go

// 预设配置工厂
func NewOpenAIProvider(apiKey, model string) *OpenAICompatibleProvider
func NewOllamaProvider(baseURL, model string) *OpenAICompatibleProvider
func NewInternalOpenAIProvider(baseURL, apiKey, model string, headers map[string]string) *OpenAICompatibleProvider
```

### 2.5 Enricher 接口

```go
// internal/enricher/enricher.go

// Enricher 增强器接口
type Enricher interface {
    Enrich(ctx context.Context, spec *openapi3.T, opts *EnrichOptions) (*openapi3.T, error)
}

// EnrichOptions 增强选项
type EnrichOptions struct {
    Language string // 运行时语言覆盖
}
```

---

## 3. Prompt 模板系统

### 3.1 模板类型

```go
// internal/enricher/prompt/templates.go

type TemplateType string

const (
    TemplateTypeAPI      TemplateType = "api"       // API 操作描述
    TemplateTypeSchema   TemplateType = "schema"    // Schema 字段描述
    TemplateTypeParam    TemplateType = "param"     // 参数描述
    TemplateTypeResponse TemplateType = "response"  // 响应描述
)
```

### 3.2 模板上下文

```go
// Context 传递给模板的上下文
type TemplateContext struct {
    Type     TemplateType
    Language string

    // 最小上下文
    Path         string // API 路径，如 "GET /users/{id}"
    SchemaName   string // Schema 名称，如 "User"
    FieldName    string // 字段名，如 "userId"
    ParamName    string // 参数名
    ResponseCode string // 响应状态码

    // 元信息
    FieldType string
    Required  bool
}
```

### 3.3 内置模板示例

```yaml
# API 操作模板
system: |
  You are an API documentation expert. Generate concise, clear descriptions.
  Respond in {{.Language}} language.
  Output format: JSON with "summary" and "description" fields.

user: |
  API Endpoint: {{.Path}}
  HTTP Method: {{.Method}}

  Generate the summary (one line) and description (1-3 sentences) for this API.
```

```yaml
# Schema 字段模板
system: |
  You are an API documentation expert. Generate concise field descriptions.
  Respond in {{.Language}} language.
  Output format: JSON mapping field names to descriptions.

user: |
  Schema: {{.SchemaName}}
  Fields:
  {{range .Fields}}- {{.Name}} ({{.Type}}, {{if .Required}}required{{else}}optional{{end}})
  {{end}}

  Generate a description for each field.
```

### 3.4 自定义模板

```go
// Loader 加载自定义模板
type Loader struct {
    templateDir string
}

func (l *Loader) Load(templateType TemplateType) (*Template, error)
```

---

## 4. 分组批处理逻辑

### 4.1 元素定义

```go
// internal/enricher/processor/processor.go

// EnrichmentElement 待增强的元素
type EnrichmentElement struct {
    Type     TemplateType
    Path     string // OpenAPI Spec 中的路径
    Context  TemplateContext
    SetValue func(description string) // 设置描述的回调
}

// Batch 待处理的批次
type Batch struct {
    Type     TemplateType
    Elements []EnrichmentElement
}
```

### 4.2 分组规则

| 元素类型 | 分组依据 | 示例 |
|---------|---------|------|
| API 操作 | 按 tag 分组 | `["user-controller"]` 下的所有操作 |
| Schema 字段 | 按 Schema 分组 | `User` Schema 的所有字段 |
| 参数 | 按 API 操作分组 | `GET /users` 的所有参数 |
| 响应 | 按 API 操作分组 | `GET /users` 的所有响应 |

### 4.3 批处理器

```go
// internal/enricher/processor/batch.go

type BatchProcessor struct {
    provider    Provider
    templateMgr *TemplateManager
    config      *Config
}

func (p *BatchProcessor) ProcessBatch(ctx context.Context, batch *Batch) error
```

### 4.4 并发控制

```go
// internal/enricher/processor/concurrent.go

type ConcurrentProcessor struct {
    batchProcessor *BatchProcessor
    concurrency    int
    semaphore      chan struct{}
}

func (p *ConcurrentProcessor) ProcessAll(ctx context.Context, batches []*Batch) error
```

---

## 5. CLI 命令

### 5.1 generate 命令更新

```bash
spec-forge generate [path] [flags]

新增 Flags:
  --skip-enrich     跳过 AI 增强
  --language string  描述语言 (默认 "en")
```

### 5.2 enrich 命令

```bash
spec-forge enrich <spec-file> [flags]

Flags:
  --provider string           LLM provider (openai, anthropic, ollama, custom)
  --model string              模型名称
  --language string           描述语言 (默认 "en")
  --output, -o string         输出文件 (默认覆盖输入)
  --concurrency int           并发数 (默认 3)
  --timeout duration          单次调用超时 (默认 30s)
  --custom-base-url string    自定义 provider 地址
  --custom-api-key-env string API Key 环境变量名 (默认 "LLM_API_KEY")
  --prompt-template-dir string 自定义模板目录
```

### 5.3 完整流程

```
┌─────────┐    ┌─────────┐    ┌───────────┐    ┌───────────┐    ┌─────────┐    ┌─────────┐
│ Detect  │───▶│  Patch  │───▶│ Generate  │───▶│ Validate  │───▶│ Enrich   │───▶│ Restore │
└─────────┘    └─────────┘    └───────────┘    └───────────┘    └───────────┘    └─────────┘
                                                                    │
                                                           --skip-enrich
                                                                    │
                                                                    ▼
                                                               跳过此步骤
```

---

## 6. 配置文件

```yaml
# .spec-forge.yaml

enrich:
  # Provider 配置
  provider: openai  # openai, anthropic, ollama, custom
  model: gpt-4o

  # 输出配置
  language: en

  # 性能配置
  concurrency: 3
  timeout: 30s

  # 自定义 provider（当 provider: custom 时）
  customBaseURL: https://ai.company.com/v1
  customAPIKeyEnv: COMPANY_AI_API_KEY
  customHeaders:
    X-Tenant-ID: my-team

  # 高级配置
  promptTemplateDir: ./prompts
```

---

## 7. 环境变量

| 变量 | 说明 |
|------|------|
| `OPENAI_API_KEY` | OpenAI API Key |
| `ANTHROPIC_API_KEY` | Anthropic API Key |
| `LLM_API_KEY` | 默认/自定义 API Key |

**配置优先级：** flag > env > config file > default

---

## 8. 错误处理

### 8.1 错误类型

```go
// EnrichmentError 增强过程中的错误
type EnrichmentError struct {
    Type    string // "llm_call", "parse", "template", "config"
    Message string
    Cause   error
}

// PartialEnrichmentError 部分增强失败
type PartialEnrichmentError struct {
    TotalBatches  int
    FailedBatches int
    Errors        []error
}
```

### 8.2 处理策略

| 场景 | 处理方式 |
|------|---------|
| 配置错误（无 API Key） | 立即终止，显示配置提示 |
| 单批次 LLM 调用失败 | 跳过该批次，保留原始描述，继续其他批次 |
| 响应解析失败 | 跳过该批次，记录警告日志 |
| 全部批次失败 | 返回 PartialEnrichmentError，spec 仍然有效 |
| Context 取消 | 立即终止，返回已处理的部分结果 |

### 8.3 用户提示

```
No LLM API key found. Please configure one of:

  # OpenAI
  export OPENAI_API_KEY=sk-...

  # Anthropic
  export ANTHROPIC_API_KEY=sk-ant-...

  # Or skip enrichment
  spec-forge generate . --skip-enrich
```

---

## 9. 测试策略

### 9.1 单元测试

| 测试文件 | 覆盖内容 |
|---------|---------|
| `enricher_test.go` | Enricher 接口实现、完整流程 |
| `provider_test.go` | Provider 接口、工厂函数 |
| `openai_compatible_test.go` | HTTP 请求构建、响应解析 |
| `templates_test.go` | 模板渲染、变量替换 |
| `loader_test.go` | 自定义模板加载、验证 |
| `processor_test.go` | 元素分组、批次构建 |
| `batch_test.go` | 批量调用、响应解析 |
| `concurrent_test.go` | 并发控制、信号量 |

### 9.2 Mock Provider

```go
type MockProvider struct {
    GenerateFunc func(ctx context.Context, prompt string) (string, error)
}
```

### 9.3 集成测试

```go
//go:build integration

func TestEnricher_OpenAI_Real(t *testing.T) {
    if os.Getenv("OPENAI_API_KEY") == "" {
        t.Skip("OPENAI_API_KEY not set")
    }
    // 真实调用测试
}
```

---

## 10. 依赖

```go
// go.mod
require (
    github.com/getkin/kin-openapi v0.133.0  // 已有
    // 无需新增依赖，使用标准库 net/http 调用 OpenAI 兼容 API
)
```

---

## 11. 使用示例

```bash
# 完整流程（包含 AI 增强）
spec-forge generate ./my-project

# 跳过 AI 增强
spec-forge generate ./my-project --skip-enrich

# 指定中文描述
spec-forge generate ./my-project --language zh

# 使用公司内部 AI
spec-forge generate ./my-project \
  --provider custom \
  --custom-base-url https://ai.company.com/v1 \
  --custom-api-key-env COMPANY_AI_KEY

# 单独增强已生成的 spec
spec-forge enrich openapi.json --language zh
```

---

## 12. 设计决策总结

| 项目 | 决策 |
|------|------|
| 增强范围 | 全面增强（API、Schema、参数、响应） |
| 调用策略 | 分组批量调用 |
| 上下文 | 最小上下文（元素 + 所在结构） |
| Provider 架构 | 通用 OpenAI 兼容 + 预设配置 |
| v1 支持的 Provider | OpenAI、Anthropic、Ollama、Custom |
| 配置方式 | API Key 仅环境变量，其他可用配置文件 |
| 输出语言 | 默认英文，`--language` 可配置 |
| 错误处理 | 跳过失败批次，继续处理 |
| 并发控制 | 可配置并发数，默认 3 |
| CLI 集成 | `generate --skip-enrich` 跳过 |

---

## 13. 后续扩展

- **丰富上下文模式：** 通过 flag 启用，传递相邻元素信息
- **智谱 GLM 支持：** 实现 Provider 接口
- **Streaming 输出：** 实时显示增强进度
- **缓存机制：** 避免重复调用相同内容
