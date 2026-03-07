// internal/extractor/gin/types.go
package gin

// Info contains information about a Gin project.
type Info struct {
	GoVersion    string        // Go version from go.mod
	ModuleName   string        // Module path
	GinVersion   string        // gin dependency version
	HasGin       bool          // Has gin dependency
	MainFiles    []string      // main.go or files with route registration
	HandlerFiles []string      // Handler file list
	RouterGroups []RouterGroup // Detected router groups
}

// RouterGroup represents a Gin router group.
type RouterGroup struct {
	BasePath string  // Group base path (e.g., "/api/v1")
	Routes   []Route // Routes in this group
}

// Route represents a single Gin route.
type Route struct {
	Method      string   // HTTP method: GET, POST, PUT, DELETE, PATCH
	Path        string   // Route path (e.g., "/users/:id")
	FullPath    string   // Full path including group prefix
	HandlerName string   // Handler function name
	HandlerFile string   // File containing handler definition
	Middlewares []string // Middleware names
}

// HandlerInfo contains information extracted from a handler function.
type HandlerInfo struct {
	PathParams   []ParamInfo    // Path parameters (c.Param)
	QueryParams  []ParamInfo    // Query parameters (c.Query)
	HeaderParams []ParamInfo    // Header parameters (c.GetHeader)
	BodyType     string         // Request body type (from ShouldBindJSON)
	Responses    []ResponseInfo // Response info (from c.JSON calls)
}

// ParamInfo represents a parameter extracted from handler.
type ParamInfo struct {
	Name     string // Parameter name
	GoType   string // Go type name
	Required bool   // Whether parameter is required
}

// ResponseInfo represents a response from handler analysis.
type ResponseInfo struct {
	StatusCode int    // HTTP status code
	GoType     string // Response Go type
}

// HandlerRef represents a reference to a handler function.
type HandlerRef struct {
	Name         string // Function name (empty for anonymous functions)
	File         string // File path
	IsAnonymous  bool   // Whether it's an anonymous function
	ReceiverType string // Receiver type for methods (empty for functions)
}
