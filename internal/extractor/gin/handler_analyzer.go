package gin

import (
	"go/ast"
	"go/token"
	"log/slog"
	"strings"
)

const (
	goTypeString  = "string"
	goTypeArray   = "array"
	goTypeInteger = "integer"
	goTypeBoolean = "boolean"
	goTypeNumber  = "number"
)

// HandlerAnalyzer analyzes Gin handler functions.
type HandlerAnalyzer struct {
	fset        *token.FileSet
	files       map[string]*ast.File
	typeCache   map[string]*ast.TypeSpec
	helperFuncs map[string]map[string]*ast.FuncDecl // pkg → name → package-level funcs taking *gin.Context as first param
	callerPkg   string                              // set per AnalyzeHandler call for scoped helper lookup
}

// NewHandlerAnalyzer creates a new HandlerAnalyzer instance.
func NewHandlerAnalyzer(fset *token.FileSet, files map[string]*ast.File) *HandlerAnalyzer {
	a := &HandlerAnalyzer{
		fset:        fset,
		files:       files,
		typeCache:   make(map[string]*ast.TypeSpec),
		helperFuncs: make(map[string]map[string]*ast.FuncDecl),
	}
	a.discoverHelperFuncs()
	return a
}

// isGinContextType checks if an AST expression represents *gin.Context.
func isGinContextType(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "gin" && sel.Sel.Name == "Context"
}

// AnalyzeHandler analyzes a handler function and extracts information.
func (a *HandlerAnalyzer) AnalyzeHandler(fn *ast.FuncDecl) (*HandlerInfo, error) {
	slog.Debug("Analyzing handler", "name", fn.Name.Name)

	// Determine caller's package for scoped helper function lookup
	a.callerPkg = a.pkgForFunc(fn)

	info := &HandlerInfo{
		PathParams:   []ParamInfo{},
		QueryParams:  []ParamInfo{},
		HeaderParams: []ParamInfo{},
		FormParams:   []ParamInfo{},
		FileParams:   []ParamInfo{},
		Responses:    []ResponseInfo{},
	}

	// Inspect the function body
	if fn.Body == nil {
		slog.Debug("Handler has no body", "name", fn.Name.Name)
		return info, nil
	}

	// Build variable type map for type inference
	varTypeMap := a.buildVarTypeMap(fn.Body)

	// Check if handler has form context (comments, PostForm, FormFile, etc.)
	// This helps infer that ShouldBind should use form binding
	hasFormContext := a.hasFormContext(fn)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if node, ok := n.(*ast.CallExpr); ok {
			a.parseHandlerCall(node, info, varTypeMap, hasFormContext)
			a.parseHelperCall(node, info, varTypeMap)
		}
		return true
	})

	// Infer parameter types from strconv conversions and conditional checks
	a.inferParamTypes(fn.Body, info)

	// For form binding, expand struct fields into form params
	if info.BindingSrc == BindingSourceForm && info.BodyType != "" {
		formParams := a.extractFormParamsFromType(info.BodyType)
		info.FormParams = append(info.FormParams, formParams...)
	}

	return info, nil
}

// hasFormContext checks if the handler contains form-related calls or comments.
// This is used to infer that ShouldBind should use form binding.
func (a *HandlerAnalyzer) hasFormContext(fn *ast.FuncDecl) bool {
	// Check function comments for form-related keywords
	if fn.Doc != nil {
		for _, comment := range fn.Doc.List {
			if isFormRelatedWord(comment.Text) {
				return true
			}
		}
	}

	// Check function name for form-related keywords
	if isFormRelatedWord(fn.Name.Name) {
		return true
	}

	// Check body for form-related calls
	if fn.Body != nil {
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					switch sel.Sel.Name {
					case "PostForm", "DefaultPostForm", "FormFile", "MultipartForm":
						found = true
						return false
					}
				}
			}
			return true
		})
		if found {
			return true
		}
	}

	return false
}

// buildVarTypeMap builds a map of variable names to their types.
func (a *HandlerAnalyzer) buildVarTypeMap(body *ast.BlockStmt) map[string]string {
	varTypeMap := make(map[string]string)

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.DeclStmt:
			a.extractVarDeclTypes(node, varTypeMap)
		case *ast.AssignStmt:
			a.extractAssignTypes(node, varTypeMap)
		}
		return true
	})

	if len(varTypeMap) > 0 {
		slog.Debug("Built variable type map", "entries", len(varTypeMap))
	}
	return varTypeMap
}

// extractVarDeclTypes extracts types from var declarations.
func (a *HandlerAnalyzer) extractVarDeclTypes(node *ast.DeclStmt, varTypeMap map[string]string) {
	genDecl, ok := node.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range valueSpec.Names {
			if i < len(valueSpec.Values) {
				// Check if value is a composite literal with type
				if comp, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
					switch t := comp.Type.(type) {
					case *ast.Ident:
						varTypeMap[name.Name] = t.Name
					case *ast.SelectorExpr:
						if x, ok := t.X.(*ast.Ident); ok {
							varTypeMap[name.Name] = x.Name + "." + t.Sel.Name
						}
					}
				}
			}
			// Also check explicit type
			if valueSpec.Type != nil {
				switch t := valueSpec.Type.(type) {
				case *ast.Ident:
					varTypeMap[name.Name] = t.Name
				case *ast.SelectorExpr:
					if x, ok := t.X.(*ast.Ident); ok {
						varTypeMap[name.Name] = x.Name + "." + t.Sel.Name
					}
				}
			}
		}
	}
}

