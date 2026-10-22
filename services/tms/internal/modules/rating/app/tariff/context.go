package tariff

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"go.uber.org/fx"
)

var _ tariffports.ContextSource = (*Contexts)(nil)

type ContextsParams struct {
	fx.In

	Tables   *Tables
	Codes    *Codes
	Settings *Settings
}

type Contexts struct {
	tables   *Tables
	codes    *Codes
	settings *Settings
}

func NewContexts(p ContextsParams) *Contexts {
	return &Contexts{tables: p.Tables, codes: p.Codes, settings: p.Settings}
}

func (c *Contexts) Context(
	ctx context.Context,
	req *tariffports.ContextRequest,
) (*rules.TariffContext, error) {
	settings, err := c.settings.For(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	tables, err := c.tables.Shapes(ctx, req.TenantInfo, settings.DistanceUnit)
	if err != nil {
		return nil, err
	}
	catalog, err := c.codes.Catalog(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	codes := ChargeCodes(catalog)
	return &rules.TariffContext{
		Libraries: req.Libraries,
		Tables:    tables,
		Codes:     &codes,
		Settings:  settings,
	}, nil
}

func (c *Contexts) LoadCells(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	needs []rules.TariffNeed,
) error {
	return c.tables.LoadCells(ctx, tenant, tariffCtx, needs)
}

func (c *Contexts) LoadTables(
	ctx context.Context,
	tenant pagination.TenantInfo,
	tariffCtx *rules.TariffContext,
	tables []string,
) error {
	return c.tables.LoadWhole(ctx, tenant, tariffCtx, tables)
}
