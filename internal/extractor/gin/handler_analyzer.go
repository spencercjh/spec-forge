package gin

import (
	"fmt"
	"go/ast"
	"go/token"
	"log/slog"
	"strings"
)

const (
	goTypeString = "string"
)

// HandlerAnalyzer analyzes Gin handler functions.
type HandlerAnalyzer struct {
	fset      *token.FileSet
	files     map[string]*ast.File
	typeCache map[string]*ast.TypeSpec
}

// NewHandlerAnalyzer creates a new HandlerAnalyzer instance.
func NewHandlerAnalyzer(fset *token.FileSet, files map[string]*ast.File) *HandlerAnalyzer {
	return &HandlerAnalyzer{
		fset:      fset,
		files:     files,
		typeCache: make(map[string]*ast.TypeSpec),
	}
}

// AnalyzeHandler analyzes a handler function and extracts information.
func (a *HandlerAnalyzer) AnalyzeHandler(fn *ast.FuncDecl) (*HandlerInfo, error) {
	slog.Debug("Analyzing handler", "name", fn.Name.Name)

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
		}
		return true
	})

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

// isFormRelatedWord checks if text contains form-related words as whole words.
// This prevents false positives like "performAction", "platform", "information".
func isFormRelatedWord(text string) bool {
	lower := strings.ToLower(text)

	// Define whole-word patterns to match
	patterns := []string{"form", "formdata", "form-data", "multipart"}

	for _, pattern := range patterns {
		// Check for pattern preceded by start of string or non-letter
		// and followed by end of string or non-letter
		for i := 0; i <= len(lower)-len(pattern); i++ {
			if lower[i:i+len(pattern)] == pattern {
				// Check prefix (start of string or non-letter)
				prefixOK := i == 0 || !isLetter(lower[i-1])
				// Check suffix (end of string or non-letter)
				suffixOK := i+len(pattern) == len(lower) || !isLetter(lower[i+len(pattern)])
				if prefixOK && suffixOK {
					return true
				}
			}
		}
	}

	return false
}

// isLetter checks if a byte is a letter (a-z).
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
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

// inferTypeFromVarName tries to infer type from variable name using common patterns.
// e.g., "user" -> "User", "users" -> "User", "result" -> "Result"
func inferTypeFromVarName(varName string) string {
	if varName == "" {
		return ""
	}

	// Common variable name -> type mappings
	mappings := map[string]string{
		"user":   "User",
		"users":  "User",
		"req":    "",
		"result": "",
		"data":   "",
		"item":   "",
		"items":  "",
	}

	if typ, ok := mappings[varName]; ok {
		return typ
	}

	// Try to capitalize first letter (heuristic)
	// e.g., "pageResult" -> "PageResult"
	if varName[0] >= 'a' && varName[0] <= 'z' {
		return string(varName[0]-'a'+'A') + varName[1:]
	}

	return varName
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

	// Categorize methods and dispatch to specialized handlers
	switch method {
	case "Param":
		a.extractParam(call, info)
	case "Query", "DefaultQuery":
		a.extractQueryParam(call, info, method == "Query")
	case "GetHeader":
		a.extractHeaderParam(call, info)
	case "ShouldBindJSON", "BindJSON":
		a.extractBodyType(call, info, varTypeMap, BindingSourceJSON)
	case "ShouldBind", "Bind":
		// ShouldBind auto-detects based on Content-Type
		// Use form binding if we detect form context in the handler
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
		// Query binding extracts query parameters, not body
		a.extractQueryBinding(call, info, varTypeMap)
	case "PostForm":
		a.extractFormParam(call, info, true)
	case "DefaultPostForm":
		a.extractFormParam(call, info, false)
	case "FormFile":
		a.extractFileParam(call, info)
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
func (a *HandlerAnalyzer) extractQueryParam(call *ast.CallExpr, info *HandlerInfo, _ bool) {
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
		return "array"
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return goTypeString
}

// findTypeSpec finds a type definition by name.
func (a *HandlerAnalyzer) findTypeSpec(name string) *ast.TypeSpec {
	// Check cache first
	if cached, ok := a.typeCache[name]; ok {
		return cached
	}

	for _, file := range a.files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok && typeSpec.Name.Name == name {
					a.typeCache[name] = typeSpec
					return typeSpec
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

// extractResponse extracts response information from c.JSON(), c.XML(), etc. calls.
func (a *HandlerAnalyzer) extractResponse(call *ast.CallExpr, info *HandlerInfo, varTypeMap map[string]string, goType string) {
	if len(call.Args) < 2 {
		return
	}
	statusCode := extractStatusCode(call.Args[0])
	if goType == "" {
		// Pass status code to determine if we should unwrap Data field
		goType = extractTypeFromResponseWithStatus(call.Args[1], varTypeMap, statusCode)
	}
	if goType != "" {
		slog.Debug("Extracted response type", "status", statusCode, "type", goType)
	}
	info.Responses = append(info.Responses, ResponseInfo{
		StatusCode: statusCode,
		GoType:     goType,
	})
}

// extractTypeFromArg extracts type name from a binding argument.
func extractTypeFromArg(expr ast.Expr, varTypeMap map[string]string) string {
	// Pattern: &variable or &Struct{}
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return ""
	}

	// Check for composite literal: &Type{}
	if comp, ok := unary.X.(*ast.CompositeLit); ok {
		if ident, ok := comp.Type.(*ast.Ident); ok {
			return ident.Name
		}
		if sel, ok := comp.Type.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				return x.Name + "." + sel.Sel.Name
			}
		}
	}

	// Check for variable: &variable
	if ident, ok := unary.X.(*ast.Ident); ok {
		// First check if we have type info from variable declaration
		if varType, exists := varTypeMap[ident.Name]; exists {
			return varType
		}
		// If we don't know the variable's type, treat it as unknown
		return ""
	}

	return ""
}

// extractStatusCode extracts HTTP status code from expression.
func extractStatusCode(expr ast.Expr) int {
	// Integer literal
	if lit, ok := expr.(*ast.BasicLit); ok {
		if lit.Kind == token.INT {
			var code int
			// Try to parse
			if _, err := fmt.Sscanf(lit.Value, "%d", &code); err == nil {
				return code
			}
		}
	}

	// http.StatusOK reference
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "http" {
			return statusCodeFromName(sel.Sel.Name)
		}
	}

	return 200 // Default
}

