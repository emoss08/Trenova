// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListDedicatedLaneSuggestionRequest struct {
	Filter           *ports.LimitOffsetQueryOptions
	Status           *dedicatedlane.SuggestionStatus `query:"status"`
	CustomerID       *pulid.ID                       `query:"customerId"`
	IncludeExpired   bool                            `query:"includeExpired"`
	IncludeProcessed bool                            `query:"includeProcessed"`
}

type GetDedicatedLaneSuggestionByIDRequest struct {
	ID     pulid.ID
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type UpdateSuggestionStatusRequest struct {
	SuggestionID  pulid.ID
	Status        dedicatedlane.SuggestionStatus
	ProcessedByID *pulid.ID
	ProcessedAt   *int64
	RejectReason  *string
}

type DedicatedLaneSuggestionRepository interface {
	List(
		ctx context.Context,
		req *ListDedicatedLaneSuggestionRequest,
	) (*ports.ListResult[*dedicatedlane.DedicatedLaneSuggestion], error)

	GetByID(
		ctx context.Context,
		req *GetDedicatedLaneSuggestionByIDRequest,
	) (*dedicatedlane.DedicatedLaneSuggestion, error)

	Create(
		ctx context.Context,
		suggestion *dedicatedlane.DedicatedLaneSuggestion,
	) (*dedicatedlane.DedicatedLaneSuggestion, error)

	Update(
		ctx context.Context,
		suggestion *dedicatedlane.DedicatedLaneSuggestion,
	) (*dedicatedlane.DedicatedLaneSuggestion, error)

	UpdateStatus(
		ctx context.Context,
		req *UpdateSuggestionStatusRequest,
	) (*dedicatedlane.DedicatedLaneSuggestion, error)

	Delete(
		ctx context.Context,
		id pulid.ID,
		orgID pulid.ID,
		buID pulid.ID,
	) error

	// ExpireOldSuggestions marks suggestions as expired based on their ExpiresAt timestamp
	ExpireOldSuggestions(
		ctx context.Context,
		orgID pulid.ID,
		buID pulid.ID,
	) (int64, error)

	// CheckForDuplicatePattern checks if a similar suggestion already exists
	CheckForDuplicatePattern(
		ctx context.Context,
		req *FindDedicatedLaneByShipmentRequest,
	) (*dedicatedlane.DedicatedLaneSuggestion, error)
}
