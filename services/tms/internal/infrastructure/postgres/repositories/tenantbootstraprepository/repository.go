package tenantbootstraprepository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	bootstrapScopeReason = "create a new tenant's business unit, organization, owner and defaults before the tenant exists"
	lockScopeReason      = "serialize tenant provisioning so instance-wide signup caps hold under concurrency"
	provisioningLockKey  = "trenova:tenant-provisioning"
)

var (
	ErrOrganizationRequired = errors.New("tenant bootstrap: organization is required")
	ErrOwnerRequired        = errors.New("tenant bootstrap: owner is required")
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.TenantBootstrapRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tenant-bootstrap-repository"),
	}
}

func (r *repository) LockProvisioning(ctx context.Context) error {
	ctx = dbscope.WithSystem(ctx, lockScopeReason)

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).
			NewRaw("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", provisioningLockKey).
			Exec(ctx); err != nil {
			return fmt.Errorf("lock tenant provisioning: %w", err)
		}

		return nil
	})
}

func (r *repository) Bootstrap(
	ctx context.Context,
	req *repositories.BootstrapTenantRequest,
) (*repositories.BootstrapTenantResult, error) {
	if req.Organization == nil {
		return nil, ErrOrganizationRequired
	}
	if req.Owner == nil {
		return nil, ErrOwnerRequired
	}

	ctx = dbscope.WithSystem(ctx, bootstrapScopeReason)

	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*repositories.BootstrapTenantResult, error) {
		db := r.db.DBForContext(ctx)

		if req.Organization.StateID.IsNil() {
			stateID, err := r.resolveState(ctx, req.StateAbbreviation)
			if err != nil {
				return nil, err
			}
			req.Organization.StateID = stateID
		}

		code, err := tenantbootstrap.AvailableBusinessUnitCode(ctx, db)
		if err != nil {
			return nil, err
		}

		slug, err := tenantbootstrap.AvailableLoginSlug(ctx, db, req.LoginSlugBase)
		if err != nil {
			return nil, err
		}
		req.Organization.LoginSlug = slug
		req.Organization.BucketName = tenantbootstrap.BucketNameFor(slug)

		username, err := tenantbootstrap.AvailableUsername(ctx, db, req.UsernameBase)
		if err != nil {
			return nil, err
		}
		req.Owner.Username = username

		result, err := tenantbootstrap.Bootstrap(ctx, db, tenantbootstrap.BootstrapParams{
			BusinessUnit: &tenant.BusinessUnit{Name: req.BusinessUnitName, Code: code},
			Organization: req.Organization,
			Owner:        req.Owner,
			Now:          req.Now,
		})
		if err != nil {
			r.l.Error("failed to bootstrap tenant", zap.Error(err))
			return nil, err
		}

		return &repositories.BootstrapTenantResult{
			BusinessUnit: result.BusinessUnit,
			Organization: result.Organization,
			Owner:        result.Owner,
			AdminRoleID:  result.AdminRole.ID,
		}, nil
	})
}

func (r *repository) resolveState(ctx context.Context, abbreviation string) (pulid.ID, error) {
	cols := buncolgen.UsStateColumns
	state := new(usstate.UsState)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(state).
		Column(cols.ID.Bare()).
		Where("upper("+cols.Abbreviation.Qualified()+") = ?", strings.ToUpper(strings.TrimSpace(abbreviation))).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve placeholder state %q: %w", abbreviation, dberror.HandleNotFoundError(err, "State"))
	}

	return state.ID, nil
}
