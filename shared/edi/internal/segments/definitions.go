package segments

import (
	"fmt"
	"strings"
)

// DataType represents the type of an EDI element
type DataType string

const (
	DataTypeAN DataType = "AN" // Alphanumeric
	DataTypeID DataType = "ID" // Identifier (from code list)
	DataTypeDT DataType = "DT" // Date
	DataTypeTM DataType = "TM" // Time
	DataTypeR  DataType = "R"  // Decimal
	DataTypeN  DataType = "N"  // Numeric
	DataTypeN0 DataType = "N0" // Numeric with implied decimal
	DataTypeB  DataType = "B"  // Binary
)

// Usage indicates whether an element is required, optional, or conditional
type Usage string

const (
	UsageRequired    Usage = "R" // Required
	UsageOptional    Usage = "O" // Optional
	UsageConditional Usage = "C" // Conditional
	UsageNotUsed     Usage = "N" // Not used
)

// ElementDefinition defines a single element within a segment
type ElementDefinition struct {
	ID          string      // Element ID (e.g., "B201", "ISA01")
	Name        string      // Human-readable name
	DataType    DataType    // Data type
	MinLength   int         // Minimum length
	MaxLength   int         // Maximum length
	Usage       Usage       // Required/Optional/Conditional
	Position    int         // Position in segment (1-based)
	CodeList    []string    // Valid values for ID type
	Description string      // Detailed description
	Example     string      // Example value
	Components  []Component // For composite elements
}

// Component defines a sub-element within a composite element
type Component struct {
	ID        string
	Name      string
	DataType  DataType
	MinLength int
	MaxLength int
	Usage     Usage
	Position  int
}

// SegmentDefinition defines the structure of an EDI segment
type SegmentDefinition struct {
	ID        string              // Segment ID (e.g., "ISA", "B2", "ST")
	Name      string              // Human-readable name
	Purpose   string              // Segment purpose/description
	Elements  []ElementDefinition // Element definitions
	MinOccurs int                 // Minimum occurrences in transaction
	MaxOccurs int                 // Maximum occurrences (-1 = unlimited)
	Loop      string              // Loop/group this segment belongs to
	Position  int                 // Position in transaction
	Example   string              // Example segment
	Version   string              // X12 version (004010, 005010, etc.)
}

