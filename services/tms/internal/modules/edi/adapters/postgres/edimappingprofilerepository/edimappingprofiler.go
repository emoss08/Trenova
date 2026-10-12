//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edimappingprofilerepository

import (
	"context"

	"github.com/emoss08/trenova/pkg/buncolgen/edicol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
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

func New(p Params) ediports.EDIMappingProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-mapping-profile-repository"),
	}
}

func (r *repository) GetMappingProfile(
	ctx context.Context,
	req ediports.GetMappingProfileRequest,
) (*edi.EDIMappingProfile, error) {
	partner, err := r.getPartnerByID(ctx, ediports.GetEDIPartnerByIDRequest{
		ID:         req.PartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return r.ensureMappingProfile(ctx, partner)
}

func (r *repository) getPartnerByID(
	ctx context.Context,
	req ediports.GetEDIPartnerByIDRequest,
) (*edi.EDIPartner, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIPartner, error) {
		entity := new(edi.EDIPartner)
		cols := edicol.EDIPartnerColumns
		rel := edicol.EDIPartnerRelations

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Relation(rel.InternalOrganization).
			Relation(rel.Connection).
			Relation(rel.DefaultTransport).
			Where(cols.ID.Eq(), req.ID).
			Apply(edicol.EDIPartnerApplyTenant(req.TenantInfo)).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "EDIPartner")
		}

		return entity, nil
	})
}

func (r *repository) ListMappingProfiles(
	ctx context.Context,
	req *ediports.ListEDIMappingProfilesRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
			entities := make([]*edi.EDIMappingProfile, 0, req.Filter.Pagination.SafeLimit())
			cols := edicol.EDIMappingProfileColumns
			rel := edicol.EDIMappingProfileRelations

			total, err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&entities).
				Relation(rel.Partner).
				Relation(rel.Entries).
				Apply(edicol.EDIMappingProfileApplyTenant(req.Filter.TenantInfo)).
				Limit(int64(req.Filter.Pagination.SafeLimit())).
				Offset(int64(req.Filter.Pagination.SafeOffset())).
				Order(cols.CreatedAt.OrderDesc()).
				ScanAndCount(ctx)
			if err != nil {
				return nil, err
			}

			return &pagination.ListResult[*edi.EDIMappingProfile]{
				Items: entities,
				Total: int(total),
			}, nil
		},
	)
}

func (r *repository) ListMappingProfilesCursor(
	ctx context.Context,
	req *ediports.ListEDIMappingProfilesRequest,
) (*pagination.CursorListResult[*edi.EDIMappingProfile], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDIMappingProfile], error) {
			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*edi.EDIMappingProfile)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						sq = querybuilder.ApplyFiltersWithoutSort(
							sq,
							"emp",
							req.Filter,
							(*edi.EDIMappingProfile)(nil),
						)
						return applyMappingProfileListFilters(sq, req)
					}).
					Count(ctx)
				if err != nil {
					return nil, err
				}
				totalCount = new(int(total))
			}

			return dbhelper.CursorList(ctx, dbhelper.CursorListParams[*edi.EDIMappingProfile]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*edi.EDIMappingProfile) *bun.SelectQuery {
					rel := edicol.EDIMappingProfileRelations
					return dba.
						NewSelect().
						Model(entities).
						ColumnExpr(edicol.EDIMappingProfileTable.All()).
						Relation(rel.Partner).
						Relation(rel.Entries)
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					sq, applyErr := querybuilder.ApplyCursorFilters(
						sq,
						"emp",
						req.Filter,
						req.Cursor,
						(*edi.EDIMappingProfile)(nil),
					)
					if applyErr != nil {
						return sq, applyErr
					}
					return applyMappingProfileListFilters(sq, req), nil
				},
			})
		},
	)
}

func applyMappingProfileListFilters(
	q *bun.SelectQuery,
	req *ediports.ListEDIMappingProfilesRequest,
) *bun.SelectQuery {
	if !req.PartnerID.IsNil() {
		q = q.Where(edicol.EDIMappingProfileColumns.EDIPartnerID.Eq(), req.PartnerID)
	}
	return q
}

func (r *repository) SelectMappingProfileOptions(
	ctx context.Context,
	req *ediports.EDIMappingProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
			col := edicol.EDIMappingProfileColumns

			return dbhelper.SelectOptions[*edi.EDIMappingProfile](
				ctx,
				r.db.DBForContext(ctx),
				req.SelectQueryRequest,
				&dbhelper.SelectOptionsConfig{
					ColumnRefs: []buncolgen.Column{
						col.ID,
						col.BusinessUnitID,
						col.OrganizationID,
						col.EDIPartnerID,
						col.Name,
						col.Description,
					},
					OrgColumnRef:     &col.OrganizationID,
					BuColumnRef:      &col.BusinessUnitID,
					SearchColumnRefs: []buncolgen.Column{col.Name, col.Description},
					QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
						if req.PartnerID.IsNotNil() {
							q = q.Where(col.EDIPartnerID.Eq(), req.PartnerID)
						}

						q = q.Relation(edicol.EDIMappingProfileRelations.Partner)

						return q
					},
				},
			)
		},
	)
}

func (r *repository) GetMappingProfileByID(
	ctx context.Context,
	req ediports.GetMappingProfileByIDRequest,
) (*edi.EDIMappingProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIMappingProfile, error) {
		profile := new(edi.EDIMappingProfile)
		rel := edicol.EDIMappingProfileRelations

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(profile).
			Relation(rel.Partner).
			Relation(rel.Entries).
			Where(edicol.EDIMappingProfileColumns.ID.Eq(), req.ProfileID).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return edicol.EDIMappingProfileScopeTenant(sq, req.TenantInfo)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "EDIMappingProfile")
		}

		return profile, nil
	})
}

