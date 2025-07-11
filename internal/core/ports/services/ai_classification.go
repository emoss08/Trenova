package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/ai"
)

// AIClassificationService defines the interface for AI-powered location classification
type AIClassificationService interface {
	// ClassifyLocation classifies a single location using AI
	ClassifyLocation(
		ctx context.Context,
		req *ai.ClassificationRequest,
	) (*ai.ClassificationResponse, error)

	// ClassifyLocationBatch classifies multiple locations in batch
	ClassifyLocationBatch(
		ctx context.Context,
		req *ai.BatchClassificationRequest,
	) (*ai.BatchClassificationResponse, error)
}
