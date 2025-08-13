package segments

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
)

// SegmentSchema represents a segment definition loaded from JSON
type SegmentSchema struct {
	ID          string            `json:"id"`          // e.g., "AAA", "ISA", "B2"
	Name        string            `json:"name"`        // e.g., "Request Validation"
	Purpose     string            `json:"purpose"`     // Detailed description
	Position    int               `json:"position"`    // Position in transaction set
	Loop        string            `json:"loop"`        // Loop identifier if applicable
	MaxUse      int               `json:"max_use"`     // Maximum occurrences (-1 = unlimited)
	Version     string            `json:"version"`     // X12 version
	Elements    []ElementSchema   `json:"elements"`    // Element definitions
	Example     string            `json:"example"`     // Example segment
	Notes       []string          `json:"notes"`       // Implementation notes
}

// ElementSchema represents an element definition within a segment
type ElementSchema struct {
	Position    int               `json:"position"`    // e.g., 1 for AAA-01
	RefID       string            `json:"ref_id"`      // e.g., "1073"
	Name        string            `json:"name"`        // e.g., "Yes/No Condition or Response Code"
	Type        string            `json:"type"`        // e.g., "ID", "AN", "DT", "N"
	DataType    string            `json:"data_type"`   // More specific: "Identifier (ID)"
	Requirement string            `json:"requirement"` // "Mandatory", "Optional", "Conditional"
	Required    bool              `json:"required"`    // Alternative boolean field for required
	MinLength   int               `json:"min_length"`  
	MaxLength   int               `json:"max_length"`
	RepeatCount int               `json:"repeat"`      // How many times element can repeat
	Description string            `json:"description"` // Detailed description
	Codes       []CodeValue       `json:"codes"`       // Valid code values for ID types
	Components  []ComponentSchema `json:"components"`  // For composite elements
	Rules       []ValidationRule  `json:"rules"`       // Additional validation rules
}

