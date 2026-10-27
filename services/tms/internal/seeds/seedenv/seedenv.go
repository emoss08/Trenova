package seedenv

import (
	"context"
	"iter"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/dbscope"
)

const (
	systemReason = "apply seeds and their bookkeeping across every organization"
	tenantsLimit = 1_000_000
)

type configKey struct{}

func WithConfig(ctx context.Context, cfg *config.Config) context.Context {
	return context.WithValue(ctx, configKey{}, cfg)
}

func Config(ctx context.Context) *config.Config {
	cfg, _ := ctx.Value(configKey{}).(*config.Config)
	if cfg == nil {
		return &config.Config{}
	}

	return cfg
}

type Env struct {
	Config func() *config.Config
}

func (e Env) System(ctx context.Context) context.Context {
	return dbscope.WithSystem(e.withConfig(ctx), systemReason)
}

func (e Env) Tenants(ctx context.Context, db *keel.DB) iter.Seq2[seed.Tenant, error] {
	return func(yield func(seed.Tenant, error) bool) {
		o := tenantdb.Organizations
		orgs, err := o.AllTenants().
			Pick(o.ID, o.BusinessUnitID, o.Name).
			Order(o.CreatedAt.Asc(), o.ID.Asc()).
			Limit(tenantsLimit).
			List(e.System(ctx), db)
		if err != nil {
			yield(seed.Tenant{}, err)
			return
		}

		for _, org := range orgs {
			if !yield(e.tenant(org), nil) {
				return
			}
		}
	}
}

func (e Env) tenant(org *tenant.Organization) seed.Tenant {
	scope := dbscope.Tenant{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID}

	return seed.Tenant{
		Key:  org.ID.String(),
		Name: org.Name,
		Context: func(ctx context.Context) context.Context {
			return dbscope.WithTenant(e.withConfig(ctx), scope)
		},
	}
}

func (e Env) withConfig(ctx context.Context) context.Context {
	if e.Config == nil {
		return ctx
	}

	return WithConfig(ctx, e.Config())
}

func TenantOf(ctx context.Context) dbscope.Tenant {
	t, _ := dbscope.TenantFrom(ctx)
	return t
}
