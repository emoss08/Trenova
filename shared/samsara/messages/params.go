package messages

import (
	"net/url"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type ListParams struct {
	EndMs      int64
	DurationMs int64
}

func (p ListParams) Validate() error {
	if p.DurationMs < 0 {
		return ErrDurationInvalid
	}
	return nil
}

func (p ListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetInt64(values, "endMs", p.EndMs)
	httpx.SetInt64(values, "durationMs", p.DurationMs)
	return values
}