func (r *repository) SaveMappingItems(
	ctx context.Context,
	req *ediports.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*edi.EDIMappingProfileItem, error) {
		partner, err := r.getPartnerByID(ctx, ediports.GetEDIPartnerByIDRequest{
			ID:         req.PartnerID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}

		profile, err := r.ensureMappingProfile(ctx, partner)
		if err != nil {
			return nil, err
		}

		items := make([]*edi.EDIMappingProfileItem, 0, len(req.Items))
		for _, item := range req.Items {
			if item == nil {
				continue
			}
			item.BusinessUnitID = req.TenantInfo.BuID
			item.OrganizationID = req.TenantInfo.OrgID
			item.EDIPartnerID = req.PartnerID
			item.MappingProfileID = profile.ID
			item.UpdatedByID = req.ActorID
			if item.CreatedByID.IsNil() {
				item.CreatedByID = req.ActorID
			}
			items = append(items, item)
		}

		if len(items) == 0 {
			return []*edi.EDIMappingProfileItem{}, nil
		}
		itemCols := edicol.EDIMappingProfileItemColumns

		_, err = r.db.DBForContext(ctx).
			NewInsert().
			Model(&items).
			On(`CONFLICT ("edi_partner_id", "business_unit_id", "organization_id", "entity_type", "source_id") DO UPDATE`).
			Set(itemCols.TargetID.SetExcluded()).
			Set(itemCols.TargetLabel.SetExcluded()).
			Set(itemCols.SourceLabel.SetExcluded()).
			Set(itemCols.UpdatedByID.SetExcluded()).
			Set(itemCols.UpdatedAt.SetExpr(dbhelper.NowEpoch)).
			Set(itemCols.Version.SetExpr(itemCols.Version.Qualified() + " + 1")).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, err
		}

		return items, nil
	})
}

func (r *repository) SaveMappingProfileItems(
	ctx context.Context,
	req *ediports.SaveMappingProfileItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	profile, err := r.GetMappingProfileByID(ctx, ediports.GetMappingProfileByIDRequest{
		ProfileID:  req.ProfileID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return r.SaveMappingItems(ctx, &ediports.SaveMappingItemsRequest{
		PartnerID:  profile.EDIPartnerID,
		TenantInfo: req.TenantInfo,
		ActorID:    req.ActorID,
		Items:      req.Items,
	})
}

func (r *repository) DeleteMappingItem(
	ctx context.Context,
	req ediports.DeleteMappingItemRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		results, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*edi.EDIMappingProfileItem)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				cols := edicol.EDIMappingProfileItemColumns
				return edicol.EDIMappingProfileItemScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.MappingItemID).
					Where(cols.EDIPartnerID.Eq(), req.PartnerID)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		return dberror.CheckRowsAffected(
			results,
			"EDIMappingProfileItem",
			req.MappingItemID.String(),
		)
	})
}

func (r *repository) DeleteMappingProfileItem(
	ctx context.Context,
	req ediports.DeleteMappingProfileItemRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		results, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*edi.EDIMappingProfileItem)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				cols := edicol.EDIMappingProfileItemColumns
				return edicol.EDIMappingProfileItemScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.MappingItemID).
					Where(cols.MappingProfileID.Eq(), req.ProfileID)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		return dberror.CheckRowsAffected(
			results,
			"EDIMappingProfileItem",
			req.MappingItemID.String(),
		)
	})
}

func (r *repository) GetMappingItems(
	ctx context.Context,
	req ediports.GetMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIMappingProfileItem, error) {
		items := make([]*edi.EDIMappingProfileItem, 0, len(req.SourceIDs))
		if len(req.SourceIDs) == 0 {
			return items, nil
		}
		cols := edicol.EDIMappingProfileItemColumns

		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			Apply(edicol.EDIMappingProfileItemApplyTenant(req.TenantInfo)).
			Where(cols.EDIPartnerID.Eq(), req.PartnerID).
			Where(cols.SourceID.In(), bun.List(req.SourceIDs))

		if len(req.EntityTypes) > 0 {
			query = query.Where(cols.EntityType.In(), bun.List(req.EntityTypes))
		}

		if err := query.Scan(ctx); err != nil {
			return nil, err
		}

		return items, nil
	})
}

func (r *repository) ensureMappingProfile(
	ctx context.Context,
	partner *edi.EDIPartner,
) (*edi.EDIMappingProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIMappingProfile, error) {
		profile := new(edi.EDIMappingProfile)
		cols := edicol.EDIMappingProfileColumns

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(profile).
			Relation(edicol.EDIMappingProfileRelations.Entries).
			Where(cols.OrganizationID.Eq(), partner.OrganizationID).
			Where(cols.BusinessUnitID.Eq(), partner.BusinessUnitID).
			Where(cols.EDIPartnerID.Eq(), partner.ID).
			Scan(ctx)
		if err == nil {
			return profile, nil
		}
		if !dberror.IsNotFoundError(err) {
			return nil, err
		}

		profile = &edi.EDIMappingProfile{
			BusinessUnitID: partner.BusinessUnitID,
			OrganizationID: partner.OrganizationID,
			EDIPartnerID:   partner.ID,
			Name:           partner.Name + " Mapping Profile",
		}

		if _, err = r.db.DBForContext(ctx).
			NewInsert().
			Model(profile).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, err
		}

		return profile, nil
	})
}
