package util

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaginationParams struct {
	Page  int
	Limit int
}

type PaginatedResponse struct {
	Data  interface{} `json:"data"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
	Pages int         `json:"pages"`
}

func ParsePagination(c *gin.Context) PaginationParams {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 0 {
		page = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return PaginationParams{Page: page, Limit: limit}
}

func NewPaginatedResponse(data interface{}, total int64, params PaginationParams) PaginatedResponse {
	pages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return PaginatedResponse{
		Data:  data,
		Total: total,
		Page:  params.Page,
		Limit: params.Limit,
		Pages: pages,
	}
}
