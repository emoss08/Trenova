//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package ediconnectionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

func New(p Params) repositories.EDIConnectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-connection-repository"),
	}
}

func (r *repository) ListConnections(
	ctx context.Context,
	req *repositories.ListEDIConnectionsRequest,
) (*pagination.ListResult[*edi.EDIConnection], error) {
	entities := make([]*edi.EDIConnection, 0, req.Filter.Pagination.SafeLimit())
	rel := buncolgen.EDIConnectionRelations

	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation(rel.SourceOrganization).
		Relation(rel.TargetOrganization).
		Relation(rel.SourcePartner).
		Relation(rel.TargetPartner).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterConnectionsQuery(query, req)
		}).
		Order(buncolgen.EDIConnectionColumns.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*edi.EDIConnection]{
		Items: entities,
		Total: total,
	}, nil
}

func filterConnectionsQuery(
	query *bun.SelectQuery,
	req *repositories.ListEDIConnectionsRequest,
) *bun.SelectQuery {
	query = applyConnectionTenantFilter(query, req.Filter.TenantInfo)
	if req.Filter.Query == "" {
		return query
	}

	term := "%" + req.Filter.Query + "%"
	cols := buncolgen.EDIConnectionColumns
	orgCols := buncolgen.OrganizationColumns

	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr(cols.Method.TextILike(), term).
			WhereOr(cols.Status.TextILike(), term).
			WhereOr(orgCols.Name.WithAlias("source_organization").ILike(), term).
			WhereOr(orgCols.Name.WithAlias("target_organization").ILike(), term)
	})
}

func (r *repository) GetConnectionByID(
	ctx context.Context,
	req repositories.GetEDIConnectionByIDRequest,
) (*edi.EDIConnection, error) {
	entity := new(edi.EDIConnection)
	cols := buncolgen.EDIConnectionColumns
	rel := buncolgen.EDIConnectionRelations

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation(rel.SourceOrganization).
		Relation(rel.TargetOrganization).
		Relation(rel.SourcePartner).
		Relation(rel.TargetPartner).
		Where(cols.ID.Eq(), req.ID).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return applyConnectionTenantFilter(query, req.TenantInfo)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIConnection")
	}

	return entity, nil
}

func (r *repository) GetConnectionForUpdate(
	ctx context.Context,
	req repositories.GetEDIConnectionForUpdateRequest,
) (*edi.EDIConnection, error) {
	entity := new(edi.EDIConnection)
	cols := buncolgen.EDIConnectionColumns

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return applyConnectionTenantFilter(query, req.TenantInfo)
		}).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIConnection")
	}

	return entity, nil
}

func (r *repository) GetActiveConnectionForPartner(
	ctx context.Context,
	req repositories.GetActiveEDIConnectionForPartnerRequest,
) (*edi.EDIConnection, error) {
	entity := new(edi.EDIConnection)
	cols := buncolgen.EDIConnectionColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Where(cols.Method.Eq(), req.Method).
		Where(cols.Status.Eq(), edi.ConnectionStatusActive).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(
				buncolgen.Expr("({0} = ? AND {1} = ?)", cols.SourceOrganizationID, cols.SourcePartnerID),
				req.TenantInfo.OrgID,
				req.PartnerID,
			).
				WhereOr(
					buncolgen.Expr("({0} = ? AND {1} = ?)", cols.TargetOrganizationID, cols.TargetPartnerID),
					req.TenantInfo.OrgID,
					req.PartnerID,
				)
		})

	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIConnection")
	}

	return entity, nil
}

func (r *repository) CreateConnection(
	ctx context.Context,
	entity *edi.EDIConnection,
) (*edi.EDIConnection, error) {
	if err := r.ensureTargetOrganizationInBusinessUnit(
		ctx,
		entity.TargetOrganizationID,
		entity.BusinessUnitID,
	); err != nil {
		return nil, err
	}

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) UpdateConnection(
	ctx context.Context,
	entity *edi.EDIConnection,
) (*edi.EDIConnection, error) {
	ov := entity.Version
	entity.Version++
	cols := buncolgen.EDIConnectionColumns

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDIConnection", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) ensureMappingProfile(
	ctx context.Context,
	partner *edi.EDIPartner,
) error {
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
		return nil
	}
	if !dberror.IsNotFoundError(err) {
		return err
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
		return err
	}

	return nil
}

func (r *repository) ensureTargetOrganizationInBusinessUnit(
	ctx context.Context,
	targetOrganizationID pulid.ID,
	businessUnitID pulid.ID,
) error {
	exists, err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr(buncolgen.OrganizationTable.As(buncolgen.OrganizationTable.Alias)).
		Where(buncolgen.OrganizationColumns.ID.Eq(), targetOrganizationID).
		Where(buncolgen.OrganizationColumns.BusinessUnitID.Eq(), businessUnitID).
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

func (r *repository) AcceptInternalConnection(
	ctx context.Context,
	req *repositories.CreateInternalEDIConnectionAcceptanceRequest,
) (*edi.EDIConnection, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(txCtx).
			NewInsert().
			Model(req.SourcePartner).
			Returning("*").
			Exec(txCtx); err != nil {
			return err
		}
		if err := r.ensureMappingProfile(txCtx, req.SourcePartner); err != nil {
			return err
		}

		if _, err := r.db.DBForContext(txCtx).
			NewInsert().
			Model(req.TargetPartner).
			Returning("*").
			Exec(txCtx); err != nil {
			return err
		}
		if err := r.ensureMappingProfile(txCtx, req.TargetPartner); err != nil {
			return err
		}

		req.SourceProfile.EDIPartnerID = req.SourcePartner.ID
		req.TargetProfile.EDIPartnerID = req.TargetPartner.ID
		if _, err := r.db.DBForContext(txCtx).
			NewInsert().
			Model(req.SourceProfile).
			Returning("*").
			Exec(txCtx); err != nil {
			return err
		}
		if _, err := r.db.DBForContext(txCtx).
			NewInsert().
			Model(req.TargetProfile).
			Returning("*").
			Exec(txCtx); err != nil {
			return err
		}

		req.SourcePartner.DefaultTransportID = req.SourceProfile.ID
		req.TargetPartner.DefaultTransportID = req.TargetProfile.ID
		if _, err := r.db.DBForContext(txCtx).
			NewUpdate().
			Model(req.SourcePartner).
			WherePK().
			Column(buncolgen.EDIPartnerColumns.DefaultTransportID.Bare()).
			Exec(txCtx); err != nil {
			return err
		}
		if _, err := r.db.DBForContext(txCtx).
			NewUpdate().
			Model(req.TargetPartner).
			WherePK().
			Column(buncolgen.EDIPartnerColumns.DefaultTransportID.Bare()).
			Exec(txCtx); err != nil {
			return err
		}

		req.Connection.SourcePartnerID = req.SourcePartner.ID
		req.Connection.TargetPartnerID = req.TargetPartner.ID
		updated, err := r.UpdateConnection(txCtx, req.Connection)
		if err != nil {
			return err
		}
		req.Connection = updated
		return nil
	})
	if err != nil {
		return nil, err
	}

	req.Connection.SourcePartner = req.SourcePartner
	req.Connection.TargetPartner = req.TargetPartner
	return req.Connection, nil
}

func applyConnectionTenantFilter(
	query *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
) *bun.SelectQuery {
	cols := buncolgen.EDIConnectionColumns

	return query.
		Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(cols.SourceOrganizationID.Eq(), tenantInfo.OrgID).
				WhereOr(cols.TargetOrganizationID.Eq(), tenantInfo.OrgID)
		})
}
