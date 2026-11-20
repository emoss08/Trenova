package tenantjobscontract

import (
	"strconv"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	RemovalWorkflowName  = "CloudTenantRemovalWorkflow"
	SessionsWorkflowName = "CloudTenantSessionsWorkflow"

	removalWorkflowPrefix  = "cloud-tenant-removal/"
	sessionsWorkflowPrefix = "cloud-tenant-sessions/"
)

type TenantPayload struct {
	InstanceID     string   `json:"instanceId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Revision       int64    `json:"revision"`
}

func (p *TenantPayload) Tenant() dbscope.Tenant {
	return dbscope.Tenant{OrganizationID: p.OrganizationID, BusinessUnitID: p.BusinessUnitID}
}

func RemovalWorkflowID(instanceID string) string {
	return removalWorkflowPrefix + instanceID
}

func SessionsWorkflowID(instanceID string, revision int64) string {
	return sessionsWorkflowPrefix + instanceID + "/" + strconv.FormatInt(revision, 10)
}
