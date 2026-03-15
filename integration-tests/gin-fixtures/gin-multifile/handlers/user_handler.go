package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"gin-multifile/models"
)

// ListUsers handles GET /api/users
func ListUsers(c *gin.Context) {
	c.JSON(http.StatusOK, []models.User{
		{ID: 1, Username: "user1", Email: "user1@example.com"},
	})
}

// CreateUser handles POST /api/users
func CreateUser(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		ID:       1,
		Username: req.Username,
		Email:    req.Email,
	}
	c.JSON(http.StatusCreated, user)
}

// GetUser handles GET /api/users/:id
func GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	c.JSON(http.StatusOK, models.User{
		ID:       id,
		Username: "user1",
		Email:    "user1@example.com",
	})
}

// UpdateUser handles PUT /api/users/:id
func UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.User{
		ID:       id,
		Username: req.Username,
		Email:    req.Email,
	})
}

// DeleteUser handles DELETE /api/users/:id
func DeleteUser(c *gin.Context) {
	_, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	c.Status(http.StatusNoContent)
}
