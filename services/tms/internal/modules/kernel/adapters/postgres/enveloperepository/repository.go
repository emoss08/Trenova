package enveloperepository

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
)

const (
	listTenantsScopeReason = "list every organization to rewrap its sealed values to the active key"

	listTenantsSQL = "SELECT id, business_unit_id FROM organizations ORDER BY id"

	idColumn             = "id"
	organizationIDColumn = "organization_id"
	businessUnitIDColumn = "business_unit_id"
)

var (
	errNotAColumn      = errors.New("not a column of a table")
	errOtherTable      = errors.New("not a column of the value's table")
	errNoTenantColumns = errors.New("the table has no tenant columns")
	errMissingColumn   = errors.New("the table has no such column")
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
	ID             pulid.ID
	BusinessUnitID pulid.ID
}

type envelopeRecord struct {
	ID             string
	OrganizationID string
	BusinessUnitID string
	Sealed         string
	Extra          []string
}

type source struct {
	name       string
	table      keel.Handle[keel.Row]
	id         keel.StringCol
	org        keel.Col
	bu         *keel.Col
	value      keel.Col
	extra      []keel.Col
	extraNames []string
	untouched  []keel.Assignment
}

func (r *repository) ListTenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, listTenantsScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		rows, err := keel.QueryAll(
			ctx,
			r.db.Keel(ctx),
			pgx.RowToStructByPos[tenantRow],
			listTenantsSQL,
		)
		if err != nil {
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
		src, err := resolve(req.Column)
		if err != nil {
			return nil, err
		}

		records, err := keel.Into[envelopeRecord](ctx, r.db.Keel(ctx), src.listQuery(req))
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", src.name, err)
		}

		rows := make([]*kernelports.EnvelopeRow, 0, len(records))
		for i := range records {
			row, rowErr := src.envelopeRow(&records[i])
			if rowErr != nil {
				return nil, fmt.Errorf("list %s: %w", src.name, rowErr)
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
		src, err := resolve(col)
		if err != nil {
			return false, err
		}

		present, err := src.table.Where(src.matches(row)...).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check %s: %w", src.name, err)
		}
		return present, nil
	})
}

func (r *repository) Swap(ctx context.Context, req *kernelports.SwapEnvelopeRequest) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		src, err := resolve(req.Column)
		if err != nil {
			return false, err
		}

		replacement := keel.DynLit(req.Replacement)
		if req.Column.JSONMap {
			replacement = replacement.Cast(keel.TypeJSON)
		}

		sets := slices.Concat(src.untouched, []keel.Assignment{src.value.ToExpr(replacement)})
		affected, err := src.table.Where(src.matches(req.Row)...).
			Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return false, fmt.Errorf("rewrap %s: %w", src.name, err)
		}
		return affected == 1, nil
	})
}

func (r *repository) SwapVersion(
	ctx context.Context,
	req *kernelports.SwapEnvelopeVersionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		src, err := resolve(&req.Object.Column)
		if err != nil {
			return err
		}
		version, err := src.column(req.Object.Version)
		if err != nil {
			return fmt.Errorf("%s version: %w", src.name, err)
		}

		set := version.ToExpr(keel.DynLit(req.Replacement))
		if req.Replacement == "" {
			set = version.ToNull()
		}
		versionText := keel.DynOf(version).Cast(keel.TypeString)
		expected := versionText.Eq(req.Expected)
		if req.Expected == "" {
			expected = versionText.Coalesce("").Eq("")
		}

		sets := slices.Concat(src.untouched, []keel.Assignment{set})
		if _, err = src.table.Where(slices.Concat(src.matches(req.Row), []keel.Pred{expected})...).
			Update(ctx, r.db.Keel(ctx), sets...); err != nil {
			return fmt.Errorf("record the rewrapped version of %s: %w", src.name, err)
		}
		return nil
	})
}

