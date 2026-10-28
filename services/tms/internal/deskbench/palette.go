package deskbench

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
)

func (s *Session) OpenPalette(
	ctx context.Context,
	label, title, question string,
) (*Conversation, error) {
	ctx = s.Context(ctx)

	thread, err := s.bench.Assistant.StartAsk(ctx, &assistantports.AskRequest{
		Content:    question,
		TenantInfo: s.Tenant,
	}, &s.Actor)
	if err != nil {
		return nil, fmt.Errorf("ask from the command palette: %w", err)
	}

	benchTitle := benchTitlePrefix + strings.TrimSpace(title)
	thread, err = s.bench.Assistant.UpdateThread(ctx, &assistantports.UpdateThreadRequest{
		ThreadID:   thread.ID,
		TenantInfo: s.Tenant,
		Title:      &benchTitle,
	}, &s.Actor)
	if err != nil {
		return nil, fmt.Errorf("title the palette conversation: %w", err)
	}

	definition, err := s.Agent(ctx, thread.AgentDefinitionID.String())
	if err != nil {
		return nil, err
	}

	return &Conversation{
		Label:   label,
		session: s,
		Agent:   definition,
		Thread:  thread,
		queue:   s.bench.watcher.watch(thread.ID),
		palette: true,
	}, nil
}
