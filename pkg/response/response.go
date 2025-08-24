package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse represents a standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page      int   `json:"page"`
	PerPage   int   `json:"per_page"`
	Total     int64 `json:"total"`
	TotalPage int   `json:"total_pages"`
}

// PaginationResponse represents a paginated API response
type PaginationResponse struct {
	Success    bool           `json:"success"`
	Message    string         `json:"message,omitempty"`
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

// ValidationError represents a validation error response
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Success sends a successful response
func Success(c *gin.Context, data interface{}, message ...string) {
	msg := "Success"
	if len(message) > 0 {
		msg = message[0]
	}
	
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
	})
}

// Created sends a created response (201)
func Created(c *gin.Context, data interface{}, message ...string) {
	msg := "Resource created successfully"
	if len(message) > 0 {
		msg = message[0]
	}
	
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
	})
}

// NoContent sends a no content response (204)
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest sends a bad request error response (400)
func BadRequest(c *gin.Context, message string, errors ...interface{}) {
	var errorData interface{}
	if len(errors) > 0 {
		errorData = errors[0]
	}
	
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Message: message,
		Error:   errorData,
	})
}

// Unauthorized sends an unauthorized error response (401)
func Unauthorized(c *gin.Context, message ...string) {
	msg := "Unauthorized"
	if len(message) > 0 {
		msg = message[0]
	}
	
	c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Message: msg,
	})
}

// Forbidden sends a forbidden error response (403)
func Forbidden(c *gin.Context, message ...string) {
	msg := "Forbidden"
	if len(message) > 0 {
		msg = message[0]
	}
	
	c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Message: msg,
	})
}

// NotFound sends a not found error response (404)
func NotFound(c *gin.Context, message ...string) {
	msg := "Resource not found"
	if len(message) > 0 {
		msg = message[0]
	}
	
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Message: msg,
	})
}

// Conflict sends a conflict error response (409)
func Conflict(c *gin.Context, message ...string) {
	msg := "Resource conflict"
	if len(message) > 0 {
		msg = message[0]
	}
	
	c.JSON(http.StatusConflict, APIResponse{
		Success: false,
		Message: msg,
	})
}

// UnprocessableEntity sends an unprocessable entity error response (422)
func UnprocessableEntity(c *gin.Context, message string, errors ...interface{}) {
	var errorData interface{}
	if len(errors) > 0 {
		errorData = errors[0]
	}
	
	c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Message: message,
		Error:   errorData,
	})
}

// InternalServerError sends an internal server error response (500)
func InternalServerError(c *gin.Context, message ...string) {
	msg := "Internal server error"
	if len(message) > 0 {
		msg = message[0]
	}
	
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Message: msg,
	})
}

// Paginated sends a paginated response
func Paginated(c *gin.Context, data interface{}, page, perPage int, total int64, message ...string) {
	msg := "Success"
	if len(message) > 0 {
		msg = message[0]
	}
	
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	
	c.JSON(http.StatusOK, PaginationResponse{
		Success: true,
		Message: msg,
		Data:    data,
		Pagination: PaginationMeta{
			Page:      page,
			PerPage:   perPage,
			Total:     total,
			TotalPage: totalPages,
		},
	})
}

// ValidationErrors sends validation error response
func ValidationErrors(c *gin.Context, errors []string) {
	var validationErrors []ValidationError
	for _, err := range errors {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "validation",
			Message: err,
		})
	}
	
	UnprocessableEntity(c, "Validation failed", validationErrors)
}