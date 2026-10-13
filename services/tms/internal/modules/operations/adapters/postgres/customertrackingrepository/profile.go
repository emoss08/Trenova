package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/buncolgen/customertrackingcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type profileRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProfiles(p Params) customertrackingports.ProfileRepository {
	return &profileRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.customer-tracking-profile-repository"),
	}
}

func (r *profileRepository) GetByCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*customertracking.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customertracking.Profile, error) {
		cols := customertrackingcol.ProfileColumns
		entity := new(customertracking.Profile)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return customertrackingcol.ProfileScopeTenant(sq, tenantInfo).
					Where(cols.CustomerID.Eq(), customerID)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("get customer tracking profile: %w", err)
		}
		return entity, nil
	})
}

func (r *profileRepository) ListByCustomers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerIDs []pulid.ID,
) ([]*customertracking.Profile, error) {
	if len(customerIDs) == 0 {
		return []*customertracking.Profile{}, nil
	}
	if len(customerIDs) > maxCustomerBatch {
		customerIDs = customerIDs[:maxCustomerBatch]
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customertracking.Profile, error) {
		cols := customertrackingcol.ProfileColumns
		entities := make([]*customertracking.Profile, 0, len(customerIDs))
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return customertrackingcol.ProfileScopeTenant(sq, tenantInfo).
					Where(cols.CustomerID.In(), bun.List(customerIDs))
			}).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list customer tracking profiles: %w", err)
		}
		return entities, nil
	})
}

func (r *profileRepository) Create(ctx context.Context, entity *customertracking.Profile) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Exec(ctx); err != nil {
			return fmt.Errorf("insert customer tracking profile: %w", err)
		}
		return nil
	})
}

func (r *profileRepository) Update(ctx context.Context, entity *customertracking.Profile) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := customertrackingcol.ProfileColumns
		previous := entity.Version
		entity.Version++
		result, err := r.db.DBForContext(ctx).NewUpdate().
			Model(entity).
			Column(
				cols.LinkEnabled.Bare(),
				cols.VisibleFields.Bare(),
				cols.LinkTTLDays.Bare(),
				cols.NotifyPickedUp.Bare(),
				cols.NotifyDelayed.Bare(),
				cols.NotifyDelivered.Bare(),
				cols.NotifyPodReady.Bare(),
				cols.IncludeLink.Bare(),
				cols.Recipients.Bare(),
				cols.NotificationsEnabledAt.Bare(),
				cols.Version.Bare(),
				cols.UpdatedAt.Bare(),
			).
			WherePK().
			Where(cols.Version.Eq(), previous).
			Exec(ctx)
		if err != nil {
			entity.Version = previous
			return fmt.Errorf("update customer tracking profile: %w", err)
		}
		if err = dberror.CheckRowsAffected(
			result,
			"Customer tracking settings",
			entity.ID.String(),
		); err != nil {
			entity.Version = previous
			return err
		}
		return nil
	})
}
