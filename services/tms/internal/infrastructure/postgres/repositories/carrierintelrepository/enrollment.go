package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultNeedingSyncLimit   = 500
	maxNeedingSyncLimit       = 5000
	defaultActiveLimit        = 1000
	maxActiveLimit            = 5000
	desiredEnrollmentCapacity = 64
)

var (
	enrollPendingVendorStates = []carrierintel.VendorState{
		carrierintel.VendorStateUnknown,
		carrierintel.VendorStatePendingAdd,
	}
	unenrollPendingVendorStates = []carrierintel.VendorState{
		carrierintel.VendorStateActive,
		carrierintel.VendorStatePendingRemove,
		carrierintel.VendorStatePendingAdd,
	}
)

type enrollmentRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewEnrollmentRepository(p Params) repositories.CarrierMonitoringEnrollmentRepository {
	return &enrollmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-monitoring-enrollment-repository"),
	}
}

func (r *enrollmentRepository) ListBySubjects(
	ctx context.Context,
	req *repositories.ListEnrollmentsBySubjectRequest,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	if len(req.SubjectIDs) == 0 {
		return []*carrierintel.CarrierMonitoringEnrollment{}, nil
	}

	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	entities := make([]*carrierintel.CarrierMonitoringEnrollment, 0, len(req.SubjectIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierMonitoringEnrollmentScopeTenant(sq, req.TenantInfo).
				Where(cols.Provider.Eq(), req.Provider).
				Where(cols.SubjectType.Eq(), req.SubjectType).
				Where(cols.SubjectID.In(), bun.List(req.SubjectIDs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier monitoring enrollments by subject", zap.Error(err))
		return nil, fmt.Errorf("list carrier monitoring enrollments by subject: %w", err)
	}

	return entities, nil
}

func (r *enrollmentRepository) GetByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	if len(carrierIDs) == 0 {
		return []*carrierintel.CarrierMonitoringEnrollment{}, nil
	}

	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	entities := make([]*carrierintel.CarrierMonitoringEnrollment, 0, len(carrierIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierMonitoringEnrollmentScopeTenant(sq, tenantInfo).
				Where(cols.CarrierID.In(), bun.List(carrierIDs))
		}).
		OrderExpr(cols.DesiredState.Expr("{} = ? DESC"), carrierintel.DesiredStateEnrolled).
		Order(cols.UpdatedAt.OrderDesc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get carrier monitoring enrollments by carrier", zap.Error(err))
		return nil, fmt.Errorf("get carrier monitoring enrollments by carrier: %w", err)
	}

	return entities, nil
}

func (r *enrollmentRepository) ListDesired(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	entities := make([]*carrierintel.CarrierMonitoringEnrollment, 0, desiredEnrollmentCapacity)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierMonitoringEnrollmentScopeTenant(sq, tenantInfo).
				Where(cols.Provider.Eq(), provider).
				Where(cols.DesiredState.Eq(), carrierintel.DesiredStateEnrolled)
		}).
		Order(cols.ID.OrderAsc()).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list desired carrier monitoring enrollments", zap.Error(err))
		return nil, fmt.Errorf("list desired carrier monitoring enrollments: %w", err)
	}

	return entities, nil
}

func buildEnrollmentUpsert(
	db bun.IDB,
	rows *[]*carrierintel.CarrierMonitoringEnrollment,
) *bun.InsertQuery {
	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	return db.NewInsert().
		Model(rows).
		On(
			"CONFLICT (organization_id, business_unit_id, provider, subject_type, subject_id) DO UPDATE",
		).
		Set(cols.DesiredState.SetExcluded()).
		Set(cols.VendorState.SetExcluded()).
		Set(cols.Reason.SetExcluded()).
		Set(cols.SubjectName.SetExcluded()).
		Set(cols.DOTNumber.SetExcluded()).
		Set(cols.DocketNumber.SetExcluded()).
		Set(cols.CarrierID.SetExcluded()).
		Set(cols.Mode.SetExcluded()).
		Set(cols.LastUsedAt.SetExcluded()).
		Set(cols.UnenrolledAt.SetExcluded()).
		Set(cols.UpdatedAt.SetExcluded()).
		Set(cols.Version.IncConflict(1))
}

func (r *enrollmentRepository) Upsert(
	ctx context.Context,
	entities []*carrierintel.CarrierMonitoringEnrollment,
) error {
	if len(entities) == 0 {
		return nil
	}

	rows := entities
	if _, err := buildEnrollmentUpsert(r.db.DBForContext(ctx), &rows).Exec(ctx); err != nil {
		r.l.Error("failed to upsert carrier monitoring enrollments", zap.Error(err))
		return fmt.Errorf("upsert carrier monitoring enrollments: %w", err)
	}

	return nil
}

