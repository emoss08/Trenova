package inboundtools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/triageworkservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramAttachmentID = "attachmentId"
	paramShipmentID   = "shipmentId"
	paramPages        = "pages"
	paramFromPage     = "fromPage"
	paramToPage       = "toPage"
	maxFilingRanges   = 50
	maxFilingPage     = 10000
)

type triageFiling interface {
	File(
		ctx context.Context,
		req *triageworkservice.FileRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanFile(
		ctx context.Context,
		req *triageworkservice.FileRequest,
	) (*triageworkservice.FilePlan, error)
	Unfile(
		ctx context.Context,
		req *triageworkservice.UnfileRequest,
	) (*inboundmessage.InboundMessage, error)
	PlanUnfile(
		ctx context.Context,
		req *triageworkservice.UnfileRequest,
	) (*triageworkservice.UnfilePlan, error)
}

func errAttachmentElsewhere(messageID pulid.ID) error {
	return fmt.Errorf("that attachment is not one of message %s's attachments; take the "+
		"attachment id from inbox_get_inbound_message for this message", messageID)
}

type filedAttachmentView struct {
	FiledShipmentID string `json:"filedShipmentId,omitempty"`
	FiledAs         string `json:"filedAs,omitempty"`
	FiledAt         int64  `json:"filedAt,omitempty"`
}

type filedDocumentView struct {
	DocumentType string `json:"documentType"`
	Pages        string `json:"pages"`
}

type documentStatusView struct {
	Status string `json:"status"`
}

func filedAttachmentOf(attachment *inboundmessage.InboundAttachment) *filedAttachmentView {
	return &filedAttachmentView{
		FiledShipmentID: idText(attachment.FiledShipmentID),
		FiledAs:         attachment.FiledAs,
		FiledAt:         attachment.FiledAt,
	}
}

func attachmentChange(
	before, after *inboundmessage.InboundAttachment,
) (*agent.RecordChange, error) {
	return toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceInboundMessage,
			ID:       before.MessageID,
			Label:    before.FileName,
		},
		filedAttachmentOf(before),
		filedAttachmentOf(after),
		toolpreview.WithRefs(map[string]permission.Resource{
			"filedShipmentId": permission.ResourceShipment,
		}),
		toolpreview.Labels(map[string]string{
			"filedShipmentId": "Filed on",
			"filedAs":         "Filed as",
			"filedAt":         "Filed at",
		}),
		toolpreview.Volatile("filedAt"),
	)
}

func documentStatusChange(id pulid.ID, from, to document.Status) (*agent.RecordChange, error) {
	return toolpreview.Changed(
		toolpreview.Record{Resource: permission.ResourceDocument, ID: id},
		&documentStatusView{Status: string(from)},
		&documentStatusView{Status: string(to)},
	)
}

func pageSpan(from, to int) string {
	if from == to {
		return strconv.Itoa(from)
	}

	return strconv.Itoa(from) + "-" + strconv.Itoa(to)
}

type fileCall struct {
	messageID pulid.ID
	file      *triageworkservice.FileRequest
}

func NewFileInboundAttachmentTool(p triageToolParams) agenttoolports.AgentTool {
	return newFileInboundAttachmentTool(p.Inbox, p.Work)
}