// extractAssignTypes extracts types from short variable declarations.
func (a *HandlerAnalyzer) extractAssignTypes(node *ast.AssignStmt, varTypeMap map[string]string) {
	if node.Tok != token.DEFINE {
		return
	}
	for i, lhs := range node.Lhs {
		if i >= len(node.Rhs) {
			continue
		}
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		// Check if RHS is a composite literal with type
		if comp, ok := node.Rhs[i].(*ast.CompositeLit); ok {
			switch t := comp.Type.(type) {
			case *ast.Ident:
				varTypeMap[ident.Name] = t.Name
			case *ast.SelectorExpr:
				if x, ok := t.X.(*ast.Ident); ok {
					varTypeMap[ident.Name] = x.Name + "." + t.Sel.Name
				}
			}
		} else {
			// Try to infer type from variable name using heuristic
			if inferredType := inferTypeFromVarName(ident.Name); inferredType != "" {
				varTypeMap[ident.Name] = inferredType
			}
		}
	}
}

// parseHandlerCall parses a call expression in a handler.
func (a *HandlerAnalyzer) parseHandlerCall(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string, hasFormContext bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Check if receiver is a context variable
	if _, ok = sel.X.(*ast.Ident); !ok {
		return
	}

	method := sel.Sel.Name

	// Dispatch parameter extraction methods
	if a.dispatchParamMethods(method, call, info, varTypeMap) {
		return
	}
	// Dispatch binding methods
	if a.dispatchBindingMethods(method, call, info, varTypeMap, hasFormContext) {
		return
	}
	// Dispatch response methods
	a.dispatchResponseMethods(method, call, info, varTypeMap)
}

// dispatchParamMethods handles parameter extraction methods (Param, Query, GetHeader, etc.).
func (a *HandlerAnalyzer) dispatchParamMethods(method string, call *ast.CallExpr, info *HandlerInfo, _ map[string]string) bool {
	switch method {
	case "Param":
		a.extractParam(call, info)
	case "Query", "DefaultQuery": //nolint:goconst // AST method name literals
		a.extractQueryParam(call, info)
	case "GetHeader":
		a.extractHeaderParam(call, info)
	default:
		return false
	}
	return true
}

// dispatchBindingMethods handles body/form binding methods.
func (a *HandlerAnalyzer) dispatchBindingMethods(method string, call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string, hasFormContext bool) bool {
	switch method {
	case "ShouldBindJSON", "BindJSON":
		a.extractBodyType(call, info, varTypeMap, BindingSourceJSON)
	case "ShouldBind", "Bind":
		if hasFormContext {
			a.extractBodyType(call, info, varTypeMap, BindingSourceForm)
		} else {
			a.extractBodyType(call, info, varTypeMap, BindingSourceJSON)
		}
	case "ShouldBindXML", "BindXML":
		a.extractBodyType(call, info, varTypeMap, BindingSourceXML)
	case "ShouldBindYAML", "BindYAML":
		a.extractBodyType(call, info, varTypeMap, BindingSourceYAML)
	case "ShouldBindQuery", "BindQuery":
		a.extractQueryBinding(call, info, varTypeMap)
	case "PostForm":
		a.extractFormParam(call, info, true)
	case "DefaultPostForm":
		a.extractFormParam(call, info, false)
	case "FormFile":
		a.extractFileParam(call, info)
	default:
		return false
	}
	return true
}

// dispatchResponseMethods handles response methods (JSON, XML, etc.).
func (a *HandlerAnalyzer) dispatchResponseMethods(method string, call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string) {
	switch method {
	case "JSON", "XML", "YAML":
		a.extractResponse(call, info, varTypeMap, "")
	case "String":
		a.extractResponse(call, info, varTypeMap, goTypeString)
	case "Data":
		a.extractResponse(call, info, varTypeMap, "binary")
	case "Redirect":
		a.extractResponse(call, info, varTypeMap, "")
	}
}

// extractParam extracts path parameter from c.Param() call.
func (a *HandlerAnalyzer) extractParam(call *ast.CallExpr, info *HandlerInfo) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.PathParams = append(info.PathParams, ParamInfo{
			Name:     name,
			GoType:   goTypeString,
			Required: true,
		})
	}
}

// extractQueryParam extracts query parameter from c.Query() or c.DefaultQuery() call.
func (a *HandlerAnalyzer) extractQueryParam(call *ast.CallExpr, info *HandlerInfo) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		// In Gin, Query returns empty string when missing and does not enforce presence
		// Mark query params as optional (required=false) unless explicit validation exists
		info.QueryParams = append(info.QueryParams, ParamInfo{
			Name:     name,
			GoType:   goTypeString,
			Required: false,
		})
	}
}

