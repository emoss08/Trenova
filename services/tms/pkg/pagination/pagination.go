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

func GetNextPageURL(c *gin.Context, limit, offset, totalRows int) string {
	// if the next page is the last page, return an empty string
	if offset+limit >= totalRows {
		return ""
	}

	return buildPageURL(c.Request, offset+limit, limit)
}

func GetPreviousPageURL(c *gin.Context, limit, offset int) string {
	if offset == 0 {
		return ""
	}

	return buildPageURL(c.Request, max(offset-limit, 0), limit)
}
