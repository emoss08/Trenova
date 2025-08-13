package validation

import (
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// JSON-driven schema for partner/version-specific validation rules.

type Schema struct {
	Version     string         `json:"version"`
	Transaction string         `json:"transaction"`
	Segment     []SegRule      `json:"segment_rules"`
	Element     []ElemRule     `json:"element_rules"`
	Presence    []PresenceRule `json:"presence_rules"`
}

type SeverityString string

func (s SeverityString) ToSeverity() Severity {
	switch strings.ToUpper(string(s)) {
	case "WARN", "WARNING":
		return Warning
	default:
		return Error
	}
}

type SegRule struct {
	Tag      string         `json:"tag"`
	Min      int            `json:"min"`
	Max      int            `json:"max"` // -1 for unbounded
	Severity SeverityString `json:"severity,omitempty"`
}

type ElemRule struct {
	Tag      string         `json:"tag"`
	Element  int            `json:"element"` // 1-based index (e.g., B2-02)
	Required bool           `json:"required,omitempty"`
	Allowed  []string       `json:"allowed_values,omitempty"`
	MinLen   *int           `json:"min_len,omitempty"`
	MaxLen   *int           `json:"max_len,omitempty"`
	Versions []string       `json:"applies_to_versions,omitempty"`
	Severity SeverityString `json:"severity,omitempty"`
	When     []Condition    `json:"when,omitempty"` // apply rule only when all conditions match (usually same segment)
}

type Condition struct {
	Tag     string `json:"tag"`
	Element int    `json:"element"` // 1-based; 0 means just presence of tag
	Equals  string `json:"equals,omitempty"`
}

// PresenceRule requires each condition to occur at least EachMin times.
// Example: require both LD and UL stops at least once each.
type PresenceRule struct {
	Description string         `json:"description,omitempty"`
	Conditions  []Condition    `json:"conditions"`
	EachMin     int            `json:"each_min"`
	Severity    SeverityString `json:"severity,omitempty"`
}

func LoadSchema(path string) (*Schema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Schema
	if err := sonic.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ValidateWithSchema runs validations defined in a JSON Schema.
func ValidateWithSchema(segs []x12.Segment, sch *Schema) []Issue {
	issues := make([]Issue, 0)
	// Segment rules
	for _, r := range sch.Segment {
		count := 0
		for _, s := range segs {
			if strings.EqualFold(s.Tag, r.Tag) {
				count++
			}
		}
		if r.Min > 0 && count < r.Min {
			issues = append(
				issues,
				Issue{
					Severity: r.Severity.ToSeverity(),
					Code:     fmt.Sprintf("%s.MIN", r.Tag),
					Message:  fmt.Sprintf("at least %d %s required", r.Min, r.Tag),
				},
			)
		}
		if r.Max >= 0 && count > r.Max {
			issues = append(
				issues,
				Issue{
					Severity: r.Severity.ToSeverity(),
					Code:     fmt.Sprintf("%s.MAX", r.Tag),
					Message:  fmt.Sprintf("no more than %d %s allowed", r.Max, r.Tag),
				},
			)
		}
	}

	// Element rules
	for _, r := range sch.Element {
		if len(r.Versions) > 0 {
			ver := x12.ExtractVersion(segs)
			ok := false
			for _, v := range r.Versions {
				if v == ver {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		for i, s := range segs {
			if !strings.EqualFold(s.Tag, r.Tag) {
				continue
			}
			// Evaluate conditional guards (within same segment by default)
			if len(r.When) > 0 {
				matchedAll := true
				for _, c := range r.When {
					// If Tag not set or matches current, check within this segment; otherwise skip this rule for now.
					if c.Tag != "" && !strings.EqualFold(c.Tag, s.Tag) {
						matchedAll = false
						break
					}
					if c.Element <= 0 { // presence of tag is implied by reaching here
						continue
					}
					idxc := c.Element - 1
					valc := ""
					if idxc >= 0 && idxc < len(s.Elements) && len(s.Elements[idxc]) > 0 {
						valc = s.Elements[idxc][0]
					}
					if c.Equals != "" && valc != c.Equals {
						matchedAll = false
						break
					}
				}
				if !matchedAll {
					continue
				}
			}
			idx := r.Element - 1
			val := ""
			if idx >= 0 && idx < len(s.Elements) && len(s.Elements[idx]) > 0 {
				val = s.Elements[idx][0]
			}
			if r.Required && strings.TrimSpace(val) == "" {
				issues = append(
					issues,
					Issue{
						Severity:     r.Severity.ToSeverity(),
						Code:         fmt.Sprintf("%s-%02d.REQUIRED", r.Tag, r.Element),
						Message:      fmt.Sprintf("%s-%02d is required", r.Tag, r.Element),
						SegmentIndex: i,
						Tag:          s.Tag,
						ElementIndex: r.Element,
					},
				)
				continue
			}
			if len(r.Allowed) > 0 && val != "" {
				ok := false
				for _, a := range r.Allowed {
					if a == val {
						ok = true
						break
					}
				}
				if !ok {
					issues = append(
						issues,
						Issue{
							Severity: r.Severity.ToSeverity(),
							Code:     fmt.Sprintf("%s-%02d.VALUE", r.Tag, r.Element),
							Message: fmt.Sprintf(
								"%s-%02d value '%s' not allowed",
								r.Tag,
								r.Element,
								val,
							),
							SegmentIndex: i,
							Tag:          s.Tag,
							ElementIndex: r.Element,
						},
					)
				}
			}
			if r.MinLen != nil && len(val) > 0 && len(val) < *r.MinLen {
				issues = append(
					issues,
					Issue{
						Severity: r.Severity.ToSeverity(),
						Code:     fmt.Sprintf("%s-%02d.MINLEN", r.Tag, r.Element),
						Message: fmt.Sprintf(
							"%s-%02d length < %d",
							r.Tag,
							r.Element,
							*r.MinLen,
						),
						SegmentIndex: i,
						Tag:          s.Tag,
						ElementIndex: r.Element,
					},
				)
			}
			if r.MaxLen != nil && len(val) > *r.MaxLen {
				issues = append(
					issues,
					Issue{
						Severity: r.Severity.ToSeverity(),
						Code:     fmt.Sprintf("%s-%02d.MAXLEN", r.Tag, r.Element),
						Message: fmt.Sprintf(
							"%s-%02d length > %d",
							r.Tag,
							r.Element,
							*r.MaxLen,
						),
						SegmentIndex: i,
						Tag:          s.Tag,
						ElementIndex: r.Element,
					},
				)
			}
		}
	}

	// Presence rules
	for _, pr := range sch.Presence {
		for _, c := range pr.Conditions {
			count := 0
			for _, s := range segs {
				if !strings.EqualFold(s.Tag, c.Tag) {
					continue
				}
				if c.Element <= 0 {
					count++
					continue
				}
				idx := c.Element - 1
				val := ""
				if idx >= 0 && idx < len(s.Elements) && len(s.Elements[idx]) > 0 {
					val = s.Elements[idx][0]
				}
				if c.Equals == "" || val == c.Equals {
					count++
				}
			}
			if count < pr.EachMin {
				code := fmt.Sprintf("PRESENCE.%s-%02d", c.Tag, c.Element)
				msg := pr.Description
				if msg == "" {
					msg = fmt.Sprintf(
						"require at least %d occurrence(s) of %s where element %d == '%s'",
						pr.EachMin,
						c.Tag,
						c.Element,
						c.Equals,
					)
				}
				issues = append(
					issues,
					Issue{Severity: pr.Severity.ToSeverity(), Code: code, Message: msg},
				)
			}
		}
	}
	return issues
}
