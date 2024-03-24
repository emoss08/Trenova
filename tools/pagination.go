package tools

import (
	"fmt"
	"net/http"
	"strconv"
)

func PaginationParams(r *http.Request) (int, int, error) {
	query := r.URL.Query()

	// Default values
	defaultOffset := 0
	defaultLimit := 10

	offsetStr := query.Get("offset")
	limitStr := query.Get("limit")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = defaultOffset
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = defaultLimit
	}

	return offset, limit, nil
}

func buildPageURL(r *http.Request, offset, limit int) string {
	query := r.URL.Query()
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s?%s", scheme, r.Host, r.URL.Path, query.Encode())
}

func GetNextPageURL(r *http.Request, limit, offset, totalRows int) string {
	if offset+limit >= totalRows {
		return ""
	}
	return buildPageURL(r, offset+limit, limit)
}

func GetPrevPageURL(r *http.Request, limit, offset int) string {
	if offset == 0 {
		return ""
	}
	prevOffset := offset - limit
	if prevOffset < 0 {
		prevOffset = 0
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return buildPageURL(r, prevOffset, limit)
}
