package dedicatedlane

import (
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/shopspring/decimal"
)

// ConfigService provides configuration management for dedicated lane pattern detection
type ConfigService struct {
	defaultConfig *dedicatedlane.PatternDetectionConfig
}

// NewConfigService creates a new configuration service with default settings
func NewConfigService() *ConfigService {
	return &ConfigService{
		defaultConfig: dedicatedlane.DefaultPatternDetectionConfig(),
	}
}

// GetDefaultConfig returns the default pattern detection configuration
func (cs *ConfigService) GetDefaultConfig() *dedicatedlane.PatternDetectionConfig {
	return cs.defaultConfig
}

// GetConfigForOrganization returns pattern detection config for a specific organization
// This could be extended to allow per-organization customization
func (cs *ConfigService) GetConfigForOrganization(
	orgID string,
) *dedicatedlane.PatternDetectionConfig {
	// For now, return default config
	// In the future, this could query a database for organization-specific settings
	return cs.defaultConfig
}

// CreateCustomConfig creates a custom configuration with overrides
func (cs *ConfigService) CreateCustomConfig(
	overrides map[string]any,
) *dedicatedlane.PatternDetectionConfig {
	config := &dedicatedlane.PatternDetectionConfig{
		MinFrequency:          cs.defaultConfig.MinFrequency,
		AnalysisWindowDays:    cs.defaultConfig.AnalysisWindowDays,
		MinConfidenceScore:    cs.defaultConfig.MinConfidenceScore,
		SuggestionTTLDays:     cs.defaultConfig.SuggestionTTLDays,
		RequireExactMatch:     cs.defaultConfig.RequireExactMatch,
		WeightRecentShipments: cs.defaultConfig.WeightRecentShipments,
	}

	// Apply overrides
	if val, ok := overrides["minFrequency"].(int64); ok {
		config.MinFrequency = val
	}
	if val, ok := overrides["analysisWindowDays"].(int64); ok {
		config.AnalysisWindowDays = val
	}
	if val, ok := overrides["minConfidenceScore"].(float64); ok {
		config.MinConfidenceScore = decimal.NewFromFloat(val)
	}
	if val, ok := overrides["suggestionTTLDays"].(int64); ok {
		config.SuggestionTTLDays = val
	}
	if val, ok := overrides["requireExactMatch"].(bool); ok {
		config.RequireExactMatch = val
	}
	if val, ok := overrides["weightRecentShipments"].(bool); ok {
		config.WeightRecentShipments = val
	}

	return config
}

// ValidateConfig validates a pattern detection configuration
func (cs *ConfigService) ValidateConfig(config *dedicatedlane.PatternDetectionConfig) error {
	if config.MinFrequency < 1 {
		return errors.New("MinFrequency must be at least 1")
	}
	if config.AnalysisWindowDays < 1 {
		return errors.New("AnalysisWindowDays must be at least 1")
	}
	if config.MinConfidenceScore.LessThan(decimal.Zero) ||
		config.MinConfidenceScore.GreaterThan(decimal.NewFromFloat(1.0)) {
		return errors.New("MinConfidenceScore must be between 0 and 1")
	}
	if config.SuggestionTTLDays < 1 {
		return errors.New("SuggestionTTLDays must be at least 1")
	}
	return nil
}

// GetPresetConfigs returns common preset configurations
func (cs *ConfigService) GetPresetConfigs() map[string]*dedicatedlane.PatternDetectionConfig {
	return map[string]*dedicatedlane.PatternDetectionConfig{
		"conservative": {
			MinFrequency:          5,
			AnalysisWindowDays:    120,
			MinConfidenceScore:    decimal.NewFromFloat(0.8),
			SuggestionTTLDays:     45,
			RequireExactMatch:     true,
			WeightRecentShipments: true,
		},
		"standard": cs.defaultConfig,
		"aggressive": {
			MinFrequency:          2,
			AnalysisWindowDays:    60,
			MinConfidenceScore:    decimal.NewFromFloat(0.5),
			SuggestionTTLDays:     14,
			RequireExactMatch:     false,
			WeightRecentShipments: true,
		},
	}
}
