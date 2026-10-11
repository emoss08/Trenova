package accountingsyncjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Connections           accountingsyncports.AccountingConnectionService
	ConnectionsRepository accountingsyncports.AccountingConnectionRepository
	Mappings              accountingsyncports.AccountingMappingService
	Sync                  accountingsyncports.AccountingSyncService
	Dispatcher            accountingsyncports.AccountingSyncDispatcher
	Inbound               accountingsyncports.AccountingInboundService
	Poller                accountingsyncports.AccountingChangePoller
	Drift                 accountingsyncports.AccountingDriftReconciler
	DriftChecker          accountingsyncports.AccountingDriftChecker
	Publisher             agentports.AgentEventPublisher `optional:"true"`
	Logger                *zap.Logger
}

type Activities struct {
	connections  accountingsyncports.AccountingConnectionService
	connRepo     accountingsyncports.AccountingConnectionRepository
	mappings     accountingsyncports.AccountingMappingService
	sync         accountingsyncports.AccountingSyncService
	dispatcher   accountingsyncports.AccountingSyncDispatcher
	inbound      accountingsyncports.AccountingInboundService
	poller       accountingsyncports.AccountingChangePoller
	drift        accountingsyncports.AccountingDriftReconciler
	driftChecker accountingsyncports.AccountingDriftChecker
	publisher    agentports.AgentEventPublisher
	l            *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		connections:  p.Connections,
		connRepo:     p.ConnectionsRepository,
		mappings:     p.Mappings,
		sync:         p.Sync,
		dispatcher:   p.Dispatcher,
		inbound:      p.Inbound,
		poller:       p.Poller,
		drift:        p.Drift,
		driftChecker: p.DriftChecker,
		publisher:    p.Publisher,
		l:            p.Logger.Named("job.accounting-sync"),
	}
}

func (a *Activities) CheckAccountingConnectionsActivity(
	ctx context.Context,
) (*HealthSweepResult, error) {
	result := &HealthSweepResult{}
	for page := range healthMaxPages {
		sweep, err := a.connections.CheckDue(ctx, healthPageSize)
		if err != nil {
			return result, err
		}
		result.Pages = page + 1
		result.Listed += sweep.Listed
		result.Checked += sweep.Checked
		result.Failed += sweep.Failed
		activity.RecordHeartbeat(ctx, result.Pages)

		if sweep.Listed < healthPageSize || sweep.Checked == 0 {
			break
		}
	}

	if result.Failed > 0 {
		a.l.Warn("accounting connection checks failed",
			zap.Int("failed", result.Failed), zap.Int("checked", result.Checked))
	}
	return result, nil
}

func referenceError(err error) error {
	if err == nil {
		return nil
	}
	if errortypes.IsError(err) || errortypes.IsBusinessError(err) ||
		errortypes.IsNotFoundError(err) {
		return temporaltype.NewNonRetryableError(err.Error(), err).ToTemporalError()
	}
	return err
}

func (a *Activities) MarkAccountingReferenceRefreshStartedActivity(
	ctx context.Context,
	payload *RefreshReferencePayload,
) error {
	return referenceError(
		a.mappings.MarkRefreshStarted(ctx, payload.TenantInfo(), payload.ConnectionID),
	)
}

func (a *Activities) MarkAccountingReferenceRefreshFinishedActivity(
	ctx context.Context,
	payload *RefreshReferencePayload,
	failure string,
) error {
	return referenceError(
		a.mappings.MarkRefreshFinished(ctx, payload.TenantInfo(), payload.ConnectionID, failure),
	)
}

func (a *Activities) PullAccountingReferenceActivity(
	ctx context.Context,
	payload *RefreshReferencePayload,
	kind accountingsync.ReferenceKind,
) (*ReferenceKindResult, error) {
	pull, err := a.mappings.PullReference(ctx, payload.TenantInfo(), payload.ConnectionID, kind)
	if err != nil {
		return nil, referenceError(err)
	}
	return &ReferenceKindResult{
		Kind:    string(pull.Kind),
		Fetched: pull.Fetched,
		Removed: pull.Removed,
	}, nil
}

func (a *Activities) RescoreAccountingMappingsActivity(
	ctx context.Context,
	payload *RefreshReferencePayload,
) (*RescoreReferenceResult, error) {
	result, err := a.mappings.Rescore(ctx, payload.TenantInfo(), payload.ConnectionID)
	if err != nil {
		return nil, referenceError(err)
	}
	return &RescoreReferenceResult{
		Targets:    result.Targets,
		Created:    result.Created,
		Updated:    result.Updated,
		Proposed:   result.Proposed,
		NeedsModel: result.NeedsModel,
	}, nil
}

func (a *Activities) AccountingMappingModelPassActivity(
	ctx context.Context,
	payload *RefreshReferencePayload,
	mappingIDs []pulid.ID,
) (int, error) {
	stop := modelcall.Heartbeat(ctx)
	defer stop()

	applied, err := a.mappings.ModelPass(
		ctx,
		payload.TenantInfo(),
		payload.ConnectionID,
		mappingIDs,
	)
	if err == nil {
		return applied, nil
	}
	if errors.Is(err, context.Canceled) {
		return applied, err
	}
	if modelcall.Transient(err) && !modelcall.FinalAttempt(ctx, modelPassAttempts) {
		return applied, modelcall.Classify(err)
	}
	a.l.Warn("the accounting mapping model pass failed; keeping the deterministic proposals",
		zap.String("connectionId", payload.ConnectionID.String()), zap.Error(err))
	return applied, nil
}

func (a *Activities) ListAccountingReferenceConnectionsActivity(
	ctx context.Context,
	afterID pulid.ID,
) (*ReferenceSweepPage, error) {
	conns, err := a.connRepo.ListActive(
		ctx,
		accountingsyncports.ListActiveAccountingConnectionsRequest{
			AfterID: afterID,
			Limit:   referenceSweepPageSize,
		},
	)
	if err != nil {
		return nil, err
	}
	page := &ReferenceSweepPage{
		Connections: make([]RefreshReferencePayload, 0, len(conns)),
		More:        len(conns) == referenceSweepPageSize,
	}
	for _, conn := range conns {
		page.Connections = append(page.Connections, RefreshReferencePayload{
			OrganizationID: conn.OrganizationID,
			BusinessUnitID: conn.BusinessUnitID,
			ConnectionID:   conn.ID,
		})
		page.LastID = conn.ID
	}
	return page, nil
}
