package podchasejobs

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/agent/app/podchase"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/podchasejobs/podchasecontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	// chasePageSize is the shipments one activity asks about: each is a run
	// record, a dispatched call and a recorded proposal, no model call.
	chasePageSize   = 25
	chaseRetryAfter = 5 * time.Minute
	promptVersion   = "routine.pod-chase.v1"
)

type ActivitiesParams struct {
	fx.In

	Queue   workqueueports.Queue
	Facts   podchaseports.FactsReader
	RunRepo agentruntimeports.AgentRunRepository
	Asker   *podchase.Asker
	Logger  *zap.Logger
}

type Activities struct {
	queue   workqueueports.Queue
	facts   podchaseports.FactsReader
	runRepo agentruntimeports.AgentRunRepository
	asker   *podchase.Asker
	l       *zap.Logger
}

func NewActivities(
	p ActivitiesParams, //nolint:gocritic // fx requires an fx.In parameter struct by value
) *Activities {
	return &Activities{
		queue:   p.Queue,
		facts:   p.Facts,
		runRepo: p.RunRepo,
		asker:   p.Asker,
		l:       p.Logger.Named("job.pod-chase"),
	}
}

// ChasePODsActivity asks for a page of the tenant's queued shipments' proofs
// of delivery. A shipment whose request could not be filed is due again
// after chaseRetryAfter; the rest are settled.
func (a *Activities) ChasePODsActivity(
	ctx context.Context,
	payload *temporaljobs.DrainPayload,
) (*temporaljobs.DrainPage, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())

	return temporaljobs.DrainDue(ctx, &temporaljobs.DrainDueParams{
		Queue:      a.queue,
		Kind:       podchasecontract.DrainKind,
		Tenant:     tenant,
		PageSize:   chasePageSize,
		RetryAfter: chaseRetryAfter,
		Handle: func(ctx context.Context, shipments []pulid.ID, agents map[pulid.ID]string) map[pulid.ID]error {
			failed := make(map[pulid.ID]error)
			for definitionID, group := range byAgent(shipments, agents) {
				for id, err := range a.chase(ctx, tenant, definitionID, group) {
					failed[id] = err
				}
			}

			return failed
		},
	})
}

func byAgent(shipments []pulid.ID, agents map[pulid.ID]string) map[pulid.ID][]pulid.ID {
	groups := make(map[pulid.ID][]pulid.ID)
	for _, id := range shipments {
		definitionID := pulid.ID(agents[id])
		groups[definitionID] = append(groups[definitionID], id)
	}

	return groups
}

// chase asks about one agent's shipments. A shipment no longer missing its
// proof of delivery, or already in an open run of the agent, is settled as
// it is. So is one nobody can be asked about: a model could not reach them
// either, and the Watchtower already shows a person the shipment.
func (a *Activities) chase(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
	shipments []pulid.ID,
) map[pulid.ID]error {
	failed := make(map[pulid.ID]error)
	failAll := func(err error) map[pulid.ID]error {
		for _, id := range shipments {
			failed[id] = err
		}
		return failed
	}

	c, err := a.asker.For(ctx, tenant, definitionID)
	if err != nil {
		return failAll(err)
	}
	if c == nil {
		return failed
	}

	facts, err := a.facts.ListPODChaseFacts(ctx, &podchaseports.ListFactsRequest{
		TenantInfo:      tenant,
		ShipmentIDs:     shipments,
		DeliveredBefore: timeutils.NowUnix(),
	})
	if err != nil {
		return failAll(err)
	}
	open, err := a.runRepo.ListOpenSubjects(ctx, &agentruntimeports.ListOpenSubjectsRequest{
		TenantInfo:   tenant,
		DefinitionID: definitionID,
		SubjectIDs:   shipments,
	})
	if err != nil {
		return failAll(err)
	}
	busy := make(map[pulid.ID]struct{}, len(open))
	for _, id := range open {
		busy[id] = struct{}{}
	}

	for _, fact := range facts {
		if ctx.Err() != nil {
			failed[fact.ShipmentID] = ctx.Err()
			continue
		}
		if _, ok := busy[fact.ShipmentID]; ok {
			continue
		}
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx, fact.ShipmentID.String())
		}
		_, cErr := a.asker.Ask(ctx, c, fact)
		if cErr != nil && !errors.Is(cErr, podchase.ErrNobodyToAsk) {
			a.l.Warn("could not ask for a proof of delivery; it is tried again",
				zap.String("shipmentId", fact.ShipmentID.String()),
				zap.Error(cErr))
			failed[fact.ShipmentID] = cErr
		}
	}

	return failed
}
