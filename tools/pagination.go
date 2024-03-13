package tools

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func PaginationParams(r *http.Request) (int, int, error) {
	query := r.URL.Query()

	offsetStr := query.Get("offset")
	if offsetStr == "" {
		offsetStr = "0"
	}

	limitStr := query.Get("limit")
	if limitStr == "" {
		limitStr = "10"
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		return 0, 0, fmt.Errorf("invalid offset value: %s", offsetStr)
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit > 100 {
		return 0, 0, fmt.Errorf("invalid limit value: %s", limitStr)
	}

	return offset, limit, nil
}

func buildPageURL(r *http.Request, offset, limit int) string {
	var queryString string
	for key, values := range r.URL.Query() {
		if key != "offset" && key != "limit" {
			for _, value := range values {
				queryString += fmt.Sprintf("%s=%s&", key, value)
			}
		}
	}

	queryString += fmt.Sprintf("offset=%d&limit=%d", offset, limit)
	queryString = strings.TrimSuffix(queryString, "&")

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s?%s", scheme, r.Host, r.URL.Path, queryString)
}

func GetNextPageURL(r *http.Request, offset, limit int, totalRows int) string {
	if offset+limit >= totalRows {
		return ""
	}
	return buildPageURL(r, offset+limit, limit)
}

func GetPrevPageURL(r *http.Request, offset, limit int) string {
	if offset == 0 {
		return ""
	}
	prevOffset := maximumLimitAndOffset(0, offset-limit)
	return buildPageURL(r, prevOffset, limit)
}

func maximumLimitAndOffset(a, b int) int {
	if a > b {
		return a
	}
	return b
}
