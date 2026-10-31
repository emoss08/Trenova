package palettecontract

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const AnswerWorkflowName = "PaletteAnswerWorkflow"

// WorkflowID names a question by the person who asked it, so a reader can
// only ever reach their own: the id is built from the session, never taken
// from the request.
func WorkflowID(ref paletteports.AskRef) string {
	return "palette-answer/" + ref.TenantInfo.OrgID.String() + "/" +
		ref.TenantInfo.UserID.String() + "/" + ref.AskID.String()
}

type AnswerInput struct {
	OrganizationID pulid.ID              `json:"organizationId"`
	BusinessUnitID pulid.ID              `json:"businessUnitId"`
	UserID         pulid.ID              `json:"userId"`
	AskID          pulid.ID              `json:"askId"`
	Question       paletteports.Question `json:"question"`
}

func (i *AnswerInput) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: i.OrganizationID, BuID: i.BusinessUnitID, UserID: i.UserID}
}

// Actor is the person who asked; every read is made as them.
func (i *AnswerInput) Actor() *permissionports.RequestActor {
	return permissionports.UserActor(i.TenantInfo())
}
