//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package edirepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
)

func (r *repository) ListConnections(
	ctx context.Context,
	req *repositories.ListEDIConnectionsRequest,
) (*pagination.ListResult[*edi.EDIConnection], error) {
	entities := make([]*edi.EDIConnection, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("SourceOrganization").
		Relation("TargetOrganization").
		Relation("SourcePartner").
		Relation("TargetPartner").
		Apply(func(query *bun.SelectQuery) *bun.SelectQuery {
			return filterConnectionsQuery(query, req)
		}).
		Order("ec.created_at DESC").
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
	return query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.WhereOr("ec.method::text ILIKE ?", term).
			WhereOr("ec.status::text ILIKE ?", term).
			WhereOr("source_organization.name ILIKE ?", term).
			WhereOr("target_organization.name ILIKE ?", term)
	})
}

func (r *repository) GetConnectionByID(
	ctx context.Context,
	req repositories.GetEDIConnectionByIDRequest,
) (*edi.EDIConnection, error) {
	entity := new(edi.EDIConnection)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("SourceOrganization").
		Relation("TargetOrganization").
		Relation("SourcePartner").
		Relation("TargetPartner").
		Where("ec.id = ?", req.ID).
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
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ec.id = ?", req.ID).
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
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where("ec.business_unit_id = ?", req.TenantInfo.BuID).
		Where("ec.method = ?", req.Method).
		Where("ec.status = ?", edi.ConnectionStatusActive).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr(
				"(ec.source_organization_id = ? AND ec.source_partner_id = ?)",
				req.TenantInfo.OrgID,
				req.PartnerID,
			).WhereOr(
				"(ec.target_organization_id = ? AND ec.target_partner_id = ?)",
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

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
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
		if _, err := r.ensureMappingProfile(txCtx, req.SourcePartner); err != nil {
			return err
		}

		if _, err := r.db.DBForContext(txCtx).
			NewInsert().
			Model(req.TargetPartner).
			Returning("*").
			Exec(txCtx); err != nil {
			return err
		}
		if _, err := r.ensureMappingProfile(txCtx, req.TargetPartner); err != nil {
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
			Column("default_transport_id").
			Exec(txCtx); err != nil {
			return err
		}
		if _, err := r.db.DBForContext(txCtx).
			NewUpdate().
			Model(req.TargetPartner).
			WherePK().
			Column("default_transport_id").
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
	return query.
		Where("ec.business_unit_id = ?", tenantInfo.BuID).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.WhereOr("ec.source_organization_id = ?", tenantInfo.OrgID).
				WhereOr("ec.target_organization_id = ?", tenantInfo.OrgID)
		})
}
