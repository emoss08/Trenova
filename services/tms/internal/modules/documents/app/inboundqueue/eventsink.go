package inboundqueue

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ inboundmessageports.TriageEventSink = (*EventSink)(nil)

type EventSinkParams struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	QueueRepo    inboundmessageports.TriageQueueRepository
	ActivityRepo inboundmessageports.TriageActivityRepository
	Notifier     *Notifier
}

type EventSink struct {
	l            *zap.Logger
	db           kernelports.Transactor
	queueRepo    inboundmessageports.TriageQueueRepository
	activityRepo inboundmessageports.TriageActivityRepository
	notifier     *Notifier
	now          func() int64
}

func NewEventSink(p EventSinkParams) *EventSink {
	return &EventSink{
		l:            p.Logger.Named("service.triage-event-sink"),
		db:           p.DB,
		queueRepo:    p.QueueRepo,
		activityRepo: p.ActivityRepo,
		notifier:     p.Notifier,
		now:          timeutils.NowUnix,
	}
}

func (e *EventSink) ShipmentDelivered(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	e.wake(ctx, &inboundmessageports.WakeByEventRequest{
		TenantInfo: tenantInfo,
		Event:      inboundmessage.SnoozeEventShipmentDelivered,
		WatchID:    shipmentID,
	})
}

func (e *EventSink) DocumentAttached(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	documentTypeID pulid.ID,
) {
	if documentTypeID.IsNil() {
		return
	}

	e.wake(ctx, &inboundmessageports.WakeByEventRequest{
		TenantInfo:     tenantInfo,
		Event:          inboundmessage.SnoozeEventDocumentAttached,
		WatchID:        shipmentID,
		DocumentTypeID: documentTypeID,
	})
}

func (e *EventSink) InvoicePaid(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	e.wake(ctx, &inboundmessageports.WakeByEventRequest{
		TenantInfo: tenantInfo,
		Event:      inboundmessage.SnoozeEventInvoicePaid,
		WatchID:    shipmentID,
	})
}

func (e *EventSink) wake(ctx context.Context, req *inboundmessageports.WakeByEventRequest) {
	if req.WatchID.IsNil() || req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return
	}

	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	})
	req.Now = e.now()

	var woken []*inboundmessageports.QueueState
	err := e.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var wErr error
		woken, wErr = e.queueRepo.WakeByEvent(txCtx, *req)
		if wErr != nil || len(woken) == 0 {
			return wErr
		}

		return e.activityRepo.Insert(txCtx, eventActivities(woken, req))
	})
	if err != nil {
		e.l.Error("could not wake the messages snoozed on an event",
			zap.String("event", string(req.Event)),
			zap.String("watchId", req.WatchID.String()),
			zap.Error(err))

		return
	}
	if len(woken) == 0 {
		return
	}

	messages := make([]*inboundmessage.InboundMessage, 0, len(woken))
	for _, state := range woken {
		messages = append(messages, inboundmessageports.MessageFromState(state))
	}
	e.notifier.MessagesChanged(ctx, messages...)
}

func eventActivities(
	woken []*inboundmessageports.QueueState,
	req *inboundmessageports.WakeByEventRequest,
) []*triage.Activity {
	activities := make([]*triage.Activity, 0, len(woken))
	for _, state := range woken {
		activity := activityOf(state, triage.ActivityWoken).About(string(req.Event), "")
		activity.RecordID = req.WatchID
		activities = append(activities, activity)
	}

	return activities
}
