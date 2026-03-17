package main

import (
	"gin-multifile/handlers"
	"gin-multifile/models"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// User routes
	userGroup := r.Group("/api/users")
	{
		userGroup.GET("", handlers.ListUsers)
		userGroup.POST("", handlers.CreateUser)
		userGroup.GET("/:id", handlers.GetUser)
		userGroup.PUT("/:id", handlers.UpdateUser)
		userGroup.DELETE("/:id", handlers.DeleteUser)
	}

	// Product routes
	productGroup := r.Group("/api/products")
	{
		productGroup.GET("", handlers.ListProducts)
		productGroup.POST("", handlers.CreateProduct)
	}

	r.Run(":" + models.DefaultPort)
}
