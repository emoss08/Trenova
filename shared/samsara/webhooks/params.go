package webhooks

import (
	"net/url"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type ListParams struct {
	IDs   []string
	Limit int
	After string
}

func (p ListParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	return nil
}

func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetString(values, "after", p.After)
	return values
}
