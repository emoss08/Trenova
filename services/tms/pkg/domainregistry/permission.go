package domainregistry

// The actual implementation is in pkg/permissionregistry to avoid import cycles

import "github.com/emoss08/trenova/pkg/permissionregistry"

type (
	PermissionAware     = permissionregistry.PermissionAware
	OperationDefinition = permissionregistry.OperationDefinition
)

var (
	BuildOperationDefinition       = permissionregistry.BuildOperationDefinition
	StandardCRUDOperations         = permissionregistry.StandardCRUDOperations
	StandardExportImportOperations = permissionregistry.StandardExportImportOperations
	StandardArchiveOperations      = permissionregistry.StandardArchiveOperations
	StandardWorkflowOperations     = permissionregistry.StandardWorkflowOperations
	StandardAssignmentOperations   = permissionregistry.StandardAssignmentOperations
	StandardCompositeOperations    = permissionregistry.StandardCompositeOperations
)
