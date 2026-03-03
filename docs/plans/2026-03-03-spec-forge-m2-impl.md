# M2: Spring Detection and Patch Implementation Summary

> **日期：** 2026-03-03
> **状态：** 已完成
> **里程碑：** M2

---

## 概述

本文档记录 M2 里程碑的实际实现细节，与设计文档 `2026-03-03-spec-forge-m2-design.md` 配套阅读。

---

## 1. 实现的文件结构

```
internal/
└── extractor/
    ├── types.go                    # 核心类型定义
    ├── types_test.go               # 类型测试
    └── spring/
        ├── springdoc.go            # 共享常量
        ├── detector.go             # 项目检测
        ├── detector_test.go        # 检测测试
        ├── patcher.go              # Patch 逻辑
        ├── patcher_test.go         # Patch 测试
        ├── maven.go                # Maven 解析器
        ├── maven_test.go           # Maven 测试
        ├── gradle.go               # Gradle 解析器
        └── gradle_test.go          # Gradle 测试

cmd/
├── spring.go                       # spring detect/patch 命令
└── generate.go                     # generate 命令（含 --keep-patched）

integration-tests/
├── maven-springboot-openapi-demo/  # 单模块 Maven 示例
├── gradle-springboot-openapi-demo/ # 单模块 Gradle 示例
├── maven-multi-module-demo/        # 多模块 Maven 示例
└── gradle-multi-module-demo/       # 多模块 Gradle 示例
```

---

## 2. 核心类型实现

### types.go

```go
package extractor

// BuildTool 表示构建工具类型
type BuildTool string

const (
    BuildToolMaven  BuildTool = "maven"
    BuildToolGradle BuildTool = "gradle"
)

// 默认版本常量
const (
    DefaultSpringdocVersion      = "3.0.2"
    DefaultSpringdocMavenPlugin  = "1.5"
    DefaultSpringdocGradlePlugin = "1.9.0"
)

// ProjectInfo 包含检测到的项目信息
type ProjectInfo struct {
    BuildTool          BuildTool
    BuildFilePath      string
    SpringBootVersion  string
    HasSpringdocDeps   bool
    HasSpringdocPlugin bool
    SpringdocVersion   string

    // 多模块项目支持
    IsMultiModule    bool
    Modules          []string
    MainModule       string
    MainModulePath   string
}

// PatchOptions 配置 patch 行为
type PatchOptions struct {
    DryRun              bool
    Force               bool
    SpringdocVersion    string
    MavenPluginVersion  string
    GradlePluginVersion string
    KeepPatched         bool  // 新增：控制是否保留修改
}
```

### spring/springdoc.go

```go
package spring

// Springdoc 常量（Maven/Gradle 通用）
const (
    SpringdocGroupID             = "org.springdoc"
    SpringdocWebMVCArtifactID    = "springdoc-openapi-starter-webmvc-ui"
    SpringdocMavenPluginArtifact = "springdoc-openapi-maven-plugin"
    SpringdocGradlePluginID      = "org.springdoc.openapi-gradle-plugin"
)

// Spring Boot 常量
const (
    SpringBootParentGroupID    = "org.springframework.boot"
    SpringBootParentArtifactID = "spring-boot-starter-parent"
)
```

---

## 3. 核心组件实现

### spring/detector.go

**职责：** 项目检测入口，组合 Maven/Gradle 解析器

```go
type Detector struct {
    mavenParser  *MavenParser
    gradleParser *GradleParser
}

func (d *Detector) Detect(projectPath string) (*extractor.ProjectInfo, error)
func (d *Detector) detectMavenProject(projectPath, pomPath string) (*extractor.ProjectInfo, error)
func (d *Detector) detectGradleProject(projectPath, gradlePath string) (*extractor.ProjectInfo, error)
```

**检测流程：**
1. 检查 `pom.xml` 或 `build.gradle` 是否存在
2. 解析构建文件
3. 检测多模块项目（Maven: `<modules>`, Gradle: `settings.gradle`）
4. 如果是多模块，查找主模块（有 Spring Boot 插件的模块）
5. 提取 Spring Boot 版本、springdoc 依赖/插件状态

### spring/patcher.go

**职责：** Patch 逻辑，包含文件恢复机制

```go
type PatchResult struct {
    DependencyAdded bool
    PluginAdded     bool
    BuildFilePath   string
    OriginalContent string  // 原始文件内容，用于恢复
}

type Patcher struct {
    detector     *Detector
    mavenParser  *MavenParser
    gradleParser *GradleParser
}

func (p *Patcher) Patch(projectPath string, opts *extractor.PatchOptions) (*PatchResult, error)
func (p *Patcher) Restore(buildFilePath, originalContent string) error
```

**Patch 流程：**
1. 调用 `detector.Detect()` 获取项目信息
2. 确定目标构建文件（多模块用主模块路径）
3. **保存原始文件内容到 `OriginalContent`**
4. 根据构建工具调用相应的 patch 方法
5. 返回 `PatchResult`

**恢复机制：**
```go
// generate 命令中的使用
if !generateKeepPatched && result.OriginalContent != "" {
    defer func() {
        patcher.Restore(result.BuildFilePath, result.OriginalContent)
    }()
}
```

### spring/maven.go

**职责：** Maven pom.xml 解析、修改、多模块支持