// ComponentSchema represents a component within a composite element
type ComponentSchema struct {
	Position    int         `json:"position"`
	RefID       string      `json:"ref_id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	MinLength   int         `json:"min_length"`
	MaxLength   int         `json:"max_length"`
	Requirement string      `json:"requirement"`
	Codes       []CodeValue `json:"codes"`
}

// CodeValue represents a valid code for an ID element
type CodeValue struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Notes       string `json:"notes,omitempty"`
}

// ValidationRule represents additional validation logic
type ValidationRule struct {
	Type        string         `json:"type"`        // "conditional", "cross-field", "format"
	Condition   string         `json:"condition"`   // Expression or rule description
	Fields      []string       `json:"fields"`      // Related fields
	Message     string         `json:"message"`     // Error message
	Severity    string         `json:"severity"`    // "error", "warning"
}

// SegmentRegistry manages all segment schemas
type SegmentRegistry struct {
	segments map[string]map[string]*SegmentSchema // version -> segment ID -> schema
	mu       sync.RWMutex
	basePath string
}

// NewSegmentRegistry creates a new registry
func NewSegmentRegistry(basePath string) *SegmentRegistry {
	return &SegmentRegistry{
		segments: make(map[string]map[string]*SegmentSchema),
		basePath: basePath,
	}
}

// LoadFromDirectory loads all segment schemas from a directory structure
// Expected structure: basePath/version/segments/*.json
func (r *SegmentRegistry) LoadFromDirectory() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Walk through version directories
	versionDirs, err := os.ReadDir(r.basePath)
	if err != nil {
		return fmt.Errorf("failed to read base directory: %w", err)
	}

	for _, versionDir := range versionDirs {
		if !versionDir.IsDir() {
			continue
		}

		version := versionDir.Name()
		segmentPath := filepath.Join(r.basePath, version, "segments")
		
		if _, err := os.Stat(segmentPath); os.IsNotExist(err) {
			continue
		}

		// Load all JSON files in the segments directory
		files, err := filepath.Glob(filepath.Join(segmentPath, "*.json"))
		if err != nil {
			return fmt.Errorf("failed to glob segment files for version %s: %w", version, err)
		}

		if r.segments[version] == nil {
			r.segments[version] = make(map[string]*SegmentSchema)
		}

		for _, file := range files {
			schema, err := r.loadSchemaFile(file)
			if err != nil {
				return fmt.Errorf("failed to load schema %s: %w", file, err)
			}
			
			// Set version if not specified in file
			if schema.Version == "" {
				schema.Version = version
			}
			
			r.segments[version][schema.ID] = schema
		}
	}

	return nil
}

// LoadSchema loads a single segment schema from a file
func (r *SegmentRegistry) loadSchemaFile(path string) (*SegmentSchema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var schema SegmentSchema
	if err := sonic.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// GetSegment retrieves a segment schema
func (r *SegmentRegistry) GetSegment(version, segmentID string) (*SegmentSchema, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versionMap, ok := r.segments[version]
	if !ok {
		return nil, fmt.Errorf("version %s not found", version)
	}

	schema, ok := versionMap[strings.ToUpper(segmentID)]
	if !ok {
		return nil, fmt.Errorf("segment %s not found in version %s", segmentID, version)
	}

	return schema, nil
}

// RegisterSegment adds or updates a segment schema
func (r *SegmentRegistry) RegisterSegment(schema *SegmentSchema) {
	r.mu.Lock()
	defer r.mu.Unlock()

	version := schema.Version
	if version == "" {
		version = "004010" // default
	}

	if r.segments[version] == nil {
		r.segments[version] = make(map[string]*SegmentSchema)
	}

	r.segments[version][schema.ID] = schema
}

// ListSegments returns all segment IDs for a version
func (r *SegmentRegistry) ListSegments(version string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versionMap, ok := r.segments[version]
	if !ok {
		return nil
	}

	ids := make([]string, 0, len(versionMap))
	for id := range versionMap {
		ids = append(ids, id)
	}
	return ids
}

// SchemaBuilder builds EDI segments from schemas
type SchemaBuilder struct {
	registry *SegmentRegistry
	version  string
}

// NewSchemaBuilder creates a new schema-based builder
func NewSchemaBuilder(registry *SegmentRegistry, version string) *SchemaBuilder {
	return &SchemaBuilder{
		registry: registry,
		version:  version,
	}
}

// BuildSegment creates a segment string from values
func (b *SchemaBuilder) BuildSegment(segmentID string, values map[string]string, elemSep, compSep byte) (string, error) {
	schema, err := b.registry.GetSegment(b.version, segmentID)
	if err != nil {
		return "", err
	}

	// Start with segment ID
	parts := []string{schema.ID}

	// Process each element in order
	for _, elem := range schema.Elements {
		value := ""
		
		// Try different key formats
		keys := []string{
			fmt.Sprintf("%s%02d", schema.ID, elem.Position),     // e.g., "AAA01"
			fmt.Sprintf("%s-%02d", schema.ID, elem.Position),    // e.g., "AAA-01"
			fmt.Sprintf("%d", elem.Position),                    // e.g., "1"
			elem.RefID,                                          // e.g., "1073"
			elem.Name,                                           // Full name
		}

		for _, key := range keys {
			if v, ok := values[key]; ok {
				value = v
				break
			}
		}

		// Validate value
		if err := b.validateElement(elem, value); err != nil {
			return "", fmt.Errorf("element %s-%02d: %w", schema.ID, elem.Position, err)
		}

		parts = append(parts, value)
	}

	// Trim trailing empty elements
	for i := len(parts) - 1; i > 0; i-- {
		if parts[i] != "" {
			break
		}
		parts = parts[:i]
	}

	return strings.Join(parts, string(elemSep)), nil
}

// validateElement validates a value against element schema
func (b *SchemaBuilder) validateElement(elem ElementSchema, value string) error {
	// Check requirement
	isRequired := strings.ToLower(elem.Requirement) == "mandatory" || 
	              strings.ToLower(elem.Requirement) == "required"
	
	if isRequired && value == "" {
		return fmt.Errorf("%s is required", elem.Name)
	}

	if value == "" {
		return nil // Optional and empty
	}

	// Check length
	if len(value) < elem.MinLength {
		return fmt.Errorf("%s length %d is less than minimum %d", elem.Name, len(value), elem.MinLength)
	}

	if elem.MaxLength > 0 && len(value) > elem.MaxLength {
		return fmt.Errorf("%s length %d exceeds maximum %d", elem.Name, len(value), elem.MaxLength)
	}

	// Check codes for ID types
	if strings.Contains(strings.ToUpper(elem.Type), "ID") && len(elem.Codes) > 0 {
		valid := false
		for _, code := range elem.Codes {
			if code.Code == value {
				valid = true
				break
			}
		}
		if !valid {
			validCodes := make([]string, len(elem.Codes))
			for i, c := range elem.Codes {
				validCodes[i] = c.Code
			}
			return fmt.Errorf("value '%s' not in allowed codes: %v", value, validCodes)
		}
	}

	return nil
}

// ParseToSchema parses raw segment elements using schema
func (b *SchemaBuilder) ParseToSchema(segmentID string, elements [][]string) (map[string]any, error) {
	schema, err := b.registry.GetSegment(b.version, segmentID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any)
	result["_segment_id"] = schema.ID
	result["_segment_name"] = schema.Name
	result["_segment_purpose"] = schema.Purpose

	for i, elem := range schema.Elements {
		if i >= len(elements) {
			break
		}

		elementData := elements[i]
		if len(elementData) == 0 {
			continue
		}

		key := fmt.Sprintf("%s%02d", schema.ID, elem.Position)
		
		// Handle composite elements
		if len(elem.Components) > 0 && len(elementData) > 1 {
			compMap := make(map[string]string)
			for j, comp := range elem.Components {
				if j < len(elementData) {
					compKey := fmt.Sprintf("%s_%02d", comp.RefID, comp.Position)
					compMap[compKey] = elementData[j]
				}
			}
			result[key] = compMap
		} else {
			// Simple element
			value := ""
			if len(elementData) > 0 {
				value = elementData[0]
			}
			
			result[key] = value
			result[key+"_name"] = elem.Name
			
			// Add code description if available
			if strings.Contains(strings.ToUpper(elem.Type), "ID") {
				for _, code := range elem.Codes {
					if code.Code == value {
						result[key+"_description"] = code.Description
						break
					}
				}
			}
		}
	}

	return result, nil
}