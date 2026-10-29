package tendereffects

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenteventservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentinvalidationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Effects struct {
	l                *zap.Logger
	repo             tenderports.TenderRepository
	eventService     shipmenteventports.ShipmentEventService
	auditService     auditports.AuditService
	invalidator      shipmentinvalidationports.ShipmentInvalidator
	notifications    *notificationservice.Service
	dispatchControls dispatchcontrolports.DispatchControlRepository
	intelGate        carrierintelports.CarrierIntelGate
	lifecycle        carrierintelports.CarrierLifecycleObserver
}

func (s *Effects) LogAudit(p *AuditParams) {
	if s.auditService == nil {
		return
	}

	params := &auditports.LogActionParams{
		Resource:       permission.ResourceTender,
		ResourceID:     p.TenderID.String(),
		Operation:      p.Operation,
		UserID:         p.UserID,
		OrganizationID: p.TenantInfo.OrgID,
		BusinessUnitID: p.TenantInfo.BuID,
	}
	if p.Current != nil {
		params.CurrentState = jsonutils.MustToJSON(p.Current)
	}
	if p.UserID.IsNil() {
		params.PrincipalType = permissionports.PrincipalTypeSystem
		params.PrincipalID = permissionports.SystemPrincipalID
	}

	options := []auditports.LogOption{auditservice.WithComment(p.Comment)}
	if p.Previous != nil {
		params.PreviousState = jsonutils.MustToJSON(p.Previous)
		if p.Current != nil {
			options = append(options, auditservice.WithDiff(p.Previous, p.Current))
		}
	}

	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log tender audit action", zap.Error(err))
	}
}

func (s *Effects) PublishShipmentInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	action string,
) {
	if offer.Tender == nil {
		return
	}
	shipmentinvalidationports.InvalidateShipments(
		ctx,
		s.invalidator,
		shipmentinvalidationports.ShipmentInvalidationByUser(
			tenantInfo,
			offer.Tender.ShipmentID,
			action,
		),
	)
}

type Params struct {
	fx.In

	Logger           *zap.Logger
	Repo             tenderports.TenderRepository
	EventService     shipmenteventports.ShipmentEventService
	AuditService     auditports.AuditService
	Invalidator      shipmentinvalidationports.ShipmentInvalidator
	Notifications    *notificationservice.Service                   `optional:"true"`
	DispatchControls dispatchcontrolports.DispatchControlRepository `optional:"true"`
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Effects {
	return &Effects{
		l:                p.Logger.Named("service.tender"),
		repo:             p.Repo,
		eventService:     p.EventService,
		auditService:     p.AuditService,
		invalidator:      p.Invalidator,
		notifications:    p.Notifications,
		dispatchControls: p.DispatchControls,
	}
}

func (s *Effects) SetIntelGate(value carrierintelports.CarrierIntelGate) {
	s.intelGate = value
}

func (s *Effects) SetLifecycleObserver(value carrierintelports.CarrierLifecycleObserver) {
	s.lifecycle = value
}

func (s *Effects) MarkCarriersUsed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offers []*tender.TenderOffer,
) {
	if s.lifecycle == nil || len(offers) == 0 {
		return
	}
	ids := make([]pulid.ID, 0, len(offers))
	for _, offer := range offers {
		ids = append(ids, offer.CarrierID)
	}
	s.lifecycle.CarriersUsed(ctx, tenantInfo, ids)
}

// NotifyDispatch raises a global in-app notification for dispatchers. Failures
// are logged only: a notification must never fail the workflow transition it
// decorates.
func (s *Effects) NotifyDispatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	n *DispatchNotification,
) {
	if s.notifications == nil {
		return
	}

	link := n.Link
	if link == "" {
		link = dispatchConsoleLink
	}
	buID := tenantInfo.BuID
	entity := &notification.Notification{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: &buID,
		EventType:      n.EventType,
		Priority:       n.Priority,
		Channel:        notification.ChannelGlobal,
		Title:          n.Title,
		Message:        n.Message,
		Source:         notificationSource,
		Data:           map[string]any{"link": link},
	}
	if n.CorrelationID != "" {
		correlationID := n.CorrelationID
		entity.CorrelationID = &correlationID
	}

	if _, err := s.notifications.Create(ctx, entity); err != nil {
		s.l.Warn("failed to create tender notification",
			zap.Error(err), zap.String("eventType", n.EventType))
	}
}

func (s *Effects) RecordLateOfferResponse(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	offer *tender.TenderOffer,
	action tender.ResponseAction,
	source tender.ResponseSource,
) {
	if err := s.repo.RecordLateOfferResponse(ctx, &tenderports.RecordLateOfferResponseRequest{
		TenantInfo: tenantInfo,
		OfferID:    offer.ID,
		Action:     action,
		Source:     source,
		OccurredAt: timeutils.NowUnix(),
	}); err != nil {
		s.l.Warn("failed to record late tender response", zap.Error(err))
	}

	s.RecordEvent(ctx, shipmenteventservice.BuildTenderLateResponse(
		shipmenteventservice.TenantRefFor(tenantInfo),
		TenderRefForOffer(offer),
		CarrierNameOf(offer),
		action,
		source,
		shipmenteventservice.ActorFor(tenantInfo),
	))
}

// Policy is how the organization's tenders treat counter-offers and an
// exhausted waterfall. Settings that cannot be read fall back to the default
// policy rather than stopping a tender mid-flight.
func (s *Effects) Policy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) dispatchcontrol.TenderPolicy {
	if s.dispatchControls == nil {
		return dispatchcontrol.DefaultTenderPolicy()
	}
	control, err := s.dispatchControls.GetByOrgID(
		ctx,
		dispatchcontrolports.GetDispatchControlRequest{
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to read tender policy; using defaults", zap.Error(err))
		}
		return dispatchcontrol.DefaultTenderPolicy()
	}
	return control.TenderPolicy()
}

func (s *Effects) IntelGates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
	refresh bool,
) map[pulid.ID]*carrier.IntelGate {
	if s.intelGate == nil || len(carrierIDs) == 0 {
		return nil
	}
	if refresh {
		s.intelGate.EnsureFresh(ctx, &carrierintelports.CarrierIntelEnsureFreshRequest{
			TenantInfo: tenantInfo,
			CarrierIDs: carrierIDs,
		})
	}
	gates, err := s.intelGate.GateFor(ctx, tenantInfo, carrierIDs)
	if err != nil {
		s.l.Warn("failed to evaluate carrier intelligence gates", zap.Error(err))
		return nil
	}
	return gates
}

func (s *Effects) RecordEvent(
	ctx context.Context,
	params *shipmenteventports.RecordShipmentEventParams,
) {
	if params == nil || params.ShipmentID.IsNil() {
		return
	}

	if err := s.eventService.Record(ctx, params); err != nil {
		s.l.Warn("failed to record tender event", zap.Error(err))
	}
}
