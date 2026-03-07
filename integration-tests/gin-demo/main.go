package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id" binding:"required"`
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age,omitempty"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age,omitempty"`
}

func main() {
	r := gin.Default()

	// Direct routes
	r.GET("/users", listUsers)
	r.POST("/users", createUser)

	// Route group
	api := r.Group("/api/v1")
	api.GET("/users/:id", getUser)
	api.PUT("/users/:id", updateUser)
	api.DELETE("/users/:id", deleteUser)

	r.Run()
}

func listUsers(c *gin.Context) {
	page := c.Query("page")
	size := c.DefaultQuery("size", "10")
	_ = page
	_ = size
	c.JSON(http.StatusOK, []User{})
}

func createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, User{ID: 1, Name: req.Name})
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	_ = id
	c.JSON(http.StatusOK, User{ID: 1, Name: "Test"})
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	_ = id
	var req CreateUserRequest
	c.ShouldBindJSON(&req)
	c.JSON(http.StatusOK, User{ID: 1})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")
	_ = id
	c.Status(http.StatusNoContent)
}
