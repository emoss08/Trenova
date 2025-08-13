package services

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/core"
	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"github.com/emoss08/trenova/shared/edi/internal/registry"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// IntegratedParser combines profile-aware parsing with validation
type IntegratedParser struct {
	profileManager  *profiles.ProfileManager
	profileParser   *ProfileParser
	validationSvc   *ValidationService
	ackHandler      *core.AcknowledgmentHandler
	segmentRegistry *segments.SegmentRegistry
	configManager   *config.ConfigManager
}

// IntegratedParserOptions configures the integrated parser
type IntegratedParserOptions struct {
	ProfilePath      string
	SchemaPath       string
	StrictMode       bool
	AutoAck          bool // Automatically generate acknowledgments
	ValidateProfiles bool // Validate profiles on load
}

// NewIntegratedParser creates a new integrated parser with all services
func NewIntegratedParser(opts IntegratedParserOptions) (*IntegratedParser, error) {
	segRegistry := segments.NewSegmentRegistry(opts.SchemaPath)
	if err := segRegistry.LoadFromDirectory(); err != nil {
		return nil, fmt.Errorf("failed to load segment schemas: %w", err)
	}

	delims := x12.DefaultDelimiters()
	txRegistry := registry.NewTransactionRegistry(segRegistry, delims)
	configMgr := config.NewConfigManager()
	profileMgr := profiles.NewProfileManager(opts.ProfilePath)
	cfg204 := config.Example204Config()
	configMgr.SaveConfig(cfg204)
	txRegistry.RegisterConfig(cfg204)
	cfg997 := config.Example997Config()
	configMgr.SaveConfig(cfg997)
	txRegistry.RegisterConfig(cfg997)
	profileParser := NewProfileParser(profileMgr)
	validationSvc := NewValidationService(txRegistry, segRegistry, configMgr)
	ackHandler := core.NewAcknowledgmentHandler(segRegistry, profileMgr)

	return &IntegratedParser{
		profileManager:  profileMgr,
		profileParser:   profileParser,
		validationSvc:   validationSvc,
		ackHandler:      ackHandler,
		segmentRegistry: segRegistry,
		configManager:   configMgr,
	}, nil
}

// ParseRequest contains the input for parsing
type ParseRequest struct {
	Data            []byte
	PartnerID       string
	ValidateContent bool
	GenerateAck     bool
	AckType         core.AcknowledgmentType
	Context         map[string]any
}

// ParseResponse contains the parsing results
type ParseResponse struct {
	Document         *x12.Document
	Profile          *profiles.PartnerProfile
	ValidationIssues []validation.Issue
	IsValid          bool
	Acknowledgment   *core.GenerateResponse
	Statistics       ParseStatistics
}

// ParseStatistics contains parsing statistics
type ParseStatistics struct {
	SegmentCount      int
	TransactionCount  int
	ParseTimeMs       int64
	ValidationTimeMs  int64
	ProfileLoadTimeMs int64
	TotalTimeMs       int64
}

