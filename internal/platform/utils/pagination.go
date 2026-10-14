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

type Pagination struct {
	CurrentPage  int   `json:"current_page" example:"1"`
	ItemsPerPage int   `json:"items_per_page" example:"20"`
	TotalItems   int64 `json:"total_items" example:"42"`
	TotalPages   int   `json:"total_pages" example:"3"`
	HasPrevPage  bool  `json:"has_prev_page" example:"false"`
	HasNextPage  bool  `json:"has_next_page" example:"true"`
	PrevPage     *int  `json:"prev_page" example:"1"`
	NextPage     *int  `json:"next_page" example:"2"`
}

func NewPagination(page, limit int, total int64) Pagination {
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	p := Pagination{
		CurrentPage:  page,
		ItemsPerPage: limit,
		TotalItems:   total,
		TotalPages:   totalPages,
		HasPrevPage:  page > 1,
		HasNextPage:  page < totalPages,
	}
	if p.HasPrevPage {
		prev := page - 1
		p.PrevPage = &prev
	}
	if p.HasNextPage {
		next := page + 1
		p.NextPage = &next
	}
	return p
}
