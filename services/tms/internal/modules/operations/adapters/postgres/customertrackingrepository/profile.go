package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking/customertrackingdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var profileErrors = dberror.Mapping{
	Entity: "Customer tracking settings",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_customer_tracking_profiles_customer",
			Field:   "customerId",
			Message: "Customer already has tracking settings",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_customer_tracking_profiles_customer",
			Field:   "customerId",
			Message: "Customer does not exist in your organization",
		},
	},
}

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
	_ pagination.TenantInfo,
	customerID pulid.ID,
) (*customertracking.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customertracking.Profile, error) {
		p := customertrackingdb.Profiles
		entity, err := p.Where(p.CustomerID.Eq(customerID)).One(ctx, r.db.Keel(ctx))
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
	_ pagination.TenantInfo,
	customerIDs []pulid.ID,
) ([]*customertracking.Profile, error) {
	if len(customerIDs) == 0 {
		return []*customertracking.Profile{}, nil
	}
	if len(customerIDs) > maxCustomerBatch {
		customerIDs = customerIDs[:maxCustomerBatch]
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*customertracking.Profile, error) {
		p := customertrackingdb.Profiles
		entities, err := p.Where(p.CustomerID.In(customerIDs...)).
			Limit(len(customerIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list customer tracking profiles: %w", err)
		}
		return entities, nil
	})
}

func (r *profileRepository) Create(ctx context.Context, entity *customertracking.Profile) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := customertrackingdb.Profiles.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return fmt.Errorf(
				"insert customer tracking profile: %w",
				profileErrors.Map(err, entity.ID.String()),
			)
		}
		return nil
	})
}

func (r *profileRepository) Update(ctx context.Context, entity *customertracking.Profile) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := customertrackingdb.Profiles
		if err := p.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			p.LinkEnabled,
			p.VisibleFields,
			p.LinkTTLDays,
			p.NotifyPickedUp,
			p.NotifyDelayed,
			p.NotifyDelivered,
			p.NotifyPodReady,
			p.IncludeLink,
			p.Recipients,
			p.NotificationsEnabledAt,
			p.UpdatedAt,
		); err != nil {
			return profileErrors.Map(err, entity.ID.String())
		}
		return nil
	})
}
