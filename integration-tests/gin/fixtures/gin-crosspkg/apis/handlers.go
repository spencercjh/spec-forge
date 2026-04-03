package apis

import (
	"net/http"

	"gin-crosspkg/models"
	"github.com/gin-gonic/gin"
)

// Ping handles GET /ping
func Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}

// CreateProject handles POST /projects
func CreateProject(c *gin.Context) {
	var req models.CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, models.Project{
		Name:  req.ProjectName,
		Quota: req.Quota,
	})
}

// GetProject handles GET /projects/:name
func GetProject(c *gin.Context) {
	name := c.Param("name")
	tenant := c.Query("tenant")

	_ = name
	_ = tenant

	c.JSON(http.StatusOK, models.Project{
		Name:  name,
		Quota: 100,
	})
}

// UpdateProject handles PUT /projects/:name
func UpdateProject(c *gin.Context) {
	name := c.Param("name")

	var req models.UpdateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = name
	_ = req

	c.JSON(http.StatusOK, models.Project{
		Name:  name,
		Quota: req.Quota,
	})
}

// DeleteProject handles DELETE /projects/:name
func DeleteProject(c *gin.Context) {
	name := c.Param("name")
	_ = name

	c.Status(http.StatusNoContent)
}
