package x12

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Document represents a parsed EDI X12 document
type Document struct {
	Segments   []Segment
	Delimiters Delimiters
	Metadata   DocumentMetadata
}

// DocumentMetadata contains metadata about the document
type DocumentMetadata struct {
	ISAControlNumber string
	GSControlNumber  string
	STControlNumber  string
	TransactionType  string
	Version          string
	SenderID         string
	ReceiverID       string
	Date             string
	Time             string
}

// Parser provides EDI parsing capabilities
type Parser struct {
	delims    Delimiters
	profileID string // Partner profile ID for custom parsing behavior
}

// NewParser creates a new EDI parser
func NewParser() *Parser {
	return &Parser{
		delims: DefaultDelimiters(),
	}
}

// NewParserWithDelimiters creates a parser with custom delimiters
func NewParserWithDelimiters(delims Delimiters) *Parser {
	return &Parser{
		delims: delims,
	}
}

// NewParserWithProfile creates a parser configured for a specific partner profile
func NewParserWithProfile(profileID string, delims Delimiters) *Parser {
	return &Parser{
		delims:    delims,
		profileID: profileID,
	}
}

// Parse parses EDI content from a reader
func (p *Parser) Parse(r io.Reader) (*Document, error) {
	// Read all content
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, fmt.Errorf("failed to read EDI content: %w", err)
	}

	content := buf.Bytes()

	// ! Detect delimiters from ISA segment if not custom
	if p.delims.Element == 0 {
		detectedDelims, err := DetectDelimiters(content)
		if err == nil && detectedDelims.Element != 0 {
			p.delims = detectedDelims
		} else {
			p.delims = DefaultDelimiters()
		}
	}

	segments, err := ParseSegments(content, p.delims)
	if err != nil {
		return nil, fmt.Errorf("failed to parse segments: %w", err)
	}

	doc := &Document{
		Segments:   segments,
		Delimiters: p.delims,
		Metadata:   p.extractMetadata(segments),
	}

	return doc, nil
}

// extractMetadata extracts metadata from parsed segments
func (p *Parser) extractMetadata(segments []Segment) DocumentMetadata {
	meta := DocumentMetadata{}

	for _, seg := range segments {
		switch strings.ToUpper(seg.Tag) {
		case "ISA":
			if len(seg.Elements) >= 13 {
				if len(seg.Elements[5]) > 0 {
					meta.SenderID = strings.TrimSpace(seg.Elements[5][0])
				}
				if len(seg.Elements[7]) > 0 {
					meta.ReceiverID = strings.TrimSpace(seg.Elements[7][0])
				}
				if len(seg.Elements[8]) > 0 {
					meta.Date = seg.Elements[8][0]
				}
				if len(seg.Elements[9]) > 0 {
					meta.Time = seg.Elements[9][0]
				}
				if len(seg.Elements[12]) > 0 {
					meta.ISAControlNumber = seg.Elements[12][0]
				}
			}
		case "GS":
			if len(seg.Elements) >= 8 {
				if len(seg.Elements[5]) > 0 {
					meta.GSControlNumber = seg.Elements[5][0]
				}
				if len(seg.Elements[7]) > 0 {
					meta.Version = seg.Elements[7][0]
				}
			}
		case "ST":
			if len(seg.Elements) >= 2 {
				if len(seg.Elements[0]) > 0 {
					meta.TransactionType = seg.Elements[0][0]
				}
				if len(seg.Elements[1]) > 0 {
					meta.STControlNumber = seg.Elements[1][0]
				}
			}
		}
	}

	return meta
}

// DefaultDelimiters returns the default X12 delimiters
func DefaultDelimiters() Delimiters {
	return Delimiters{
		Element:    '*',
		Component:  ':',
		Segment:    '~',
		Repetition: '^',
	}
}

// Validate performs basic structure validation on the document
func (d *Document) Validate() error {
	if len(d.Segments) == 0 {
		return fmt.Errorf("document has no segments")
	}

	// Check for ISA/IEA envelope
	if d.Segments[0].Tag != "ISA" {
		return fmt.Errorf("document must start with ISA segment")
	}

	// Find last non-empty segment (handle trailing newlines/empty segments)
	lastSegmentIndex := len(d.Segments) - 1
	for lastSegmentIndex > 0 && d.Segments[lastSegmentIndex].Tag == "" {
		lastSegmentIndex--
	}

	if d.Segments[lastSegmentIndex].Tag != "IEA" {
		return fmt.Errorf("document must end with IEA segment")
	}

	// Check for matching control numbers
	isaControl := ""
	ieaControl := ""

	if len(d.Segments[0].Elements) >= 13 && len(d.Segments[0].Elements[12]) > 0 {
		isaControl = d.Segments[0].Elements[12][0]
	}

	lastSeg := d.Segments[lastSegmentIndex]
	if len(lastSeg.Elements) >= 2 && len(lastSeg.Elements[1]) > 0 {
		ieaControl = lastSeg.Elements[1][0]
	}

	if isaControl != ieaControl {
		return fmt.Errorf(
			"ISA control number %s does not match IEA control number %s",
			isaControl,
			ieaControl,
		)
	}

	return nil
}
