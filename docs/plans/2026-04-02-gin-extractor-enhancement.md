# Gin Extractor Enhancement Plan

## Overview

Spec-forge 的 Gin extractor 基于 Go AST 静态分析生成 OpenAPI spec，与 swaggo/swag（基于注解）定位不同。经过与真实项目 `openapi` 的对比，识别出多项差距。本文档规划改进方案，目标是在不侵入业务代码的前提下，尽可能缩小与 swaggo 生成结果的差距。

## 当前差距总结

基于 `/home/caijh/codes/open/openapi` 项目，对比 spec-forge 生成的 `openapi.yaml`（OpenAPI 3.0.3）和 swaggo 生成的 `docs/swagger.yaml`（Swagger 2.0）：

| # | 差距                                      | 严重度 | 分类       | 根因                                              |
|---|-----------------------------------------|-----|----------|-------------------------------------------------|
| 1 | Response schema 大部分丢失，几乎都显示 `"Success"` | 高   | AST 可修复  | handler analyzer 无法穿透 `done()` helper 提取实际返回类型  |
| 2 | Schema 缺少 embedded struct 字段            | 高   | AST 可修复  | schema_extractor.go:261 直接 `continue` 跳过嵌入字段    |
| 3 | 多了 `/swagger/{any}` 等非业务路由              | 中   | AST 可修复  | AST parser 无路由过滤机制                              |
| 4 | 参数类型全部是 string，无法区分 int/bool 等          | 中   | AST 可修复  | handler analyzer 将所有 c.Query/c.Param 标记为 string |
| 5 | 缺少 description、tags、summary 友好名         | 中   | Enricher | swaggo 靠注解，我们靠 LLM                              |
| 6 | Response wrapper (Rsp) 结构体字段不全          | 中   | AST 可修复  | embedded struct 未展开导致 BaseRsp 字段丢失              |
| 7 | 缺少 example/enum/default/maxLength 等约束   | 低   | 需要注解     | AST 无法推断业务级约束                                   |
| 8 | operationId 是 `apis.CreateProject` 格式   | 低   | 格式调整     | 需要清理包名前缀                                        |
| 9 | 重复路由（新旧两套路由都被提取）                        | 低   | 配置项      | 用户可能需要 exclude 选项                               |

## 改进方案

### P0: Response Schema 提取增强

**问题**: 真实项目中 99% 的 handler 通过 `done(c, data, err)` 返回响应。`done()` 内部调用 `c.JSON(http.StatusOK, rsp)`，其中 `rsp` 类型是 `msgs.Rsp`。当前 analyzer 只看到 `c.JSON` 调用的直接参数，无法穿透 helper 函数。

**方案**: 实现两层改进：

#### 方案 A: 识别常见 response wrapper 模式

在 `handler_analyzer.go` 中增加对以下模式的支持：

1. **`done(c, data, err)` 模式**
   - 检测 handler 调用 `done(c, expr1, expr2)` 形式
   - 第一个非 Context 参数视为响应数据
   - 通过 `varTypeMap` 追踪变量类型，提取实际 GoType
   - 生成 200 响应，schema 为提取到的数据类型
   - 生成 default 错误响应，schema 为 wrapper（如 `Rsp`）

2. **`done(c, err)` 模式** (无 data)
   - 仅生成 default 错误响应

3. **`done(c, data)` 模式**
   - 生成 200 响应，schema 为 data 类型

4. **自定义 helper 识别**
   - 维护内置 helper 名单: `done`, `response`, `respond`, `writeJSON`, `sendJSON`
   - 通过签名匹配（第一个参数 `*gin.Context`）识别自定义 helper

#### 方案 B: 跨函数调用追踪

对于非标准 helper，实现轻量级的跨函数调用追踪：

```
handler CreateProject
  └─ calls done(c, data, err)
       └─ resolved: done() calls c.JSON(200, Rsp{Data: fields[0]})
```

1. 在 AST parser 阶段，记录所有"helper 函数"（接收 `*gin.Context` 的包级函数）
2. 当 handler body 中调用 helper 时，进入 helper 函数体查找 `c.JSON`/`c.XML` 调用
3. 结合调用点的实参和 helper 内部的形参映射，推断实际响应类型

**实现要点**:

```go
// handler_analyzer.go 新增

// ResponseHelper 描述一个已知的响应 helper 函数
type ResponseHelper struct {
    FuncName  string   // 函数名，如 "done"
    DataParam int      // data 参数位置 (0-indexed, 跳过 Context)
    ErrParam  int      // error 参数位置 (-1 表示无)
    StatusOK  int      // 成功时 HTTP 状态码
}

// 内置 helper 名单
var builtinHelpers = []ResponseHelper{
    {FuncName: "done", DataParam: 0, ErrParam: 1, StatusOK: 200},
    {FuncName: "respond", DataParam: 0, ErrParam: 1, StatusOK: 200},
}
```

