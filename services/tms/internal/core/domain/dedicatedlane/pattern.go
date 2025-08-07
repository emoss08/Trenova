/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package dedicatedlane

import (
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// PatternDetectionConfig holds configuration for pattern detection
type PatternDetectionConfig struct {
	MinFrequency          int64           `json:"minFrequency"`          // Minimum occurrences to trigger suggestion
	AnalysisWindowDays    int64           `json:"analysisWindowDays"`    // Number of days to look back
	MinConfidenceScore    decimal.Decimal `json:"minConfidenceScore"`    // Minimum confidence score to create suggestion
	SuggestionTTLDays     int64           `json:"suggestionTtlDays"`     // Days before suggestion expires
	RequireExactMatch     bool            `json:"requireExactMatch"`     // Whether to require exact equipment/service matches
	WeightRecentShipments bool            `json:"weightRecentShipments"` // Give more weight to recent shipments
}

// DefaultPatternDetectionConfig returns sensible defaults
func DefaultPatternDetectionConfig() *PatternDetectionConfig {
	return &PatternDetectionConfig{
		MinFrequency:          3,
		AnalysisWindowDays:    90,
		MinConfidenceScore:    decimal.NewFromFloat(0.7),
		SuggestionTTLDays:     30,
		RequireExactMatch:     false,
		WeightRecentShipments: true,
	}
}

// PatternMatch represents a detected shipping pattern
type PatternMatch struct {
	OrganizationID        pulid.ID             `json:"organizationId"`
	BusinessUnitID        pulid.ID             `json:"businessUnitId"`
	CustomerID            pulid.ID             `json:"customerId"`
	OriginLocationID      pulid.ID             `json:"originLocationId"`
	DestinationLocationID pulid.ID             `json:"destinationLocationId"`
	ServiceTypeID         *pulid.ID            `json:"serviceTypeId,omitzero"`
	ShipmentTypeID        *pulid.ID            `json:"shipmentTypeId,omitzero"`
	TrailerTypeID         *pulid.ID            `json:"trailerTypeId,omitzero"`
	TractorTypeID         *pulid.ID            `json:"tractorTypeId,omitzero"`
	FrequencyCount        int64                `json:"frequencyCount"`
	ConfidenceScore       decimal.Decimal      `json:"confidenceScore"`
	AverageFreightCharge  *decimal.NullDecimal `json:"averageFreightCharge,omitzero"`
	TotalFreightValue     *decimal.NullDecimal `json:"totalFreightValue,omitzero"`
	FirstShipmentDate     int64                `json:"firstShipmentDate"`
	LastShipmentDate      int64                `json:"lastShipmentDate"`
	ShipmentIDs           []pulid.ID           `json:"shipmentIds"`
	PatternDetails        map[string]any       `json:"patternDetails"`
}

// PatternAnalysisRequest represents a request to analyze patterns
type PatternAnalysisRequest struct {
	Config          *PatternDetectionConfig `json:"config,omitempty"` // Optional: override default config
	ExcludeExisting bool                    `json:"excludeExisting"`  // Skip patterns that already have dedicated lanes
}

// PatternAnalysisResult represents the result of pattern analysis
type PatternAnalysisResult struct {
	TotalPatternsDetected  int64                     `json:"totalPatternsDetected"`
	PatternsAboveThreshold int64                     `json:"patternsAboveThreshold"`
	ConfigsUsed            []*PatternDetectionConfig `json:"configsUsed"`
	Patterns               []*PatternMatch           `json:"patterns"`
	ProcessingTimeMs       int64                     `json:"processingTimeMs"`
}

// SuggestionAcceptRequest represents a request to accept a suggestion
type SuggestionAcceptRequest struct {
	SuggestionID      pulid.ID  `json:"suggestionId"`
	OrganizationID    pulid.ID  `json:"organizationId"`
	BusinessUnitID    pulid.ID  `json:"businessUnitId"`
	ProcessedByID     pulid.ID  `json:"processedById"`
	DedicatedLaneName *string   `json:"dedicatedLaneName,omitempty"` // Override suggested name
	PrimaryWorkerID   *pulid.ID `json:"primaryWorkerId"`
	SecondaryWorkerID *pulid.ID `json:"secondaryWorkerId,omitempty"`
	AutoAssign        bool      `json:"autoAssign"`
}

// SuggestionRejectRequest represents a request to reject a suggestion
type SuggestionRejectRequest struct {
	SuggestionID   pulid.ID `json:"suggestionId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ProcessedByID  pulid.ID `json:"processedById"`
	RejectReason   string   `json:"rejectReason,omitempty"`
}
