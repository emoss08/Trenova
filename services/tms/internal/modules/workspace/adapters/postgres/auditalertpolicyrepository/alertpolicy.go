package auditalertpolicyrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var policyErrors = dberror.Mapping{Entity: "Audit alert policy"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) auditports.AlertPolicyRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.audit-alert-policy-repository")}
}

func (r *repository) ListForTenant(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*audit.AlertPolicy, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.AlertPolicy, error) {
		p := auditdb.AlertPolicies
		return p.Query().Order(p.Kind.Asc()).Limit(len(audit.AlertDefinitions)+1).List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Upsert(
	ctx context.Context,
	policy *audit.AlertPolicy,
) (*audit.AlertPolicy, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*audit.AlertPolicy, error) {
		p := auditdb.AlertPolicies
		if _, err := p.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*audit.AlertPolicy{policy},
			keel.OnConflict(p.OrganizationID, p.BusinessUnitID, p.Kind).Update(
				p.Enabled,
				p.Threshold,
				p.WindowMinutes,
			),
		); err != nil {
			return nil, policyErrors.Map(err, string(policy.Kind))
		}
		return p.Where(p.Kind.Eq(policy.Kind)).One(ctx, r.db.Keel(ctx))
	})
}