**预期效果**: 对于 `done(c, data, err)` 调用：
- 如果 `data` 类型为 `models.Project`，生成 `200` 响应引用 `Project` schema
- 同时生成 `default` 错误响应引用 `Rsp` schema
- 覆盖真实项目中 ~95% 的 handler

### P0: Embedded Struct 展开

**问题**: `schema_extractor.go:261` 直接跳过嵌入字段。导致 `Rsp`（嵌入 `BaseRsp`）的 `code`/`msg`/`_cost`/`_err` 字段全部丢失。

**方案**: 实现 Go 语言的字段提升规则：

```
type BaseRsp struct {
    Code int    `json:"code"`
    Msg  string `json:"msg"`
}
type Rsp struct {
    BaseRsp           // 嵌入字段
    Data any   `json:"data"`
}
```

生成 OpenAPI schema 应为：
```yaml
Rsp:
  type: object
  properties:
    code:
      type: integer
    msg:
      type: string
    data:
      type: object
```

**实现逻辑**:

```go
// schema_extractor.go 修改 extractStructSchema

if len(field.Names) == 0 {
    // Embedded field
    embeddedType := resolveFieldType(field.Type, filePkg, imports)
    if embeddedType != "" {
        // 递归提取嵌入类型的字段
        embeddedSchema := e.extractStructSchema(embeddedType, ...)
        if embeddedSchema != nil {
            for name, prop := range embeddedSchema.Properties {
                // 字段提升：嵌入字段的属性提升到外层
                if _, exists := schema.Properties[name]; !exists {
                    schema.Properties[name] = prop
                }
                // 嵌入字段的 required 也提升
            }
            for _, req := range embeddedSchema.Required {
                schema.Required = append(schema.Required, req)
            }
        }
    }
    continue
}
```

**边界情况处理**:
- 指针嵌入 `*BaseRsp` — 解引用后处理
- 跨包嵌入 `msgs.BaseRsp` — 通过 import 解析
- 多层嵌入 `A embeds B embeds C` — 递归展开
- 嵌入接口 — 跳过（无法推断字段）
- 循环嵌入 — 使用 visited map 防止无限递归（已有机制）

### P1: 路由过滤

**问题**: AST parser 提取了 `/swagger/{any}` 等 framework/docs 路由，这些不是业务 API。

**方案**: 在 `ast_parser.go` 的 `ExtractRoutes` 中增加默认过滤 + 可配置过滤：

```go
// 默认过滤前缀
var defaultExcludePrefixes = []string{
    "/swagger",
    "/docs",
    "/debug",
    "/static",
    "/public",
    "/favicon.ico",
}

// RouteFilterConfig 路由过滤配置
type RouteFilterConfig struct {
    ExcludePrefixes []string // 排除的前缀
    ExcludeExact    []string // 精确排除的路径
}
```

**过滤时机**: 在 route 提取后、handler 分析前过滤，避免浪费分析资源。

**CLI 集成**:
```bash
spec-forge generate ./project --exclude-route /swagger --exclude-route /debug
```

### P1: 参数类型推断

**问题**: 所有 c.Query/c.Param/c.GetHeader 返回的参数类型均为 string。

**方案**: 多策略推断参数类型：

#### 策略 1: 从 `strconv` 调用推断

```go
// 检测模式
offset, _ := strconv.Atoi(c.Query("offset"))  // → integer
limit := c.GetInt("limit")                      // → integer
```

在 handler analyzer 中追踪 `strconv.Atoi`/`strconv.ParseInt`/`strconv.ParseBool` 等调用，匹配其输入是否为 `c.Query`/`c.Param` 调用结果，从而推断参数类型。

#### 策略 2: 从 binding 结构体推断

```go
// 如果 handler 使用 ShouldBindQuery
var req ListRequest
if err := c.ShouldBindQuery(&req); err != nil { ... }
```

从 `ListRequest` 结构体字段类型推断 query 参数的 OpenAPI 类型。

#### 策略 3: 从条件判断推断 bool

```go
if c.Query("verbose") == "true" { ... }
// → boolean
```

**类型映射**:

| Go 调用                               | OpenAPI 类型    |
|-------------------------------------|---------------|
| `strconv.Atoi` / `strconv.ParseInt` | `integer`     |
| `strconv.ParseBool`                 | `boolean`     |
| `strconv.ParseFloat`                | `number`      |
| `c.GetInt` / `c.GetInt64`           | `integer`     |
| `c.GetBool`                         | `boolean`     |
| 直接使用 c.Query 值                      | `string` (默认) |

