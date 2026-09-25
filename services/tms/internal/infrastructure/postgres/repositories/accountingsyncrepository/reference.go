package accountingsyncrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	referenceUpsertBatch   = 500
	defaultReferenceSearch = 25
	maxReferenceSearch     = 100
)

type ReferenceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type referenceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewReferenceRepository(p ReferenceParams) repositories.AccountingReferenceObjectRepository {
	return &referenceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-reference-repository"),
	}
}

func (r *referenceRepository) Upsert(
	ctx context.Context,
	req *repositories.UpsertAccountingReferenceObjectsRequest,
) error {
	if len(req.Objects) == 0 {
		return nil
	}

	for _, obj := range req.Objects {
		obj.OrganizationID = req.TenantInfo.OrgID
		obj.BusinessUnitID = req.TenantInfo.BuID
		obj.ConnectionID = req.ConnectionID
		obj.LastSeenAt = req.SeenAt
		obj.RemovedAt = nil
	}

	cols := buncolgen.AccountingReferenceObjectColumns
	for start := 0; start < len(req.Objects); start += referenceUpsertBatch {
		batch := req.Objects[start:min(start+referenceUpsertBatch, len(req.Objects))]
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(&batch).
			On("CONFLICT (organization_id, business_unit_id, connection_id, kind, external_id) DO UPDATE").
			Set(cols.Name.SetExcluded()).
			Set(cols.SearchName.SetExcluded()).
			Set(cols.FullyQualifiedName.SetExcluded()).
			Set(cols.Number.SetExcluded()).
			Set(cols.Description.SetExcluded()).
			Set(cols.Classification.SetExcluded()).
			Set(cols.AccountType.SetExcluded()).
			Set(cols.AccountSubType.SetExcluded()).
			Set(cols.ItemType.SetExcluded()).
			Set(cols.SubType.SetExcluded()).
			Set(cols.ParentExternalID.SetExcluded()).
			Set(cols.Active.SetExcluded()).
			Set(cols.CurrencyCode.SetExcluded()).
			Set(cols.SyncToken.SetExcluded()).
			Set(cols.CompanyName.SetExcluded()).
			Set(cols.Email.SetExcluded()).
			Set(cols.AddressLine1.SetExcluded()).
			Set(cols.City.SetExcluded()).
			Set(cols.State.SetExcluded()).
			Set(cols.PostalCode.SetExcluded()).
			Set(cols.IncomeAccountExternalID.SetExcluded()).
			Set(cols.DueDays.SetExcluded()).
			Set(cols.Is1099.SetExcluded()).
			Set(cols.ProviderUpdatedAt.SetExcluded()).
			Set(cols.LastSeenAt.SetExcluded()).
			Set(cols.RemovedAt.SetNull()).
			Set(cols.UpdatedAt.SetExcluded()).
			Exec(ctx); err != nil {
			return fmt.Errorf("upsert accounting reference objects: %w", err)
		}
	}

	return nil
}

func (r *referenceRepository) MarkRemovedUnseen(
	ctx context.Context,
	req *repositories.MarkAccountingReferenceRemovedRequest,
) (int64, error) {
	cols := buncolgen.AccountingReferenceObjectColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingReferenceObject)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AccountingReferenceObjectScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ConnectionID.Eq(), req.ConnectionID).
				Where(cols.Kind.Eq(), req.Kind).
				Where(cols.LastSeenAt.Lt(), req.SeenBefore).
				Where(cols.RemovedAt.IsNull())
		}).
		Set(cols.RemovedAt.Set(), req.At).
		Set(cols.UpdatedAt.Set(), req.At).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return results.RowsAffected()
}

func (r *referenceRepository) ListByKind(
	ctx context.Context,
	req *repositories.ListAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	entities := make([]*accountingsync.AccountingReferenceObject, 0)
	cols := buncolgen.AccountingReferenceObjectColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingReferenceObjectApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Kind.Eq(), req.Kind).
		Order(cols.SearchName.OrderAsc(), cols.ExternalID.OrderAsc())
	if req.UsableOnly {
		query = applyUsable(query)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *referenceRepository) Search(
	ctx context.Context,
	req *repositories.SearchAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultReferenceSearch
	}
	limit = min(limit, maxReferenceSearch)

	entities := make([]*accountingsync.AccountingReferenceObject, 0, limit)
	cols := buncolgen.AccountingReferenceObjectColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingReferenceObjectApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.Kind.Eq(), req.Kind).
		Limit(limit)
	if req.UsableOnly {
		query = applyUsable(query)
	}

	if term := stringutils.NormalizeName(req.Query); term != "" {
		pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
		prefix := stringutils.EscapeLikePattern(term) + "%"
		query = query.
			WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where(cols.SearchName.Like(), pattern).
					WhereOr(cols.Number.ILike(), prefix).
					WhereOr(cols.ExternalID.Eq(), strings.TrimSpace(req.Query))
			}).
			OrderExpr(cols.SearchName.Expr("CASE WHEN {} LIKE ? THEN 0 ELSE 1 END"), prefix)
	}
	query = query.Order(cols.SearchName.OrderAsc(), cols.ExternalID.OrderAsc())

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *referenceRepository) GetByExternalIDs(
	ctx context.Context,
	req *repositories.GetAccountingReferenceObjectsRequest,
) ([]*accountingsync.AccountingReferenceObject, error) {
	if len(req.ExternalIDs) == 0 {
		return []*accountingsync.AccountingReferenceObject{}, nil
	}

	entities := make([]*accountingsync.AccountingReferenceObject, 0, len(req.ExternalIDs))
	cols := buncolgen.AccountingReferenceObjectColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(buncolgen.AccountingReferenceObjectApplyTenant(req.TenantInfo)).
		Where(cols.ConnectionID.Eq(), req.ConnectionID).
		Where(cols.ExternalID.In(), bun.List(req.ExternalIDs))
	if req.Kind != "" {
		query = query.Where(cols.Kind.Eq(), req.Kind)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func applyUsable(q *bun.SelectQuery) *bun.SelectQuery {
	cols := buncolgen.AccountingReferenceObjectColumns
	return q.
		Where(cols.Active.IsTrue()).
		Where(cols.RemovedAt.IsNull()).
		WhereGroup(" AND ", func(g *bun.SelectQuery) *bun.SelectQuery {
			return g.Where(cols.ItemType.IsNull()).
				WhereOr(cols.ItemType.NotIn(), bun.List([]string{
					accountingsync.ItemTypeCategory,
					accountingsync.ItemTypeGroup,
				}))
		})
}
