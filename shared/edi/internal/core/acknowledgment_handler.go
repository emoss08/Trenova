package core

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/shared/edi/internal/ack"
	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// AcknowledgmentType represents the type of acknowledgment
type AcknowledgmentType string

const (
	Ack997 = AcknowledgmentType("997")
	Ack999 = AcknowledgmentType("999")
)

// AcknowledgmentHandler provides a unified interface for generating EDI acknowledgments
type AcknowledgmentHandler struct {
	registry       *segments.SegmentRegistry
	profileManager *profiles.ProfileManager
	delims         x12.Delimiters
}

// NewAcknowledgmentHandler creates a new acknowledgment handler
func NewAcknowledgmentHandler(
	registry *segments.SegmentRegistry,
	profileManager *profiles.ProfileManager,
) *AcknowledgmentHandler {
	return &AcknowledgmentHandler{
		registry:       registry,
		profileManager: profileManager,
		delims:         x12.DefaultDelimiters(),
	}
}

// GenerateRequest contains the parameters for generating an acknowledgment
type GenerateRequest struct {
	Type      AcknowledgmentType
	PartnerID string
	Original  *x12.Document
	Issues    []validation.Issue
	Accepted  bool
	Context   map[string]any
}

// GenerateResponse contains the generated acknowledgment
type GenerateResponse struct {
	EDI        string
	Segments   []x12.Segment
	Statistics AckStatistics
}

// AckStatistics contains statistics about the acknowledgment
type AckStatistics struct {
	TransactionSetsReceived int
	TransactionSetsAccepted int
	SegmentsReceived        int
	SegmentsAccepted        int
	ErrorsReported          int
	WarningsReported        int
}

// Generate creates an acknowledgment based on the request
func (h *AcknowledgmentHandler) Generate(
	req GenerateRequest,
) (*GenerateResponse, error) {
	var profile *profiles.PartnerProfile
	if req.PartnerID != "" {
		if p, err := h.profileManager.GetProfile(req.PartnerID); err == nil {
			profile = p
			h.delims = profile.GetDelimiters()
		}
	}

	switch req.Type {
	case Ack997:
		return h.generate997(req)
	case Ack999:
		return h.generate999(req)
	default:
		return nil, fmt.Errorf("unsupported acknowledgment type: %s", req.Type)
	}
}

func (h *AcknowledgmentHandler) generate997(
	req GenerateRequest,
) (*GenerateResponse, error) {
	edi := ack.Generate997(req.Original.Segments, h.delims, req.Issues)

	segments, err := h.parseEDIToSegmentsWithDelimiters(edi, h.delims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated 997: %w", err)
	}

	stats := h.calculateStatistics(req)

	return &GenerateResponse{
		EDI:        edi,
		Segments:   segments,
		Statistics: stats,
	}, nil
}

func (h *AcknowledgmentHandler) generate999(req GenerateRequest) (*GenerateResponse, error) {
	txBlock := x12.TxBlock{
		SetID:   req.Original.Metadata.TransactionType,
		Control: req.Original.Metadata.STControlNumber,
	}

	txs := []x12.TxBlock{txBlock}
	txAccepted := []bool{req.Accepted}

	edi := ack.Generate999(req.Original.Segments, h.delims, txs, txAccepted)

	segments, err := h.parseEDIToSegmentsWithDelimiters(edi, h.delims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated 999: %w", err)
	}

	stats := h.calculateStatistics(req)

	return &GenerateResponse{
		EDI:        edi,
		Segments:   segments,
		Statistics: stats,
	}, nil
}

func (h *AcknowledgmentHandler) parseEDIToSegmentsWithDelimiters(
	edi string,
	delims x12.Delimiters,
) ([]x12.Segment, error) {
	var segments []x12.Segment

	lines := strings.Split(edi, string(delims.Segment))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, string(delims.Element))
		if len(parts) == 0 {
			continue
		}

		tag := parts[0]
		elements := [][]string{}

		for i := 1; i < len(parts); i++ {
			if strings.Contains(parts[i], string(delims.Component)) {
				components := strings.Split(parts[i], string(delims.Component))
				elements = append(elements, components)
			} else {
				elements = append(elements, []string{parts[i]})
			}
		}

		segments = append(segments, x12.Segment{
			Tag:      tag,
			Elements: elements,
		})
	}

	return segments, nil
}

func (h *AcknowledgmentHandler) calculateStatistics(req GenerateRequest) AckStatistics {
	stats := AckStatistics{
		TransactionSetsReceived: 1,
		SegmentsReceived:        len(req.Original.Segments),
	}

	if req.Accepted {
		stats.TransactionSetsAccepted = 1
		stats.SegmentsAccepted = stats.SegmentsReceived
	} else {
		stats.TransactionSetsAccepted = 0
		stats.SegmentsAccepted = 0
	}

	for _, issue := range req.Issues {
		switch issue.Severity {
		case validation.Error:
			stats.ErrorsReported++
		case validation.Warning:
			stats.WarningsReported++
		}
	}

	if stats.ErrorsReported > 0 && req.Accepted {
		stats.SegmentsAccepted = stats.SegmentsReceived - stats.ErrorsReported
	}

	return stats
}
