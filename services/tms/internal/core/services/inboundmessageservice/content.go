package inboundmessageservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/htmlutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// ContentUnavailableCode marks a message whose body and attachments could not
// be read from its provider, so it went to review on its metadata alone.
const ContentUnavailableCode = "CONTENT_UNAVAILABLE"

// FetchContent reads what a metadata-only webhook left out — the body, the
// threading headers and the attachments — and writes it onto the staged
// message.
//
// It is idempotent: a message already settled is left alone, the body is
// rewritten to the same value, and a file that already has an upload session,
// a document or a recorded failure is not uploaded again. So a retried activity
// costs a second read from the provider and nothing else.
//
// An error wraps services.ErrInboundContentRejected when another attempt will
// not help and services.ErrInboundContentUnavailable when it might.
func (s *Service) FetchContent(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	message, err := s.messageRepo.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenantInfo,
		IncludeAttachments: true,
	})
	if err != nil {
		return err
	}

	return s.fetchContent(ctx, message, tenantInfo)
}

func (s *Service) fetchContent(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	tenantInfo pagination.TenantInfo,
) error {
	mailbox := message.Mailbox
	if mailbox == nil {
		return fmt.Errorf("message %s has no mailbox to read its provider from", message.ID)
	}
	if !mailbox.Provider.FetchesContent() {
		return nil
	}
	if message.Status != inboundmessage.StatusReceived &&
		message.Status != inboundmessage.StatusProcessing {
		return nil
	}

	apiKey, err := s.mailboxAPIKey(mailbox)
	if err != nil {
		return err
	}
	if s.fetcher == nil {
		return fmt.Errorf("%w: this installation cannot read %s messages",
			services.ErrInboundContentRejected, mailbox.Provider)
	}

	content, err := s.fetcher.Fetch(ctx, &services.InboundContentRequest{
		Provider:          mailbox.Provider,
		APIKey:            apiKey,
		ProviderMessageID: message.ProviderMessageID,
	})
	if err != nil {
		return err
	}

	updated, err := s.applyContent(ctx, message, content)
	if err != nil {
		return err
	}

	s.stageFetchedAttachments(ctx, updated, content, tenantInfo)
	s.publishMessage(ctx, updated, inboxRealtimeAction)

	s.l.Info("read an inbound message's content from its provider",
		zap.String("messageId", updated.ID.String()),
		zap.String("provider", string(mailbox.Provider)),
		zap.Int("attachments", len(content.Attachments)))

	return nil
}

// mailboxAPIKey opens the mailbox's sealed key. A missing or unreadable key is
// a rejection: no number of retries supplies one.
func (s *Service) mailboxAPIKey(mailbox *inboundmessage.Mailbox) (string, error) {
	if mailbox.ProviderAPIKey == "" {
		return "", fmt.Errorf("%w: the mailbox has no %s API key",
			services.ErrInboundContentRejected, mailbox.Provider)
	}

	apiKey, err := s.encryption.DecryptString(mailbox.ProviderAPIKey)
	if err != nil {
		s.l.Error("failed to decrypt a mailbox API key",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))

		return "", fmt.Errorf("%w: the mailbox's %s API key could not be read",
			services.ErrInboundContentRejected, mailbox.Provider)
	}

	return apiKey, nil
}

// applyContent writes the body and headers onto the message. A message that
// came with HTML only is read into text, because the text is what the
// classifier and the inbox's preview both read.
func (s *Service) applyContent(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	content *services.InboundContent,
) (*inboundmessage.InboundMessage, error) {
	text := content.Text
	if strings.TrimSpace(text) == "" {
		text = htmlutils.ToText(content.HTML)
	}

	message.TextBody = boundedText(text)
	if key := s.storeRawBody(
		ctx,
		message.Mailbox,
		message.ProviderMessageID,
		content.HTML,
	); key != "" {
		message.HTMLKey = key
	}
	if message.MessageID == "" {
		message.MessageID = content.MessageID
	}
	if content.InReplyTo != "" {
		message.InReplyTo = content.InReplyTo
	}
	if len(content.References) > 0 {
		message.References = content.References
	}

	attachments := message.Attachments
	mailbox := message.Mailbox

	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}
	updated.Attachments = attachments
	updated.Mailbox = mailbox

	return updated, nil
}

