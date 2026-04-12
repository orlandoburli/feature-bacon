package handlers

import (
	"net/http"
	"strconv"
)

const (
	defaultPage    = 1
	defaultPerPage = 25
	maxPerPage     = 100
)

type PaginationResponse struct {
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

func ParsePagination(r *http.Request) (page, perPage int) {
	page = defaultPage
	perPage = defaultPerPage

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := r.URL.Query().Get("perPage"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 {
			perPage = pp
		}
	}

	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return page, perPage
}