// Parse performs integrated parsing with profile and validation
func (p *IntegratedParser) Parse(ctx context.Context, req ParseRequest) (*ParseResponse, error) {
	startTime := now()
	resp := &ParseResponse{
		ValidationIssues: []validation.Issue{},
		IsValid:          true,
	}

	profile := new(profiles.PartnerProfile)
	if req.PartnerID != "" {
		profileStart := now()
		var err error
		profile, err = p.profileManager.GetProfile(req.PartnerID)
		if err != nil {
			// Profile not found is not fatal - use defaults
			profile = nil
		} else {
			resp.Profile = profile
		}

		resp.Statistics.ProfileLoadTimeMs = elapsed(profileStart)
	}

	// TODO(Wolfred): Determine delimiters (not directly used since parser handles internally)
	if profile != nil {
		// Profile delimiters will be used by ProfileParser
	} else {
		// Parser will detect delimiters automatically
	}

	parseStart := now()
	doc := new(x12.Document)
	var err error

	if profile != nil {
		doc, err = p.profileParser.ParseWithProfile(req.Data, req.PartnerID)
		if err != nil {
			return nil, fmt.Errorf("profile parse failed: %w", err)
		}
		resp.Document = doc
	} else {
		parser := x12.NewParser()
		doc, err = parser.Parse(bytes.NewReader(req.Data))
		if err != nil {
			return nil, fmt.Errorf("parse failed: %w", err)
		}

		metadata := extractMetadata(doc.Segments)
		doc.Metadata = metadata
		resp.Document = doc
	}
	resp.Statistics.ParseTimeMs = elapsed(parseStart)
	resp.Statistics.SegmentCount = len(resp.Document.Segments)

	if req.ValidateContent {
		validationStart := now()

		valOptions := ValidationOptions{
			DetailLevel: "standard",
		}

		if profile != nil {
			valOptions.ValidationConfig = convertProfileValidationConfig(profile.ValidationConfig)
		}

		valRequest := &ValidateRequest{
			EDIContent:      string(req.Data),
			TransactionType: resp.Document.Metadata.TransactionType,
			Version:         resp.Document.Metadata.Version,
			CustomerID:      req.PartnerID,
			Options:         valOptions,
			ParsedDocument:  resp.Document, // Pass the already-parsed document
		}

		valResult, err := p.validationSvc.Validate(ctx, valRequest)
		if err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}

		resp.ValidationIssues = valResult.Issues
		resp.IsValid = valResult.Valid
		resp.Statistics.ValidationTimeMs = elapsed(validationStart)
	}

	if req.GenerateAck {
		ackType := req.AckType
		if ackType == "" {
			if resp.Document.Metadata.Version >= "005010" {
				ackType = core.Ack999
			} else {
				ackType = core.Ack997
			}
		}

		ackReq := core.GenerateRequest{
			Type:      ackType,
			PartnerID: req.PartnerID,
			Original:  resp.Document,
			Issues:    resp.ValidationIssues,
			Accepted:  resp.IsValid,
			Context:   req.Context,
		}

		ackResp, err := p.ackHandler.Generate(ackReq)
		if err != nil {
			return nil, fmt.Errorf("acknowledgment generation failed: %w", err)
		}

		resp.Acknowledgment = ackResp
	}

	for _, seg := range resp.Document.Segments {
		if seg.Tag == "ST" {
			resp.Statistics.TransactionCount++
		}
	}

	resp.Statistics.TotalTimeMs = elapsed(startTime)
	return resp, nil
}

// ValidateWithProfile validates a document using partner-specific rules
func (p *IntegratedParser) ValidateWithProfile(
	ctx context.Context,
	doc *x12.Document,
	partnerID string,
) (*ValidationResult, error) {
	profile, err := p.profileManager.GetProfile(partnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}

	var txConfig *config.TransactionConfig
	if doc.Metadata.TransactionType != "" && doc.Metadata.Version != "" {
		txConfig, _ = p.configManager.GetConfig(doc.Metadata.TransactionType, doc.Metadata.Version)

		if txConfig != nil && partnerID != "" {
			customerConfig, _ := p.configManager.GetCustomerConfig(
				doc.Metadata.TransactionType,
				doc.Metadata.Version,
				partnerID,
			)
			if customerConfig != nil && customerConfig.Active {
				if txConfig.CustomerOverrides == nil {
					txConfig.CustomerOverrides = make(map[string]config.CustomerConfig)
				}
				txConfig.CustomerOverrides[partnerID] = *customerConfig
			}
		}
	}

	valOptions := ValidationOptions{
		DetailLevel: "standard",
	}

	if profile != nil {
		valOptions.ValidationConfig = convertProfileValidationConfig(profile.ValidationConfig)
	}

	// Perform validation
	req := &ValidateRequest{
		EDIContent:      "", // Not needed since we pass ParsedDocument
		TransactionType: doc.Metadata.TransactionType,
		Version:         doc.Metadata.Version,
		CustomerID:      partnerID,
		Options:         valOptions,
		ParsedDocument:  doc, // Pass the already-parsed document
	}

	return p.validationSvc.Validate(ctx, req)
}

