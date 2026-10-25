package iamrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var scimGroupRoleMappingErrors = dberror.Mapping{
	Entity: "SCIM group role mapping",
	Unique: []dberror.Constraint{
		{
			Name:    "idx_sgrm_display_name",
			Field:   "displayName",
			Message: "A SCIM group role mapping with this display name already exists",
		},
	},
}

func withMappingRole(
	q *keel.Query[iam.SCIMGroupRoleMapping],
) *keel.Query[iam.SCIMGroupRoleMapping] {
	return q.Load(iamdb.SCIMGroupRoleMappings.Role())
}

func (r *repository) ListSCIMGroupRoleMappings(
	ctx context.Context,
	req *iamports.ListSCIMGroupRoleMappingsRequest,
) (*pagination.ListResult[*iam.SCIMGroupRoleMapping], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*iam.SCIMGroupRoleMapping], error) {
			m := iamdb.SCIMGroupRoleMappings
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				m.Where(m.DirectoryID.Eq(req.DirectoryID)),
				&listquery.Request[iam.SCIMGroupRoleMapping]{
					Filter: req.Filter,
					Shape:  withMappingRole,
					Then:   []keel.Order{m.DisplayName.Asc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list scim group role mappings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListSCIMGroupRoleMappingsConnection(
	ctx context.Context,
	req *iamports.ListSCIMGroupRoleMappingConnectionRequest,
) (*pagination.CursorListResult[*iam.SCIMGroupRoleMapping], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*iam.SCIMGroupRoleMapping], error) {
			m := iamdb.SCIMGroupRoleMappings
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				m.Where(m.DirectoryID.Eq(req.DirectoryID)),
				&listquery.Request[iam.SCIMGroupRoleMapping]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withMappingRole,
				},
			)
			if err != nil {
				r.l.Error("failed to list scim group role mappings", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) CreateSCIMGroupRoleMapping(
	ctx context.Context,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.SCIMGroupRoleMapping, error) {
		if err := iamdb.SCIMGroupRoleMappings.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to insert scim group role mapping", zap.Error(err))
			return nil, scimGroupRoleMappingErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateSCIMGroupRoleMapping(
	ctx context.Context,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*iam.SCIMGroupRoleMapping, error) {
		if err := iamdb.SCIMGroupRoleMappings.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error(
				"failed to update scim group role mapping",
				zap.String("sgrmID", entity.ID.String()),
				zap.Error(err),
			)
			return nil, scimGroupRoleMappingErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) DeleteSCIMGroupRoleMapping(
	ctx context.Context,
	_ pagination.TenantInfo,
	sgrmID pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		m := iamdb.SCIMGroupRoleMappings
		deleted, err := m.Where(m.ID.Eq(sgrmID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return scimGroupRoleMappingErrors.Map(err, sgrmID.String())
		}
		return dberror.CheckAffected(deleted, scimGroupRoleMappingErrors.Entity, sgrmID.String())
	})
}
