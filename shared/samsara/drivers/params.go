package drivers

import (
	"net/url"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type ListParams struct {
	DriverActivationStatus string
	Limit                  int
	After                  string
	ParentTagIDs           []string
	TagIDs                 []string
	AttributeValueIDs      []string
	Attributes             []string
	UpdatedAfterTime       *time.Time
	CreatedAfterTime       *time.Time
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p ListParams) Validate() error {
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 512) {
		return ErrListLimitInvalid
	}
	return nil
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "driverActivationStatus", p.DriverActivationStatus)
	httpx.SetInt(values, "limit", p.Limit)
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "attributeValueIds", p.AttributeValueIDs)
	httpx.SetTime(values, "updatedAfterTime", p.UpdatedAfterTime)
	httpx.SetTime(values, "createdAfterTime", p.CreatedAfterTime)
	for _, attr := range p.Attributes {
		if attr != "" {
			values.Add("attributes", attr)
		}
	}
	return values
}