// BuildWithProfile builds EDI from business objects using partner profile
func (p *IntegratedParser) BuildWithProfile(
	ctx context.Context,
	data any,
	partnerID string,
	transactionType string,
) (string, error) {
	profile, err := p.profileManager.GetProfile(partnerID)
	if err != nil {
		return "", fmt.Errorf("failed to load profile: %w", err)
	}

	version := "004010" // Default
	for _, tx := range profile.SupportedTransactions {
		if tx.TransactionType == transactionType && len(tx.Versions) > 0 {
			version = tx.Versions[0] // Use first supported version
			break
		}
	}

	txConfig, err := p.configManager.GetConfig(transactionType, version)
	if err != nil {
		// Create a default config if none exists
		txConfig = config.Example204Config()
		p.configManager.SaveConfig(txConfig)
	}

	customerConfig := new(config.CustomerConfig)
	if partnerID != "" {
		customerConfig, _ = p.configManager.GetCustomerConfig(
			transactionType,
			version,
			partnerID,
		)
	}

	builder := config.NewConfigurableBuilder(
		txConfig,
		customerConfig,
		p.segmentRegistry,
		profile.GetDelimiters(),
	)

	txContent, err := builder.BuildFromObject(ctx, data)
	if err != nil {
		return "", fmt.Errorf("failed to build transaction: %w", err)
	}

	return p.wrapWithEnvelopes(txContent, profile, partnerID, transactionType, version)
}

