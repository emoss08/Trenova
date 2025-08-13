package config

import (
	"fmt"

	"github.com/bytedance/sonic"
)

// TransactionConfig defines how a specific transaction type should be parsed and built
type TransactionConfig struct {
	TransactionType string                 `json:"transaction_type"` // e.g., "204", "997", "999"
	Version         string                 `json:"version"`          // e.g., "004010", "005010"
	Name            string                 `json:"name"`             // Human-readable name
	Description     string                 `json:"description"`
	
	// Structure defines the expected structure of the transaction
	Structure       TransactionStructure   `json:"structure"`
	
	// Mapping defines how to map between EDI segments and business objects
	Mappings        []SegmentMapping       `json:"mappings"`
	
	// Validation rules beyond standard X12 validation
	ValidationRules []ValidationRule       `json:"validation_rules"`
	
	// Customer-specific overrides
	CustomerOverrides map[string]CustomerConfig `json:"customer_overrides,omitempty"`
	
	// LoopMappings defines where to find loop data in business objects
	// Key is the loop start segment (e.g., "N1", "S5"), value is the path in the data
	LoopMappings map[string]string `json:"loop_mappings,omitempty"`
}

// TransactionStructure defines the expected structure of segments in a transaction
type TransactionStructure struct {
	// Required segments in order
	RequiredSegments []SegmentRequirement `json:"required_segments"`
	
	// Loops define repeating groups of segments
	Loops           []LoopDefinition     `json:"loops"`
	
	// Conditional segments based on business rules
	ConditionalSegments []ConditionalSegment `json:"conditional_segments"`
}

// SegmentRequirement defines requirements for a segment
type SegmentRequirement struct {
	SegmentID    string `json:"segment_id"`
	MinOccurs    int    `json:"min_occurs"`
	MaxOccurs    int    `json:"max_occurs"`
	Position     int    `json:"position"`      // Expected position in transaction
	Required     bool   `json:"required"`
	Description  string `json:"description"`
}

// LoopDefinition defines a loop (repeating group) in the transaction
type LoopDefinition struct {
	LoopID       string               `json:"loop_id"`
	Name         string               `json:"name"`
	MinOccurs    int                  `json:"min_occurs"`
	MaxOccurs    int                  `json:"max_occurs"`
	StartSegment string               `json:"start_segment"` // Segment that starts the loop
	Segments     []SegmentRequirement `json:"segments"`      // Segments within the loop
	NestedLoops  []LoopDefinition     `json:"nested_loops,omitempty"`
}

// ConditionalSegment defines when a segment is required based on conditions
type ConditionalSegment struct {
	SegmentID   string    `json:"segment_id"`
	Condition   Condition `json:"condition"`
	Required    bool      `json:"required"`
	Description string    `json:"description"`
}

// Condition defines when something applies
type Condition struct {
	Type       string      `json:"type"`       // "equals", "not_equals", "exists", "contains"
	Field      string      `json:"field"`      // Field path to check
	Value      any         `json:"value"`      // Value to compare
	AndConditions []Condition `json:"and,omitempty"`
	OrConditions  []Condition `json:"or,omitempty"`
}

// SegmentMapping defines how to map EDI segments to/from business objects
type SegmentMapping struct {
	SegmentID    string            `json:"segment_id"`
	ObjectPath   string            `json:"object_path"`   // Path in business object
	Direction    string            `json:"direction"`     // "inbound", "outbound", "both"
	Elements     []ElementMapping  `json:"elements"`
	Transform    string            `json:"transform,omitempty"`     // Transformation function name
	DefaultValue any               `json:"default_value,omitempty"`
}

// ElementMapping defines how to map individual elements
type ElementMapping struct {
	ElementPosition int    `json:"element_position"`
	ObjectField     string `json:"object_field"`
	DataType        string `json:"data_type"`       // "string", "number", "date", "time", "boolean"
	Format          string `json:"format,omitempty"` // For dates/times
	Transform       string `json:"transform,omitempty"`
	DefaultValue    any    `json:"default_value,omitempty"`
	Required        bool   `json:"required"`
	Validation      string `json:"validation,omitempty"` // Regex or validation rule
}

