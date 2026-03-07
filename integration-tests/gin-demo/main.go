package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// User represents a user in the system
type User struct {
	ID       int64  `json:"id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"fullName" binding:"required"`
	Age      int    `json:"age"`
}

// ApiResponse is a generic API response wrapper (using interface{} for compatibility)
type ApiResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// PageResult is a pagination result wrapper
type PageResult struct {
	Content    []User `json:"content"`
	PageNumber int    `json:"pageNumber"`
	PageSize   int    `json:"pageSize"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"totalPages"`
}

// FileUploadResult represents file upload result
type FileUploadResult struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	Message     string `json:"message"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"fullName" binding:"required"`
	Age      int    `json:"age"`
}

// ListUsersRequest represents query params for listing users
type ListUsersRequest struct {
	Page     int    `form:"page,default=0"`
	Size     int    `form:"size,default=10"`
	Username string `form:"username"`
}

// UpdateProfileRequest represents form data for updating profile
type UpdateProfileRequest struct {
	FullName string `form:"fullName"`
	Email    string `form:"email"`
	Age      int    `form:"age"`
}

// In-memory user store
var userStore = map[int64]User{
	1: {ID: 1, Username: "john_doe", Email: "john@example.com", FullName: "John Doe", Age: 30},
	2: {ID: 2, Username: "jane_smith", Email: "jane@example.com", FullName: "Jane Smith", Age: 25},
	3: {ID: 3, Username: "bob_wilson", Email: "bob@example.com", FullName: "Bob Wilson", Age: 35},
}

func main() {
	r := gin.Default()

	// API v1 routes
	api := r.Group("/api/v1")
	{
		// User routes
		api.GET("/users/:id", getUserByID)
		api.GET("/users", listUsers)
		api.POST("/users", createUser)
		api.POST("/users/upload", uploadFile)
		api.POST("/users/:id/profile", updateProfile)
	}

	r.Run()
}

// getUserByID gets a user by ID
func getUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Code:    400,
			Message: "Invalid user ID",
		})
		return
	}

	user, exists := userStore[id]
	if !exists {
		c.JSON(http.StatusNotFound, ApiResponse{
			Code:    404,
			Message: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, ApiResponse{
		Code:    200,
		Message: "Success",
		Data:    user,
	})
}

// listUsers lists users with pagination
func listUsers(c *gin.Context) {
	var req ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Code:    400,
			Message: "Invalid query parameters",
		})
		return
	}

	// Get all users
	allUsers := make([]User, 0, len(userStore))
	for _, user := range userStore {
		// Filter by username if provided
		if req.Username != "" && user.Username != req.Username {
			continue
		}
		allUsers = append(allUsers, user)
	}

	// Pagination
	total := int64(len(allUsers))
	start := req.Page * req.Size
	end := start + req.Size
	if start > len(allUsers) {
		start = len(allUsers)
	}
	if end > len(allUsers) {
		end = len(allUsers)
	}

	pageContent := allUsers[start:end]
	totalPages := 0
	if req.Size > 0 {
		totalPages = (len(allUsers) + req.Size - 1) / req.Size
	}

	result := PageResult{
		Content:    pageContent,
		PageNumber: req.Page,
		PageSize:   req.Size,
		Total:      total,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, ApiResponse{
		Code:    200,
		Message: "Success",
		Data:    result,
	})
}

// createUser creates a new user
func createUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Generate new ID
	var newID int64 = 1
	for id := range userStore {
		if id >= newID {
			newID = id + 1
		}
	}

	user := User{
		ID:       newID,
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
		Age:      req.Age,
	}
	userStore[newID] = user

	c.JSON(http.StatusCreated, ApiResponse{
		Code:    201,
		Message: "User created successfully",
		Data:    user,
	})
}

// uploadFile handles file upload
func uploadFile(c *gin.Context) {
	// Get the file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Code:    400,
			Message: "No file uploaded: " + err.Error(),
		})
		return
	}

	// Get optional userId
	userID := c.PostForm("userId")
	message := "File uploaded successfully"
	if userID != "" {
		message = "File uploaded successfully for user: " + userID
	}

	result := FileUploadResult{
		Filename:    file.Filename,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
		Message:     message,
	}

	c.JSON(http.StatusOK, ApiResponse{
		Code:    200,
		Message: "Success",
		Data:    result,
	})
}

// updateProfile updates user profile using form data
func updateProfile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Code:    400,
			Message: "Invalid user ID",
		})
		return
	}

	user, exists := userStore[id]
	if !exists {
		c.JSON(http.StatusNotFound, ApiResponse{
			Code:    404,
			Message: "User not found",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, ApiResponse{
			Code:    400,
			Message: "Invalid form data: " + err.Error(),
		})
		return
	}

	// Update fields if provided
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Age > 0 {
		user.Age = req.Age
	}

	userStore[id] = user

	c.JSON(http.StatusOK, ApiResponse{
		Code:    200,
		Message: "Profile updated successfully",
		Data:    user,
	})
}