// wrapWithEnvelopes wraps transaction content with ISA/GS/GE/IEA envelopes
func (p *IntegratedParser) wrapWithEnvelopes(
	txContent string,
	profile *profiles.PartnerProfile,
	partnerID string,
	txType, version string,
) (string, error) {
	delims := profile.GetDelimiters()
	var result strings.Builder

	interchangeControl := fmt.Sprintf("%09d", 1)
	groupControl := fmt.Sprintf("%d", 1)

	senderID := "SENDER         "   // 15 chars padded
	receiverID := "RECEIVER       " // 15 chars padded
	if partnerID != "" {
		senderID = fmt.Sprintf("%-15s", partnerID)
		if len(senderID) > 15 {
			senderID = senderID[:15]
		}
	}

	now := time.Now()
	dateStr := now.Format("060102")
	timeStr := now.Format("1504")

	isaFields := []string{
		"ISA",
		"00",                     // Auth qualifier
		"          ",             // Auth info (10 spaces)
		"00",                     // Security qualifier
		"          ",             // Security info (10 spaces)
		"ZZ",                     // Sender qualifier
		senderID,                 // Sender ID (15 chars)
		"ZZ",                     // Receiver qualifier
		receiverID,               // Receiver ID (15 chars)
		dateStr,                  // Date YYMMDD
		timeStr,                  // Time HHMM
		"U",                      // Standard ID
		"00401",                  // Version
		interchangeControl,       // Interchange control
		"0",                      // Ack requested
		"P",                      // Test/Prod
		string(delims.Component), // Component separator
	}

	result.WriteString(strings.Join(isaFields, string(delims.Element)))
	result.WriteByte(delims.Segment)

	gs := fmt.Sprintf("GS%sSM%s%s%s%s%s%s%s%s%s%s%sX%s%s",
		string(delims.Element),
		string(delims.Element),
		senderID[:min(len(senderID), 15)],
		string(delims.Element),
		receiverID[:min(len(receiverID), 15)],
		string(delims.Element),
		now.Format("20060102"),
		string(delims.Element),
		timeStr,
		string(delims.Element),
		groupControl,
		string(delims.Element),
		string(delims.Element),
		version,
	)
	result.WriteString(gs)
	result.WriteByte(delims.Segment)

	result.WriteString(txContent)

	segmentCount := strings.Count(txContent, string(delims.Segment))

	ge := fmt.Sprintf("GE%s%d%s%s",
		string(delims.Element),
		segmentCount,
		string(delims.Element),
		groupControl,
	)
	result.WriteString(ge)
	result.WriteByte(delims.Segment)

	iea := fmt.Sprintf("IEA%s1%s%s",
		string(delims.Element),
		string(delims.Element),
		interchangeControl,
	)
	result.WriteString(iea)
	result.WriteByte(delims.Segment)

	return result.String(), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// convertProfileValidationConfig converts profile validation config to segment validation config
func convertProfileValidationConfig(
	profileConfig profiles.ValidationConfig,
) *segments.ValidationConfig {
	var level segments.ValidationLevel
	switch profileConfig.Strictness {
	case "strict":
		level = segments.ValidationLevelStrict
	case "lenient":
		level = segments.ValidationLevelLenient
	case "none":
		level = segments.ValidationLevelNone
	default:
		level = segments.ValidationLevelStandard
	}

	config := &segments.ValidationConfig{
		Level: level,
		Elements: segments.ElementValidationConfig{
			EnforceMandatory:     profileConfig.EnforceRequiredElements,
			AllowExtraElements:   profileConfig.AllowUnknownSegments,
			SkipLengthValidation: !profileConfig.EnforceElementLengths,
			SkipFormatValidation: !profileConfig.EnforceElementFormats,
		},
		Codes: segments.CodeValidationConfig{
			InvalidCodeHandling: segments.CodeHandlingWarning,
			CaseSensitive:       false,
			AllowPartialMatches: false,
			AllowCustomCodes:    !profileConfig.EnforceElementFormats,
			MinLengthToValidate: 1,
		},
		PartnerOverrides: make(map[string]segments.ValidationOverride),
	}

	switch profileConfig.Strictness {
	case "strict":
		config.Codes.InvalidCodeHandling = segments.CodeHandlingError
		config.Codes.AllowCustomCodes = false
	case "lenient":
		config.Codes.InvalidCodeHandling = segments.CodeHandlingIgnore
		config.Codes.AllowCustomCodes = true
	case "none":
		config.Codes.InvalidCodeHandling = segments.CodeHandlingIgnore
		config.Codes.AllowCustomCodes = true
	default:
		config.Codes.InvalidCodeHandling = segments.CodeHandlingWarning
		config.Codes.AllowCustomCodes = true
	}

	return config
}

// LoadProfile loads a partner profile
func (p *IntegratedParser) LoadProfile(filename string) (*profiles.PartnerProfile, string, error) {
	return p.profileManager.LoadProfile(filename)
}

// SaveProfile saves a partner profile
func (p *IntegratedParser) SaveProfile(partnerID string, profile *profiles.PartnerProfile) error {
	return p.profileManager.SaveProfile(partnerID, profile)
}

// GetProfile retrieves a loaded profile
func (p *IntegratedParser) GetProfile(partnerID string) (*profiles.PartnerProfile, error) {
	return p.profileManager.GetProfile(partnerID)
}

func extractMetadata(segments []x12.Segment) x12.DocumentMetadata {
	metadata := x12.DocumentMetadata{}

	for _, seg := range segments {
		if seg.Tag == "ISA" && len(seg.Elements) >= 13 {
			if len(seg.Elements[5]) > 0 {
				metadata.SenderID = seg.Elements[5][0]
			}
			if len(seg.Elements[7]) > 0 {
				metadata.ReceiverID = seg.Elements[7][0]
			}
			if len(seg.Elements[12]) > 0 {
				metadata.ISAControlNumber = seg.Elements[12][0]
			}
		} else if seg.Tag == "GS" && len(seg.Elements) >= 8 {
			if len(seg.Elements[5]) > 0 {
				metadata.GSControlNumber = seg.Elements[5][0]
			}
			if len(seg.Elements[7]) > 0 {
				metadata.Version = seg.Elements[7][0]
			}
		} else if seg.Tag == "ST" && len(seg.Elements) >= 2 {
			if len(seg.Elements[0]) > 0 {
				metadata.TransactionType = seg.Elements[0][0]
			}
			if len(seg.Elements[1]) > 0 {
				metadata.STControlNumber = seg.Elements[1][0]
			}
		}
	}

	return metadata
}

func now() int64 {
	return time.Now().UnixMilli()
}

func elapsed(start int64) int64 {
	return time.Now().UnixMilli() - start
}
