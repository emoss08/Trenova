/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package fixtures

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/permissionbuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

// hasAction checks if a resource has a specific action
func hasAction(resource permission.Resource, action permission.Action) bool {
	if actions, exists := permission.ResourceActionMap[resource]; exists {
		return slices.Contains(actions, action)
	}
	return false
}

func GenerateResourcePermissions() []ResourcePermissions {
	var allResourcesPerms []ResourcePermissions

	for resource, actions := range permission.ResourceActionMap {
		var permissions []permissionbuilder.PermissionDefinition

		// Generate standard permissions based on the mapped actions
		for _, action := range actions {
			builder := permissionbuilder.NewPermissionBuilder(resource, action)

			// Add special dependencies for manage action (automatic dependencies will handle others)
			if action == permission.ActionManage {
				var manageDeps []struct {
					Resource permission.Resource
					Action   permission.Action
				}

				// Only add dependencies for actions that exist for this resource
				if hasAction(resource, permission.ActionRead) {
					manageDeps = append(manageDeps, struct {
						Resource permission.Resource
						Action   permission.Action
					}{Resource: resource, Action: permission.ActionRead})
				}
				if hasAction(resource, permission.ActionCreate) {
					manageDeps = append(manageDeps, struct {
						Resource permission.Resource
						Action   permission.Action
					}{Resource: resource, Action: permission.ActionCreate})
				}
				if hasAction(resource, permission.ActionUpdate) {
					manageDeps = append(manageDeps, struct {
						Resource permission.Resource
						Action   permission.Action
					}{Resource: resource, Action: permission.ActionUpdate})
				}
				if hasAction(resource, permission.ActionDelete) {
					manageDeps = append(manageDeps, struct {
						Resource permission.Resource
						Action   permission.Action
					}{Resource: resource, Action: permission.ActionDelete})
				}

				if len(manageDeps) > 0 {
					builder.WithDependencies(manageDeps...)
				}
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

// ValidateDependencies ensures all dependencies exist and there are no circular references
func ValidateDependencies(resourcePermissions []ResourcePermissions) error {
	permissionExists := make(map[string]bool)

	// First pass: build index of all permissions that will exist
	for _, rp := range resourcePermissions {
		for _, pd := range rp.Permissions {
			key := fmt.Sprintf("%s:%s", pd.Resource, pd.Action)
			permissionExists[key] = true
		}
	}

	// Second pass: validate all dependencies exist
	for _, rp := range resourcePermissions {
		for _, pd := range rp.Permissions {
			for _, dep := range pd.DependsOn {
				depKey := fmt.Sprintf("%s:%s", dep.Resource, dep.Action)
				if !permissionExists[depKey] {
					return eris.Errorf("permission %s:%s depends on %s:%s which doesn't exist",
						pd.Resource, pd.Action, dep.Resource, dep.Action)
				}
			}
		}
	}

	// Third pass: check for circular dependencies using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(permKey string) bool {
		visited[permKey] = true
		recStack[permKey] = true

		// Find the permission definition for this key
		for _, rp := range resourcePermissions {
			for _, pd := range rp.Permissions {
				if fmt.Sprintf("%s:%s", pd.Resource, pd.Action) == permKey {
					// Check all dependencies
					for _, dep := range pd.DependsOn {
						depKey := fmt.Sprintf("%s:%s", dep.Resource, dep.Action)
						if !visited[depKey] {
							if hasCycle(depKey) {
								return true
							}
						} else if recStack[depKey] {
							return true
						}
					}
					break
				}
			}
		}

		recStack[permKey] = false
		return false
	}

	for _, rp := range resourcePermissions {
		for _, pd := range rp.Permissions {
			permKey := fmt.Sprintf("%s:%s", pd.Resource, pd.Action)
			if !visited[permKey] && hasCycle(permKey) {
				return eris.Errorf("circular dependency detected involving permission %s", permKey)
			}
		}
	}

	return nil
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

	resourcePermissions := GenerateResourcePermissions()

	// Validate dependencies before proceeding
	if err := ValidateDependencies(resourcePermissions); err != nil {
		return eris.Wrap(err, "dependency validation failed")
	}

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
