package ediconnectiondraft

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	MaxValues       = 300
	MaxRequirements = 300
	MaxMappings     = 500
	MaxGaps         = 100
	MaxResolutions  = 100
	MaxSummaryRunes = 500
)

func DocumentDigest(pages []ediports.SpecPage) string {
	ordered := slices.Clone(pages)
	slices.SortStableFunc(ordered, func(a, b ediports.SpecPage) int {
		return a.Number - b.Number
	})
	hash := sha256.New()
	for _, page := range ordered {
		hash.Write([]byte("page " + strconv.Itoa(page.Number) + "\n"))
		hash.Write([]byte(page.Text))
		hash.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func validateRequest(req *ediports.ConnectionDraftRequest) error {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("", errortypes.ErrRequired, "A draft request is required")
		return multiErr
	}
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		multiErr.Add("tenant", errortypes.ErrRequired, "The organization is required")
	}
	if req.Document.ID.IsNil() {
		multiErr.Add("documentId", errortypes.ErrRequired, "The spec document is required")
	}
	if len(req.Document.Pages) == 0 {
		multiErr.Add(
			"documentId",
			errortypes.ErrInvalidOperation,
			"The spec document has no readable text yet",
		)
	}
	limit(multiErr, "values", len(req.Values), MaxValues)
	limit(multiErr, "requirements", len(req.Requirements), MaxRequirements)
	limit(multiErr, "mappings", len(req.Mappings), MaxMappings)
	limit(multiErr, "gaps", len(req.Gaps), MaxGaps)
	limit(multiErr, "resolutions", len(req.Resolutions), MaxResolutions)

	for index, value := range req.Values {
		checkFieldKey(multiErr.WithIndex("values", index), value.Field)
	}
	for index, resolution := range req.Resolutions {
		checkFieldKey(multiErr.WithIndex("resolutions", index), resolution.Field)
	}
	for index := range req.Requirements {
		requirement := &req.Requirements[index]
		checkSetDirection(
			multiErr.WithIndex("requirements", index),
			requirement.TransactionSet,
			requirement.Direction,
		)
		if strings.TrimSpace(requirement.SegmentID) == "" {
			multiErr.WithIndex("requirements", index).
				Add("segmentId", errortypes.ErrRequired, "Name the segment the spec describes")
		}
		if !validUsage(requirement.Usage) {
			multiErr.WithIndex("requirements", index).
				Add("usage", errortypes.ErrInvalid, "Usage must be Mandatory, Optional, Conditional or NotUsed")
		}
	}
	for index, mapping := range req.Mappings {
		if !mapping.EntityType.IsValid() {
			multiErr.WithIndex("mappings", index).
				Add("entityType", errortypes.ErrInvalid, "Mapping entity type is invalid")
		}
	}
	for index, gap := range req.Gaps {
		if gap.Kind != ediports.GapKindUnsupported && gap.Kind != ediports.GapKindAmbiguous {
			multiErr.WithIndex("gaps", index).
				Add("kind", errortypes.ErrInvalid, "A gap is Unsupported or Ambiguous")
		}
		if summary := strings.TrimSpace(gap.Summary); summary == "" ||
			len([]rune(summary)) > MaxSummaryRunes {
			multiErr.WithIndex("gaps", index).Add(
				"summary",
				errortypes.ErrInvalid,
				fmt.Sprintf("Summarize the gap in 1 to %d characters", MaxSummaryRunes),
			)
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func limit(multiErr *errortypes.MultiError, field string, count, maximum int) {
	if count > maximum {
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			fmt.Sprintf("Send at most %d %s", maximum, field),
		)
	}
}

func checkFieldKey(multiErr *errortypes.MultiError, key string) {
	if isSecretField(key) {
		multiErr.Add(
			"field",
			errortypes.ErrInvalidOperation,
			"Secrets, keys and certificates are never taken from a document; "+
				"leave them for the person to enter",
		)
		return
	}
	if _, ok := fieldCatalog[key]; !ok {
		multiErr.Add(
			"field",
			errortypes.ErrInvalid,
			fmt.Sprintf("Unknown field %q", key),
		)
	}
}

func checkSetDirection(
	multiErr *errortypes.MultiError,
	set edi.TransactionSet,
	direction edi.DocumentDirection,
) {
	if !slices.Contains(supportedSets, set) {
		multiErr.Add(
			"transactionSet",
			errortypes.ErrInvalid,
			"Transaction set must be 204, 210, 214 or 990",
		)
	}
	if !slices.Contains(directions, direction) {
		multiErr.Add("direction", errortypes.ErrInvalid, "Direction must be Inbound or Outbound")
	}
}

func validUsage(usage ediports.RequirementUsage) bool {
	switch usage {
	case ediports.RequirementUsageMandatory,
		ediports.RequirementUsageOptional,
		ediports.RequirementUsageConditional,
		ediports.RequirementUsageNotUsed:
		return true
	default:
		return false
	}
}
