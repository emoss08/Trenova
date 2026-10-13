package assistantservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen/agentdefinitioncol"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// askAgentPageLimit bounds the general assistants read to pick the one a
// quick question goes to; an organization has one, or a few.
const askAgentPageLimit = 100

// StartAsk opens the thread a quick question from the palette is answered on.
// It is the same turn as any other, on a thread created for the question and
// hidden until it is kept: the general assistant answers it, and what the turn
// produced is saved there so "Open in Desk" has somewhere to go.
func (s *Service) StartAsk(
	ctx context.Context,
	req *agentports.AskRequest,
	actor *permissionports.RequestActor,
) (*conversation.Thread, error) {
	content := strings.TrimSpace(req.Content)
	multiErr := errortypes.NewMultiError()
	if content == "" {
		multiErr.Add("content", errortypes.ErrRequired, "Question cannot be empty")
	}
	if page := req.Page.Normalized(); page != nil {
		page.Validate("context", multiErr)
	}
	validateMentions(req.Mentions, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	definition, err := s.askAgent(ctx, req.TenantInfo, actor)
	if err != nil {
		return nil, err
	}

	return s.StartThread(ctx, &agentports.StartThreadRequest{
		AgentDefinitionID: definition.ID,
		Title:             truncateRunes(content, maxTitleRunes),
		TenantInfo:        req.TenantInfo,
		Origin:            conversation.ThreadOriginAsk,
	}, actor)
}

// askAgent is who answers a quick question: the organization's general
// assistant, when it is on and open to the person. No other agent stands in
// for it, since each is built for its own desk and would answer every
// question from that desk's records.
func (s *Service) askAgent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) (*agentdefinition.Definition, error) {
	usable, err := s.usableAgents(ctx, actor)
	if err != nil {
		return nil, err
	}
	if !usable.Assistant {
		return nil, errAgentsWithheld()
	}

	result, err := s.definitions.List(
		ctx,
		generalAssistants(tenant, askAgentPageLimit, usable.Audience()),
	)
	if err != nil {
		return nil, err
	}
	for _, definition := range result.Items {
		if definition != nil && usable.Allows(definition) {
			return definition, nil
		}
	}

	return nil, s.noAskAgent(ctx, tenant)
}

// noAskAgent says why there is nobody to ask: no general assistant is on, or
// none that is on is open to the person.
func (s *Service) noAskAgent(ctx context.Context, tenant pagination.TenantInfo) error {
	enabled, err := s.definitions.List(ctx, generalAssistants(tenant, 1, nil))
	if err != nil {
		return err
	}
	if len(enabled.Items) > 0 {
		return errortypes.NewAuthorizationError(
			"Nova answers quick questions through the general assistant, which is not open to " +
				"you. An administrator can give one of your roles access to it in AI Control.",
		)
	}

	return errortypes.NewBusinessError(
		"Nova answers quick questions through the general assistant, which is turned off. " +
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
				agentdefinitioncol.DefinitionFilter.Template(
					dbtype.OpEqual,
					string(agentdefinition.TemplateGeneralAssistant),
				),
			},
		},
		EnabledOnly: true,
		ChatOnly:    true,
		Audience:    audience,
	}
}

func (s *Service) everyRead() []string {
	if s.queries == nil {
		return nil
	}

	tools := s.queries.All()
	reads := make([]string, 0, len(tools))
	for _, tool := range tools {
		if _, extension := agentextension.ExtensionForTool(tool.Name()); extension {
			continue
		}
		reads = append(reads, tool.Name())
	}

	return reads
}
