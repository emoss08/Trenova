package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/replycheck"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/extractionartifact"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/toolartifacts"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

// showAttachments puts the extraction of each file attached to this message
// beside the conversation before the agent answers, and tells the agent
// where it is so the reply can point to it rather than read it again.
func (s *Service) showAttachments(
	ctx context.Context,
	thread *conversation.Thread,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
	attachments []agentturn.RuntimeAttachment,
) {
	if len(attachments) == 0 || thread.Origin.AnswersInline() {
		return
	}
	recorder := s.newArtifactRecorder(ctx, thread, tenant, actor, nil)
	if recorder == nil {
		return
	}

	for idx := range attachments {
		documentID, err := pulid.Parse(attachments[idx].DocumentID)
		if err != nil {
			continue
		}
		if saved := recorder.AttachedExtraction(documentID); saved != nil {
			attachments[idx].Shown = replycheck.ArtifactRef(toolartifacts.ShownArtifact(saved))
		}
	}
}

// showFinishedAttachments shows the extraction of a file whose reading
// finished while the agent was answering. It announces the card on the
// turn's stream, so the pane opens on it as on any card the turn made.
func (s *Service) showFinishedAttachments(
	ctx context.Context,
	thread *conversation.Thread,
	plan *TurnPlan,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
	emit kernelports.AssistantStreamEmitter,
) {
	if (len(plan.Attachments) == 0 && len(plan.Readings) == 0) || plan.Inline {
		return
	}
	recorder := s.newArtifactRecorder(ctx, thread, tenant, actor, emit)
	if recorder == nil {
		return
	}

	for idx := range plan.Attachments {
		recorder.AttachedExtraction(plan.Attachments[idx].DocumentID)
	}
	for _, documentID := range plan.Readings {
		recorder.AttachedExtraction(documentID)
	}
}

// AttachmentExtracted shows the extraction of a file attached in a
// conversation once document intelligence has read it. Reading a file takes
// longer than most turns, so this is how the card usually arrives: after the
// reply, in the conversation the file was sent in, read as the person who
// sent it. A file uploaded and never sent gets no card.
func (s *Service) AttachmentExtracted(ctx context.Context, event *agenteventports.AgentEvent) {
	if event == nil || event.Kind != agenttrigger.EventDocumentExtracted ||
		s.documents == nil || s.artifacts == nil {
		return
	}
	log := s.logger.With(zap.String("documentId", event.SubjectID.String()))

	doc, err := s.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         event.SubjectID,
		TenantInfo: event.TenantInfo,
	})
	if err != nil {
		log.Debug("extracted document could not be read for its card", zap.Error(err))
		return
	}
	threads := document.ConversationIDs([]*document.Document{doc})
	if len(threads) != 1 || doc.UploadedByID.IsNil() {
		return
	}

	tenant := pagination.TenantInfo{
		OrgID:  doc.OrganizationID,
		BuID:   doc.BusinessUnitID,
		UserID: doc.UploadedByID,
	}
	sent, err := s.conversations.CarriesAttachment(ctx, &conversationports.CarriesAttachmentRequest{
		ThreadID:   threads[0],
		DocumentID: doc.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		log.Warn("could not tell whether an extracted file was sent", zap.Error(err))
		return
	}
	if !sent {
		return
	}

	thread, err := s.conversations.GetThread(ctx, conversationports.GetThreadRequest{
		ID:         threads[0],
		UserID:     doc.UploadedByID,
		TenantInfo: tenant,
	})
	if err != nil {
		log.Debug("conversation of an extracted file could not be read", zap.Error(err))
		return
	}
	if thread.Origin.AnswersInline() {
		return
	}

	recorder := s.newArtifactRecorder(ctx, thread, tenant, permissionports.UserActor(tenant), nil)
	recorder.AttachedExtraction(doc.ID)
}

// UnreadFileRun is the work a tool call that found a file still being read
// belongs to.
type UnreadFileRun struct {
	ThreadID     pulid.ID
	DefinitionID pulid.ID
	Actor        *permissionports.RequestActor
	// Delegated is a task another agent handed over, which reports back to
	// that agent instead of waiting.
	Delegated bool
}

// WaitOnUnreadFile parks the conversation on a file agent_read_attachment found
// still being read. The conversation picks up again on its own when the
// reading finishes, with the file read, so the person never has to ask how
// it is going; until then the composer shows what it is waiting on. A file
// read in full ends any wait on it, so a conversation that got the file
// another way is not picked up again for nothing.
func (s *Service) WaitOnUnreadFile(
	ctx context.Context,
	run *UnreadFileRun,
	observation *agentruntimeports.ToolObservation,
) {
	if s.readWaits == nil || run == nil || run.Delegated || run.ThreadID.IsNil() ||
		run.Actor == nil || observation == nil || observation.Failed ||
		observation.Call.Name != agentdefinition.CoreToolReadAttachment {
		return
	}
	read, ok := artifactpayload.ToJSONDocument(observation.Data)
	if !ok {
		return
	}
	reading := typeutils.StringOfTrimmed(read.Fields["reading"])
	documentID, err := pulid.Parse(
		typeutils.StringOfTrimmed(read.Fields[extractionartifact.DraftDocumentIDParam]),
	)
	if reading == "" || err != nil {
		return
	}

	req := &agentwaitports.WaitForDocumentReadRequest{
		Actor:        run.Actor,
		DefinitionID: run.DefinitionID,
		ThreadID:     run.ThreadID,
		DocumentID:   documentID,
		FileName:     typeutils.StringOfTrimmed(read.Fields["fileName"]),
	}
	if !typeutils.BoolOf(read.Fields["stillBeingRead"]) {
		if err = s.readWaits.StopWaitingForDocumentRead(ctx, req); err != nil {
			s.logger.Warn("could not stop waiting on a file the agent has read",
				zap.String("documentId", documentID.String()), zap.Error(err))
		}
		return
	}

	_, err = s.readWaits.WaitForDocumentRead(ctx, req)
	switch {
	case err == nil:
	case errortypes.IsBusinessError(err):
		s.logger.Debug("no wait set on a file still being read",
			zap.String("documentId", documentID.String()), zap.Error(err))
	default:
		s.logger.Warn("could not wait on a file still being read",
			zap.String("documentId", documentID.String()), zap.Error(err))
	}
}

// showReading shows the extraction of a file whose reading a wait was
// watching, as the turn that picks the wait up begins, and tells the agent
// where the card is.
func (s *Service) showReading(
	ctx context.Context,
	thread *conversation.Thread,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
	documentID pulid.ID,
) string {
	if documentID.IsNil() || thread.Origin.AnswersInline() {
		return ""
	}
	saved := s.newArtifactRecorder(ctx, thread, tenant, actor, nil).AttachedExtraction(documentID)
	if saved == nil {
		return ""
	}

	return "\n\nWhat was read off the file is shown beside the conversation as " +
		replycheck.ArtifactRef(
			toolartifacts.ShownArtifact(saved),
		) + ". When the reply is about what " +
		"was read, point to it by writing that link, exactly so, inside the sentence; do not " +
		"publish the reading as a document."
}

func readingsOf(documentID pulid.ID) []pulid.ID {
	if documentID.IsNil() {
		return nil
	}

	return []pulid.ID{documentID}
}
