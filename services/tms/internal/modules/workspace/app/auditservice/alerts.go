package auditservice

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/maypok86/otter/v2"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	alertPolicyCacheSize = 20_000
	alertPolicyTTL       = time.Minute
	alertObserveTimeout  = 3 * time.Second
	alertCounterPrefix   = "audit:alert:"
)

type AlertsParams struct {
	fx.In

	Policies  auditports.AlertPolicyRepository
	Redis     *redis.Client `optional:"true"`
	Workflows kernelports.WorkflowStarter
	Metrics   *metrics.Registry
	Logger    *zap.Logger
}

type alertPolicyCache interface {
	GetIfPresent(key auditports.TenantKey) (map[audit.AlertKind]*audit.AlertPolicy, bool)
	Set(
		key auditports.TenantKey,
		value map[audit.AlertKind]*audit.AlertPolicy,
	) (map[audit.AlertKind]*audit.AlertPolicy, bool)
	Invalidate(key auditports.TenantKey) (map[audit.AlertKind]*audit.AlertPolicy, bool)
}

type Alerts struct {
	policies  auditports.AlertPolicyRepository
	redis     *redis.Client
	workflows kernelports.WorkflowStarter
	cache     alertPolicyCache
	metrics   *metrics.Registry
	l         *zap.Logger
}

func NewAlerts(p AlertsParams) *Alerts {
	return &Alerts{
		policies:  p.Policies,
		redis:     p.Redis,
		workflows: p.Workflows,
		cache: otter.Must(
			&otter.Options[auditports.TenantKey, map[audit.AlertKind]*audit.AlertPolicy]{
				MaximumSize: alertPolicyCacheSize,
				ExpiryCalculator: otter.ExpiryWriting[
					auditports.TenantKey, map[audit.AlertKind]*audit.AlertPolicy,
				](alertPolicyTTL),
			},
		),
		metrics: p.Metrics,
		l:       p.Logger.Named("service.audit-alerts"),
	}
}

func (a *Alerts) Observe(ctx context.Context, entries []*audit.Entry) {
	if a == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), alertObserveTimeout)
	defer cancel()

	for _, entry := range entries {
		if kind, ok := audit.AlertKindOf(entry); ok {
			a.raiseIfEnabled(ctx, entry, kind, entry.ID.String(), 1)
		}
		if kind, ok := audit.CountedAlertKindOf(entry); ok {
			a.count(ctx, entry, kind)
		}
	}
}

func (a *Alerts) InvalidatePolicies(tenant auditports.TenantKey) {
	if a == nil {
		return
	}
	a.cache.Invalidate(tenant)
}

func (a *Alerts) policy(
	ctx context.Context,
	entry *audit.Entry,
	kind audit.AlertKind,
) (*audit.AlertPolicy, bool) {
	def, ok := audit.AlertDefinitionFor(kind)
	if !ok {
		return nil, false
	}

	key := auditports.TenantKey{
		OrganizationID: entry.OrganizationID,
		BusinessUnitID: entry.BusinessUnitID,
	}
	policies, cached := a.cache.GetIfPresent(key)
	if !cached {
		tenantCtx := dbscope.WithValidTenant(ctx, dbscope.Tenant{
			OrganizationID: entry.OrganizationID,
			BusinessUnitID: entry.BusinessUnitID,
		})
		stored, err := a.policies.ListForTenant(tenantCtx, pagination.TenantInfo{
			OrgID: entry.OrganizationID,
			BuID:  entry.BusinessUnitID,
		})
		if err != nil {
			a.l.Warn("audit alert policies unavailable; using the defaults", zap.Error(err))
		}
		policies = make(map[audit.AlertKind]*audit.AlertPolicy, len(stored))
		for _, policy := range stored {
			policies[policy.Kind] = policy
		}
		if err == nil {
			a.cache.Set(key, policies)
		}
	}

	if policy, found := policies[kind]; found {
		return policy, policy.Enabled
	}

	return audit.DefaultAlertPolicy(def), true
}

func (a *Alerts) raiseIfEnabled(
	ctx context.Context,
	entry *audit.Entry,
	kind audit.AlertKind,
	dedupe string,
	count int64,
) {
	policy, enabled := a.policy(ctx, entry, kind)
	if !enabled {
		return
	}
	a.raise(ctx, entry, kind, policy, dedupe, count)
}

func (a *Alerts) count(ctx context.Context, entry *audit.Entry, kind audit.AlertKind) {
	if a.redis == nil {
		return
	}
	policy, enabled := a.policy(ctx, entry, kind)
	if !enabled {
		return
	}

	window := time.Duration(policy.WindowMinutes) * time.Minute
	bucket := entry.Timestamp / int64(window.Seconds())
	key := fmt.Sprintf(
		"%s%s:%s:%s:%s:%d",
		alertCounterPrefix,
		entry.OrganizationID,
		entry.BusinessUnitID,
		kind,
		entry.PrincipalID,
		bucket,
	)

	pipe := a.redis.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window+time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		a.l.Warn(
			"audit alert counter unavailable",
			zap.String("kind", string(kind)),
			zap.Error(err),
		)
		return
	}

	if incr.Val() != int64(policy.Threshold) {
		return
	}
	a.raise(
		ctx,
		entry,
		kind,
		policy,
		entry.PrincipalID.String()+":"+strconv.FormatInt(bucket, 10),
		incr.Val(),
	)
}

func (a *Alerts) raise(
	ctx context.Context,
	entry *audit.Entry,
	kind audit.AlertKind,
	policy *audit.AlertPolicy,
	dedupe string,
	count int64,
) {
	key := fmt.Sprintf("%s/%s", kind, dedupe)
	payload := &auditports.AlertPayload{
		TenantInfo: pagination.TenantInfo{
			OrgID: entry.OrganizationID,
			BuID:  entry.BusinessUnitID,
		},
		Kind:          kind,
		DedupeKey:     key,
		EntryID:       entry.ID,
		Action:        entry.Action,
		Resource:      entry.Resource,
		ResourceID:    entry.ResourceID,
		PrincipalID:   entry.PrincipalID,
		PrincipalType: entry.PrincipalType,
		ActorName:     entry.ActorName,
		EntityRef:     entry.EntityRef,
		Comment:       entry.Comment,
		Count:         count,
		Threshold:     policy.Threshold,
		WindowMinutes: policy.WindowMinutes,
		OccurredAt:    entry.Timestamp,
	}

	_, err := a.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID: fmt.Sprintf(
			"audit-alert/%s/%s/%s",
			entry.OrganizationID,
			entry.BusinessUnitID,
			key,
		),
		TaskQueue:                temporaltype.AuditTaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionTimeout: 10 * time.Minute,
	}, auditports.AlertWorkflowName, payload)
	if err != nil {
		a.l.Error("audit alert could not be raised",
			zap.String("kind", string(kind)),
			zap.String("entryId", entry.ID.String()),
			zap.Error(err),
		)
		return
	}

	if a.metrics != nil {
		a.metrics.Audit.RecordAlert(string(kind))
	}
}
