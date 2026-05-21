package tenantprovisioningrepository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) repositories.TenantProvisioningRepository {
	return &repository{db: p.DB}
}

func (r *repository) UpsertProvisioningSnapshot(
	ctx context.Context,
	req *tenant.ProvisioningRequest,
) (*tenant.ProvisioningResult, error) {
	result := &tenant.ProvisioningResult{
		BusinessUnitID: req.Customer.ID,
		OrganizationID: req.Workspace.ID,
	}

	err := r.db.WithTx(ctx, ports.TxOptions{
		Isolation:   sql.LevelReadCommitted,
		LockTimeout: 5 * time.Second,
	}, func(txCtx context.Context, _ bun.Tx) error {
		buUpserted, err := r.upsertBusinessUnit(txCtx, req.Customer)
		if err != nil {
			return err
		}
		orgUpserted, err := r.upsertOrganization(txCtx, req.Workspace)
		if err != nil {
			return err
		}

		result.BusinessUnitsUpserted = buUpserted
		result.OrganizationsUpserted = orgUpserted
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repository) upsertBusinessUnit(
	ctx context.Context,
	customer tenant.ProvisioningCustomer,
) (int, error) {
	now := timeutils.NowUnix()
	row := &businessUnitRow{
		ID:        customer.ID,
		Name:      strings.TrimSpace(customer.Name),
		Code:      strings.ToUpper(strings.TrimSpace(customer.Code)),
		Metadata:  normalizeMetadata(customer.Metadata),
		CreatedAt: now,
		UpdatedAt: now,
	}

	res, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(row).
		On("CONFLICT (id) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("code = EXCLUDED.code").
		Set("metadata = COALESCE(EXCLUDED.metadata, '{}'::jsonb)").
		Set("version = business_units.version + 1").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("upsert provisioned business unit: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count upserted business units: %w", err)
	}

	return int(rows), nil
}

func (r *repository) upsertOrganization(
	ctx context.Context,
	workspace tenant.ProvisioningWorkspace,
) (int, error) {
	stateID, err := r.resolveStateID(ctx, workspace)
	if err != nil {
		return 0, err
	}

	now := timeutils.NowUnix()
	row := &organizationRow{
		ID:             workspace.ID,
		StateID:        stateID,
		BusinessUnitID: workspace.BusinessUnitID,
		Name:           strings.TrimSpace(workspace.Name),
		LoginSlug:      strings.TrimSpace(workspace.LoginSlug),
		ScacCode:       strings.ToUpper(strings.TrimSpace(workspace.ScacCode)),
		DOTNumber:      strings.TrimSpace(workspace.DOTNumber),
		BucketName:     strings.TrimSpace(workspace.BucketName),
		AddressLine1:   strings.TrimSpace(workspace.AddressLine1),
		AddressLine2:   strings.TrimSpace(workspace.AddressLine2),
		City:           strings.TrimSpace(workspace.City),
		PostalCode:     strings.TrimSpace(workspace.PostalCode),
		Timezone:       strings.TrimSpace(workspace.Timezone),
		TaxID:          strings.TrimSpace(workspace.TaxID),
		Metadata:       normalizeMetadata(workspace.Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	res, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(row).
		On("CONFLICT (id) DO UPDATE").
		Set("state_id = EXCLUDED.state_id").
		Set("business_unit_id = EXCLUDED.business_unit_id").
		Set("name = EXCLUDED.name").
		Set("login_slug = EXCLUDED.login_slug").
		Set("scac_code = EXCLUDED.scac_code").
		Set("dot_number = EXCLUDED.dot_number").
		Set("bucket_name = EXCLUDED.bucket_name").
		Set("address_line1 = EXCLUDED.address_line1").
		Set("address_line2 = EXCLUDED.address_line2").
		Set("city = EXCLUDED.city").
		Set("postal_code = EXCLUDED.postal_code").
		Set("timezone = EXCLUDED.timezone").
		Set("tax_id = EXCLUDED.tax_id").
		Set("metadata = COALESCE(EXCLUDED.metadata, '{}'::jsonb)").
		Set("version = organizations.version + 1").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("upsert provisioned organization: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count upserted organizations: %w", err)
	}

	return int(rows), nil
}

func (r *repository) resolveStateID(
	ctx context.Context,
	workspace tenant.ProvisioningWorkspace,
) (pulid.ID, error) {
	if workspace.StateID.IsNotNil() {
		var exists bool
		if err := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("us_states").
			ColumnExpr("true").
			Where("id = ?", workspace.StateID).
			Scan(ctx, &exists); err != nil {
			return pulid.Nil, errortypes.NewValidationError(
				"workspace.stateId",
				errortypes.ErrInvalid,
				"Workspace state ID is not recognized",
			)
		}

		return workspace.StateID, nil
	}

	abbreviation := strings.ToUpper(strings.TrimSpace(workspace.State))
	var stateID pulid.ID
	if err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr("us_states").
		ColumnExpr("id").
		Where("upper(abbreviation) = ?", abbreviation).
		Scan(ctx, &stateID); err != nil {
		return pulid.Nil, errortypes.NewValidationError(
			"workspace.state",
			errortypes.ErrInvalid,
			"Workspace state is not recognized",
		)
	}

	return stateID, nil
}

func normalizeMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	return metadata
}

type businessUnitRow struct {
	bun.BaseModel `bun:"table:business_units"`

	ID        pulid.ID       `bun:"id,pk"`
	Name      string         `bun:"name"`
	Code      string         `bun:"code"`
	Metadata  map[string]any `bun:"metadata"`
	CreatedAt int64          `bun:"created_at"`
	UpdatedAt int64          `bun:"updated_at"`
}

type organizationRow struct {
	bun.BaseModel `bun:"table:organizations"`

	ID             pulid.ID       `bun:"id,pk"`
	StateID        pulid.ID       `bun:"state_id"`
	BusinessUnitID pulid.ID       `bun:"business_unit_id"`
	Name           string         `bun:"name"`
	LoginSlug      string         `bun:"login_slug"`
	ScacCode       string         `bun:"scac_code"`
	DOTNumber      string         `bun:"dot_number"`
	BucketName     string         `bun:"bucket_name"`
	AddressLine1   string         `bun:"address_line1"`
	AddressLine2   string         `bun:"address_line2"`
	City           string         `bun:"city"`
	PostalCode     string         `bun:"postal_code"`
	Timezone       string         `bun:"timezone"`
	TaxID          string         `bun:"tax_id"`
	Metadata       map[string]any `bun:"metadata"`
	CreatedAt      int64          `bun:"created_at"`
	UpdatedAt      int64          `bun:"updated_at"`
}
