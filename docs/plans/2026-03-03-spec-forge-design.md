# Spec Forge v1 设计文档

> **日期：** 2026-03-03
> **状态：** 已批准
> **版本：** v1.0

---

## 概述

Spec Forge 是一个从代码自动生成 OpenAPI 规范的 Go CLI 工具。

**核心价值链：** 源代码 → 框架提取 → AI增强 → OpenAPI规范 → 文档平台

**与同类工具的差异：**
- 多框架支持（v1: Java Spring）
- AI 自动补全接口和字段描述
- 3-way-merge 解决线上冲突（后续版本）
- 优秀架构便于扩展

---

## 1. 核心架构

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI (cobra)                          │
│  spec-forge generate | extract | enrich | publish           │
│  spec-forge spring patch | detect                           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Config (viper)                          │
│  优先级: flag > env > config file > default                 │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   Extractor   │───▶│   Enricher    │───▶│   Publisher   │
│               │    │ (langchaingo) │    │ (Local File)  │
│  ┌─────────┐  │    └───────────────┘    └───────────────┘
│  │Detector │  │            │
│  └─────────┘  │            ▼
│  ┌─────────┐  │    ┌───────────────┐
│  │Generator│  │    │ LLM Providers │
│  └─────────┘  │    │ - OpenAI      │
│  ┌─────────┐  │    │ - Anthropic   │
│  │Validator│  │    │ - Ollama      │
│  └─────────┘  │    │ - 智谱 GLM     │
└───────────────┘    └───────────────┘
```

### 数据流

```
Spring 项目 → springdoc 插件 → openapi.json → Enricher (LLM) → 增强后 openapi.yaml
```

### 关键决策

**不定义中间格式，直接使用 OpenAPI 3.0 Spec 作为数据结构**

---

## 2. 组件设计

### 2.1 Extractor

Extractor 负责从源代码提取 OpenAPI 规范。

| 模块 | 职责 |
|------|------|
| **Detector** | 检测项目类型（Maven/Gradle）、Spring 版本、springdoc 依赖是否存在 |
| **Generator** | 调用 Maven/Gradle 插件生成 openapi.json |
| **Validator** | 验证生成的 OpenAPI Spec 是否有效 |

### 2.2 Enricher

Enricher 使用 LLM 增强 OpenAPI 规范的描述信息。

- 使用 langchaingo 作为 LLM 客户端
- 支持 OpenAI、Anthropic、Ollama、智谱 GLM 四个提供商
- 为缺少描述的接口和字段生成描述

### 2.3 Publisher

Publisher 负责将 OpenAPI 规范发布到目标平台。

- v1 只实现本地文件输出（YAML/JSON）
- 定义通用接口，方便后续扩展

---

## 3. CLI 命令设计

### 命令结构

```bash
# 主命令
spec-forge generate [flags]     # 一键生成

# 子命令
spec-forge extract [flags]      # 只提取
spec-forge enrich [flags]       # 只增强
spec-forge publish [flags]      # 只发布

# 框架特定命令
spec-forge spring patch [flags] # 添加 springdoc 依赖
spec-forge spring detect        # 检测项目信息

# 版本
spec-forge version
```

### 全局参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--config, -c` | 配置文件路径 | `.spec-forge.yaml` |
| `--output, -o` | 输出目录 | `./openapi` |
| `--format, -f` | 输出格式 | `yaml` |
| `--verbose, -v` | 详细日志 | `false` |

### extract 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--strict` | 严格模式，验证失败则报错 | `false` |

### enrich 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--llm-provider` | LLM 提供商 | - |
| `--llm-model` | 模型名称 | - |
| `--no-enrich` | 跳过 AI 增强 | `false` |

### spring patch 参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--dry-run` | 只打印修改，不执行 | `false` |
| `--force` | 强制覆盖依赖版本 | `false` |

---

## 4. 配置文件设计

### 最小配置

```yaml
# .spec-forge.yaml (可选)
enrich:
  provider: openai
  model: gpt-4
  # apiKey 通过环境变量 LLM_API_KEY 提供
```

### 配置优先级

