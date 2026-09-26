package aidocumentservice

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/stringutils"
)

var _ serviceports.ExtractionContract = Contract{}

type Contract struct{}

func NewContract() serviceports.ExtractionContract { return Contract{} }

func (Contract) CompletionRequest(
	fileName string,
	pages []serviceports.AIDocumentPage,
) *serviceports.StructuredCompletionRequest {
	return newExtractCall(&serviceports.AIExtractRequest{
		FileName: fileName,
		Pages:    pages,
	}).request()
}

func (Contract) PageLimit() int { return extractPageLimit }

func (Contract) FieldKeys() []string { return slices.Clone(extractFieldKeys) }

func (Contract) ParseReply(text string) (*serviceports.AIExtractResult, error) {
	parsed := new(extractResponse)
	if err := decodeStructured(text, parsed); err != nil {
		return nil, err
	}

	return convertExtractResponse(parsed), nil
}

func (Contract) FormatReply(result *serviceports.AIExtractResult) (string, error) {
	reply := extractResponse{
		MissingFields: []string{},
		Signals:       []string{},
		Fields:        []extractFieldResponse{},
		Stops:         []*serviceports.AIDocumentStop{},
		Conflicts:     []*serviceports.AIDocumentConflict{},
	}
	if result != nil {
		reply.DocumentKind = result.DocumentKind
		reply.OverallConfidence = clampAIConfidence(result.OverallConfidence)
		reply.ReviewStatus = normalizeReviewStatus(result.ReviewStatus)
		reply.Fields = replyFields(result.Fields)
		reply.Stops = replyStops(result.Stops)
	}

	encoded, err := sonic.Marshal(&reply)
	if err != nil {
		return "", fmt.Errorf("encode extraction reply: %w", err)
	}

	return string(encoded), nil
}

func replyFields(fields map[string]serviceports.AIDocumentField) []extractFieldResponse {
	out := make([]extractFieldResponse, 0, min(len(fields), maxExtractFields))
	for _, key := range extractFieldKeys {
		field, ok := fields[key]
		if !ok || field.Value == "" {
			continue
		}
		if len(out) >= maxExtractFields {
			break
		}
		alternatives := field.AlternativeValues
		if alternatives == nil {
			alternatives = []string{}
		}
		out = append(out, extractFieldResponse{
			Key:               key,
			Label:             stringutils.TruncateRunes(field.Label, maxFieldLabelRunes),
			Value:             stringutils.TruncateRunes(field.Value, maxFieldValueRunes),
			Confidence:        clampAIConfidence(field.Confidence),
			EvidenceExcerpt:   stringutils.TruncateRunes(field.EvidenceExcerpt, maxEvidenceRunes),
			PageNumber:        field.PageNumber,
			ReviewRequired:    field.ReviewRequired,
			Conflict:          field.Conflict,
			Source:            stringutils.TruncateRunes(field.Source, maxFieldSourceRunes),
			AlternativeValues: alternatives,
		})
	}

	return out
}

func replyStops(stops []*serviceports.AIDocumentStop) []*serviceports.AIDocumentStop {
	out := make([]*serviceports.AIDocumentStop, 0, min(len(stops), maxExtractStops))
	for _, stop := range stops {
		if stop == nil {
			continue
		}
		if len(out) >= maxExtractStops {
			break
		}
		out = append(out, &serviceports.AIDocumentStop{
			Sequence:            stop.Sequence,
			Role:                stop.Role,
			Name:                stringutils.TruncateRunes(stop.Name, maxStopNameRunes),
			AddressLine1:        stringutils.TruncateRunes(stop.AddressLine1, maxStopAddressRunes),
			AddressLine2:        stringutils.TruncateRunes(stop.AddressLine2, maxStopAddressRunes),
			City:                stringutils.TruncateRunes(stop.City, maxStopCityRunes),
			State:               stringutils.TruncateRunes(stop.State, maxStopStateRunes),
			PostalCode:          stringutils.TruncateRunes(stop.PostalCode, maxStopPostalRunes),
			Date:                stringutils.TruncateRunes(stop.Date, maxStopDateRunes),
			TimeWindow:          stringutils.TruncateRunes(stop.TimeWindow, maxStopTimeWindowRunes),
			AppointmentRequired: stop.AppointmentRequired,
			PageNumber:          stop.PageNumber,
			EvidenceExcerpt:     stringutils.TruncateRunes(stop.EvidenceExcerpt, maxEvidenceRunes),
			Confidence:          clampAIConfidence(stop.Confidence),
			ReviewRequired:      stop.ReviewRequired,
			Source:              stringutils.TruncateRunes(stop.Source, maxFieldSourceRunes),
		})
	}
	slices.SortStableFunc(out, func(a, b *serviceports.AIDocumentStop) int {
		return cmp.Compare(a.Sequence, b.Sequence)
	})

	return out
}
