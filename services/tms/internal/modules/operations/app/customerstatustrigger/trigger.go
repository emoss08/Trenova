package customerstatustrigger

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenteventports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/ttlcache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const notifyingCacheTTL = time.Minute

type Params struct {
	fx.In

	Requester customertrackingports.EvaluationRequester
	Facts     customertrackingports.FactRepository
	Logger    *zap.Logger
}

// Trigger asks for a shipment's status emails to be evaluated when it changes
// in a way that could owe its customer one: a stop is reached or left, its
// status moves, or its projected arrivals are refreshed. Organizations with no
// customer rule on are skipped without starting anything.
type Trigger struct {
	requester customertrackingports.EvaluationRequester
	notifying tenantSet
	l         *zap.Logger
}

var (
	_ shipmenteventports.ShipmentEventObserver = (*Trigger)(nil)
	_ stopetaports.Observer                    = (*Trigger)(nil)
)

func New(p Params) *Trigger {
	return &Trigger{
		requester: p.Requester,
		notifying: newNotifyingTenants(p.Facts),
		l:         p.Logger.Named("customer-status-trigger"),
	}
}

func Evaluates(eventType shipmentevent.Type) bool {
	switch eventType {
	case shipmentevent.TypeShipmentUpdated,
		shipmentevent.TypeStatusChanged,
		shipmentevent.TypeShipmentCanceled,
		shipmentevent.TypeShipmentUncanceled,
		shipmentevent.TypeMoveStatusChanged,
		shipmentevent.TypeMoveDeparted,
		shipmentevent.TypeMoveArrived,
		shipmentevent.TypeStopCompleted:
		return true
	case shipmentevent.TypeShipmentCreated,
		shipmentevent.TypeDriverAssigned,
		shipmentevent.TypeDriverReassigned,
		shipmentevent.TypeDriverUnassigned,
		shipmentevent.TypeCarrierAssigned,
		shipmentevent.TypeCarrierUnassigned,
		shipmentevent.TypeOwnershipTransferred,
		shipmentevent.TypeTenderOffered,
		shipmentevent.TypeTenderAccepted,
		shipmentevent.TypeTenderDeclined,
		shipmentevent.TypeTenderExpired,
		shipmentevent.TypeTenderWithdrawn,
		shipmentevent.TypeTenderNeedsReview,
		shipmentevent.TypeRoutingGuideExhausted,
		shipmentevent.TypeTenderLateResponse,
		shipmentevent.TypeTenderDeliveryFailed,
		shipmentevent.TypeTenderEntrySkipped,
		shipmentevent.TypeTenderEntryWarned,
		shipmentevent.TypeHoldPlaced,
		shipmentevent.TypeHoldUpdated,
		shipmentevent.TypeHoldReleased,
		shipmentevent.TypeCommentPosted:
		return false
	default:
		return false
	}
}

func (t *Trigger) OnShipmentEvent(ctx context.Context, event *shipmentevent.Event) error {
	if event == nil || !Evaluates(event.Type) {
		return nil
	}
	tenant := pagination.TenantInfo{OrgID: event.OrganizationID, BuID: event.BusinessUnitID}
	if !t.notifying.has(ctx, tenant) {
		return nil
	}
	t.requester.RequestEvaluation(ctx, tenant, event.ShipmentID)
	return nil
}

func (t *Trigger) StopEtasRefreshed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) {
	if len(shipmentIDs) == 0 || !t.notifying.has(ctx, tenantInfo) {
		return
	}
	for _, shipmentID := range shipmentIDs {
		t.requester.RequestEvaluation(ctx, tenantInfo, shipmentID)
	}
}

type tenantSet interface {
	has(ctx context.Context, tenant pagination.TenantInfo) bool
}

type tenantKey struct {
	orgID pulid.ID
	buID  pulid.ID
}

type notifyingTenants struct {
	facts customertrackingports.FactRepository
	cache *ttlcache.Cache[struct{}, map[tenantKey]struct{}]
}

func newNotifyingTenants(facts customertrackingports.FactRepository) tenantSet {
	return &notifyingTenants{
		facts: facts,
		cache: ttlcache.New[struct{}, map[tenantKey]struct{}](ttlcache.Options{
			TTL:        notifyingCacheTTL,
			MaxEntries: 1,
		}),
	}
}

// has reports whether any customer in the organization has a status email rule
// on. When the answer cannot be read it says yes: a wasted evaluation is cheap,
// a missed email is not.
func (n *notifyingTenants) has(ctx context.Context, tenant pagination.TenantInfo) bool {
	set, ok := n.cache.Get(struct{}{})
	if !ok {
		tenants, err := n.facts.ListNotifyingTenants(ctx)
		if err != nil {
			return true
		}
		set = make(map[tenantKey]struct{}, len(tenants))
		for _, t := range tenants {
			set[tenantKey{orgID: t.OrgID, buID: t.BuID}] = struct{}{}
		}
		n.cache.Set(struct{}{}, set)
	}
	_, found := set[tenantKey{orgID: tenant.OrgID, buID: tenant.BuID}]
	return found
}
