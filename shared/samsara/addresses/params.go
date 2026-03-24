package addresses

import (
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type ListParams struct {
	Limit            int
	After            string
	ParentTagIDs     []string
	TagIDs           []string
	CreatedAfterTime *time.Time
}

//nolint:gocritic // value receiver keeps call sites simple for immutable params.
func (p ListParams) Validate() error {
	if p.Limit == 0 {
		return nil
	}
	if p.Limit < 1 || p.Limit > 512 {
		return ErrListLimitOutOfRange
	}
	return nil
}

//nolint:gocritic // value receiver keeps call sites simple for immutable params.
func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetTime(values, "createdAfterTime", p.CreatedAfterTime)
	return values
}