func (r *enrollmentRepository) SaveSyncState(
	ctx context.Context,
	entities []*carrierintel.CarrierMonitoringEnrollment,
) error {
	if len(entities) == 0 {
		return nil
	}

	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		dba := r.db.DBForContext(c)
		for _, entity := range entities {
			if entity == nil {
				continue
			}
			if _, uErr := dba.NewUpdate().
				Model(entity).
				Column(
					cols.VendorState.Bare(),
					cols.ProviderRef.Bare(),
					cols.OwnedByTrenova.Bare(),
					cols.EnrolledAt.Bare(),
					cols.UnenrolledAt.Bare(),
					cols.LastSyncedAt.Bare(),
					cols.FailureCount.Bare(),
					cols.LastError.Bare(),
					cols.UpdatedAt.Bare(),
				).
				WherePK().
				Exec(c); uErr != nil {
				return fmt.Errorf("save enrollment %s sync state: %w", entity.ID, uErr)
			}
		}
		return nil
	})
	if err != nil {
		r.l.Error("failed to save carrier monitoring enrollment sync state", zap.Error(err))
		return dberror.MapRetryableTransactionError(
			err,
			"Carrier monitoring enrollment is busy. Retry the request.",
		)
	}

	return nil
}

func (r *enrollmentRepository) ListNeedingSync(
	ctx context.Context,
	req *repositories.ListEnrollmentsNeedingSyncRequest,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultNeedingSyncLimit),
		1,
		maxNeedingSyncLimit,
	)

	entities := make([]*carrierintel.CarrierMonitoringEnrollment, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierMonitoringEnrollmentScopeTenant(sq, req.TenantInfo).
				Where(cols.Provider.Eq(), req.Provider).
				WhereGroup(" AND ", func(outer *bun.SelectQuery) *bun.SelectQuery {
					return outer.
						WhereGroup(" OR ", func(enroll *bun.SelectQuery) *bun.SelectQuery {
							return enroll.
								Where(cols.DesiredState.Eq(), carrierintel.DesiredStateEnrolled).
								Where(cols.VendorState.In(), bun.List(enrollPendingVendorStates))
						}).
						WhereGroup(" OR ", func(unenroll *bun.SelectQuery) *bun.SelectQuery {
							return unenroll.
								Where(cols.DesiredState.Eq(), carrierintel.DesiredStateNotEnrolled).
								Where(cols.VendorState.In(), bun.List(unenrollPendingVendorStates))
						})
				})
		}).
		Order(cols.UpdatedAt.OrderAsc(), cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier monitoring enrollments needing sync", zap.Error(err))
		return nil, fmt.Errorf("list carrier monitoring enrollments needing sync: %w", err)
	}

	return entities, nil
}

func (r *enrollmentRepository) ListActive(
	ctx context.Context,
	req *repositories.ListActiveEnrollmentsRequest,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultActiveLimit),
		1,
		maxActiveLimit,
	)

	entities := make([]*carrierintel.CarrierMonitoringEnrollment, 0, limit)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierMonitoringEnrollmentScopeTenant(sq, req.TenantInfo).
				Where(cols.Provider.Eq(), req.Provider).
				Where(cols.DesiredState.Eq(), carrierintel.DesiredStateEnrolled).
				Where(cols.VendorState.Eq(), carrierintel.VendorStateActive)
		}).
		Order(cols.ID.OrderAsc()).
		Limit(limit)
	if req.Mode != "" {
		q = q.Where(cols.Mode.Eq(), req.Mode)
	}
	if req.ConfirmedBefore != nil {
		q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.LastConfirmedAt.IsNull()).
				WhereOr(cols.LastConfirmedAt.Lt(), *req.ConfirmedBefore)
		})
	}
	if !req.AfterID.IsNil() {
		q = q.Where(cols.ID.Gt(), req.AfterID)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list active carrier monitoring enrollments", zap.Error(err))
		return nil, fmt.Errorf("list active carrier monitoring enrollments: %w", err)
	}

	return entities, nil
}

