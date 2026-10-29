//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edimappingprofilerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediscope"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	partnerErrors = dberror.Mapping{Entity: "EDIPartner"}
	profileErrors = dberror.Mapping{Entity: "EDIMappingProfile"}
	itemErrors    = dberror.Mapping{Entity: "EDIMappingProfileItem"}
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

func New(p Params) edipartnerports.EDIMappingProfileRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-mapping-profile-repository"),
	}
}

func (r *repository) GetMappingProfile(
	ctx context.Context,
	req edipartnerports.GetMappingProfileRequest,
) (*edi.EDIMappingProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDIMappingProfile, error) {
		partner, err := r.getPartnerByID(ctx, req.TenantInfo, req.PartnerID)
		if err != nil {
			return nil, err
		}

		return r.ensureMappingProfile(ctx, partner)
	})
}

func (r *repository) getPartnerByID(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (*edi.EDIPartner, error) {
	ep := edidb.EDIPartners
	entity, err := ediscope.Tenant(ctx, ep.Where(ep.ID.Eq(id)), tenant).
		Load(ep.InternalOrganization(), ep.Connection(), ep.DefaultTransport()).
		One(ctx, r.db.Keel(ctx))
	if err != nil {
		return nil, partnerErrors.Map(err, id.String())
	}

	return entity, nil
}

func (r *repository) ListMappingProfiles(
	ctx context.Context,
	req *edipartnerports.ListEDIMappingProfilesRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
			mp := edidb.EDIMappingProfiles
			q := withPartnerAndEntries(mp.Query()).Order(mp.CreatedAt.Desc())

			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func (r *repository) ListMappingProfilesCursor(
	ctx context.Context,
	req *edipartnerports.ListEDIMappingProfilesRequest,
) (*pagination.CursorListResult[*edi.EDIMappingProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDIMappingProfile], error) {
			mp := edidb.EDIMappingProfiles
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				mp.Where(keel.When(req.PartnerID.IsNotNil(), mp.EDIPartnerID.Eq(req.PartnerID))),
				&listquery.Request[edi.EDIMappingProfile]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withPartnerAndEntries,
				},
			)
		},
	)
}

func (r *repository) SelectMappingProfileOptions(
	ctx context.Context,
	req *edipartnerports.EDIMappingProfileSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDIMappingProfile], error) {
			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				mappingProfileOptionsQuery(req),
				req.SelectQueryRequest,
				mappingProfileSearch,
			)
		},
	)
}

func mappingProfileOptionsQuery(
	req *edipartnerports.EDIMappingProfileSelectOptionsRequest,
) *keel.Query[*edi.EDIMappingProfile] {
	mp := edidb.EDIMappingProfiles
	return mp.Pick(
		mp.ID,
		mp.BusinessUnitID,
		mp.OrganizationID,
		mp.EDIPartnerID,
		mp.Name,
		mp.Description,
	).
		Where(keel.When(req.PartnerID.IsNotNil(), mp.EDIPartnerID.Eq(req.PartnerID))).
		Load(mp.Partner())
}

func mappingProfileSearch(s string) keel.Pred {
	mp := edidb.EDIMappingProfiles
	return keel.Or(mp.Name.ContainsFold(s), mp.Description.ContainsFold(s))
}

func (r *repository) GetMappingProfileByID(
	ctx context.Context,
	req edipartnerports.GetMappingProfileByIDRequest,
) (*edi.EDIMappingProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDIMappingProfile, error) {
		mp := edidb.EDIMappingProfiles
		profile, err := withPartnerAndEntries(mp.Where(mp.ID.Eq(req.ProfileID))).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, req.ProfileID.String())
		}

		return profile, nil
	})
}

func mappingProfilesByIDs(ids []pulid.ID) *keel.Query[*edi.EDIMappingProfile] {
	mp := edidb.EDIMappingProfiles

	return mp.Where(mp.ID.In(ids...)).Limit(len(ids))
}

