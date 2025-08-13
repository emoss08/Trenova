package segments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// TransactionAssembler builds EDI transactions from segments
type TransactionAssembler struct {
	registry  *SegmentRegistry
	processor *SegmentProcessor
	builder   *SchemaBuilder
	delims    x12.Delimiters
}

// NewTransactionAssembler creates a new assembler
func NewTransactionAssembler(
	registry *SegmentRegistry,
	processor *SegmentProcessor,
	delims x12.Delimiters,
) *TransactionAssembler {
	return &TransactionAssembler{
		registry:  registry,
		processor: processor,
		builder:   NewSchemaBuilder(registry, "004010"), // Default version
		delims:    delims,
	}
}

// SetVersion sets the X12 version for building
func (a *TransactionAssembler) SetVersion(version string) {
	a.builder = NewSchemaBuilder(a.registry, version)
}

// EDIDocument represents a complete EDI document
type EDIDocument struct {
	Interchanges []Interchange `json:"interchanges"`
	Version      string        `json:"version"`
	Metadata     DocumentMeta  `json:"metadata"`
}

// Interchange represents an ISA..IEA envelope
type Interchange struct {
	Header     InterchangeHeader  `json:"header"`
	Groups     []FunctionalGroup  `json:"groups"`
	Trailer    InterchangeTrailer `json:"trailer"`
	ControlNum string             `json:"control_number"`
}

// InterchangeHeader represents ISA segment data
type InterchangeHeader struct {
	SenderID    string            `json:"sender_id"`
	ReceiverID  string            `json:"receiver_id"`
	Date        time.Time         `json:"date"`
	ControlNum  string            `json:"control_number"`
	TestProd    string            `json:"test_prod"`
	SegmentData map[string]string `json:"segment_data"`
}

// InterchangeTrailer represents IEA segment data
type InterchangeTrailer struct {
	GroupCount int    `json:"group_count"`
	ControlNum string `json:"control_number"`
}

// FunctionalGroup represents a GS..GE envelope
type FunctionalGroup struct {
	Header       GroupHeader   `json:"header"`
	Transactions []Transaction `json:"transactions"`
	Trailer      GroupTrailer  `json:"trailer"`
	ControlNum   string        `json:"control_number"`
}

// GroupHeader represents GS segment data
type GroupHeader struct {
	FunctionalID string            `json:"functional_id"`
	SenderCode   string            `json:"sender_code"`
	ReceiverCode string            `json:"receiver_code"`
	Date         time.Time         `json:"date"`
	ControlNum   string            `json:"control_number"`
	Version      string            `json:"version"`
	SegmentData  map[string]string `json:"segment_data"`
}

// GroupTrailer represents GE segment data
type GroupTrailer struct {
	TransactionCount int    `json:"transaction_count"`
	ControlNum       string `json:"control_number"`
}

// Transaction represents a ST..SE envelope
type Transaction struct {
	Header     TransactionHeader   `json:"header"`
	Segments   []*ProcessedSegment `json:"segments"`
	Trailer    TransactionTrailer  `json:"trailer"`
	Type       string              `json:"type"` // e.g., "204", "997", "999"
	ControlNum string              `json:"control_number"`
}

// TransactionHeader represents ST segment data
type TransactionHeader struct {
	TransactionID string            `json:"transaction_id"`
	ControlNum    string            `json:"control_number"`
	SegmentData   map[string]string `json:"segment_data"`
}

// TransactionTrailer represents SE segment data
type TransactionTrailer struct {
	SegmentCount int    `json:"segment_count"`
	ControlNum   string `json:"control_number"`
}

