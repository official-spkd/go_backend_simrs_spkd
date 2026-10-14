package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response is the standard envelope for every API response.
type Response struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"Operation completed successfully"`
	Data    any    `json:"data"`
}

// ErrorResponse documents the envelope returned for failures (data is always null).
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"something went wrong"`
	Data    any    `json:"data"`
}

// PaginatedResponse is the standard envelope for paginated lists.
type PaginatedResponse struct {
	Success    bool       `json:"success" example:"true"`
	Message    string     `json:"message" example:"Items retrieved successfully"`
	Pagination Pagination `json:"pagination"`
	Data       any        `json:"data"`
}

func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Response{Success: true, Message: message, Data: data})
}

func SuccessPaginated(c *gin.Context, message string, data any, p Pagination) {
	c.JSON(http.StatusOK, PaginatedResponse{Success: true, Message: message, Pagination: p, Data: data})
}

func AbortWithError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{Success: false, Message: message})
}
