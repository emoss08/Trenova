package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListDedicatedLaneSuggestionRequest struct {
	Filter           *pagination.QueryOptions        `json:"filter"           form:"filter"`
	Status           *dedicatedlane.SuggestionStatus `json:"status"           form:"status"`
	CustomerID       *pulid.ID                       `json:"customerId"       form:"customerId"`
	IncludeExpired   bool                            `json:"includeExpired"   form:"includeExpired"`
	IncludeProcessed bool                            `json:"includeProcessed" form:"includeProcessed"`
}

type GetDedicatedLaneSuggestionByIDRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type UpdateSuggestionStatusRequest struct {
	SuggestionID  pulid.ID                       `json:"suggestionId"  form:"suggestionId"`
	Status        dedicatedlane.SuggestionStatus `json:"status"        form:"status"`
	ProcessedByID *pulid.ID                      `json:"processedById" form:"processedById"`
	ProcessedAt   *int64                         `json:"processedAt"   form:"processedAt"`
	RejectReason  *string                        `json:"rejectReason"  form:"rejectReason"`
}

type DeleteDedicatedLaneSuggestionRequest struct {
	ID     pulid.ID `json:"id"     form:"id"`
	OrgID  pulid.ID `json:"orgId"  form:"orgId"`
	BuID   pulid.ID `json:"buId"   form:"buId"`
	UserID pulid.ID `json:"userId" form:"userId"`
}

type SuggestionAcceptRequest struct {
	SuggestionID      pulid.ID  `json:"suggestionId"`
	OrgID             pulid.ID  `json:"orgId"`
	BuID              pulid.ID  `json:"buId"`
	ProcessedByID     pulid.ID  `json:"processedById"`
	DedicatedLaneName *string   `json:"dedicatedLaneName,omitempty"` // Override suggested name
	PrimaryWorkerID   *pulid.ID `json:"primaryWorkerId"`
	SecondaryWorkerID *pulid.ID `json:"secondaryWorkerId,omitempty"`
	AutoAssign        bool      `json:"autoAssign"`
}

type SuggestionRejectRequest struct {
	SuggestionID  pulid.ID `json:"suggestionId"`
	OrgID         pulid.ID `json:"orgId"`
	BuID          pulid.ID `json:"buId"`
	ProcessedByID pulid.ID `json:"processedById"`
	RejectReason  string   `json:"rejectReason,omitempty"`
}

type DedicatedLaneSuggestionRepository interface {
	List(
		ctx context.Context,
		req *ListDedicatedLaneSuggestionRequest,
	) (*pagination.ListResult[*dedicatedlane.Suggestion], error)
	GetByID(
		ctx context.Context,
		req *GetDedicatedLaneSuggestionByIDRequest,
	) (*dedicatedlane.Suggestion, error)
	Create(
		ctx context.Context,
		suggestion *dedicatedlane.Suggestion,
	) (*dedicatedlane.Suggestion, error)
	Update(
		ctx context.Context,
		suggestion *dedicatedlane.Suggestion,
	) (*dedicatedlane.Suggestion, error)
	UpdateStatus(
		ctx context.Context,
		req *UpdateSuggestionStatusRequest,
	) (*dedicatedlane.Suggestion, error)
	Delete(
		ctx context.Context,
		req *DeleteDedicatedLaneSuggestionRequest,
	) error
	ExpireOldSuggestions(
		ctx context.Context,
		orgID pulid.ID,
		buID pulid.ID,
	) (int64, error)
	CheckForDuplicatePattern(
		ctx context.Context,
		req *FindDedicatedLaneByShipmentRequest,
	) (*dedicatedlane.Suggestion, error)
}
