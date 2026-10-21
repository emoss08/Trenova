package enveloperepository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

const (
	listTenantsScopeReason = "list every organization to rewrap its sealed values to the active key"

	idField     = "envelope_row_id"
	orgField    = "envelope_row_org"
	buField     = "envelope_row_bu"
	sealedField = "envelope_row_sealed"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) kernelports.EnvelopeRepository {
	return &repository{db: p.DB}
}

type tenantRow struct {
	bun.BaseModel `bun:"table:organizations,alias:org"`

	ID             pulid.ID `bun:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func (r *repository) ListTenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, listTenantsScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		cols := tenantcol.OrganizationColumns
		rows := make([]tenantRow, 0)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			Column(cols.ID.String(), cols.BusinessUnitID.String()).
			Order(cols.ID.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("list organizations: %w", err)
		}

		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(
				tenants,
				pagination.TenantInfo{OrgID: row.ID, BuID: row.BusinessUnitID},
			)
		}
		return tenants, nil
	})
}

func (r *repository) ListRows(
	ctx context.Context,
	req *kernelports.ListEnvelopeRowsRequest,
) ([]*kernelports.EnvelopeRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*kernelports.EnvelopeRow, error) {
		col := req.Column
		q := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("? AS ?", bun.Ident(col.Table.Name), bun.Ident(col.Table.Alias)).
			ColumnExpr(col.ID.Expr("{}::text")+" AS ?", bun.Ident(idField)).
			ColumnExpr(col.OrganizationID.Expr("{}::text")+" AS ?", bun.Ident(orgField)).
			ColumnExpr(col.Value.Expr("{}::text")+" AS ?", bun.Ident(sealedField))
		if hasBusinessUnit(col) {
			q = q.ColumnExpr(col.BusinessUnitID.Expr("{}::text")+" AS ?", bun.Ident(buField))
		} else {
			q = q.ColumnExpr("?::text AS ?", req.Tenant.BuID.String(), bun.Ident(buField))
		}
		for i := range col.Extra {
			q = q.ColumnExpr(
				col.Extra[i].Expr("COALESCE({}::text, '')")+" AS ?",
				bun.Ident(extraField(i)),
			)
		}
		q = scopeTenant(q, col, req.Tenant).
			Where(col.Value.IsNotNull()).
			Where(col.Value.Expr("{}::text <> ''"))
		if req.After != "" {
			q = q.Where(col.ID.Expr("{}::text > ?"), req.After)
		}

		scanned := make([]map[string]any, 0, req.Limit)
		if err := q.
			OrderExpr(col.ID.Expr("{}::text ASC")).
			Limit(int64(req.Limit)).
			Scan(ctx, &scanned); err != nil {
			return nil, fmt.Errorf("list %s: %w", col.Name, err)
		}

		rows := make([]*kernelports.EnvelopeRow, 0, len(scanned))
		for _, values := range scanned {
			row, err := envelopeRow(col, values)
			if err != nil {
				return nil, fmt.Errorf("list %s: %w", col.Name, err)
			}
			rows = append(rows, row)
		}
		return rows, nil
	})
}

func (r *repository) Present(
	ctx context.Context,
	col *kernelports.EnvelopeColumn,
	row *kernelports.EnvelopeRow,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		q := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("? AS ?", bun.Ident(col.Table.Name), bun.Ident(col.Table.Alias)).
			Where(col.ID.Expr("{}::text = ?"), row.ID).
			Where(col.Value.Expr("{}::text = ?"), row.Sealed)
		present, err := scopeTenant(q, col, row.Tenant).Exists(ctx)
		if err != nil {
			return false, fmt.Errorf("check %s: %w", col.Name, err)
		}
		return present, nil
	})
}

func (r *repository) Swap(ctx context.Context, req *kernelports.SwapEnvelopeRequest) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		col := req.Column
		cast := ""
		if col.JSONMap {
			cast = "::jsonb"
		}
		q := r.db.DBForContext(ctx).
			NewUpdate().
			TableExpr("? AS ?", bun.Ident(col.Table.Name), bun.Ident(col.Table.Alias)).
			Set("? = ?"+cast, bun.Ident(col.Value.Name), req.Replacement).
			Where(col.ID.Expr("{}::text = ?"), req.Row.ID).
			Where(col.Value.Expr("{}::text = ?"), req.Row.Sealed)
		result, err := scopeTenantUpdate(q, col, req.Row.Tenant).Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("rewrap %s: %w", col.Name, err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("rewrap %s: %w", col.Name, err)
		}
		return affected == 1, nil
	})
}

func (r *repository) SwapVersion(
	ctx context.Context,
	req *kernelports.SwapEnvelopeVersionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		col := &req.Object.Column
		version := req.Object.Version
		q := r.db.DBForContext(ctx).
			NewUpdate().
			TableExpr("? AS ?", bun.Ident(col.Table.Name), bun.Ident(col.Table.Alias)).
			Set(version.Set(), nullable(req.Replacement)).
			Where(col.ID.Expr("{}::text = ?"), req.Row.ID).
			Where(col.Value.Expr("{}::text = ?"), req.Row.Sealed)
		q = scopeTenantUpdate(q, col, req.Row.Tenant)
		if req.Expected == "" {
			q = q.Where(version.Expr("COALESCE({}::text, '') = ''"))
		} else {
			q = q.Where(version.Expr("{}::text = ?"), req.Expected)
		}
		if _, err := q.Exec(ctx); err != nil {
			return fmt.Errorf("record the rewrapped version of %s: %w", col.Name, err)
		}
		return nil
	})
}

func hasBusinessUnit(col *kernelports.EnvelopeColumn) bool {
	return col.BusinessUnitID.Name != ""
}

func scopeTenant(
	q *bun.SelectQuery,
	col *kernelports.EnvelopeColumn,
	tenant pagination.TenantInfo,
) *bun.SelectQuery {
	q = q.Where(col.OrganizationID.Eq(), tenant.OrgID)
	if hasBusinessUnit(col) {
		q = q.Where(col.BusinessUnitID.Eq(), tenant.BuID)
	}
	return q
}

func scopeTenantUpdate(
	q *bun.UpdateQuery,
	col *kernelports.EnvelopeColumn,
	tenant pagination.TenantInfo,
) *bun.UpdateQuery {
	q = q.Where(col.OrganizationID.Eq(), tenant.OrgID)
	if hasBusinessUnit(col) {
		q = q.Where(col.BusinessUnitID.Eq(), tenant.BuID)
	}
	return q
}

func envelopeRow(
	col *kernelports.EnvelopeColumn,
	values map[string]any,
) (*kernelports.EnvelopeRow, error) {
	orgID, err := pulid.Parse(text(values[orgField]))
	if err != nil {
		return nil, err
	}
	buID, err := pulid.Parse(text(values[buField]))
	if err != nil {
		return nil, err
	}

	row := &kernelports.EnvelopeRow{
		ID:     text(values[idField]),
		Tenant: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Sealed: text(values[sealedField]),
		Extra:  make(map[string]string, len(col.Extra)),
	}
	for i := range col.Extra {
		row.Extra[col.Extra[i].Name] = text(values[extraField(i)])
	}
	return row, nil
}

func extraField(i int) string {
	return "envelope_row_extra_" + strconv.Itoa(i)
}

func text(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
