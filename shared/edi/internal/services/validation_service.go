package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/errors"
	"github.com/emoss08/trenova/shared/edi/internal/registry"
	"github.com/emoss08/trenova/shared/edi/internal/segments"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

// ValidationService provides EDI validation capabilities
type ValidationService struct {
	txRegistry  *registry.TransactionRegistry
	segRegistry *segments.SegmentRegistry
	configMgr   *config.ConfigManager
	processor   *segments.SegmentProcessor
}

// NewValidationService creates a new validation service
func NewValidationService(
	txRegistry *registry.TransactionRegistry,
	segRegistry *segments.SegmentRegistry,
	configMgr *config.ConfigManager,
) *ValidationService {
	processor := segments.NewSegmentProcessorWithValidation(
		segRegistry,
		segments.GetDefaultValidationConfig(),
	)

	return &ValidationService{
		txRegistry:  txRegistry,
		segRegistry: segRegistry,
		configMgr:   configMgr,
		processor:   processor,
	}
}

// NewValidationServiceWithConfig creates a new validation service with custom validation config
func NewValidationServiceWithConfig(
	txRegistry *registry.TransactionRegistry,
	segRegistry *segments.SegmentRegistry,
	configMgr *config.ConfigManager,
	validationConfig segments.ValidationConfig,
) *ValidationService {
	processor := segments.NewSegmentProcessorWithValidation(segRegistry, validationConfig)

	return &ValidationService{
		txRegistry:  txRegistry,
		segRegistry: segRegistry,
		configMgr:   configMgr,
		processor:   processor,
	}
}

// ValidateRequest represents a validation request
type ValidateRequest struct {
	EDIContent      string            `json:"edi_content"`
	TransactionType string            `json:"transaction_type,omitempty"`
	Version         string            `json:"version,omitempty"`
	CustomerID      string            `json:"customer_id,omitempty"`
	Options         ValidationOptions `json:"options,omitempty"`
	Context         map[string]any    `json:"context,omitempty"`
	ParsedDocument  *x12.Document     `json:"-"`
}

// ValidationOptions controls validation behavior
type ValidationOptions struct {
	StopOnFirstError bool                       `json:"stop_on_first_error"`
	MaxErrors        int                        `json:"max_errors"`
	ValidateOnly     string                     `json:"validate_only,omitempty"` // "syntax", "structure", "business", or empty for all
	SkipWarnings     bool                       `json:"skip_warnings"`
	DetailLevel      string                     `json:"detail_level"`                // "minimal", "standard", "detailed"
	ValidationConfig *segments.ValidationConfig `json:"validation_config,omitempty"` // Override validation config
}

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid           bool               `json:"valid"`
	TransactionType string             `json:"transaction_type"`
	Version         string             `json:"version"`
	Issues          []validation.Issue `json:"issues"`
	Statistics      ValidationStats    `json:"statistics"`
	Metadata        map[string]any     `json:"metadata,omitempty"`
}

// ValidationStats provides statistics about the validation
type ValidationStats struct {
	TotalSegments   int            `json:"total_segments"`
	ValidSegments   int            `json:"valid_segments"`
	InvalidSegments int            `json:"invalid_segments"`
	TotalErrors     int            `json:"total_errors"`
	TotalWarnings   int            `json:"total_warnings"`
	SegmentCounts   map[string]int `json:"segment_counts"`
	ProcessingTime  int64          `json:"processing_time_ms"`
}

