package limitoffsetpagination

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/internal/pkg/ctx"
	"github.com/trenova-app/transport/internal/pkg/validator"
)

func HandlePaginatedRequest[T any](
	c *fiber.Ctx,
	eh *validator.ErrorHandler,
	reqCtx *ctx.RequestContext,
	handler ports.PageableHandler[T],
) error {
	pg, err := Params(c)
	if err != nil {
		return eh.HandleError(c, err)
	}

	filter := &ports.LimitOffsetQueryOptions{
		TenantOpts: &ports.TenantOptions{
			OrgID:  reqCtx.OrgID,
			BuID:   reqCtx.BuID,
			UserID: reqCtx.UserID,
		},
		Limit:  pg.Limit,
		Offset: pg.Offset,
	}

	result, err := handler(c, filter)
	if err != nil {
		return eh.HandleError(c, err)
	}

	nextURL := GetNextPageURL(c, pg.Limit, pg.Offset, result.Total)
	prevURL := GetPrevPageURL(c, pg.Limit, pg.Offset)

	return c.JSON(ports.Response[[]T]{
		Count:   result.Total,
		Results: result.Items,
		Next:    nextURL,
		Prev:    prevURL,
	})
}

func Params(c *fiber.Ctx) (*Info, error) {
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

	return &Info{
		Offset: offset,
		Limit:  limit,
	}, nil
}

func buildPageURL(c *fiber.Ctx, offset, limit int) string {
	query := c.Request().URI().QueryArgs()
	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))

	scheme := "http"
	if c.Secure() {
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
