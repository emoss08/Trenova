package assistanttest

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type StubDefinitionList struct {
	agentdefinitionports.AgentDefinitionRepository

	Definitions []*agentdefinition.Definition
	Last        *agentdefinitionports.ListAgentDefinitionRequest
	Requests    []*agentdefinitionports.ListAgentDefinitionRequest
}

func (s *StubDefinitionList) List(
	_ context.Context,
	req *agentdefinitionports.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	s.Last = req
	s.Requests = append(s.Requests, req)

	items := make([]*agentdefinition.Definition, 0, len(s.Definitions))
	for _, definition := range s.Definitions {
		if matchesTemplateFilter(req.Filter, definition) {
			items = append(items, definition)
		}
	}

	return &pagination.ListResult[*agentdefinition.Definition]{Items: items}, nil
}

func matchesTemplateFilter(
	filter *pagination.QueryOptions,
	definition *agentdefinition.Definition,
) bool {
	if filter == nil {
		return true
	}
	for _, field := range filter.FieldFilters {
		if field.Field == "template" && field.Operator == dbtype.OpEqual &&
			field.Value != string(definition.Template) {
			return false
		}
	}

	return true
}

func (s *StubDefinitionList) GetByID(
	_ context.Context,
	req agentdefinitionports.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	for _, definition := range s.Definitions {
		if definition.ID == req.ID {
			return definition, nil
		}
	}

	return nil, errors.New("not found")
}

type CreatingConversations struct {
	StubConversations

	Created *conversation.Thread
}

func (c *CreatingConversations) CreateThread(
	_ context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	thread.ID = pulid.MustNew("athr_")
	c.Created = thread
	c.Thread = thread

	return thread, nil
}

func ChatDefinition(name string, template agenttemplate.Template) *agentdefinition.Definition {
	d := Definition()
	d.ID = pulid.MustNew("agdef_")
	d.Name = name
	d.Template = template
	d.TriggerMode = agentdefinition.TriggerChat

	return d
}
