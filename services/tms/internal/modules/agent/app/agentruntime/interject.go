package agentruntime

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolcall"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/worldwatch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/zap"
)

func (s *Service) watchWorld(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	now int64,
) *worldwatch.State {
	if s.changes == nil || req.Delegation != nil || req.Actor == nil {
		return nil
	}
	cursor, err := s.changes.Head(ctx, req.Actor.OrganizationID, req.Actor.BusinessUnitID)
	if err != nil {
		s.logger.Debug("the turn will not watch its records for changes", zap.Error(err))
		return nil
	}

	return worldwatch.OpenWorld(cursor, worldwatch.WatchedRecords(req), now)
}

func (t *Turn) noteWorld(fx TurnEffects, call *llmports.ToolCall, outcome *toolcall.Result) {
	if t.world == nil {
		return
	}
	t.world.Read(call, outcome.Summary)
	if outcome.Action != nil && outcome.Action.Executed {
		t.world.Wrote(outcome.Action, fx.Now())
	}
}

func (s *Service) interject(t *Turn, fx TurnEffects) {
	if t.req.Delegation != nil {
		return
	}
	var now int64
	var check *worldwatch.Check
	if t.world != nil {
		now = fx.Now()
		check = t.world.Due(now)
	}
	got := fx.Interject(t, check)
	for _, steer := range got.Steers {
		t.steer(fx, steer)
	}
	if check == nil {
		return
	}
	if changes := t.world.News(got.World, now); len(changes) > 0 {
		t.worldChanged(fx, changes)
	}
}

func (t *Turn) steer(fx TurnEffects, steer worldwatch.Steer) {
	content := strings.TrimSpace(steer.Content)
	if content == "" {
		return
	}
	mentions := agentpagecontext.NormalizeEntityRefs(steer.Mentions)
	for _, mention := range mentions {
		t.world.Watch(mention)
	}

	t.messages = append(
		t.messages,
		llmports.Message{
			Role:    llmports.RoleUser,
			Content: worldwatch.SteerPrompt(content, mentions),
		},
		llmports.NoticeMessage(worldwatch.SteerNotice),
	)
	t.result.Messages = append(t.result.Messages, conversation.Message{
		Role:      conversation.RoleUser,
		Kind:      conversation.MessageKindSteer,
		Content:   content,
		Mentions:  mentions,
		CreatedAt: fx.Now(),
	})
	fx.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventSteered,
		Data: assistantstreamports.AssistantSteeredEvent{
			ID:       steer.ID,
			Content:  content,
			Mentions: mentions,
		},
	})
}

func (t *Turn) worldChanged(fx TurnEffects, changes []realtimeports.WatchedRecordChange) {
	notice := worldwatch.ChangeNotice(changes, t.req.Actor)
	clear(t.repeats.Reads)

	t.messages = append(t.messages, llmports.NoticeMessage(notice))
	t.result.Messages = append(t.result.Messages, conversation.Message{
		Role:         conversation.RoleUser,
		Kind:         conversation.MessageKindWorldChange,
		Content:      notice,
		WorldChanges: changes,
		CreatedAt:    fx.Now(),
	})
	fx.Emit(kernelports.StreamEvent{
		Event: assistantstreamports.AssistantEventWorldChanged,
		Data:  assistantstreamports.AssistantWorldChangedEvent{Changes: changes},
	})
}
