# Spec Forge

[![Go Report Card](https://goreportcard.com/badge/github.com/spencercjh/spec-forge)](https://goreportcard.com/report/github.com/spencercjh/spec-forge)
[![GoDoc](https://godoc.org/github.com/spencercjh/spec-forge?status.svg)](https://godoc.org/github.com/spencercjh/spec-forge)
[![CI](https://github.com/spencercjh/spec-forge/actions/workflows/ci.yml/badge.svg)](https://github.com/spencercjh/spec-forge/actions/workflows/ci.yml)
[![Copilot code review](https://github.com/spencercjh/spec-forge/actions/workflows/copilot-pull-request-reviewer/copilot-pull-request-reviewer/badge.svg)](https://github.com/spencercjh/spec-forge/actions/workflows/copilot-pull-request-reviewer/copilot-pull-request-reviewer)
[![Dependabot Updates](https://github.com/spencercjh/spec-forge/actions/workflows/dependabot/dependabot-updates/badge.svg)](https://github.com/spencercjh/spec-forge/actions/workflows/dependabot/dependabot-updates)

一款解决 OpenAPI 规范生成碎片化、痛苦体验的 CLI 工具 —— 自动检测你的框架，从源代码生成准确的规范，并通过 AI 进行增强。

## 快速开始

```bash
go install github.com/spencercjh/spec-forge@latest
spec-forge generate ./path/to/project
```

详细的安装和使用指南请参见 [Quick Start Guide](./docs/quick-start.md)（英文）。

---

## 为什么选择 Spec Forge？

从后端代码生成 OpenAPI 规范比想象中困难。现有工具迫使你做出痛苦的取舍：繁琐的注释破坏重构、无人维护的生成器产生错误输出，或者手动维护的规范与代码脱节。

Spec Forge 通过 **零注释 AST 分析**（针对 Go Web 框架）、在官方工具不足时提供 **稳健的生成能力**，以及真正理解代码结构的 **AI 增强** 来解决这些问题。

### 技术负责人的困境

> 你是一名技术负责人。产品经理刚告诉你，你们团队的 API 需要在下周交给另一个团队对接。他们需要正式的 API 文档 —— OpenAPI 规范，而不是 Markdown 文件。
>
> 你询问后端开发人员。一脸茫然。
>
> "我们从未生成过 API 文档。我们就是写代码，偶尔更新一下内部 wiki。"
>
> 你检查代码库。数百个 API，横跨 Spring Boot、Gin 和 go-zero 服务。没有注释，没有现有的规范。只有手写的 Markdown 表格，上次更新还是三个月前。

这是大多数工程团队的现实。**API 文档是事后诸葛亮**，因为工具太复杂、太碎片化，或者需要开发者从未养成的习惯。

**Spec Forge 改变了这个等式。** 一条命令就能从现有代码生成准确的规范 —— 无需添加注释，无需复杂配置，无需"文档冲刺"。你的团队交付可用的 API *和* 正式的文档，而无需改变他们的编码方式。

### Go Web 框架：注释陷阱

**Gin 等框架** —— 主流解决方案是 [`swaggo/swag`](https://github.com/swaggo/swag)，它是一个注释噩梦：

```go
// @Summary      根据 ID 获取用户
// @Description  通过唯一标识符获取用户详情
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      int     true  "用户 ID"
// @Success      200  {object}  User    "找到用户"
// @Failure      404  {object}  Error   "用户不存在"
// @Failure      500  {object}  Error   "内部服务器错误"
// @Router       /users/{id} [get]
func GetUser(c *gin.Context) { ... }
```

**注释黑洞：**

1. **无类型检查** —— 这些注释对 Go 编译器是不可见的。将 `User` 重命名为 `UserResponse`，你的规范会静默崩溃。Swag 只有在生成时（如果你记得运行的话）才能发现这个问题。

2. **重构地狱** —— 修改字段名、添加查询参数，或拆分 handler？你要手动更新跨多个文件的数十个注释。这些注释成为了第二个脆弱的代码库，镜像着你的真实代码。

3. **视觉污染** —— 每个 handler 有 10 行噪音。一个中等规模的 API 会累积数百行注释，遮蔽了实际逻辑。

4. **默认过时** —— 因为没有编译时验证，规范与现实脱节。开发者忘记重新生成，或者更糟的是，因为规范经常错误而不再信任它。

**Spec Forge 零注释要求。** 我们解析 Go AST 从 `gin.Engine` 提取路由，分析 handler 签名，直接映射请求/响应结构体。重命名类型，规范自动更新。无需维护注释，没有过期引用，没有视觉噪音。

### go-zero：停滞的生成器

go-zero 官方的 `goctl api swagger` 工具在基础场景下可以工作，但 **开发已经停滞**：

- **卡在 Swagger 2.0** —— 生成 Swagger 2.0 规范而非现代 OpenAPI 3.x，需要转换才能用于现代工具
- **问题响应缓慢** —— 社区报告的 bug 和功能请求得到的维护者响应有限
- **小瑕疵** —— 各种边界情况（特定字段标签、复杂嵌套）产生不完美的输出，需要手动清理
- **生态系统碎片化** —— 没有统一维护的替代方案；团队要么修补工具，要么维护分支

**Spec Forge 补充 go-zero**，提供更稳健的生成器，直接生成干净的 OpenAPI 3.x 规范，处理官方工具遗漏的边界情况。

### LLM 增强：代码优先，AI 加持

原始生成的规范准确但稀疏 —— 有结构，没有故事。字段有类型但没有描述，API 有路径但没有上下文。

**错误的方式：** 让 LLM 从头开始写整个规范。它会产生幻觉类型、虚构字段，产生的规范在代码变更的那一刻就与现实脱节。

**Spec Forge 的方式：**

1. **解析实际代码** —— AST 分析保证规范结构与你真实的类型匹配
2. **生成基础规范** —— 准确的路径、结构、参数，零幻觉
3. **AI 增强** —— LLM 基于真实结构添加人类可读的描述

```yaml
# 增强前
properties:
  user_id:
    type: string
    format: uuid

# 增强后
properties:
  user_id:
    type: string
    format: uuid
    description: "用户账户的唯一标识符，使用 UUID v4 生成"
```

LLM **从不** 虚构类型或改变结构 —— 它只为我们已经验证存在的内容添加描述。这保持了规范的准确性，同时使其对人类友好且支持 AI 代理。

### 其他框架

**Spring Boot** —— springdoc 工作良好，但需要手动配置依赖。Spec Forge 自动修补你的 `pom.xml` 或 `build.gradle` 并运行生成流水线。

**gRPC / Protobuf** —— 工具生态一片混乱：`protoc-gen-openapi` 无人维护，`buf` 缺少官方 OpenAPI 文档。Spec Forge 封装 `protoc-gen-connect-openapi` —— 一个维护良好、原生 OpenAPI 3.x 的解决方案。

**Hertz / Kitex (CloudWeGo)** —— 官方 OpenAPI 文档已过时。Spec Forge 将把 `hertz-contrib/swagger-generate` 中的可用工具封装成一条命令（即将推出）。

**为什么框架已有文档工具还需要 Spec Forge？** 每个框架的生态需要不同的设置、依赖和 CI 配置。Spec Forge 通过提供统一的生成界面来降低接入成本 —— 一条命令、一个配置文件，跨所有服务输出一致的规范。这使得集中式 CI/CD 流水线能够自动生成、验证、增强和发布 API 文档，无论各团队选择了哪种框架。

---

## 工作原理

```
源代码 → 检测 → 修补 → 生成 → 验证 → 增强 → 发布
```

1. **检测** —— 识别项目类型（Spring Boot、Gin、go-zero、gRPC）
2. **修补** —— 如缺少则添加必需的依赖/插件
3. **生成** —— 运行框架特定的生成
4. **验证** —— 验证 OpenAPI 规范合规性
5. **增强** —— 使用 LLM 添加描述（可选）
6. **发布** —— 输出到文件或发布到平台

---

## 功能特性

- 🔍 **自动检测** —— Spring Boot、Gin、go-zero、gRPC
- 🔧 **自动修补** —— 自动添加依赖/插件
- 🤖 **AI 增强** —— LLM 生成的描述
- 🌐 **多提供商** —— OpenAI、Anthropic、Ollama、自定义
- ✍️ **Gin 零注释** —— 纯 AST 分析

---

## 支持的框架

| 框架                                     | 状态     | 指南                |
|----------------------------------------|--------|-------------------|
| [Spring Boot](./docs/spring-boot.md)   | ✅ 可用   | Java/Maven/Gradle |
| [Gin](./docs/gin.md)                   | ✅ 可用   | Go，零注释            |
| [go-zero](./docs/go-zero.md)           | ✅ 可用   | Go                |
| [gRPC (protoc)](./docs/grpc-protoc.md) | ✅ 可用   | Protobuf          |
| [Hertz](./docs/hertz.md)               | 🚧 计划中 | Go                |
| [Kitex](./docs/kitex.md)               | 🚧 计划中 | Go                |

---

## 配置

在当前工作目录创建 `.spec-forge.yaml`：

```yaml
enrich:
  enabled: true
  provider: openai
  model: gpt-4o
  language: zh

output:
  dir: ./openapi
  format: yaml
```

**注意：**
- AI 增强需要通过环境变量提供 API 密钥：
  - OpenAI: `OPENAI_API_KEY`
  - Anthropic: `ANTHROPIC_API_KEY`
  - 自定义提供商: `LLM_API_KEY`
- 配置文件只在当前工作目录读取，不会自动读取项目目录。如果在其他目录运行 `spec-forge generate ./path/to/project`，请确保配置文件在当前目录。

查看 [.spec-forge.example.yaml](.spec-forge.example.yaml) 了解所有选项。

---

## 文档

- [Quick Start](./docs/quick-start.md) —— 安装和第一步（英文）
- [Configuration](./docs/configuration.md) —— 所有配置选项（英文）
- [AI Enrichment](./docs/ai-enrichment.md) —— LLM 提供商和提示词（英文）
- [Publishing](./docs/publishing.md) —— ReadMe.com 及其他（英文）

---

## 许可证

MIT
