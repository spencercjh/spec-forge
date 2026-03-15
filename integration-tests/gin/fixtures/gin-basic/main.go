package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Item represents a simple item
type Item struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func main() {
	r := gin.Default()

	// Basic routes
	r.GET("/items", listItems)
	r.POST("/items", createItem)
	r.GET("/items/:id", getItem)

	r.Run()
}

// listItems lists all items
func listItems(c *gin.Context) {
	c.JSON(http.StatusOK, []Item{
		{ID: 1, Name: "item1"},
	})
}

// createItem creates a new item
func createItem(c *gin.Context) {
	var item Item
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// getItem gets an item by ID
func getItem(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, Item{ID: 1, Name: id})
}
