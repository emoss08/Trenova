package dedicatedlane

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*suggestionPermission)(nil)

type suggestionPermission struct{}

func NewSuggestionPermission() permissionregistry.PermissionAware {
	return &suggestionPermission{}
}

func (c suggestionPermission) GetResourceName() string {
	return "dedicated_lane_suggestion"
}

func (c suggestionPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View dedicated lane suggestions information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpApprove,
			"approve",
			"Approve",
			"Approve dedicated lane suggestion",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpReject,
			"reject",
			"Reject",
			"Reject dedicated lane suggestion",
		),
	}
}

func (c suggestionPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage":    permission.OpRead | permission.OpApprove | permission.OpReject,
		"read_only": permission.OpRead,
	}
}

func (c suggestionPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (c suggestionPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpApprove,
		permission.OpReject,
	}
}
