package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// TransactionBuilder defines the interface for building EDI transactions
type TransactionBuilder interface {
	Build(ctx context.Context, data interface{}) (string, error)
	Parse(ctx context.Context, segments []x12.Segment) (interface{}, error)
	GetTransactionType() string
	GetVersion() string
}

// BaseBuilder provides common functionality for all transaction builders
type BaseBuilder struct {
	registry  *segments.SegmentRegistry
	builder   *segments.SchemaBuilder
	processor *segments.SegmentProcessor
	assembler *segments.TransactionAssembler
	delims    x12.Delimiters
	version   string
}

// NewBaseBuilder creates a new base builder
func NewBaseBuilder(
	registry *segments.SegmentRegistry,
	version string,
	delims x12.Delimiters,
) *BaseBuilder {
	processor := segments.NewSegmentProcessor(registry)
	builder := segments.NewSchemaBuilder(registry, version)
	assembler := segments.NewTransactionAssembler(registry, processor, delims)

	return &BaseBuilder{
		registry:  registry,
		builder:   builder,
		processor: processor,
		assembler: assembler,
		delims:    delims,
		version:   version,
	}
}

// BuildSegment builds a segment using the schema or falls back to manual construction
func (b *BaseBuilder) BuildSegment(segmentID string, values map[string]string) (string, error) {
	segment, err := b.builder.BuildSegment(segmentID, values, b.delims.Element, b.delims.Component)
	if err == nil {
		return segment, nil
	}

	return b.buildSegmentManually(segmentID, values)
}

// buildSegmentManually constructs a segment without schema
func (b *BaseBuilder) buildSegmentManually(
	segmentID string,
	values map[string]string,
) (string, error) {
	parts := []string{segmentID}

	switch segmentID {
	case "ISA":
		// ISA has fixed 16 elements
		for i := 1; i <= 16; i++ {
			if val, ok := values[fmt.Sprintf("%d", i)]; ok {
				parts = append(parts, val)
			} else {
				parts = append(parts, "")
			}
		}

	case "GS":
		// GS has 8 elements
		for i := 1; i <= 8; i++ {
			if val, ok := values[fmt.Sprintf("%d", i)]; ok {
				parts = append(parts, val)
			} else {
				parts = append(parts, "")
			}
		}

	case "ST":
		// ST has 2-3 elements
		for i := 1; i <= 3; i++ {
			if val, ok := values[fmt.Sprintf("%d", i)]; ok {
				parts = append(parts, val)
			}
		}

	case "SE":
		// SE has 2 elements
		for i := 1; i <= 2; i++ {
			if val, ok := values[fmt.Sprintf("%d", i)]; ok {
				parts = append(parts, val)
			} else {
				parts = append(parts, "")
			}
		}

	case "GE":
		// GE has 2 elements
		for i := 1; i <= 2; i++ {
			if val, ok := values[fmt.Sprintf("%d", i)]; ok {
				parts = append(parts, val)
			} else {
				parts = append(parts, "")
			}
		}

	case "IEA":
		// IEA has 2 elements
		for i := 1; i <= 2; i++ {
			if val, ok := values[fmt.Sprintf("%d", i)]; ok {
				parts = append(parts, val)
			} else {
				parts = append(parts, "")
			}
		}

	default:
		// For unknown segments, use ordered keys
		maxKey := 0
		for key := range values {
			var keyNum int
			if _, err := fmt.Sscanf(key, "%d", &keyNum); err == nil && keyNum > maxKey {
				maxKey = keyNum
			}
		}

		for i := 1; i <= maxKey; i++ {
			if val, ok := values[fmt.Sprintf("%d", i)]; ok {
				parts = append(parts, val)
			} else {
				parts = append(parts, "")
			}
		}
	}

	return strings.Join(parts, string(b.delims.Element)), nil
}

// GenerateControlNumber generates a control number
func (b *BaseBuilder) GenerateControlNumber() string {
	return fmt.Sprintf("%09d", time.Now().Unix()%1000000000)
}

