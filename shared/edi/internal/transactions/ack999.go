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

// Ack999Builder builds 999 Implementation Acknowledgments
type Ack999Builder struct {
	*BaseBuilder
}

// NewAck999Builder creates a new 999 builder
func NewAck999Builder(registry *segments.SegmentRegistry, version string, delims x12.Delimiters) *Ack999Builder {
	return &Ack999Builder{
		BaseBuilder: NewBaseBuilder(registry, version, delims),
	}
}

// GetTransactionType returns "999"
func (b *Ack999Builder) GetTransactionType() string {
	return "999"
}

// GetVersion returns the X12 version
func (b *Ack999Builder) GetVersion() string {
	return b.version
}

// Ack999Request represents the data needed to build a 999
type Ack999Request struct {
	OriginalISA       InterchangeEnvelope
	OriginalGS        FunctionalGroupEnvelope
	Transactions      []TransactionInfo
	Issues            []validation.Issue
	GenerateTimestamp time.Time
	ImplementationRef string // Implementation convention reference
}

// Build constructs a 999 acknowledgment
func (b *Ack999Builder) Build(ctx context.Context, data interface{}) (string, error) {
	req, ok := data.(*Ack999Request)
	if !ok {
		return "", fmt.Errorf("invalid data type for 999 builder")
	}
	
	// Generate control numbers
	isaCtrl := b.GenerateControlNumber()
	gsCtrl := b.GenerateShortControlNumber()
	stCtrl := "0001"
	
	// Build the 999 transaction content
	var txContent strings.Builder
	
	// Build ST segment with implementation reference
	stValues := map[string]string{
		"1": "999",
		"2": stCtrl,
	}
	if req.ImplementationRef != "" {
		stValues["3"] = req.ImplementationRef
	}
	stSegment, err := b.BuildSegment("ST", stValues)
	if err != nil {
		return "", fmt.Errorf("failed to build ST: %w", err)
	}
	txContent.WriteString(stSegment)
	txContent.WriteByte(b.delims.Segment)
	
	// Build AK1 - Functional Group Response Header
	ak1Values := map[string]string{
		"1": req.OriginalGS.FunctionalID,
		"2": req.OriginalGS.ControlNumber,
		"3": req.OriginalGS.Version,
	}
	ak1Segment, err := b.BuildSegment("AK1", ak1Values)
	if err != nil {
		return "", fmt.Errorf("failed to build AK1: %w", err)
	}
	txContent.WriteString(ak1Segment)
	txContent.WriteByte(b.delims.Segment)
	
	// Process each transaction
	for _, tx := range req.Transactions {
		// Build AK2 - Transaction Set Response Header
		ak2Values := map[string]string{
			"1": tx.SetID,
			"2": tx.Control,
		}
		if req.ImplementationRef != "" {
			ak2Values["3"] = req.ImplementationRef
		}
		ak2Segment, err := b.BuildSegment("AK2", ak2Values)
		if err != nil {
			return "", fmt.Errorf("failed to build AK2: %w", err)
		}
		txContent.WriteString(ak2Segment)
		txContent.WriteByte(b.delims.Segment)
		
		// Add IK3/IK4 segments for errors in this transaction
		txIssues := b.filterIssuesForTransaction(req.Issues, tx)
		for _, issue := range txIssues {
			if issue.Level == "segment" {
				// Build IK3 for segment-level errors
				ik3Values := map[string]string{
					"1": issue.Tag,
					"2": fmt.Sprintf("%d", issue.SegmentIndex),
					"3": issue.LoopID,
					"4": b.getIK304Code(issue),
				}
				ik3Segment, err := b.BuildSegment("IK3", ik3Values)
				if err != nil {
					continue
				}
				txContent.WriteString(ik3Segment)
				txContent.WriteByte(b.delims.Segment)
				
				// Add CTX for context if present
				if issue.Context != "" {
					ctxValues := map[string]string{
						"1": "SITUATIONAL" + string(b.delims.Component) + issue.Context,
					}
					ctxSegment, err := b.BuildSegment("CTX", ctxValues)
					if err == nil {
						txContent.WriteString(ctxSegment)
						txContent.WriteByte(b.delims.Segment)
					}
				}
				
				// Add IK4 for element-level errors within the segment
				if issue.ElementPosition > 0 {
					ik4Values := map[string]string{
						"1": fmt.Sprintf("%d", issue.ElementPosition),
						"2": issue.ElementRef,
						"3": b.getIK403Code(issue),
						"4": issue.BadValue,
					}
					ik4Segment, err := b.BuildSegment("IK4", ik4Values)
					if err == nil {
						txContent.WriteString(ik4Segment)
						txContent.WriteByte(b.delims.Segment)
					}
				}
			}
		}
		
		// Build IK5 - Transaction Set Response Trailer
		ackCode := "A" // Accepted
		if len(txIssues) > 0 {
			for _, issue := range txIssues {
				if issue.Severity == validation.Error {
					ackCode = "R" // Rejected
					break
				}
			}
		}
		
		ik5Values := map[string]string{
			"1": ackCode,
		}
		// Add up to 5 error codes
		errorCount := 0
		for _, issue := range txIssues {
			if errorCount >= 5 {
				break
			}
			if issue.Code != "" {
				ik5Values[fmt.Sprintf("%d", errorCount+2)] = issue.Code
				errorCount++
			}
		}
		
		ik5Segment, err := b.BuildSegment("IK5", ik5Values)
		if err != nil {
			return "", fmt.Errorf("failed to build IK5: %w", err)
		}
		txContent.WriteString(ik5Segment)
		txContent.WriteByte(b.delims.Segment)
	}
	
	// Build AK9 - Functional Group Response Trailer
	groupAckCode := "A" // Accepted
	rejectedCount := 0
	for _, issue := range req.Issues {
		if issue.Severity == validation.Error {
			rejectedCount++
		}
	}
	
	acceptedCount := len(req.Transactions)
	if rejectedCount > 0 {
		groupAckCode = "P" // Partially Accepted
		acceptedCount = len(req.Transactions) - rejectedCount
	}
	
	ak9Values := map[string]string{
		"1": groupAckCode,
		"2": fmt.Sprintf("%d", len(req.Transactions)),
		"3": fmt.Sprintf("%d", len(req.Transactions)),
		"4": fmt.Sprintf("%d", acceptedCount),
	}
	
	// Add functional group error code if present
	if rejectedCount > 0 {
		ak9Values["5"] = b.getAK905Code(req.Issues)
	}
	
	ak9Segment, err := b.BuildSegment("AK9", ak9Values)
	if err != nil {
		return "", fmt.Errorf("failed to build AK9: %w", err)
	}
	txContent.WriteString(ak9Segment)
	txContent.WriteByte(b.delims.Segment)
	
	// Build SE segment
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
	
	// Create envelopes
	interchange := InterchangeEnvelope{
		AuthQualifier:     req.OriginalISA.AuthQualifier,
		AuthInfo:          req.OriginalISA.AuthInfo,
		SecurityQualifier: req.OriginalISA.SecurityQualifier,
		SecurityInfo:      req.OriginalISA.SecurityInfo,
		// Swap sender/receiver for acknowledgment
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
		FunctionalID: "FA", // Functional Acknowledgment
		// Swap sender/receiver for acknowledgment
		SenderCode:        req.OriginalGS.ReceiverCode,
		ReceiverCode:      req.OriginalGS.SenderCode,
		Date:              req.GenerateTimestamp,
		ControlNumber:     gsCtrl,
		ResponsibleAgency: req.OriginalGS.ResponsibleAgency,
		Version:           req.OriginalGS.Version,
		TransactionCount:  1,
		Content:           txContent.String(),
	}
	
	// Build complete message with envelopes
	return b.BuildEnvelope(interchange, []FunctionalGroupEnvelope{group})
}

