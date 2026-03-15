package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// This fixture tests unsupported/edge cases that should be handled gracefully:
// - Wildcard routes
// - Inline anonymous handlers
// - Generic handlers
// - Static file serving

func main() {
	r := gin.Default()

	// Standard route - should work
	r.GET("/api/health", healthCheck)

	// Wildcard route - should be skipped or handled gracefully
	r.GET("/api/files/*filepath", serveFiles)

	// Inline anonymous handler - extractor may not handle this well
	r.GET("/api/inline", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "inline handler"})
	})

	// Another inline handler with complex logic
	r.POST("/api/inline-complex", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Complex processing logic
		result := processData(data)
		c.JSON(http.StatusOK, result)
	})

	// Route group with middleware
	authorized := r.Group("/api/admin")
	authorized.Use(authMiddleware())
	{
		authorized.GET("/dashboard", dashboardHandler)
	}

	r.Run()
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func serveFiles(c *gin.Context) {
	filepath := c.Param("filepath")
	c.JSON(http.StatusOK, gin.H{"filepath": filepath})
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

func dashboardHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"dashboard": "data"})
}

func processData(data map[string]interface{}) map[string]interface{} {
	// Some processing logic
	return map[string]interface{}{
		"processed": true,
		"input":     data,
	}
}
