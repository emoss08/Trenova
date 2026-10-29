package inboundreplyservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	maxReplyReferences       = 20
	alternateLookbackSeconds = 90 * 24 * 60 * 60
	replySubjectPrefix       = "Re: "

	failureNoOutbound  = "Outbound email is not set up on this installation, so the reply was not sent."
	failureNoProfile   = "No email profile is set up for this organization, so the reply was not sent. An administrator can add one in the email settings."
	failureUnqueued    = "Trenova could not hand the reply to the email provider. Try sending it again."
	failureBounced     = "The receiving server refused it."
	failureComplained  = "The recipient marked it as spam."
	failureSuppressed  = "The address is on the organization's suppression list."
	failureUnspecified = "The email provider did not accept it."
)

type SendRequest struct {
	TenantInfo     pagination.TenantInfo
	UserID         pulid.ID
	MessageID      pulid.ID
	Subject        string
	Body           string
	ToAddress      string
	TemplateID     pulid.ID
	DraftedByNova  bool
	DocumentIDs    []pulid.ID
	IdempotencyKey string
}

func (s *Service) Send(ctx context.Context, req *SendRequest) (*triage.Reply, error) {
	if req.IdempotencyKey != "" {
		existing, err := s.replies.GetByIdempotencyKey(
			ctx,
			inboundmessageports.GetInboundReplyByKeyRequest{
				TenantInfo:     req.TenantInfo,
				IdempotencyKey: req.IdempotencyKey,
			},
		)
		switch {
		case err == nil:
			return existing, nil
		case !errortypes.IsNotFoundError(err):
			return nil, err
		}
	}

	message, err := s.message(ctx, req.TenantInfo, req.MessageID, false)
	if err != nil {
		return nil, err
	}
	if message.Mailbox == nil {
		return nil, errortypes.NewBusinessError("The message's mailbox no longer exists")
	}

	reply, err := s.newOutgoingReply(message, req)
	if err != nil {
		return nil, err
	}

	attachments, err := s.resolveAttachments(ctx, &documentports.ResolveMailAttachmentsRequest{
		TenantInfo:  personTenant(req.TenantInfo, req.UserID),
		DocumentIDs: req.DocumentIDs,
		Recipients:  []string{reply.ToAddress},
	})
	if err != nil {
		return nil, err
	}
	reply.Attachments = snapshotOf(attachments)

	created, err := s.replies.Create(ctx, reply)
	if err != nil {
		if req.IdempotencyKey != "" && dberror.IsUniqueConstraintViolation(err) {
			return s.replies.GetByIdempotencyKey(
				ctx,
				inboundmessageports.GetInboundReplyByKeyRequest{
					TenantInfo:     req.TenantInfo,
					IdempotencyKey: req.IdempotencyKey,
				},
			)
		}

		return nil, err
	}

	if _, err = s.replies.DeleteDraft(ctx, inboundmessageports.InboundReplyMessageRequest{
		TenantInfo: req.TenantInfo,
		MessageID:  message.ID,
	}); err != nil {
		s.l.Warn("sent a reply but could not clear the message's draft",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}

	return s.dispatch(ctx, message, created, emailAttachmentsOf(attachments))
}

func (s *Service) newOutgoingReply(
	message *inboundmessage.InboundMessage,
	req *SendRequest,
) (*triage.Reply, error) {
	to := message.FromAddress
	if strings.TrimSpace(req.ToAddress) != "" {
		normalized, err := emailutils.Normalize(req.ToAddress)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"toAddress",
				errortypes.ErrInvalid,
				"The reply must go to an email address",
			)
		}
		to = normalized
	}

	key, err := newMessageKey(message.Mailbox.Address)
	if err != nil {
		return nil, err
	}

	reply := &triage.Reply{
		OrganizationID: message.OrganizationID,
		BusinessUnitID: message.BusinessUnitID,
		MessageID:      message.ID,
		MailboxID:      message.MailboxID,
		ThreadKey:      message.ThreadKey,
		Status:         triage.ReplySending,
		FromAddress:    message.Mailbox.Address,
		ToAddress:      strings.ToLower(strings.TrimSpace(to)),
		Subject:        strings.TrimSpace(req.Subject),
		Body:           strings.TrimSpace(req.Body),
		DraftedByNova:  req.DraftedByNova,
		TemplateID:     req.TemplateID,
		MessageKey:     key,
		IdempotencyKey: req.IdempotencyKey,
		CreatedByID:    req.UserID,
		SentByID:       req.UserID,
	}

	multiErr := errortypes.NewMultiError()
	reply.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return reply, nil
}

