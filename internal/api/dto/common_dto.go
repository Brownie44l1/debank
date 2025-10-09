package dto

// ==============================================
// COMMON RESPONSE DTOs
// ==============================================

// ErrorResponse - Standard error format
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// SuccessResponse - Generic success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginationRequest - Common pagination parameters
type PaginationRequest struct {
	Page    int `form:"page" binding:"omitempty,min=1"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

// PaginationMeta - Pagination metadata
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// HealthResponse - API health check
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version,omitempty"`
}