// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package utils

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
