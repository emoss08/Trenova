package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// Ack997Builder builds 997 Functional Acknowledgments
type Ack997Builder struct {
	*BaseBuilder
}

// NewAck997Builder creates a new 997 builder
func NewAck997Builder(
	registry *segments.SegmentRegistry,
	version string,
	delims x12.Delimiters,
) *Ack997Builder {
	return &Ack997Builder{
		BaseBuilder: NewBaseBuilder(registry, version, delims),
	}
}

// GetTransactionType returns "997"
func (b *Ack997Builder) GetTransactionType() string {
	return "997"
}

// GetVersion returns the X12 version
func (b *Ack997Builder) GetVersion() string {
	return b.version
}

// Ack997Request represents the data needed to build a 997
type Ack997Request struct {
	OriginalISA       InterchangeEnvelope
	OriginalGS        FunctionalGroupEnvelope
	Transactions      []TransactionInfo
	Issues            []validation.Issue
	GenerateTimestamp time.Time
}

// TransactionInfo holds information about a transaction set
type TransactionInfo struct {
	SetID      string
	Control    string
	StartIndex int
	EndIndex   int
}

// Build constructs a 997 acknowledgment
func (b *Ack997Builder) Build(ctx context.Context, data any) (string, error) {
	req, ok := data.(*Ack997Request)
	if !ok {
		return "", fmt.Errorf("invalid data type for 997 builder")
	}

	isaCtrl := b.GenerateControlNumber()
	gsCtrl := b.GenerateShortControlNumber()
	stCtrl := "0001"

	var txContent strings.Builder

	stValues := map[string]string{
		"1": "997",
		"2": stCtrl,
	}
	stSegment, err := b.BuildSegment("ST", stValues)
	if err != nil {
		return "", fmt.Errorf("failed to build ST: %w", err)
	}
	txContent.WriteString(stSegment)
	txContent.WriteByte(b.delims.Segment)

	ak1Values := map[string]string{
		"1": req.OriginalGS.FunctionalID,
		"2": req.OriginalGS.ControlNumber,
	}
	ak1Segment, err := b.BuildSegment("AK1", ak1Values)
	if err != nil {
		return "", fmt.Errorf("failed to build AK1: %w", err)
	}
	txContent.WriteString(ak1Segment)
	txContent.WriteByte(b.delims.Segment)

	for _, tx := range req.Transactions {
		ak2Values := map[string]string{
			"1": tx.SetID,
			"2": tx.Control,
		}
		ak2Segment, err := b.BuildSegment("AK2", ak2Values)
		if err != nil {
			return "", fmt.Errorf("failed to build AK2: %w", err)
		}
		txContent.WriteString(ak2Segment)
		txContent.WriteByte(b.delims.Segment)

		txHasErrors := false
		for _, issue := range req.Issues {
			if issue.Severity == validation.Error {
				if issue.SegmentIndex > 0 &&
					(issue.SegmentIndex < tx.StartIndex || issue.SegmentIndex > tx.EndIndex) {
					continue
				}
				txHasErrors = true

				if issue.Tag != "" && issue.SegmentIndex > 0 {
					ak3Values := map[string]string{
						"1": issue.Tag,
						"2": fmt.Sprintf("%d", issue.SegmentIndex),
						"3": "",
						"4": issue.Code,
					}
					ak3Segment, err := b.BuildSegment("AK3", ak3Values)
					if err == nil {
						txContent.WriteString(ak3Segment)
						txContent.WriteByte(b.delims.Segment)
					}
				}
			}
		}

		ackCode := "A"
		if txHasErrors {
			ackCode = "R"
		}

		ak5Values := map[string]string{
			"1": ackCode,
		}
		ak5Segment, err := b.BuildSegment("AK5", ak5Values)
		if err != nil {
			return "", fmt.Errorf("failed to build AK5: %w", err)
		}
		txContent.WriteString(ak5Segment)
		txContent.WriteByte(b.delims.Segment)
	}

	groupAckCode := "A"
	for _, issue := range req.Issues {
		if issue.Severity == validation.Error {
			groupAckCode = "E"
			break
		}
	}

	ak9Values := map[string]string{
		"1": groupAckCode,
		"2": fmt.Sprintf("%d", len(req.Transactions)),
		"3": fmt.Sprintf("%d", len(req.Transactions)),
		"4": fmt.Sprintf("%d", len(req.Transactions)),
	}
	ak9Segment, err := b.BuildSegment("AK9", ak9Values)
	if err != nil {
		return "", fmt.Errorf("failed to build AK9: %w", err)
	}
	txContent.WriteString(ak9Segment)
	txContent.WriteByte(b.delims.Segment)

	segmentCount := strings.Count(txContent.String(), string(b.delims.Segment)) + 1
	seValues := map[string]string{
		"1": fmt.Sprintf("%d", segmentCount),
		"2": stCtrl,
	}
	seSegment, err := b.BuildSegment("SE", seValues)
	if err != nil {
		return "", fmt.Errorf("failed to build SE: %w", err)
	}
	txContent.WriteString(seSegment)
	txContent.WriteByte(b.delims.Segment)

	interchange := InterchangeEnvelope{
		AuthQualifier:     req.OriginalISA.AuthQualifier,
		AuthInfo:          req.OriginalISA.AuthInfo,
		SecurityQualifier: req.OriginalISA.SecurityQualifier,
		SecurityInfo:      req.OriginalISA.SecurityInfo,
		SenderQualifier:   req.OriginalISA.ReceiverQualifier,
		SenderID:          req.OriginalISA.ReceiverID,
		ReceiverQualifier: req.OriginalISA.SenderQualifier,
		ReceiverID:        req.OriginalISA.SenderID,
		Date:              req.GenerateTimestamp,
		StandardsID:       req.OriginalISA.StandardsID,
		Version:           req.OriginalISA.Version,
		ControlNumber:     isaCtrl,
		AckRequested:      "0",
		Usage:             req.OriginalISA.Usage,
	}

	group := FunctionalGroupEnvelope{
		FunctionalID:      "FA",
		SenderCode:        req.OriginalGS.ReceiverCode,
		ReceiverCode:      req.OriginalGS.SenderCode,
		Date:              req.GenerateTimestamp,
		ControlNumber:     gsCtrl,
		ResponsibleAgency: req.OriginalGS.ResponsibleAgency,
		Version:           req.OriginalGS.Version,
		TransactionCount:  1,
		Content:           txContent.String(),
	}

	return b.BuildEnvelope(interchange, []FunctionalGroupEnvelope{group})
}

// Parse parses raw segments into 997 structure
func (b *Ack997Builder) Parse(ctx context.Context, segments []x12.Segment) (interface{}, error) {
	// This would parse incoming 997 acknowledgments
	// Implementation would extract AK1, AK2, AK3, AK5, AK9 segments
	// and build a structured representation
	return nil, fmt.Errorf("997 parsing not yet implemented")
}
