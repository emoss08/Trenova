package inboundreplyservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	attachmentsField     = "documentIds"
	attachmentNameJoiner = ", "
)

func (s *Service) resolveAttachments(
	ctx context.Context,
	req *documentports.ResolveMailAttachmentsRequest,
) ([]*documentports.MailAttachment, error) {
	if len(req.DocumentIDs) == 0 {
		return []*documentports.MailAttachment{}, nil
	}
	if s.documents == nil {
		return nil, errortypes.NewBusinessError(
			"Files cannot be attached to replies on this installation",
		)
	}
	if req.Field == "" {
		req.Field = attachmentsField
	}

	return s.documents.Resolve(ctx, req)
}

func personTenant(tenantInfo pagination.TenantInfo, userID pulid.ID) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: tenantInfo.OrgID, BuID: tenantInfo.BuID, UserID: userID}
}

func snapshotOf(resolved []*documentports.MailAttachment) []triage.ReplyAttachment {
	if len(resolved) == 0 {
		return nil
	}

	out := make([]triage.ReplyAttachment, 0, len(resolved))
	for _, attachment := range resolved {
		out = append(out, triage.ReplyAttachment{
			DocumentID:  attachment.Document.ID,
			FileName:    attachment.Attachment.FileName,
			ContentType: attachment.Attachment.ContentType,
			ByteSize:    attachment.Attachment.SizeBytes,
		})
	}

	return out
}

func emailAttachmentsOf(resolved []*documentports.MailAttachment) []emailports.EmailAttachment {
	if len(resolved) == 0 {
		return nil
	}

	out := make([]emailports.EmailAttachment, 0, len(resolved))
	for _, attachment := range resolved {
		out = append(out, attachment.Attachment)
	}

	return out
}

func attachmentDetail(reply *triage.Reply) string {
	return strings.Join(reply.AttachmentNames(), attachmentNameJoiner)
}

func (s *Service) currentDraftAttachments(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) ([]triage.ReplyAttachment, error) {
	draft, found, err := s.currentDraft(ctx, message)
	if err != nil || !found {
		return nil, err
	}

	return draft.Attachments, nil
}

func (s *Service) currentDraft(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (*triage.Reply, bool, error) {
	drafts, err := s.replies.DraftsByMessageIDs(
		ctx,
		inboundmessageports.InboundRepliesByMessagesRequest{
			TenantInfo: tenantOf(message),
			MessageIDs: []pulid.ID{message.ID},
		},
	)
	if err != nil {
		return nil, false, err
	}

	draft, ok := drafts[message.ID]

	return draft, ok && draft != nil, nil
}

func (s *Service) draftAttachments(
	ctx context.Context,
	req *SaveDraftRequest,
	message *inboundmessage.InboundMessage,
) ([]triage.ReplyAttachment, error) {
	if !req.ReplaceAttachments {
		return s.currentDraftAttachments(ctx, message)
	}

	resolved, err := s.resolveAttachments(ctx, &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  personTenant(req.TenantInfo, req.UserID),
		DocumentIDs: req.DocumentIDs,
	})
	if err != nil {
		return nil, err
	}

	return snapshotOf(resolved), nil
}

func (s *Service) novaAttachments(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (attachments []triage.ReplyAttachment, evidence bool) {
	kept, err := s.currentDraftAttachments(ctx, message)
	if err != nil {
		s.l.Warn("could not read the draft's attachments before Nova drafted",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}

	found := s.detentionEvidence(ctx, message)
	if found == nil {
		return kept, false
	}
	for i := range kept {
		if kept[i].DocumentID == found.DocumentID {
			return kept, true
		}
	}

	return append(kept, *found), true
}

func standingBy(message *inboundmessage.InboundMessage) bool {
	return message.Classification == inboundmessage.ClassificationDetentionDispute &&
		message.ActionState.StopsDecision == inboundmessage.StopsDecisionStandBy &&
		message.MatchedShipmentID.IsNotNil()
}

func (s *Service) detentionEvidence(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) *triage.ReplyAttachment {
	if s.evidence == nil || s.documents == nil || !standingBy(message) {
		return nil
	}

	log := s.l.With(zap.String("messageId", message.ID.String()))
	evidence, err := s.evidence.DetentionEvidence(
		ctx,
		&inboundmessageports.TriageDetentionEvidenceRequest{
			TenantInfo: tenantOf(message),
			ShipmentID: message.MatchedShipmentID,
		},
	)
	if err != nil {
		log.Warn("could not find the detention record to attach to a reply", zap.Error(err))

		return nil
	}
	if evidence == nil || evidence.DocumentID.IsNil() {
		return nil
	}

	resolved, err := s.documents.Resolve(ctx, &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  tenantOf(message),
		DocumentIDs: []pulid.ID{evidence.DocumentID},
		Field:       attachmentsField,
		System:      true,
	})
	if err != nil || len(resolved) == 0 {
		log.Warn("the detention record cannot be attached to a reply",
			zap.String("documentId", evidence.DocumentID.String()), zap.Error(err))

		return nil
	}

	snapshot := snapshotOf(resolved)

	return &snapshot[0]
}