func newFileInboundAttachmentTool(
	inbox inboundMessageReader,
	filing triageFiling,
) agenttoolports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_file_attachment",
			Artifact: common.ShipmentRecordEntity,
			Description: "File an email attachment on a shipment as a typed document, whole " +
				"or split into page ranges that each become their own document. Whole, it is " +
				"filed as the type you name, or as the kind it was read as when you name " +
				"none. Only a PDF splits. A file that was not read, or is already filed, is " +
				"refused; inbox_unfile_attachment undoes a filing.",
			Resource:    permission.ResourceDocument,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Condition:   inboundCondition(inbox),
			Reversible:  true,
			Rationale: "Files a document already in Trenova on a shipment; nobody outside is " +
				"told and it can be unfiled.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The message the attachment came on, from inbox_list_inbound_messages, " +
						"newest first or by query.",
				),
				paramAttachmentID: agenttoolschema.IDText(
					"The attachment, from the attachments inbox_get_inbound_message lists for the " +
						"message.",
				),
				paramShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
					"The shipment to file it on: the message's matched shipment from "+
						"inbox_get_inbound_message, or one from shipment_list."),
				paramDocumentTypeID: agenttoolschema.RecordIDText(
					permission.ResourceDocumentType,
					"Optional, for the whole file: what to file it as, from "+
						"document_list_types. Ignored when pages are given.",
				),
				paramPages: filingPagesProperty(),
			},
			Required:    []string{paramMessageID, paramAttachmentID, paramShipmentID},
			SearchTerms: []string{"file attachment", "attach POD", "split pdf", "file on load"},
			Recipe:      []string{toolGetInboundMessage, "inbox_file_attachment"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[*fileCall, *triageworkservice.FilePlan]{
			Request: fileRequest,
			Plan: func(
				ctx context.Context,
				call *fileCall,
				_ *agenttoolports.ToolExecuteParams,
			) (*triageworkservice.FilePlan, error) {
				return filing.PlanFile(ctx, call.file)
			},
			Refused: func(*fileCall) string {
				return "Would file an email attachment on a shipment."
			},
			Render: renderFiling,
			Run: func(
				ctx context.Context,
				call *fileCall,
				_ *agenttoolports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				if err := requireAttachmentOn(ctx, inbox, call.file.TenantInfo,
					call.messageID, call.file.AttachmentID); err != nil {
					return nil, err
				}
				message, err := filing.File(ctx, call.file)
				if err != nil {
					return nil, err
				}

				return &agent.ToolExecutionResult{
					Action: "filed",
					Kind:   "email attachment",
					Name:   describeInboundMessage(message),
					IDs: map[string]string{
						paramMessageID:  message.ID.String(),
						paramShipmentID: call.file.ShipmentID.String(),
					},
					Record: common.RecordOf(common.ShipmentRecordEntity, call.file.ShipmentID),
				}, nil
			},
		},
	)
}

func filingPagesProperty() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeArray,
		toolschema.KeyDescription: "Optional: page ranges of a PDF to file as " +
			"separate documents instead of the whole file. Ranges may not overlap.",
		toolschema.KeyMinItems: 1,
		toolschema.KeyMaxItems: maxFilingRanges,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramFromPage: map[string]any{
					toolschema.KeyType:        toolschema.TypeInteger,
					toolschema.KeyDescription: "The first page, counting from 1.",
					toolschema.KeyMinimum:     1,
				},
				paramToPage: map[string]any{
					toolschema.KeyType:        toolschema.TypeInteger,
					toolschema.KeyDescription: "The last page, inclusive.",
					toolschema.KeyMinimum:     1,
				},
				paramDocumentTypeID: agenttoolschema.RecordIDText(
					permission.ResourceDocumentType,
					"What these pages are, from document_list_types.",
				),
			},
			toolschema.KeyRequired: []string{
				paramFromPage, paramToPage, paramDocumentTypeID,
			},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func fileRequest(params *agenttoolports.ToolExecuteParams) (*fileCall, error) {
	messageID, err := common.RequirePulid(params.Params, paramMessageID)
	if err != nil {
		return nil, err
	}
	attachmentID, err := common.RequirePulid(params.Params, paramAttachmentID)
	if err != nil {
		return nil, err
	}
	shipmentID, err := common.RequirePulid(params.Params, paramShipmentID)
	if err != nil {
		return nil, err
	}
	documentType, err := common.OptionalID(params.Params, paramDocumentTypeID)
	if err != nil {
		return nil, err
	}
	pages, err := pageRanges(params.Params)
	if err != nil {
		return nil, err
	}

	return &fileCall{
		messageID: messageID,
		file: &triageworkservice.FileRequest{
			TenantInfo:     tenantOnly(params),
			UserID:         actorOf(params),
			AttachmentID:   attachmentID,
			ShipmentID:     shipmentID,
			DocumentTypeID: documentType,
			Pages:          pages,
		},
	}, nil
}

