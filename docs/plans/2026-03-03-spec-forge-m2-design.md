# M2: Spring 检测和 Patch 设计文档

> **日期：** 2026-03-03
> **状态：** 已实现
> **里程碑：** M2

---

## 概述

M2 实现 Spring 项目的检测和 Patch 功能，为后续的 OpenAPI 提取做准备。

**核心功能：**
- `spec-forge spring detect` - 检测 Spring 项目信息
- `spec-forge spring patch` - 添加 springdoc 依赖和插件
- `spec-forge generate` - 一键生成（含自动 patch 和恢复）

---

## 1. 目录结构

```
internal/
└── extractor/
    ├── types.go                # 核心类型定义
    └── spring/
        ├── springdoc.go        # 共享常量 (Maven/Gradle 通用)
        ├── detector.go         # 项目检测入口
        ├── patcher.go          # Patch 逻辑
        ├── maven.go            # Maven 解析和修改 (使用 vifraa/gopom)
        └── gradle.go           # Gradle 解析和修改 (使用 scagogogo/gradle-parser)
```

**文件职责：**
| 文件 | 职责 |
|------|------|
| `springdoc.go` | 共享常量定义 (GroupID, ArtifactID 等) |
| `detector.go` | 项目检测入口，组合 maven/gradle parser |
| `patcher.go` | Patch 逻辑，包含 OriginalContent 保存和 Restore 方法 |
| `maven.go` | Maven pom.xml 解析、修改、多模块支持 |
| `gradle.go` | Gradle build.gradle 解析、文本修改、多模块支持 |

---

## 2. 数据结构

```go
// BuildTool 表示构建工具类型
type BuildTool string

const (
    BuildToolMaven  BuildTool = "maven"
    BuildToolGradle BuildTool = "gradle"
)

// ProjectInfo 包含检测到的项目信息
type ProjectInfo struct {
    BuildTool          BuildTool  // Maven or Gradle
    BuildFilePath      string     // pom.xml or build.gradle 路径
    SpringBootVersion  string     // Spring Boot 版本
    HasSpringdocDeps   bool       // 是否已有 springdoc 依赖
    HasSpringdocPlugin bool       // 是否已有 springdoc 插件
    SpringdocVersion   string     // 现有的 springdoc 版本（如果有）

    // 多模块项目支持
    IsMultiModule    bool     // 是否为多模块项目
    Modules          []string // 模块列表
    MainModule       string   // 主模块名称（包含 Spring Boot 插件的模块）
    MainModulePath   string   // 主模块的构建文件路径
}

// PatchOptions 配置 patch 行为
type PatchOptions struct {
    DryRun              bool   // 只打印修改，不写入
    Force               bool   // 强制覆盖已有依赖
    SpringdocVersion    string // springdoc 版本（默认内置）
    MavenPluginVersion  string // Maven 插件版本（默认内置）
    GradlePluginVersion string // Gradle 插件版本（默认内置）
    KeepPatched         bool   // 是否保留修改后的文件（默认 generate=false, spring patch=true）
}

// PatchResult 包含 patch 操作的结果
type PatchResult struct {
    DependencyAdded bool
    PluginAdded     bool
    BuildFilePath   string
    OriginalContent string // 原始文件内容，用于恢复
}
```

---

## 3. 默认配置

**内置默认值（约定优于配置）：**
```go
const (
    DefaultSpringdocVersion       = "3.0.2"
    DefaultSpringdocMavenPlugin   = "1.5"
    DefaultSpringdocGradlePlugin  = "1.9.0"
)
```

**可选配置覆盖（.spec-forge.yaml）：**
```yaml
spring:
  springdocVersion: "3.0.2"
  mavenPluginVersion: "1.5"
  gradlePluginVersion: "1.9.0"
```

---

## 4. spring detect 命令

### 功能

检测 Spring 项目的基本信息。

### 输出格式（人类可读）

```
Spring Project Detection Results
================================
Build Tool:           Maven
Build File:           pom.xml
Spring Boot:          4.0.3
springdoc Dependency: ✅ Present (3.0.2)
springdoc Plugin:     ✅ Configured
```

**多模块项目输出：**
```
Multi-Module:         ✅ Yes
Modules:              [shared-lib user-service]
Main Module:          user-service
Main Module Path:     /path/to/user-service/pom.xml
```

如果缺少依赖或插件：
```
springdoc Dependency: ❌ Not found
springdoc Plugin:     ❌ Not configured
```

### 检测逻辑

1. 检查项目根目录是否有 `pom.xml` 或 `build.gradle`
2. 解析构建文件
3. **检查多模块项目：**
   - Maven: 解析 `<modules>` 元素
   - Gradle: 解析 `settings.gradle` 的 `include` 语句
4. **如果是多模块项目：**
   - 查找包含 Spring Boot 插件的主模块
   - 使用主模块的构建文件作为 patch 目标
5. 提取 Spring Boot 版本（从 parent 或 dependencyManagement）
6. 检查 springdoc 依赖是否存在
7. 检查 springdoc 插件是否配置
8. 输出人类可读的结果

---

## 5. spring patch 命令

### 功能

为 Spring 项目添加 springdoc 依赖和插件配置。

### 依赖 vs 插件的作用

| 组件 | 作用 |
|------|------|
| **依赖** (`springdoc-openapi-starter-webmvc-ui`) | 提供 Swagger UI 网页和 `/v3/api-docs` 端点 |
| **插件** (`springdoc-openapi-maven-plugin`) | 在构建时抓取 API 文档并生成 openapi.json/yaml |

### Maven 项目添加内容

