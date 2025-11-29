package documenttemplate

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/permissionregistry"
)

var _ permissionregistry.PermissionAware = (*documentTemplatePermission)(nil)

type documentTemplatePermission struct{}

func NewDocumentTemplatePermission() permissionregistry.PermissionAware {
	return &documentTemplatePermission{}
}

func (p documentTemplatePermission) GetResourceName() string {
	return "document_template"
}

func (p documentTemplatePermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Create new document templates",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View document template information",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpUpdate,
			"update",
			"Update",
			"Modify document template details",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove document templates",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpExport,
			"export",
			"Export",
			"Export document templates",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpImport,
			"import",
			"Import",
			"Import document templates",
		),
	}
}

func (p documentTemplatePermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage":    permission.OpCreate | permission.OpRead | permission.OpUpdate | permission.OpDelete | permission.OpExport | permission.OpImport,
		"read_only": permission.OpRead,
	}
}

func (p documentTemplatePermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (p documentTemplatePermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpCreate,
		permission.OpUpdate,
		permission.OpDelete,
	}
}

var _ permissionregistry.PermissionAware = (*generatedDocumentPermission)(nil)

type generatedDocumentPermission struct{}

func NewGeneratedDocumentPermission() permissionregistry.PermissionAware {
	return &generatedDocumentPermission{}
}

func (p generatedDocumentPermission) GetResourceName() string {
	return "generated_document"
}

func (p generatedDocumentPermission) GetSupportedOperations() []permissionregistry.OperationDefinition {
	return []permissionregistry.OperationDefinition{
		permissionregistry.BuildOperationDefinition(
			permission.OpCreate,
			"create",
			"Create",
			"Generate new documents",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpRead,
			"read",
			"Read",
			"View generated documents",
		),
		permissionregistry.BuildOperationDefinition(
			permission.OpDelete,
			"delete",
			"Delete",
			"Remove generated documents",
		),
	}
}

func (p generatedDocumentPermission) GetCompositeOperations() map[string]permission.Operation {
	return map[string]permission.Operation{
		"manage":    permission.OpCreate | permission.OpRead | permission.OpDelete,
		"read_only": permission.OpRead,
	}
}

func (p generatedDocumentPermission) GetDefaultOperation() permission.Operation {
	return permission.OpRead
}

func (p generatedDocumentPermission) GetOperationsRequiringApproval() []permission.Operation {
	return []permission.Operation{
		permission.OpDelete,
	}
}
