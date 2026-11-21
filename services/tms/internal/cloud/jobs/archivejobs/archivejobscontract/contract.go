package archivejobscontract

import (
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ArchiveWorkflowName = "CloudTenantArchiveWorkflow"
	RestoreWorkflowName = "CloudTenantRestoreWorkflow"
	ImportWorkflowName  = "CloudTenantImportWorkflow"
	NightlyWorkflowName = "CloudTenantNightlyArchiveWorkflow"
)

type CommandPayload struct {
	CommandID      string   `json:"commandId"`
	InstanceID     string   `json:"instanceId"`
	OrganizationID pulid.ID `json:"organizationId,omitempty"`
	BusinessUnitID pulid.ID `json:"businessUnitId,omitempty"`
}

func (p *CommandPayload) Tenant() dbscope.Tenant {
	return dbscope.Tenant{OrganizationID: p.OrganizationID, BusinessUnitID: p.BusinessUnitID}
}

func WorkflowID(kind tenantcommand.Kind, commandID string) string {
	return "cloud-tenant-" + string(kind) + "/" + commandID
}

func WorkflowName(kind tenantcommand.Kind) string {
	switch kind {
	case tenantcommand.KindRestore:
		return RestoreWorkflowName
	case tenantcommand.KindImport:
		return ImportWorkflowName
	case tenantcommand.KindArchive:
		return ArchiveWorkflowName
	default:
		return ArchiveWorkflowName
	}
}
