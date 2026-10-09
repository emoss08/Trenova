package realtimeinvalidation

import "github.com/emoss08/trenova/internal/core/domain/permission"

const RecordChangePrefix = "audited:"

func RecordChangeResource(resource permission.Resource) string {
	return RecordChangePrefix + string(resource)
}

func IsRecordChange(resource permission.Resource, operation permission.Operation) bool {
	if resource == "" || resource == permission.ResourceAuditLog {
		return false
	}

	return operation != "" && operation != permission.OpRead && operation != permission.OpExport
}

func RecordChangeAction(operation permission.Operation) string {
	if operation == permission.OpCreate ||
		operation == permission.OpDuplicate ||
		operation == permission.OpImport {
		return "created"
	}
	if operation == permission.OpDelete {
		return "deleted"
	}
	return "updated"
}
