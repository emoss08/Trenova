package inboundreplyservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagenova"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundreplyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type SaveDraftRequest struct {
	TenantInfo         pagination.TenantInfo
	UserID             pulid.ID
	MessageID          pulid.ID
	Subject            string
	Body               string
	DocumentIDs        []pulid.ID
	ReplaceAttachments bool
}

type DraftReplyRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	MessageID  pulid.ID
	Mode       triagenova.Mode
}

type DraftPlan struct {
	Message *inboundmessage.InboundMessage
	Current *triage.Reply
	Draft   *triage.Reply
}

func (s *Service) SaveDraft(ctx context.Context, req *SaveDraftRequest) (*triage.Reply, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID, false)
	if err != nil {
		return nil, err
	}

	draft, err := s.savedDraft(ctx, message, req)
	if err != nil {
		return nil, err
	}

	saved, err := s.replies.UpsertDraft(ctx, draft)
	if err != nil {
		return nil, err
	}
	s.changed(ctx, message)

	return saved, nil
}

func (s *Service) PlanSaveDraft(ctx context.Context, req *SaveDraftRequest) (*DraftPlan, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID, false)
	if err != nil {
		return nil, err
	}

	draft, err := s.savedDraft(ctx, message, req)
	if err != nil {
		return nil, err
	}

	current, _, err := s.currentDraft(ctx, message)
	if err != nil {
		return nil, err
	}

	return &DraftPlan{Message: message, Current: current, Draft: draft}, nil
}

func (s *Service) savedDraft(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	req *SaveDraftRequest,
) (*triage.Reply, error) {
	draft := s.draftFor(message, req.UserID)
	if subject := strings.TrimSpace(req.Subject); subject != "" {
		draft.Subject = subject
	}
	draft.Body = strings.TrimSpace(req.Body)
	if err := validateDraft(draft); err != nil {
		return nil, err
	}

	attachments, err := s.draftAttachments(ctx, req, message)
	if err != nil {
		return nil, err
	}
	draft.Attachments = attachments

	return draft, nil
}

func (s *Service) PlanDiscardDraft(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*DraftPlan, error) {
	message, err := s.message(ctx, tenantInfo, messageID, true)
	if err != nil {
		return nil, err
	}

	current, _, err := s.currentDraft(ctx, message)
	if err != nil {
		return nil, err
	}

	return &DraftPlan{Message: message, Current: current}, nil
}

func (s *Service) DiscardDraft(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundmessage.InboundMessage, error) {
	message, err := s.message(ctx, tenantInfo, messageID, true)
	if err != nil {
		return nil, err
	}

	removed, err := s.replies.DeleteDraft(ctx, inboundreplyports.InboundReplyMessageRequest{
		TenantInfo: tenantInfo,
		MessageID:  messageID,
	})
	if err != nil {
		return nil, err
	}
	if removed > 0 {
		s.changed(ctx, message)
	}

	return message, nil
}

func (s *Service) DraftReply(ctx context.Context, req *DraftReplyRequest) (*triage.Reply, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID, true)
	if err != nil {
		return nil, err
	}

	return s.draftReplyFor(ctx, message, req)
}

func (s *Service) DraftReplyFor(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	mode triagenova.Mode,
) (*triage.Reply, error) {
	if message.Attachments == nil {
		attachments, err := s.messages.ListAttachments(ctx, message.ID, tenantOf(message))
		if err != nil {
			return nil, err
		}
		message.Attachments = attachments
	}

	return s.draftReplyFor(ctx, message, &DraftReplyRequest{
		TenantInfo: tenantOf(message),
		MessageID:  message.ID,
		Mode:       mode,
	})
}

func (s *Service) PlanDraftReply(
	ctx context.Context,
	req *DraftReplyRequest,
) (*DraftPlan, error) {
	message, err := s.message(ctx, req.TenantInfo, req.MessageID, true)
	if err != nil {
		return nil, err
	}
	if err = s.canDraft(message, req.Mode); err != nil {
		return nil, err
	}

	current, _, err := s.currentDraft(ctx, message)
	if err != nil {
		return nil, err
	}

	draft := s.draftFor(message, req.UserID)
	draft.DraftedByNova = true

	return &DraftPlan{Message: message, Current: current, Draft: draft}, nil
}

func (s *Service) canDraft(message *inboundmessage.InboundMessage, mode triagenova.Mode) error {
	if !s.nova.Available(mode) {
		return triagenova.ErrUnavailable
	}
	if message.Mailbox == nil {
		return errortypes.NewBusinessError("The message's mailbox no longer exists")
	}

	return nil
}

func (s *Service) draftReplyFor(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	req *DraftReplyRequest,
) (*triage.Reply, error) {
	if err := s.canDraft(message, req.Mode); err != nil {
		return nil, err
	}

	mc := s.messageContext(ctx, message)
	if message.MatchedShipmentID.IsNotNil() && s.stops != nil &&
		message.Classification == inboundmessage.ClassificationDetentionDispute {
		stops, err := s.stops.StopTimes(ctx, tenantOf(message), message.MatchedShipmentID)
		if err != nil {
			s.l.Warn("could not read stop times for a drafted reply",
				zap.String("messageId", message.ID.String()), zap.Error(err))
		}
		mc.stops = stops
	}

	mc.attachments, mc.evidence = s.novaAttachments(ctx, message)

	body, err := s.writeReply(ctx, req, mc)
	if err != nil {
		return nil, err
	}

	draft := s.draftFor(message, req.UserID)
	draft.Body = body
	draft.DraftedByNova = true
	draft.Attachments = mc.attachments
	if err = validateDraft(draft); err != nil {
		return nil, err
	}

	saved, err := s.replies.UpsertDraft(ctx, draft)
	if err != nil {
		return nil, err
	}

	activity := triage.NewActivity(message, triage.ActivityReplyDrafted).
		ByNova().
		About(saved.Subject, saved.ToAddress)
	activity.RecordID = saved.ID
	s.record(ctx, activity)
	s.changed(ctx, message)

	return saved, nil
}

func (s *Service) draftFor(message *inboundmessage.InboundMessage, userID pulid.ID) *triage.Reply {
	from := ""
	if message.Mailbox != nil {
		from = message.Mailbox.Address
	}

	return &triage.Reply{
		OrganizationID: message.OrganizationID,
		BusinessUnitID: message.BusinessUnitID,
		MessageID:      message.ID,
		MailboxID:      message.MailboxID,
		ThreadKey:      message.ThreadKey,
		Status:         triage.ReplyDraft,
		FromAddress:    from,
		ToAddress:      strings.ToLower(strings.TrimSpace(message.FromAddress)),
		Subject:        replySubject(message.Subject),
		CreatedByID:    userID,
	}
}

func validateDraft(draft *triage.Reply) error {
	multiErr := errortypes.NewMultiError()
	draft.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
