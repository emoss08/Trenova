package segments

import "strings"

// ValidationLevel defines how strict validation should be
type ValidationLevel string

const (
	ValidationLevelStrict   ValidationLevel = "strict"   // Enforce all schema rules strictly
	ValidationLevelStandard ValidationLevel = "standard" // Standard validation with some flexibility
	ValidationLevelLenient  ValidationLevel = "lenient"  // Minimal validation, focus on structure
	ValidationLevelNone     ValidationLevel = "none"     // No validation beyond basic structure
)

// ValidationConfig controls how validation is performed
type ValidationConfig struct {
	Level            ValidationLevel               `json:"level"`
	Elements         ElementValidationConfig       `json:"elements"`
	Codes            CodeValidationConfig          `json:"codes"`
	PartnerOverrides map[string]ValidationOverride `json:"partner_overrides,omitempty"`
}

// ElementValidationConfig controls element validation
type ElementValidationConfig struct {
	EnforceMandatory     bool `json:"enforce_mandatory"`
	AllowExtraElements   bool `json:"allow_extra_elements"`
	SkipLengthValidation bool `json:"skip_length_validation"`
	SkipFormatValidation bool `json:"skip_format_validation"`
}

// CodeValidationConfig controls code validation
type CodeValidationConfig struct {
	InvalidCodeHandling CodeHandling `json:"invalid_code_handling"`
	CaseSensitive       bool         `json:"case_sensitive"`
	AllowPartialMatches bool         `json:"allow_partial_matches"`
	AllowCustomCodes    bool         `json:"allow_custom_codes"`
	MinLengthToValidate int          `json:"min_length_to_validate"`
}

// CodeHandling defines how to handle invalid codes
type CodeHandling string

const (
	CodeHandlingError   CodeHandling = "error"   // Treat as validation error
	CodeHandlingWarning CodeHandling = "warning" // Treat as warning
	CodeHandlingIgnore  CodeHandling = "ignore"  // Ignore invalid codes
)

// ValidationOverride allows partner-specific validation overrides
type ValidationOverride struct {
	Level            ValidationLevel                      `json:"level,omitempty"`
	Elements         ElementValidationConfig              `json:"elements,omitempty"`
	Codes            CodeValidationConfig                 `json:"codes,omitempty"`
	SegmentOverrides map[string]SegmentValidationOverride `json:"segment_overrides,omitempty"`
}

// SegmentValidationOverride allows overriding validation for specific segments
type SegmentValidationOverride struct {
	Skip             bool                              `json:"skip,omitempty"`
	ElementOverrides map[int]ElementValidationOverride `json:"element_overrides,omitempty"`
}

// ElementValidationOverride allows overriding validation for specific elements
type ElementValidationOverride struct {
	Skip          bool     `json:"skip,omitempty"`
	Requirement   string   `json:"requirement,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
	Pattern       string   `json:"pattern,omitempty"`
}

// GetDefaultValidationConfig returns a default configuration
func GetDefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		Level: ValidationLevelStandard,
		Elements: ElementValidationConfig{
			EnforceMandatory:     true,
			AllowExtraElements:   true,
			SkipLengthValidation: false,
			SkipFormatValidation: false,
		},
		Codes: CodeValidationConfig{
			InvalidCodeHandling: CodeHandlingWarning,
			CaseSensitive:       false,
			AllowPartialMatches: false,
			AllowCustomCodes:    true,
			MinLengthToValidate: 2,
		},
		PartnerOverrides: make(map[string]ValidationOverride),
	}
}

// GetStrictValidationConfig returns a strict configuration
func GetStrictValidationConfig() ValidationConfig {
	return ValidationConfig{
		Level: ValidationLevelStrict,
		Elements: ElementValidationConfig{
			EnforceMandatory:     true,
			AllowExtraElements:   false,
			SkipLengthValidation: false,
			SkipFormatValidation: false,
		},
		Codes: CodeValidationConfig{
			InvalidCodeHandling: CodeHandlingError,
			CaseSensitive:       true,
			AllowPartialMatches: false,
			AllowCustomCodes:    false,
			MinLengthToValidate: 1,
		},
		PartnerOverrides: make(map[string]ValidationOverride),
	}
}

// GetLenientValidationConfig returns a lenient configuration
func GetLenientValidationConfig() ValidationConfig {
	return ValidationConfig{
		Level: ValidationLevelLenient,
		Elements: ElementValidationConfig{
			EnforceMandatory:     false,
			AllowExtraElements:   true,
			SkipLengthValidation: true,
			SkipFormatValidation: true,
		},
		Codes: CodeValidationConfig{
			InvalidCodeHandling: CodeHandlingIgnore,
			CaseSensitive:       false,
			AllowPartialMatches: true,
			AllowCustomCodes:    true,
			MinLengthToValidate: 0,
		},
		PartnerOverrides: make(map[string]ValidationOverride),
	}
}

// ShouldValidateCode determines if a code should be validated based on configuration
func (c *ValidationConfig) ShouldValidateCode(code string, hasSchemaCodes bool) bool {
	if c.Level == ValidationLevelNone {
		return false
	}

	if c.Level == ValidationLevelLenient && c.Codes.AllowCustomCodes {
		return false
	}

	if !hasSchemaCodes {
		return false
	}

	if len(code) < c.Codes.MinLengthToValidate {
		return false
	}

	return true
}

// IsCodeMatch checks if a value matches a schema code based on configuration
func (c *ValidationConfig) IsCodeMatch(value, schemaCode string) bool {
	if !c.Codes.CaseSensitive {
		value = strings.ToUpper(value)
		schemaCode = strings.ToUpper(schemaCode)
	}

	if c.Codes.AllowPartialMatches {
		return strings.HasPrefix(schemaCode, value) || strings.HasPrefix(value, schemaCode)
	}

	return value == schemaCode
}

// GetValidationSeverity returns the appropriate severity for a validation failure
func (c *ValidationConfig) GetValidationSeverity(validationType string) string {
	switch c.Level {
	case ValidationLevelStrict:
		return "error"
	case ValidationLevelStandard:
		switch validationType {
		case "code":
			if c.Codes.InvalidCodeHandling == CodeHandlingWarning {
				return "warning"
			}
			return "error"
		case "mandatory":
			return "error"
		default:
			return "warning"
		}
	case ValidationLevelLenient:
		return "warning"
	case ValidationLevelNone:
		return "info"
	default:
		return "warning"
	}
}