// extractHeaderParam extracts header parameter from c.GetHeader() call.
func (a *HandlerAnalyzer) extractHeaderParam(call *ast.CallExpr, info *HandlerInfo) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.HeaderParams = append(info.HeaderParams, ParamInfo{
			Name:     name,
			GoType:   goTypeString,
			Required: false,
		})
	}
}

// extractBodyType extracts body type from binding calls like c.ShouldBindJSON().
func (a *HandlerAnalyzer) extractBodyType(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string, source BindingSource) {
	if len(call.Args) < 1 {
		return
	}
	if typeName := extractTypeFromArg(call.Args[0], varTypeMap); typeName != "" {
		info.BodyType = typeName
		info.BindingSrc = source
		slog.Debug("Extracted body type", "type", typeName, "source", source)
	}
}

// extractQueryBinding handles ShouldBindQuery by extracting query parameters from struct tags.
func (a *HandlerAnalyzer) extractQueryBinding(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string) {
	if len(call.Args) < 1 {
		return
	}
	typeName := extractTypeFromArg(call.Args[0], varTypeMap)
	if typeName == "" {
		return
	}

	info.BodyType = typeName
	info.BindingSrc = BindingSourceQuery
	slog.Debug("Extracted query binding type", "type", typeName)

	// Expand struct form tags into query parameters
	queryParams := a.extractQueryParamsFromType(typeName)
	info.QueryParams = append(info.QueryParams, queryParams...)
}

// extractQueryParamsFromType extracts query parameters from a struct type's form tags.
func (a *HandlerAnalyzer) extractQueryParamsFromType(typeName string) []ParamInfo {
	return a.extractParamsFromType(typeName)
}

// extractFormParamsFromType extracts form parameters from a struct type's form tags.
func (a *HandlerAnalyzer) extractFormParamsFromType(typeName string) []ParamInfo {
	return a.extractParamsFromType(typeName)
}

// extractParamsFromType extracts parameters from a struct type's tags.
func (a *HandlerAnalyzer) extractParamsFromType(typeName string) []ParamInfo {
	var params []ParamInfo

	typeSpec := a.findTypeSpec(typeName)
	if typeSpec == nil {
		return params
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return params
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		paramName := fieldName
		isRequired := false

		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")

			// Check form tag first, then json tag as fallback
			if formTag := extractTagValue(tag, "form"); formTag != "" {
				parts := strings.Split(formTag, ",")
				if parts[0] != "" {
					paramName = parts[0]
				}
			} else if jsonTag := extractTagValue(tag, "json"); jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" && parts[0] != "-" {
					paramName = parts[0]
				}
			}

			// Check binding tag for required
			if bindingTag := extractTagValue(tag, "binding"); bindingTag != "" {
				if strings.Contains(bindingTag, "required") {
					isRequired = true
				}
			}
		}

		if paramName == "" || paramName == "-" {
			continue
		}

		goType := a.extractGoTypeName(field.Type)

		params = append(params, ParamInfo{
			Name:     paramName,
			GoType:   goType,
			Required: isRequired,
		})
	}

	return params
}

// extractGoTypeName extracts the Go type name from an AST expression.
func (a *HandlerAnalyzer) extractGoTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return a.extractGoTypeName(t.X)
	case *ast.ArrayType:
		return goTypeArray
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return goTypeString
}

// findTypeSpec finds a type definition by name.
// Supports both local types ("User") and cross-package types ("models.User").
func (a *HandlerAnalyzer) findTypeSpec(name string) *ast.TypeSpec {
	// Check cache first
	if cached, ok := a.typeCache[name]; ok {
		return cached
	}

	// For cross-package references (e.g., "models.User"), try to find by short name
	shortName := name
	if idx := strings.LastIndex(name, "."); idx != -1 {
		shortName = name[idx+1:]
	}

	// Try both full name and short name
	namesToTry := []string{name}
	if shortName != name {
		namesToTry = append(namesToTry, shortName)
	}

	for _, tryName := range namesToTry {
		for _, file := range a.files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if ok && typeSpec.Name.Name == tryName {
						a.typeCache[name] = typeSpec
						return typeSpec
					}
				}
			}
		}
	}
	return nil
}

// extractFormParam extracts form parameter from c.PostForm() or c.DefaultPostForm() call.
func (a *HandlerAnalyzer) extractFormParam(call *ast.CallExpr, info *HandlerInfo, required bool) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.FormParams = append(info.FormParams, ParamInfo{
			Name:     name,
			GoType:   goTypeString,
			Required: required,
		})
	}
}

// extractFileParam extracts file parameter from c.FormFile() call.
func (a *HandlerAnalyzer) extractFileParam(call *ast.CallExpr, info *HandlerInfo) {
	if len(call.Args) < 1 {
		return
	}
	if name := extractStringLiteral(call.Args[0]); name != "" {
		info.FileParams = append(info.FileParams, ParamInfo{
			Name:     name,
			GoType:   "file",
			Required: true,
		})
	}
}
