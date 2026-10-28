package shipmentetaservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
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
		filter *shipmentports.DispatchBoardFilter,
	) ([]*shipmentports.BoardMove, error)
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

type ReaderParams struct {
	fx.In

	Shipments       shipmentports.ShipmentTrackingRepository
	Console         shipmentports.DispatchConsoleRepository
	Telematics      telematicsports.TelematicsRepository
	ServiceFailures servicefailureports.ServiceFailureRepository
	StopEtas        stopetaports.Repository
}

type RefresherParams struct {
	fx.In

	Shipments        shipmentports.ShipmentTrackingRepository
	Console          shipmentports.DispatchConsoleRepository
	Telematics       telematicsports.TelematicsRepository
	ServiceFailures  servicefailureports.ServiceFailureRepository
	StopEtas         stopetaports.Repository
	CheckCalls       checkcallports.CheckCallRepository
	DispatchControls dispatchcontrolports.DispatchControlRepository
	Routes           routeestimateports.RouteEstimator
	Realtime         realtimeports.RealtimeService `optional:"true"`
	Observers        []stopetaports.Observer       `group:"stop_eta_observers"`
	Logger           *zap.Logger
}

type Dependencies struct {
	Shipments       shipmentports.ShipmentTrackingRepository
	Board           BoardMoveLister
	Workers         WorkerLister
	Telematics      TelematicsReader
	ServiceFailures ServiceFailureLister
	StopEtas        stopetaports.Repository
	CheckCalls      CheckCallReader
	Controls        ControlReader
	Routes          routeestimateports.RouteEstimator
	Realtime        realtimeports.RealtimeService
	Observers       []stopetaports.Observer
	Logger          *zap.Logger
	Now             func() time.Time
}

type Service struct {
	shipments shipmentports.ShipmentTrackingRepository
	board     BoardMoveLister
	workers   WorkerLister
	telem     TelematicsReader
	failures  ServiceFailureLister
	stopEtas  stopetaports.Repository
	calls     CheckCallReader
	controls  ControlReader
	routes    routeestimateports.RouteEstimator
	realtime  realtimeports.RealtimeService
	observers []stopetaports.Observer
	l         *zap.Logger
	now       func() time.Time
}

var (
	_ shipmenttrackingports.ShipmentEtaReader      = (*Service)(nil)
	_ shipmenttrackingports.ShipmentTrackingReader = (*Service)(nil)
	_ stopetaports.Refresher                       = (*Service)(nil)
)

func New(
	p ReaderParams, //nolint:gocritic // fx params are passed by value
) shipmenttrackingports.ShipmentEtaReader {
	return fromReaderParams(p)
}

func NewTrackingReader(
	p ReaderParams, //nolint:gocritic // fx params are passed by value
) shipmenttrackingports.ShipmentTrackingReader {
	return fromReaderParams(p)
}

func NewRefresher(
	p RefresherParams, //nolint:gocritic // fx params are passed by value
) stopetaports.Refresher {
	return NewWithDependencies(&Dependencies{
		Shipments:       p.Shipments,
		Board:           p.Console,
		Workers:         p.Console,
		Telematics:      p.Telematics,
		ServiceFailures: p.ServiceFailures,
		StopEtas:        p.StopEtas,
		CheckCalls:      p.CheckCalls,
		Controls:        p.DispatchControls,
		Routes:          p.Routes,
		Realtime:        p.Realtime,
		Observers:       p.Observers,
		Logger:          p.Logger,
		Now:             time.Now,
	})
}

func fromReaderParams(p ReaderParams) *Service { //nolint:gocritic // dependency injection
	return NewWithDependencies(&Dependencies{
		Shipments:       p.Shipments,
		Board:           p.Console,
		Workers:         p.Console,
		Telematics:      p.Telematics,
		ServiceFailures: p.ServiceFailures,
		StopEtas:        p.StopEtas,
		Now:             time.Now,
	})
}

func NewWithDependencies(d *Dependencies) *Service {
	logger := d.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}

	return &Service{
		shipments: d.Shipments,
		board:     d.Board,
		workers:   d.Workers,
		telem:     d.Telematics,
		failures:  d.ServiceFailures,
		stopEtas:  d.StopEtas,
		calls:     d.CheckCalls,
		controls:  d.Controls,
		routes:    d.Routes,
		realtime:  d.Realtime,
		observers: d.Observers,
		l:         logger.Named("shipment-eta-service"),
		now:       now,
	}
}