func resolve(col *kernelports.EnvelopeColumn) (*source, error) {
	table, valueIndex, ok := keel.ColumnOf(col.Value)
	if !ok {
		return nil, fmt.Errorf("%s value: %w", col.Name, errNotAColumn)
	}
	if len(table.Tenant) == 0 {
		return nil, fmt.Errorf("%s (%s): %w", col.Name, table.Name, errNoTenantColumns)
	}

	ref := &keel.Ref{Table: table}
	src := &source{
		name:  col.Name,
		table: keel.Handle[keel.Row]{Ref: ref},
		value: keel.Col{Ref: ref, Index: valueIndex},
	}

	id, err := src.named(idColumn)
	if err != nil {
		return nil, err
	}
	src.id = keel.StringCol{Col: id}
	if src.org, err = src.named(organizationIDColumn); err != nil {
		return nil, err
	}
	if bu, buErr := src.named(businessUnitIDColumn); buErr == nil {
		src.bu = &bu
	}

	src.extra = make([]keel.Col, 0, len(col.Extra))
	src.extraNames = make([]string, 0, len(col.Extra))
	for _, e := range col.Extra {
		c, colErr := src.column(e)
		if colErr != nil {
			return nil, fmt.Errorf("%s extra: %w", col.Name, colErr)
		}
		src.extra = append(src.extra, c)
		src.extraNames = append(src.extraNames, table.Columns[c.Index].Name)
	}

	src.untouched = untouched(ref)
	return src, nil
}

func untouched(ref *keel.Ref) []keel.Assignment {
	sets := make([]keel.Assignment, 0, 2)
	for _, index := range []int32{ref.Table.Version, ref.Table.AutoUpdate} {
		if index < 0 {
			continue
		}
		c := keel.Col{Ref: ref, Index: index}
		sets = append(sets, c.ToExpr(c))
	}
	return sets
}

func (s *source) named(name string) (keel.Col, error) {
	table := s.table.Ref.Table
	index := table.Column(name)
	if index < 0 || !table.Columns[index].Stored() {
		return keel.Col{}, fmt.Errorf("%s (%s) %s: %w", s.name, table.Name, name, errMissingColumn)
	}
	return keel.Col{Ref: s.table.Ref, Index: index}, nil
}

func (s *source) column(e keel.Expr) (keel.Col, error) {
	table, index, ok := keel.ColumnOf(e)
	if !ok {
		return keel.Col{}, errNotAColumn
	}
	if table != s.table.Ref.Table {
		return keel.Col{}, errOtherTable
	}
	return keel.Col{Ref: s.table.Ref, Index: index}, nil
}

func (s *source) listQuery(req *kernelports.ListEnvelopeRowsRequest) *keel.Query[keel.Row] {
	bu := keel.Expr(keel.DynLit(req.Tenant.BuID.String()).Cast(keel.TypeString))
	if s.bu != nil {
		bu = keel.DynOf(*s.bu).Cast(keel.TypeString)
	}

	return s.table.Pick(
		keel.DynOf(s.id).Cast(keel.TypeString),
		keel.DynOf(s.org).Cast(keel.TypeString),
		bu,
		keel.DynOf(s.value).Cast(keel.TypeString),
		s.extraArray(),
	).
		Where(
			s.value.IsNotNull(),
			keel.DynOf(s.value).Cast(keel.TypeString).Ne(""),
			keel.When(req.After != "", s.id.Gt(req.After)),
		).
		Order(s.id.Asc()).
		Limit(req.Limit)
}

func (s *source) extraArray() keel.Expr {
	if len(s.extra) == 0 {
		return keel.DynLit([]string{})
	}

	items := make([]any, len(s.extra))
	for i := range s.extra {
		items[i] = keel.DynOf(s.extra[i]).Cast(keel.TypeString).Coalesce("")
	}
	return keel.DynArray(items...)
}

func (s *source) matches(row *kernelports.EnvelopeRow) []keel.Pred {
	return []keel.Pred{
		s.id.Eq(row.ID),
		keel.DynOf(s.value).Cast(keel.TypeString).Eq(row.Sealed),
	}
}

func (s *source) envelopeRow(record *envelopeRecord) (*kernelports.EnvelopeRow, error) {
	orgID, err := pulid.Parse(record.OrganizationID)
	if err != nil {
		return nil, err
	}
	buID, err := pulid.Parse(record.BusinessUnitID)
	if err != nil {
		return nil, err
	}

	row := &kernelports.EnvelopeRow{
		ID:     record.ID,
		Tenant: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		Sealed: record.Sealed,
		Extra:  make(map[string]string, len(s.extraNames)),
	}
	for i, name := range s.extraNames {
		if i < len(record.Extra) {
			row.Extra[name] = record.Extra[i]
		}
	}
	return row, nil
}
