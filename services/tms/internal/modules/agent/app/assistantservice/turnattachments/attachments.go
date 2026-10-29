package turnattachments

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
)

const (
	// MaxAttachments bounds one message's files. Each is read into the
	// prompt, so the bound is a context budget as much as a courtesy.
	MaxAttachments = conversationqueue.MaxMessageAttachments
	// AttachmentResourceType is the resource an attachment is uploaded
	// against: the thread, so ownership is a plain comparison.
	AttachmentResourceType = document.ConversationResourceType
)

// attachmentSlot is one attached file as it was read: what is kept on the
// message and shown to the model, or why it was refused. A duplicate leaves
// its slot empty.
type attachmentSlot struct {
	stored  *conversation.MessageAttachment
	runtime agentturn.RuntimeAttachment
	problem string
}

func readingOf(status documentcontent.Status) agentturn.AttachmentReading {
	switch status {
	case documentcontent.StatusExtracted, documentcontent.StatusIndexed:
		return agentturn.AttachmentReadingDone
	case documentcontent.StatusFailed:
		return agentturn.AttachmentReadingFailed
	case documentcontent.StatusPending, documentcontent.StatusExtracting:
		return agentturn.AttachmentReadingPending
	default:
		return agentturn.AttachmentReadingPending
	}
}
