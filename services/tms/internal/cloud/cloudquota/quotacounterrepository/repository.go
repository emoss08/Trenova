package quotacounterrepository

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrUnsupportedMeter = errors.New("meter has no quota counter")
	ErrTenantRequired   = errors.New("quota counting requires an organization and business unit")
)

const lockStatement = "SELECT pg_advisory_xact_lock(hashtextextended(?, 0))"

type counter func(ctx context.Context, db bun.IDB, req *repositories.QuotaCountRequest) (int64, error)

var counters = map[platformcatalog.MeterKey]counter{
	platformcatalog.MeterShipmentsTotal:          countShipments,
	platformcatalog.MeterRecurringShipmentSeries: countRecurringShipments,
	platformcatalog.MeterCustomersTotal:          countCustomers,
	platformcatalog.MeterLocationsTotal:          countLocations,
	platformcatalog.MeterWorkersTotal:            countWorkers,
	platformcatalog.MeterTractorsTotal:           countTractors,
	platformcatalog.MeterTrailersTotal:           countTrailers,
	platformcatalog.MeterUserSeats:               countUserSeats,
	platformcatalog.MeterDocumentUploads:         countDocuments,
	platformcatalog.MeterDocumentStorageBytes:    sumDocumentBytes,
	platformcatalog.MeterAIAssistantMessages:     countAssistantMessages,
	platformcatalog.MeterAISpendCents:            sumAISpendCents,
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.QuotaCounterRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.quota-counter-repository"),
	}
}

func Meters() []platformcatalog.MeterKey {
	return slices.Sorted(maps.Keys(counters))
}

func (r *repository) Supports(meter platformcatalog.MeterKey) bool {
	_, ok := counters[meter]
	return ok
}

func (r *repository) Lock(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	meter platformcatalog.MeterKey,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if tenantInfo.OrgID.IsNil() {
			return ErrTenantRequired
		}

		key := tenantInfo.OrgID.String() + ":" + string(meter)
		if _, err := r.db.DBForContext(ctx).NewRaw(lockStatement, key).Exec(ctx); err != nil {
			r.l.Error("failed to take quota lock",
				zap.String("organizationId", tenantInfo.OrgID.String()),
				zap.String("meter", string(meter)),
				zap.Error(err),
			)
			return fmt.Errorf("lock quota %s: %w", meter, err)
		}

		return nil
	})
}

func (r *repository) Count(ctx context.Context, req *repositories.QuotaCountRequest) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
			return 0, ErrTenantRequired
		}

		count, ok := counters[req.Meter]
		if !ok {
			return 0, fmt.Errorf("%w: %s", ErrUnsupportedMeter, req.Meter)
		}

		used, err := count(ctx, r.db.DBForContext(ctx), req)
		if err != nil {
			r.l.Error("failed to count quota usage",
				zap.String("organizationId", req.TenantInfo.OrgID.String()),
				zap.String("meter", string(req.Meter)),
				zap.Error(err),
			)
			return 0, fmt.Errorf("count quota %s: %w", req.Meter, err)
		}

		return used, nil
	})
}

func (r *repository) OrganizationTimezone(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		cols := buncolgen.OrganizationColumns

		var timezone string
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*tenant.Organization)(nil)).
			Column(cols.Timezone.Bare()).
			Where(cols.ID.Eq(), tenantInfo.OrgID).
			Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
			Limit(1).
			Scan(ctx, &timezone)
		if err != nil {
			return "", dberror.HandleNotFoundError(err, "Organization")
		}

		return timezone, nil
	})
}

