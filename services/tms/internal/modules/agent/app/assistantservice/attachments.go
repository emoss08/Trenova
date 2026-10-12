package assistantservice

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// MaxAttachments bounds one message's files. Each is read into the
	// prompt, so the bound is a context budget as much as a courtesy.
	MaxAttachments = conversation.MaxMessageAttachments
	// AttachmentResourceType is the resource an attachment is uploaded
	// against: the thread, so ownership is a plain comparison.
	AttachmentResourceType = document.ConversationResourceType
)

// turnContext is what the person handed over with a message besides the
// words: the page, the files, and the records they named. It is stored on
// the user turn and rendered into the prompt, and nowhere else.
type turnContext struct {
	page        *agent.PageContext
	attachments []conversation.MessageAttachment
	runtime     []agentdefinition.RuntimeAttachment
	mentions    []agent.EntityRef
}

// validateMentions bounds and checks the records a person named.
func validateMentions(
	mentions []agent.EntityRef,
	multiErr *errortypes.MultiError,
) []agent.EntityRef {
	normalized := agent.NormalizeEntityRefs(mentions)
	agent.ValidateEntityRefs("mentions", normalized, multiErr)

	return normalized
}

// resolveAttachments turns document ids into what the turn stores and what
// the prompt says. Every document must be one this person uploaded to this
// thread; an id that is not is refused as a validation error rather than
// read, because the alternative is a way to read any document by attaching
// it. The content read is best effort: a file whose extraction has not
// finished is named with its status, not dropped.
func (s *Service) resolveAttachments(
	ctx context.Context,
	thread *conversation.Thread,
	ids []pulid.ID,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
) ([]conversation.MessageAttachment, []agentdefinition.RuntimeAttachment, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	multiErr := errortypes.NewMultiError()
	if len(ids) > MaxAttachments {
		multiErr.Add("attachmentDocumentIds", errortypes.ErrInvalid,
			"At most "+strconv.Itoa(MaxAttachments)+" files can be attached to one message")

		return nil, nil, multiErr
	}
	if s.documents == nil {
		return nil, nil, errortypes.NewBusinessError(
			"Attachments are not available on this deployment",
		)
	}

	// Each file is read on its own, side by side: the files are independent,
	// and one after the other they were two round trips each. What comes back
	// is put in the order the files were attached, and so are the refusals.
	slots := make([]attachmentSlot, len(ids))
	seen := make(map[pulid.ID]struct{}, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		if id.IsNil() {
			slots[i].problem = "Attachment identifier is invalid"
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		slot := &slots[i]
		wg.Go(func() {
			*slot = s.readAttachment(ctx, thread, id, actor, tenant)
		})
	}
	wg.Wait()

	stored := make([]conversation.MessageAttachment, 0, len(ids))
	runtime := make([]agentdefinition.RuntimeAttachment, 0, len(ids))
	for i := range slots {
		slot := &slots[i]
		if slot.problem != "" {
			multiErr.Add(
				"attachmentDocumentIds["+strconv.Itoa(i)+"]",
				errortypes.ErrInvalid,
				slot.problem,
			)
			continue
		}
		if slot.stored == nil {
			continue
		}
		stored = append(stored, *slot.stored)
		runtime = append(runtime, slot.runtime)
	}
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	return stored, runtime, nil
}

// attachmentSlot is one attached file as it was read: what is kept on the
// message and shown to the model, or why it was refused. A duplicate leaves
// its slot empty.
type attachmentSlot struct {
	stored  *conversation.MessageAttachment
	runtime agentdefinition.RuntimeAttachment
	problem string
}

func (s *Service) readAttachment(
	ctx context.Context,
	thread *conversation.Thread,
	id pulid.ID,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
) attachmentSlot {
	doc, err := s.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
	if err != nil {
		return attachmentSlot{problem: "Attachment was not found"}
	}
	if !doc.AttachedBy(thread.ID, actor.UserID) {
		return attachmentSlot{problem: "Attachment does not belong to this conversation"}
	}

	runtime := s.describeAttachment(ctx, doc.ID, doc.OriginalName, doc.FileType, tenant)
	runtime.FileSize = doc.FileSize

	return attachmentSlot{
		stored: &conversation.MessageAttachment{
			DocumentID:  doc.ID,
			FileName:    doc.OriginalName,
			ContentType: doc.FileType,
			FileSize:    doc.FileSize,
			PoorlyRead:  runtime.PoorlyRead,
		},
		runtime: runtime,
	}
}

// describeAttachment reads what document intelligence has made of a file so
// far. Nothing here fails the turn: a file that cannot be read yet is named
// as pending, and the model is told to come back to it with the tool.
func (s *Service) describeAttachment(
	ctx context.Context,
	documentID pulid.ID,
	fileName, contentType string,
	tenant pagination.TenantInfo,
) agentdefinition.RuntimeAttachment {
	attachment := agentdefinition.RuntimeAttachment{
		DocumentID:  documentID.String(),
		FileName:    fileName,
		ContentType: contentType,
		Reading:     agentdefinition.AttachmentReadingPending,
	}
	if s.contents == nil {
		return attachment
	}

	content, err := s.contents.GetContent(ctx, documentID, tenant)
	if err != nil || content == nil {
		if err != nil {
			s.logger.Debug("attachment content not readable yet",
				zap.String("document", documentID.String()),
				zap.Error(err),
			)
		}

		return attachment
	}

	attachment.Reading = readingOf(content.Status)
	attachment.PageCount = content.PageCount
	attachment.Kind = content.DetectedDocumentKind
	attachment.PoorlyRead = content.PoorlyRead()
	attachment.Text = strings.TrimSpace(content.ContentText)

	return attachment
}

func (s *Service) earlierFileTexts(
	ctx context.Context,
	history []conversation.Message,
	tenant pagination.TenantInfo,
) map[string]string {
	if s.contents == nil {
		return nil
	}

	ids := make([]pulid.ID, 0, MaxAttachments)
	for idx := len(history) - 1; idx >= 0 && len(ids) < MaxAttachments; idx-- {
		for _, attachment := range history[idx].Attachments {
			if len(ids) < MaxAttachments && !slices.Contains(ids, attachment.DocumentID) {
				ids = append(ids, attachment.DocumentID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	texts := make([]string, len(ids))
	var wg sync.WaitGroup
	for idx, id := range ids {
		wg.Go(func() {
			if content, err := s.contents.GetContent(
				ctx,
				id,
				tenant,
			); err == nil &&
				content != nil {
				texts[idx] = strings.TrimSpace(content.ContentText)
			}
		})
	}
	wg.Wait()

	files := make(map[string]string, len(ids))
	for idx, id := range ids {
		if texts[idx] != "" {
			files[id.String()] = texts[idx]
		}
	}

	return files
}

func readingOf(status documentcontent.Status) agentdefinition.AttachmentReading {
	switch status {
	case documentcontent.StatusExtracted, documentcontent.StatusIndexed:
		return agentdefinition.AttachmentReadingDone
	case documentcontent.StatusFailed:
		return agentdefinition.AttachmentReadingFailed
	case documentcontent.StatusPending, documentcontent.StatusExtracting:
		return agentdefinition.AttachmentReadingPending
	default:
		return agentdefinition.AttachmentReadingPending
	}
}

// attachTurnContext records what the person handed over on their turn only:
// the assistant and tool messages that follow are about the same things, but
// the fact worth keeping is what the question came with.
func attachTurnContext(messages []conversation.Message, tc turnContext) {
	for i := range messages {
		if messages[i].Role != conversation.RoleUser {
			continue
		}
		messages[i].PageContext = tc.page
		if len(tc.attachments) > 0 {
			messages[i].Attachments = tc.attachments
		}
		if len(tc.mentions) > 0 {
			messages[i].Mentions = tc.mentions
		}

		return
	}
}
