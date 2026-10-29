package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactrecorder"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) recorderReads() *artifactrecorder.Reads {
	return &artifactrecorder.Reads{
		Attribution:   s.attribution(),
		Conversations: s.conversations,
		Runtime:       s.runtime,
		Boards:        s.boards,
		Documents:     s.documents,
		Contents:      s.contents,
	}
}

// newArtifactRecorder returns nil when there is nowhere to keep artifacts,
// and every method on a nil recorder is a no-op: a turn without a pane is
// still a turn.
func (s *Service) newArtifactRecorder(
	ctx context.Context,
	thread *conversation.Thread,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
	emit kernelports.AssistantStreamEmitter,
) *artifactrecorder.Recorder {
	return artifactrecorder.Open(ctx, artifactrecorder.Deps{
		Repo:      s.artifacts,
		Proposals: s.proposals,
		Activity:  s.activity,
		Logger:    s.logger,
		Reads:     s.recorderReads(),
	}, thread, tenant, actor, emit)
}
