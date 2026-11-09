package auditrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var entryErrors = dberror.Mapping{Entity: "Audit Entry"}

var summaryColumns = []string{
	"id", "business_unit_id", "organization_id", "timestamp", "resource", "resource_id",
	"operation", "action", "category", "retention_class", "channel", "principal_type",
	"principal_id", "user_id", "api_key_id", "actor_name", "ai_run_id", "entity_ref",
	"correlation_id", "trace_id", "ip_address", "user_agent", "comment", "sensitive_data",
	"critical",
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) auditports.AuditRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.audit-repository"),
	}
}

func withPrincipals(q keel.Query[*audit.Entry]) keel.Query[*audit.Entry] {
	e := auditdb.Entries
	return q.Load(e.User(), e.APIKey())
}

func accessPreds(access auditports.EntryAccess) ([]keel.Pred, bool) {
	return accessPredsOn(auditdb.Entries, access)
}

func accessPredsOn(e *auditdb.EntriesTable, access auditports.EntryAccess) ([]keel.Pred, bool) {
	preds := make([]keel.Pred, 0, 2)
	if !access.AllResources {
		if len(access.Resources) == 0 {
			return nil, false
		}
		names := make([]string, 0, len(access.Resources))
		for _, resource := range access.Resources {
			names = append(names, resource.String())
		}
		preds = append(preds, e.Resource.In(permissionResources(names)...))
	}
	if !access.IncludeSecurity {
		preds = append(preds, e.Category.Ne(audit.CategorySecurity))
	}
	return preds, true
}

const requestLookback = int64(60 * 60)

func firstOfRequest(access auditports.EntryAccess) keel.Pred {
	e := auditdb.Entries
	p := auditdb.Entries.As("earlier_entry")
	earlier, _ := accessPredsOn(p, access)
	earlier = append(earlier,
		p.CorrelationID.EqCol(e.CorrelationID),
		keel.Int64Of(p.Timestamp).GteExpr(e.Timestamp.Sub(keel.Int64(requestLookback))),
		keel.Or(
			keel.Int64Of(p.Timestamp).LtExpr(e.Timestamp),
			keel.And(
				keel.Int64Of(p.Timestamp).EqExpr(e.Timestamp),
				keel.Raw("? < ?", p.ID, e.ID),
			),
		),
	)
	return keel.NotExists(p.Where(earlier...))
}

func narrowed(filter *pagination.QueryOptions) bool {
	return filter != nil && (filter.Query != "" || len(filter.FieldFilters) > 0 ||
		len(filter.FilterGroups) > 0 || len(filter.GeoFilters) > 0 || len(filter.AggregateFilters) > 0)
}

func permissionResources(names []string) []permission.Resource {
	out := make([]permission.Resource, 0, len(names))
	for _, name := range names {
		out = append(out, permission.Resource(name))
	}
	return out
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *auditports.ListAuditEntriesRequest,
) (*pagination.CursorListResult[*audit.Entry], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*audit.Entry], error) {
			preds, allowed := accessPreds(req.Access)
			if !allowed {
				return &pagination.CursorListResult[*audit.Entry]{
					Items: []*audit.Entry{},
				}, nil
			}

			if req.Filter != nil && len(req.Filter.Sort) == 0 {
				req.Filter.Sort = []domaintypes.SortField{
					{Field: "timestamp", Direction: dbtype.SortDirectionDesc},
				}
			}

			e := auditdb.Entries
			if req.ResourceID != "" {
				preds = append(preds, e.ResourceID.Eq(req.ResourceID))
			}
			if req.ChangedField != "" {
				preds = append(preds, e.Changes.HasKey(req.ChangedField))
			}
			if req.OnePerRequest && req.ResourceID == "" && !narrowed(req.Filter) {
				preds = append(preds, firstOfRequest(req.Access))
			}

			lq := &listquery.Request[audit.Entry]{
				Filter: req.Filter,
				Cursor: req.Cursor,
				Shape:  withPrincipals,
			}
			switch {
			case req.IncludeDetail:
			case req.IncludeChanges:
				lq.Columns = append(append([]string{}, summaryColumns...), "changes")
			default:
				lq.Columns = summaryColumns
			}

			result, err := listquery.Connection(ctx, r.db.Keel(ctx), e.Where(preds...), lq)
			if err != nil {
				r.l.Error("failed to list audit entry connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req *auditports.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*audit.Entry, error) {
		preds, allowed := accessPreds(req.Access)
		if !allowed {
			return nil, entryErrors.Map(keel.ErrNotFound, req.EntryID.String())
		}

		e := auditdb.Entries
		preds = append(preds, e.ID.Eq(req.EntryID))
		entity, err := withPrincipals(e.Where(preds...)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, entryErrors.Map(err, req.EntryID.String())
		}

		return entity, nil
	})
}