**依赖（添加到 `<dependencies>`）：**
```xml
<dependency>
    <groupId>org.springdoc</groupId>
    <artifactId>springdoc-openapi-starter-webmvc-ui</artifactId>
    <version>${springdoc.version}</version>
</dependency>
```

**插件（添加到 `<build><plugins>`）：**
```xml
<plugin>
    <groupId>org.springdoc</groupId>
    <artifactId>springdoc-openapi-maven-plugin</artifactId>
    <version>${springdoc.plugin.version}</version>
    <executions>
        <execution>
            <goals>
                <goal>generate</goal>
            </goals>
        </execution>
    </executions>
</plugin>
```

### Gradle 项目添加内容

**插件（添加到 `plugins` block）：**
```groovy
id 'org.springdoc.openapi-gradle-plugin' version "${springdocPluginVersion}"
```

**依赖（添加到 `dependencies` block）：**
```groovy
implementation 'org.springdoc:springdoc-openapi-starter-webmvc-ui:${springdocVersion}'
```

### Patch 逻辑

```
1. 运行 detect 获取项目信息
2. 如果已有依赖和插件：
   - 如果 --force，继续
   - 否则跳过，提示用户
3. 确定目标构建文件：
   - 单模块：使用根目录的构建文件
   - 多模块：使用主模块的构建文件
4. 保存原始文件内容到 OriginalContent
5. 根据构建工具类型修改文件
6. 如果 --dry-run，只打印修改，不写入文件
7. 返回 PatchResult（包含 OriginalContent）
```

### 文件恢复机制

由于 `gopom` 序列化会破坏原始 pom.xml 格式（注释、格式化等），实现了文件恢复机制：

```go
// Patcher 提供 Restore 方法
func (p *Patcher) Restore(buildFilePath, originalContent string) error

// generate 命令默认恢复原始文件
defer patcher.Restore(result.BuildFilePath, result.OriginalContent)
```

**命令行为差异：**
| 命令 | 默认行为 |
|------|----------|
| `generate` | 自动恢复原始文件 (`--keep-patched` 可保留修改) |
| `spring patch` | 保留修改 (`KeepPatched=true`) |

---

## 6. 多模块项目支持

### 检测逻辑

**Maven 多模块：**
1. 解析父 POM 的 `<modules>` 元素获取模块列表
2. 遍历模块，检查哪个模块有 `spring-boot-maven-plugin`
3. 该模块即为主模块

**Gradle 多模块：**
1. 解析 `settings.gradle` 的 `include` 语句获取模块列表
2. 遍历模块的 `build.gradle`，检查哪个有 `org.springframework.boot` 插件
3. 该模块即为主模块

### Patch 目标

| 项目类型 | Patch 目标 |
|----------|------------|
| 单模块 Maven | 根目录 `pom.xml` |
| 多模块 Maven | 主模块的 `pom.xml` |
| 单模块 Gradle | 根目录 `build.gradle` |
| 多模块 Gradle | 主模块的 `build.gradle` |

---

## 7. 依赖库

| 功能 | 库 | 用途 |
|------|-----|------|
| Maven 解析 | `github.com/vifraa/gopom` | 解析和修改 pom.xml |
| Gradle 解析 | `github.com/scagogogo/gradle-parser` | 解析 build.gradle |

**注意：** gopom 序列化会破坏原始格式，因此使用保存/恢复机制。

---

## 8. 错误处理

| 错误场景 | 处理方式 |
|----------|----------|
| 找不到构建文件 | 返回错误，提示用户检查目录 |
| 构建文件格式错误 | 返回错误，提示解析失败 |
| 无法确定 Spring Boot 版本 | 警告，但继续执行 |
| 写入文件失败 | 返回错误，保留原文件 |
| 多模块项目找不到主模块 | 使用根目录构建文件 |

---

## 9. 测试策略

### 单元测试

- `types_test.go` - 测试类型常量
- `detector_test.go` - 测试 detect 逻辑（含多模块）
- `maven_test.go` - 测试 Maven 解析和修改（含多模块）
- `gradle_test.go` - 测试 Gradle 解析和修改（含多模块）
- `patcher_test.go` - 测试 patch 逻辑（含边缘情况）

### 边缘情况测试

- pom.xml 只有 pluginManagement
- pom.xml 没有 dependencies 节点
- pom.xml 没有 build 节点
- pom.xml 有 dependencyManagement
- build.gradle 没有 plugins block
- build.gradle 没有 dependencies block
- Force 选项强制覆盖
- DryRun 模式不写入

### 集成测试

使用 `integration-tests/` 下的 demo 项目：
- `maven-springboot-openapi-demo/` - 单模块 Maven
- `gradle-springboot-openapi-demo/` - 单模块 Gradle
- `maven-multi-module-demo/` - 多模块 Maven
- `gradle-multi-module-demo/` - 多模块 Gradle

---

## 10. CLI 命令参数

### spring detect

| 参数 | 说明 |
|------|------|
| `[path]` | 项目路径，默认当前目录 |

### spring patch

| 参数 | 说明 |
|------|------|
| `[path]` | 项目路径，默认当前目录 |
| `--dry-run` | 只打印修改，不写入文件 |
| `--force` | 强制覆盖已有依赖和插件 |

### generate

| 参数 | 说明 |
|------|------|
| `[path]` | 项目路径，默认当前目录 |
| `--keep-patched` | 生成后保留修改的 pom/build.gradle（默认恢复） |

---

## 11. 后续扩展

M2 完成后，M3 将实现：
- Generator: 调用 Maven/Gradle 插件生成 OpenAPI Spec
- Validator: 验证生成的 OpenAPI Spec