func pageRanges(params map[string]any) ([]triageworkservice.PageRange, error) {
	if _, given := params[paramPages]; !given {
		return nil, nil
	}

	var raw []struct {
		FromPage       int    `json:"fromPage"`
		ToPage         int    `json:"toPage"`
		DocumentTypeID string `json:"documentTypeId"`
	}
	if err := common.DecodeParam(params, paramPages, &raw); err != nil {
		return nil, err
	}
	if len(raw) > maxFilingRanges {
		return nil, fmt.Errorf("parameter %q holds %d ranges; at most %d are filed at once",
			paramPages, len(raw), maxFilingRanges)
	}

	pages := make([]triageworkservice.PageRange, 0, len(raw))
	for idx, page := range raw {
		if page.FromPage < 1 || page.ToPage < page.FromPage || page.ToPage > maxFilingPage {
			return nil, fmt.Errorf("%s[%d] must run from page 1 or later, first page before "+
				"last", paramPages, idx)
		}
		typeID, err := pulid.Parse(page.DocumentTypeID)
		if err != nil {
			return nil, fmt.Errorf("%s[%d].%s is not a valid id: %w",
				paramPages, idx, paramDocumentTypeID, err)
		}
		pages = append(pages, triageworkservice.PageRange{
			FromPage:       page.FromPage,
			ToPage:         page.ToPage,
			DocumentTypeID: typeID,
		})
	}

	return pages, nil
}

func renderFiling(call *fileCall, plan *triageworkservice.FilePlan) (*agent.ToolPreview, error) {
	summary := "Would file " + plan.Attachment.FileName + " on the shipment"
	if plan.Message.ID != call.messageID {
		return refusedPreview(summary+".", errAttachmentElsewhere(call.messageID)), nil
	}

	filed, err := attachmentChange(plan.Attachment, plan.After)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{filed}

	if len(plan.Ranges) == 0 {
		summary += " as " + plan.DocumentType.Name
		if plan.Replaces.IsNotNil() {
			replaced, replaceErr := documentStatusChange(plan.Replaces,
				document.StatusActive, document.StatusArchived)
			if replaceErr != nil {
				return nil, replaceErr
			}
			changes = append(changes, replaced)
			summary += ", archiving the one already on file"
		}

		return toolpreview.Build(summary+".", changes...), nil
	}

	for _, rng := range plan.Ranges {
		created, createErr := toolpreview.Create(
			toolpreview.Record{Resource: permission.ResourceDocument, Label: rng.FileName},
			&filedDocumentView{
				DocumentType: rng.DocumentType,
				Pages:        pageSpan(rng.FromPage, rng.ToPage),
			},
		)
		if createErr != nil {
			return nil, createErr
		}
		changes = append(changes, created)
	}

	preview := toolpreview.Build(
		fmt.Sprintf("%s as %d separate documents.", summary, len(plan.Ranges)),
		changes...,
	)
	if plan.PagesUnchecked {
		preview.Summary += " The file's page count is not known yet, so a range past its " +
			"last page is refused when this runs."
		preview.Partial = true
	}

	return preview, nil
}

func requireAttachmentOn(
	ctx context.Context,
	inbox inboundMessageReader,
	tenant pagination.TenantInfo,
	messageID, attachmentID pulid.ID,
) error {
	message, err := inbox.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenant,
		IncludeAttachments: true,
	})
	if err != nil {
		return err
	}
	for _, attachment := range message.Attachments {
		if attachment.ID == attachmentID {
			return nil
		}
	}

	return errAttachmentElsewhere(messageID)
}

type unfileCall struct {
	messageID pulid.ID
	unfile    *triageworkservice.UnfileRequest
}

func NewUnfileInboundAttachmentTool(p triageToolParams) agenttoolports.AgentTool {
	return newUnfileInboundAttachmentTool(p.Inbox, p.Work)
}