func (r *repository) GetMappingProfilesByIDs(
	ctx context.Context,
	req edipartnerports.GetMappingProfilesByIDsRequest,
) ([]*edi.EDIMappingProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIMappingProfile, error) {
		if len(req.ProfileIDs) == 0 {
			return []*edi.EDIMappingProfile{}, nil
		}

		entities, err := mappingProfilesByIDs(req.ProfileIDs).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get edi mapping profiles", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) SaveMappingItems(
	ctx context.Context,
	req *edipartnerports.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*edi.EDIMappingProfileItem, error) {
		partner, err := r.getPartnerByID(ctx, req.TenantInfo, req.PartnerID)
		if err != nil {
			return nil, err
		}

		profile, err := r.ensureMappingProfile(ctx, partner)
		if err != nil {
			return nil, err
		}

		return r.upsertItems(ctx, profile, req)
	})
}

func (r *repository) upsertItems(
	ctx context.Context,
	profile *edi.EDIMappingProfile,
	req *edipartnerports.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	items := make([]*edi.EDIMappingProfileItem, 0, len(req.Items))
	sourceIDs := make([]pulid.ID, 0, len(req.Items))
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
		sourceIDs = append(sourceIDs, item.SourceID)
	}
	if len(items) == 0 {
		return []*edi.EDIMappingProfileItem{}, nil
	}

	db := r.db.Keel(ctx)
	mi := edidb.EDIMappingProfileItems
	_, err := mi.Upsert(ctx, db, items, keel.OnConflict(
		mi.EDIPartnerID,
		mi.BusinessUnitID,
		mi.OrganizationID,
		mi.EntityType,
		mi.SourceID,
	).Update(mi.TargetID, mi.TargetLabel, mi.SourceLabel, mi.UpdatedByID))
	if err != nil {
		return nil, itemErrors.Map(err, req.PartnerID.String())
	}

	stored, err := ediscope.Tenant(
		ctx,
		mi.Where(mi.EDIPartnerID.Eq(req.PartnerID), mi.SourceID.In(sourceIDs...)),
		req.TenantInfo,
	).All(ctx, db)
	if err != nil {
		return nil, err
	}

	type itemKey struct {
		entityType edi.MappingEntityType
		sourceID   pulid.ID
	}
	byKey := make(map[itemKey]*edi.EDIMappingProfileItem, len(stored))
	for _, row := range stored {
		byKey[itemKey{entityType: row.EntityType, sourceID: row.SourceID}] = row
	}

	saved := make([]*edi.EDIMappingProfileItem, 0, len(items))
	for _, item := range items {
		if row, ok := byKey[itemKey{entityType: item.EntityType, sourceID: item.SourceID}]; ok {
			*item = *row
		}
		saved = append(saved, item)
	}

	return saved, nil
}

func (r *repository) SaveMappingProfileItems(
	ctx context.Context,
	req *edipartnerports.SaveMappingProfileItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*edi.EDIMappingProfileItem, error) {
		profile, err := r.GetMappingProfileByID(ctx, edipartnerports.GetMappingProfileByIDRequest{
			ProfileID:  req.ProfileID,
			TenantInfo: req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}

		return r.SaveMappingItems(ctx, &edipartnerports.SaveMappingItemsRequest{
			PartnerID:  profile.EDIPartnerID,
			TenantInfo: req.TenantInfo,
			ActorID:    req.ActorID,
			Items:      req.Items,
		})
	})
}

func (r *repository) DeleteMappingItem(
	ctx context.Context,
	req edipartnerports.DeleteMappingItemRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		mi := edidb.EDIMappingProfileItems
		return r.deleteItem(
			ctx,
			req.MappingItemID,
			mi.Where(mi.ID.Eq(req.MappingItemID), mi.EDIPartnerID.Eq(req.PartnerID)),
		)
	})
}

