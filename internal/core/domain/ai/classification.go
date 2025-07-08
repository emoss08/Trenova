package ai

import (
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports"
)

// ClassificationRequest represents a request to classify a location
type ClassificationRequest struct {
	TenantOpts  *ports.TenantOptions `json:"tenantOpts"`
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	Address     *string              `json:"address,omitempty"`
}

// ClassificationResponse represents the AI classification result
type ClassificationResponse struct {
	Category              location.Category      `json:"category"`
	CategoryID            string                 `json:"categoryId"`
	FacilityType          *location.FacilityType `json:"facilityType,omitempty"`
	Confidence            float64                `json:"confidence"`
	Reasoning             string                 `json:"reasoning"`
	AlternativeCategories []AlternativeCategory  `json:"alternativeCategories"`
}

// AlternativeCategory represents an alternative classification option
type AlternativeCategory struct {
	Category   location.Category `json:"category"`
	CategoryID string            `json:"categoryId"`
	Confidence float64           `json:"confidence"`
}

// BatchClassificationRequest represents multiple locations to classify
type BatchClassificationRequest struct {
	Locations []ClassificationRequest `json:"locations"`
}

// BatchClassificationResponse represents multiple classification results
type BatchClassificationResponse struct {
	Results []ClassificationResponse `json:"results"`
}