// stageFetchedAttachments carries each fetched file to the row the webhook
// created for it, matched by the provider's id. A row the provider no longer
// lists is recorded as missing rather than left waiting.
func (s *Service) stageFetchedAttachments(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	content *services.InboundContent,
	tenantInfo pagination.TenantInfo,
) {
	if len(message.Attachments) == 0 {
		return
	}

	byID := make(map[string]*services.InboundContentAttachment, len(content.Attachments))
	for i := range content.Attachments {
		byID[content.Attachments[i].ProviderID] = &content.Attachments[i]
	}

	for _, row := range message.Attachments {
		if attachmentSettled(row) {
			continue
		}

		fetched, ok := byID[row.ProviderAttachmentID]
		switch {
		case !ok:
			s.recordAttachmentFailure(ctx, row, "The provider did not return this file.")
		case fetched.FailureText != "":
			s.recordAttachmentFailure(ctx, row, fetched.FailureText)
		case s.uploads == nil:
			s.recordAttachmentFailure(ctx, row,
				"This installation cannot store attachments, so the file was not kept.")
		default:
			row.ByteSize = int64(len(fetched.Content))
			s.stageAttachment(ctx, tenantInfo, message, row, fetched.Content)
		}
	}
}

// attachmentSettled is a row with nothing left for a fetch to do.
func attachmentSettled(row *inboundmessage.InboundAttachment) bool {
	return row.UploadSessionID.IsNotNil() || row.DocumentID.IsNotNil() || row.FailureText != ""
}

// MarkContentUnavailable sends a message to review saying why its content
// could not be read, and records the same on every file still waiting, so
// nothing on the message looks as though it is still being worked on.
func (s *Service) MarkContentUnavailable(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
	reason string,
) error {
	rows, err := s.messageRepo.ListAttachments(ctx, messageID, tenantInfo)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if !attachmentSettled(row) {
			s.recordAttachmentFailure(ctx, row,
				"The file could not be read from the provider with the message.")
		}
	}

	return s.MarkFailed(ctx, messageID, tenantInfo, ContentUnavailableCode, reason)
}

// ContentFailureText says, in words the inbox can show, why a message's
// content could not be read and what fixes it.
func ContentFailureText(cause error) string {
	detail := cause.Error()
	for _, sentinel := range []error{
		services.ErrInboundContentRejected,
		services.ErrInboundContentUnavailable,
	} {
		detail = strings.TrimPrefix(detail, sentinel.Error()+": ")
	}

	text := "The body and attachments of this message could not be read from the provider: " +
		detail + "."
	if errors.Is(cause, services.ErrInboundContentRejected) {
		text += " Set a full access API key on the mailbox in Inbound mailboxes; mail that " +
			"arrives afterwards is read in full."
	}

	return text
}

// inlineFetchTimeout bounds a fetch that runs without a worker: long enough to
// download a message's attachments, short enough that a provider that never
// answers does not hold a goroutine indefinitely.
const inlineFetchTimeout = 10 * time.Minute

// failurePersistTimeout bounds writing down a failed inline fetch. It runs on
// its own context because the fetch's may be the thing that ran out: a fetch
// that timed out would otherwise leave the message at Received with nothing
// else ever coming to read it.
const failurePersistTimeout = 30 * time.Second

// readContentDetached runs the inline fetch off the request path. The delivery
// has already been written down, so the provider is answered at once rather
// than waiting on attachment downloads and timing out into a redelivery. The
// context keeps the request's tenant scope but not its cancellation.
func (s *Service) readContentDetached(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) {
	if message.Mailbox == nil || !message.Mailbox.Provider.FetchesContent() {
		return
	}

	detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), inlineFetchTimeout)
	go func() {
		defer cancel()
		s.fetchContentInline(detached, message)
	}()
}

// fetchContentInline reads a message's content on an installation with no
// worker, where nothing else would ever read it. A failure is written onto the message rather than returned, because
// the delivery itself succeeded.
func (s *Service) fetchContentInline(ctx context.Context, message *inboundmessage.InboundMessage) {
	if message.Mailbox == nil || !message.Mailbox.Provider.FetchesContent() {
		return
	}

	tenantInfo := pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID}

	err := s.fetchContent(ctx, message, tenantInfo)
	if err == nil {
		return
	}

	s.l.Warn("could not read an inbound message's content",
		zap.String("messageId", message.ID.String()), zap.Error(err))

	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), failurePersistTimeout)
	defer cancel()
	if markErr := s.MarkContentUnavailable(
		persistCtx, message.ID, tenantInfo, ContentFailureText(err),
	); markErr != nil {
		s.l.Error("could not record that an inbound message's content is unavailable",
			zap.String("messageId", message.ID.String()), zap.Error(markErr))
	}
}
