package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/permissionbuilder"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/fatih/color"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dbfixture"
)

// PermissionDefinition defines a permission with its dependencies
type PermissionDefinition struct {
	Resource    permission.Resource
	Action      permission.Action
	Scope       permission.Scope
	Description string
	DependsOn   []struct {
		Resource permission.Resource
		Action   permission.Action
	}
	FieldSettings []*permission.FieldPermission
}

// ResourcePermissions defines all permissions for a specific resource
type ResourcePermissions struct {
	Resource    permission.Resource
	Permissions []permissionbuilder.PermissionDefinition
}

func generateResourcePermissions() []ResourcePermissions {
	var allResourcesPerms []ResourcePermissions

	for resource, actions := range permission.ResourceActionMap {
		var permissions []permissionbuilder.PermissionDefinition

		// Generate standard permissions based on the mapped actions
		for _, action := range actions {
			builder := permissionbuilder.NewPermissionBuilder(resource, action)

			// Add standard dependencies for certain actions
			switch action {
			case permission.ActionUpdate, permission.ActionDelete:
				builder.WithDependencies(struct {
					Resource permission.Resource
					Action   permission.Action
				}{
					Resource: resource,
					Action:   permission.ActionRead,
				})
			case permission.ActionManage:
				builder.WithDependencies(struct {
					Resource permission.Resource
					Action   permission.Action
				}{
					Resource: resource,
					Action:   permission.ActionRead,
				},
					struct {
						Resource permission.Resource
						Action   permission.Action
					}{
						Resource: resource,
						Action:   permission.ActionCreate,
					},
					struct {
						Resource permission.Resource
						Action   permission.Action
					}{
						Resource: resource,
						Action:   permission.ActionUpdate,
					},
					struct {
						Resource permission.Resource
						Action   permission.Action
					}{
						Resource: resource,
						Action:   permission.ActionDelete,
					},
				)
			}

			switch resource {
			case permission.ResourceUser:
				if action == permission.ActionCreate {
					builder.WithFieldSettings(&permission.FieldPermission{
						Field:      "email_address",
						Actions:    []permission.Action{action},
						AuditLevel: permission.AuditChanges,
					})
				}
			case permission.ResourceOrganization:
				if action == permission.ActionModifyField {
					builder.WithFieldSettings(&permission.FieldPermission{
						Field:      "logo_url",
						Actions:    []permission.Action{action},
						AuditLevel: permission.AuditChanges,
					})
				}
			}

			permissions = append(permissions, builder.Build())
		}

		allResourcesPerms = append(allResourcesPerms, ResourcePermissions{
			Resource:    resource,
			Permissions: permissions,
		})
	}

	return allResourcesPerms
}

func LoadPermissions(ctx context.Context, db *bun.DB, fixture *dbfixture.Fixture) error {
	org := fixture.MustRow("Organization.trenova").(*organization.Organization)
	bu := fixture.MustRow("BusinessUnit.trenova").(*businessunit.BusinessUnit)

	exists, err := db.NewSelect().Model((*permission.Role)(nil)).
		Where("name = ?", "Super Administrator").
		Exists(ctx)
	if err != nil {
		return eris.Wrap(err, "check if super admin role exists")
	}
	if exists {
		return ErrAdminAccountAlreadyExists
	}

	// get admin user
	var adminUser user.User
	err = db.NewSelect().
		Model(&adminUser).
		Where("email_address = ?", "admin@trenova.app").
		Scan(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to get admin user")
	}

	resourcePermissions := generateResourcePermissions()

	// Create super admin role
	superAdminRole := &permission.Role{
		Name:           "Super Administrator",
		Description:    "Has complete system access with full administrative capabilities",
		RoleType:       permission.RoleTypeSystem,
		BusinessUnitID: bu.ID,
		OrganizationID: org.ID,
		IsSystem:       true,
		Priority:       100,
		Status:         domain.StatusActive,
	}

	// Create and track all permissions
	permissionMap := make(map[string]*permission.Permission)
	var allPermissions []*permission.Permission

	if err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// Create role
		if _, err = tx.NewInsert().Model(superAdminRole).Exec(c); err != nil {
			return eris.Wrap(err, "insert super admin role")
		}

		// First pass: Create all permissions without dependencies
		for _, rp := range resourcePermissions {
			for _, pd := range rp.Permissions {
				perm := &permission.Permission{
					Resource:         pd.Resource,
					Action:           pd.Action,
					Scope:            pd.Scope,
					Description:      pd.Description,
					IsSystemLevel:    true,
					CreatedAt:        time.Now().Unix(),
					UpdatedAt:        time.Now().Unix(),
					FieldPermissions: pd.FieldSettings,
				}

				if _, err = tx.NewInsert().Model(perm).Exec(c); err != nil {
					return eris.Wrapf(err, "insert permission for %s:%s", pd.Resource, pd.Action)
				}

				key := fmt.Sprintf("%s:%s", pd.Resource, pd.Action)
				permissionMap[key] = perm
				allPermissions = append(allPermissions, perm)
			}
		}

		// Second pass: Update dependencies
		for _, rp := range resourcePermissions {
			for _, pd := range rp.Permissions {
				if len(pd.DependsOn) == 0 {
					continue
				}

				key := fmt.Sprintf("%s:%s", pd.Resource, pd.Action)
				perm := permissionMap[key]

				var depIDs []pulid.ID
				for _, dep := range pd.DependsOn {
					depKey := fmt.Sprintf("%s:%s", dep.Resource, dep.Action)
					if depPerm, depExists := permissionMap[depKey]; depExists {
						depIDs = append(depIDs, depPerm.ID)
					}
				}

				perm.Dependencies = depIDs
				if _, err = tx.NewUpdate().
					Model(perm).
					Column("dependencies").
					Where("id = ?", perm.ID).
					Exec(c); err != nil {
					return eris.Wrapf(err, "update dependencies for permission %s", perm.ID)
				}
			}
		}

		// Create role-permission associations
		rolePerms := make([]*permission.RolePermission, len(allPermissions))
		for i, perm := range allPermissions {
			rolePerms[i] = &permission.RolePermission{
				BusinessUnitID: bu.ID,
				OrganizationID: org.ID,
				RoleID:         superAdminRole.ID,
				PermissionID:   perm.ID,
			}
		}

		if _, err = tx.NewInsert().Model(&rolePerms).Exec(c); err != nil {
			return eris.Wrap(err, "insert role permissions")
		}

		// Assign role to admin user
		userRole := &user.UserRole{
			BusinessUnitID: bu.ID,
			OrganizationID: org.ID,
			UserID:         adminUser.ID,
			RoleID:         superAdminRole.ID,
		}

		if _, err = tx.NewInsert().Model(userRole).Exec(c); err != nil {
			return eris.Wrap(err, "assign role to admin")
		}

		return nil
	}); err != nil {
		return eris.Wrap(err, "transaction failed")
	}

	color.Green(
		"✓ Created Super Administrator role with comprehensive permissions and dependencies",
	)
	return nil
}
