package models

// Project represents a project resource.
type Project struct {
	Name  string  `json:"name"`
	Quota float64 `json:"quota"`
}

// CreateProjectReq is the request body for creating a project.
type CreateProjectReq struct {
	ProjectName string  `json:"project_name" binding:"required"`
	Quota       float64 `json:"quota" binding:"required"`
}

// UpdateProjectReq is the request body for updating a project.
type UpdateProjectReq struct {
	Quota float64 `json:"quota" binding:"required"`
}