```
flag > env > config file > default
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `LLM_PROVIDER` | LLM 提供商 |
| `LLM_MODEL` | 模型名称 |
| `LLM_API_KEY` | API 密钥（敏感信息只推荐环境变量） |

---

## 5. 项目结构

```
spec-forge/
├── cmd/
│   └── spec-forge/
│       └── main.go              # CLI 入口
├── internal/
│   ├── cmd/                     # cobra 命令定义
│   │   ├── root.go
│   │   ├── generate.go
│   │   ├── extract.go
│   │   ├── enrich.go
│   │   ├── publish.go
│   │   └── spring/
│   │       ├── patch.go
│   │       └── detect.go
│   ├── config/                  # 配置加载
│   ├── extractor/               # Extractor 组件
│   │   ├── extractor.go         # 接口定义
│   │   └── spring/
│   │       ├── detector.go      # 项目检测
│   │       ├── generator.go     # 调用 Maven/Gradle
│   │       ├── validator.go     # OpenAPI 验证
│   │       └── patcher.go       # 依赖注入
│   ├── enricher/                # Enricher 组件
│   │   ├── enricher.go          # 接口定义
│   │   └── llm/
│   │       └── langchain.go     # langchaingo 实现
│   └── publisher/               # Publisher 组件
│       ├── publisher.go         # 接口定义
│       └── local/
│           └── file.go          # 本地文件输出
├── pkg/                         # 可公开使用的包
│   └── openapi/
│       └── spec.go              # OpenAPI 辅助函数
├── configs/
│   └── .spec-forge.yaml         # 示例配置
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 6. 技术选型

| 组件 | 选择 | 说明 |
|------|------|------|
| CLI 框架 | `spf13/cobra` | 最流行的 Go CLI 框架 |
| 配置管理 | `spf13/viper` | 支持 flag/env/config 多源配置 |
| OpenAPI 处理 | `getkin/kin-openapi` | OpenAPI 3.0 解析和验证 |
| Maven 解析 | `vifraa/gopom` | 解析 pom.xml |
| Gradle 解析 | `scagogogo/gradle-parser` | 解析 build.gradle |
| LLM 客户端 | `tmc/langchaingo` | 多提供商支持 |
| YAML 处理 | `gopkg.in/yaml.v3` | 配置文件解析 |

### Go 版本和工具

- **Go 版本：** 1.26
- **Lint：** golangci-lint v2.9.0
- **格式化：** gofmt, goimports

---

## 7. v1 版本范围

### 包含功能

- ✅ CLI 基础框架（cobra + viper）
- ✅ Spring 项目检测
- ✅ `spring patch` 添加 springdoc 依赖
- ✅ 调用 Maven/Gradle 生成 OpenAPI Spec
- ✅ OpenAPI Spec 验证
- ✅ LLM 增强（支持 OpenAI/Anthropic/Ollama/智谱）
- ✅ 本地文件输出（YAML/JSON）

### 不包含（后续版本）

- ❌ 3-way-merge 冲突解决
- ❌ 其他框架支持（Go、Python）
- ❌ 外部平台发布（Postman、Apifox 等）

---

## 8. 开发里程碑

| 里程碑 | 内容 |
|--------|------|
| **M1** | 项目脚手架（go mod、CLI、配置） |
| **M2** | Spring 检测和 Patch（Detector、Patcher） |
| **M3** | Extractor（Generator、Validator） |
| **M4** | Enricher（langchaingo 集成） |
| **M5** | Publisher（本地文件输出） |
| **M6** | 集成测试和文档 |

---

## 9. 测试策略

### 单元测试

- 每个模块都需要单元测试
- 使用标准 `testing` 包
- 表驱动测试

### 集成测试

- 需要用户提供 Spring demo 项目
- 测试完整流程：patch → extract → enrich → publish
- 验证生成的 OpenAPI Spec 正确性

---

## 10. 接口设计

### Extractor 接口

```go
type Extractor interface {
    // Detect 检测项目信息
    Detect(ctx context.Context, projectPath string) (*ProjectInfo, error)

    // Extract 提取 OpenAPI 规范
    Extract(ctx context.Context, projectPath string) (*openapi3.T, error)

    // Validate 验证 OpenAPI 规范
    Validate(ctx context.Context, spec *openapi3.T) error
}
```

### Enricher 接口

```go
type Enricher interface {
    // Enrich 增强 OpenAPI 规范的描述
    Enrich(ctx context.Context, spec *openapi3.T) (*openapi3.T, error)
}
```

### Publisher 接口

```go
type Publisher interface {
    // Publish 发布 OpenAPI 规范
    Publish(ctx context.Context, spec *openapi3.T, opts PublishOptions) error
}
```

---

## 附录

### springdoc-openapi 依赖

Maven:
```xml
<dependency>
    <groupId>org.springdoc</groupId>
    <artifactId>springdoc-openapi-starter-webmvc-ui</artifactId>
    <version>2.3.0</version>
</dependency>
```

Gradle:
```groovy
implementation 'org.springdoc:springdoc-openapi-starter-webmvc-ui:2.3.0'
```
