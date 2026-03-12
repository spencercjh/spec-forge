# Integration Tests 改进计划

## 调研总结

### 当前测试结构

| 测试文件 | 测试类型 | 问题 |
|---------|---------|------|
| `spring_maven_test.go` | 集成测试（非E2E） | 直接调用 `internal/extractor/spring` 内部包 |
| `spring_gradle_test.go` | 集成测试（非E2E） | 直接调用 `internal/extractor/spring` 内部包 |
| `gozero_test.go` | 集成测试（非E2E） | 直接调用 `internal/extractor/gozero` 内部包 |
| `gin_demo_test.go` | 集成测试（非E2E） | 直接调用 `internal/extractor/gin` 内部包 |
| `grpc_protoc_test.go` | 集成测试（非E2E） | 直接调用 `internal/extractor/grpcprotoc` 内部包 |
| `error_test.go` | 单元测试 | 只测试 `executor.CommandNotFoundError`，过于简单 |
| `mock_provider_test.go` | 辅助文件 | 不是测试，只是 mock provider 实现 |

### 核心问题：不是真正的 E2E 测试

真正的 E2E 测试应该通过 **Cobra CLI** 层执行命令，验证完整的用户工作流程。当前测试直接调用内部包函数，属于**集成测试**，不是 E2E 测试。

**当前测试方式（非E2E）：**
```go
// 直接调用内部包
detector := spring.NewDetector()
info, err := detector.Detect(projectPath)
gen := spring.NewGenerator()
result, err := gen.Generate(ctx, projectPath, info, opts)
```

**Cobra 推荐 E2E 测试方式：**
```go
// 通过 ExecuteContext 测试完整 CLI 流程
rootCmd := cmd.NewRootCommand()
rootCmd.SetArgs([]string{"generate", "./demo"})
rootCmd.SetOut(&stdout)
rootCmd.SetErr(&stderr)
err := rootCmd.ExecuteContext(ctx)
```

### 缺失的测试覆盖

| 功能 | 当前状态 | 说明 |
|------|---------|------|
| CLI 参数解析 | ❌ 未测试 | 未测试各种 flag 组合 |
| 配置文件加载 | ❌ 未测试 | 未测试 `.spec-forge.yaml` 解析 |
| 错误输出格式 | ❌ 未测试 | 未验证 CLI 错误信息格式 |
| `enrich` 命令 | ⚠️ 部分测试 | 使用 mock provider，未测试完整 CLI 流程 |
| `publish` 命令 | ❌ 未测试 | 完全未测试 |
| 多模块项目 | ⚠️ 有 demo 无测试 | Maven/Gradle 多模块只有 demo 项目，没有自动化测试 |

---

## 改进计划

### Phase 1: 重构现有测试分类

将现有测试从 `e2e` 改为 `integration`，保留其价值但不混淆概念。

**操作：**
1. 重命名测试文件：`spring_maven_test.go` → `integration_spring_maven_test.go`
2. 修改 build tag：`//go:build e2e` → `//go:build integration`
3. 更新 README.md 文档说明测试分类

**文件列表：**
- `spring_maven_test.go` → `integration_spring_maven_test.go`
- `spring_gradle_test.go` → `integration_spring_gradle_test.go`
- `gozero_test.go` → `integration_gozero_test.go`
- `gin_demo_test.go` → `integration_gin_test.go`
- `grpc_protoc_test.go` → `integration_grpc_protoc_test.go`

### Phase 2: 使用 Cobra 标准方案创建 E2E 测试

Cobra 的推荐测试方式是使用 `SetArgs()` + `ExecuteContext()` + `SetOut/SetErr` 捕获输出。

**需要在 `cmd/` 包暴露的接口：**

```go
// cmd/root.go - 添加 NewRootCommand

// NewRootCommand creates a fresh root command instance for testing.
// This avoids global state pollution between tests.
func NewRootCommand() *cobra.Command {
    c := &cobra.Command{
        Use:   "spec-forge",
        Short: "Generate OpenAPI specifications from source code",
        Long: `Spec Forge is a CLI tool that automatically generates OpenAPI specifications...`,
        Version: "0.1.0",
    }

    // 添加所有子命令
    c.AddCommand(newGenerateCmd())
    c.AddCommand(newEnrichCmd())
    c.AddCommand(newPublishCmd())
    // ...

    return c
}

// Execute 使用全局 rootCmd（保持兼容性）
func Execute() {
    err := rootCmd.Execute()
    // ...
}
```

**测试代码示例：**