### P2: Enricher 增强 — Tags 和 Summary

**问题**: 生成的 spec 缺少 tags 和友好的 summary/operationId。

**方案**: 扩展 enricher 的 prompt 模板：

#### Tags 推断

根据路由前缀自动分组：
- `/api/v1/projects/*` → tag: `projects`
- `/api/v1/users/*` → tag: `users`
- `/api/v1/tenants/*` → tag: `tenants`

可以在 extractor 阶段直接实现，无需 LLM：

```go
func inferTag(path string) string {
    parts := strings.Split(strings.Trim(path, "/"), "/")
    // /api/v1/projects/{name} → "projects"
    for i, p := range parts {
        if !strings.HasPrefix(p, "{") && i >= 2 {
            return p
        }
    }
    return ""
}
```

#### Summary/OperationId 清理

将 `apis.CreateProject` 转为 `CreateProject`，去除包名前缀。

### P2: Binding Tag 解析增强

**问题**: 当前只从 `binding:"required"` 提取 required，未利用其他 tag 信息。

**方案**: 扩展 `applyTags` 函数支持更多 binding/validate tag：

| Tag                               | OpenAPI 映射                 |
|-----------------------------------|----------------------------|
| `binding:"required"`              | `required: true`           |
| `binding:"min=3"`                 | `minLength: 3`             |
| `binding:"max=16"`                | `maxLength: 16`            |
| `binding:"oneof=active inactive"` | `enum: [active, inactive]` |
| `binding:"email"`                 | `format: email`            |
| `binding:"url"`                   | `format: uri`              |
| `validate:"min=0"`                | `minimum: 0`               |
| `validate:"max=100"`              | `maximum: 100`             |

### P3: 可选 — Exclude/Include 路由配置

**问题**: 真实项目中存在新旧两套路由（`/api/v1/projects/{project}/rjobs` vs `/api/v1/tenant/{tenant}/projects/{project}/rjobs`），都被提取。

**方案**: 在 `.spec-forge.yaml` 或 CLI flag 中支持：

```yaml
gin:
  excludeRoutes:
    - "/api/v1/projects/{project}/rjob*"  # 旧路由
  excludePrefixes:
    - "/swagger"
```

```bash
spec-forge generate ./project --exclude-route-prefix /swagger
```

## 实现优先级与计划

| 阶段      | 内容                               | 预计影响                    | 复杂度 |
|---------|----------------------------------|-------------------------|-----|
| Phase 1 | Embedded struct 展开               | 修复所有 schema 的字段完整性      | 低   |
| Phase 2 | Response schema 提取 (done helper) | 修复 ~95% handler 的响应体    | 中   |
| Phase 3 | 路由过滤 (默认排除)                      | 去除噪音路由                  | 低   |
| Phase 4 | 参数类型推断                           | query/path/header 参数类型化 | 中   |
| Phase 5 | Tags + Summary 清理                | 改善文档可读性                 | 低   |
| Phase 6 | Binding tag 增强                   | 补充约束信息                  | 低   |
| Phase 7 | 可选路由排除配置                         | 处理重复路由                  | 低   |

## 验证方案

每个 Phase 完成后，使用 `openapi` 真实项目验证：

```bash
# 生成 spec
go run . generate /home/caijh/codes/open/openapi --output-dir /tmp/spec-test --output yaml --skip-enrich --skip-publish

# 对比与 swaggo 的差异
diff /tmp/spec-test/openapi.yaml /home/caijh/codes/open/openapi/docs/swagger.yaml
```

关键指标：
- Schema 数量：从当前 13 个提升到接近 swaggo 的 ~15 个
- 响应 schema 覆盖率：从当前 ~5% 提升到 >80%
- 无效路由数：从当前 1 个降至 0
- 参数类型准确率：从当前 0% (全 string) 提升到 >60%

## 已知限制（AST 无法解决）

以下差距需要 swaggo 式注解或 LLM enricher，不在 AST 增强范围内：

1. **example 值** — 需要注解 `@Param ... example: "my-project"`
2. **业务级 required** — 如 tenant 是否必填取决于业务逻辑
3. **复杂响应 schema (allOf)** — 如 Ping 返回 `Rsp + Pong` 的组合
4. **description** — 中文描述需要 LLM enricher 或注解
5. **consumes/produces** — 对于 Gin 通常都是 JSON，可在 generator 中硬编码默认值
6. **host/basePath** — OpenAPI 3.0 已废弃这些字段，用 servers 替代
7. **securityDefinitions** — 需要从 middleware 或配置推断
