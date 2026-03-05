# M5: Schema 字段与 API 参数增强设计文档

> **日期：** 2026-03-05
> **状态：** 已批准
> **里程碑：** M5

---

## 概述

M5 扩展 Enricher 功能，为 OpenAPI Spec 的 Schema 字段和 API 参数添加 AI 生成的描述。

**核心功能：**
- 递归增强所有 Schema 及嵌套属性
- 增强 API 参数（path/query/header/cookie）
- 混合调用策略（≤10 字段批量，>10 拆分）
- 可扩展的上下文提取架构

---

## 1. 整体架构

### 1.1 处理流程

```
┌─────────────────────────────────────────────────────────────┐
│                    Enricher.Enrich()                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Phase 1: 收集阶段 (collectElements)                        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 1. 遍历 API 端点                                       │  │
│  │    - 收集 API 操作（M4 已有）                          │  │
│  │    - 收集 API 参数（新增）                             │  │
│  │    - 收集请求/响应 Schema → 递归收集字段（新增）        │  │
│  │    - 记录已处理 Schema（避免重复）                      │  │
│  │ 2. 遍历未引用的独立 Schema（新增）                      │  │
│  │    - 递归收集字段                                      │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Phase 2: 分组阶段 (GroupByType)                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ 按 TemplateType 分组：                                 │  │
│  │ - Batch(API): 所有 API 操作                            │  │
│  │ - Batch(Param): 所有 API 参数                          │  │
│  │ - Batch(Schema): 每个 Schema 作为一个批次               │  │
│  │   (字段数 > 10 时拆分为多个批次)                        │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Phase 3: 处理阶段 (ConcurrentProcessor)                    │
│  并发调用 LLM，设置描述值                                   │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 按 API 边界聚合处理

采用方案 B：按 API 边界聚合处理，便于未来扩展基于 API 语义的增强。

```
1. 遍历每个 API 端点
   - 增强 API 操作描述
   - 增强该 API 的参数
   - 增强响应 Schema（递归处理嵌套）
2. 遍历未引用的独立 Schema
```

---

## 2. 新增包结构

### 2.1 context 包

```
internal/context/
├── extractor.go          # ContextExtractor 接口定义
├── types.go              # EnrichmentContext、SchemaContext、FieldMeta
└── noop_extractor.go     # NoOpExtractor 默认实现
```

**核心类型**：

```go
// internal/context/types.go

// EnrichmentContext 包含用于 LLM 增强的上下文信息
type EnrichmentContext struct {
    // 项目级信息
    ProjectName string
    Framework   string // "spring-boot", "gin", etc.

    // Schema 上下文
    Schemas map[string]*SchemaContext // key: schema name
}

// SchemaContext 包含单个 Schema 的上下文
type SchemaContext struct {
    // 从 OpenAPI Spec 可得
    Name   string
    Fields []FieldMeta

    // 扩展信息（由 ContextExtractor 填充）
    Description string              // 类/结构体的文档注释
    Package     string              // 所属包名
    Annotations map[string]string   // 语言特定的注解/标签
}

// FieldMeta 字段元信息
type FieldMeta struct {
    Name     string
    Type     string
    Required bool

    // 扩展信息
    Description string // 字段注释
    Tags        map[string]string // 如 json:"user_id" validate:"required"
}
```

**接口定义**：

```go
// internal/context/extractor.go

// ContextExtractor 从源代码提取上下文信息
type ContextExtractor interface {
    // Extract 从项目中提取上下文
    // spec 参数：已生成的 OpenAPI Spec，作为提取的骨架
    Extract(ctx context.Context, projectPath string, spec *openapi3.T) (*EnrichmentContext, error)

    // Name 返回提取器名称
    Name() string
}
```

**NoOpExtractor**：

```go
// internal/context/noop_extractor.go

// NoOpExtractor 默认实现，仅从 Spec 中提取基础信息
type NoOpExtractor struct{}

func (e *NoOpExtractor) Extract(_ context.Context, _ string, spec *openapi3.T) (*EnrichmentContext, error) {
    // 仅从 Spec 中提取 Schema/Field 信息，不做源代码解析
    // 降级处理时记录日志
}
```

### 2.2 enricher 包扩展

```
internal/enricher/
├── enricher.go               # 扩展：collectElements() 方法
├── processor/
│   ├── processor.go          # 扩展：SchemaElement 类型
│   ├── batch.go              # 扩展：Schema 批量解析
│   └── schema.go             # 新增：Schema 递归收集
└── prompt/
    └── templates.go          # 调整：Schema/Param 模板优化
```

---

## 3. 核心实现

### 3.1 collectElements 扩展

```go
// internal/enricher/enricher.go

