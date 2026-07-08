//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edicommunicationprofilerepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
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

func New(p Params) repositories.EDICommunicationProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-communication-profile-repository"),
	}
}

func (r *repository) ListProfiles(
	ctx context.Context,
	req *repositories.ListEDICommunicationProfilesRequest,
) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
	entities := make([]*edi.EDICommunicationProfile, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.EDICommunicationProfileColumns

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(buncolgen.EDICommunicationProfileRelations.Partner).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFilters(
				query,
				"ecp",
				req.Filter,
				(*edi.EDICommunicationProfile)(nil),
			)
		}).
		Apply(buncolgen.EDICommunicationProfileApplyTenant(req.Filter.TenantInfo)).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDICommunicationProfile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) SelectProfileOptions(
	ctx context.Context,
	req *repositories.EDICommunicationProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
	entities := make(
		[]*edi.EDICommunicationProfile,
		0,
		req.SelectQueryRequest.Pagination.SafeLimit(),
	)
	cols := buncolgen.EDICommunicationProfileColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			cols.ID.Bare(),
			cols.BusinessUnitID.Bare(),
			cols.OrganizationID.Bare(),
			cols.EDIConnectionID.Bare(),
			cols.EDIPartnerID.Bare(),
			cols.Method.Bare(),
			cols.Status.Bare(),
			cols.Name.Bare(),
			cols.Description.Bare(),
		).
		Apply(buncolgen.EDICommunicationProfileApplyTenant(req.SelectQueryRequest.TenantInfo))

	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}
	if req.Method != "" {
		query = query.Where(cols.Method.Eq(), req.Method)
	}
	if req.PartnerID.IsNotNil() {
		query = query.Where(cols.EDIPartnerID.Eq(), req.PartnerID)
	}
	query = applyCommunicationProfileSearch(query, req.SelectQueryRequest.Query)

	total, err := query.
		Order(cols.Name.OrderAsc()).
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDICommunicationProfile]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetProfileByID(
	ctx context.Context,
	req repositories.GetEDICommunicationProfileByIDRequest,
) (*edi.EDICommunicationProfile, error) {
	entity := new(edi.EDICommunicationProfile)
	cols := buncolgen.EDICommunicationProfileColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(buncolgen.EDICommunicationProfileRelations.Partner).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.EDICommunicationProfileApplyTenant(req.TenantInfo)).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDICommunicationProfile")
	}

	return entity, nil
}

func applyCommunicationProfileSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	cols := buncolgen.EDICommunicationProfileColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.Name.LowerLike(), term).
			WhereOr(cols.Description.LowerLike(), term).
			WhereOr(cols.Method.TextILike(), term)
	})
}

func (r *repository) GetActiveProfileByPartner(
	ctx context.Context,
	req repositories.GetActiveEDICommunicationProfileByPartnerRequest,
) (*edi.EDICommunicationProfile, error) {
	entity := new(edi.EDICommunicationProfile)
	cols := buncolgen.EDICommunicationProfileColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.EDIPartnerID.Eq(), req.PartnerID).
		Apply(buncolgen.EDICommunicationProfileApplyTenant(req.TenantInfo)).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		Limit(1)
	if len(req.Methods) > 0 {
		query = query.Where(cols.Method.In(), bun.List(req.Methods))
	} else {
		query = query.Where(cols.Method.Eq(), req.Method)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDICommunicationProfile")
	}

	return entity, nil
}

