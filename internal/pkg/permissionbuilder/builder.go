// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package permissionbuilder

import (
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/permission"
)

type PermissionBuilder struct {
	resource      permission.Resource
	action        permission.Action
	scope         permission.Scope
	description   string
	fieldSettings []*permission.FieldPermission
	dependsOn     []struct {
		Resource permission.Resource
		Action   permission.Action
	}
	skipAutoDependencies bool
}

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

func NewPermissionBuilder(
	resource permission.Resource,
	action permission.Action,
) *PermissionBuilder {
	return &PermissionBuilder{
		resource: resource,
		action:   action,
		scope:    permission.ScopeGlobal, // Default scope level
	}
}

// isActionAvailableForResource checks if an action is available for a resource
func isActionAvailableForResource(resource permission.Resource, action permission.Action) bool {
	if actions, exists := permission.ResourceActionMap[resource]; exists {
		return slices.Contains(actions, action)
	}
	return false
}

func (pb *PermissionBuilder) WithScope(scope permission.Scope) *PermissionBuilder {
	pb.scope = scope
	return pb
}

func (pb *PermissionBuilder) WithDescription(desc string) *PermissionBuilder {
	pb.description = desc
	return pb
}

func (pb *PermissionBuilder) WithFieldSettings(
	settings ...*permission.FieldPermission,
) *PermissionBuilder {
	pb.fieldSettings = append(pb.fieldSettings, settings...)
	return pb
}

func (pb *PermissionBuilder) WithDependencies(deps ...struct {
	Resource permission.Resource
	Action   permission.Action
},
) *PermissionBuilder {
	pb.dependsOn = append(pb.dependsOn, deps...)
	return pb
}

func (pb *PermissionBuilder) SkipAutoDependencies() *PermissionBuilder {
	pb.skipAutoDependencies = true
	return pb
}

func (pb *PermissionBuilder) Build() PermissionDefinition {
	if pb.description == "" {
		pb.description = fmt.Sprintf("%s %s", pb.action, pb.resource)
	}

	// Automatically add dependencies based on action type (unless skipped)
	if !pb.skipAutoDependencies {
		pb.addAutomaticDependencies()
	}

	return PermissionDefinition{
		Resource:      pb.resource,
		Action:        pb.action,
		Scope:         pb.scope,
		Description:   pb.description,
		FieldSettings: pb.fieldSettings,
		DependsOn:     pb.dependsOn,
	}
}

// addAutomaticDependencies adds logical dependencies based on action types
func (pb *PermissionBuilder) addAutomaticDependencies() {
	// Skip auto-dependencies for read and manage actions as they have special handling
	if pb.action == permission.ActionRead || pb.action == permission.ActionManage {
		return
	}

	readDep := struct {
		Resource permission.Resource
		Action   permission.Action
	}{
		Resource: pb.resource,
		Action:   permission.ActionRead,
	}

	// Determine if this action should depend on read
	if pb.shouldDependOnRead() {
		pb.addReadDependency(readDep)
	}

	// Handle special cases
	pb.handleSpecialDependencies()
}

// shouldDependOnRead determines if the action should depend on read permission
func (pb *PermissionBuilder) shouldDependOnRead() bool {
	readDependentActions := []permission.Action{
		permission.ActionCreate, permission.ActionUpdate, permission.ActionDelete,
		permission.ActionModifyField, permission.ActionArchive, permission.ActionRestore,
		permission.ActionApprove, permission.ActionReject, permission.ActionSubmit,
		permission.ActionAssign, permission.ActionReassign, permission.ActionExport,
		permission.ActionDuplicate, permission.ActionViewField, permission.ActionCancel,
		permission.ActionComplete, permission.ActionImport, permission.ActionAudit,
		permission.ActionDelegate, permission.ActionConfigure, permission.ActionSplit,
		permission.ActionReadyToBill, permission.ActionReleaseToBilling,
		permission.ActionBulkTransfer, permission.ActionReviewInvoice, permission.ActionPostInvoice,
		permission.ActionManageDefaults, permission.ActionShare,
	}

	return slices.Contains(readDependentActions, pb.action)
}

// addReadDependency adds read dependency if not already present and if read action exists
func (pb *PermissionBuilder) addReadDependency(readDep struct {
	Resource permission.Resource
	Action   permission.Action
}) {
	// Only add the dependency if the read action is available for this resource
	if isActionAvailableForResource(pb.resource, permission.ActionRead) &&
		!pb.hasDependency(readDep) {
		pb.dependsOn = append(pb.dependsOn, readDep)
	}
}

// handleSpecialDependencies handles actions that need additional dependencies beyond read
func (pb *PermissionBuilder) handleSpecialDependencies() {
	if pb.action == permission.ActionDuplicate {
		// Duplicate actions should depend on both read and create permissions
		createDep := struct {
			Resource permission.Resource
			Action   permission.Action
		}{
			Resource: pb.resource,
			Action:   permission.ActionCreate,
		}
		// Only add create dependency if create action exists for this resource
		if isActionAvailableForResource(pb.resource, permission.ActionCreate) &&
			!pb.hasDependency(createDep) {
			pb.dependsOn = append(pb.dependsOn, createDep)
		}
	}
}

// hasDependency checks if a dependency already exists
func (pb *PermissionBuilder) hasDependency(dep struct {
	Resource permission.Resource
	Action   permission.Action
}) bool {
	for _, existing := range pb.dependsOn {
		if existing.Resource == dep.Resource && existing.Action == dep.Action {
			return true
		}
	}
	return false
}
