package main

import (
	customapis "gin-crosspkg/apis"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/ping", customapis.Ping)
	r.POST("/projects", customapis.CreateProject)
	r.GET("/projects/:name", customapis.GetProject)
	r.PUT("/projects/:name", customapis.UpdateProject)
	r.DELETE("/projects/:name", customapis.DeleteProject)

	r.Run(":8080")
}
