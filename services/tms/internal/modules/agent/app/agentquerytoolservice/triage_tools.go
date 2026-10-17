package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
)

const minutesPerHour = 60

type triageAssigneeReader interface {
	Assignees(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*inboundmessageports.AssigneeView, error)
	Settings(ctx context.Context, tenantInfo pagination.TenantInfo) (*triage.Settings, error)
}

type triageAssigneeRow struct {
	UserID  string `json:"userId"`
	Name    string `json:"name"`
	Waiting int    `json:"waiting"`
	OnCall  bool   `json:"onCall,omitempty"`
}

type triageAssigneesView struct {
	People       []triageAssigneeRow `json:"people"`
	OnCallUserID string              `json:"onCallUserId,omitempty"`
	EndOfDay     string              `json:"endOfDay"`
}

type listTriageAssigneesTool struct {
	triage triageAssigneeReader
}

func newListTriageAssigneesTool(reader triageAssigneeReader) agentports.AgentQueryTool {
	return &listTriageAssigneesTool{triage: reader}
}

func (t *listTriageAssigneesTool) Name() string { return "list_triage_assignees" }

func (t *listTriageAssigneesTool) Description() string {
	return "List the people who can work the Triage queue, with how many waiting messages " +
		"each already holds, and the on-call person who takes the queue at the end of the " +
		"working day. The ids here are the owners assign_inbound_messages and " +
		"hand_off_triage take."
}

func (t *listTriageAssigneesTool) SearchTerms() []string {
	return []string{"who works triage", "on call", "inbox team", "who can take emails"}
}

func (t *listTriageAssigneesTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
}

func (t *listTriageAssigneesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInboundMessage})
}

func (t *listTriageAssigneesTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	people, err := t.triage.Assignees(ctx, tenant)
	if err != nil {
		return nil, err
	}
	settings, err := t.triage.Settings(ctx, tenant)
	if err != nil {
		return nil, err
	}

	view := triageAssigneesView{
		People:       make([]triageAssigneeRow, 0, len(people)),
		OnCallUserID: pulidString(settings.OnCallUserID),
		EndOfDay: fmt.Sprintf(
			"%02d:%02d",
			settings.EndOfDayMinute/minutesPerHour,
			settings.EndOfDayMinute%minutesPerHour,
		),
	}
	for _, person := range people {
		if person == nil {
			continue
		}
		view.People = append(view.People, triageAssigneeRow{
			UserID:  person.UserID.String(),
			Name:    person.Name,
			Waiting: person.Waiting,
			OnCall:  person.UserID == settings.OnCallUserID,
		})
	}

	return view, nil
}