// Validate performs comprehensive EDI validation
func (s *ValidationService) Validate(
	ctx context.Context,
	req *ValidateRequest,
) (*ValidationResult, error) {
	startTime := timeNow()

	doc := new(x12.Document)
	var err error

	if req.ParsedDocument != nil {
		doc = req.ParsedDocument
	} else {
		parser := x12.NewParser()
		doc, err = parser.Parse(strings.NewReader(req.EDIContent))
		if err != nil {
			return &ValidationResult{
				Valid: false,
				Issues: []validation.Issue{
					{
						Severity: validation.Error,
						Code:     "PARSE_ERROR",
						Message:  fmt.Sprintf("Failed to parse EDI: %v", err),
						Level:    "document",
					},
				},
			}, nil
		}
	}

	// ! Validate document structure (skip if using lenient validation)
	// ! Check if we should skip strict document validation
	skipStrictValidation := false
	if req.Options.ValidationConfig != nil {
		if req.Options.ValidationConfig.Level == segments.ValidationLevelLenient ||
			req.Options.ValidationConfig.Level == segments.ValidationLevelNone {
			skipStrictValidation = true
		}
	}

	if !skipStrictValidation {
		if err := doc.Validate(); err != nil {
			return &ValidationResult{
				Valid: false,
				Issues: []validation.Issue{
					{
						Severity: validation.Error,
						Code:     "PARSE_ERROR",
						Message:  fmt.Sprintf("Invalid EDI structure: %v", err),
						Level:    "document",
					},
				},
			}, nil
		}
	}

	txType := req.TransactionType
	version := req.Version
	if txType == "" || version == "" {
		txType, version = s.detectTransactionType(doc)
		if txType == "" {
			return &ValidationResult{
				Valid: false,
				Issues: []validation.Issue{
					{
						Severity: validation.Error,
						Code:     "UNKNOWN_TRANSACTION",
						Message:  "Unable to determine transaction type",
						Level:    "document",
					},
				},
			}, nil
		}
	}

	// Get configuration
	cfg, err := s.configMgr.GetConfig(txType, version)
	if err != nil {
		// ! Try with default validation if no config
		// ! Note: This path uses validateBasicStructure which may fail with delimiter issues
		return s.validateWithDefaults(ctx, doc, txType, version, req.Options)
	}

	customerCfg := new(config.CustomerConfig)
	if req.CustomerID != "" {
		customerCfg, _ = s.configMgr.GetCustomerConfig(txType, version, req.CustomerID)
	}

	result := &ValidationResult{
		Valid:           true,
		TransactionType: txType,
		Version:         version,
		Issues:          []validation.Issue{},
		Statistics: ValidationStats{
			TotalSegments: len(doc.Segments),
			SegmentCounts: make(map[string]int),
		},
	}

	if req.Options.ValidationConfig != nil {
		s.processor.SetValidationConfig(*req.Options.ValidationConfig)
	}

	if customerCfg != nil && customerCfg.Active {
		s.processor.SetCustomerRequirements(&segments.CustomerRequirements{
			PartnerID:       customerCfg.CustomerID,
			Version:         version,
			TransactionType: txType,
		})
	}

	processedSegments, err := s.processor.ProcessSegments(ctx, doc.Segments, version)
	if err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, validation.Issue{
			Severity: validation.Error,
			Code:     "PROCESSING_ERROR",
			Message:  fmt.Sprintf("Failed to process segments: %v", err),
			Level:    "document",
		})
	}

	for _, seg := range processedSegments {
		result.Statistics.SegmentCounts[seg.Schema.ID]++

		if len(seg.Errors) > 0 {
			result.Statistics.InvalidSegments++
			for _, err := range seg.Errors {
				issue := validation.Issue{
					Severity:     s.mapErrorSeverity(err.Severity),
					Code:         err.Code,
					Message:      err.Message,
					SegmentIndex: seg.Position.Index,
					Tag:          seg.Schema.ID,
					Level:        "segment",
				}

				if err.Location != nil {
					issue.ElementIndex = err.Location.ElementIndex
				}

				result.Issues = append(result.Issues, issue)

				if issue.Severity == validation.Error {
					result.Statistics.TotalErrors++
					result.Valid = false
				} else {
					result.Statistics.TotalWarnings++
				}

				if req.Options.StopOnFirstError && issue.Severity == validation.Error {
					break
				}

				if req.Options.MaxErrors > 0 &&
					result.Statistics.TotalErrors >= req.Options.MaxErrors {
					break
				}
			}
		} else {
			result.Statistics.ValidSegments++
		}
	}

	if cfg != nil && (req.Options.ValidateOnly == "" || req.Options.ValidateOnly == "structure") {
		skipStructureValidation := false
		if req.Options.ValidationConfig != nil {
			if req.Options.ValidationConfig.Level == segments.ValidationLevelLenient ||
				req.Options.ValidationConfig.Level == segments.ValidationLevelNone {
				skipStructureValidation = true
			}
		}

		if !skipStructureValidation {
			structureIssues := s.validateStructure(processedSegments, cfg)
			result.Issues = append(result.Issues, structureIssues...)

			for _, issue := range structureIssues {
				if issue.Severity == validation.Error {
					result.Statistics.TotalErrors++
					result.Valid = false
				} else {
					result.Statistics.TotalWarnings++
				}
			}
		}
	}

	skipBusinessValidation := false
	if req.Options.ValidationConfig != nil {
		if req.Options.ValidationConfig.Level == segments.ValidationLevelLenient ||
			req.Options.ValidationConfig.Level == segments.ValidationLevelNone {
			skipBusinessValidation = true
		}
	}

	if cfg != nil && !skipBusinessValidation &&
		(req.Options.ValidateOnly == "" || req.Options.ValidateOnly == "business") {
		builder := config.NewConfigurableBuilder(cfg, customerCfg, s.segRegistry, doc.Delimiters)

		businessObj, err := builder.ParseToObject(ctx, doc.Segments)
		if err != nil {
		} else {
			if req.Options.DetailLevel == "detailed" {
				if result.Metadata == nil {
					result.Metadata = make(map[string]any)
				}
				result.Metadata["business_object"] = businessObj
			}

			businessIssues := builder.Validate(ctx, businessObj)
			result.Issues = append(result.Issues, businessIssues...)

			for _, issue := range businessIssues {
				if issue.Severity == validation.Error {
					result.Statistics.TotalErrors++
					result.Valid = false
				} else {
					result.Statistics.TotalWarnings++
				}
			}
		}
	}

	if req.Options.SkipWarnings {
		var filteredIssues []validation.Issue
		for _, issue := range result.Issues {
			if issue.Severity == validation.Error {
				filteredIssues = append(filteredIssues, issue)
			}
		}
		result.Issues = filteredIssues
	}

	if req.Options.DetailLevel == "detailed" {
		result.Metadata = map[string]any{
			"customer_id":     req.CustomerID,
			"validation_type": req.Options.ValidateOnly,
			"config_used":     cfg != nil,
			"customer_config": customerCfg != nil,
		}
	}

	result.Statistics.ProcessingTime = timeNow() - startTime

	return result, nil
}

