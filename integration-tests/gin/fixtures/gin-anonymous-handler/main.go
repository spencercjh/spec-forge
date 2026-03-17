package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// This fixture focuses on anonymous handler patterns
// and how the extractor should handle them

func main() {
	r := gin.Default()

	// Simple inline handler
	r.GET("/simple", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"type": "simple"})
	})

	// Inline handler with query binding
	r.GET("/query", func(c *gin.Context) {
		var query struct {
			Name  string `form:"name"`
			Limit int    `form:"limit,default=10"`
		}
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, query)
	})

	// Inline handler with JSON body
	r.POST("/create", func(c *gin.Context) {
		var body struct {
			Title string `json:"title" binding:"required"`
			Desc  string `json:"description"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":    1,
			"title": body.Title,
		})
	})

	// Inline handler with path param
	r.GET("/item/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	// Multiple inline handlers in a group
	api := r.Group("/api")
	{
		api.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "up"})
		})

		api.POST("/data", func(c *gin.Context) {
			var data map[string]interface{}
			if err := c.ShouldBindJSON(&data); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, data)
		})
	}

	// Inline handler stored in variable
	updateHandler := func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"updated": id})
	}
	r.PUT("/update/:id", updateHandler)

	r.Run()
}
