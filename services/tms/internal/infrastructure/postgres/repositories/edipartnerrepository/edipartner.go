package edipartnerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports"
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

func New(p Params) repositories.EDIPartnerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-partner-repository"),
	}
}

func (r *repository) filterPartnersQuery(
	q *bun.SelectQuery,
	req *repositories.ListEDIPartnersRequest,
) *bun.SelectQuery {
	cols := buncolgen.EDIPartnerColumns
	rel := buncolgen.EDIPartnerRelations

	q = querybuilder.ApplyFilters(
		q,
		"ep",
		req.Filter,
		(*edi.EDIPartner)(nil),
	)

	return q.
		Relation(rel.InternalOrganization).
		Relation(rel.Connection).
		Relation(rel.DefaultTransport).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		Order(cols.CreatedAt.OrderDesc())
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
	col := buncolgen.EDIPartnerColumns

	return dbhelper.SelectOptions[*edi.EDIPartner](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			ColumnRefs: []buncolgen.Column{
				col.ID,
				col.BusinessUnitID,
				col.OrganizationID,
				col.Kind,
				col.Code,
				col.Name,
				col.InternalOrganizationID,
				col.EDIConnectionID,
				col.DefaultTransportID,
				col.EnabledForInbound,
				col.EnabledForOutbound,
			},
			OrgColumnRef:     &col.OrganizationID,
			BuColumnRef:      &col.BusinessUnitID,
			SearchColumnRefs: []buncolgen.Column{col.Name, col.Code},
			EntityName:       "EDIPartner",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				if req.Kind != "" {
					q = q.Where(col.Kind.Eq(), req.Kind)
				}

				if req.EnabledForOutbound {
					q = q.Where(col.EnabledForOutbound.IsTrue())
				}

				return q
			},
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetEDIPartnerByIDRequest,
) (*edi.EDIPartner, error) {
	entity := new(edi.EDIPartner)
	cols := buncolgen.EDIPartnerColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.EDIPartnerApplyTenant(req.TenantInfo)).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartner")
	}

	return entity, nil
}

func (r *repository) Create(ctx context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(c).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); err != nil {
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

func (r *repository) ensureMappingProfile(
	ctx context.Context,
	partner *edi.EDIPartner,
) (*edi.EDIMappingProfile, error) {
	profile := new(edi.EDIMappingProfile)
	cols := buncolgen.EDIMappingProfileColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(profile).
		Relation(buncolgen.EDIMappingProfileRelations.Entries).
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
}

func (r *repository) Update(ctx context.Context, entity *edi.EDIPartner) (*edi.EDIPartner, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.EDIPartnerColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Column(
			cols.Kind.Bare(),
			cols.Status.Bare(),
			cols.Code.Bare(),
			cols.Name.Bare(),
			cols.Description.Bare(),
			cols.InternalOrganizationID.Bare(),
			cols.EDIConnectionID.Bare(),
			cols.CustomerID.Bare(),
			cols.DefaultTransportID.Bare(),
			cols.DefaultMappingProfileID.Bare(),
			cols.DefaultValidationProfileID.Bare(),
			cols.Timezone.Bare(),
			cols.Country.Bare(),
			cols.ContactName.Bare(),
			cols.ContactEmail.Bare(),
			cols.ContactPhone.Bare(),
			cols.EnabledForInbound.Bare(),
			cols.EnabledForOutbound.Bare(),
			cols.Settings.Bare(),
			cols.Version.Bare(),
			cols.UpdatedAt.Bare(),
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
	cols := buncolgen.EDIPartnerColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Join(`JOIN "organizations" AS "target_org"`).
		JoinOn(buncolgen.OrganizationColumns.ID.WithAlias("target_org").EqColumn(cols.OrganizationID)).
		JoinOn(
			buncolgen.OrganizationColumns.BusinessUnitID.WithAlias("target_org").
				EqColumn(cols.BusinessUnitID),
		).
		Where(cols.OrganizationID.Eq(), req.TargetOrganizationID).
		Where(cols.BusinessUnitID.Eq(), req.BusinessUnitID).
		Where(cols.InternalOrganizationID.Eq(), req.SourceOrganizationID).
		Where(cols.Kind.Eq(), edi.PartnerKindInternal).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		Where(cols.EnabledForInbound.IsTrue()).
		Order(cols.CreatedAt.OrderAsc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "ReciprocalEDIPartner")
	}

	return entity, nil
}
