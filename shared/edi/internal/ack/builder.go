package ack

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// AckBuilder provides schema-driven acknowledgment generation
type AckBuilder struct {
	registry *segments.SegmentRegistry
	builder  *segments.SchemaBuilder
	delims   x12.Delimiters
}

// NewAckBuilder creates a new acknowledgment builder
func NewAckBuilder(
	registry *segments.SegmentRegistry,
	version string,
	delims x12.Delimiters,
) *AckBuilder {
	return &AckBuilder{
		registry: registry,
		builder:  segments.NewSchemaBuilder(registry, version),
		delims:   delims,
	}
}

// Build997 generates a 997 functional acknowledgment using schemas
func (b *AckBuilder) Build997(
	originalSegs []x12.Segment,
	issues []validation.Issue,
) (string, error) {
	var result strings.Builder

	isaVals := b.extractISAValues(originalSegs)
	gsVals := b.extractGSValues(originalSegs)

	now := time.Now().UTC()
	isaCtrl := fmt.Sprintf("%09d", now.Unix()%1000000000)
	gsCtrl := fmt.Sprintf("%d", now.Unix()%100000)
	stCtrl := "0001"

	isaSegment, err := b.buildISA(isaVals, isaCtrl, now, true)
	if err != nil {
		return "", fmt.Errorf("failed to build ISA: %w", err)
	}
	result.WriteString(isaSegment)
	result.WriteByte(b.delims.Segment)

	gsSegment, err := b.buildGS(gsVals, gsCtrl, now, true)
	if err != nil {
		return "", fmt.Errorf("failed to build GS: %w", err)
	}
	result.WriteString(gsSegment)
	result.WriteByte(b.delims.Segment)

	stSegment := fmt.Sprintf(
		"ST%s997%s%s",
		string(b.delims.Element),
		string(b.delims.Element),
		stCtrl,
	)
	result.WriteString(stSegment)
	result.WriteByte(b.delims.Segment)

	ak1Segment := fmt.Sprintf(
		"AK1%s%s%s%s",
		string(b.delims.Element),
		gsVals["functionalID"],
		string(b.delims.Element),
		gsVals["controlNumber"],
	)
	result.WriteString(ak1Segment)
	result.WriteByte(b.delims.Segment)

	transactions := b.extractTransactions(originalSegs)
	for _, tx := range transactions {
		ak2Segment := fmt.Sprintf(
			"AK2%s%s%s%s",
			string(b.delims.Element),
			tx.SetID,
			string(b.delims.Element),
			tx.Control,
		)
		result.WriteString(ak2Segment)
		result.WriteByte(b.delims.Segment)

		txIssues := b.filterIssuesForTransaction(issues, tx)
		for _, issue := range txIssues {
			ak3Segment := fmt.Sprintf("AK3%s%s%s%d%s%s%s%s",
				string(b.delims.Element), issue.Tag,
				string(b.delims.Element), issue.SegmentIndex,
				string(b.delims.Element), "", // Loop ID (optional)
				string(b.delims.Element), issue.Code)
			result.WriteString(ak3Segment)
			result.WriteByte(b.delims.Segment)
		}

		ackCode := "A" // Accepted
		if len(txIssues) > 0 {
			for _, issue := range txIssues {
				if issue.Severity == validation.Error {
					ackCode = "E" // Accepted with errors
					break
				}
			}
		}
		ak5Segment := fmt.Sprintf("AK5%s%s", string(b.delims.Element), ackCode)
		result.WriteString(ak5Segment)
		result.WriteByte(b.delims.Segment)
	}

	ackCode := "A" // Accepted
	for _, issue := range issues {
		if issue.Severity == validation.Error {
			ackCode = "E" // Accepted with errors
			break
		}
	}
	ak9Segment := fmt.Sprintf("AK9%s%s%s%d%s%d%s%d",
		string(b.delims.Element), ackCode,
		string(b.delims.Element), len(transactions),
		string(b.delims.Element), len(transactions),
		string(b.delims.Element), len(transactions))
	result.WriteString(ak9Segment)
	result.WriteByte(b.delims.Segment)

	seSegment := fmt.Sprintf(
		"SE%s%d%s%s",
		string(b.delims.Element),
		b.countSegments(result.String())+1,
		string(b.delims.Element),
		stCtrl,
	)
	result.WriteString(seSegment)
	result.WriteByte(b.delims.Segment)

	geSegment := fmt.Sprintf(
		"GE%s1%s%s",
		string(b.delims.Element),
		string(b.delims.Element),
		gsCtrl,
	)
	result.WriteString(geSegment)
	result.WriteByte(b.delims.Segment)

	ieaSegment := fmt.Sprintf(
		"IEA%s1%s%s",
		string(b.delims.Element),
		string(b.delims.Element),
		isaCtrl,
	)
	result.WriteString(ieaSegment)
	result.WriteByte(b.delims.Segment)

	return result.String(), nil
}