func (r *repository) ListInboundPollingProfiles(
	ctx context.Context,
) ([]*edi.EDICommunicationProfile, error) {
	entities := make([]*edi.EDICommunicationProfile, 0)
	cols := buncolgen.EDICommunicationProfileColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		Where(cols.Method.In(), bun.List([]edi.ConnectionMethod{
			edi.ConnectionMethodSFTP,
			edi.ConnectionMethodVAN,
		})).
		Where("ecp.edi_partner_id IS NOT NULL").
		Where("COALESCE(TRIM(ecp.config->>'inboundDirectory'), '') <> ''").
		Order(cols.CreatedAt.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) RecordInboundPollOutcome(
	ctx context.Context,
	req repositories.RecordEDIProfilePollOutcomeRequest,
) error {
	cols := buncolgen.EDICommunicationProfileColumns
	update := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*edi.EDICommunicationProfile)(nil)).
		Set("last_poll_attempt_at = ?", req.PolledAt).
		Where(cols.ID.Eq(), req.ProfileID).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID)
	if req.Success {
		update = update.
			Set("last_poll_success_at = ?", req.PolledAt).
			Set("last_poll_error = NULL")
	} else {
		update = update.Set("last_poll_error = ?", req.Error)
	}
	_, err := update.Exec(ctx)
	return err
}

func (r *repository) CountStaleInboundPollingProfiles(
	ctx context.Context,
	staleBefore int64,
) (int64, error) {
	cols := buncolgen.EDICommunicationProfileColumns
	count, err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*edi.EDICommunicationProfile)(nil)).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		Where(cols.Method.In(), bun.List([]edi.ConnectionMethod{
			edi.ConnectionMethodSFTP,
			edi.ConnectionMethodVAN,
		})).
		Where("ecp.edi_partner_id IS NOT NULL").
		Where("COALESCE(TRIM(ecp.config->>'inboundDirectory'), '') <> ''").
		Where("COALESCE(ecp.last_poll_success_at, ecp.created_at) < ?", staleBefore).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (r *repository) GetActiveAS2ProfileByIdentifiers(
	ctx context.Context,
	req repositories.GetActiveAS2ProfileByIdentifiersRequest,
) (*edi.EDICommunicationProfile, error) {
	entity := new(edi.EDICommunicationProfile)
	cols := buncolgen.EDICommunicationProfileColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		Where(cols.Method.Eq(), edi.ConnectionMethodAS2).
		Where("ecp.edi_partner_id IS NOT NULL").
		Where("TRIM(ecp.config->>'localAS2Id') = ?", req.LocalAS2ID).
		Where("TRIM(ecp.config->>'partnerAS2Id') = ?", req.PartnerAS2ID).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDICommunicationProfile")
	}
	return entity, nil
}

func (r *repository) CreateProfile(
	ctx context.Context,
	entity *edi.EDICommunicationProfile,
) (*edi.EDICommunicationProfile, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateProfile(
	ctx context.Context,
	entity *edi.EDICommunicationProfile,
) (*edi.EDICommunicationProfile, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.EDICommunicationProfileColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Column(
			cols.EDIConnectionID.Bare(),
			cols.EDIPartnerID.Bare(),
			cols.Method.Bare(),
			cols.Status.Bare(),
			cols.Name.Bare(),
			cols.Description.Bare(),
			cols.Config.Bare(),
			cols.EncryptedSecrets.Bare(),
			cols.Version.Bare(),
			cols.UpdatedAt.Bare(),
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(
		results,
		"EDICommunicationProfile",
		entity.ID.String(),
	); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ListProfilesCursor(
	ctx context.Context,
	req *repositories.ListEDICommunicationProfilesRequest,
) (*pagination.CursorListResult[*edi.EDICommunicationProfile], error) {
	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*edi.EDICommunicationProfile)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFiltersWithoutSort(
				sq,
				"ecp",
				req.Filter,
				(*edi.EDICommunicationProfile)(nil),
			)
		}).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*edi.EDICommunicationProfile]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: &total,
		Query: func(entities *[]*edi.EDICommunicationProfile) *bun.SelectQuery {
			return dba.
				NewSelect().
				Model(entities).
				ColumnExpr(buncolgen.EDICommunicationProfileTable.All()).
				Relation(buncolgen.EDICommunicationProfileRelations.Partner)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				"ecp",
				req.Filter,
				req.Cursor,
				(*edi.EDICommunicationProfile)(nil),
			)
		},
	})
}
