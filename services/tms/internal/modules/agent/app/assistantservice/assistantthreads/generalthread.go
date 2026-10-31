package assistantthreads

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/agentaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// generalAssistantPageLimit bounds the general assistants read to pick the one
// a conversation goes to; an organization has one, or a few.
const generalAssistantPageLimit = 100

const definitionTemplateField = "template"

// StartGeneralThread opens a Desk conversation with the organization's
// general assistant. A command palette answer the person carries on is
// seeded into one.
func (s *Service) StartGeneralThread(
	ctx context.Context,
	req *assistantports.GeneralThreadRequest,
	actor *permissionports.RequestActor,
) (*conversation.Thread, error) {
	definition, err := s.generalAssistant(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	return s.StartThread(ctx, &assistantports.StartThreadRequest{
		AgentDefinitionID: definition.ID,
		Title:             conversation.TitleFrom(req.Title),
		TenantInfo:        req.TenantInfo,
		Origin:            conversation.ThreadOriginDesk,
	}, actor)
}

// generalAssistant is who carries a palette answer on: the organization's
// general assistant, when it is on and open to the person. No other agent
// stands in for it, since each is built for its own desk and would answer
// every question from that desk's records.
func (s *Service) generalAssistant(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) (*agentdefinition.Definition, error) {
	usable, err := s.access().UsableAgents(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !usable.Assistant {
		return nil, agentaccess.ErrAgentsWithheld()
	}

	result, err := s.definitions.List(
		ctx,
		generalAssistants(tenant, generalAssistantPageLimit, usable.Audience()),
	)
	if err != nil {
		return nil, err
	}
	for _, definition := range result.Items {
		if definition != nil && usable.Allows(definition) {
			return definition, nil
		}
	}

	return nil, s.noGeneralAssistant(ctx, tenant)
}

// noGeneralAssistant says why there is nobody to carry on with: no general
// assistant is on, or none that is on is open to the person.
func (s *Service) noGeneralAssistant(ctx context.Context, tenant pagination.TenantInfo) error {
	enabled, err := s.definitions.List(ctx, generalAssistants(tenant, 1, nil))
	if err != nil {
		return err
	}
	if len(enabled.Items) > 0 {
		return errortypes.NewAuthorizationError(
			"Opening this in the Desk needs the general assistant, which is not open to you. " +
				"An administrator can give one of your roles access to it in AI Control.",
		)
	}

	return errortypes.NewBusinessError(
		"Opening this in the Desk needs the general assistant, which is turned off. " +
			"Turn it on in AI Control.",
	)
}

func generalAssistants(
	tenant pagination.TenantInfo,
	limit int,
	audience *kernelports.AgentAudience,
) *agentdefinitionports.ListAgentDefinitionRequest {
	return &agentdefinitionports.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: limit},
			FieldFilters: []domaintypes.FieldFilter{
				{
					Field:    definitionTemplateField,
					Operator: dbtype.OpEqual,
					Value:    string(agenttemplate.TemplateGeneralAssistant),
				},
			},
		},
		EnabledOnly: true,
		ChatOnly:    true,
		Audience:    audience,
	}
}
