package permissionbuilder

import (
	"fmt"

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

func NewPermissionBuilder(resource permission.Resource, action permission.Action) *PermissionBuilder {
	return &PermissionBuilder{
		resource: resource,
		action:   action,
		scope:    permission.ScopeGlobal, // Default scope level
	}
}

func (pb *PermissionBuilder) WithScope(scope permission.Scope) *PermissionBuilder {
	pb.scope = scope
	return pb
}

func (pb *PermissionBuilder) WithDescription(desc string) *PermissionBuilder {
	pb.description = desc
	return pb
}

func (pb *PermissionBuilder) WithFieldSettings(settings ...*permission.FieldPermission) *PermissionBuilder {
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

func (pb *PermissionBuilder) Build() PermissionDefinition {
	if pb.description == "" {
		pb.description = fmt.Sprintf("%s %s", pb.action, pb.resource)
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
