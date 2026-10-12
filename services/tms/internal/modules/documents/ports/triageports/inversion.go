package triageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type InboundCorrection struct {
	TenantInfo              pagination.TenantInfo
	MessageID               pulid.ID
	CapturedByID            pulid.ID
	SenderDomain            string
	PredictedClassification inboundmessage.Classification
	ConfirmedClassification inboundmessage.Classification
	PredictedConfidence     float64
	Facts                   []inboundmessage.Fact
	CapturedAt              int64
}

type InboundCorrectionRecorder interface {
	RecordInboundCorrection(ctx context.Context, correction *InboundCorrection) error
}

type InboundModelAvailability interface {
	ReadsInboundMail(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error)
}

type TenderLoad struct {
	Index            int
	OriginCity       string
	OriginState      string
	DestinationCity  string
	DestinationState string
	PickupFrom       int64
	PickupTo         int64
}

type FindOverlappingLoadsRequest struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
	Loads      []TenderLoad
}

type OverlappingLoad struct {
	LoadIndex  int
	ShipmentID pulid.ID
}

type TenderOverlapFinder interface {
	FindOverlappingLoads(
		ctx context.Context,
		req *FindOverlappingLoadsRequest,
	) ([]OverlappingLoad, error)
}

type TriageConflictChecker interface {
	CheckConflicts(
		ctx context.Context,
		message *inboundmessage.InboundMessage,
	) ([]inboundmessage.Conflict, error)
}