func countModel(
	ctx context.Context,
	db bun.IDB,
	model any,
	apply func(*bun.SelectQuery) *bun.SelectQuery,
) (int64, error) {
	count, err := db.NewSelect().Model(model).Apply(apply).Count(ctx)
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

func countShipments(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(ctx, db, (*shipment.Shipment)(nil), buncolgen.ShipmentApplyTenant(req.TenantInfo))
}

func countRecurringShipments(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(
		ctx,
		db,
		(*recurringshipment.RecurringShipment)(nil),
		buncolgen.RecurringShipmentApplyTenant(req.TenantInfo),
	)
}

func countCustomers(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(ctx, db, (*customer.Customer)(nil), buncolgen.CustomerApplyTenant(req.TenantInfo))
}

func countLocations(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(ctx, db, (*location.Location)(nil), buncolgen.LocationApplyTenant(req.TenantInfo))
}

func countWorkers(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(ctx, db, (*worker.Worker)(nil), buncolgen.WorkerApplyTenant(req.TenantInfo))
}

func countTractors(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(ctx, db, (*tractor.Tractor)(nil), buncolgen.TractorApplyTenant(req.TenantInfo))
}

func countTrailers(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(ctx, db, (*trailer.Trailer)(nil), buncolgen.TrailerApplyTenant(req.TenantInfo))
}

func countUserSeats(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	memberships := buncolgen.OrganizationMembershipColumns
	users := buncolgen.UserColumns
	now := req.WindowEnd
	if now <= 0 {
		now = timeutils.NowUnix()
	}

	count, err := db.NewSelect().
		Model((*tenant.OrganizationMembership)(nil)).
		Join(
			"JOIN "+buncolgen.UserTable.As(buncolgen.UserTable.Alias)+" ON "+
				users.ID.EqColumn(memberships.UserID),
		).
		Apply(buncolgen.OrganizationMembershipApplyTenant(req.TenantInfo)).
		Where(users.Username.Ne(), tenant.SystemUsername).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(memberships.ExpiresAt.IsNull()).
				WhereOr(memberships.ExpiresAt.Gt(), now)
		}).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

func countDocuments(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	return countModel(ctx, db, (*document.Document)(nil), buncolgen.DocumentApplyTenant(req.TenantInfo))
}

func sumDocumentBytes(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	var total int64
	err := db.NewSelect().
		Model((*document.Document)(nil)).
		ColumnExpr(buncolgen.DocumentColumns.FileSize.Expr("COALESCE(SUM({}), 0)::bigint")).
		Apply(buncolgen.DocumentApplyTenant(req.TenantInfo)).
		Scan(ctx, &total)
	if err != nil {
		return 0, err
	}

	return total, nil
}

func countAssistantMessages(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	cols := buncolgen.AssistantTurnColumns

	q := db.NewSelect().
		Model((*conversation.AssistantTurn)(nil)).
		Apply(buncolgen.AssistantTurnApplyTenant(req.TenantInfo)).
		Where(cols.Origin.Eq(), conversation.AssistantTurnOriginPerson)
	q = applyWindow(q, cols.CreatedAt, req)

	count, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

func sumAISpendCents(
	ctx context.Context,
	db bun.IDB,
	req *repositories.QuotaCountRequest,
) (int64, error) {
	cols := buncolgen.AIUsageRecordColumns

	var total int64
	q := db.NewSelect().
		Model((*aiusage.AIUsageRecord)(nil)).
		ColumnExpr(cols.CostUSD.Expr("CEIL(COALESCE(SUM({}), 0) * 100)::bigint")).
		Apply(buncolgen.AIUsageRecordApplyTenant(req.TenantInfo))
	q = applyWindow(q, cols.CreatedAt, req)

	if err := q.Scan(ctx, &total); err != nil {
		return 0, err
	}

	return total, nil
}

func applyWindow(
	q *bun.SelectQuery,
	column buncolgen.Column,
	req *repositories.QuotaCountRequest,
) *bun.SelectQuery {
	if req.WindowStart > 0 {
		q = q.Where(column.Gte(), req.WindowStart)
	}
	if req.WindowEnd > 0 {
		q = q.Where(column.Lt(), req.WindowEnd)
	}

	return q
}