// ValidationRule defines custom validation rules
type ValidationRule struct {
	RuleID      string    `json:"rule_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // "error", "warning", "info"
	Type        string    `json:"type"`     // "field", "cross_field", "business_rule"
	Condition   Condition `json:"condition"`
	Message     string    `json:"message"`
	ErrorCode   string    `json:"error_code"`
}

// CustomerConfig defines customer-specific configuration
type CustomerConfig struct {
	CustomerID   string                        `json:"customer_id"`
	CustomerName string                        `json:"customer_name"`
	Active       bool                          `json:"active"`
	
	// Override segment requirements
	SegmentOverrides map[string]SegmentRequirement `json:"segment_overrides,omitempty"`
	
	// Additional validation rules
	AdditionalRules []ValidationRule              `json:"additional_rules,omitempty"`
	
	// Custom mappings
	CustomMappings []SegmentMapping              `json:"custom_mappings,omitempty"`
	
	// Default values for elements
	DefaultValues map[string]map[int]any        `json:"default_values,omitempty"` // segmentID -> elementPos -> value
	
	// Transformation rules
	Transformations []TransformationRule         `json:"transformations,omitempty"`
}

// TransformationRule defines how to transform data
type TransformationRule struct {
	RuleID      string `json:"rule_id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // "uppercase", "lowercase", "trim", "pad", "replace", "format"
	Field       string `json:"field"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// ConfigManager manages transaction configurations
type ConfigManager struct {
	configs   map[string]*TransactionConfig // key: "type:version" e.g., "204:004010"
	templates map[string]*MappingTemplate
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configs:   make(map[string]*TransactionConfig),
		templates: make(map[string]*MappingTemplate),
	}
}

// LoadConfig loads a transaction configuration from JSON
func (m *ConfigManager) LoadConfig(data []byte) error {
	var config TransactionConfig
	if err := sonic.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	key := fmt.Sprintf("%s:%s", config.TransactionType, config.Version)
	m.configs[key] = &config
	return nil
}

// GetConfig retrieves a transaction configuration
func (m *ConfigManager) GetConfig(transactionType, version string) (*TransactionConfig, error) {
	key := fmt.Sprintf("%s:%s", transactionType, version)
	config, exists := m.configs[key]
	if !exists {
		return nil, fmt.Errorf("configuration not found for %s", key)
	}
	return config, nil
}

// GetCustomerConfig retrieves customer-specific configuration
func (m *ConfigManager) GetCustomerConfig(transactionType, version, customerID string) (*CustomerConfig, error) {
	config, err := m.GetConfig(transactionType, version)
	if err != nil {
		return nil, err
	}
	
	customerConfig, exists := config.CustomerOverrides[customerID]
	if !exists {
		// Return empty config if no overrides
		return &CustomerConfig{
			CustomerID: customerID,
			Active:     true,
		}, nil
	}
	
	return &customerConfig, nil
}

// MappingTemplate defines a reusable mapping template
type MappingTemplate struct {
	TemplateID   string           `json:"template_id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	BaseType     string           `json:"base_type"` // Transaction type this is based on
	Mappings     []SegmentMapping `json:"mappings"`
	Transformations []TransformationRule `json:"transformations"`
}

