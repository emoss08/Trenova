package edirepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
)

func (r *repository) ListProfiles(
	ctx context.Context,
	req *repositories.ListEDICommunicationProfilesRequest,
) (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
	entities := make([]*edi.EDICommunicationProfile, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("Partner").
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return querybuilder.ApplyFilters(query, "ecp", req.Filter, (*edi.EDICommunicationProfile)(nil))
		}).
		Where("ecp.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("ecp.business_unit_id = ?", req.Filter.TenantInfo.BuID).
		Order("ecp.created_at DESC").
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

func (r *repository) GetProfileByID(
	ctx context.Context,
	req repositories.GetEDICommunicationProfileByIDRequest,
) (*edi.EDICommunicationProfile, error) {
	entity := new(edi.EDICommunicationProfile)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Partner").
		Where("ecp.id = ?", req.ID).
		Where("ecp.organization_id = ?", req.TenantInfo.OrgID).
		Where("ecp.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDICommunicationProfile")
	}

	return entity, nil
}

func (r *repository) GetActiveProfileByPartner(
	ctx context.Context,
	req repositories.GetActiveEDICommunicationProfileByPartnerRequest,
) (*edi.EDICommunicationProfile, error) {
	entity := new(edi.EDICommunicationProfile)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ecp.edi_partner_id = ?", req.PartnerID).
		Where("ecp.organization_id = ?", req.TenantInfo.OrgID).
		Where("ecp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("ecp.method = ?", req.Method).
		Where("ecp.status = ?", domaintypes.StatusActive).
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
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
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

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Column(
			"edi_connection_id",
			"edi_partner_id",
			"method",
			"status",
			"name",
			"description",
			"config",
			"encrypted_secrets",
			"version",
			"updated_at",
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDICommunicationProfile", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