func (e *Enricher) collectElements(spec *openapi3.T, language string) *processor.SpecCollector {
    collector := &processor.SpecCollector{}
    processedSchemas := make(map[string]bool) // 追踪已处理的 Schema

    // 1. 提取上下文（可选）
    var enrichCtx *context.EnrichmentContext
    if e.contextExtractor != nil {
        enrichCtx, _ = e.contextExtractor.Extract(ctx, projectPath, spec)
    }

    // 2. 遍历 API 端点
    if spec.Paths != nil {
        for _, pathStr := range spec.Paths.InMatchingOrder() {
            pathItem := spec.Paths.Value(pathStr)

            for _, item := range getOperations(pathItem) {
                // 2a. 收集 API 操作（现有逻辑）
                collector.AddAPIElement(item.method, pathStr, item.op, language)

                // 2b. 收集 API 参数（新增）
                for _, param := range item.op.Parameters {
                    if param.Value != nil && param.Value.Description == "" {
                        collector.AddParamElement(pathStr, item.method, param.Value, language)
                    }
                }

                // 2c. 收集响应 Schema（新增，递归）
                for _, response := range item.op.Responses.Map() {
                    e.collectSchemaFromResponse(response, collector, processedSchemas, language)
                }

                // 2d. 收集请求 Body Schema（新增）
                if item.op.RequestBody != nil {
                    e.collectSchemaFromRequestBody(item.op.RequestBody, collector, processedSchemas, language)
                }
            }
        }
    }

    // 3. 遍历未引用的独立 Schema（新增）
    if spec.Components != nil && spec.Components.Schemas != nil {
        for schemaName := range spec.Components.Schemas {
            if !processedSchemas[schemaName] {
                schemaRef := spec.Components.Schemas[schemaName]
                e.collectSchemaFields(schemaName, schemaRef, collector, processedSchemas, language)
            }
        }
    }

    // 4. 附加上下文信息（用于 Prompt）
    collector.SetContext(enrichCtx)

    return collector
}
```

### 3.2 Schema 递归收集

```go
// internal/enricher/processor/schema.go

const MaxSchemaDepth = 5

// SchemaElement 表示待增强的 Schema
type SchemaElement struct {
    SchemaName string
    Fields     []FieldElement
    Context    prompt.TemplateContext
}

// FieldElement 表示待增强的字段
type FieldElement struct {
    FieldName string
    FieldType string
    Required  bool
    SetValue  func(description string)
}

// CollectSchemaFields 递归收集 Schema 字段
func CollectSchemaFields(
    schemaName string,
    schemaRef *openapi3.SchemaRef,
    collector *SpecCollector,
    processed map[string]bool,
    language string,
    depth int,
) {
    // 防止无限递归
    if depth > MaxSchemaDepth {
        slog.Warn("max schema depth reached", "schema", schemaName)
        return
    }

    // 防止重复处理
    if processed[schemaName] {
        return
    }
    processed[schemaName] = true

    schema := schemaRef.Value
    if schema == nil {
        return
    }

    // 收集当前 Schema 的字段
    var fields []FieldElement
    for propName, propRef := range schema.Properties {
        prop := propRef.Value
        if prop == nil || prop.Description != "" {
            continue // 跳过已有描述的字段
        }

        field := FieldElement{
            FieldName: propName,
            FieldType: getSchemaType(prop),
            Required:  containsString(schema.Required, propName),
        }
        field.SetValue = func(desc string) {
            prop.Description = desc
        }
        fields = append(fields, field)

        // 递归处理嵌套对象
        if prop.Type == "object" && prop.Properties != nil {
            nestedName := schemaName + "." + propName
            CollectSchemaFields(nestedName, propRef, collector, processed, language, depth+1)
        }

        // 递归处理数组元素
        if prop.Type == "array" && prop.Items != nil {
            nestedName := schemaName + "." + propName + "[]"
            CollectSchemaFields(nestedName, prop.Items, collector, processed, language, depth+1)
        }
    }

    if len(fields) > 0 {
        collector.AddSchemaElement(schemaName, fields, language)
    }
}
```

### 3.3 混合批量策略

```go
// internal/enricher/processor/batch.go

const MaxFieldsPerBatch = 10

// GroupSchemasByBatch 按混合策略分组
func (c *SpecCollector) GroupSchemasByBatch() []*Batch {
    var batches []*Batch

    // API 和 Param 批次（现有逻辑）
    batches = append(batches, c.groupAPIElements()...)
    batches = append(batches, c.groupParamElements()...)

    // Schema 批次（混合策略）
    for _, schema := range c.schemas {
        fields := schema.Fields

        if len(fields) <= MaxFieldsPerBatch {
            // 单批次
            batches = append(batches, &Batch{
                Type:     prompt.TemplateTypeSchema,
                Elements: []EnrichmentElement{schemaToElement(schema)},
            })
        } else {
            // 拆分为多个批次
            for i := 0; i < len(fields); i += MaxFieldsPerBatch {
                end := i + MaxFieldsPerBatch
                if end > len(fields) {
                    end = len(fields)
                }
                subSchema := SchemaElement{
                    SchemaName: schema.SchemaName,
                    Fields:     fields[i:end],
                }
                batches = append(batches, &Batch{
                    Type:     prompt.TemplateTypeSchema,
                    Elements: []EnrichmentElement{schemaToElement(subSchema)},
                })
            }
        }
    }

    return batches
}
```

### 3.4 Schema 响应解析

```go
// internal/enricher/processor/batch.go

