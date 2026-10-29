package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/modelstream"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ToolSummaries(definition *agentdefinition.Definition) []agentturn.ToolSummary {
	return s.access.ToolSummaries(definition)
}

// PermittedTools keeps the named tools the actor may use, the same check the
// turn's own tool set is narrowed by.
func (s *Service) PermittedTools(
	ctx context.Context,
	actor *permissionports.RequestActor,
	names []string,
) []string {
	return s.access.PermittedTools(ctx, actor, names)
}

func (s *Service) AvailableCommands(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) []agentextension.Command {
	return s.access.AvailableCommands(ctx, actor, definition)
}

func (s *Service) Fingerprint(
	definition *agentdefinition.Definition,
	providerID pulid.ID,
	model string,
) *agentfingerprint.Fingerprint {
	return s.access.Fingerprint(definition, providerID, model)
}

func (s *Service) MarkToolEffects(messages []conversation.Message) {
	s.access.MarkToolEffects(messages)
}

func (s *Service) ReadRecord(
	ctx context.Context,
	actor *permissionports.RequestActor,
	ref *agentrecord.RecordRef,
) (*toolaccess.RecordRead, bool) {
	return s.access.ReadRecord(ctx, actor, ref)
}

func (s *Service) ReadRecordWith(
	ctx context.Context,
	actor *permissionports.RequestActor,
	name string,
	id string,
) (*toolaccess.RecordRead, bool) {
	return s.access.ReadRecordWith(ctx, actor, name, id)
}

// StreamCompletion asks the model for the next step of a turn, streaming the
// reply to emit as it arrives.
func (s *Service) StreamCompletion(
	ctx context.Context,
	req *llmports.ChatCompletionRequest,
	emit kernelports.AssistantStreamEmitter,
) (modelstream.Reply, error) {
	return s.streamer.StreamCompletion(ctx, req, emit)
}
