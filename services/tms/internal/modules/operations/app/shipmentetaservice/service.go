package shipmentetaservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const positionLookbackSeconds = int64(24 * 60 * 60)

type BoardMoveLister interface {
	ListBoardMoves(
		ctx context.Context,
		filter *dispatchboardports.DispatchBoardFilter,
	) ([]*dispatchboardports.BoardMove, error)
}

type WorkerLister interface {
	ListWorkersByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerIDs []pulid.ID,
	) ([]*worker.Worker, error)
}

type TelematicsReader interface {
	ListVehiclePositions(
		ctx context.Context,
		req *telematicsports.ListVehiclePositionsRequest,
	) ([]*telematics.VehiclePosition, error)
	ListWorkerHOSStates(
		ctx context.Context,
		req *telematicsports.ListWorkerHOSStatesRequest,
	) ([]*telematics.WorkerHOSState, error)
}

type ServiceFailureLister interface {
	ListUnresolvedByShipmentIDs(
		ctx context.Context,
		req *servicefailureports.ServiceFailuresByShipmentIDsRequest,
	) ([]*servicefailure.ServiceFailure, error)
}

type CheckCallReader interface {
	LatestForMoves(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveIDs []pulid.ID,
	) (map[pulid.ID]*checkcall.CheckCall, error)
}

type ControlReader interface {
	GetByOrgID(
		ctx context.Context,
		req dispatchcontrolports.GetDispatchControlRequest,
	) (*dispatchcontrol.DispatchControl, error)
}

type SourcesParams struct {
	fx.In

	Console         dispatchboardports.DispatchConsoleRepository
	Telematics      telematicsports.TelematicsRepository
	ServiceFailures servicefailureports.ServiceFailureRepository
	StopEtas        stopetaports.Repository
}

// Sources reads what tracking is built from: the stored stop ETAs, each open move's
// assignment, its tractor's latest position, its driver's hours of service and exemptions,
// and the shipment's unresolved service failures. The reader and the refresher share it.
type Sources struct {
	board    BoardMoveLister
	workers  WorkerLister
	telem    TelematicsReader
	failures ServiceFailureLister
	stopEtas stopetaports.Repository
}

func NewSources(
	p SourcesParams, //nolint:gocritic // fx params are passed by value
) *Sources {
	return &Sources{
		board:    p.Console,
		workers:  p.Console,
		telem:    p.Telematics,
		failures: p.ServiceFailures,
		stopEtas: p.StopEtas,
	}
}

type ReaderParams struct {
	fx.In

	Shipments shipmentports.ShipmentTrackingRepository
	Sources   *Sources
}

// Reader answers what tracking and ETAs show for a set of shipments, from what was stored
// at the last refresh and the live sources.
type Reader struct {
	shipments shipmentports.ShipmentTrackingRepository
	sources   *Sources
	now       func() time.Time
}

var (
	_ shipmenttrackingports.ShipmentEtaReader      = (*Reader)(nil)
	_ shipmenttrackingports.ShipmentTrackingReader = (*Reader)(nil)
)

// NewReader is the process's one tracking reader, bound to both reader ports:
// ShipmentEtaReader and ShipmentTrackingReader.
func NewReader(
	p ReaderParams, //nolint:gocritic // fx params are passed by value
) *Reader {
	return &Reader{
		shipments: p.Shipments,
		sources:   p.Sources,
		now:       time.Now,
	}
}

type RefresherParams struct {
	fx.In

	Shipments        shipmentports.ShipmentTrackingRepository
	Sources          *Sources
	StopEtas         stopetaports.Repository
	CheckCalls       checkcallports.CheckCallRepository
	DispatchControls dispatchcontrolports.DispatchControlRepository
	Routes           routeestimateports.RouteEstimator
	Realtime         realtimeports.RealtimeService
	Observers        []stopetaports.Observer `group:"stop_eta_observers"`
	Logger           *zap.Logger
}

// Refresher recomputes stop ETAs, stores them and tells the observers. It is a type apart
// from the reader because the observers it announces to read tracking.
type Refresher struct {
	shipments shipmentports.ShipmentTrackingRepository
	sources   *Sources
	stopEtas  stopetaports.Repository
	calls     CheckCallReader
	controls  ControlReader
	routes    routeestimateports.RouteEstimator
	realtime  realtimeports.RealtimeService
	observers []stopetaports.Observer
	l         *zap.Logger
	now       func() time.Time
}

var _ stopetaports.Refresher = (*Refresher)(nil)

func NewRefresher(
	p RefresherParams, //nolint:gocritic // fx params are passed by value
) *Refresher {
	return &Refresher{
		shipments: p.Shipments,
		sources:   p.Sources,
		stopEtas:  p.StopEtas,
		calls:     p.CheckCalls,
		controls:  p.DispatchControls,
		routes:    p.Routes,
		realtime:  p.Realtime,
		observers: p.Observers,
		l:         p.Logger.Named("shipment-eta-refresher"),
		now:       time.Now,
	}
}