// Build999 generates a 999 implementation acknowledgment
func (b *AckBuilder) Build999(
	originalSegs []x12.Segment,
	issues []validation.Issue,
) (string, error) {
	var result strings.Builder

	isaVals := b.extractISAValues(originalSegs)
	gsVals := b.extractGSValues(originalSegs)

	now := time.Now().UTC()
	isaCtrl := fmt.Sprintf("%09d", now.Unix()%1000000000)
	gsCtrl := fmt.Sprintf("%d", now.Unix()%100000)
	stCtrl := "0001"

	isaSegment, err := b.buildISA(isaVals, isaCtrl, now, true)
	if err != nil {
		return "", fmt.Errorf("failed to build ISA: %w", err)
	}
	result.WriteString(isaSegment)
	result.WriteByte(b.delims.Segment)

	gsSegment, err := b.buildGS(gsVals, gsCtrl, now, true)
	if err != nil {
		return "", fmt.Errorf("failed to build GS: %w", err)
	}
	result.WriteString(gsSegment)
	result.WriteByte(b.delims.Segment)

	stSegment := fmt.Sprintf(
		"ST%s999%s%s%s005010",
		string(b.delims.Element),
		string(b.delims.Element),
		stCtrl,
		string(b.delims.Element),
	)
	result.WriteString(stSegment)
	result.WriteByte(b.delims.Segment)

	ak1Segment := fmt.Sprintf("AK1%s%s%s%s%s%s",
		string(b.delims.Element), gsVals["functionalID"],
		string(b.delims.Element), gsVals["controlNumber"],
		string(b.delims.Element), gsVals["version"])
	result.WriteString(ak1Segment)
	result.WriteByte(b.delims.Segment)

	transactions := b.extractTransactions(originalSegs)
	for _, tx := range transactions {
		ak2Segment := fmt.Sprintf("AK2%s%s%s%s%s005010",
			string(b.delims.Element), tx.SetID,
			string(b.delims.Element), tx.Control,
			string(b.delims.Element))
		result.WriteString(ak2Segment)
		result.WriteByte(b.delims.Segment)

		txIssues := b.filterIssuesForTransaction(issues, tx)
		for _, issue := range txIssues {
			if issue.Level == "segment" {
				ik3Segment := fmt.Sprintf("IK3%s%s%s%d%s%s%s%s",
					string(b.delims.Element), issue.Tag,
					string(b.delims.Element), issue.SegmentIndex,
					string(b.delims.Element), issue.LoopID,
					string(b.delims.Element), b.getIK304Code(issue))
				result.WriteString(ik3Segment)
				result.WriteByte(b.delims.Segment)

				if issue.Context != "" {
					ctxSegment := fmt.Sprintf("CTX%sSITUATIONAL%s%s",
						string(b.delims.Element), string(b.delims.Component), issue.Context)
					result.WriteString(ctxSegment)
					result.WriteByte(b.delims.Segment)
				}

				if issue.ElementPosition > 0 {
					ik4Segment := fmt.Sprintf("IK4%s%d%s%s%s%s%s%s",
						string(b.delims.Element), issue.ElementPosition,
						string(b.delims.Element), issue.ElementRef,
						string(b.delims.Element), b.getIK403Code(issue),
						string(b.delims.Element), issue.BadValue)
					result.WriteString(ik4Segment)
					result.WriteByte(b.delims.Segment)
				}
			}
		}

		ackCode := "A" // Accepted
		if len(txIssues) > 0 {
			for _, issue := range txIssues {
				if issue.Severity == validation.Error {
					ackCode = "R" // Rejected
					break
				}
			}
		}
		ik5Segment := fmt.Sprintf("IK5%s%s", string(b.delims.Element), ackCode)
		result.WriteString(ik5Segment)
		result.WriteByte(b.delims.Segment)
	}

	ackCode := "A" // Accepted
	for _, issue := range issues {
		if issue.Severity == validation.Error {
			ackCode = "P" // Partially Accepted
			break
		}
	}
	ak9Segment := fmt.Sprintf("AK9%s%s%s%d%s%d%s%d",
		string(b.delims.Element), ackCode,
		string(b.delims.Element), len(transactions),
		string(b.delims.Element), len(transactions),
		string(b.delims.Element), len(transactions))
	result.WriteString(ak9Segment)
	result.WriteByte(b.delims.Segment)

	seSegment := fmt.Sprintf(
		"SE%s%d%s%s",
		string(b.delims.Element),
		b.countSegments(result.String())+1,
		string(b.delims.Element),
		stCtrl,
	)
	result.WriteString(seSegment)
	result.WriteByte(b.delims.Segment)

	geSegment := fmt.Sprintf(
		"GE%s1%s%s",
		string(b.delims.Element),
		string(b.delims.Element),
		gsCtrl,
	)
	result.WriteString(geSegment)
	result.WriteByte(b.delims.Segment)

	ieaSegment := fmt.Sprintf(
		"IEA%s1%s%s",
		string(b.delims.Element),
		string(b.delims.Element),
		isaCtrl,
	)
	result.WriteString(ieaSegment)
	result.WriteByte(b.delims.Segment)

	return result.String(), nil
}