// Validate checks if a value is valid for this element
func (e *ElementDefinition) Validate(value string) error {
	if e.Usage == UsageRequired && strings.TrimSpace(value) == "" {
		return fmt.Errorf("element %s is required", e.ID)
	}

	if e.Usage == UsageNotUsed && value != "" {
		return fmt.Errorf("element %s should not be used", e.ID)
	}

	if value == "" && e.Usage != UsageRequired {
		return nil
	}

	if len(value) < e.MinLength {
		return fmt.Errorf(
			"element %s length %d is less than minimum %d",
			e.ID,
			len(value),
			e.MinLength,
		)
	}

	if len(value) > e.MaxLength {
		return fmt.Errorf("element %s length %d exceeds maximum %d", e.ID, len(value), e.MaxLength)
	}

	if e.DataType == DataTypeID && len(e.CodeList) > 0 {
		valid := false
		for _, code := range e.CodeList {
			if code == value {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf(
				"element %s value '%s' not in allowed list: %v",
				e.ID,
				value,
				e.CodeList,
			)
		}
	}

	switch e.DataType {
	case DataTypeDT:
		if len(value) != 8 && len(value) != 6 {
			return fmt.Errorf("element %s date must be CCYYMMDD or YYMMDD format", e.ID)
		}
	case DataTypeTM:
		if len(value) != 4 && len(value) != 6 {
			return fmt.Errorf("element %s time must be HHMM or HHMMSS format", e.ID)
		}
	case DataTypeN, DataTypeN0:
		for _, c := range value {
			if c < '0' || c > '9' {
				return fmt.Errorf("element %s must be numeric", e.ID)
			}
		}
	}

	return nil
}

// FormatValue formats a value according to the element definition
func (e *ElementDefinition) FormatValue(value string) string {
	formatted := value

	if (e.DataType == DataTypeN || e.DataType == DataTypeN0) && len(value) < e.MinLength {
		formatted = strings.Repeat("0", e.MinLength-len(value)) + value
	}

	if e.DataType == DataTypeAN && len(value) < e.MinLength {
		formatted = value + strings.Repeat(" ", e.MinLength-len(value))
	}

	if len(formatted) > e.MaxLength {
		formatted = formatted[:e.MaxLength]
	}

	return formatted
}

// SegmentBuilder helps construct segments from definitions
type SegmentBuilder struct {
	definition *SegmentDefinition
	values     map[int]string   // Position -> value
	components map[int][]string // Position -> component values
}

// NewSegmentBuilder creates a builder for a segment
func NewSegmentBuilder(def *SegmentDefinition) *SegmentBuilder {
	return &SegmentBuilder{
		definition: def,
		values:     make(map[int]string),
		components: make(map[int][]string),
	}
}

// SetElement sets the value for an element at position (1-based)
func (b *SegmentBuilder) SetElement(position int, value string) error {
	if position < 1 || position > len(b.definition.Elements) {
		return fmt.Errorf("invalid position %d for segment %s", position, b.definition.ID)
	}

	elem := b.definition.Elements[position-1]
	if err := elem.Validate(value); err != nil {
		return fmt.Errorf("segment %s position %d: %w", b.definition.ID, position, err)
	}

	b.values[position] = elem.FormatValue(value)
	return nil
}

// SetElementByID sets the value for an element by its ID
func (b *SegmentBuilder) SetElementByID(elementID string, value string) error {
	for i, elem := range b.definition.Elements {
		if elem.ID == elementID {
			return b.SetElement(i+1, value)
		}
	}
	return fmt.Errorf("element %s not found in segment %s", elementID, b.definition.ID)
}

// SetComponents sets component values for a composite element
func (b *SegmentBuilder) SetComponents(position int, values []string) error {
	if position < 1 || position > len(b.definition.Elements) {
		return fmt.Errorf("invalid position %d for segment %s", position, b.definition.ID)
	}

	elem := b.definition.Elements[position-1]
	if len(elem.Components) == 0 {
		return fmt.Errorf("element at position %d is not composite", position)
	}

	if len(values) > len(elem.Components) {
		return fmt.Errorf("too many components: got %d, max %d", len(values), len(elem.Components))
	}

	b.components[position] = values
	return nil
}

// Build constructs the segment string with the given delimiters
func (b *SegmentBuilder) Build(elemSep, compSep byte) string {
	var parts []string

	parts = append(parts, b.definition.ID)

	maxPos := len(b.definition.Elements)
	for i := 1; i <= maxPos; i++ {
		if comps, hasComps := b.components[i]; hasComps {
			compParts := make([]string, len(comps))
			copy(compParts, comps)
			parts = append(parts, strings.Join(compParts, string(compSep)))
		} else if val, hasVal := b.values[i]; hasVal {
			parts = append(parts, val)
		} else {
			parts = append(parts, "")
		}
	}

	for i := len(parts) - 1; i > 0; i-- {
		if parts[i] != "" {
			break
		}
		parts = parts[:i]
	}

	return strings.Join(parts, string(elemSep))
}

// Validate checks if all required elements are set
func (b *SegmentBuilder) Validate() error {
	for i, elem := range b.definition.Elements {
		pos := i + 1
		if elem.Usage == UsageRequired {
			if _, hasVal := b.values[pos]; !hasVal {
				if _, hasComps := b.components[pos]; !hasComps {
					return fmt.Errorf("required element %s at position %d not set", elem.ID, pos)
				}
			}
		}
	}
	return nil
}

// SegmentParser parses raw segment data using definitions
type SegmentParser struct {
	definition *SegmentDefinition
}

// NewSegmentParser creates a parser for a segment definition
func NewSegmentParser(def *SegmentDefinition) *SegmentParser {
	return &SegmentParser{definition: def}
}

// Parse parses raw element values into a structured format
func (p *SegmentParser) Parse(elements [][]string) (map[string]any, error) {
	result := make(map[string]any)
	result["_segment"] = p.definition.ID
	result["_name"] = p.definition.Name

	for i, elem := range p.definition.Elements {
		if i >= len(elements) {
			if elem.Usage == UsageRequired {
				return nil, fmt.Errorf("required element %s at position %d missing", elem.ID, i+1)
			}
			continue
		}

		elementData := elements[i]
		if len(elementData) == 0 {
			continue
		}

		if len(elem.Components) > 0 && len(elementData) > 1 {
			compMap := make(map[string]string)
			for j, comp := range elem.Components {
				if j < len(elementData) {
					compMap[comp.ID] = elementData[j]
				}
			}
			result[elem.ID] = compMap
		} else {
			value := ""
			if len(elementData) > 0 {
				value = elementData[0]
			}

			if err := elem.Validate(value); err != nil {
				return nil, fmt.Errorf("validation failed: %w", err)
			}

			result[elem.ID] = value
		}
	}

	return result, nil
}
