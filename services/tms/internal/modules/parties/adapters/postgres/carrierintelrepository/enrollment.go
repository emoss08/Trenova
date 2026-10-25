package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	defaultNeedingSyncLimit = 500
	maxNeedingSyncLimit     = 5000
	defaultActiveLimit      = 1000
	maxActiveLimit          = 5000
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

func NewEnrollmentRepository(p Params) carrierintelports.CarrierMonitoringEnrollmentRepository {
	return &enrollmentRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-monitoring-enrollment-repository"),
	}
}

func (r *enrollmentRepository) ListBySubjects(
	ctx context.Context,
	req *carrierintelports.ListEnrollmentsBySubjectRequest,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
			if len(req.SubjectIDs) == 0 {
				return []*carrierintel.CarrierMonitoringEnrollment{}, nil
			}

			en := carrierinteldb.CarrierMonitoringEnrollments
			entities, err := en.Where(
				en.Provider.Eq(req.Provider),
				en.SubjectType.Eq(req.SubjectType),
				en.SubjectID.In(req.SubjectIDs...),
			).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to list carrier monitoring enrollments by subject",
					zap.Error(err),
				)
				return nil, fmt.Errorf("list carrier monitoring enrollments by subject: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *enrollmentRepository) GetByCarrierIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
			if len(carrierIDs) == 0 {
				return []*carrierintel.CarrierMonitoringEnrollment{}, nil
			}

			en := carrierinteldb.CarrierMonitoringEnrollments
			entities, err := en.Where(en.CarrierID.In(carrierIDs...)).
				Order(
					keel.Desc(keel.Raw("? = ?", en.DesiredState, string(carrierintel.DesiredStateEnrolled))),
					en.UpdatedAt.Desc(),
				).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get carrier monitoring enrollments by carrier", zap.Error(err))
				return nil, fmt.Errorf("get carrier monitoring enrollments by carrier: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *enrollmentRepository) ListDesired(
	ctx context.Context,
	_ pagination.TenantInfo,
	provider integration.Type,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
			en := carrierinteldb.CarrierMonitoringEnrollments
			entities, err := en.Where(
				en.Provider.Eq(provider),
				en.DesiredState.Eq(carrierintel.DesiredStateEnrolled),
			).
				Order(en.ID.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list desired carrier monitoring enrollments", zap.Error(err))
				return nil, fmt.Errorf("list desired carrier monitoring enrollments: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *enrollmentRepository) Upsert(
	ctx context.Context,
	entities []*carrierintel.CarrierMonitoringEnrollment,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entities) == 0 {
			return nil
		}

		en := carrierinteldb.CarrierMonitoringEnrollments
		_, err := en.Upsert(
			ctx,
			r.db.Keel(ctx),
			entities,
			keel.OnConflict(
				en.OrganizationID,
				en.BusinessUnitID,
				en.Provider,
				en.SubjectType,
				en.SubjectID,
			).Update(
				en.DesiredState,
				en.VendorState,
				en.Reason,
				en.SubjectName,
				en.DOTNumber,
				en.DocketNumber,
				en.CarrierID,
				en.Mode,
				en.LastUsedAt,
				en.UnenrolledAt,
			),
		)
		if err != nil {
			r.l.Error("failed to upsert carrier monitoring enrollments", zap.Error(err))
			return fmt.Errorf("upsert carrier monitoring enrollments: %w", err)
		}

		return nil
	})
}

func (r *enrollmentRepository) SaveSyncState(
	ctx context.Context,
	entities []*carrierintel.CarrierMonitoringEnrollment,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entities) == 0 {
			return nil
		}

		en := carrierinteldb.CarrierMonitoringEnrollments
		db := r.db.Keel(ctx)
		now := timeutils.NowUnix()
		for _, entity := range entities {
			if entity == nil {
				continue
			}
			entity.UpdatedAt = now
			if _, err := en.Where(en.ID.Eq(entity.ID)).Update(
				ctx,
				db,
				en.VendorState.To(entity.VendorState),
				en.ProviderRef.To(entity.ProviderRef),
				en.OwnedByTrenova.To(entity.OwnedByTrenova),
				en.EnrolledAt.ToExpr(keel.Val(entity.EnrolledAt)),
				en.UnenrolledAt.ToExpr(keel.Val(entity.UnenrolledAt)),
				en.LastSyncedAt.ToExpr(keel.Val(entity.LastSyncedAt)),
				en.FailureCount.To(int64(entity.FailureCount)),
				en.LastError.To(entity.LastError),
				en.UpdatedAt.To(now),
				en.Version.ToExpr(en.Version),
			); err != nil {
				r.l.Error("failed to save carrier monitoring enrollment sync state", zap.Error(err))
				return dberror.MapRetryableTransactionError(
					fmt.Errorf("save enrollment %s sync state: %w", entity.ID, err),
					"Carrier monitoring enrollment is busy. Retry the request.",
				)
			}
		}

		return nil
	})
}

func (r *enrollmentRepository) ListNeedingSync(
	ctx context.Context,
	req *carrierintelports.ListEnrollmentsNeedingSyncRequest,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
			en := carrierinteldb.CarrierMonitoringEnrollments
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultNeedingSyncLimit),
				1,
				maxNeedingSyncLimit,
			)

			entities, err := en.Where(en.Provider.Eq(req.Provider), pendingSyncPredicate()).
				Order(en.UpdatedAt.Asc(), en.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to list carrier monitoring enrollments needing sync",
					zap.Error(err),
				)
				return nil, fmt.Errorf("list carrier monitoring enrollments needing sync: %w", err)
			}

			return entities, nil
		},
	)
}

