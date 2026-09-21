package assistantservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// MaxAttachments bounds one message's files. Each is read into the
	// prompt, so the bound is a context budget as much as a courtesy.
	MaxAttachments = 5
	// AttachmentResourceType is the resource an attachment is uploaded
	// against: the thread, so ownership is a plain comparison.
	AttachmentResourceType = "assistant_thread"
	// attachmentExcerptRunes is how much of a file's text rides in the
	// prompt; the rest waits behind get_document_summary.
	attachmentExcerptRunes = 1200
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
func validateMentions(mentions []agent.EntityRef, multiErr *errortypes.MultiError) []agent.EntityRef {
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
	actor *services.RequestActor,
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
		return nil, nil, errortypes.NewBusinessError("Attachments are not available on this deployment")
	}

	stored := make([]conversation.MessageAttachment, 0, len(ids))
	runtime := make([]agentdefinition.RuntimeAttachment, 0, len(ids))
	seen := make(map[pulid.ID]struct{}, len(ids))
	for i, id := range ids {
		field := "attachmentDocumentIds[" + strconv.Itoa(i) + "]"
		if id.IsNil() {
			multiErr.Add(field, errortypes.ErrInvalid, "Attachment identifier is invalid")
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}

		doc, err := s.documents.GetByID(ctx, repositories.GetDocumentByIDRequest{
			ID:         id,
			TenantInfo: tenant,
		})
		if err != nil {
			multiErr.Add(field, errortypes.ErrInvalid, "Attachment was not found")
			continue
		}
		if doc.UploadedByID != actor.UserID ||
			doc.ResourceType != AttachmentResourceType ||
			doc.ResourceID != thread.ID.String() {
			multiErr.Add(field, errortypes.ErrInvalid, "Attachment does not belong to this conversation")
			continue
		}

		stored = append(stored, conversation.MessageAttachment{
			DocumentID:  doc.ID,
			FileName:    doc.OriginalName,
			ContentType: doc.FileType,
			FileSize:    doc.FileSize,
		})
		runtime = append(runtime, s.describeAttachment(ctx, doc.ID, doc.OriginalName, doc.FileType, tenant))
	}
	if multiErr.HasErrors() {
		return nil, nil, multiErr
	}

	return stored, runtime, nil
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
		Status:      string(documentcontent.StatusPending),
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

	attachment.Status = string(content.Status)
	attachment.PageCount = content.PageCount
	attachment.Kind = content.DetectedDocumentKind
	if text := strings.TrimSpace(content.ContentText); text != "" {
		attachment.Excerpt = text
	}

	return attachment
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
