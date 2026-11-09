package driverpositioningservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/driverpositioningports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	eventType      = "dash.positioning"
	timeLayout     = "Mon 3:04 PM"
	outlookHorizon = int64(demandforecast.OutlookSlots) * dispatchmarket.SlotSeconds
)

type driverNotifier interface {
	Notify(ctx context.Context, req *drivernotificationservice.DriverNotification)
}

type forecasts interface {
	Forecaster(
		ctx context.Context,
		req *demandforecastports.ForecasterRequest,
	) (*demandforecast.Forecaster, error)
	Location(ctx context.Context, tenantInfo pagination.TenantInfo) *time.Location
	Economics(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (margin, capCents int64, err error)
}

type starts interface {
	ListDriverStarts(
		ctx context.Context,
		req *demandforecastports.DriverStartsRequest,
	) ([]demandforecastports.DriverStart, error)
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         driverpositioningports.Repository
	Starts       demandforecastports.Repository
	Forecasts    demandforecastports.Service
	AuditService auditports.AuditService
	DriverNotify *drivernotificationservice.Service
	Clock        timeutils.Clock
}

type Service struct {
	l         *zap.Logger
	repo      driverpositioningports.Repository
	starts    starts
	forecasts forecasts
	audit     auditports.AuditService
	notify    driverNotifier
	clock     timeutils.Clock
}

var _ driverpositioningports.Service = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Service{
		l:         p.Logger.Named("service.driver-positioning"),
		repo:      p.Repo,
		starts:    p.Starts,
		forecasts: p.Forecasts,
		audit:     p.AuditService,
		notify:    p.DriverNotify,
		clock:     p.Clock,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req *driverpositioningports.CreateRequest,
) (*driverpositioning.Positioning, error) {
	now := s.clock()
	if err := validateTarget(req, now); err != nil {
		return nil, err
	}

	start, err := s.driverStart(ctx, req.TenantInfo, req.WorkerID, now)
	if err != nil {
		return nil, err
	}
	from := dispatchmarket.Point{Latitude: start.Latitude, Longitude: start.Longitude}
	fromCell, _ := dispatchmarket.CellOf(from)
	if fromCell == req.ToCell {
		return nil, errortypes.NewBusinessError("This driver is already in that area.")
	}

	margin, capCents, err := s.forecasts.Economics(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	forecaster, err := s.forecasts.Forecaster(ctx, &demandforecastports.ForecasterRequest{
		TenantInfo:    req.TenantInfo,
		Now:           now,
		Cells:         []int64{fromCell, req.ToCell},
		MarginPercent: margin,
		CapCents:      capCents,
	})
	if err != nil {
		return nil, err
	}
	area, ok := forecaster.Area(req.ToCell)
	if !ok || !forecaster.Ready() {
		return nil, errortypes.NewBusinessError(
			"That area is not in the freight forecast, so a driver cannot be sent there from the outlook.",
		)
	}

	target := dispatchmarket.CenterOf(req.ToCell)
	miles := geoutils.PlanningRoadMiles(
		from.Latitude,
		from.Longitude,
		target.Latitude,
		target.Longitude,
	)
	freeAt := max(start.FreeAt, now)
	fromLabel := ""
	if fromArea, known := forecaster.Area(fromCell); known {
		fromLabel = fromArea.Label
	}
	positioning := &driverpositioning.Positioning{
		WorkerID:    req.WorkerID,
		FromCell:    fromCell,
		FromLabel:   fromLabel,
		ToCell:      req.ToCell,
		ToLabel:     area.Label,
		ToLatitude:  target.Latitude,
		ToLongitude: target.Longitude,
		SlotStart:   req.SlotStart,
		FreeAt:      freeAt,
		ArriveAt:    freeAt + geoutils.PlanningDriveSeconds(miles),
		Miles:       miles,
		ValueCents: forecaster.ValueAt(
			demandforecast.WindowKey{Cell: req.ToCell, Start: req.SlotStart},
			req.WorkerID,
		).Cents,
		Status:      driverpositioning.StatusActive,
		CreatedByID: req.Actor.AuditActor().UserID,
	}

	replaced, err := s.repo.Replace(ctx, &driverpositioningports.ReplaceRequest{
		TenantInfo:  req.TenantInfo,
		Positioning: positioning,
	})
	if err != nil {
		return nil, err
	}

	if replaced.Previous != nil {
		s.record(ctx, req.Actor, req.TenantInfo, replaced.Previous, nil, permission.OpCancel,
			"Positioning toward "+replaced.Previous.ToLabel+" replaced by one toward "+
				positioning.ToLabel)
	}
	s.record(ctx, req.Actor, req.TenantInfo, positioning, nil, permission.OpCreate,
		"Driver sent toward "+positioning.ToLabel)
	s.tellDriver(ctx, req.TenantInfo, positioning, false)

	return positioning, nil
}

func (s *Service) CallOff(
	ctx context.Context,
	req *driverpositioningports.CallOffRequest,
) (*driverpositioning.Positioning, error) {
	before, err := s.repo.Get(ctx, &driverpositioningports.GetRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	positioning, err := s.repo.Cancel(ctx, &driverpositioningports.CancelRequest{
		TenantInfo:   req.TenantInfo,
		ID:           req.ID,
		CanceledByID: req.Actor.AuditActor().UserID,
		CanceledAt:   s.clock(),
	})
	if err != nil {
		return nil, err
	}

	s.record(ctx, req.Actor, req.TenantInfo, positioning, before, permission.OpCancel,
		"Positioning toward "+positioning.ToLabel+" called off")
	s.tellDriver(ctx, req.TenantInfo, positioning, true)

	return positioning, nil
}

func (s *Service) ListLive(
	ctx context.Context,
	req *driverpositioningports.LiveRequest,
) ([]*driverpositioning.Positioning, error) {
	return s.repo.ListLive(ctx, req)
}

func validateTarget(req *driverpositioningports.CreateRequest, now int64) error {
	multiErr := errortypes.NewMultiError()
	if req.WorkerID.IsNil() {
		multiErr.Add("workerId", errortypes.ErrRequired, "Choose a driver to send")
	}
	if req.ToCell <= 0 {
		multiErr.Add("toCell", errortypes.ErrRequired, "Choose an area to send the driver to")
	}
	first := dispatchmarket.SlotStart(now)
	if req.SlotStart != dispatchmarket.SlotStart(req.SlotStart) ||
		req.SlotStart < first || req.SlotStart >= first+outlookHorizon {
		multiErr.Add("slotStart", errortypes.ErrInvalid,
			"Choose a window in the next 48 hours of the outlook")
	}
	return multiErr.Err()
}

func (s *Service) driverStart(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	now int64,
) (demandforecastports.DriverStart, error) {
	rows, err := s.starts.ListDriverStarts(ctx, &demandforecastports.DriverStartsRequest{
		TenantInfo: tenantInfo,
		Now:        now,
		WorkerIDs:  []pulid.ID{workerID},
	})
	if err != nil {
		return demandforecastports.DriverStart{}, err
	}
	if len(rows) == 0 {
		return demandforecastports.DriverStart{}, errortypes.NewBusinessError(
			"This driver can't be sent: they are not available for dispatch, or nobody knows where they are.",
		)
	}

	return rows[0], nil
}

func (s *Service) record(
	ctx context.Context,
	actor *permissionports.RequestActor,
	tenantInfo pagination.TenantInfo,
	current, previous *driverpositioning.Positioning,
	operation permission.Operation,
	comment string,
) {
	auditActor := actor.AuditActor()
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceDriverPositioning,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	if err := s.audit.LogAction(ctx, params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log positioning audit", zap.Error(err))
	}
}

func (s *Service) tellDriver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	positioning *driverpositioning.Positioning,
	canceled bool,
) {
	location := s.forecasts.Location(ctx, tenantInfo)
	s.notify.Notify(ctx, &drivernotificationservice.DriverNotification{
		TenantInfo: tenantInfo,
		WorkerID:   positioning.WorkerID,
		EventType:  eventType,
		Priority:   notification.PriorityHigh,
		Context: notificationtemplate.DriverNotificationContext{
			AreaName:    positioning.ToLabel,
			ArriveBy:    time.Unix(positioning.ArriveAt, 0).In(location).Format(timeLayout),
			FreightFrom: time.Unix(positioning.SlotStart, 0).In(location).Format(timeLayout),
			Canceled:    canceled,
		},
		Link: "/dash",
		RelatedEntities: map[string]any{
			"positioningId": positioning.ID.String(),
		},
	})
}
