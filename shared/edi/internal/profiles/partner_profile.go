package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// PartnerProfile defines all partner-specific EDI configuration
type PartnerProfile struct {
	// These fields are removed as they're stored in the DB model:
	// PartnerID, PartnerName, Active, Description
	Format                FormatConfig         `json:"format"`
	SupportedTransactions []TransactionSupport `json:"supported_transactions"`
	ValidationConfig      ValidationConfig     `json:"validation"`
	ReferenceConfig       ReferenceConfig      `json:"references"`
	PartyRoles            map[string][]string  `json:"party_roles"`
	BusinessRules         []BusinessRule       `json:"business_rules,omitempty"`
	Transformations       TransformationConfig `json:"transformations,omitzero"`
	CommunicationConfig   CommunicationConfig  `json:"communication,omitzero"`
}

// FormatConfig defines EDI format specifications
type FormatConfig struct {
	Delimiters         DelimiterConfig `json:"delimiters"`
	Encoding           string          `json:"encoding,omitempty"`           // UTF-8, ASCII, etc.
	LineEnding         string          `json:"line_ending,omitempty"`        // CR, LF, CRLF, NONE
	SegmentTerminator  string          `json:"segment_terminator,omitempty"` // Can be hex like \x0A for newline
	ISAFieldLengths    []int           `json:"isa_field_lengths,omitempty"`  // Fixed widths for ISA fields
	PadCharacter       string          `json:"pad_character,omitempty"`      // Character for padding
	TrimTrailingSpaces bool            `json:"trim_trailing_spaces,omitempty"`
	PreserveWhitespace bool            `json:"preserve_whitespace,omitempty"`
}

// DelimiterConfig supports multiple delimiter representations
type DelimiterConfig struct {
	Element       string `json:"element"`
	Component     string `json:"component,omitempty"`
	Segment       string `json:"segment"`
	Repetition    string `json:"repetition,omitempty"`
	ElementHex    string `json:"element_hex,omitempty"`
	ComponentHex  string `json:"component_hex,omitempty"`
	SegmentHex    string `json:"segment_hex,omitempty"`
	RepetitionHex string `json:"repetition_hex,omitempty"`
}

// TransactionSupport defines support for specific transaction types
type TransactionSupport struct {
	TransactionType    string            `json:"transaction_type"` // 204, 214, 990, etc.
	Versions           []string          `json:"versions"`         // 004010, 005010, etc.
	SchemaPath         string            `json:"schema_path,omitempty"`
	Required           bool              `json:"required"`
	Direction          string            `json:"direction"` // inbound, outbound, both
	RequiredSegments   []string          `json:"required_segments,omitempty"`
	OptionalSegments   []string          `json:"optional_segments,omitempty"`
	ProhibitedSegments []string          `json:"prohibited_segments,omitempty"`
	LoopRequirements   []LoopRequirement `json:"loop_requirements,omitempty"`
}

// LoopRequirement defines loop-specific requirements
type LoopRequirement struct {
	LoopID    string `json:"loop_id"`
	MinOccurs int    `json:"min_occurs"`
	MaxOccurs int    `json:"max_occurs"`
	Required  bool   `json:"required"`
}

// ValidationConfig defines validation rules
type ValidationConfig struct {
	Strictness               string           `json:"strictness"` // strict, lenient, custom
	EnforceSegmentOrder      bool             `json:"enforce_segment_order"`
	EnforceSegmentCounts     bool             `json:"enforce_segment_counts"`
	AllowUnknownSegments     bool             `json:"allow_unknown_segments"`
	EnforceRequiredElements  bool             `json:"enforce_required_elements"`
	EnforceElementFormats    bool             `json:"enforce_element_formats"`
	EnforceElementLengths    bool             `json:"enforce_element_lengths"`
	ValidateControlNumbers   bool             `json:"validate_control_numbers"`
	UniqueControlNumbers     bool             `json:"unique_control_numbers"`
	RequirePickupAndDelivery bool             `json:"require_pickup_and_delivery,omitempty"`
	RequireShipmentID        bool             `json:"require_shipment_id,omitempty"`
	CustomRules              []ValidationRule `json:"custom_rules,omitempty"`
}

