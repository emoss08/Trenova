package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Provisioner struct {
	Definitions map[string]*agentdefinition.Definition
	Asked       []agentdefinitionports.EnsureSystemAgentRequest
}

func (p *Provisioner) EnsureSystem(
	_ context.Context,
	req agentdefinitionports.EnsureSystemAgentRequest,
) (*agentdefinition.Definition, error) {
	p.Asked = append(p.Asked, req)
	definition, ok := p.Definitions[req.SystemKey]
	if !ok {
		definition, _ = agentpreset.NewPageAgent(req.SystemKey, req.TenantInfo, req.SystemKey)
		definition.ID = pulid.MustNew("agdef_")
		p.Definitions[req.SystemKey] = definition
	}

	return definition, nil
}

type PageThreads struct {
	Live     map[conversationports.PageThreadKey]*conversation.Thread
	Claimed  []*conversation.Thread
	archived []conversationports.ArchiveSubjectThreadsRequest
}

func keyOf(thread *conversation.Thread) conversationports.PageThreadKey {
	return conversationports.PageThreadKey{
		TenantInfo: pagination.TenantInfo{
			OrgID: thread.OrganizationID,
			BuID:  thread.BusinessUnitID,
		},
		UserID:      thread.UserID,
		Origin:      thread.Origin,
		SubjectType: thread.SubjectType,
		SubjectID:   thread.SubjectID,
	}
}

func (p *PageThreads) GetPageThread(
	_ context.Context,
	key *conversationports.PageThreadKey,
) (*conversation.Thread, error) {
	lookup := *key
	lookup.TenantInfo.UserID = pulid.Nil
	if thread, ok := p.Live[lookup]; ok {
		return thread, nil
	}

	return nil, errortypes.NewNotFoundError("Thread not found")
}

func (p *PageThreads) ClaimPageThread(
	_ context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	key := keyOf(thread)
	if existing, ok := p.Live[key]; ok {
		return existing, nil
	}
	thread.ID = pulid.MustNew("athr_")
	p.Live[key] = thread
	p.Claimed = append(p.Claimed, thread)

	return thread, nil
}

func (p *PageThreads) ArchiveSubjectThreads(
	_ context.Context,
	req *conversationports.ArchiveSubjectThreadsRequest,
) (int, error) {
	p.archived = append(p.archived, *req)
	closed := 0
	for key, thread := range p.Live {
		if key.Origin == req.Origin && key.SubjectID == req.SubjectID {
			thread.Status = conversation.ThreadStatusArchived
			delete(p.Live, key)
			closed++
		}
	}

	return closed, nil
}
