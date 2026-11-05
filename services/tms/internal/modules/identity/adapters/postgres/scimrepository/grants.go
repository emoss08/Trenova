package scimrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac/rbacdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim"
	"github.com/emoss08/trenova/internal/modules/identity/domain/scim/scimdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const pendingScopeReason = "find the SCIM directories, across every organization, whose group roles wait to be reconciled; each is then reconciled in its own tenant scope"

func (r *repository) MarkPending(
	ctx context.Context,
	directoryID pulid.ID,
	userIDs []pulid.ID,
) error {
	if len(userIDs) == 0 {
		return nil
	}

	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		su := scimdb.Users
		if _, err := su.Where(su.DirectoryID.Eq(directoryID), su.UserID.In(userIDs...)).
			Update(ctx, r.db.Keel(ctx), su.RolesVersion.Increment(1)); err != nil {
			return userErrors.Map(err, directoryID.String())
		}

		return nil
	})
}

func (r *repository) MarkGroupPending(ctx context.Context, directoryID, groupID pulid.ID) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		su := scimdb.Users
		gm := scimdb.GroupMembers
		if _, err := su.Where(
			su.DirectoryID.Eq(directoryID),
			su.UserID.InQuery(gm.Pick(gm.UserID).Where(
				gm.DirectoryID.Eq(directoryID),
				gm.GroupID.Eq(groupID),
			)),
		).Update(ctx, r.db.Keel(ctx), su.RolesVersion.Increment(1)); err != nil {
			return userErrors.Map(err, groupID.String())
		}

		return nil
	})
}

func (r *repository) MarkOrganizationPending(ctx context.Context) ([]pulid.ID, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		db := r.db.Keel(ctx)
		su := scimdb.Users
		if _, err := su.Where(su.RolesVersion.Gte(0)).
			Update(ctx, db, su.RolesVersion.Increment(1)); err != nil {
			return nil, userErrors.Map(err, "")
		}

		sd := iamdb.SCIMDirectories
		directories, err := keel.Into[pulid.ID](ctx, db, sd.Pick(sd.ID).Where(sd.Enabled.IsTrue()))
		if err != nil {
			return nil, userErrors.Map(err, "")
		}

		return directories, nil
	})
}

func (r *repository) ListPending(
	ctx context.Context,
	directoryID pulid.ID,
	limit int,
) ([]*scim.User, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*scim.User, error) {
		su := scimdb.Users
		links, err := su.Where(
			su.DirectoryID.Eq(directoryID),
			su.RolesReconciledVersion.NeCol(su.RolesVersion),
		).
			Order(su.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, userErrors.Map(err, directoryID.String())
		}

		return links, nil
	})
}

func (r *repository) ListPendingDirectories(
	ctx context.Context,
	limit int,
) ([]scimports.PendingDirectory, error) {
	ctx = dbscope.WithSystem(ctx, pendingScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]scimports.PendingDirectory, error) {
		su := scimdb.Users
		directories, err := keel.Into[scimports.PendingDirectory](
			ctx,
			r.db.Keel(ctx),
			su.AllTenants().
				Group(su.OrganizationID, su.BusinessUnitID, su.DirectoryID).
				Pick(su.OrganizationID, su.BusinessUnitID, su.DirectoryID).
				Where(su.RolesReconciledVersion.NeCol(su.RolesVersion)).
				Limit(limit),
		)
		if err != nil {
			return nil, userErrors.Map(err, "")
		}

		return directories, nil
	})
}

func (r *repository) CompleteReconcile(ctx context.Context, link *scim.User) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		su := scimdb.Users
		if _, err := su.Where(
			su.ID.Eq(link.ID),
			su.RolesReconciledVersion.Lt(link.RolesVersion),
		).Update(ctx, r.db.Keel(ctx), su.RolesReconciledVersion.To(link.RolesVersion)); err != nil {
			return userErrors.Map(err, link.UserID.String())
		}

		return nil
	})
}

func (r *repository) RoleSources(
	ctx context.Context,
	directoryID, userID pulid.ID,
) ([]*scim.RoleSource, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*scim.RoleSource, error) {
		gm := scimdb.GroupMembers
		sg := scimdb.Groups
		m := iamdb.SCIMGroupRoleMappings

		rows, err := keel.Into[scim.RoleSource](
			ctx,
			r.db.Keel(ctx),
			gm.Join(sg, sg.ID.EqCol(gm.GroupID), sg.DirectoryID.EqCol(gm.DirectoryID)).
				Join(
					m,
					m.DirectoryID.EqCol(gm.DirectoryID),
					keel.Or(
						m.ExternalGroupID.EqCol(sg.ExternalID),
						m.ExternalGroupID.Lower().EqExpr(sg.DisplayName.Lower()),
					),
				).
				Pick(m.RoleID, m.ID, m.SavedByID, sg.ID, sg.DisplayName).
				Where(
					gm.DirectoryID.Eq(directoryID),
					gm.UserID.Eq(userID),
					m.SavedByID.IsNotNull(),
				).
				Order(m.RoleID.Asc(), m.CreatedAt.Asc()),
		)
		if err != nil {
			return nil, grantErrors.Map(err, userID.String())
		}

		sources := make([]*scim.RoleSource, 0, len(rows))
		for i := range rows {
			sources = append(sources, &rows[i])
		}

		return sources, nil
	})
}

func (r *repository) ListGrants(
	ctx context.Context,
	directoryID, userID pulid.ID,
) ([]*scim.RoleGrant, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*scim.RoleGrant, error) {
		rg := scimdb.RoleGrants
		grants, err := rg.Where(rg.DirectoryID.Eq(directoryID), rg.UserID.Eq(userID)).
			Order(rg.RoleID.Asc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, grantErrors.Map(err, userID.String())
		}

		return grants, nil
	})
}

func (r *repository) CreateGrant(ctx context.Context, grant *scim.RoleGrant) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := scimdb.RoleGrants.Create(ctx, r.db.Keel(ctx), grant); err != nil {
			return grantErrors.Map(err, grant.AssignmentID.String())
		}

		return nil
	})
}

func (r *repository) DeleteDirectoryAssignments(
	ctx context.Context,
	directoryID pulid.ID,
) ([]*rbac.UserRoleAssignment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*rbac.UserRoleAssignment, error) {
		db := r.db.Keel(ctx)
		ura := rbacdb.UserRoleAssignments
		rg := scimdb.RoleGrants

		assignments, err := ura.Where(
			ura.ID.InQuery(rg.Pick(rg.AssignmentID).Where(rg.DirectoryID.Eq(directoryID))),
		).Lock(keel.ForUpdate).All(ctx, db)
		if err != nil {
			return nil, grantErrors.Map(err, directoryID.String())
		}
		if len(assignments) == 0 {
			return assignments, nil
		}

		ids := make([]pulid.ID, len(assignments))
		for i, assignment := range assignments {
			ids[i] = assignment.ID
		}
		if _, err = ura.Where(ura.ID.In(ids...)).Delete(ctx, db); err != nil {
			return nil, grantErrors.Map(err, directoryID.String())
		}

		return assignments, nil
	})
}
