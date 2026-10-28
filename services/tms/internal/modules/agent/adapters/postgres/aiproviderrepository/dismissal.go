package aiproviderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"go.uber.org/fx"
)

type DismissalParams struct {
	fx.In

	DB *postgres.Connection
}

type dismissalRepository struct {
	db *postgres.Connection
}

func NewDismissalRepository(
	p DismissalParams,
) aiproviderports.AIProviderFailureDismissalRepository {
	return &dismissalRepository{db: p.DB}
}

func (r *dismissalRepository) List(
	ctx context.Context,
	req *aiproviderports.ListFailureDismissalsRequest,
) ([]*aiprovider.FailureDismissal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiprovider.FailureDismissal, error) {
		d := aiproviderdb.FailureDismissals
		dismissals, err := d.Where(d.UserID.Eq(req.UserID)).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list provider failure dismissals: %w", err)
		}
		return dismissals, nil
	})
}

func (r *dismissalRepository) Upsert(
	ctx context.Context,
	dismissal *aiprovider.FailureDismissal,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := aiproviderdb.FailureDismissals
		_, err := d.Upsert(ctx, r.db.Keel(ctx), []*aiprovider.FailureDismissal{dismissal},
			keel.OnConflict(d.OrganizationID, d.BusinessUnitID, d.UserID, d.ProviderID).
				Update(d.FailureAt).
				Set(d.CreatedAt.ToExpr(d.CreatedAt.Excluded())),
		)
		if err != nil {
			return fmt.Errorf("dismiss provider failure: %w", err)
		}
		return nil
	})
}

func (r *dismissalRepository) Delete(
	ctx context.Context,
	req *aiproviderports.DeleteFailureDismissalRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := aiproviderdb.FailureDismissals
		_, err := d.Where(d.UserID.Eq(req.UserID), d.ProviderID.Eq(req.ProviderID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("restore provider failure: %w", err)
		}
		return nil
	})
}
