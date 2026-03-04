# M3: Generator 和 Validator 实现文档

> **日期：** 2026-03-04
> **状态：** 已实现
> **里程碑：** M3

---

## 概述

M3 实现 OpenAPI Spec 的生成和验证功能。

**核心功能：**
- `spec-forge generate` - 完整流程: Detect → Patch → **Generate** → **Validate** → Restore
- Generator 调用 Maven/Gradle springdoc 插件生成 OpenAPI Spec
- Validator 使用 kin-openapi 验证生成的 Spec

---

## 1. 目录结构

```
internal/
├── executor/
│   ├── executor.go           # 命令执行器
│   └── executor_test.go
│
├── extractor/
│   ├── types.go              # GenerateOptions, GenerateResult, ValidateResult
│   └── spring/
│       ├── detector.go       # 项目检测器
│       ├── patcher.go        # 构建文件补丁器
│       ├── maven.go          # Maven POM 解析和配置
│       ├── gradle.go         # Gradle 构建文件解析
│       ├── generator.go      # OpenAPI 生成器
│       └── *_test.go         # 各模块测试
│
└── validator/
    ├── validator.go          # 验证器入口
    ├── openapi.go            # kin-openapi 实现
    └── validator_test.go
```

---

## 2. 核心类型

### types.go 新增类型

```go
// GenerateOptions 配置 OpenAPI spec 生成
type GenerateOptions struct {
    OutputDir  string        // 输出目录 (default: project target/build dir)
    OutputFile string        // 输出文件名 (default: "openapi")
    Format     string        // 输出格式: "json" or "yaml" (default: "json")
    Timeout    time.Duration // 命令执行超时 (default: 5 minutes)
    SkipTests  bool          // 跳过测试 (default: true)
}

// GenerateResult 生成结果
type GenerateResult struct {
    SpecFilePath string // 生成的 spec 文件绝对路径
    Format       string // 输出格式
}

// ValidateResult 验证结果
type ValidateResult struct {
    Valid  bool     // 是否有效
    Errors []string // 验证错误列表
}
```

---

## 3. 构建工具命令解析

### Wrapper 优先策略

Generator 在执行 Maven/Gradle 命令时，按以下优先级选择命令：

| 优先级 | 位置 | Maven 命令 | Gradle 命令 |
|--------|------|------------|-------------|
| 1 | 项目根目录 | `./mvnw` | `./gradlew` |
| 2 | 父目录（多模块）| `/path/to/mvnw` | `/path/to/gradlew` |
| 3 | 系统 PATH | `mvn` | `gradle` |

### 解析逻辑

```go
// resolveMavenCommand 解析 Maven 命令
// 优先级: 项目根目录 mvnw > 父目录 mvnw > 系统 mvn
func (g *Generator) resolveMavenCommand(workDir string) string {
    // 1. 检查当前目录的 mvnw
    mvnwPath := filepath.Join(workDir, "mvnw")
    if _, err := os.Stat(mvnwPath); err == nil {
        return "./mvnw"
    }

    // 2. 向上查找父目录的 mvnw (多模块项目)
    currentDir := workDir
    for {
        parentDir := filepath.Dir(currentDir)
        if parentDir == currentDir {
            break // 到达根目录
        }

        mvnwInParent := filepath.Join(parentDir, "mvnw")
        if _, err := os.Stat(mvnwInParent); err == nil {
            absPath, _ := filepath.Abs(mvnwInParent)
            return absPath
        }

        // 检查是否已离开项目（无 pom.xml）
        pomInParent := filepath.Join(parentDir, "pom.xml")
        if _, err := os.Stat(pomInParent); os.IsNotExist(err) {
            break
        }

        currentDir = parentDir
    }

    // 3. 回退到系统 Maven
    return "mvn"
}
```

**Gradle 的 `resolveGradleCommand` 逻辑相同。**

### 错误处理

当命令不存在时，提供清晰的安装提示：

```
Error: maven generation failed: command 'mvn' not found in PATH
Hint: Install Maven from https://maven.apache.org/install.html or use your package manager
```

---

## 4. 生成流程

> **⚠️ 重要约束：** 必须参考 springdoc 官方文档 https://springdoc.org/#plugins 确认正确的命令。

### Maven 生成

根据 springdoc 官方文档，Maven 插件使用 `verify` 阶段触发生成：

```go
func (g *Generator) generateMaven(ctx context.Context, workDir string, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
    mavenCmd := g.resolveMavenCommand(workDir)

    // 官方文档: mvn verify
    // 参考: https://springdoc.org/#plugins
    args := []string{
        "verify",
        "-DskipTests",
        "-Dspringdoc.outputFormat=" + opts.Format,
    }

    result, err := g.executor.Execute(ctx, &executor.ExecuteOptions{
        Command:    mavenCmd,
        Args:       args,
        WorkingDir: workDir,
        Timeout:    opts.Timeout,
    })
    // ...
}
```

### Gradle 生成

根据 springdoc 官方文档，Gradle 插件使用 `generateOpenApiDocs` 任务：

