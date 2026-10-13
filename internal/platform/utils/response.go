package utils

import "github.com/gin-gonic/gin"

type ErrorBody struct {
	Error string `json:"error" example:"something went wrong"`
}

func AbortWithError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: message})
}
