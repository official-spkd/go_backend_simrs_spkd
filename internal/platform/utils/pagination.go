package utils

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

func ParsePagination(c *gin.Context) (page, limit int) {
	page = queryInt(c, "page", defaultPage)
	limit = min(queryInt(c, "limit", defaultLimit), maxLimit)
	return page, limit
}

func queryInt(c *gin.Context, key string, fallback int) int {
	if v, err := strconv.Atoi(c.Query(key)); err == nil && v > 0 {
		return v
	}
	return fallback
}
