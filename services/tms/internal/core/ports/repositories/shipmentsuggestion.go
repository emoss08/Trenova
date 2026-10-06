package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListSuggestionDecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Since      int64
}

type DeleteSuggestionDecisionRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Key        string
}

type PruneSuggestionDecisionsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Before     int64
}

type ShipmentSuggestionDecisionRepository interface {
	ListSince(
		ctx context.Context,
		req *ListSuggestionDecisionsRequest,
	) ([]*shipmentsuggestion.DecisionRecord, error)
	Upsert(
		ctx context.Context,
		record *shipmentsuggestion.DecisionRecord,
	) (*shipmentsuggestion.DecisionRecord, error)
	Delete(
		ctx context.Context,
		req *DeleteSuggestionDecisionRequest,
	) (*shipmentsuggestion.DecisionRecord, error)
	Prune(ctx context.Context, req *PruneSuggestionDecisionsRequest) error
}