```go
func (g *Generator) generateGradle(ctx context.Context, workDir string, opts *extractor.GenerateOptions) (*extractor.GenerateResult, error) {
    gradleCmd := g.resolveGradleCommand(workDir)

    // 官方文档: gradle clean generateOpenApiDocs
    // 参考: https://springdoc.org/#plugins
    args := []string{
        "generateOpenApiDocs",
        "-x", "test",
    }

    result, err := g.executor.Execute(ctx, &executor.ExecuteOptions{
        Command:    gradleCmd,
        Args:       args,
        WorkingDir: workDir,
        Timeout:    opts.Timeout,
    })
    // ...
}
```

### 输出文件查找

生成后自动查找输出文件：

| 构建工具 | 输出目录 | 文件名 |
|----------|----------|--------|
| Maven | `target/` | `openapi.json` 或 `openapi.yaml` |
| Gradle | `build/` | `openapi.json` 或 `openapi.yaml` |

---

## 5. 验证流程

使用 `kin-openapi` 库验证生成的 Spec：

```go
func (l *openapiLoader) LoadAndValidate(ctx context.Context, specPath string) (*extractor.ValidateResult, error) {
    loader := openapi3.NewLoader()
    loader.IsExternalRefsAllowed = true

    doc, err := loader.LoadFromFile(specPath)
    if err != nil {
        return &extractor.ValidateResult{
            Valid:  false,
            Errors: []string{fmt.Sprintf("failed to parse: %v", err)},
        }, nil
    }

    if err := doc.Validate(ctx); err != nil {
        return &extractor.ValidateResult{
            Valid:  false,
            Errors: []string{formatValidationError(err)},
        }, nil
    }

    return &extractor.ValidateResult{Valid: true}, nil
}
```

---

## 6. CLI 命令

### generate 命令

```bash
spec-forge generate [path] [flags]

Flags:
  --keep-patched     保留打补丁的文件（默认恢复）
  --skip-validate    跳过验证
  --timeout          命令超时时间（默认 5m）
```

### 完整流程

```
┌─────────┐    ┌─────────┐    ┌───────────┐    ┌───────────┐    ┌─────────┐
│ Detect  │───▶│  Patch  │───▶│ Generate  │───▶│ Validate  │───▶│ Restore │
└─────────┘    └─────────┘    └───────────┘    └───────────┘    └─────────┘
     │              │               │                │              │
     │              │               │                │              │
     ▼              ▼               ▼                ▼              ▼
  识别项目     添加 springdoc    调用 mvn/gradle   验证 OpenAPI   恢复原文件
  构建工具    依赖和插件        生成 spec         spec 有效性
```

---

## 7. 依赖

```go
// go.mod
require (
    github.com/getkin/kin-openapi v0.133.0
)
```

---

## 8. 测试覆盖

| 测试文件 | 覆盖内容 |
|----------|----------|
| `executor_test.go` | 命令执行、超时、错误处理 |
| `generator_test.go` | Maven/Gradle 生成、Wrapper 解析、文件查找、正确命令验证 |
| `maven_test.go` | POM 解析、spring-boot 插件配置检测和添加 |
| `patcher_test.go` | 单模块/多模块补丁、Restore 机制、DryRun 模式 |
| `validator_test.go` | 有效/无效 spec、格式错误、引用验证 |

---

## 9. 关键设计决策

### 9.1 参考官方文档（强制约束）

> **⚠️ 极其重要：** 实现 springdoc 相关功能时，**必须**参考官方文档 https://springdoc.org/#plugins

**决策：** 所有 springdoc 命令必须与官方文档保持一致

**原因：**
- springdoc 插件的工作方式与直觉不同
- 直接调用 `springdoc:generate` 无法正常工作，需要通过 `verify` 阶段触发
- 官方文档明确指定了正确的命令：
  - **Maven**: `mvn verify`
  - **Gradle**: `gradle clean generateOpenApiDocs`
- 错误的命令会导致 "Connection refused" 或 CLI 阻塞等问题

**教训：**
- 不要猜测构建工具的行为，始终参考官方文档
- springdoc 插件需要在运行时访问 `/v3/api-docs` 端点，因此需要启动 Spring Boot 应用

### 9.2 多模块项目 spring-boot 配置

**决策：** 多模块 Maven 项目需要配置 `spring-boot-maven-plugin` 的 start/stop goals

**原因：**
- springdoc 插件需要 Spring Boot 应用运行才能生成 OpenAPI spec
- 多模块项目的子模块可能没有配置 start/stop goals
- Patcher 自动检测并添加配置：
  ```xml
  <execution>
      <id>start-stop-for-openapi</id>
      <goals>
          <goal>start</goal>
          <goal>stop</goal>
      </goals>
  </execution>
  ```

**实现位置：** `internal/extractor/spring/maven.go` 中的 `ConfigureSpringBootPlugin` 函数

### 9.3 Wrapper 优先

**决策：** 优先使用项目自带的 Maven/Gradle Wrapper

**原因：**
- Wrapper 确保使用项目预期的版本
- 不依赖用户本地安装的工具
- 多模块项目的 wrapper 通常在根目录

### 9.4 错误提示

**决策：** 当命令不存在时提供安装提示

**原因：**
- 用户可能不知道需要安装什么
- 提供官方文档链接帮助快速解决

### 9.5 验证可选

**决策：** 提供 `--skip-validate` 选项

**原因：**
- 某些项目可能有非标准的 OpenAPI 扩展
- 允许用户在验证失败时仍然获取 spec 文件

---

## 10. 后续工作

M3 已完成，M4 将实现：
- Enrich: 使用 LLM 增强 OpenAPI Spec 的描述信息
