package forms

import (
	"net/url"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type TemplateListParams struct {
	IDs   []string
	After string
}

func (p TemplateListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetString(values, "after", p.After)
	return values
}

type SubmissionListParams struct {
	IDs     []string
	Include []string
}

func (p SubmissionListParams) Query() url.Values {
	values := url.Values{}
	httpx.SetStringsCSV(values, "ids", p.IDs)
	httpx.SetStringsCSV(values, "include", p.Include)
	return values
}