func (b *AckBuilder) extractISAValues(segs []x12.Segment) map[string]string {
	values := make(map[string]string)
	for _, seg := range segs {
		if strings.ToUpper(seg.Tag) == "ISA" {
			if len(seg.Elements) >= 16 {
				values["authQual"] = b.getElement(seg, 0, 0)
				values["authInfo"] = b.getElement(seg, 1, 0)
				values["secQual"] = b.getElement(seg, 2, 0)
				values["secInfo"] = b.getElement(seg, 3, 0)
				values["senderQual"] = b.getElement(seg, 4, 0)
				values["senderID"] = b.getElement(seg, 5, 0)
				values["receiverQual"] = b.getElement(seg, 6, 0)
				values["receiverID"] = b.getElement(seg, 7, 0)
				values["standard"] = b.getElement(seg, 10, 0)
				values["version"] = b.getElement(seg, 11, 0)
				values["ackRequested"] = b.getElement(seg, 13, 0)
				values["usage"] = b.getElement(seg, 14, 0)
			}
			break
		}
	}
	return values
}

func (b *AckBuilder) extractGSValues(segs []x12.Segment) map[string]string {
	values := make(map[string]string)
	for _, seg := range segs {
		if strings.ToUpper(seg.Tag) == "GS" {
			if len(seg.Elements) >= 8 {
				values["functionalID"] = b.getElement(seg, 0, 0)
				values["senderCode"] = b.getElement(seg, 1, 0)
				values["receiverCode"] = b.getElement(seg, 2, 0)
				values["controlNumber"] = b.getElement(seg, 5, 0)
				values["responsibleAgency"] = b.getElement(seg, 6, 0)
				values["version"] = b.getElement(seg, 7, 0)
			}
			break
		}
	}
	return values
}

func (b *AckBuilder) buildISA(
	original map[string]string,
	ctrlNum string,
	now time.Time,
	swap bool,
) (string, error) {
	var parts []string
	parts = append(parts, "ISA")
	parts = append(parts, original["authQual"])
	parts = append(parts, original["authInfo"])
	parts = append(parts, original["secQual"])
	parts = append(parts, original["secInfo"])

	if swap {
		parts = append(parts, original["receiverQual"])
		parts = append(parts, original["receiverID"])
		parts = append(parts, original["senderQual"])
		parts = append(parts, original["senderID"])
	} else {
		parts = append(parts, original["senderQual"])
		parts = append(parts, original["senderID"])
		parts = append(parts, original["receiverQual"])
		parts = append(parts, original["receiverID"])
	}

	parts = append(parts, now.Format("060102")) // YYMMDD
	parts = append(parts, now.Format("1504"))   // HHMM
	parts = append(parts, original["standard"])
	parts = append(parts, original["version"])
	parts = append(parts, ctrlNum)
	parts = append(parts, "0") // No interchange acknowledgment requested
	parts = append(parts, original["usage"])
	parts = append(parts, string(b.delims.Component))

	return strings.Join(parts, string(b.delims.Element)), nil
}