// parseSchemaResponse 解析 Schema 字段描述响应
func parseSchemaResponse(response string, fields []FieldElement) {
    response = stripMarkdownCodeBlock(response)

    var result map[string]string
    if err := json.Unmarshal([]byte(response), &result); err != nil {
        slog.Warn("failed to parse schema response", "error", err)
        return
    }

    // 将描述映射到字段
    for _, field := range fields {
        if desc, ok := result[field.FieldName]; ok {
            field.SetValue(desc)
        }
    }
}
```

---

## 4. Prompt 模板调整

### 4.1 Schema 模板

```go
TemplateTypeSchema: {
    System: `You are an API documentation expert. Generate concise field descriptions.
Respond in {{.Language}} language.
Output format: JSON object mapping field names to their descriptions.
Example: {"userId": "The unique identifier of the user", "email": "The user's email address"}`,
    User: `Schema: {{.SchemaName}}
Fields:
{{range .Fields}}- {{.Name}} ({{.Type}}, {{if .Required}}required{{else}}optional{{end}})
{{end}}

Generate a concise description for each field. Keep descriptions brief (1-2 sentences).`,
},
```

### 4.2 Param 模板

```go
TemplateTypeParam: {
    System: `You are an API documentation expert. Generate concise parameter descriptions.
Respond in {{.Language}} language.
Output format: JSON with "description" field.`,
    User: `API Endpoint: {{.Method}} {{.Path}}
Parameter: {{.ParamName}}
Type: {{.FieldType}}
In: {{.ParamIn}}
Required: {{.Required}}

Generate a brief description for this parameter.`,
},
```

---

## 5. 错误处理

| 场景 | 处理方式 |
|------|---------|
| Schema 无字段 | 跳过，不调用 LLM |
| Schema 字段响应解析失败 | 跳过该 Schema，记录警告日志，继续其他 |
| 单个字段描述设置失败 | 跳过该字段，继续其他字段 |
| ContextExtractor 失败 | 降级为 NoOpExtractor，记录警告日志 |
| 递归深度过大 | 限制最大深度为 5，超过则记录警告并停止递归 |
| 循环引用 Schema | 通过 processedSchemas map 检测并跳过 |

---

## 6. 测试策略

### 6.1 新增测试文件

```
internal/
├── context/
│   ├── extractor_test.go         # NoOpExtractor 测试
│   └── types_test.go             # 类型测试
│
└── enricher/
    ├── enricher_test.go          # 扩展：Schema/Param 收集测试
    └── processor/
        ├── batch_test.go         # 扩展：Schema 批量解析测试
        └── schema_test.go        # 新增：Schema 递归收集测试
```

### 6.2 测试用例

**Schema 收集**：
- 简单 Schema（≤10 字段）
- 复杂 Schema（>10 字段，需拆分）
- 嵌套 Schema（递归处理）
- 循环引用 Schema（检测并跳过）
- 无字段 Schema（跳过）

**参数收集**：
- Path 参数
- Query 参数
- Header 参数
- 混合参数
- 已有描述的参数（跳过）

**响应解析**：
- JSON 格式响应
- Markdown 代码块包裹的 JSON
- 非格式化响应（跳过并记录日志）

---

## 7. 文件变更清单

### 7.1 新增文件

| 文件 | 说明 |
|------|------|
| `internal/context/extractor.go` | ContextExtractor 接口定义 |
| `internal/context/types.go` | EnrichmentContext、SchemaContext、FieldMeta 类型 |
| `internal/context/noop_extractor.go` | NoOpExtractor 默认实现 |
| `internal/enricher/processor/schema.go` | Schema 递归收集逻辑 |

### 7.2 修改文件

| 文件 | 变更 |
|------|------|
| `internal/enricher/enricher.go` | 扩展 collectElements()，新增 Schema/Param 收集 |
| `internal/enricher/processor/processor.go` | 新增 SchemaElement、FieldElement 类型 |
| `internal/enricher/processor/batch.go` | 新增 Schema 批量解析、混合策略分组 |
| `internal/enricher/prompt/templates.go` | 调整 Schema/Param 模板 |

---

## 8. 后续扩展

M5 之后的潜在扩展：

1. **Spring Context Extractor** - 从 Java 源代码提取 Javadoc、注解等丰富上下文
2. **Go Context Extractor** - 从 Go 源代码提取 struct tag、注释
3. **Response 增强** - 为 API 响应状态码生成描述
4. **缓存机制** - 避免重复增强相同 Schema

---

## 9. 设计决策总结

| 项目 | 决策 |
|------|------|
| Schema 范围 | 递归增强所有 Schema 及嵌套属性 |
| 调用策略 | 混合：≤10 字段批量，>10 拆分 |
| 参数增强 | 是，M5 一起实现 |
| 处理顺序 | 按 API 边界聚合处理 |
| 上下文提取 | 独立 `internal/context/` 包 |
| M5 交付 | NoOpExtractor 仅从 Spec 提取，Spring Extractor 后续版本 |
| 降级处理 | NoOpExtractor + 警告日志 |
| 最大递归深度 | 5 |
