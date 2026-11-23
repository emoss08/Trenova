package desktranscriptexport

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	messagesPerRead     = 2000
	maxThreadMessages   = 10_000
	deletedPersonName   = "A removed person"
	deletedAgentName    = "A removed agent"
	unnamedAgentDefault = "Assistant"
)

type Owner struct {
	Name  string
	Email string
}

type Assembler struct {
	conversations desktranscriptports.ConversationRepository
	users         tenantports.UserRepository
	definitions   agentdefinitionports.AgentDefinitionRepository
	renderer      desktranscriptports.TranscriptRenderer
}

type AssemblerParams struct {
	Conversations desktranscriptports.ConversationRepository
	Users         tenantports.UserRepository
	Definitions   agentdefinitionports.AgentDefinitionRepository
	Renderer      desktranscriptports.TranscriptRenderer
}

func NewAssembler(p *AssemblerParams) *Assembler {
	return &Assembler{
		conversations: p.Conversations,
		users:         p.Users,
		definitions:   p.Definitions,
		renderer:      p.Renderer,
	}
}

func (a *Assembler) Owners(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]Owner, error) {
	owners := make(map[pulid.ID]Owner, len(ids))
	ids = sliceutils.Dedupe(ids)
	if len(ids) == 0 {
		return owners, nil
	}

	users, err := a.users.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    ids,
	})
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user != nil {
			owners[user.ID] = ownerOf(user)
		}
	}

	return owners, nil
}

func ownerOf(user *tenant.User) Owner {
	return Owner{Name: user.Name, Email: user.EmailAddress}
}

func (a *Assembler) AgentNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	names := make(map[pulid.ID]string, len(ids))
	ids = sliceutils.Dedupe(ids)
	if len(ids) == 0 {
		return names, nil
	}

	definitions, err := a.definitions.ListByIDs(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByIDsRequest{IDs: ids, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	for _, definition := range definitions {
		if definition != nil {
			names[definition.ID] = definition.Name
		}
	}

	return names, nil
}

func ownerName(owners map[pulid.ID]Owner, id pulid.ID) string {
	if owner, ok := owners[id]; ok && owner.Name != "" {
		return owner.Name
	}

	return deletedPersonName
}

func agentName(names map[pulid.ID]string, id pulid.ID) string {
	if id.IsNil() {
		return unnamedAgentDefault
	}
	if name, ok := names[id]; ok && name != "" {
		return name
	}

	return deletedAgentName
}

func (a *Assembler) Describe(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	threads []*conversation.Thread,
) ([]*desktranscript.Conversation, error) {
	out := make([]*desktranscript.Conversation, len(threads))
	if len(threads) == 0 {
		return out, nil
	}

	userIDs := make([]pulid.ID, 0, len(threads))
	agentIDs := make([]pulid.ID, 0, len(threads))
	threadIDs := make([]pulid.ID, 0, len(threads))
	for _, thread := range threads {
		userIDs = append(userIDs, thread.UserID)
		agentIDs = append(agentIDs, thread.AgentDefinitionID)
		threadIDs = append(threadIDs, thread.ID)
	}

	owners, err := a.Owners(ctx, tenantInfo, userIDs)
	if err != nil {
		return nil, err
	}
	agents, err := a.AgentNames(ctx, tenantInfo, agentIDs)
	if err != nil {
		return nil, err
	}
	counts, err := a.conversations.CountMessages(ctx, desktranscriptports.ConversationsByIDsRequest{
		TenantInfo: tenantInfo,
		IDs:        threadIDs,
	})
	if err != nil {
		return nil, err
	}

	for idx, thread := range threads {
		item := desktranscript.ConversationOf(thread)
		item.OwnerName = ownerName(owners, thread.UserID)
		item.AgentName = agentName(agents, thread.AgentDefinitionID)
		item.MessageCount = counts[thread.ID]
		out[idx] = item
	}

	return out, nil
}

type RenderBatch struct {
	TenantInfo pagination.TenantInfo
	Threads    []*conversation.Thread
	Location   *time.Location
	ExportedAt int64
}

type Visit func(thread *conversation.Thread, rendered *desktranscriptports.RenderedConversation) error

func (a *Assembler) Render(ctx context.Context, batch *RenderBatch, visit Visit) error {
	if len(batch.Threads) == 0 {
		return nil
	}

	userIDs := make([]pulid.ID, 0, len(batch.Threads))
	agentIDs := make([]pulid.ID, 0, len(batch.Threads))
	threadIDs := make([]pulid.ID, 0, len(batch.Threads))
	for _, thread := range batch.Threads {
		userIDs = append(userIDs, thread.UserID)
		agentIDs = append(agentIDs, thread.AgentDefinitionID)
		threadIDs = append(threadIDs, thread.ID)
	}
	owners, err := a.Owners(ctx, batch.TenantInfo, userIDs)
	if err != nil {
		return err
	}
	agents, err := a.AgentNames(ctx, batch.TenantInfo, agentIDs)
	if err != nil {
		return err
	}
	counts, err := a.conversations.CountMessages(ctx, desktranscriptports.ConversationsByIDsRequest{
		TenantInfo: batch.TenantInfo,
		IDs:        threadIDs,
	})
	if err != nil {
		return err
	}

	for _, group := range readGroups(batch.Threads, counts) {
		if err = ctx.Err(); err != nil {
			return err
		}
		messages, readErr := a.messagesOf(ctx, batch.TenantInfo, group, counts)
		if readErr != nil {
			return readErr
		}
		for _, thread := range group {
			owner := owners[thread.UserID]
			rendered, renderErr := a.renderer.RenderConversation(
				ctx,
				&desktranscriptports.RenderRequest{
					TenantInfo: batch.TenantInfo,
					Thread:     thread,
					Messages:   messages[thread.ID],
					OwnerName:  ownerName(owners, thread.UserID),
					OwnerEmail: owner.Email,
					AgentName:  agentName(agents, thread.AgentDefinitionID),
					Location:   batch.Location,
					ExportedAt: batch.ExportedAt,
				},
			)
			if renderErr != nil {
				return renderErr
			}
			if err = visit(thread, rendered); err != nil {
				return err
			}
		}
	}

	return nil
}

func readGroups(
	threads []*conversation.Thread,
	counts map[pulid.ID]int,
) [][]*conversation.Thread {
	groups := make([][]*conversation.Thread, 0, 1)
	current := make([]*conversation.Thread, 0, len(threads))
	held := 0
	for _, thread := range threads {
		size := min(counts[thread.ID], maxThreadMessages)
		if len(current) > 0 && held+size > messagesPerRead {
			groups = append(groups, current)
			current = make([]*conversation.Thread, 0, len(threads))
			held = 0
		}
		current = append(current, thread)
		held += size
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}

	return groups
}

func (a *Assembler) messagesOf(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	group []*conversation.Thread,
	counts map[pulid.ID]int,
) (map[pulid.ID][]conversation.Message, error) {
	ids := make([]pulid.ID, 0, len(group))
	limit := 0
	for _, thread := range group {
		ids = append(ids, thread.ID)
		limit += min(counts[thread.ID], maxThreadMessages)
	}
	byThread := make(map[pulid.ID][]conversation.Message, len(group))
	if limit == 0 {
		return byThread, nil
	}

	messages, err := a.conversations.ListMessages(ctx, &desktranscriptports.ThreadMessagesRequest{
		TenantInfo: tenantInfo,
		ThreadIDs:  ids,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	for idx := range messages {
		threadID := messages[idx].ThreadID
		byThread[threadID] = append(byThread[threadID], messages[idx])
	}

	return byThread, nil
}