func (r *repository) Insert(ctx context.Context, entries []*audit.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx = insertScope(ctx, entries)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := auditdb.Entries.CreateMany(ctx, r.db.Keel(ctx), entries); err != nil {
			r.l.Error(
				"failed to insert audit entries",
				zap.Int("count", len(entries)),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}

func insertScope(ctx context.Context, entries []*audit.Entry) context.Context {
	first := entries[0]
	for _, entry := range entries[1:] {
		if entry.OrganizationID != first.OrganizationID ||
			entry.BusinessUnitID != first.BusinessUnitID {
			return dbscope.WithSystem(
				ctx,
				"record audit entries that span organizations in one statement",
			)
		}
	}

	if current, ok := dbscope.TenantFrom(ctx); ok {
		if current.OrganizationID == first.OrganizationID &&
			current.BusinessUnitID == first.BusinessUnitID {
			return ctx
		}
		return dbscope.WithSystem(
			ctx,
			"record an audit entry about another organization than the caller's",
		)
	}

	return dbscope.WithValidTenant(ctx, dbscope.Tenant{
		OrganizationID: first.OrganizationID,
		BusinessUnitID: first.BusinessUnitID,
	})
}

func (r *repository) ExportPage(
	ctx context.Context,
	req *auditports.ExportPageRequest,
) ([]*audit.Entry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.Entry, error) {
		e := auditdb.Entries
		export := req.Export
		preds := []keel.Pred{
			e.Timestamp.Gte(export.RangeFrom),
			e.Timestamp.Lt(export.RangeTo),
		}
		if !export.IncludeSecurity {
			preds = append(preds, e.Category.Ne(audit.CategorySecurity))
		}
		filters := export.Filters
		if len(filters.Resources) > 0 {
			preds = append(preds, e.Resource.In(permissionResources(filters.Resources)...))
		}
		if len(filters.Operations) > 0 {
			ops := make([]permission.Operation, 0, len(filters.Operations))
			for _, op := range filters.Operations {
				ops = append(ops, permission.Operation(op))
			}
			preds = append(preds, e.Operation.In(ops...))
		}
		if len(filters.Categories) > 0 {
			cats := make([]audit.Category, 0, len(filters.Categories))
			for _, c := range filters.Categories {
				cats = append(cats, audit.Category(c))
			}
			preds = append(preds, e.Category.In(cats...))
		}
		if len(filters.Channels) > 0 {
			chans := make([]audit.Channel, 0, len(filters.Channels))
			for _, c := range filters.Channels {
				chans = append(chans, audit.Channel(c))
			}
			preds = append(preds, e.Channel.In(chans...))
		}
		if filters.PrincipalID != "" {
			preds = append(preds, e.PrincipalID.Eq(pulid.ID(filters.PrincipalID)))
		}
		if filters.ResourceID != "" {
			preds = append(preds, e.ResourceID.Eq(filters.ResourceID))
		}
		if req.AfterID.IsNotNil() {
			preds = append(preds, keel.Or(
				e.Timestamp.Gt(req.AfterTS),
				keel.And(e.Timestamp.Eq(req.AfterTS), e.ID.Gt(req.AfterID)),
			))
		}

		rows, err := e.Where(preds...).
			Order(e.Timestamp.Asc(), e.ID.Asc()).
			Limit(req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read audit export page: %w", err)
		}
		return rows, nil
	})
}