// DocumentMeta contains metadata about the document
type DocumentMeta struct {
	CreatedAt    time.Time         `json:"created_at"`
	ProcessedAt  time.Time         `json:"processed_at"`
	SourceFile   string            `json:"source_file,omitempty"`
	PartnerID    string            `json:"partner_id,omitempty"`
	Direction    string            `json:"direction"` // "inbound" or "outbound"
	Stats        ProcessingStats   `json:"stats"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// ProcessingStats contains processing statistics
type ProcessingStats struct {
	TotalSegments  int            `json:"total_segments"`
	TotalErrors    int            `json:"total_errors"`
	ProcessingTime time.Duration  `json:"processing_time"`
	SegmentCounts  map[string]int `json:"segment_counts"`
}

// ParseToDocument parses raw segments into a structured document
func (a *TransactionAssembler) ParseToDocument(
	ctx context.Context,
	segments []x12.Segment,
) (*EDIDocument, error) {
	startTime := time.Now()

	processed, err := a.processor.ProcessSegments(ctx, segments, a.builder.version)
	if err != nil {
		return nil, fmt.Errorf("failed to process segments: %w", err)
	}

	doc := &EDIDocument{
		Interchanges: []Interchange{},
		Version:      a.builder.version,
		Metadata: DocumentMeta{
			CreatedAt:   time.Now(),
			ProcessedAt: time.Now(),
			Direction:   "inbound",
			Stats: ProcessingStats{
				TotalSegments: len(segments),
				SegmentCounts: a.countSegments(segments),
			},
		},
	}

	// Build document structure
	if err := a.buildDocumentStructure(doc, processed); err != nil {
		return nil, fmt.Errorf("failed to build document structure: %w", err)
	}

	doc.Metadata.Stats.ProcessingTime = time.Since(startTime)

	return doc, nil
}

// BuildDocument constructs an EDI document from structured data
func (a *TransactionAssembler) BuildDocument(
	ctx context.Context,
	doc *EDIDocument,
) (string, error) {
	var result strings.Builder

	for _, interchange := range doc.Interchanges {
		isaSegment, err := a.buildISA(interchange.Header)
		if err != nil {
			return "", fmt.Errorf("failed to build ISA: %w", err)
		}
		result.WriteString(isaSegment)
		result.WriteByte(a.delims.Segment)

		for _, group := range interchange.Groups {
			// Build GS
			gsSegment, err := a.buildGS(group.Header)
			if err != nil {
				return "", fmt.Errorf("failed to build GS: %w", err)
			}
			result.WriteString(gsSegment)
			result.WriteByte(a.delims.Segment)

			for _, transaction := range group.Transactions {
				// Build ST
				stSegment, err := a.buildST(transaction.Header)
				if err != nil {
					return "", fmt.Errorf("failed to build ST: %w", err)
				}
				result.WriteString(stSegment)
				result.WriteByte(a.delims.Segment)

				// Build transaction segments
				for _, seg := range transaction.Segments {
					if seg.Schema.ID == "ST" || seg.Schema.ID == "SE" {
						continue // Skip envelope segments
					}

					segmentStr, err := a.buildSegmentFromProcessed(seg)
					if err != nil {
						return "", fmt.Errorf("failed to build segment %s: %w", seg.Schema.ID, err)
					}
					result.WriteString(segmentStr)
					result.WriteByte(a.delims.Segment)
				}

				// Build SE
				seSegment, err := a.buildSE(transaction.Trailer)
				if err != nil {
					return "", fmt.Errorf("failed to build SE: %w", err)
				}
				result.WriteString(seSegment)
				result.WriteByte(a.delims.Segment)
			}

			// Build GE
			geSegment, err := a.buildGE(group.Trailer)
			if err != nil {
				return "", fmt.Errorf("failed to build GE: %w", err)
			}
			result.WriteString(geSegment)
			result.WriteByte(a.delims.Segment)
		}

		// Build IEA
		ieaSegment, err := a.buildIEA(interchange.Trailer)
		if err != nil {
			return "", fmt.Errorf("failed to build IEA: %w", err)
		}
		result.WriteString(ieaSegment)
		result.WriteByte(a.delims.Segment)
	}

	return result.String(), nil
}

// Build997Acknowledgment creates a 997 acknowledgment from processed segments
func (a *TransactionAssembler) Build997Acknowledgment(
	ctx context.Context,
	original *EDIDocument,
	errors []ProcessingError,
) (*EDIDocument, error) {
	ack := &EDIDocument{
		Version: original.Version,
		Metadata: DocumentMeta{
			CreatedAt:   time.Now(),
			ProcessedAt: time.Now(),
			Direction:   "outbound",
			PartnerID:   original.Metadata.PartnerID,
			CustomFields: map[string]string{
				"original_control": original.Interchanges[0].ControlNum,
				"ack_type":         "997",
			},
		},
	}

	for _, origInterchange := range original.Interchanges {
		ackInterchange := Interchange{
			Header: InterchangeHeader{
				SenderID:   origInterchange.Header.ReceiverID,
				ReceiverID: origInterchange.Header.SenderID,
				Date:       time.Now(),
				ControlNum: a.generateControlNumber(),
				TestProd:   origInterchange.Header.TestProd,
			},
			Groups: []FunctionalGroup{},
		}

		for _, origGroup := range origInterchange.Groups {
			ackGroup := FunctionalGroup{
				Header: GroupHeader{
					FunctionalID: "FA", // Functional Acknowledgment
					SenderCode:   origGroup.Header.ReceiverCode,
					ReceiverCode: origGroup.Header.SenderCode,
					Date:         time.Now(),
					ControlNum:   a.generateControlNumber(),
					Version:      origGroup.Header.Version,
				},
				Transactions: []Transaction{},
			}

			ack997 := a.build997Transaction(origGroup, errors)
			ackGroup.Transactions = append(ackGroup.Transactions, ack997)

			ackInterchange.Groups = append(ackInterchange.Groups, ackGroup)
		}

		ack.Interchanges = append(ack.Interchanges, ackInterchange)
	}

	return ack, nil
}

func (a *TransactionAssembler) buildDocumentStructure(
	doc *EDIDocument,
	segments []*ProcessedSegment,
) error {
	var currentInterchange *Interchange
	var currentGroup *FunctionalGroup
	var currentTransaction *Transaction

	for _, seg := range segments {
		switch seg.Schema.ID {
		case "ISA":
			interchange := Interchange{
				Header: a.extractISAHeader(seg),
				Groups: []FunctionalGroup{},
			}
			doc.Interchanges = append(doc.Interchanges, interchange)
			currentInterchange = &doc.Interchanges[len(doc.Interchanges)-1]

		case "GS":
			if currentInterchange == nil {
				return fmt.Errorf("GS segment without ISA")
			}
			group := FunctionalGroup{
				Header:       a.extractGSHeader(seg),
				Transactions: []Transaction{},
			}
			currentInterchange.Groups = append(currentInterchange.Groups, group)
			currentGroup = &currentInterchange.Groups[len(currentInterchange.Groups)-1]

		case "ST":
			if currentGroup == nil {
				return fmt.Errorf("ST segment without GS")
			}
			transaction := Transaction{
				Header:   a.extractSTHeader(seg),
				Segments: []*ProcessedSegment{},
				Type:     a.getStringValue(seg.Data, "ST01"),
			}
			currentGroup.Transactions = append(currentGroup.Transactions, transaction)
			currentTransaction = &currentGroup.Transactions[len(currentGroup.Transactions)-1]

		case "SE":
			if currentTransaction != nil {
				currentTransaction.Trailer = a.extractSETrailer(seg)
				currentTransaction = nil
			}

		case "GE":
			if currentGroup != nil {
				currentGroup.Trailer = a.extractGETrailer(seg)
				currentGroup = nil
			}

		case "IEA":
			if currentInterchange != nil {
				currentInterchange.Trailer = a.extractIEATrailer(seg)
				currentInterchange = nil
			}

		default:
			if currentTransaction != nil {
				currentTransaction.Segments = append(currentTransaction.Segments, seg)
			}
		}
	}

	return nil
}

func (a *TransactionAssembler) buildISA(header InterchangeHeader) (string, error) {
	values := header.SegmentData
	if values == nil {
		values = make(map[string]string)
	}

	values["ISA01"] = a.getOrDefault(values, "ISA01", "00")
	values["ISA02"] = a.getOrDefault(values, "ISA02", strings.Repeat(" ", 10))
	values["ISA03"] = a.getOrDefault(values, "ISA03", "00")
	values["ISA04"] = a.getOrDefault(values, "ISA04", strings.Repeat(" ", 10))
	values["ISA05"] = a.getOrDefault(values, "ISA05", "ZZ")
	values["ISA06"] = fmt.Sprintf("%-15s", header.SenderID)
	values["ISA07"] = a.getOrDefault(values, "ISA07", "ZZ")
	values["ISA08"] = fmt.Sprintf("%-15s", header.ReceiverID)
	values["ISA09"] = header.Date.Format("060102")
	values["ISA10"] = header.Date.Format("1504")
	values["ISA11"] = a.getOrDefault(values, "ISA11", "U")
	values["ISA12"] = a.getOrDefault(values, "ISA12", "00401")
	values["ISA13"] = fmt.Sprintf("%09s", header.ControlNum)
	values["ISA14"] = a.getOrDefault(values, "ISA14", "0")
	values["ISA15"] = header.TestProd
	values["ISA16"] = string(a.delims.Component)

	return a.builder.BuildSegment("ISA", values, a.delims.Element, a.delims.Component)
}

func (a *TransactionAssembler) buildSegmentFromProcessed(seg *ProcessedSegment) (string, error) {
	values := make(map[string]string)

	for key, value := range seg.Data {
		if strings.HasPrefix(key, "_") || strings.HasSuffix(key, "_name") ||
			strings.HasSuffix(key, "_description") {
			continue
		}

		values[key] = fmt.Sprintf("%v", value)
	}

	return a.builder.BuildSegment(seg.Schema.ID, values, a.delims.Element, a.delims.Component)
}

func (a *TransactionAssembler) build997Transaction(
	origGroup FunctionalGroup,
	errors []ProcessingError,
) Transaction {
	return Transaction{
		Type:       "997",
		ControlNum: a.generateControlNumber(),
		Header: TransactionHeader{
			TransactionID: "997",
			ControlNum:    a.generateControlNumber(),
		},
		Segments: []*ProcessedSegment{
			// NOTE: AK1, AK2, etc. segments would be built here
		},
	}
}

func (a *TransactionAssembler) countSegments(segments []x12.Segment) map[string]int {
	counts := make(map[string]int)
	for _, seg := range segments {
		counts[seg.Tag]++
	}
	return counts
}

func (a *TransactionAssembler) generateControlNumber() string {
	return fmt.Sprintf("%09d", time.Now().Unix()%1000000000)
}

func (a *TransactionAssembler) getOrDefault(m map[string]string, key, defaultValue string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return defaultValue
}

func (a *TransactionAssembler) getStringValue(data map[string]any, key string) string {
	if v, ok := data[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (a *TransactionAssembler) extractISAHeader(seg *ProcessedSegment) InterchangeHeader {
	return InterchangeHeader{
		SenderID:    a.getStringValue(seg.Data, "ISA06"),
		ReceiverID:  a.getStringValue(seg.Data, "ISA08"),
		ControlNum:  a.getStringValue(seg.Data, "ISA13"),
		TestProd:    a.getStringValue(seg.Data, "ISA15"),
		SegmentData: a.convertDataToStrings(seg.Data),
	}
}

func (a *TransactionAssembler) extractGSHeader(seg *ProcessedSegment) GroupHeader {
	return GroupHeader{
		FunctionalID: a.getStringValue(seg.Data, "GS01"),
		SenderCode:   a.getStringValue(seg.Data, "GS02"),
		ReceiverCode: a.getStringValue(seg.Data, "GS03"),
		ControlNum:   a.getStringValue(seg.Data, "GS06"),
		Version:      a.getStringValue(seg.Data, "GS08"),
		SegmentData:  a.convertDataToStrings(seg.Data),
	}
}

func (a *TransactionAssembler) extractSTHeader(seg *ProcessedSegment) TransactionHeader {
	return TransactionHeader{
		TransactionID: a.getStringValue(seg.Data, "ST01"),
		ControlNum:    a.getStringValue(seg.Data, "ST02"),
		SegmentData:   a.convertDataToStrings(seg.Data),
	}
}

func (a *TransactionAssembler) extractSETrailer(seg *ProcessedSegment) TransactionTrailer {
	count := 0
	if v := a.getStringValue(seg.Data, "SE01"); v != "" {
		fmt.Sscanf(v, "%d", &count)
	}
	return TransactionTrailer{
		SegmentCount: count,
		ControlNum:   a.getStringValue(seg.Data, "SE02"),
	}
}

func (a *TransactionAssembler) extractGETrailer(seg *ProcessedSegment) GroupTrailer {
	count := 0
	if v := a.getStringValue(seg.Data, "GE01"); v != "" {
		fmt.Sscanf(v, "%d", &count)
	}
	return GroupTrailer{
		TransactionCount: count,
		ControlNum:       a.getStringValue(seg.Data, "GE02"),
	}
}

func (a *TransactionAssembler) extractIEATrailer(seg *ProcessedSegment) InterchangeTrailer {
	count := 0
	if v := a.getStringValue(seg.Data, "IEA01"); v != "" {
		fmt.Sscanf(v, "%d", &count)
	}
	return InterchangeTrailer{
		GroupCount: count,
		ControlNum: a.getStringValue(seg.Data, "IEA02"),
	}
}

func (a *TransactionAssembler) convertDataToStrings(data map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range data {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

func (a *TransactionAssembler) buildGS(header GroupHeader) (string, error) {
	values := header.SegmentData
	if values == nil {
		values = make(map[string]string)
	}

	values["GS01"] = header.FunctionalID
	values["GS02"] = header.SenderCode
	values["GS03"] = header.ReceiverCode
	values["GS04"] = header.Date.Format("20060102")
	values["GS05"] = header.Date.Format("1504")
	values["GS06"] = header.ControlNum
	values["GS07"] = a.getOrDefault(values, "GS07", "X")
	values["GS08"] = header.Version

	return a.builder.BuildSegment("GS", values, a.delims.Element, a.delims.Component)
}

func (a *TransactionAssembler) buildST(header TransactionHeader) (string, error) {
	values := header.SegmentData
	if values == nil {
		values = make(map[string]string)
	}

	values["ST01"] = header.TransactionID
	values["ST02"] = header.ControlNum

	return a.builder.BuildSegment("ST", values, a.delims.Element, a.delims.Component)
}

func (a *TransactionAssembler) buildSE(trailer TransactionTrailer) (string, error) {
	values := map[string]string{
		"SE01": fmt.Sprintf("%d", trailer.SegmentCount),
		"SE02": trailer.ControlNum,
	}

	return a.builder.BuildSegment("SE", values, a.delims.Element, a.delims.Component)
}

func (a *TransactionAssembler) buildGE(trailer GroupTrailer) (string, error) {
	values := map[string]string{
		"GE01": fmt.Sprintf("%d", trailer.TransactionCount),
		"GE02": trailer.ControlNum,
	}

	return a.builder.BuildSegment("GE", values, a.delims.Element, a.delims.Component)
}

func (a *TransactionAssembler) buildIEA(trailer InterchangeTrailer) (string, error) {
	values := map[string]string{
		"IEA01": fmt.Sprintf("%d", trailer.GroupCount),
		"IEA02": trailer.ControlNum,
	}

	return a.builder.BuildSegment("IEA", values, a.delims.Element, a.delims.Component)
}

// ProcessingError represents an error during processing
type ProcessingError struct {
	Segment  string `json:"segment"`
	Position int    `json:"position"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}