// Example204Config creates a default 204 configuration
func Example204Config() *TransactionConfig {
	return &TransactionConfig{
		TransactionType: "204",
		Version:         "004010",
		Name:            "Motor Carrier Load Tender",
		Description:     "Used to tender a shipment to a motor carrier",
		Structure: TransactionStructure{
			RequiredSegments: []SegmentRequirement{
				{
					SegmentID:   "ST",
					MinOccurs:   1,
					MaxOccurs:   1,
					Position:    1,
					Required:    true,
					Description: "Transaction Set Header",
				},
				{
					SegmentID:   "B2",
					MinOccurs:   1,
					MaxOccurs:   1,
					Position:    2,
					Required:    true,
					Description: "Beginning Segment for Shipment Information Transaction",
				},
				{
					SegmentID:   "SE",
					MinOccurs:   1,
					MaxOccurs:   1,
					Position:    9999,
					Required:    true,
					Description: "Transaction Set Trailer",
				},
			},
			Loops: []LoopDefinition{
				{
					LoopID:       "N1",
					Name:         "Party Identification",
					MinOccurs:    1,
					MaxOccurs:    10,
					StartSegment: "N1",
					Segments: []SegmentRequirement{
						{
							SegmentID:   "N1",
							MinOccurs:   1,
							MaxOccurs:   1,
							Required:    true,
							Description: "Party Name",
						},
						{
							SegmentID:   "N3",
							MinOccurs:   0,
							MaxOccurs:   2,
							Required:    false,
							Description: "Party Address",
						},
						{
							SegmentID:   "N4",
							MinOccurs:   0,
							MaxOccurs:   1,
							Required:    false,
							Description: "Geographic Location",
						},
						{
							SegmentID:   "G61",
							MinOccurs:   0,
							MaxOccurs:   3,
							Required:    false,
							Description: "Contact",
						},
					},
				},
				{
					LoopID:       "S5",
					Name:         "Stop Off Details",
					MinOccurs:    2,
					MaxOccurs:    999,
					StartSegment: "S5",
					Segments: []SegmentRequirement{
						{
							SegmentID:   "S5",
							MinOccurs:   1,
							MaxOccurs:   1,
							Required:    true,
							Description: "Stop Off Details",
						},
						{
							SegmentID:   "G62",
							MinOccurs:   0,
							MaxOccurs:   2,
							Required:    false,
							Description: "Date/Time",
						},
					},
				},
			},
			ConditionalSegments: []ConditionalSegment{
				{
					SegmentID: "B2A",
					Condition: Condition{
						Type:  "not_equals",
						Field: "B2.01",
						Value: "00",
					},
					Required:    true,
					Description: "Set Purpose required when not original",
				},
			},
		},
		Mappings: []SegmentMapping{
			{
				SegmentID:  "ST",
				ObjectPath: "",
				Direction:  "outbound",
				Elements: []ElementMapping{
					{
						ElementPosition: 1,
						ObjectField:     "",
						DataType:        "string",
						Required:        true,
						DefaultValue:    "204",
					},
					{
						ElementPosition: 2,
						ObjectField:     "",
						DataType:        "string",
						Required:        true,
						DefaultValue:    "00001",
					},
				},
			},
			{
				SegmentID:  "SE",
				ObjectPath: "",
				Direction:  "outbound",
				Elements: []ElementMapping{
					{
						ElementPosition: 1,
						ObjectField:     "",
						DataType:        "string",
						Required:        true,
						DefaultValue:    "0", // Will be calculated
					},
					{
						ElementPosition: 2,
						ObjectField:     "",
						DataType:        "string",
						Required:        true,
						DefaultValue:    "00001",
					},
				},
			},
			{
				SegmentID:  "B2",
				ObjectPath: "shipment",
				Direction:  "both",
				Elements: []ElementMapping{
					{
						ElementPosition: 1,
						ObjectField:     "tariff_service_code",
						DataType:        "string",
						Required:        false,
					},
					{
						ElementPosition: 2,
						ObjectField:     "scac",
						DataType:        "string",
						Required:        true,
						Transform:       "uppercase",
					},
					{
						ElementPosition: 3,
						ObjectField:     "shipment_id",
						DataType:        "string",
						Required:        true,
					},
					{
						ElementPosition: 4,
						ObjectField:     "payment_method",
						DataType:        "string",
						Required:        true,
						Validation:      "^(CC|PP|TP|DC)$",
					},
				},
			},
			{
				SegmentID:  "B2A",
				ObjectPath: "shipment",
				Direction:  "both",
				Elements: []ElementMapping{
					{
						ElementPosition: 1,
						ObjectField:     "purpose_code",
						DataType:        "string",
						Required:        false,
						DefaultValue:    "00",
					},
					{
						ElementPosition: 2,
						ObjectField:     "trans_method",
						DataType:        "string",
						Required:        false,
						DefaultValue:    "LT",
					},
				},
			},
			{
				SegmentID:  "N1",
				ObjectPath: "parties[]",
				Direction:  "both",
				Elements: []ElementMapping{
					{
						ElementPosition: 1,
						ObjectField:     "entity_code",
						DataType:        "string",
						Required:        true,
						Validation:      "^(SH|CN|BT|SF|ST)$",
					},
					{
						ElementPosition: 2,
						ObjectField:     "name",
						DataType:        "string",
						Required:        true,
					},
					{
						ElementPosition: 3,
						ObjectField:     "id_qualifier",
						DataType:        "string",
						Required:        false,
						DefaultValue:    "93",
					},
					{
						ElementPosition: 4,
						ObjectField:     "id_code",
						DataType:        "string",
						Required:        false,
					},
				},
			},
			{
				SegmentID:  "S5",
				ObjectPath: "stops[]",
				Direction:  "both",
				Elements: []ElementMapping{
					{
						ElementPosition: 1,
						ObjectField:     "stop_number",
						DataType:        "integer",
						Required:        true,
					},
					{
						ElementPosition: 2,
						ObjectField:     "reason_code",
						DataType:        "string",
						Required:        true,
						Validation:      "^(CL|CU|LD|UL)$",
					},
				},
			},
		},
		ValidationRules: []ValidationRule{
			{
				RuleID:      "204_SHIPPER_REQUIRED",
				Name:        "Shipper Required",
				Description: "204 must have at least one shipper (N1*SH or N1*SF)",
				Severity:    "error",
				Type:        "business_rule",
				Condition: Condition{
					Type:  "not_exists",
					Field: "N1[01=SH|SF]",
				},
				Message:   "204 transaction must include a shipper (N1*SH or N1*SF)",
				ErrorCode: "MISSING_SHIPPER",
			},
			{
				RuleID:      "204_CONSIGNEE_REQUIRED",
				Name:        "Consignee Required",
				Description: "204 must have at least one consignee (N1*CN)",
				Severity:    "error",
				Type:        "business_rule",
				Condition: Condition{
					Type:  "not_exists",
					Field: "N1[01=CN]",
				},
				Message:   "204 transaction must include a consignee (N1*CN)",
				ErrorCode: "MISSING_CONSIGNEE",
			},
			{
				RuleID:      "204_MIN_STOPS",
				Name:        "Minimum Stops",
				Description: "204 must have at least 2 stops (pickup and delivery)",
				Severity:    "error",
				Type:        "business_rule",
				Condition: Condition{
					Type:  "count_less_than",
					Field: "stops",
					Value: 2,
				},
				Message:   "204 transaction must have at least 2 stops",
				ErrorCode: "INSUFFICIENT_STOPS",
			},
		},
	}
}