func newUnfileInboundAttachmentTool(
	inbox inboundMessageReader,
	filing triageFiling,
) agenttoolports.AgentTool {
	return common.NewReportingReceivableTool(
		&common.ReceivableSpec{
			Name:     "inbox_unfile_attachment",
			Artifact: inboundMessageEntity,
			Description: "Take an email attachment off the shipment it was filed on, when " +
				"it went on the wrong load or as the wrong type. A whole file returns " +
				"to the message and any document it replaced is restored; the documents a " +
				"split made are deleted. inbox_get_inbound_message shows where each attachment is " +
				"filed.",
			Resource:    permission.ResourceDocument,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Rationale: "Takes a document off a shipment inside Trenova and deletes the " +
				"documents a split made, so a person approves it.",
			Properties: map[string]any{
				paramMessageID: messageIDProperty(
					"The message the attachment came on, from inbox_list_inbound_messages, " +
						"newest first or by query.",
				),
				paramAttachmentID: agenttoolschema.IDText(
					"The filed attachment, from the attachments inbox_get_inbound_message lists " +
						"for the message.",
				),
			},
			Required:    []string{paramMessageID, paramAttachmentID},
			SearchTerms: []string{"unfile attachment", "wrong load", "take document off"},
			Recipe:      []string{toolGetInboundMessage, "inbox_unfile_attachment"},
			Target:      targetMessage,
		},
		common.ReceivablePlan[*unfileCall, *triageworkservice.UnfilePlan]{
			Request: func(params *agenttoolports.ToolExecuteParams) (*unfileCall, error) {
				messageID, err := common.RequirePulid(params.Params, paramMessageID)
				if err != nil {
					return nil, err
				}
				attachmentID, err := common.RequirePulid(params.Params, paramAttachmentID)
				if err != nil {
					return nil, err
				}

				return &unfileCall{
					messageID: messageID,
					unfile: &triageworkservice.UnfileRequest{
						TenantInfo:   tenantOnly(params),
						UserID:       actorOf(params),
						AttachmentID: attachmentID,
					},
				}, nil
			},
			Plan: func(
				ctx context.Context,
				call *unfileCall,
				_ *agenttoolports.ToolExecuteParams,
			) (*triageworkservice.UnfilePlan, error) {
				return filing.PlanUnfile(ctx, call.unfile)
			},
			Refused: func(*unfileCall) string {
				return "Would take an email attachment back off its shipment."
			},
			Render: renderUnfiling,
			Run: func(
				ctx context.Context,
				call *unfileCall,
				_ *agenttoolports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				if err := requireAttachmentOn(ctx, inbox, call.unfile.TenantInfo,
					call.messageID, call.unfile.AttachmentID); err != nil {
					return nil, err
				}
				message, err := filing.Unfile(ctx, call.unfile)
				if err != nil {
					return nil, err
				}

				return messageResult("unfiled an attachment of", message), nil
			},
		},
	)
}

func renderUnfiling(
	call *unfileCall,
	plan *triageworkservice.UnfilePlan,
) (*agent.ToolPreview, error) {
	summary := "Would take " + plan.Attachment.FileName + " back off the shipment it was " +
		"filed on"
	if plan.Message.ID != call.messageID {
		return refusedPreview(summary+".", errAttachmentElsewhere(call.messageID)), nil
	}

	unfiled, err := attachmentChange(plan.Attachment, plan.After)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{unfiled}

	for _, split := range plan.Removed {
		deleted, deleteErr := toolpreview.Delete(
			toolpreview.Record{
				Resource: permission.ResourceDocument,
				ID:       split.DocumentID,
				Label:    split.DocumentType,
			},
			&filedDocumentView{
				DocumentType: split.DocumentType,
				Pages:        pageSpan(split.FromPage, split.ToPage),
			},
		)
		if deleteErr != nil {
			return nil, deleteErr
		}
		changes = append(changes, deleted)
	}
	if len(plan.Removed) > 0 {
		summary += fmt.Sprintf(", deleting the %d documents its split made", len(plan.Removed))
	}

	if plan.Restores.IsNotNil() {
		restored, restoreErr := documentStatusChange(plan.Restores,
			document.StatusArchived, document.StatusActive)
		if restoreErr != nil {
			return nil, restoreErr
		}
		changes = append(changes, restored)
		summary += ", and restore the document it had replaced"
	}

	return toolpreview.Build(summary+".", changes...), nil
}
