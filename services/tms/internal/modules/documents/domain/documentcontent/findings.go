package documentcontent

import (
	"cmp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/jsonutils"
)

const structuredIntelligenceKey = "intelligence"

type Findings struct {
	Kind          string          `json:"kind,omitempty"`
	ReviewStatus  string          `json:"reviewStatus,omitempty"`
	Fields        []FoundField    `json:"fields,omitempty"`
	Stops         []FoundStop     `json:"stops,omitempty"`
	MissingFields []string        `json:"missingFields,omitempty"`
	Conflicts     []FoundConflict `json:"conflicts,omitempty"`
}

type FoundField struct {
	Key            string  `json:"key"`
	Label          string  `json:"label"`
	Value          string  `json:"value"`
	Confidence     float64 `json:"confidence,omitempty"`
	PageNumber     int     `json:"pageNumber,omitempty"`
	ReviewRequired bool    `json:"reviewRequired,omitempty"`
}

type FoundStop struct {
	Sequence            int     `json:"sequence"`
	Role                string  `json:"role"`
	Name                string  `json:"name,omitempty"`
	AddressLine1        string  `json:"addressLine1,omitempty"`
	AddressLine2        string  `json:"addressLine2,omitempty"`
	City                string  `json:"city,omitempty"`
	State               string  `json:"state,omitempty"`
	PostalCode          string  `json:"postalCode,omitempty"`
	Date                string  `json:"date,omitempty"`
	TimeWindow          string  `json:"timeWindow,omitempty"`
	AppointmentRequired bool    `json:"appointmentRequired,omitempty"`
	PageNumber          int     `json:"pageNumber,omitempty"`
	Confidence          float64 `json:"confidence,omitempty"`
	ReviewRequired      bool    `json:"reviewRequired,omitempty"`
}

type FoundConflict struct {
	Label       string   `json:"label"`
	Values      []string `json:"values"`
	PageNumbers []int    `json:"pageNumbers,omitempty"`
}

type storedIntelligence struct {
	Kind          string                `json:"kind"`
	ReviewStatus  string                `json:"reviewStatus"`
	Fields        map[string]FoundField `json:"fields"`
	Stops         []FoundStop           `json:"stops"`
	MissingFields []string              `json:"missingFields"`
	Conflicts     []FoundConflict       `json:"conflicts"`
}

func (c *Content) Findings() *Findings {
	raw, ok := c.StructuredData[structuredIntelligenceKey]
	if !ok || raw == nil {
		return nil
	}

	var stored storedIntelligence
	if err := jsonutils.Convert(raw, &stored); err != nil {
		return nil
	}

	fields := make([]FoundField, 0, len(stored.Fields))
	for key, field := range stored.Fields {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		field.Key = key
		fields = append(fields, field)
	}
	slices.SortFunc(fields, func(a, b FoundField) int { return cmp.Compare(a.Key, b.Key) })

	findings := &Findings{
		Kind:          stored.Kind,
		ReviewStatus:  stored.ReviewStatus,
		Fields:        fields,
		Stops:         stored.Stops,
		MissingFields: stored.MissingFields,
		Conflicts:     stored.Conflicts,
	}
	if findings.Kind == "" && len(findings.Fields) == 0 && len(findings.Stops) == 0 {
		return nil
	}

	return findings
}
