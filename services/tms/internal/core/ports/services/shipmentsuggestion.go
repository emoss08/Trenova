package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipmentsuggestion"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SuggestionKind string

const (
	SuggestionCoverage       = SuggestionKind("Coverage")
	SuggestionTender         = SuggestionKind("Tender")
	SuggestionDelayNotice    = SuggestionKind("DelayNotice")
	SuggestionHoursOfService = SuggestionKind("HoursOfService")
	SuggestionDetention      = SuggestionKind("Detention")
	SuggestionRetender       = SuggestionKind("Retender")
)

type SuggestionTone string

const (
	SuggestionToneDanger  = SuggestionTone("Danger")
	SuggestionToneWarning = SuggestionTone("Warning")
	SuggestionToneAccent  = SuggestionTone("Accent")
	SuggestionToneBrand   = SuggestionTone("Brand")
)

type SuggestionActionType string

const (
	SuggestionActionAssignDriver    = SuggestionActionType("AssignDriver")
	SuggestionActionTenderCarrier   = SuggestionActionType("TenderCarrier")
	SuggestionActionNotifyCustomer  = SuggestionActionType("NotifyCustomer")
	SuggestionActionApproveDetention = SuggestionActionType("ApproveDetention")
	SuggestionActionReview          = SuggestionActionType("Review")
)

type SuggestionAction struct {
	Type                  SuggestionActionType
	Label                 string
	MoveID                pulid.ID
	WorkerID              pulid.ID
	TractorID             pulid.ID
	CarrierID             pulid.ID
	DetentionOccurrenceID pulid.ID
	Message               string
}

type ShipmentSuggestion struct {
	Key         string
	Kind        SuggestionKind
	Tone        SuggestionTone
	ShipmentID  pulid.ID
	ProNumber   string
	Title       string
	Reason      string
	Impact      []string
	Primary     SuggestionAction
	ManualLabel string
	DueAt       *int64
	Deferred    bool
}

type ShipmentSuggestionQueue struct {
	Items            []*ShipmentSuggestion
	HandledThisShift int
	Narrated         bool
}

type SuggestionRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Timezone   string
}

type DecideSuggestionRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Key        string
	Decision   shipmentsuggestion.Decision
}

type UndoSuggestionRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Key        string
}

type ShipmentSuggestionReader interface {
	Suggestions(ctx context.Context, req *SuggestionRequest) (*ShipmentSuggestionQueue, error)
}

type ShipmentSuggestionDecider interface {
	Decide(ctx context.Context, req *DecideSuggestionRequest) error
	Undo(ctx context.Context, req *UndoSuggestionRequest) error
}
