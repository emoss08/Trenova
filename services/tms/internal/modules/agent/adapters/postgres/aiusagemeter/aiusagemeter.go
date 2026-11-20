package aiusagemeter

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage/aiusagedb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

const (
	hour                    = 3600
	organizationsPageReason = "add up the Document Intelligence pages each organization of a shared server extracted in each hour, in one statement, for the control plane's usage report; each organization is counted only for itself"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Result struct {
	fx.Out

	Pages managedusageports.HourlyCounter `group:"managed_hourly_counters"`
}

func New(p Params) Result {
	return Result{Pages: &pages{db: p.DB}}
}

type pages struct {
	db *postgres.Connection
}

func (c *pages) Meter() platformcatalog.MeterKey {
	return platformcatalog.MeterDocumentIntelligencePages
}

func (c *pages) CountByHour(
	ctx context.Context,
	req *managedusageports.HourCountRequest,
) ([]managedusageports.HourCount, error) {
	if tenant, ok := req.Scope.Tenant(); ok {
		ctx = dbscope.WithTenant(ctx, tenant)
	} else {
		ctx = dbscope.WithSystem(
			ctx,
			"add up the pages Document Intelligence extracted in each hour on this managed instance",
		)
	}
	return dbtx.Read(ctx, c.db, func(ctx context.Context) ([]managedusageports.HourCount, error) {
		u := aiusagedb.AIUsageRecords
		return byHour(ctx, c.db.Keel(ctx), &hourly{
			meter: c.Meter(),
			value: keel.CastInt64(keel.NumericOf(u.Pages.Sum()).Coalesce(0)),
			filter: keel.And(
				keel.BoolOf(u.Succeeded),
				u.Feature.Eq(aiusage.FeatureDocumentIntelligenceExtract),
			),
			req: req,
		})
	})
}

func (c *pages) CountByHourForOrganizations(
	ctx context.Context,
	req *managedusageports.OrganizationsHourCountRequest,
) ([]managedusageports.OrganizationHourCount, error) {
	if len(req.OrganizationIDs) == 0 {
		return []managedusageports.OrganizationHourCount{}, nil
	}
	ctx = dbscope.WithSystem(ctx, organizationsPageReason)
	return dbtx.Read(
		ctx,
		c.db,
		func(ctx context.Context) ([]managedusageports.OrganizationHourCount, error) {
			u := aiusagedb.AIUsageRecords
			bucket := keel.Int64Of(u.CreatedAt).Div(keel.Int64(hour)).Mul(keel.Int64(hour))
			counts, err := keel.Into[managedusageports.OrganizationHourCount](
				ctx,
				c.db.Keel(ctx),
				u.Where(
					u.OrganizationID.In(req.OrganizationIDs...),
					u.CreatedAt.Gte(req.Since),
					u.CreatedAt.Lt(req.Until),
					keel.BoolOf(u.Succeeded),
					u.Feature.Eq(aiusage.FeatureDocumentIntelligenceExtract),
				).
					AllTenants().
					Group(u.OrganizationID, bucket).
					Pick(
						u.OrganizationID,
						bucket,
						keel.CastInt64(keel.NumericOf(u.Pages.Sum()).Coalesce(0)),
					),
			)
			if err != nil {
				return nil, fmt.Errorf("count %s by hour and organization: %w", c.Meter(), err)
			}
			return counts, nil
		},
	)
}

type hourly struct {
	meter  platformcatalog.MeterKey
	value  keel.Expr
	filter keel.Pred
	req    *managedusageports.HourCountRequest
}

func byHour(ctx context.Context, db *keel.DB, h *hourly) ([]managedusageports.HourCount, error) {
	u := aiusagedb.AIUsageRecords
	bucket := keel.Int64Of(u.CreatedAt).Div(keel.Int64(hour)).Mul(keel.Int64(hour))
	counts, err := keel.Into[managedusageports.HourCount](
		ctx,
		db,
		tenantscope.Scoped(
			ctx,
			u.Where(u.CreatedAt.Gte(h.req.Since), u.CreatedAt.Lt(h.req.Until), h.filter),
		).
			Group(bucket).
			Pick(bucket, h.value),
	)
	if err != nil {
		return nil, fmt.Errorf("count %s by hour: %w", h.meter, err)
	}
	return counts, nil
}