func (r *repository) DeleteExpired(
	ctx context.Context,
	req auditports.DeleteExpiredAuditEntriesRequest,
) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete audit entries past an organization's retention during the retention sweep",
	)
	return postgres.DeleteUnderAuditRetention(
		ctx,
		r.db,
		func(ctx context.Context, db *keel.DB) (int64, error) {
			return keel.Exec(ctx, db, `
				WITH doomed AS (
					SELECT e.id, e.timestamp
					FROM audit_entries e
					WHERE e.organization_id = $1
						AND e.business_unit_id = $2
						AND e.retention_class = $3
						AND e.timestamp < $4
						AND NOT EXISTS (
							SELECT 1 FROM audit_legal_holds h
							WHERE h.organization_id = e.organization_id
								AND h.business_unit_id = e.business_unit_id
								AND h.released_at IS NULL
								AND (h.resource IS NULL OR (h.resource = e.resource
									AND (h.resource_id IS NULL OR h.resource_id = e.resource_id)))
						)
					ORDER BY e.timestamp
					LIMIT $5
				)
				DELETE FROM audit_entries e
				USING doomed d
				WHERE e.id = d.id
					AND e.timestamp = d.timestamp
					AND e.organization_id = $1
					AND e.business_unit_id = $2`,
				req.OrgID.String(),
				req.BuID.String(),
				string(req.Class),
				req.Before,
				req.Limit,
			)
		},
	)
}

func (r *repository) PartitionHorizon(ctx context.Context) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"read how far ahead the audit trail's shared monthly partitions reach",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		horizon, err := keel.QueryAll(ctx, r.db.Keel(ctx), pgx.RowTo[int64], `
			SELECT COALESCE(max(
				(regexp_match(pg_get_expr(c.relpartbound, c.oid), 'TO \(''?(-?\d+)''?\)'))[1]::BIGINT
			), 0)
			FROM pg_inherits i
			JOIN pg_class c ON c.oid = i.inhrelid
			WHERE i.inhparent = 'public.audit_entries'::regclass`)
		if err != nil {
			return 0, fmt.Errorf("read audit partition horizon: %w", err)
		}
		if len(horizon) == 0 {
			return 0, nil
		}
		return horizon[0], nil
	})
}

func (r *repository) TenantsWithEntries(
	ctx context.Context,
	from, to int64,
) ([]auditports.TenantKey, error) {
	ctx = dbscope.WithSystem(ctx, "find the organizations with audit entries to seal")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]auditports.TenantKey, error) {
		e := auditdb.Entries
		type row struct {
			OrganizationID pulid.ID
			BusinessUnitID pulid.ID
		}
		rows, err := keel.Into[row](ctx, r.db.Keel(ctx), e.AllTenants().
			Distinct().
			Pick(e.OrganizationID, e.BusinessUnitID).
			Where(e.Timestamp.Gte(from), e.Timestamp.Lt(to)))
		if err != nil {
			return nil, fmt.Errorf("list tenants with audit entries: %w", err)
		}
		out := make([]auditports.TenantKey, 0, len(rows))
		for _, rw := range rows {
			out = append(out, auditports.TenantKey{
				OrganizationID: rw.OrganizationID,
				BusinessUnitID: rw.BusinessUnitID,
			})
		}
		return out, nil
	})
}

type requestCountRow struct {
	CorrelationID string
	Entries       int64
}

func (r *repository) CountByRequest(
	ctx context.Context,
	req *auditports.CountByRequestRequest,
) (map[string]int64, error) {
	out := make(map[string]int64, len(req.CorrelationIDs))
	if len(req.CorrelationIDs) == 0 {
		return out, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[string]int64, error) {
		preds, allowed := accessPreds(req.Access)
		if !allowed {
			return out, nil
		}

		e := auditdb.Entries
		preds = append(preds,
			e.CorrelationID.In(req.CorrelationIDs...),
			e.Timestamp.Gte(req.From),
			e.Timestamp.Lte(req.To),
		)

		var rows []requestCountRow
		err := keel.Batch(ctx, r.db.Keel(ctx), keel.IntoSlice(
			e.Group(e.CorrelationID).
				Pick(e.CorrelationID, keel.CastInt64(keel.Count())).
				Where(preds...),
			&rows,
		))
		if err != nil {
			r.l.Error("failed to count audit entries by request", zap.Error(err))
			return nil, err
		}

		for _, row := range rows {
			out[row.CorrelationID] = row.Entries
		}

		return out, nil
	})
}
