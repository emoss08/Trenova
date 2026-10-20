package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	maxListedAttachments   = 25
	attachmentReadingWait  = 20 * time.Second
	attachmentReadingPoll  = time.Second
	attachmentListNote     = "Read one with agent_read_attachment and its documentId."
	attachmentNoneNote     = "No file has been attached to this conversation."
	attachmentStillReading = "The file is still being read. This conversation now waits for it: " +
		"when the reading finishes it picks up again on its own, and you carry on then with the " +
		"file read. Tell the person in one short sentence that you will carry on as soon as the " +
		"file is read, and end your turn. Do not ask them to send their message again, and do " +
		"not guess what the file says."
	attachmentOutsideThread = "document %s was not attached to this conversation by the person you are working for"
)

var errNoConversation = errors.New(
	"agent_read_attachment reads the files a person attached to a conversation, and this run has none",
)

type readAttachmentTool struct {
	documents documentports.DocumentRepository
	contents  contentReader
	wait      time.Duration
	poll      time.Duration
}

func newReadAttachmentTool(
	documents documentports.DocumentRepository,
	contents contentReader,
) agentports.AgentQueryTool {
	return &readAttachmentTool{
		documents: documents,
		contents:  contents,
		wait:      attachmentReadingWait,
		poll:      attachmentReadingPoll,
	}
}

func (t *readAttachmentTool) Name() string { return agentdefinition.CoreToolReadAttachment }

func (t *readAttachmentTool) SearchTerms() []string {
	return []string{
		"read attachment", "attached file", "uploaded file", "read the pdf",
		"read the rate confirmation", "the file I sent", "document I attached",
	}
}

func (t *readAttachmentTool) Description() string {
	return "Read a file the person attached to this conversation: its text in reading " +
		"order and what was read from it. It says how many pages it has, what kind of " +
		"document it looks like and the fields found. Leave documentId out to list every file attached to this conversation, " +
		"newest first. The text of a file of more than one page marks where each page begins " +
		"(--- page 2 ---). Read one page, or the part past where a long text stops with " +
		"offset. A file still being read is waited for."
}

func (t *readAttachmentTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: withTextWindow(map[string]any{
			paramDocumentID: agenttoolschema.RecordIDText(permission.ResourceDocument,
				"The file's id, from the attachments on a message or from agent_read_attachment "+
					"called without it, which lists the files attached to this conversation."),
		}),
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *readAttachmentTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceAssistant,
		scope:    agent.ToolScopeSelf,
		reads:    agent.ExternalReadAlways,
		source:   agent.TaintSourceAttachment,
		rationale: "Reads text from a file the person attached to their own conversation; " +
			"nothing changes and nothing is sent.",
	})
}

type conversationFiles struct {
	Files []attachedFile `json:"files"`
	Total int            `json:"total"`
	Note  string         `json:"note"`
}

type attachedFile struct {
	DocumentID  string `json:"documentId"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	AttachedAt  int64  `json:"attachedAt"`
}

func (t *readAttachmentTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	person := personOf(params.Actor)
	if params.ThreadID.IsNil() || person.IsNil() {
		return nil, errNoConversation
	}

	documentID, given, err := optionalPulid(params.Params, paramDocumentID)
	if err != nil {
		return nil, err
	}
	window, err := textWindowOf(params.Params)
	if err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	if !given {
		return t.list(ctx, params.ThreadID, person, tenant)
	}

	doc, err := t.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenant,
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf(attachmentOutsideThread, documentID)
		}

		return nil, err
	}
	if !doc.AttachedBy(params.ThreadID, person) {
		return nil, fmt.Errorf(attachmentOutsideThread, documentID)
	}

	content, err := t.settled(ctx, doc.ID, tenant)
	if err != nil {
		return nil, fmt.Errorf("read what was extracted from document %s: %w", doc.ID, err)
	}

	summary, err := summarizeDocument(doc, content, window)
	if err != nil {
		return nil, err
	}
	if content == nil || content.StillBeingRead(timeutils.NowUnix()) {
		summary.StillBeingRead = true
		summary.Note = attachmentStillReading
	}

	return summary, nil
}

func (t *readAttachmentTool) list(
	ctx context.Context,
	threadID, person pulid.ID,
	tenant pagination.TenantInfo,
) (*conversationFiles, error) {
	docs, err := t.documents.GetByResourceID(ctx, &documentports.GetDocumentsByResourceRequest{
		TenantInfo:   tenant,
		ResourceID:   threadID.String(),
		ResourceType: document.ConversationResourceType,
	})
	if err != nil {
		return nil, err
	}

	files := make([]attachedFile, 0, min(len(docs), maxListedAttachments))
	total := 0
	for _, doc := range docs {
		if doc == nil || !doc.AttachedBy(threadID, person) {
			continue
		}
		total++
		if len(files) == maxListedAttachments {
			continue
		}
		files = append(files, attachedFile{
			DocumentID:  doc.ID.String(),
			FileName:    doc.OriginalName,
			ContentType: doc.FileType,
			FileSize:    doc.FileSize,
			AttachedAt:  doc.CreatedAt,
		})
	}

	result := &conversationFiles{Files: files, Total: total, Note: attachmentListNote}
	if total == 0 {
		result.Note = attachmentNoneNote
	}

	return result, nil
}

func (t *readAttachmentTool) settled(
	ctx context.Context,
	documentID pulid.ID,
	tenant pagination.TenantInfo,
) (*documentcontent.Content, error) {
	deadline := time.Now().Add(t.wait)
	for {
		content, err := t.contents.GetContent(ctx, documentID, tenant)
		switch {
		case err != nil && !isNotFound(err):
			return nil, err
		case err != nil:
			content = nil
		case content != nil && !content.StillBeingRead(timeutils.NowUnix()):
			return content, nil
		}
		if !time.Now().Add(t.poll).Before(deadline) {
			return content, nil
		}

		timer := time.NewTimer(t.poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
