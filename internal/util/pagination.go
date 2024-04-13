package util

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func PaginationParams(c *fiber.Ctx) (int, int, error) {
	// Default values
	defaultOffset := 0
	defaultLimit := 10

	offsetStr := c.Query("offset")
	limitStr := c.Query("limit")

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

func buildPageURL(c *fiber.Ctx, offset, limit int) string {
	query := c.Request().URI().QueryArgs()
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))

	scheme := "http"
	if c.Protocol() == "https" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s?%s", scheme, c.Hostname(), c.Path(), query.QueryString())
}

func GetNextPageURL(c *fiber.Ctx, limit, offset, totalRows int) string {
	if offset+limit >= totalRows {
		return ""
	}
	return buildPageURL(c, offset+limit, limit)
}

func GetPrevPageURL(c *fiber.Ctx, limit, offset int) string {
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
	return buildPageURL(c, prevOffset, limit)
}