func (r *repository) DeleteMappingProfileItem(
	ctx context.Context,
	req edipartnerports.DeleteMappingProfileItemRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		mi := edidb.EDIMappingProfileItems
		return r.deleteItem(
			ctx,
			req.MappingItemID,
			mi.Where(mi.ID.Eq(req.MappingItemID), mi.MappingProfileID.Eq(req.ProfileID)),
		)
	})
}

func (r *repository) deleteItem(
	ctx context.Context,
	id pulid.ID,
	q *keel.Query[*edi.EDIMappingProfileItem],
) error {
	n, err := q.Delete(ctx, r.db.Keel(ctx))
	if err != nil {
		return itemErrors.Map(err, id.String())
	}

	return dberror.CheckAffected(n, "EDIMappingProfileItem", id.String())
}

func (r *repository) GetMappingItems(
	ctx context.Context,
	req edipartnerports.GetMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDIMappingProfileItem, error) {
		if len(req.SourceIDs) == 0 {
			return []*edi.EDIMappingProfileItem{}, nil
		}

		mi := edidb.EDIMappingProfileItems
		q := mi.Where(
			mi.EDIPartnerID.Eq(req.PartnerID),
			mi.SourceID.In(req.SourceIDs...),
			keel.When(len(req.EntityTypes) > 0, mi.EntityType.In(req.EntityTypes...)),
		)

		return ediscope.Tenant(ctx, q, req.TenantInfo).All(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) GetMappingSummary(
	ctx context.Context,
	req edipartnerports.GetEDIMappingSummaryRequest,
) (*edipartnerports.EDIMappingSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*edipartnerports.EDIMappingSummary, error) {
			db := r.db.Keel(ctx)
			summary := &edipartnerports.EDIMappingSummary{
				Counts: make([]*edipartnerports.EDIMappingEntityCount, 0, 8),
			}

			mp := edidb.EDIMappingProfiles
			profile, err := mp.Pick(mp.ID, mp.UpdatedAt).
				Where(mp.EDIPartnerID.Eq(req.PartnerID)).
				First(ctx, db)
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return summary, nil
				}
				return nil, err
			}
			summary.ProfileID = profile.ID
			summary.UpdatedAt = profile.UpdatedAt

			mi := edidb.EDIMappingProfileItems
			counts, err := keel.Into[edipartnerports.EDIMappingEntityCount](
				ctx,
				db,
				mi.Where(mi.EDIPartnerID.Eq(req.PartnerID)).
					Group(mi.EntityType).
					Pick(mi.EntityType, keel.Count(), mi.UpdatedAt.Max()).
					Order(mi.EntityType.Asc()),
			)
			if err != nil {
				return nil, err
			}
			for i := range counts {
				summary.Counts = append(summary.Counts, &counts[i])
				summary.UpdatedAt = max(summary.UpdatedAt, counts[i].UpdatedAt)
			}

			return summary, nil
		},
	)
}

func (r *repository) ensureMappingProfile(
	ctx context.Context,
	partner *edi.EDIPartner,
) (*edi.EDIMappingProfile, error) {
	db := r.db.Keel(ctx)
	mp := edidb.EDIMappingProfiles
	tenant := pagination.TenantInfo{OrgID: partner.OrganizationID, BuID: partner.BusinessUnitID}
	profile, err := ediscope.Tenant(ctx, mp.Where(mp.EDIPartnerID.Eq(partner.ID)), tenant).
		Load(mp.Entries()).
		One(ctx, db)
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
	if err = mp.Create(ctx, db, profile); err != nil {
		return nil, profileErrors.Map(err, profile.ID.String())
	}

	return profile, nil
}

func withPartnerAndEntries(
	q *keel.Query[*edi.EDIMappingProfile],
) *keel.Query[*edi.EDIMappingProfile] {
	mp := edidb.EDIMappingProfiles
	return q.Load(mp.Partner(), mp.Entries())
}