// ValidationRule defines a custom validation rule
type ValidationRule struct {
	RuleID          string `json:"rule_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Severity        string `json:"severity"` // error, warning, info
	SegmentTag      string `json:"segment_tag,omitempty"`
	ElementPosition int    `json:"element_position,omitempty"`
	Condition       string `json:"condition"` // regex, value match, etc.
	ErrorMessage    string `json:"error_message"`
}

// ReferenceConfig defines reference number handling
type ReferenceConfig struct {
	CustomerPO       []string          `json:"customer_po"`
	BillOfLading     []string          `json:"bill_of_lading"`
	ShipmentRef      []string          `json:"shipment_ref"`
	ShipmentIDQuals  []string          `json:"shipment_id_quals"`
	ShipmentIDMode   string            `json:"shipment_id_mode"` // ref_first, always_b2, etc.
	CustomReferences map[string]string `json:"custom_references,omitempty"`
}

// BusinessRule defines partner-specific business logic
type BusinessRule struct {
	RuleID    string            `json:"rule_id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"` // mapping, validation, transformation
	Condition map[string]string `json:"condition,omitempty"`
	Action    map[string]string `json:"action"`
	Priority  int               `json:"priority,omitempty"`
}

// TransformationConfig defines data transformations
type TransformationConfig struct {
	EmitISODateTime  bool                      `json:"emit_iso_datetime"`
	Timezone         string                    `json:"timezone"`
	DateFormat       string                    `json:"date_format,omitempty"`
	TimeFormat       string                    `json:"time_format,omitempty"`
	ServiceLevelMap  map[string]string         `json:"service_level_map,omitempty"`
	EquipmentTypeMap map[string]string         `json:"equipment_type_map,omitempty"`
	StopTypeMap      map[string]string         `json:"stop_type_map,omitempty"`
	AccessorialMap   map[string]string         `json:"accessorial_map,omitempty"`
	DefaultValues    map[string]map[int]string `json:"default_values,omitempty"`
	FieldTransforms  []FieldTransform          `json:"field_transforms,omitempty"`
}

// FieldTransform defines field-level transformations
type FieldTransform struct {
	SegmentTag      string            `json:"segment_tag"`
	ElementPosition int               `json:"element_position"`
	TransformType   string            `json:"transform_type"` // uppercase, lowercase, trim, pad, etc.
	Parameters      map[string]string `json:"parameters,omitempty"`
}

// CommunicationConfig defines communication settings
type CommunicationConfig struct {
	Protocol            string `json:"protocol"` // FTP, SFTP, AS2, API, etc.
	Host                string `json:"host,omitempty"`
	Port                int    `json:"port,omitempty"`
	Path                string `json:"path,omitempty"`
	Username            string `json:"username,omitempty"`
	AuthMethod          string `json:"auth_method,omitempty"` // password, key, certificate
	InboundFilePattern  string `json:"inbound_file_pattern,omitempty"`
	OutboundFilePattern string `json:"outbound_file_pattern,omitempty"`
	AutoAcknowledge     bool   `json:"auto_acknowledge"`
	AckTimeout          int    `json:"ack_timeout_minutes,omitempty"`
}

// ProfileEntry contains a profile with its partner ID
type ProfileEntry struct {
	PartnerID string
	Profile   *PartnerProfile
}

// ProfileManager manages partner profiles
type ProfileManager struct {
	profiles map[string]*ProfileEntry // map[partnerID]*ProfileEntry
	basePath string
}

// NewProfileManager creates a new profile manager
func NewProfileManager(basePath string) *ProfileManager {
	return &ProfileManager{
		profiles: make(map[string]*ProfileEntry),
		basePath: basePath,
	}
}

// LoadProfile loads a partner profile from file
// Returns the profile and the partner ID extracted from the filename
func (m *ProfileManager) LoadProfile(filename string) (*PartnerProfile, string, error) {
	path := filepath.Join(m.basePath, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read profile file: %w", err)
	}

	// Extract partner ID from filename (e.g., "meritor-4010.json" -> "meritor-4010")
	partnerID := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))

	var profile PartnerProfile
	if err := sonic.Unmarshal(data, &profile); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	if err := m.processProfile(&profile); err != nil {
		return nil, "", fmt.Errorf("invalid profile: %w", err)
	}

	m.profiles[partnerID] = &ProfileEntry{
		PartnerID: partnerID,
		Profile:   &profile,
	}

	return &profile, partnerID, nil
}

// processProfile processes special configurations in the profile
func (m *ProfileManager) processProfile(profile *PartnerProfile) error {
	if err := m.processDelimiters(profile); err != nil {
		return err
	}

	// Set defaults for profile fields
	if profile.Format.Encoding == "" {
		profile.Format.Encoding = "UTF-8"
	}

	if profile.ValidationConfig.Strictness == "" {
		profile.ValidationConfig.Strictness = "strict"
	}

	return nil
}

// processDelimiters processes delimiter configurations including hex values
func (m *ProfileManager) processDelimiters(profile *PartnerProfile) error {
	delims := &profile.Format.Delimiters

	if delims.ElementHex != "" {
		delims.Element = m.hexToString(delims.ElementHex)
	}
	if delims.ComponentHex != "" {
		delims.Component = m.hexToString(delims.ComponentHex)
	}
	if delims.SegmentHex != "" {
		delims.Segment = m.hexToString(delims.SegmentHex)
	}
	if delims.RepetitionHex != "" {
		delims.Repetition = m.hexToString(delims.RepetitionHex)
	}

	if profile.Format.SegmentTerminator != "" {
		switch profile.Format.SegmentTerminator {
		case "\\n", "LF":
			delims.Segment = "\n"
		case "\\r", "CR":
			delims.Segment = "\r"
		case "\\r\\n", "CRLF":
			delims.Segment = "\r\n"
		default:
			if strings.HasPrefix(profile.Format.SegmentTerminator, "0x") ||
				strings.HasPrefix(profile.Format.SegmentTerminator, "\\x") {
				delims.Segment = m.hexToString(profile.Format.SegmentTerminator)
			}
		}
	}

	if delims.Element == "" {
		return fmt.Errorf("element delimiter is required")
	}
	if delims.Segment == "" {
		return fmt.Errorf("segment delimiter is required")
	}

	return nil
}

// hexToString converts hex representation to string
func (m *ProfileManager) hexToString(hex string) string {
	hex = strings.TrimPrefix(hex, "0x")
	hex = strings.TrimPrefix(hex, "\\x")

	if len(hex) == 2 {
		var b byte
		fmt.Sscanf(hex, "%02x", &b)
		return string(b)
	}

	return hex
}

// GetProfile retrieves a partner profile
func (m *ProfileManager) GetProfile(partnerID string) (*PartnerProfile, error) {
	entry, exists := m.profiles[partnerID]
	if !exists {
		return nil, fmt.Errorf("profile not found for partner: %s", partnerID)
	}
	return entry.Profile, nil
}

// GetDelimiters returns X12 delimiters from profile
func (p *PartnerProfile) GetDelimiters() x12.Delimiters {
	delims := x12.Delimiters{}

	if len(p.Format.Delimiters.Element) > 0 {
		delims.Element = p.Format.Delimiters.Element[0]
	}
	if len(p.Format.Delimiters.Component) > 0 {
		delims.Component = p.Format.Delimiters.Component[0]
	}
	if len(p.Format.Delimiters.Segment) > 0 {
		delims.Segment = p.Format.Delimiters.Segment[0]
	}
	if len(p.Format.Delimiters.Repetition) > 0 {
		delims.Repetition = p.Format.Delimiters.Repetition[0]
	}

	return delims
}

// SaveProfile saves a partner profile to file with the given partner ID
func (m *ProfileManager) SaveProfile(partnerID string, profile *PartnerProfile) error {
	filename := fmt.Sprintf("%s.json", strings.ToLower(partnerID))
	path := filepath.Join(m.basePath, filename)

	data, err := sonic.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write profile file: %w", err)
	}

	m.profiles[partnerID] = &ProfileEntry{
		PartnerID: partnerID,
		Profile:   profile,
	}

	return nil
}

// ListProfiles returns all loaded profiles with their IDs
func (m *ProfileManager) ListProfiles() []*ProfileEntry {
	profiles := make([]*ProfileEntry, 0, len(m.profiles))
	for _, p := range m.profiles {
		profiles = append(profiles, p)
	}
	return profiles
}
