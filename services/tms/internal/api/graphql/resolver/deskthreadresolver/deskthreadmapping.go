package deskthreadresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/deskmapping"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantturnservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationchapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/assistantjobs/assistantcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantturnports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

type results struct {
	Results any `json:"results"`
}

type chapterItems struct {
	Items []*conversationchapter.Chapter `json:"items"`
}

func chapterList(items []*conversationchapter.Chapter) chapterItems {
	if items == nil {
		items = []*conversationchapter.Chapter{}
	}

	return chapterItems{Items: items}
}

func liveTurns(turns []*assistantturnports.LiveAssistantTurn) []*gqlmodel.AssistantLiveTurn {
	items := make([]*gqlmodel.AssistantLiveTurn, 0, len(turns))
	for _, turn := range turns {
		items = append(items, &gqlmodel.AssistantLiveTurn{
			TurnID:      turn.ID.String(),
			ThreadID:    turn.ThreadID.String(),
			ThreadTitle: turn.ThreadTitle,
			Origin:      string(turn.Origin),
			StartedAt:   int(turn.StartedAt),
		})
	}

	return items
}

func threadUpdate(
	thread *conversationports.GetThreadRequest,
	input *gqlmodel.UpdateAssistantThreadInput,
) *assistantports.UpdateThreadRequest {
	update := &assistantports.UpdateThreadRequest{
		ThreadID:    thread.ID,
		TenantInfo:  thread.TenantInfo,
		Title:       input.Title,
		Pinned:      input.Pinned,
		AutoCompact: input.AutoCompact,
	}
	if input.PinnedFacts != nil {
		update.PinnedFacts = &input.PinnedFacts
	}

	return update
}

type message struct {
	content         string
	page            *agentpagecontext.PageContext
	surface         agentpagecontext.Surface
	providerID      pulid.ID
	providerChosen  bool
	attachments     []pulid.ID
	mentions        []agentpagecontext.EntityRef
	directedAgentID pulid.ID
}

func turnMessage(input *gqlmodel.AssistantMessageInput) (*message, error) {
	page, err := deskmapping.PageContext(input.Context)
	if err != nil {
		return nil, err
	}
	providerID, chosen, err := deskmapping.Provider("providerId", input.ProviderID)
	if err != nil {
		return nil, err
	}
	attachments, err := deskmapping.IDs("attachmentDocumentIds", input.AttachmentDocumentIds)
	if err != nil {
		return nil, err
	}
	directed, err := gqlid.ParseOptionalIDField("directedAgentId", input.DirectedAgentID)
	if err != nil {
		return nil, err
	}

	return &message{
		content:         input.Content,
		page:            page,
		surface:         agentpagecontext.Surface(gqlscalar.DerefString(input.Surface)),
		providerID:      providerID,
		providerChosen:  chosen,
		attachments:     attachments,
		mentions:        deskmapping.Mentions(input.Mentions),
		directedAgentID: directed,
	}, nil
}

func (r *MutationResolver) askWorker(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	thread *conversationports.GetThreadRequest,
	msg *message,
) (*conversationturn.AssistantTurn, error) {
	if _, err := r.AssistantService.GetThread(ctx, thread); err != nil {
		return nil, err
	}

	actor := *actorutil.FromAuthContext(authCtx)

	return r.AssistantTurnService.StartTurn(
		ctx,
		assistantturnservice.StartRequest{
			ThreadID:   thread.ID,
			UserID:     thread.UserID,
			TenantInfo: thread.TenantInfo,
			Origin:     conversationturn.AssistantTurnOriginPerson,
			Input:      msg.content,
		},
		func(turn *conversationturn.AssistantTurn) (string, error) {
			run, err := assistantcontract.StartTurnWorkflow(ctx, r.WorkflowStarter, turn,
				assistantcontract.TurnStart{
					Actor:   actor,
					Content: msg.content,
					Request: assistantcontract.AssistantTurnRequest{
						Page:                  msg.page,
						Surface:               msg.surface,
						Mentions:              msg.mentions,
						AttachmentDocumentIDs: msg.attachments,
						PreferredProviderID:   msg.providerID,
						ProviderChosen:        msg.providerChosen,
						DirectedAgentID:       msg.directedAgentID,
					},
				})
			if err != nil {
				return "", err
			}

			return run.GetID(), nil
		},
	)
}
