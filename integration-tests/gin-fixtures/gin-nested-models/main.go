package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Nested models fixture tests:
// - Deeply nested structs
// - Embedded structs
// - Slice of structs
// - Pointer fields
// - Custom types

// Address represents a physical address
type Address struct {
	Street  string `json:"street" binding:"required"`
	City    string `json:"city" binding:"required"`
	Country string `json:"country" binding:"required"`
	ZipCode string `json:"zipCode"`
}

// Company represents a company with nested address
type Company struct {
	ID        int64     `json:"id" binding:"required"`
	Name      string    `json:"name" binding:"required"`
	Address   Address   `json:"address" binding:"required"`
	Employees []Employee `json:"employees"`
	CreatedAt time.Time `json:"createdAt"`
}

// Employee represents an employee
type Employee struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name" binding:"required"`
	Email     string  `json:"email" binding:"required,email"`
	Department string `json:"department"`
	Manager   *Employee `json:"manager,omitempty"` // Pointer to another Employee (recursive)
}

// CreateCompanyRequest represents a request to create a company
type CreateCompanyRequest struct {
	Name    string  `json:"name" binding:"required"`
	Address Address `json:"address" binding:"required"`
}

// UpdateCompanyRequest represents a request to update a company
type UpdateCompanyRequest struct {
	Name    *string  `json:"name,omitempty"`    // Pointer field
	Address *Address `json:"address,omitempty"` // Pointer to nested struct
}

// PaginatedResponse is a generic paginated response
type PaginatedResponse struct {
	Data       []Company `json:"data"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
	TotalPages int       `json:"totalPages"`
}

// CustomString is a custom string type
type CustomString string

// Metadata contains arbitrary metadata
type Metadata struct {
	Tags    []CustomString     `json:"tags"`
	Extra   map[string]string  `json:"extra"`
}

// Resource represents a resource with metadata
type Resource struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Metadata Metadata `json:"metadata"`
}

func main() {
	r := gin.Default()

	// Company routes
	r.GET("/companies", listCompanies)
	r.POST("/companies", createCompany)
	r.GET("/companies/:id", getCompany)
	r.PUT("/companies/:id", updateCompany)
	r.GET("/companies/:id/employees", listCompanyEmployees)

	// Resource routes
	r.GET("/resources", listResources)
	r.POST("/resources", createResource)

	r.Run()
}

func listCompanies(c *gin.Context) {
	var query struct {
		Page     int    `form:"page,default=0"`
		PageSize int    `form:"pageSize,default=10"`
		Country  string `form:"country"`
	}

	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Data:       []Company{},
		Total:      0,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: 0,
	})
}

func createCompany(c *gin.Context) {
	var req CreateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	company := Company{
		ID:        1,
		Name:      req.Name,
		Address:   req.Address,
		CreatedAt: time.Now(),
	}
	c.JSON(http.StatusCreated, company)
}

func getCompany(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, Company{
		ID:   1,
		Name: "Company " + id,
		Address: Address{
			Street:  "123 Main St",
			City:    "New York",
			Country: "USA",
		},
	})
}

func updateCompany(c *gin.Context) {
	var req UpdateCompanyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, Company{
		ID: 1,
	})
}

func listCompanyEmployees(c *gin.Context) {
	// id := c.Param("id")
	c.JSON(http.StatusOK, []Employee{
		{
			ID:   1,
			Name: "John Doe",
			Manager: &Employee{
				ID:   2,
				Name: "Jane Smith",
			},
		},
	})
}

func listResources(c *gin.Context) {
	c.JSON(http.StatusOK, []Resource{
		{
			ID:   1,
			Name: "Resource 1",
			Metadata: Metadata{
				Tags:  []CustomString{"tag1", "tag2"},
				Extra: map[string]string{"key": "value"},
			},
		},
	})
}

func createResource(c *gin.Context) {
	var resource Resource
	if err := c.ShouldBindJSON(&resource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resource.ID = 1
	c.JSON(http.StatusCreated, resource)
}
