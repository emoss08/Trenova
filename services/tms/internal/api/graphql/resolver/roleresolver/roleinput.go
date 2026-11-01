package roleresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/shared/pulid"
)

func roleFromInput(input *gqlmodel.RoleInput) (*rbac.Role, error) {
	var parentIDs []pulid.ID
	if input.ParentRoleIds != nil {
		ids, err := gqlid.ParsePulids("parentRoleIds", input.ParentRoleIds)
		if err != nil {
			return nil, err
		}
		parentIDs = ids
	}

	role := &rbac.Role{
		Name:               input.Name,
		Description:        gqlscalar.StringValue(input.Description),
		CoreResponsibility: rbac.CoreResponsibility(gqlscalar.StringValue(input.CoreResponsibility)),
		MaxSensitivity:     permission.FieldSensitivity(input.MaxSensitivity),
		ParentRoleIDs:      parentIDs,
	}

	if input.Permissions != nil {
		role.Permissions = make([]*rbac.ResourcePermission, 0, len(input.Permissions))
		for _, grant := range input.Permissions {
			if grant == nil {
				continue
			}
			role.Permissions = append(role.Permissions, &rbac.ResourcePermission{
				Resource:   grant.Resource,
				Operations: operationsFromStrings(grant.Operations),
				DataScope:  permission.DataScope(grant.DataScope),
			})
		}
	}

	return role, nil
}

func operationsFromStrings(values []string) []permission.Operation {
	ops := make([]permission.Operation, len(values))
	for i, value := range values {
		ops[i] = permission.Operation(value)
	}
	return ops
}

func operationStrings(ops []permission.Operation) []string {
	out := make([]string, len(ops))
	for i, op := range ops {
		out[i] = string(op)
	}
	return out
}