// Example997Config creates a default 997 configuration
func Example997Config() *TransactionConfig {
	return &TransactionConfig{
		TransactionType: "997",
		Version:         "004010",
		Name:            "Functional Acknowledgment",
		Description:     "Used to acknowledge receipt and syntactical acceptability of a functional group",
		Structure: TransactionStructure{
			RequiredSegments: []SegmentRequirement{
				{
					SegmentID:   "ST",
					MinOccurs:   1,
					MaxOccurs:   1,
					Position:    1,
					Required:    true,
					Description: "Transaction Set Header",
				},
				{
					SegmentID:   "AK1",
					MinOccurs:   1,
					MaxOccurs:   1,
					Position:    2,
					Required:    true,
					Description: "Functional Group Response Header",
				},
				{
					SegmentID:   "AK9",
					MinOccurs:   1,
					MaxOccurs:   1,
					Position:    998,
					Required:    true,
					Description: "Functional Group Response Trailer",
				},
				{
					SegmentID:   "SE",
					MinOccurs:   1,
					MaxOccurs:   1,
					Position:    999,
					Required:    true,
					Description: "Transaction Set Trailer",
				},
			},
			Loops: []LoopDefinition{
				{
					LoopID:       "AK2",
					Name:         "Transaction Set Response",
					MinOccurs:    0,
					MaxOccurs:    999999,
					StartSegment: "AK2",
					Segments: []SegmentRequirement{
						{
							SegmentID:   "AK2",
							MinOccurs:   1,
							MaxOccurs:   1,
							Required:    true,
							Description: "Transaction Set Response Header",
						},
						{
							SegmentID:   "AK3",
							MinOccurs:   0,
							MaxOccurs:   999,
							Required:    false,
							Description: "Data Segment Note",
						},
						{
							SegmentID:   "AK4",
							MinOccurs:   0,
							MaxOccurs:   99,
							Required:    false,
							Description: "Data Element Note",
						},
						{
							SegmentID:   "AK5",
							MinOccurs:   1,
							MaxOccurs:   1,
							Required:    true,
							Description: "Transaction Set Response Trailer",
						},
					},
				},
			},
		},
	}
}

// SaveConfig saves a configuration to storage
func (m *ConfigManager) SaveConfig(config *TransactionConfig) error {
	data, err := sonic.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// In a real implementation, this would save to database or file
	key := fmt.Sprintf("%s:%s", config.TransactionType, config.Version)
	m.configs[key] = config
	
	fmt.Printf("Saved configuration for %s\n", key)
	fmt.Printf("%s\n", string(data))
	
	return nil
}