func (r *enrollmentRepository) ListByProviderRefs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
	refs []string,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	if len(refs) == 0 {
		return []*carrierintel.CarrierMonitoringEnrollment{}, nil
	}

	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	entities := make([]*carrierintel.CarrierMonitoringEnrollment, 0, len(refs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierMonitoringEnrollmentScopeTenant(sq, tenantInfo).
				Where(cols.Provider.Eq(), provider).
				Where(cols.ProviderRef.In(), bun.List(refs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier monitoring enrollments by provider ref", zap.Error(err))
		return nil, fmt.Errorf("list carrier monitoring enrollments by provider ref: %w", err)
	}

	return entities, nil
}

func (r *enrollmentRepository) applyConnectionFilters(
	q *bun.SelectQuery,
	req *repositories.ListEnrollmentConnectionRequest,
) *bun.SelectQuery {
	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	if req.Provider != "" {
		q = q.Where(cols.Provider.Eq(), req.Provider)
	}
	if req.DesiredState != "" {
		q = q.Where(cols.DesiredState.Eq(), req.DesiredState)
	}
	if len(req.VendorStates) > 0 {
		q = q.Where(cols.VendorState.In(), bun.List(req.VendorStates))
	}

	return q
}

func (r *enrollmentRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListEnrollmentConnectionRequest,
) (*pagination.CursorListResult[*carrierintel.CarrierMonitoringEnrollment], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.
			NewSelect().
			Model((*carrierintel.CarrierMonitoringEnrollment)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.CarrierMonitoringEnrollmentTable.Alias,
					req.Filter,
					(*carrierintel.CarrierMonitoringEnrollment)(nil),
				)
				return r.applyConnectionFilters(sq, req)
			}).
			Count(ctx)
		if err != nil {
			log.Error("failed to count carrier monitoring enrollments", zap.Error(err))
			return nil, err
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*carrierintel.CarrierMonitoringEnrollment]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(items *[]*carrierintel.CarrierMonitoringEnrollment) *bun.SelectQuery {
				return dba.NewSelect().
					Model(items).
					ColumnExpr(buncolgen.CarrierMonitoringEnrollmentTable.All())
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				sq, applyErr := querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.CarrierMonitoringEnrollmentTable.Alias,
					req.Filter,
					req.Cursor,
					(*carrierintel.CarrierMonitoringEnrollment)(nil),
				)
				if applyErr != nil {
					return sq, applyErr
				}
				return r.applyConnectionFilters(sq, req), nil
			},
		},
	)
	if err != nil {
		log.Error("failed to list carrier monitoring enrollments", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *enrollmentRepository) TouchConfirmed(
	ctx context.Context,
	req *repositories.TouchEnrollmentsConfirmedRequest,
) error {
	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carrierintel.CarrierMonitoringEnrollment)(nil)).
		Set(cols.LastConfirmedAt.Set(), req.ConfirmedAt).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = buncolgen.CarrierMonitoringEnrollmentScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.Provider.Eq(), req.Provider).
				Where(cols.Mode.Eq(), req.Mode).
				Where(cols.DesiredState.Eq(), carrierintel.DesiredStateEnrolled)
			if len(req.SubjectIDs) == 0 {
				return uq.Where(cols.VendorState.Eq(), carrierintel.VendorStateActive)
			}
			return uq.Where(cols.SubjectID.In(), bun.List(req.SubjectIDs))
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to touch carrier monitoring enrollments", zap.Error(err))
		return fmt.Errorf("touch carrier monitoring enrollments: %w", err)
	}

	return nil
}

func (r *enrollmentRepository) Counts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider integration.Type,
) (*repositories.CarrierIntelEnrollmentCounts, error) {
	cols := buncolgen.CarrierMonitoringEnrollmentColumns
	counts := new(repositories.CarrierIntelEnrollmentCounts)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*carrierintel.CarrierMonitoringEnrollment)(nil)).
		ColumnExpr(
			buncolgen.CountFilter("desired_count", cols.DesiredState.Eq()),
			carrierintel.DesiredStateEnrolled,
		).
		ColumnExpr(
			buncolgen.CountFilter("active_count", cols.DesiredState.Eq(), cols.VendorState.Eq()),
			carrierintel.DesiredStateEnrolled,
			carrierintel.VendorStateActive,
		).
		ColumnExpr(
			buncolgen.CountFilter(
				"pending_count",
				buncolgen.Expr(
					"(({0} = ? AND {1} IN (?)) OR ({0} = ? AND {1} IN (?)))",
					cols.DesiredState,
					cols.VendorState,
				),
			),
			carrierintel.DesiredStateEnrolled,
			bun.List(enrollPendingVendorStates),
			carrierintel.DesiredStateNotEnrolled,
			bun.List(unenrollPendingVendorStates),
		).
		ColumnExpr(
			buncolgen.CountFilter("failed_count", cols.VendorState.Eq()),
			carrierintel.VendorStateFailed,
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierMonitoringEnrollmentScopeTenant(sq, tenantInfo).
				Where(cols.Provider.Eq(), provider)
		}).
		Scan(ctx, &counts.Desired, &counts.Active, &counts.Pending, &counts.Failed)
	if err != nil {
		r.l.Error("failed to count carrier monitoring enrollments", zap.Error(err))
		return nil, fmt.Errorf("count carrier monitoring enrollments: %w", err)
	}

	return counts, nil
}
