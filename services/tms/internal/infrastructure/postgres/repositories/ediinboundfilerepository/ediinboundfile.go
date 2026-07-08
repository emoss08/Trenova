//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediinboundfilerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) repositories.EDIInboundFileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-inbound-file-repository"),
	}
}

func (r *repository) ListInboundFiles(
	ctx context.Context,
	req *repositories.ListEDIInboundFilesRequest,
) (*pagination.ListResult[*edi.EDIInboundFile], error) {
	entities := make([]*edi.EDIInboundFile, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDIInboundFileColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("Partner")
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.PartnerID.IsNotNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	query = query.ExcludeColumn("raw_content")

	total, err := querybuilder.ApplyFilters(query, "eif", req.Filter, (*edi.EDIInboundFile)(nil)).
		Order(cols.ReceivedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDIInboundFile]{Items: entities, Total: total}, nil
}

func (r *repository) GetInboundFileByID(
	ctx context.Context,
	req repositories.GetEDIInboundFileByIDRequest,
) (*edi.EDIInboundFile, error) {
	entity := new(edi.EDIInboundFile)
	cols := buncolgen.EDIInboundFileColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Partner").
		Relation("CommunicationProfile").
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.EDIInboundFileApplyTenant(req.TenantInfo))
	if req.IncludeMessages {
		query = query.Relation("Messages")
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIInboundFile")
	}
	return entity, nil
}

func (r *repository) CreateInboundFile(
	ctx context.Context,
	entity *edi.EDIInboundFile,
) (*edi.EDIInboundFile, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdateInboundFile(
	ctx context.Context,
	entity *edi.EDIInboundFile,
) (*edi.EDIInboundFile, error) {
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDIInboundFile", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) PurgeRawContentBefore(
	ctx context.Context,
	req repositories.PurgeEDIRawPayloadsRequest,
) (int64, error) {
	cols := buncolgen.EDIInboundFileColumns
	subquery := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIInboundFile)(nil)).
		Column("id").
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.ReceivedAt.Lt(), req.Before).
		Where(cols.Status.In(), bun.List([]edi.InboundFileStatus{
			edi.InboundFileStatusProcessed,
			edi.InboundFileStatusDuplicate,
		})).
		Where("eif.raw_purged_at IS NULL").
		Limit(req.Limit)

	result, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*edi.EDIInboundFile)(nil)).
		Set("raw_content = ''").
		Set("raw_purged_at = ?", req.PurgedAt).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where("eif.id IN (?)", subquery).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *repository) CountQuarantinedSince(ctx context.Context, since int64) (int64, error) {
	cols := buncolgen.EDIInboundFileColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIInboundFile)(nil)).
		Where(cols.Status.Eq(), edi.InboundFileStatusQuarantined).
		Where(cols.UpdatedAt.Gte(), since).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (r *repository) ExistsByChecksum(
	ctx context.Context,
	req repositories.ExistsEDIInboundFileByChecksumRequest,
) (bool, error) {
	cols := buncolgen.EDIInboundFileColumns
	return r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIInboundFile)(nil)).
		Where(cols.CommunicationProfileID.Eq(), req.CommunicationProfileID).
		Where(cols.Checksum.Eq(), req.Checksum).
		Apply(buncolgen.EDIInboundFileApplyTenant(req.TenantInfo)).
		Exists(ctx)
}

func (r *repository) GetInboundFileStatusCounts(
	ctx context.Context,
	req repositories.GetEDIInboundFileStatusCountsRequest,
) (map[edi.InboundFileStatus]int, error) {
	cols := buncolgen.EDIInboundFileColumns
	var rows []struct {
		Status edi.InboundFileStatus `bun:"status"`
		Count  int                   `bun:"count"`
	}
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDIInboundFile)(nil)).
		ColumnExpr(cols.Status.Qualified()).
		ColumnExpr("COUNT(*) AS count").
		Apply(buncolgen.EDIInboundFileApplyTenant(req.TenantInfo)).
		GroupExpr(cols.Status.Qualified())
	if req.Since > 0 {
		query = query.Where(cols.ReceivedAt.Gte(), req.Since)
	}
	if err := query.Scan(ctx, &rows); err != nil {
		return nil, err
	}
	counts := make(map[edi.InboundFileStatus]int, len(rows))
	for _, row := range rows {
		counts[row.Status] = row.Count
	}
	return counts, nil
}

func (r *repository) ListRecentQuarantined(
	ctx context.Context,
	req repositories.ListRecentQuarantinedEDIInboundFilesRequest,
) ([]*edi.EDIInboundFile, error) {
	cols := buncolgen.EDIInboundFileColumns
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	entities := make([]*edi.EDIInboundFile, 0, limit)
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ExcludeColumn("raw_content").
		Relation("Partner").
		Where(cols.Status.Eq(), edi.InboundFileStatusQuarantined).
		Apply(buncolgen.EDIInboundFileApplyTenant(req.TenantInfo)).
		Order(cols.ReceivedAt.OrderDesc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) ListInboundFilesCursor(
	ctx context.Context,
	req *repositories.ListEDIInboundFilesRequest,
) (*pagination.CursorListResult[*edi.EDIInboundFile], error) {
	dba := r.db.DBForContext(ctx)
	cols := buncolgen.EDIInboundFileColumns
	extraFilters := func(sq *bun.SelectQuery) *bun.SelectQuery {
		if req.Status != "" {
			sq = sq.Where(cols.Status.Eq(), req.Status)
		}
		if req.PartnerID.IsNotNil() {
			sq = sq.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
		}
		return sq
	}

	total, err := dba.
		NewSelect().
		Model((*edi.EDIInboundFile)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFiltersWithoutSort(
				sq,
				"eif",
				req.Filter,
				(*edi.EDIInboundFile)(nil),
			)
			return extraFilters(sq)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*edi.EDIInboundFile]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*edi.EDIInboundFile) *bun.SelectQuery {
			return dba.
				NewSelect().
				Model(entities).
				ExcludeColumn("raw_content").
				Relation("Partner")
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			sq, applyErr := querybuilder.ApplyCursorFilters(
				sq,
				"eif",
				req.Filter,
				req.Cursor,
				(*edi.EDIInboundFile)(nil),
			)
			if applyErr != nil {
				return sq, applyErr
			}
			return extraFilters(sq), nil
		},
	})
}
