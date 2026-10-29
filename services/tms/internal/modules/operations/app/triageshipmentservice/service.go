package triageshipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/ports/triageworkports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticeservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/triagelookupports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type trackingReader interface {
	TrackingSnapshots(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		shipmentIDs []pulid.ID,
		timezone string,
	) (map[pulid.ID]*shipmenttracking.Snapshot, error)
}

type summaryReader interface {
	ListSummariesByIDs(
		ctx context.Context,
		req *shipmentports.ListShipmentSummariesRequest,
	) ([]*shipmentports.ShipmentSummary, error)
}

type shipmentCreator interface {
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
}

type holdCreator interface {
	Create(
		ctx context.Context,
		req *shipmentholdports.CreateShipmentHoldRequest,
		actor *permissionports.RequestActor,
	) (*shipmenthold.ShipmentHold, error)
}

type commentCreator interface {
	CreateSystem(
		ctx context.Context,
		req *shipmentcommentports.CreateSystemShipmentCommentRequest,
	) (*shipment.ShipmentComment, error)
}

type occurrenceReader interface {
	GetByShipment(
		ctx context.Context,
		req *detentionports.GetOccurrencesByShipmentRequest,
	) ([]*detention.DetentionOccurrence, error)
}

type detentionDecider interface {
	Waive(
		ctx context.Context,
		p detentionreview.WaiveParams,
	) (*detention.DetentionOccurrence, error)
	WaiveUninvoiced(
		ctx context.Context,
		p *detentionreview.WaiveParams,
	) (*detention.DetentionOccurrence, error)
	Dispute(
		ctx context.Context,
		p detentionreview.DisputeParams,
	) (*detention.DetentionOccurrence, error)
}

type disputeRecorder interface {
	DisputeRecord(
		ctx context.Context,
		p *detentionnoticeservice.DisputeRecordParams,
	) (*detentionnoticeservice.DisputeRecord, error)
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	Tracking    shipmenttrackingports.ShipmentTrackingReader
	Shipments   shipmentports.ShipmentRepository
	Lookup      triagelookupports.TriageLookupRepository
	Creator     shipmentserviceports.ShipmentService
	Holds       shipmentholdports.ShipmentHoldService
	Comments    shipmentcommentports.ShipmentCommentService
	Occurrences detentionports.DetentionOccurrenceRepository
	Review      *detentionreview.Service
	Notices     *detentionnoticeservice.Service
	Credits     detentionports.BilledDetentionCreditor `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	tracking    trackingReader
	summaries   summaryReader
	lookup      triagelookupports.TriageLookupRepository
	creator     shipmentCreator
	holds       holdCreator
	comments    commentCreator
	occurrences occurrenceReader
	detention   detentionDecider
	records     disputeRecorder
	credits     detentionports.BilledDetentionCreditor
	now         func() int64
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:           p.Logger.Named("service.triage-shipment"),
		tracking:    p.Tracking,
		summaries:   p.Shipments,
		lookup:      p.Lookup,
		creator:     p.Creator,
		holds:       p.Holds,
		comments:    p.Comments,
		occurrences: p.Occurrences,
		detention:   p.Review,
		records:     p.Notices,
		credits:     p.Credits,
		now:         timeutils.NowUnix,
	}
}

var (
	_ triageworkports.TriageShipmentPeeker   = (*Service)(nil)
	_ triageworkports.TriageStopTimesReader  = (*Service)(nil)
	_ triageworkports.TriageLoadCreator      = (*Service)(nil)
	_ triageworkports.TriageExceptionLogger  = (*Service)(nil)
	_ triageworkports.TriageDetentionSettler = (*Service)(nil)

	_ triageworkports.TriageDetentionEvidenceSource = (*Service)(nil)
)