```go
//go:build e2e

package e2e_test

import (
    "bytes"
    "testing"

    "github.com/spencercjh/spec-forge/cmd"
)

func TestE2E_Generate_MavenSpringBoot(t *testing.T) {
    // 创建新的命令实例
    rootCmd := cmd.NewRootCommand()

    var stdout, stderr bytes.Buffer
    rootCmd.SetOut(&stdout)
    rootCmd.SetErr(&stderr)
    rootCmd.SetArgs([]string{"generate", "./maven-springboot-openapi-demo"})

    err := rootCmd.ExecuteContext(t.Context())

    // 验证
    if err != nil {
        t.Fatalf("command failed: %v\nstderr: %s", err, stderr.String())
    }

    // 验证输出文件存在...
}
```

**需要添加的 E2E 测试：**

| 测试 | 命令 | 验证点 |
|-----|------|--------|
| `TestE2E_Generate_MavenSpringBoot` | `generate ./maven-springboot-openapi-demo` | 输出文件存在、格式正确、验证通过 |
| `TestE2E_Generate_GradleSpringBoot` | `generate ./gradle-springboot-openapi-demo` | 同上 |
| `TestE2E_Generate_Gin` | `generate ./gin-demo` | 同上 |
| `TestE2E_Generate_GoZero` | `generate ./gozero-demo` | 同上（如果 goctl 可用） |
| `TestE2E_Generate_GrpcProtoc` | `generate ./grpc-protoc-demo` | 同上（如果 protoc 可用） |
| `TestE2E_Generate_OutputFlags` | `generate -o yaml -d /tmp/output ./demo` | 输出目录和格式正确 |
| `TestE2E_Generate_InvalidProject` | `generate /nonexistent` | 错误码非零、错误信息合理 |
| `TestE2E_Enrich_Help` | `enrich --help` | 帮助信息正确 |
| `TestE2E_Publish_Help` | `publish --help` | 帮助信息正确 |
| `TestE2E_Version` | `version` | 版本信息正确 |

### Phase 3: 增强 Makefile 目标

```makefile
# 运行集成测试（测试内部包）
test-integration:
    go test -tags=integration ./integration-tests/...

# 运行真正的 E2E 测试（通过 Cobra Execute）
test-e2e:
    go test -tags=e2e ./integration-tests/...
```

### Phase 4: 缩减 internal 包暴露的 API 表面

当前 `internal/extractor/*` 下有很多不必要暴露的方法，应该改为内部方法（小写开头）。

**典型问题示例：**

```go
// 当前：不必要地暴露
func NewDetector() *Detector { ... }
func (d *Detector) Detect(path string) (*Info, error) { ... }
func (d *Detector) findPomFiles(dir string) ([]string, error) { ... } // 应该内部

// spring/maven.go
func ParsePOM(path string) (*POM, error) { ... }  // 应该内部
func ConfigureSpringBootPlugin(pom *POM) error { ... }  // 应该内部
```

**目标：**
- 只暴露 `Extractor` 接口需要的方法
- 包内辅助函数改为小写（如 `parsePOM`, `configureSpringBootPlugin`）
- 减少测试对内部实现的依赖，更多通过 CLI/接口测试

**需要调整的包：**
- `internal/extractor/spring/` - detector, generator, maven, gradle
- `internal/extractor/gozero/` - detector, generator, patcher
- `internal/extractor/gin/` - detector, generator, ast_parser
- `internal/extractor/grpcprotoc/` - detector, generator, patcher

### Phase 5: 为缺失场景补充测试

**多模块项目测试：**
- 为 `maven-multi-module-demo` 和 `gradle-multi-module-demo` 添加集成测试

**Publish 命令测试：**
- 使用 mock 的 ReadMe API 测试 publish 流程
- 验证配置文件解析

---

## 实施优先级

| 优先级 | 任务 | 预估工作量 |
|-------|------|-----------|
| P0 | Phase 1: 重命名现有测试，区分 integration 和 e2e | 小 |
| P1 | Phase 2: 创建基础 E2E 测试框架 + 2-3 个核心 E2E 测试 | 中 |
| P2 | Phase 3: 增强 Makefile 和测试辅助库 | 小 |
| P3 | Phase 4: 缩减 internal 包 API 表面 | 中 |
| P4 | Phase 5: 补充多模块项目和 publish 测试 | 中 |

---

## 相关文件

- `integration-tests/*.go` - 所有测试文件
- `integration-tests/README.md` - 测试文档
- `Makefile` - 构建脚本
- `cmd/generate.go`, `cmd/enrich.go`, `cmd/publish.go` - CLI 命令实现