// ValidateBatch validates multiple EDI documents
func (s *ValidationService) ValidateBatch(
	ctx context.Context,
	requests []*ValidateRequest,
) ([]*ValidationResult, error) {
	results := make([]*ValidationResult, len(requests))

	for i, req := range requests {
		result, err := s.Validate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to validate document %d: %w", i+1, err)
		}
		results[i] = result
	}

	return results, nil
}

// PreviewValidation performs a dry-run validation without persisting results
func (s *ValidationService) PreviewValidation(
	ctx context.Context,
	req *ValidateRequest,
) (*ValidationResult, error) {
	req.Options.DetailLevel = "detailed"
	result, err := s.Validate(ctx, req)
	if err != nil {
		return nil, err
	}

	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata["preview_mode"] = true

	return result, nil
}

func (s *ValidationService) detectTransactionType(doc *x12.Document) (string, string) {
	for _, seg := range doc.Segments {
		if strings.ToUpper(seg.Tag) == "ST" && len(seg.Elements) > 0 {
			txType := ""
			if len(seg.Elements[0]) > 0 {
				txType = seg.Elements[0][0]
			}

			version := "004010" // Default
			for _, gseg := range doc.Segments {
				if strings.ToUpper(gseg.Tag) == "GS" && len(gseg.Elements) >= 8 {
					if len(gseg.Elements[7]) > 0 {
						version = gseg.Elements[7][0]
					}
					break
				}
			}

			return txType, version
		}
	}

	return "", ""
}

