# M2: Spring 检测和 Patch 设计文档

> **日期：** 2026-03-03
> **状态：** 已批准
> **里程碑：** M2

---

## 概述

M2 实现 Spring 项目的检测和 Patch 功能，为后续的 OpenAPI 提取做准备。

**核心功能：**
- `spec-forge spring detect` - 检测 Spring 项目信息
- `spec-forge spring patch` - 添加 springdoc 依赖和插件

---

## 1. 目录结构

```
internal/
└── extractor/
    ├── extractor.go           # Extractor 接口定义
    └── spring/
        ├── detector.go         # 项目检测
        ├── patcher.go          # 依赖注入
        ├── maven.go            # Maven 解析和修改 (使用 vifraa/gopom)
        └── gradle.go           # Gradle 解析和修改 (使用 scagogogo/gradle-parser)
```

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
}

// PatchOptions 配置 patch 行为
type PatchOptions struct {
    DryRun              bool   // 只打印修改，不写入
    Force               bool   // 强制覆盖已有依赖
    SpringdocVersion    string // springdoc 版本（默认内置）
    MavenPluginVersion  string // Maven 插件版本（默认内置）
    GradlePluginVersion string // Gradle 插件版本（默认内置）
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

如果缺少依赖或插件：
```
springdoc Dependency: ❌ Not found
springdoc Plugin:     ❌ Not configured
```

### 检测逻辑

1. 检查项目根目录是否有 `pom.xml` 或 `build.gradle`
2. 解析构建文件
3. 提取 Spring Boot 版本（从 parent 或 dependencyManagement）
4. 检查 springdoc 依赖是否存在
5. 检查 springdoc 插件是否配置
6. 输出人类可读的结果

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
3. 根据构建工具类型：
   - Maven: 修改 pom.xml
   - Gradle: 修改 build.gradle
4. 如果 --dry-run，只打印修改，不写入文件
5. 输出修改结果
```

---

## 6. 依赖库

| 功能 | 库 | 用途 |
|------|-----|------|
| Maven 解析 | `github.com/vifraa/gopom` | 解析和修改 pom.xml |
| Gradle 解析 | `github.com/scagogogo/gradle-parser` | 解析和修改 build.gradle |

---

## 7. 错误处理

| 错误场景 | 处理方式 |
|----------|----------|
| 找不到构建文件 | 返回错误，提示用户检查目录 |
| 构建文件格式错误 | 返回错误，提示解析失败 |
| 无法确定 Spring Boot 版本 | 警告，但继续执行 |
| 写入文件失败 | 返回错误，保留原文件 |

---

## 8. 测试策略

### 单元测试

- `detector_test.go` - 测试 detect 逻辑
- `maven_test.go` - 测试 Maven 解析和修改
- `gradle_test.go` - 测试 Gradle 解析和修改
- `patcher_test.go` - 测试 patch 逻辑

### 集成测试

使用 `integration-tests/` 下的 Maven 和 Gradle demo 项目：
1. 运行 `spec-forge spring detect` 验证检测结果
2. 运行 `spec-forge spring patch --dry-run` 验证修改内容
3. 运行 `spec-forge spring patch` 验证实际修改
4. 再次运行 detect 验证 patch 成功

---

## 9. CLI 命令参数

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

---

## 10. 后续扩展

M2 完成后，M3 将实现：
- Generator: 调用 Maven/Gradle 插件生成 OpenAPI Spec
- Validator: 验证生成的 OpenAPI Spec
