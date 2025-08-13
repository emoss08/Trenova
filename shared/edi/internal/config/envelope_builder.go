package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// EnvelopeBuilder wraps transactions with ISA/GS/GE/IEA envelopes
type EnvelopeBuilder struct {
	delims x12.Delimiters
}

// NewEnvelopeBuilder creates a new envelope builder
func NewEnvelopeBuilder(delims x12.Delimiters) *EnvelopeBuilder {
	return &EnvelopeBuilder{
		delims: delims,
	}
}

// EnvelopeOptions contains options for building envelopes
type EnvelopeOptions struct {
	SenderID        string
	ReceiverID      string
	SenderGS        string
	ReceiverGS      string
	Version         string
	TestMode        bool
	ControlNumber   string
	GroupNumber     string
	TransactionType string
}

// WrapTransaction wraps a transaction with ISA/GS/GE/IEA envelopes
func (b *EnvelopeBuilder) WrapTransaction(transaction string, opts EnvelopeOptions) string {
	var result strings.Builder

	if opts.SenderID == "" {
		opts.SenderID = "SENDER"
	}
	if opts.ReceiverID == "" {
		opts.ReceiverID = "RECEIVER"
	}
	if opts.SenderGS == "" {
		opts.SenderGS = opts.SenderID
	}
	if opts.ReceiverGS == "" {
		opts.ReceiverGS = opts.ReceiverID
	}
	if opts.Version == "" {
		opts.Version = "004010"
	}
	if opts.ControlNumber == "" {
		opts.ControlNumber = fmt.Sprintf("%09d", time.Now().Unix()%1000000000)
	}
	if opts.GroupNumber == "" {
		opts.GroupNumber = fmt.Sprintf("%d", time.Now().Unix()%100000)
	}
	if opts.TransactionType == "" {
		opts.TransactionType = "SM" // Default functional ID
	}

	now := time.Now().UTC()
	date := now.Format("060102")
	timeStr := now.Format("1504")

	isaElements := []string{
		"ISA",
		"00",                                  // Authorization Information Qualifier
		"          ",                          // Authorization Information (10 spaces)
		"00",                                  // Security Information Qualifier
		"          ",                          // Security Information (10 spaces)
		"ZZ",                                  // Interchange ID Qualifier
		fmt.Sprintf("%-15s", opts.SenderID),   // Interchange Sender ID (15 chars)
		"ZZ",                                  // Interchange ID Qualifier
		fmt.Sprintf("%-15s", opts.ReceiverID), // Interchange Receiver ID (15 chars)
		date,                                  // Interchange Date
		timeStr,                               // Interchange Time
		"U",                                   // Repetition Separator
		opts.Version[:5],                      // Interchange Control Version Number
		opts.ControlNumber,                    // Interchange Control Number
		"0",                                   // Acknowledgment Requested
		"P",                                   // Usage Indicator (P=Production, T=Test)
		">",                                   // Component Element Separator
	}
	result.WriteString(strings.Join(isaElements, string(b.delims.Element)))
	result.WriteByte(b.delims.Segment)

	gsElements := []string{
		"GS",
		opts.TransactionType,   // Functional Identifier Code
		opts.SenderGS,          // Application Sender's Code
		opts.ReceiverGS,        // Application Receiver's Code
		now.Format("20060102"), // Date
		timeStr,                // Time
		opts.GroupNumber,       // Group Control Number
		"X",                    // Responsible Agency Code
		opts.Version,           // Version/Release/Industry Identifier Code
	}
	result.WriteString(strings.Join(gsElements, string(b.delims.Element)))
	result.WriteByte(b.delims.Segment)

	result.WriteString(transaction)

	geElements := []string{
		"GE",
		"1",              // Number of Transaction Sets Included
		opts.GroupNumber, // Group Control Number
	}
	result.WriteString(strings.Join(geElements, string(b.delims.Element)))
	result.WriteByte(b.delims.Segment)

	ieaElements := []string{
		"IEA",
		"1",                // Number of Included Functional Groups
		opts.ControlNumber, // Interchange Control Number
	}
	result.WriteString(strings.Join(ieaElements, string(b.delims.Element)))
	result.WriteByte(b.delims.Segment)

	return result.String()
}

// UnwrapTransaction extracts the transaction from an EDI document
func (b *EnvelopeBuilder) UnwrapTransaction(edi string) (string, error) {
	stIndex := strings.Index(edi, "ST"+string(b.delims.Element))
	if stIndex == -1 {
		return "", fmt.Errorf("no ST segment found")
	}

	seIndex := strings.LastIndex(edi, "SE"+string(b.delims.Element))
	if seIndex == -1 {
		return "", fmt.Errorf("no SE segment found")
	}

	seEnd := seIndex
	for i := seIndex; i < len(edi); i++ {
		if edi[i] == b.delims.Segment {
			seEnd = i + 1
			break
		}
	}

	return edi[stIndex:seEnd], nil
}
