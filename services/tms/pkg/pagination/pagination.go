package pagination

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/gin-gonic/gin"
)

func Params(c *gin.Context) (*Info, error) {
	return &Info{
		Offset: ClampOffset(helpers.QueryInt(c, "offset", DefaultOffset)),
		Limit:  helpers.QueryBounded(c, "limit", 1, MaxLimit, DefaultLimit),
	}, nil
}

func buildPageURL(req *http.Request, offset, limit int) string {
	query := req.URL.Query()

	query.Set("offset", strconv.Itoa(offset))
	query.Set("limit", strconv.Itoa(limit))

	scheme := "http"
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s?%s", scheme, req.Host, req.URL.Path, query.Encode())
}

func GetNextPageURL(c *gin.Context, limit, offset int, hasNextPage bool) string {
	if !hasNextPage {
		return ""
	}

	return buildPageURL(c.Request, offset+limit, limit)
}

func HasMore[T any](result *ListResult[T], page Info) bool {
	read := len(result.Items)
	if result.TotalExact {
		return page.SafeOffset()+read < result.Total
	}

	return read >= page.SafeLimit()
}

func GetPreviousPageURL(c *gin.Context, limit, offset int) string {
	if offset == 0 {
		return ""
	}

	return buildPageURL(c.Request, max(offset-limit, 0), limit)
}

func buildCursorPageURL(req *http.Request, after string, limit int) string {
	query := req.URL.Query()

	query.Set("after", after)
	query.Set("limit", strconv.Itoa(limit))
	query.Del("offset")

	scheme := "http"
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s?%s", scheme, req.Host, req.URL.Path, query.Encode())
}

func GetNextCursorPageURL(c *gin.Context, after string, limit int, hasNextPage bool) string {
	if !hasNextPage || after == "" {
		return ""
	}

	return buildCursorPageURL(c.Request, after, limit)
}