func newMessageKey(mailboxAddress string) (string, error) {
	domain := emailutils.Domain(mailboxAddress)
	if domain == "" {
		return "", errortypes.NewBusinessError("The mailbox address has no domain to send from")
	}

	token, ok := mailtext.MessageIDToken(pulid.MustNew("irplm_").String() + "@" + domain)
	if !ok {
		return "", errortypes.NewBusinessError("The mailbox address cannot carry a message id")
	}

	return inboundmessage.NormalizeMessageID(token), nil
}

func (s *Service) dispatch(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	reply *triage.Reply,
	attachments []emailports.EmailAttachment,
) (*triage.Reply, error) {
	if s.email == nil {
		return s.fail(ctx, &failure{message: message, reply: reply, text: failureNoOutbound})
	}

	sent, err := s.email.Send(ctx, &emailports.SendEmailRequest{
		TenantInfo:     tenantOf(message),
		Purpose:        email.PurposeOperations,
		FromEmail:      reply.FromAddress,
		To:             []string{reply.ToAddress},
		Subject:        reply.Subject,
		Text:           reply.Body,
		HTML:           stringutils.TextToHTMLParagraph(reply.Body),
		Attachments:    attachments,
		Headers:        threadHeaders(message, reply),
		IdempotencyKey: inboundmessageports.InboundReplyIdempotencyPrefix + reply.ID.String(),
	})
	if err != nil {
		return s.fail(ctx, &failure{
			message: message,
			reply:   reply,
			text:    sendFailureText(err),
			suggest: recipientRefused(err, reply.ToAddress),
			cause:   err,
		})
	}

	return s.applyEmailStatus(ctx, message, reply, emailports.EmailMessageChange{Message: sent})
}

func threadHeaders(message *inboundmessage.InboundMessage, reply *triage.Reply) map[string]string {
	headers := make(map[string]string, 3)
	if token, ok := mailtext.MessageIDToken(reply.MessageKey); ok {
		headers["Message-ID"] = token
	}
	if token, ok := mailtext.MessageIDToken(message.MessageID); ok {
		headers["In-Reply-To"] = token
		headers["References"] = mailtext.ReplyReferences(
			message.References,
			token,
			maxReplyReferences,
		)
	}

	return headers
}

func sendFailureText(err error) string {
	switch {
	case errors.Is(err, emailports.ErrNoEmailProfile):
		return failureNoProfile
	case errors.Is(err, kernelports.ErrWorkflowStarterDisabled):
		return failureUnqueued
	case errortypes.IsMultiError(err), errortypes.IsBusinessError(err),
		errortypes.IsPlanRestrictionError(err), errortypes.IsError(err):
		return emailutils.CleanFailureText(err.Error())
	default:
		return failureUnqueued
	}
}

func recipientRefused(err error, recipient string) bool {
	return strings.Contains(strings.ToLower(err.Error()), strings.ToLower(recipient))
}

type failure struct {
	message *inboundmessage.InboundMessage
	reply   *triage.Reply
	text    string
	suggest bool
	cause   error
}

func (s *tracker) fail(ctx context.Context, f *failure) (*triage.Reply, error) {
	if f.cause != nil {
		s.l.Warn("an inbound reply was not sent",
			zap.String("replyId", f.reply.ID.String()),
			zap.String("messageId", f.message.ID.String()),
			zap.Error(f.cause),
		)
	}

	f.reply.Status = triage.ReplyFailed
	f.reply.FailureText = f.text
	f.reply.SentAt = 0
	f.reply.SuggestedAddress = ""
	if f.suggest {
		f.reply.SuggestedAddress = s.suggestAddress(ctx, f.message, f.reply.ToAddress)
	}

	updated, err := s.replies.Update(ctx, f.reply)
	if err != nil {
		return nil, err
	}

	s.record(ctx, triage.NewActivity(f.message, triage.ActivityReplyFailed).
		ByPerson(updated.SentByID).
		About(updated.ToAddress, updated.FailureText))
	s.changed(ctx, f.message)

	return updated, nil
}

func (s *tracker) suggestAddress(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	refused string,
) string {
	addresses, err := s.work.AlternateAddresses(ctx, &inboundmessageports.AlternateAddressRequest{
		TenantInfo: tenantOf(message),
		Domain:     emailutils.Domain(refused),
		Exclude:    refused,
		Since:      s.now() - alternateLookbackSeconds,
		Limit:      1,
	})
	if err != nil {
		s.l.Warn("could not look for another address at the sender",
			zap.String("messageId", message.ID.String()), zap.Error(err))

		return ""
	}
	if len(addresses) == 0 {
		return ""
	}

	return addresses[0]
}