// statusCodeFromName converts http.StatusXxx to code.
func statusCodeFromName(name string) int {
	switch name {
	case "StatusOK":
		return 200
	case "StatusCreated":
		return 201
	case "StatusAccepted":
		return 202
	case "StatusNoContent":
		return 204
	case "StatusBadRequest":
		return 400
	case "StatusUnauthorized":
		return 401
	case "StatusForbidden":
		return 403
	case "StatusNotFound":
		return 404
	case "StatusInternalServerError":
		return 500
	case "StatusBadGateway":
		return 502
	case "StatusServiceUnavailable":
		return 503
	}
	return 200
}

// extractTypeFromResponseWithStatus extracts type from response argument considering HTTP status code.
// For success responses (200-299), it extracts the Data field type from wrapper types.
// For error responses (>= 400), it returns the wrapper type itself.
func extractTypeFromResponseWithStatus(expr ast.Expr, varTypeMap map[string]string, statusCode int) string {
	return extractTypeFromResponseInternal(expr, varTypeMap, statusCode, true)
}

// isGenericMapType checks if a type is a generic map type (gin.H, map[string]any, etc.)
// These types should have their Data field extracted rather than using the type itself.
func isGenericMapType(typeName string) bool {
	return typeName == "gin.H" ||
		typeName == "map[string]any" ||
		typeName == "map[string]interface{}" ||
		typeName == "H"
}

// extractTypeFromResponseInternal is the internal implementation with full control.
func extractTypeFromResponseInternal(expr ast.Expr, varTypeMap map[string]string, statusCode int, useStatus bool) string {
	switch e := expr.(type) {
	case *ast.Ident:
		// If it's a variable, try to get its actual type from the map
		if actualType, ok := varTypeMap[e.Name]; ok {
			return actualType
		}
		return e.Name
	case *ast.CompositeLit:
		return extractTypeFromCompositeLit(e, varTypeMap, statusCode, useStatus)
	case *ast.CallExpr:
		return extractTypeFromCallExpr(e, varTypeMap)
	case *ast.UnaryExpr:
		// &variable -> dereference and get the type
		if e.Op == token.AND {
			return extractTypeFromResponseInternal(e.X, varTypeMap, statusCode, useStatus)
		}
	}
	return ""
}

// extractTypeFromCompositeLit extracts type from a composite literal expression.
func extractTypeFromCompositeLit(e *ast.CompositeLit, varTypeMap map[string]string, statusCode int, useStatus bool) string {
	// First extract the composite literal type itself
	typeName := extractTypeNameFromExpr(e.Type)

	// Normalize gin.H to ginHType for consistency
	if typeName == "gin.H" || typeName == "H" {
		typeName = ginHType
	}

	// Determine if this is a success response (2xx) or error response (>= 400)
	isSuccess := !useStatus || (statusCode >= 200 && statusCode < 300)

	// For error responses with wrapper types, return the wrapper type
	// For success responses OR generic types, try to extract from Data field
	if !isSuccess && typeName != "" && !isGenericMapType(typeName) {
		// Error response with concrete wrapper type - return wrapper
		return typeName
	}

	// Try to extract from Data field for success responses or generic types
	if dataType := extractDataFieldType(e.Elts, varTypeMap, statusCode, useStatus); dataType != "" {
		return dataType
	}

	// Return the type name if we have it (fallback)
	if typeName != "" {
		return typeName
	}
	return ""
}

// extractTypeNameFromExpr extracts the type name from an AST expression.
func extractTypeNameFromExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}

// extractDataFieldType extracts type from Data field in composite literal elements.
func extractDataFieldType(elts []ast.Expr, varTypeMap map[string]string, statusCode int, useStatus bool) string {
	for _, elt := range elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Data" {
			continue
		}
		// Found Data field, recursively extract its type
		dataType := extractTypeFromResponseInternal(kv.Value, varTypeMap, statusCode, useStatus)
		if dataType != "" {
			return dataType
		}
	}
	return ""
}

// extractTypeFromCallExpr extracts type from a call expression.
func extractTypeFromCallExpr(e *ast.CallExpr, varTypeMap map[string]string) string {
	// gin.H or similar
	if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
		if sel.Sel.Name == "H" {
			return ginHType
		}
	}
	// Could be a function call that returns a type, try to extract from return type
	if ident, ok := e.Fun.(*ast.Ident); ok {
		if actualType, ok := varTypeMap[ident.Name]; ok {
			return actualType
		}
	}
	return ""
}
