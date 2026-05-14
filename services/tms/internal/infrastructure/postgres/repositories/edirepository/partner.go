package edirepository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.EDIPartnerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-repository"),
	}
}

func NewTransferRepository(p Params) repositories.EDILoadTenderTransferRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transfer-repository"),
	}
}

func (r *repository) filterPartnersQuery(
	q *bun.SelectQuery,
	req *repositories.ListEDIPartnersRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"ep",
		req.Filter,
		(*edi.EDIPartner)(nil),
	)

	return q.
		Relation("InternalOrganization").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order("ep.created_at DESC")
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListEDIPartnersRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	entities := make([]*edi.EDIPartner, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterPartnersQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIPartner]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.EDIPartnerSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDIPartner], error) {
	entities := make([]*edi.EDIPartner, 0, req.SelectQueryRequest.Pagination.SafeLimit())

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Column(
			"id",
			"business_unit_id",
			"organization_id",
			"kind",
			"status",
			"code",
			"name",
			"internal_organization_id",
			"enabled_for_inbound",
			"enabled_for_outbound",
		).
		Where("ep.organization_id = ?", req.SelectQueryRequest.TenantInfo.OrgID).
		Where("ep.business_unit_id = ?", req.SelectQueryRequest.TenantInfo.BuID).
		Where("ep.status = ?", domaintypes.StatusActive)

	if req.Kind != "" {
		query = query.Where("ep.kind = ?", req.Kind)
	}
	if req.EnabledForOutbound {
		query = query.Where("ep.enabled_for_outbound = TRUE")
	}
	query = applyPartnerSearch(query, req.SelectQueryRequest.Query)

	total, err := query.
		Order("ep.name ASC").
		Limit(req.SelectQueryRequest.Pagination.SafeLimit()).
		Offset(req.SelectQueryRequest.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIPartner]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetEDIPartnerByIDRequest,
) (*edi.EDIPartner, error) {
	entity := new(edi.EDIPartner)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ep.id = ?", req.ID).
		Where("ep.organization_id = ?", req.TenantInfo.OrgID).
		Where("ep.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartner")
	}

	return entity, nil
}