// Parse parses raw segments into 999 structure
func (b *Ack999Builder) Parse(ctx context.Context, segments []x12.Segment) (interface{}, error) {
	// This would parse incoming 999 acknowledgments
	// Implementation would extract AK1, AK2, IK3, IK4, IK5, CTX, AK9 segments
	// and build a structured representation
	return nil, fmt.Errorf("999 parsing not yet implemented")
}

// filterIssuesForTransaction filters issues for a specific transaction
func (b *Ack999Builder) filterIssuesForTransaction(issues []validation.Issue, tx TransactionInfo) []validation.Issue {
	var filtered []validation.Issue
	for _, issue := range issues {
		if issue.SegmentIndex >= tx.StartIndex && issue.SegmentIndex <= tx.EndIndex {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// getIK304Code maps issue codes to IK304 segment syntax error codes
func (b *Ack999Builder) getIK304Code(issue validation.Issue) string {
	switch issue.Code {
	case "SEG_UNRECOGNIZED":
		return "1" // Unrecognized segment ID
	case "SEG_UNEXPECTED":
		return "2" // Unexpected segment
	case "SEG_MISSING":
		return "3" // Mandatory segment missing
	case "SEG_LOOP_ERROR":
		return "4" // Loop occurs over maximum times
	case "SEG_NOT_IN_POSITION":
		return "5" // Segment exceeds maximum use
	case "SEG_NOT_IN_DEFINED":
		return "6" // Segment not in defined transaction set
	case "SEG_NOT_IN_PROPER":
		return "7" // Segment not in proper sequence
	case "SEG_HAS_DATA_ERRORS":
		return "8" // Segment has data element errors
	default:
		return "8" // Default to data element errors
	}
}

// getIK403Code maps issue codes to IK403 element syntax error codes
func (b *Ack999Builder) getIK403Code(issue validation.Issue) string {
	switch issue.Code {
	case "ELEM_MISSING":
		return "1" // Mandatory data element missing
	case "ELEM_CONDITIONAL_MISSING":
		return "2" // Conditional required data element missing
	case "ELEM_TOO_SHORT":
		return "3" // Too many data elements
	case "ELEM_TOO_LONG":
		return "4" // Data element too short
	case "ELEM_INVALID_CODE":
		return "5" // Data element too long
	case "ELEM_INVALID_CHAR":
		return "6" // Invalid character in data element
	case "ELEM_INVALID_CODE_VALUE":
		return "7" // Invalid code value
	case "ELEM_INVALID_DATE":
		return "8" // Invalid date
	case "ELEM_INVALID_TIME":
		return "9" // Invalid time
	case "ELEM_EXCLUSION_CONDITION":
		return "10" // Exclusion condition violated
	default:
		return "12" // Too many repetitions
	}
}

// getAK905Code returns the most severe functional group error code
func (b *Ack999Builder) getAK905Code(issues []validation.Issue) string {
	for _, issue := range issues {
		if issue.Severity == validation.Error {
			switch issue.Code {
			case "GROUP_CONTROL_MISMATCH":
				return "1" // Functional group not supported
			case "GROUP_VERSION_UNSUPPORTED":
				return "2" // Functional group version not supported
			case "GROUP_TRAILER_MISSING":
				return "3" // Functional group trailer missing
			case "GROUP_CONTROL_NUMBER_MISMATCH":
				return "4" // Group control number mismatch
			case "GROUP_COUNT_MISMATCH":
				return "5" // Number of included transaction sets mismatch
			case "GROUP_CONTROL_NUMBER_DUP":
				return "6" // Group control number violates syntax
			}
		}
	}
	return "" // No functional group errors
}