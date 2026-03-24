package liveshares

import (
	"fmt"
	"net/url"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type ListParams struct {
	IDs   []string
	Type  ListType
	Limit int
	After string
}

func (p ListParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 100) {
		return ErrListLimitOutOfRange
	}
	if p.Type == "" {
		return nil
	}
	switch p.Type {
	case ListTypeAll, ListTypeAssetsLocation, ListTypeAssetsNearLocation, ListTypeAssetsOnRoute:
		return nil
	default:
		return fmt.Errorf("invalid live share list type: %s", p.Type)
	}
}

func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetString(values, "type", string(p.Type))
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetString(values, "after", p.After)
	return values
}