func (s *tracker) markSent(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	reply *triage.Reply,
	sent *email.Message,
) (*triage.Reply, error) {
	reply.Status = triage.ReplySent
	reply.FailureText = ""
	reply.SuggestedAddress = ""
	reply.ProviderMessageID = sent.ProviderMessageID
	reply.SentAt = sent.SentAt
	if reply.SentAt == 0 {
		reply.SentAt = s.now()
	}

	updated, err := s.replies.Update(ctx, reply)
	if err != nil {
		return nil, err
	}

	activity := triage.NewActivity(message, triage.ActivityReplied).
		About(updated.FromAddress, attachmentDetail(updated))
	activity.RecordID = updated.ID
	if updated.SentByID.IsNotNil() {
		activity.ByPerson(updated.SentByID)
	} else {
		activity.ByNova()
	}
	s.record(ctx, activity)
	s.changed(ctx, message)

	return updated, nil
}

func (s *tracker) applyEmailStatus(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	reply *triage.Reply,
	change emailports.EmailMessageChange,
) (*triage.Reply, error) {
	sent := change.Message
	switch sent.Status {
	case email.MessageStatusSent, email.MessageStatusDelivered,
		email.MessageStatusOpened, email.MessageStatusClicked:
		if reply.Status == triage.ReplySent {
			return reply, nil
		}

		return s.markSent(ctx, message, reply, sent)
	case email.MessageStatusFailed, email.MessageStatusBounced,
		email.MessageStatusComplained, email.MessageStatusSuppressed:
		if reply.Status == triage.ReplyFailed {
			return reply, nil
		}
		text := providerFailureText(change)

		return s.fail(ctx, &failure{
			message: message,
			reply:   reply,
			text:    text,
			suggest: !strings.Contains(strings.ToLower(sent.LastError), "configuration"),
		})
	case email.MessageStatusQueued, email.MessageStatusSending:
		return reply, nil
	default:
		return reply, nil
	}
}

func providerFailureText(change emailports.EmailMessageChange) string {
	sent := change.Message
	if text := emailutils.ProviderFailureText(sent.LastError); text != "" {
		return text
	}
	if change.Event != nil {
		if text := eventDetail(change.Event.Raw); text != "" {
			return emailutils.CleanFailureText(text)
		}
	}

	switch sent.Status {
	case email.MessageStatusBounced:
		return failureBounced
	case email.MessageStatusComplained:
		return failureComplained
	case email.MessageStatusSuppressed:
		return failureSuppressed
	case email.MessageStatusQueued, email.MessageStatusSending, email.MessageStatusSent,
		email.MessageStatusDelivered, email.MessageStatusFailed, email.MessageStatusOpened,
		email.MessageStatusClicked:
		return failureUnspecified
	default:
		return failureUnspecified
	}
}

var eventDetailKeys = []string{"Description", "Details", "description", "reason", "message"}

func eventDetail(raw map[string]any) string {
	if len(raw) == 0 {
		return ""
	}
	for _, key := range eventDetailKeys {
		if value, ok := raw[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	for _, nested := range []string{"bounce", "data"} {
		if inner, ok := raw[nested].(map[string]any); ok {
			if value := eventDetail(inner); value != "" {
				return value
			}
		}
	}

	return ""
}

func (s *tracker) EmailMessageChanged(ctx context.Context, change emailports.EmailMessageChange) {
	sent := change.Message
	if sent == nil {
		return
	}
	replyID, ok := replyIDFromKey(sent.IdempotencyKey)
	if !ok {
		return
	}

	tenantInfo := pagination.TenantInfo{OrgID: sent.OrganizationID, BuID: sent.BusinessUnitID}
	ctx = dbscope.EnsureTenant(ctx, tenantInfo.DBTenant())
	reply, err := s.replies.GetByID(ctx, inboundmessageports.GetInboundReplyRequest{
		ID:         replyID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Error("could not read an inbound reply for its email status",
				zap.String("replyId", replyID.String()), zap.Error(err))
		}

		return
	}

	message, err := s.message(ctx, tenantInfo, reply.MessageID, false)
	if err != nil {
		s.l.Error("could not read the message an inbound reply answers",
			zap.String("replyId", replyID.String()), zap.Error(err))

		return
	}

	if _, err = s.applyEmailStatus(ctx, message, reply, change); err != nil {
		s.l.Error("could not record an inbound reply's email status",
			zap.String("replyId", replyID.String()),
			zap.String("status", string(sent.Status)),
			zap.Error(err),
		)
	}
}

func replyIDFromKey(key string) (pulid.ID, bool) {
	raw, found := strings.CutPrefix(key, inboundmessageports.InboundReplyIdempotencyPrefix)
	if !found {
		return pulid.Nil, false
	}

	id, err := pulid.Parse(raw)
	if err != nil || id.IsNil() {
		return pulid.Nil, false
	}

	return id, true
}

func replySubject(subject string) string {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return "Re: your message"
	}
	if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(replySubjectPrefix)) {
		return trimmed
	}

	return stringutils.TruncateRunes(replySubjectPrefix+trimmed, triage.MaxReplySubjectLength)
}