func pendingSyncPredicate() keel.Pred {
	en := carrierinteldb.CarrierMonitoringEnrollments

	return keel.Or(
		keel.And(
			en.DesiredState.Eq(carrierintel.DesiredStateEnrolled),
			en.VendorState.In(enrollPendingVendorStates...),
		),
		keel.And(
			en.DesiredState.Eq(carrierintel.DesiredStateNotEnrolled),
			en.VendorState.In(unenrollPendingVendorStates...),
		),
	)
}

func (r *enrollmentRepository) ListActive(
	ctx context.Context,
	req *carrierintelports.ListActiveEnrollmentsRequest,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
			en := carrierinteldb.CarrierMonitoringEnrollments
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultActiveLimit),
				1,
				maxActiveLimit,
			)

			var confirmedBefore keel.Pred
			if req.ConfirmedBefore != nil {
				confirmedBefore = keel.Or(
					en.LastConfirmedAt.IsNull(),
					en.LastConfirmedAt.Lt(*req.ConfirmedBefore),
				)
			}

			entities, err := en.Where(
				en.Provider.Eq(req.Provider),
				en.DesiredState.Eq(carrierintel.DesiredStateEnrolled),
				en.VendorState.Eq(carrierintel.VendorStateActive),
				keel.When(req.Mode != "", en.Mode.Eq(req.Mode)),
				confirmedBefore,
				keel.When(!req.AfterID.IsNil(), en.ID.Gt(req.AfterID)),
			).
				Order(en.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list active carrier monitoring enrollments", zap.Error(err))
				return nil, fmt.Errorf("list active carrier monitoring enrollments: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *enrollmentRepository) ListByProviderRefs(
	ctx context.Context,
	_ pagination.TenantInfo,
	provider integration.Type,
	refs []string,
) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierMonitoringEnrollment, error) {
			if len(refs) == 0 {
				return []*carrierintel.CarrierMonitoringEnrollment{}, nil
			}

			en := carrierinteldb.CarrierMonitoringEnrollments
			entities, err := en.Where(en.Provider.Eq(provider), en.ProviderRef.In(refs...)).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to list carrier monitoring enrollments by provider ref",
					zap.Error(err),
				)
				return nil, fmt.Errorf(
					"list carrier monitoring enrollments by provider ref: %w",
					err,
				)
			}

			return entities, nil
		},
	)
}

func (r *enrollmentRepository) ListConnection(
	ctx context.Context,
	req *carrierintelports.ListEnrollmentConnectionRequest,
) (*pagination.CursorListResult[*carrierintel.CarrierMonitoringEnrollment], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*carrierintel.CarrierMonitoringEnrollment], error) {
			en := carrierinteldb.CarrierMonitoringEnrollments
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				en.Where(
					keel.When(req.Provider != "", en.Provider.Eq(req.Provider)),
					keel.When(req.DesiredState != "", en.DesiredState.Eq(req.DesiredState)),
					keel.When(len(req.VendorStates) > 0, en.VendorState.In(req.VendorStates...)),
				),
				&listquery.Request[carrierintel.CarrierMonitoringEnrollment]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier monitoring enrollments", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *enrollmentRepository) TouchConfirmed(
	ctx context.Context,
	req *carrierintelports.TouchEnrollmentsConfirmedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		en := carrierinteldb.CarrierMonitoringEnrollments
		scope := en.VendorState.Eq(carrierintel.VendorStateActive)
		if len(req.SubjectIDs) > 0 {
			scope = en.SubjectID.In(req.SubjectIDs...)
		}

		_, err := en.Where(
			en.Provider.Eq(req.Provider),
			en.Mode.Eq(req.Mode),
			en.DesiredState.Eq(carrierintel.DesiredStateEnrolled),
			scope,
		).Update(
			ctx,
			r.db.Keel(ctx),
			en.LastConfirmedAt.To(req.ConfirmedAt),
			en.Version.ToExpr(en.Version),
			en.UpdatedAt.ToExpr(en.UpdatedAt),
		)
		if err != nil {
			r.l.Error("failed to touch carrier monitoring enrollments", zap.Error(err))
			return fmt.Errorf("touch carrier monitoring enrollments: %w", err)
		}

		return nil
	})
}

func (r *enrollmentRepository) Counts(
	ctx context.Context,
	_ pagination.TenantInfo,
	provider integration.Type,
) (*carrierintelports.CarrierIntelEnrollmentCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelports.CarrierIntelEnrollmentCounts, error) {
			en := carrierinteldb.CarrierMonitoringEnrollments
			enrolled := en.DesiredState.Eq(carrierintel.DesiredStateEnrolled)
			counts, err := keel.IntoOne[carrierintelports.CarrierIntelEnrollmentCounts](
				ctx,
				r.db.Keel(ctx),
				en.Where(en.Provider.Eq(provider)).Pick(
					keel.Count().Filter(enrolled),
					keel.Count().Filter(keel.And(
						enrolled,
						en.VendorState.Eq(carrierintel.VendorStateActive),
					)),
					keel.Count().Filter(pendingSyncPredicate()),
					keel.Count().Filter(en.VendorState.Eq(carrierintel.VendorStateFailed)),
				),
			)
			if err != nil {
				r.l.Error("failed to count carrier monitoring enrollments", zap.Error(err))
				return nil, fmt.Errorf("count carrier monitoring enrollments: %w", err)
			}

			return &counts, nil
		},
	)
}
