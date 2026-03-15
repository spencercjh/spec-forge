package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-multifile/models"
)

// ListProducts handles GET /api/products
func ListProducts(c *gin.Context) {
	var query struct {
		Category string `form:"category"`
		MinPrice float64 `form:"min_price"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, []models.Product{
		{ID: 1, Name: "Product 1", Price: 9.99, Category: query.Category},
	})
}

// CreateProduct handles POST /api/products
func CreateProduct(c *gin.Context) {
	var req models.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := models.Product{
		ID:          1,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Category:    req.Category,
	}
	c.JSON(http.StatusCreated, product)
}
