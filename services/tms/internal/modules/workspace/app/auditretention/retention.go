package auditretention

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	deleteBatchSize    = 5000
	maxBatchesPerClass = 200
	minPartitionMonths = 3
	day                = 24 * time.Hour
)

type Params struct {
	fx.In

	Entries    auditports.AuditRepository
	Seals      auditports.SealRepository
	AuthEvents iamports.AuthEventRepository
	Retention  tenantports.DataRetentionRepository
	Metrics    *metrics.Registry
	Logger     *zap.Logger
}

type Retention struct {
	entries    auditports.AuditRepository
	seals      auditports.SealRepository
	authEvents iamports.AuthEventRepository
	retention  tenantports.DataRetentionRepository
	metrics    *metrics.Registry
	now        func() time.Time
	l          *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Retention {
	return &Retention{
		entries:    p.Entries,
		seals:      p.Seals,
		authEvents: p.AuthEvents,
		retention:  p.Retention,
		metrics:    p.Metrics,
		now:        time.Now,
		l:          p.Logger.Named("audit.retention"),
	}
}

type SweepResult struct {
	Deleted           map[string]int64 `json:"deleted"`
	AuthEventsDeleted int64            `json:"authEventsDeleted"`
	Organizations     int              `json:"organizations"`
	Failed            int              `json:"failed"`
	SealsDeleted      int64            `json:"sealsDeleted"`
}

func (r *Retention) Sweep(ctx context.Context, heartbeat func(string)) (*SweepResult, error) {
	result := &SweepResult{Deleted: make(map[string]int64, len(audit.RetentionClasses))}
	now := r.now()

	authDeleted, err := r.authEvents.DeleteBefore(
		ctx,
		now.AddDate(0, 0, -iam.AuthEventRetentionDays).Unix(),
	)
	if err != nil {
		return nil, err
	}
	result.AuthEventsDeleted = authDeleted

	policies, err := r.retention.List(ctx)
	if err != nil {
		return nil, err
	}
	result.Organizations = len(policies.Items)

	var failures []error
	for _, policy := range policies.Items {
		for _, class := range audit.RetentionClasses {
			if heartbeat != nil {
				heartbeat(policy.OrganizationID.String() + ":" + string(class))
			}
			days := policy.AuditRetentionDays(string(class))
			before := now.Add(-time.Duration(days) * day).Unix()
			deleted, classErr := r.sweepClass(ctx, auditports.DeleteExpiredAuditEntriesRequest{
				OrgID:  policy.OrganizationID,
				BuID:   policy.BusinessUnitID,
				Class:  class,
				Before: before,
				Limit:  deleteBatchSize,
			}, heartbeat)
			result.Deleted[string(class)] += deleted
			r.metrics.Audit.RecordRetentionDeleted(string(class), deleted)
			if classErr != nil {
				failures = append(failures, classErr)
				result.Failed++
				r.l.Error("audit retention failed for an organization",
					zap.String("organizationId", policy.OrganizationID.String()),
					zap.String("class", string(class)),
					zap.Error(classErr),
				)
			}
		}
	}

	sealsDeleted, err := r.seals.DeleteSealsBefore(
		ctx,
		now.Add(-time.Duration(audit.MaxRetentionDays+1)*day).Unix(),
	)
	if err != nil {
		failures = append(failures, err)
	}
	result.SealsDeleted = sealsDeleted

	if len(failures) > 0 && result.Failed == result.Organizations*len(audit.RetentionClasses) &&
		result.Organizations > 0 {
		return result, errors.Join(failures...)
	}
	return result, nil
}

func (r *Retention) sweepClass(
	ctx context.Context,
	req auditports.DeleteExpiredAuditEntriesRequest,
	heartbeat func(string),
) (int64, error) {
	var total int64
	for range maxBatchesPerClass {
		deleted, err := r.entries.DeleteExpired(ctx, req)
		total += deleted
		if err != nil {
			return total, err
		}
		if deleted < int64(req.Limit) {
			return total, nil
		}
		if heartbeat != nil {
			heartbeat(req.OrgID.String() + ":" + string(req.Class))
		}
	}
	return total, nil
}

type PartitionResult struct {
	Horizon     int64 `json:"horizon"`
	MonthsAhead int   `json:"monthsAhead"`
	Short       bool  `json:"short"`
}

func (r *Retention) CheckPartitions(ctx context.Context) (*PartitionResult, error) {
	horizon, err := r.entries.PartitionHorizon(ctx)
	if err != nil {
		return nil, err
	}
	result := &PartitionResult{Horizon: horizon, MonthsAhead: monthsAhead(r.now(), horizon)}
	r.metrics.Audit.RecordPartitionMonthsAhead(result.MonthsAhead)
	if result.MonthsAhead < minPartitionMonths {
		result.Short = true
		r.l.Error("audit partitions run out soon; deploy, or run trenova-db, to create more",
			zap.Int("monthsAhead", result.MonthsAhead),
			zap.Time("partitionsEnd", time.Unix(horizon, 0).UTC()))
	}
	return result, nil
}

func monthsAhead(now time.Time, horizon int64) int {
	if horizon <= 0 {
		return 0
	}
	now = now.UTC()
	end := time.Unix(horizon, 0).UTC()
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	months := 0
	for start := nextMonth; !start.AddDate(0, 1, 0).After(end); start = start.AddDate(0, 1, 0) {
		months++
	}
	return months
}
