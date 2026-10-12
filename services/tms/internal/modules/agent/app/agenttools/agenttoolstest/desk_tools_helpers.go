package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/shared/pulid"
)

func DeskParams(params map[string]any) agentports.ToolExecuteParams {
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")

	return agentports.ToolExecuteParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Actor: &permissionports.RequestActor{
			UserID:         pulid.MustNew("usr_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		Params:         params,
		IdempotencyKey: "key-1",
	}
}

type StubCredentialActor struct {
	Asked *workercredentialservice.RenewalRequest `json:"-"`
}

func (s *StubCredentialActor) RequestRenewal(
	_ context.Context,
	req workercredentialservice.RenewalRequest,
) error {
	s.Asked = &req

	return nil
}