```go
type MavenParser struct{}

// 解析方法
func (p *MavenParser) Parse(pomPath string) (*gopom.Project, error)
func (p *MavenParser) GetSpringBootVersion(pom *gopom.Project) string
func (p *MavenParser) HasSpringdocDependency(pom *gopom.Project) bool
func (p *MavenParser) GetSpringdocVersion(pom *gopom.Project) string
func (p *MavenParser) HasSpringdocPlugin(pom *gopom.Project) bool

// 多模块支持
func (p *MavenParser) GetModules(pom *gopom.Project) []string
func (p *MavenParser) HasSpringBootPlugin(pom *gopom.Project) bool
func (p *MavenParser) FindMainModule(projectPath string, modules []string) (string, string)

// Patch 方法
func (p *MavenParser) AddDependency(pom *gopom.Project, groupID, artifactID, version string)
func (p *MavenParser) AddPlugin(pom *gopom.Project, groupID, artifactID, version string)
func (p *MavenParser) HasDependency(pom *gopom.Project) bool
func (p *MavenParser) HasPlugin(pom *gopom.Project) bool
func (p *MavenParser) MarshalPom(pom *gopom.Project) ([]byte, error)
```

### spring/gradle.go

**职责：** Gradle build.gradle 解析、文本修改、多模块支持

```go
type GradleParser struct{}

// 解析方法
func (p *GradleParser) Parse(gradlePath string) (*model.Project, error)
func (p *GradleParser) ParseString(content string) (*model.Project, error)
func (p *GradleParser) GetSpringBootVersion(project *model.Project) string
func (p *GradleParser) HasSpringdocDependency(project *model.Project) bool
func (p *GradleParser) GetSpringdocVersion(project *model.Project) string
func (p *GradleParser) HasSpringdocPlugin(project *model.Project) bool

// 多模块支持
func (p *GradleParser) ParseModules(settingsPath string) []string
func (p *GradleParser) HasSpringBootPlugin(project *model.Project) bool
func (p *GradleParser) FindMainModule(projectPath string, modules []string) (string, string)

// 文本修改方法（保留原始格式）
func (p *GradleParser) AddDependencyText(content, version string) string
func (p *GradleParser) AddPluginText(content, version string) string
```

**注意：** Gradle 使用文本操作而非 AST 修改，以保留原始格式。

---

## 4. 关键设计决策

### 4.1 gopom 格式问题解决方案

**问题：** `vifraa/gopom` 序列化 XML 时会丢失注释、格式化等原始内容。

**解决方案：**
1. 在 `PatchResult` 中保存 `OriginalContent`
2. 提供 `Restore()` 方法恢复原始文件
3. `generate` 命令默认恢复，`spring patch` 默认保留

### 4.2 Gradle 文本修改

**决策：** 使用文本操作而非 AST 修改

**原因：**
- gradle-parser 不支持修改
- 文本操作可以保留原始格式
- 通过检测内容变化避免重复添加

### 4.3 多模块项目处理

**决策：** Patch 主模块而非父 POM

**逻辑：**
1. 检测模块列表
2. 查找有 Spring Boot 插件的模块
3. Patch 该模块的构建文件

**原因：**
- Spring Boot 应用通常在子模块中
- 父 POM 用于依赖管理，不应添加应用依赖

---

## 5. 测试覆盖

### 单元测试

| 测试文件 | 覆盖内容 |
|----------|----------|
| `types_test.go` | 类型常量验证 |
| `detector_test.go` | 单模块/多模块检测 |
| `maven_test.go` | Maven 解析、Patch、多模块 |
| `gradle_test.go` | Gradle 解析、文本修改、多模块 |
| `patcher_test.go` | Patch 逻辑、边缘情况 |

### 边缘情况测试

- pom 只有 pluginManagement
- pom 没有 dependencies 节点
- pom 没有 build 节点
- build.gradle 没有 plugins block
- build.gradle 没有 dependencies block
- Force 选项
- DryRun 模式
- OriginalContent 保存和 Restore

### 集成测试项目

| 项目 | 类型 | 用途 |
|------|------|------|
| `maven-springboot-openapi-demo` | 单模块 Maven | 基本功能验证 |
| `gradle-springboot-openapi-demo` | 单模块 Gradle | 基本功能验证 |
| `maven-multi-module-demo` | 多模块 Maven | 多模块支持验证 |
| `gradle-multi-module-demo` | 多模块 Gradle | 多模块支持验证 |

---

## 6. CLI 命令实现

### spring detect

```go
// cmd/spring.go
var springCmd = &cobra.Command{
    Use:   "spring",
    Short: "Spring framework commands",
}

var springDetectCmd = &cobra.Command{
    Use:   "detect [path]",
    Short: "Detect Spring project information",
    RunE:  runSpringDetect,
}
```

**输出包含：**
- Build Tool 和 Build File
- Spring Boot 版本
- springdoc 依赖/插件状态
- 多模块信息（如果是多模块项目）

### spring patch

```go
var springPatchCmd = &cobra.Command{
    Use:   "patch [path]",
    Short: "Add springdoc dependencies to Spring project",
    RunE:  runSpringPatch,
}
```

**参数：**
- `--dry-run`: 只打印修改
- `--force`: 强制覆盖

### generate

```go
// cmd/generate.go
var generateCmd = &cobra.Command{
    Use:   "generate [path]",
    Short: "Generate OpenAPI specification",
    RunE:  runGenerate,
}
```

**参数：**
- `--keep-patched`: 保留修改后的构建文件（默认恢复）

---

## 7. 依赖

```go
// go.mod
require (
    github.com/vifraa/gopom v0.5.0
    github.com/scagogogo/gradle-parser v1.0.2
)
```

---

## 8. 后续工作

M2 已完成，M3 将实现：
- Generator: 调用 Maven/Gradle 插件生成 OpenAPI Spec
- Validator: 验证生成的 OpenAPI Spec