func (b *AckBuilder) buildGS(
	original map[string]string,
	ctrlNum string,
	now time.Time,
	swap bool,
) (string, error) {
	var parts []string
	parts = append(parts, "GS")
	parts = append(parts, "FA")

	if swap {
		parts = append(parts, original["receiverCode"])
		parts = append(parts, original["senderCode"])
	} else {
		parts = append(parts, original["senderCode"])
		parts = append(parts, original["receiverCode"])
	}

	parts = append(parts, now.Format("20060102")) // CCYYMMDD
	parts = append(parts, now.Format("1504"))     // HHMM
	parts = append(parts, ctrlNum)
	parts = append(parts, original["responsibleAgency"])
	parts = append(parts, original["version"])

	return strings.Join(parts, string(b.delims.Element)), nil
}

func (b *AckBuilder) buildAK2(tx TransactionInfo) (string, error) {
	values := map[string]string{
		"1": tx.SetID,   // Transaction Set ID (e.g., "204")
		"2": tx.Control, // Transaction Set Control Number
	}
	return b.builder.BuildSegment("AK2", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildAK3(issue validation.Issue) (string, error) {
	values := map[string]string{
		"1": issue.Tag,                             // Segment ID
		"2": fmt.Sprintf("%d", issue.SegmentIndex), // Segment Position
		"3": "",                                    // Loop ID (optional)
		"4": issue.Code,                            // Error code
	}
	return b.builder.BuildSegment("AK3", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildAK5(tx TransactionInfo, issues []validation.Issue) (string, error) {
	ackCode := "A" // Accepted
	if len(issues) > 0 {
		hasErrors := false
		for _, issue := range issues {
			if issue.Severity == validation.Error {
				hasErrors = true
				break
			}
		}
		if hasErrors {
			ackCode = "E" // Accepted with errors
		}
	}

	values := map[string]string{
		"1": ackCode, // Transaction Set Acknowledgment Code
	}
	return b.builder.BuildSegment("AK5", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildAK9(
	transactions []TransactionInfo,
	issues []validation.Issue,
) (string, error) {
	ackCode := "A" // Accepted
	hasErrors := false
	for _, issue := range issues {
		if issue.Severity == validation.Error {
			hasErrors = true
			break
		}
	}
	if hasErrors {
		ackCode = "E" // Accepted with errors
	}

	values := map[string]string{
		"1": ackCode,                              // Group Acknowledgment Code
		"2": fmt.Sprintf("%d", len(transactions)), // Number of transaction sets
		"3": fmt.Sprintf("%d", len(transactions)), // Number received
		"4": fmt.Sprintf("%d", len(transactions)), // Number accepted
	}
	return b.builder.BuildSegment("AK9", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) getElement(seg x12.Segment, elemIdx, compIdx int) string {
	if elemIdx < 0 || elemIdx >= len(seg.Elements) {
		return ""
	}
	if compIdx < 0 || compIdx >= len(seg.Elements[elemIdx]) {
		return ""
	}
	return seg.Elements[elemIdx][compIdx]
}

func (b *AckBuilder) countSegments(content string) int {
	return strings.Count(content, string(b.delims.Segment))
}

func (b *AckBuilder) extractTransactions(segs []x12.Segment) []TransactionInfo {
	var transactions []TransactionInfo

	for i, seg := range segs {
		if strings.ToUpper(seg.Tag) == "ST" {
			tx := TransactionInfo{
				StartIndex: i,
				SetID:      b.getElement(seg, 0, 0),
				Control:    b.getElement(seg, 1, 0),
			}

			for j := i + 1; j < len(segs); j++ {
				if strings.ToUpper(segs[j].Tag) == "SE" {
					if b.getElement(segs[j], 1, 0) == tx.Control {
						tx.EndIndex = j
						break
					}
				}
			}

			transactions = append(transactions, tx)
		}
	}

	return transactions
}

func (b *AckBuilder) filterIssuesForTransaction(
	issues []validation.Issue,
	tx TransactionInfo,
) []validation.Issue {
	var filtered []validation.Issue
	for _, issue := range issues {
		if issue.SegmentIndex >= tx.StartIndex && issue.SegmentIndex <= tx.EndIndex {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// TransactionInfo holds information about a transaction set
type TransactionInfo struct {
	SetID      string // e.g., "204"
	Control    string // Control number
	StartIndex int    // ST segment index
	EndIndex   int    // SE segment index
}

// Additional builder methods for 999

func (b *AckBuilder) buildAK2For999(tx TransactionInfo) (string, error) {
	values := map[string]string{
		"1": tx.SetID,   // Transaction Set ID
		"2": tx.Control, // Transaction Set Control Number
		"3": "005010",   // Implementation Convention Reference
	}
	return b.builder.BuildSegment("AK2", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildIK3(issue validation.Issue) (string, error) {
	values := map[string]string{
		"1": issue.Tag,                             // Segment ID Code
		"2": fmt.Sprintf("%d", issue.SegmentIndex), // Segment Position in Transaction Set
		"3": issue.LoopID,                          // Loop Segment Code (optional)
		"4": b.getIK304Code(issue),                 // Segment Syntax Error Code
	}
	return b.builder.BuildSegment("IK3", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildIK4(issue validation.Issue) (string, error) {
	values := map[string]string{
		"1": fmt.Sprintf("%d", issue.ElementPosition), // Position in Segment
		"2": issue.ElementRef,                         // Element Reference Number
		"3": b.getIK403Code(issue),                    // Element Syntax Error Code
		"4": issue.BadValue,                           // Copy of Bad Data Element
	}
	return b.builder.BuildSegment("IK4", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildIK5(tx TransactionInfo, issues []validation.Issue) (string, error) {
	ackCode := "A" // Accepted
	if len(issues) > 0 {
		hasErrors := false
		for _, issue := range issues {
			if issue.Severity == validation.Error {
				hasErrors = true
				break
			}
		}
		if hasErrors {
			ackCode = "R" // Rejected
		}
	}

	values := map[string]string{
		"1": ackCode, // Transaction Set Acknowledgment Code
	}

	if len(issues) > 0 {
		for i, issue := range issues {
			if i >= 5 {
				break // Maximum 5 error codes
			}
			values[fmt.Sprintf("%d", i+2)] = issue.Code
		}
	}

	return b.builder.BuildSegment("IK5", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildCTX(issue validation.Issue) (string, error) {
	values := map[string]string{
		"1": "SITUATIONAL", // Context Name
		"2": issue.Context, // Context Reference
	}
	return b.builder.BuildSegment("CTX", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) buildAK9For999(
	transactions []TransactionInfo,
	issues []validation.Issue,
) (string, error) {
	ackCode := "A" // Accepted
	acceptedCount := len(transactions)
	rejectedCount := 0

	for _, issue := range issues {
		if issue.Severity == validation.Error {
			rejectedCount++
		}
	}

	if rejectedCount > 0 {
		ackCode = "P" // Partially Accepted
		acceptedCount = len(transactions) - rejectedCount
	}

	values := map[string]string{
		"1": ackCode,                              // Group Acknowledgment Code
		"2": fmt.Sprintf("%d", len(transactions)), // Number of Transaction Sets Included
		"3": fmt.Sprintf("%d", len(transactions)), // Number of Transaction Sets Received
		"4": fmt.Sprintf("%d", acceptedCount),     // Number of Transaction Sets Accepted
	}

	if rejectedCount > 0 {
		values["5"] = b.getAK905Code(issues) // Functional Group Syntax Error Code
	}

	return b.builder.BuildSegment("AK9", values, b.delims.Element, b.delims.Component)
}

func (b *AckBuilder) getIK304Code(issue validation.Issue) string {
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

func (b *AckBuilder) getIK403Code(issue validation.Issue) string {
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

func (b *AckBuilder) getAK905Code(issues []validation.Issue) string {
	for _, issue := range issues {
		if issue.Severity == validation.Error {
			if issue.Code == "GROUP_CONTROL_MISMATCH" {
				return "1" // Functional group not supported
			}
			if issue.Code == "GROUP_VERSION_UNSUPPORTED" {
				return "2" // Functional group version not supported
			}
			if issue.Code == "GROUP_TRAILER_MISSING" {
				return "3" // Functional group trailer missing
			}
			if issue.Code == "GROUP_CONTROL_NUMBER_MISMATCH" {
				return "4" // Group control number in the functional group header and trailer do not agree
			}
			if issue.Code == "GROUP_COUNT_MISMATCH" {
				return "5" // Number of included transaction sets does not match actual count
			}
			if issue.Code == "GROUP_CONTROL_NUMBER_DUP" {
				return "6" // Group control number violates syntax
			}
		}
	}
	return ""
}
