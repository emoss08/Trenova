package recordquerytools

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

type Applied struct {
	Field    string `json:"field"`
	Operator string `json:"op"`
	Value    any    `json:"value,omitempty"`
	Group    int    `json:"orGroup,omitempty"`
}

type Ignored struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type Truncation struct {
	Reason string `json:"reason"`
}

type Group struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type Envelope struct {
	Entity           string            `json:"entity"`
	Items            any               `json:"items,omitempty"`
	Total            *int              `json:"total,omitempty"`
	TotalEstimated   bool              `json:"totalEstimated,omitempty"`
	HasMore          bool              `json:"hasMore"`
	NextCursor       string            `json:"nextCursor,omitempty"`
	AppliedFilters   []Applied         `json:"appliedFilters"`
	IgnoredFilters   []Ignored         `json:"ignoredFilters,omitempty"`
	Truncated        *Truncation       `json:"truncated,omitempty"`
	OmittedForAccess []string          `json:"omittedForAccess,omitempty"`
	Groups           []Group           `json:"groups,omitempty"`
	Includes         map[string]any    `json:"includes,omitempty"`
	OmittedIncludes  map[string]string `json:"omittedIncludes,omitempty"`
	Columns          []string          `json:"columns,omitempty"`
	Note             string            `json:"note,omitempty"`
	inner            any
	extra            []any
}

func (e *Envelope) TaintedMarks() []agent.SourcedRef {
	marks := sourcedMarks(e.inner)
	for _, inner := range e.extra {
		marks = append(marks, sourcedMarks(inner)...)
	}

	return marks
}

func sourcedMarks(inner any) []agent.SourcedRef {
	if carrier, ok := inner.(agent.SourcedTaintCarrier); ok {
		return carrier.TaintedMarks()
	}
	carrier, ok := inner.(agent.TaintCarrier)
	if !ok {
		return nil
	}
	records := carrier.TaintedRecords()
	out := make([]agent.SourcedRef, 0, len(records))
	for _, record := range records {
		out = append(out, agent.SourcedRef{Ref: record})
	}

	return out
}

func (e *Envelope) PageRows() int {
	if page, ok := e.inner.(agenttoolports.PageCutter); ok {
		return page.PageRows()
	}

	return 0
}

func (e *Envelope) CutPage(rows int) (any, bool) {
	page, ok := e.inner.(agenttoolports.PageCutter)
	if !ok {
		return e, false
	}
	cut, cutOK := page.CutPage(rows)
	if !cutOK {
		return e, false
	}
	shorter := *e
	shorter.inner = cut
	fill(&shorter, cut)
	shorter.Truncated = &Truncation{Reason: "the page was cut to fit; nextCursor reads the rest"}

	return &shorter, true
}

func fill(envelope *Envelope, inner any) {
	envelope.inner = inner
	raw, err := sonic.Marshal(inner)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err = sonic.Unmarshal(raw, &decoded); err != nil {
		envelope.Items = inner
		return
	}
	envelope.Items = decoded["items"]
	if envelope.Items == nil {
		envelope.Items = decoded
	}
	if total, ok := decoded["total"].(float64); ok {
		count := int(total)
		envelope.Total = &count
	}
	envelope.TotalEstimated, _ = decoded["totalEstimated"].(bool)
	envelope.HasMore, _ = decoded["hasMore"].(bool)
	envelope.NextCursor, _ = decoded["nextCursor"].(string)
	envelope.Note, _ = decoded["note"].(string)
	envelope.OmittedForAccess = textList(decoded["withheldByAccess"])
	envelope.OmittedForAccess = append(envelope.OmittedForAccess, textList(decoded["withheldByPlan"])...)
	envelope.Columns = textList(decoded["columns"])
}

func textList(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if text, isText := item.(string); isText {
			out = append(out, text)
		}
	}

	return out
}