// GenerateShortControlNumber generates a short control number
func (b *BaseBuilder) GenerateShortControlNumber() string {
	return fmt.Sprintf("%04d", time.Now().Unix()%10000)
}

// BuildEnvelope builds the ISA/IEA and GS/GE envelope
func (b *BaseBuilder) BuildEnvelope(
	interchange InterchangeEnvelope,
	groups []FunctionalGroupEnvelope,
) (string, error) {
	var result strings.Builder

	isaValues := map[string]string{
		"1":  interchange.AuthQualifier,
		"2":  interchange.AuthInfo,
		"3":  interchange.SecurityQualifier,
		"4":  interchange.SecurityInfo,
		"5":  interchange.SenderQualifier,
		"6":  fmt.Sprintf("%-15s", interchange.SenderID),
		"7":  interchange.ReceiverQualifier,
		"8":  fmt.Sprintf("%-15s", interchange.ReceiverID),
		"9":  interchange.Date.Format("060102"),
		"10": interchange.Date.Format("1504"),
		"11": interchange.StandardsID,
		"12": interchange.Version,
		"13": interchange.ControlNumber,
		"14": interchange.AckRequested,
		"15": interchange.Usage,
		"16": string(b.delims.Component),
	}

	isaSegment, err := b.BuildSegment("ISA", isaValues)
	if err != nil {
		return "", fmt.Errorf("failed to build ISA: %w", err)
	}
	result.WriteString(isaSegment)
	result.WriteByte(b.delims.Segment)

	for _, group := range groups {
		gsValues := map[string]string{
			"1": group.FunctionalID,
			"2": group.SenderCode,
			"3": group.ReceiverCode,
			"4": group.Date.Format("20060102"),
			"5": group.Date.Format("1504"),
			"6": group.ControlNumber,
			"7": group.ResponsibleAgency,
			"8": group.Version,
		}

		gsSegment, err := b.BuildSegment("GS", gsValues)
		if err != nil {
			return "", fmt.Errorf("failed to build GS: %w", err)
		}
		result.WriteString(gsSegment)
		result.WriteByte(b.delims.Segment)

		result.WriteString(group.Content)

		geValues := map[string]string{
			"1": fmt.Sprintf("%d", group.TransactionCount),
			"2": group.ControlNumber,
		}

		geSegment, err := b.BuildSegment("GE", geValues)
		if err != nil {
			return "", fmt.Errorf("failed to build GE: %w", err)
		}
		result.WriteString(geSegment)
		result.WriteByte(b.delims.Segment)
	}

	ieaValues := map[string]string{
		"1": fmt.Sprintf("%d", len(groups)),
		"2": interchange.ControlNumber,
	}

	ieaSegment, err := b.BuildSegment("IEA", ieaValues)
	if err != nil {
		return "", fmt.Errorf("failed to build IEA: %w", err)
	}
	result.WriteString(ieaSegment)
	result.WriteByte(b.delims.Segment)

	return result.String(), nil
}

// InterchangeEnvelope represents ISA/IEA envelope data
type InterchangeEnvelope struct {
	AuthQualifier     string
	AuthInfo          string
	SecurityQualifier string
	SecurityInfo      string
	SenderQualifier   string
	SenderID          string
	ReceiverQualifier string
	ReceiverID        string
	Date              time.Time
	StandardsID       string
	Version           string
	ControlNumber     string
	AckRequested      string
	Usage             string
}

// FunctionalGroupEnvelope represents GS/GE envelope data
type FunctionalGroupEnvelope struct {
	FunctionalID      string
	SenderCode        string
	ReceiverCode      string
	Date              time.Time
	ControlNumber     string
	ResponsibleAgency string
	Version           string
	TransactionCount  int
	Content           string // The transaction sets content
}

// TransactionEnvelope represents ST/SE envelope data
type TransactionEnvelope struct {
	TransactionID string
	ControlNumber string
	Version       string // For 999
	SegmentCount  int
	Content       string // The transaction content
}