func (r *repository) Create(ctx context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(c).NewInsert().Model(entity).Returning("*").Exec(c); err != nil {
			return err
		}
		_, err := r.ensureMappingProfile(c, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) CreateInternalPair(
	ctx context.Context,
	req *repositories.CreateInternalPartnerPairRequest,
) (*edi.InternalPartnerPair, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if err := r.ensureTargetOrganizationInBusinessUnit(c, req.TargetOrganizationID, req.BusinessUnitID); err != nil {
			return err
		}

		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(req.SourcePartner).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		if _, err := r.ensureMappingProfile(c, req.SourcePartner); err != nil {
			return err
		}

		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(req.TargetPartner).
			Returning("*").
			Exec(c); err != nil {
			return err
		}
		_, err := r.ensureMappingProfile(c, req.TargetPartner)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &edi.InternalPartnerPair{
		SourcePartner: req.SourcePartner,
		TargetPartner: req.TargetPartner,
	}, nil
}

func (r *repository) Update(ctx context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Column(
			"kind",
			"status",
			"code",
			"name",
			"description",
			"internal_organization_id",
			"customer_id",
			"default_transport_id",
			"default_mapping_profile_id",
			"default_validation_profile_id",
			"timezone",
			"country",
			"contact_name",
			"contact_email",
			"contact_phone",
			"enabled_for_inbound",
			"enabled_for_outbound",
			"settings",
			"version",
			"updated_at",
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "EDIPartner", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) GetReciprocalInternalPartner(
	ctx context.Context,
	req repositories.GetReciprocalInternalPartnerRequest,
) (*edi.EDIPartner, error) {
	entity := new(edi.EDIPartner)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Join(`JOIN "organizations" AS "target_org"`).
		JoinOn("target_org.id = ep.organization_id").
		JoinOn("target_org.business_unit_id = ep.business_unit_id").
		Where("ep.organization_id = ?", req.TargetOrganizationID).
		Where("ep.business_unit_id = ?", req.BusinessUnitID).
		Where("ep.internal_organization_id = ?", req.SourceOrganizationID).
		Where("ep.kind = ?", edi.PartnerKindInternal).
		Where("ep.status = ?", domaintypes.StatusActive).
		Where("ep.enabled_for_inbound = TRUE").
		Order("ep.created_at ASC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ReciprocalEDIPartner")
	}

	return entity, nil
}

func (r *repository) GetMappingProfile(
	ctx context.Context,
	req repositories.GetMappingProfileRequest,
) (*edi.EDIMappingProfile, error) {
	partner, err := r.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
		ID:         req.PartnerID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return r.ensureMappingProfile(ctx, partner)
}

func (r *repository) SaveMappingItems(
	ctx context.Context,
	req *repositories.SaveMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	partner, err := r.GetByID(ctx, repositories.GetEDIPartnerByIDRequest{
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

	_, err = r.db.DBForContext(ctx).
		NewInsert().
		Model(&items).
		On(`CONFLICT ("edi_partner_id", "business_unit_id", "organization_id", "entity_type", "source_id") DO UPDATE`).
		Set("target_id = EXCLUDED.target_id").
		Set("target_label = EXCLUDED.target_label").
		Set("source_label = EXCLUDED.source_label").
		Set("updated_by_id = EXCLUDED.updated_by_id").
		Set("updated_at = extract(epoch FROM current_timestamp)::bigint").
		Set("version = edi_mapping_profile_items.version + 1").
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) DeleteMappingItem(
	ctx context.Context,
	req repositories.DeleteMappingItemRequest,
) error {
	results, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*edi.EDIMappingProfileItem)(nil)).
		Where("id = ?", req.MappingItemID).
		Where("edi_partner_id = ?", req.PartnerID).
		Where("organization_id = ?", req.TenantInfo.OrgID).
		Where("business_unit_id = ?", req.TenantInfo.BuID).
		Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(results, "EDIMappingProfileItem", req.MappingItemID.String())
}

func (r *repository) GetMappingItems(
	ctx context.Context,
	req repositories.GetMappingItemsRequest,
) ([]*edi.EDIMappingProfileItem, error) {
	items := make([]*edi.EDIMappingProfileItem, 0, len(req.SourceIDs))
	if len(req.SourceIDs) == 0 {
		return items, nil
	}

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&items).
		Where("empi.organization_id = ?", req.TenantInfo.OrgID).
		Where("empi.business_unit_id = ?", req.TenantInfo.BuID).
		Where("empi.edi_partner_id = ?", req.PartnerID).
		Where("empi.source_id IN (?)", bun.In(req.SourceIDs))

	if len(req.EntityTypes) > 0 {
		query = query.Where("empi.entity_type IN (?)", bun.In(req.EntityTypes))
	}

	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) ensureMappingProfile(
	ctx context.Context,
	partner *edi.EDIPartner,
) (*edi.EDIMappingProfile, error) {
	profile := new(edi.EDIMappingProfile)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(profile).
		Relation("Entries").
		Where("emp.organization_id = ?", partner.OrganizationID).
		Where("emp.business_unit_id = ?", partner.BusinessUnitID).
		Where("emp.edi_partner_id = ?", partner.ID).
		Scan(ctx)
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	profile = &edi.EDIMappingProfile{
		BusinessUnitID: partner.BusinessUnitID,
		OrganizationID: partner.OrganizationID,
		EDIPartnerID:   partner.ID,
		Name:           partner.Name + " Mapping Profile",
	}

	if _, err = r.db.DBForContext(ctx).NewInsert().Model(profile).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *repository) ensureTargetOrganizationInBusinessUnit(
	ctx context.Context,
	targetOrganizationID pulid.ID,
	businessUnitID pulid.ID,
) error {
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		Table("organizations").
		Where("id = ?", targetOrganizationID).
		Where("business_unit_id = ?", businessUnitID).
		Exists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return errortypes.NewValidationError(
		"targetOrganizationId",
		errortypes.ErrInvalid,
		"Target organization must belong to the current business unit",
	)
}

func applyPartnerSearch(query *bun.SelectQuery, search string) *bun.SelectQuery {
	search = strings.TrimSpace(search)
	if search == "" {
		return query
	}

	term := "%" + strings.ToLower(search) + "%"
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("lower(ep.code) LIKE ?", term).
			WhereOr("lower(ep.name) LIKE ?", term)
	})
}