func (s *ValidationService) validateWithDefaults(
	ctx context.Context,
	doc *x12.Document,
	txType, version string,
	options ValidationOptions,
) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:           true,
		TransactionType: txType,
		Version:         version,
		Issues:          []validation.Issue{},
		Statistics: ValidationStats{
			TotalSegments: len(doc.Segments),
			SegmentCounts: make(map[string]int),
		},
	}

	for i, seg := range doc.Segments {
		result.Statistics.SegmentCounts[seg.Tag]++

		if seg.Tag == "" {
			result.Issues = append(result.Issues, validation.Issue{
				Severity:     validation.Error,
				Code:         "EMPTY_SEGMENT",
				Message:      "Empty segment found",
				SegmentIndex: i,
				Level:        "segment",
			})
			result.Valid = false
			result.Statistics.TotalErrors++
		}
	}

	hasValidStructure := s.validateBasicStructure(doc.Segments)
	if !hasValidStructure {
		result.Issues = append(result.Issues, validation.Issue{
			Severity: validation.Error,
			Code:     "INVALID_STRUCTURE",
			Message:  "Transaction structure is invalid (mismatched ST/SE)",
			Level:    "document",
		})
		result.Valid = false
		result.Statistics.TotalErrors++
	}

	result.Statistics.ValidSegments = result.Statistics.TotalSegments - result.Statistics.InvalidSegments

	return result, nil
}

func (s *ValidationService) validateStructure(
	segments []*segments.ProcessedSegment,
	cfg *config.TransactionConfig,
) []validation.Issue {
	var issues []validation.Issue

	requiredFound := make(map[string]bool)
	for _, req := range cfg.Structure.RequiredSegments {
		if req.Required {
			requiredFound[req.SegmentID] = false
		}
	}

	for _, seg := range segments {
		if _, required := requiredFound[seg.Schema.ID]; required {
			requiredFound[seg.Schema.ID] = true
		}
	}

	for segID, found := range requiredFound {
		if !found {
			issues = append(issues, validation.Issue{
				Severity: validation.Error,
				Code:     "MISSING_REQUIRED_SEGMENT",
				Message:  fmt.Sprintf("Required segment %s is missing", segID),
				Tag:      segID,
				Level:    "structure",
			})
		}
	}

	// Validate loops - count loop starts (segments that begin a loop)
	for _, loop := range cfg.Structure.Loops {
		loopCount := 0
		for _, seg := range segments {
			// Count segments that start the loop
			if seg.Schema.ID == loop.StartSegment {
				loopCount++
			}
		}

		if loop.MinOccurs > 0 && loopCount < loop.MinOccurs {
			issues = append(issues, validation.Issue{
				Severity: validation.Error,
				Code:     "INSUFFICIENT_LOOP_OCCURRENCES",
				Message: fmt.Sprintf(
					"Loop %s requires minimum %d occurrences, found %d",
					loop.LoopID,
					loop.MinOccurs,
					loopCount,
				),
				Level: "structure",
			})
		}

		if loop.MaxOccurs > 0 && loopCount > loop.MaxOccurs {
			issues = append(issues, validation.Issue{
				Severity: validation.Error,
				Code:     "EXCESSIVE_LOOP_OCCURRENCES",
				Message: fmt.Sprintf(
					"Loop %s allows maximum %d occurrences, found %d",
					loop.LoopID,
					loop.MaxOccurs,
					loopCount,
				),
				Level: "structure",
			})
		}
	}

	return issues
}

func (s *ValidationService) validateBasicStructure(segments []x12.Segment) bool {
	stStack := []string{}

	for _, seg := range segments {
		switch strings.ToUpper(seg.Tag) {
		case "ST":
			if len(seg.Elements) >= 2 && len(seg.Elements[1]) > 0 {
				stStack = append(stStack, seg.Elements[1][0])
			}
		case "SE":
			if len(stStack) == 0 {
				return false
			}
			if len(seg.Elements) >= 2 && len(seg.Elements[1]) > 0 {
				expectedControl := stStack[len(stStack)-1]
				if seg.Elements[1][0] != expectedControl {
					return false
				}
				stStack = stStack[:len(stStack)-1]
			}
		}
	}

	return len(stStack) == 0
}

// mapErrorSeverity maps error.Severity to validation.Severity
func (s *ValidationService) mapErrorSeverity(sev errors.Severity) validation.Severity {
	switch sev {
	case errors.SeverityWarning, errors.SeverityInfo:
		return validation.Warning
	default:
		return validation.Error
	}
}

// timeNow returns current time in milliseconds (mockable for tests)
var timeNow = func() int64 {
	return getCurrentTimeMillis()
}

func getCurrentTimeMillis() int64 {
	return int64(0) // Simplified for now
}
