package ediconnectiondraft

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/templates"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type requirement struct {
	input  *ediports.DraftRequirementInput
	source ediports.DraftSource
}

type templateOption struct {
	templateID pulid.ID
	versionID  pulid.ID
	name       string
	starter    bool
	createdAt  int64
	segments   []*edi.EDITemplateSegment
}

func starterName(set edi.TransactionSet, direction edi.DocumentDirection) string {
	return fmt.Sprintf("Base X12 %s %s", set, direction)
}

func templateOptions(
	tenantInfo pagination.TenantInfo,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
	versions []*edi.EDITemplateVersion,
) []templateOption {
	options := make([]templateOption, 0, len(versions)+1)
	hasStarter := false
	for _, version := range versions {
		if version.Template == nil {
			continue
		}
		name := version.Template.Name
		hasStarter = hasStarter || name == starterName(set, direction)
		options = append(options, templateOption{
			templateID: version.TemplateID,
			versionID:  version.ID,
			name:       fmt.Sprintf("%s (version %d)", name, version.VersionNumber),
			starter:    name == starterName(set, direction),
			createdAt:  version.Template.CreatedAt,
			segments:   version.Segments,
		})
	}
	if !hasStarter {
		if segments, err := templates.StarterSegments(tenantInfo, pulid.Nil, set); err == nil {
			options = append(options, templateOption{
				name:     starterName(set, direction),
				starter:  true,
				segments: segments,
			})
		}
	}
	return options
}

func unmetRequirements(
	segments []*edi.EDITemplateSegment,
	requirements []requirement,
) []ediports.DraftRequirementCheck {
	bySegment := make(map[string][]*edi.EDITemplateSegment, len(segments))
	for _, segment := range segments {
		id := strings.ToUpper(segment.SegmentID)
		bySegment[id] = append(bySegment[id], segment)
	}

	unmet := make([]ediports.DraftRequirementCheck, 0)
	for reqIndex := range requirements {
		req := &requirements[reqIndex]
		if problem := checkRequirement(bySegment, req.input); problem != "" {
			unmet = append(unmet, ediports.DraftRequirementCheck{
				SegmentID:       strings.ToUpper(req.input.SegmentID),
				ElementPosition: req.input.ElementPosition,
				Usage:           req.input.Usage,
				Codes:           req.input.Codes,
				Problem:         problem,
				Source:          req.source,
			})
		}
	}
	return unmet
}

func checkRequirement(
	bySegment map[string][]*edi.EDITemplateSegment,
	req *ediports.DraftRequirementInput,
) string {
	segmentID := strings.ToUpper(strings.TrimSpace(req.SegmentID))
	found := bySegment[segmentID]
	ref := elementRef(segmentID, req.ElementPosition)

	if req.Usage == ediports.RequirementUsageNotUsed {
		if req.ElementPosition == 0 && len(found) > 0 {
			return fmt.Sprintf("The template writes %s, which the spec says is not used", segmentID)
		}
		if req.ElementPosition > 0 && elementIn(found, req.ElementPosition) != nil {
			return fmt.Sprintf("The template fills %s, which the spec says is not used", ref)
		}
		return ""
	}
	if len(found) == 0 {
		if req.Usage == ediports.RequirementUsageMandatory {
			return fmt.Sprintf("The template never writes %s", segmentID)
		}
		return ""
	}
	if req.ElementPosition == 0 {
		if req.Usage == ediports.RequirementUsageMandatory && !anyRequired(found) {
			return fmt.Sprintf(
				"The template writes %s only when it has data; the spec requires it",
				segmentID,
			)
		}
		return ""
	}

	element := elementIn(found, req.ElementPosition)
	if element == nil {
		if req.Usage == ediports.RequirementUsageMandatory {
			return fmt.Sprintf("The template leaves %s empty", ref)
		}
		return ""
	}
	if len(req.Codes) > 0 && element.Source == edi.TemplateElementSourceConstant &&
		element.Value != "" && !containsFold(req.Codes, element.Value) {
		return fmt.Sprintf(
			"The template writes %s as %q; the spec allows %s",
			ref,
			element.Value,
			strings.Join(req.Codes, ", "),
		)
	}
	return ""
}

func elementRef(segmentID string, position int) string {
	if position == 0 {
		return segmentID
	}
	return fmt.Sprintf("%s%02d", segmentID, position)
}

func elementIn(segments []*edi.EDITemplateSegment, position int) *edi.TemplateElement {
	for _, segment := range segments {
		for index := range segment.Elements {
			if segment.Elements[index].Position == position {
				return &segment.Elements[index]
			}
		}
	}
	return nil
}

func anyRequired(segments []*edi.EDITemplateSegment) bool {
	for _, segment := range segments {
		if segment.Required {
			return true
		}
	}
	return false
}

func containsFold(values []string, value string) bool {
	return slices.ContainsFunc(values, func(candidate string) bool {
		return strings.EqualFold(strings.TrimSpace(candidate), value)
	})
}

func chooseTemplate(
	options []templateOption,
	requirements []requirement,
	currentTemplateID pulid.ID,
) (templateOption, []ediports.DraftRequirementCheck, []ediports.DraftTemplateCandidate) {
	type scored struct {
		option templateOption
		unmet  []ediports.DraftRequirementCheck
	}
	scores := make([]scored, 0, len(options))
	considered := make([]ediports.DraftTemplateCandidate, 0, len(options))
	for _, option := range options {
		unmet := unmetRequirements(option.segments, requirements)
		scores = append(scores, scored{option: option, unmet: unmet})
		considered = append(considered, ediports.DraftTemplateCandidate{
			TemplateID: option.templateID,
			VersionID:  option.versionID,
			Name:       option.name,
			Starter:    option.starter,
			Unmet:      len(unmet),
		})
	}

	better := func(a, b scored) bool {
		if len(a.unmet) != len(b.unmet) {
			return len(a.unmet) < len(b.unmet)
		}
		aCurrent := currentTemplateID.IsNotNil() && a.option.templateID == currentTemplateID
		bCurrent := currentTemplateID.IsNotNil() && b.option.templateID == currentTemplateID
		if aCurrent != bCurrent {
			return aCurrent
		}
		if a.option.starter != b.option.starter {
			return !a.option.starter
		}
		return a.option.createdAt > b.option.createdAt
	}

	best := scores[0]
	for _, candidate := range scores[1:] {
		if better(candidate, best) {
			best = candidate
		}
	}
	return best.option, best.unmet, considered
}
