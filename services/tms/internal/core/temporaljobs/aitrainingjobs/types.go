package aitrainingjobs

import (
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	AITrainingExportWorkflowName = "AITrainingExportWorkflow"
	organizationsPerExecution    = 25
	organizationPageSize         = 100
	errorTypeExportInactive      = "training_export_inactive"
)

type ExportPayload struct {
	ExportID            pulid.ID `json:"exportId"`
	AfterOrganizationID pulid.ID `json:"afterOrganizationId,omitempty"`
	AfterBusinessUnitID pulid.ID `json:"afterBusinessUnitId,omitempty"`
	NextOrdinal         int      `json:"nextOrdinal,omitempty"`
}

type ExportInput struct {
	ExportID pulid.ID `json:"exportId"`
}

type ListOrganizationsInput struct {
	AfterOrganizationID pulid.ID `json:"afterOrganizationId,omitempty"`
	AfterBusinessUnitID pulid.ID `json:"afterBusinessUnitId,omitempty"`
	Limit               int      `json:"limit"`
}

type ConsentingOrganization struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	GrantedAt      int64    `json:"grantedAt"`
}

type ExportOrganizationInput struct {
	ExportID     pulid.ID               `json:"exportId"`
	Ordinal      int                    `json:"ordinal"`
	Organization ConsentingOrganization `json:"organization"`
}

type ExportOrganizationOutcome struct {
	Examples         int  `json:"examples"`
	ConsentWithdrawn bool `json:"consentWithdrawn"`
}

type FailInput struct {
	ExportID pulid.ID `json:"exportId"`
	Message  string   `json:"message"`
}

type ExportOutcome struct {
	Status   string `json:"status"`
	Examples int    `json:"examples"`
}
