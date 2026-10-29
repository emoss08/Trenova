package inboundintake

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/webhooksig"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

var (
	// ErrUnknownMailbox is what an unrecognised token gets. It is deliberately
	// indistinguishable from any other miss: an endpoint that answered
	// differently for a real address than a made-up one would be a way to
	// enumerate which addresses exist.
	ErrUnknownMailbox = errors.New("no such mailbox")
	// ErrNotAnInboundMessage marks a delivery that verified but is not mail —
	// providers post every webhook kind to one endpoint. It is not a failure,
	// and the caller answers 200 so the provider stops retrying.
	ErrNotAnInboundMessage = errors.New("the delivery is not an inbound message")
	ErrUnsupportedProvider = errors.New("the mailbox names a provider that cannot be parsed")
	ErrMalformedPayload    = errors.New("the delivery could not be read")
	ErrUnverified          = errors.New("the delivery could not be verified")
	ErrMailboxInactive     = errors.New("the mailbox is no longer listening")
)

// ReceiveWebhookRequest is one delivery, exactly as it arrived.
type ReceiveWebhookRequest struct {
	// MailboxToken is the plain token from the URL. It is hashed here and never
	// stored or logged.
	MailboxToken string
	Body         []byte
	// The signature headers, whichever the provider sent.
	SignatureID        string
	SignatureTimestamp string
	Signature          string
	// Authorization is the request's Authorization header. Postmark sends the
	// basic-auth credentials from the webhook URL in it, and that is how a
	// Postmark delivery is verified.
	Authorization string
	// ReceivedAt is injectable so the verification window can be tested rather
	// than taken on trust.
	ReceivedAt time.Time
}

type ReceiveWebhookResult struct {
	MessageID string
	// Duplicate marks a redelivery. Providers retry aggressively, and a retry
	// has to be a no-op rather than a second shipment.
	Duplicate bool
	// Ignored marks a verified delivery that was not mail.
	Ignored bool
}

// ReceiveWebhook is the whole ingest path, in the order it has to happen:
// resolve, verify, then write. Nothing touches the database before the
// signature checks out, so an unverified delivery leaves no trace to find.
func (s *Service) ReceiveWebhook(
	ctx context.Context,
	req *ReceiveWebhookRequest,
) (*ReceiveWebhookResult, error) {
	mailbox, err := s.resolveMailbox(ctx, req.MailboxToken)
	if err != nil {
		return nil, err
	}
	ctx = dbscope.WithTenant(ctx, dbscope.Tenant{
		OrganizationID: mailbox.OrganizationID,
		BusinessUnitID: mailbox.BusinessUnitID,
	})

	if err = s.verify(ctx, mailbox, req); err != nil {
		return nil, err
	}

	message, err := parsePayload(mailbox.Provider, req.Body)
	if err != nil {
		if errors.Is(err, ErrNotAnInboundMessage) {
			return &ReceiveWebhookResult{Ignored: true}, nil
		}

		return nil, err
	}

	if mailbox.Provider.DeliversAccountWide() && !message.addressedTo(mailbox.Address) {
		s.l.Info("ignored an inbound delivery not addressed to its mailbox",
			zap.String("mailboxId", mailbox.ID.String()),
			zap.Int("recipients", len(message.Recipients)))

		return &ReceiveWebhookResult{Ignored: true}, nil
	}

	return s.stage(ctx, mailbox, message)
}

// resolveMailbox turns the URL's token into the tenant it belongs to.
//
// The token is hashed before it is used, so the plain value never reaches a
// query, a log or an index — which is the point of storing the hash rather than
// the token in the first place.
func (s *Service) resolveMailbox(
	ctx context.Context,
	token string,
) (*inboundmessage.Mailbox, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrUnknownMailbox
	}

	mailbox, err := s.mailboxRepo.GetByTokenHash(
		ctx,
		inboundmessageports.GetMailboxByTokenHashRequest{
			TokenHash: hashutils.SHA256Hex(token),
		},
	)
	if err != nil {
		// Every failure here is reported the same way. Distinguishing "no such
		// mailbox" from "the database is down" would tell whoever posted which
		// addresses are real.
		return nil, ErrUnknownMailbox
	}
	if mailbox.Status != inboundmessage.MailboxActive {
		return nil, ErrMailboxInactive
	}

	return mailbox, nil
}

func (s *Service) verify(
	ctx context.Context,
	mailbox *inboundmessage.Mailbox,
	req *ReceiveWebhookRequest,
) error {
	if mailbox.SigningSecret == "" {
		// A mailbox with no secret cannot verify anything, so it accepts
		// nothing. Silently trusting the body instead would make the endpoint
		// an open door for whoever guessed the token.
		s.l.Error("a mailbox is listening without a signing secret",
			zap.String("mailboxId", mailbox.ID.String()))

		return fmt.Errorf("%w: the mailbox has no signing secret", ErrUnverified)
	}

	secret, err := s.mailbox.OpenMailboxSecret(
		mailbox,
		inboundmessage.SigningSecretField,
		mailbox.SigningSecret,
	)
	if err != nil {
		s.l.Error("failed to decrypt a mailbox signing secret",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))

		return fmt.Errorf("%w: the signing secret could not be read", ErrUnverified)
	}

	if err = verifyDelivery(mailbox.Provider, secret, req); err != nil {
		s.l.Warn("rejected an unverified inbound delivery",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))

		return fmt.Errorf("%w: %w", ErrUnverified, err)
	}

	return nil
}

// verifyDelivery checks a delivery by the scheme of the provider the mailbox
// is configured for — never by whichever headers the request happens to bring,
// or a caller could pick the check they found easier to satisfy. Resend signs
// with Svix; Postmark does not sign inbound mail, and sends back the basic-auth
// credentials from the webhook URL instead.
func verifyDelivery(
	provider inboundmessage.Provider,
	secret string,
	req *ReceiveWebhookRequest,
) error {
	switch provider {
	case inboundmessage.ProviderResend:
		return webhooksig.VerifySvix(webhooksig.SvixParams{
			Secret:    secret,
			ID:        req.SignatureID,
			Timestamp: req.SignatureTimestamp,
			Signature: req.Signature,
			Body:      req.Body,
			Now:       req.ReceivedAt,
		})
	case inboundmessage.ProviderPostmark:
		return webhooksig.VerifyBasicAuth(secret, req.Authorization)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedProvider, provider)
	}
}

// stage writes the message down before anything tries to understand it.
//
// The order matters: a message that was received and not yet classified is
// recoverable, while one that was classified and never written is a message the
// sender believes arrived and nobody can find.
func (s *Service) stage(
	ctx context.Context,
	mailbox *inboundmessage.Mailbox,
	parsed *providerMessage,
) (*ReceiveWebhookResult, error) {
	existing, err := s.messageRepo.GetByProviderID(
		ctx,
		inboundmessageports.GetInboundMessageByProviderIDRequest{
			MailboxID:         mailbox.ID,
			ProviderMessageID: parsed.ProviderMessageID,
		},
	)
	if err == nil && existing != nil {
		return &ReceiveWebhookResult{MessageID: existing.ID.String(), Duplicate: true}, nil
	}

	htmlKey := s.storeRawBody(ctx, mailbox, parsed.ProviderMessageID, parsed.HTMLBody)

	entity := &inboundmessage.InboundMessage{
		BusinessUnitID:    mailbox.BusinessUnitID,
		OrganizationID:    mailbox.OrganizationID,
		MailboxID:         mailbox.ID,
		ProviderMessageID: parsed.ProviderMessageID,
		MessageID:         parsed.MessageID,
		InReplyTo:         parsed.InReplyTo,
		References:        parsed.References,
		FromAddress:       parsed.FromAddress,
		FromName:          parsed.FromName,
		ToAddresses:       parsed.ToAddresses,
		CcAddresses:       parsed.CcAddresses,
		Subject:           parsed.Subject,
		TextBody:          boundedText(parsed.TextBody),
		HTMLKey:           htmlKey,
		ReceivedAt:        parsed.ReceivedAt,
		SpamScore:         parsed.SpamScore,
		Status:            inboundmessage.StatusReceived,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	tenantInfo := pagination.TenantInfo{OrgID: mailbox.OrganizationID, BuID: mailbox.BusinessUnitID}
	blocked := mailbox.Blocks(entity.FromAddress)
	if blocked {
		entity.Status = inboundmessage.StatusIgnored
		entity.ReviewedAt = s.now()
		entity.ReviewNote = blockedSenderNote
	}
	threadKey, err := s.threadKeyFor(ctx, &threadLookup{
		tenant:     tenantInfo,
		ownKey:     inboundmessage.NormalizeMessageID(entity.MessageID),
		inReplyTo:  entity.InReplyTo,
		references: entity.References,
	})
	if err != nil {
		s.l.Warn("could not find the thread an inbound message belongs to; it starts its own",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))
	}
	entity.ThreadKey = threadKey

	attachments := make([]*inboundmessage.InboundAttachment, 0, len(parsed.Attachments))
	for _, attachment := range parsed.Attachments {
		attachments = append(attachments, &inboundmessage.InboundAttachment{
			FileName:             attachment.FileName,
			ContentType:          attachment.ContentType,
			ByteSize:             int64(len(attachment.Content)),
			Kind:                 inboundmessage.AttachmentUnknown,
			ProviderAttachmentID: attachment.ProviderID,
		})
	}

	created, err := s.messageRepo.Create(ctx, entity, attachments)
	if err != nil {
		return nil, err
	}

	s.l.Info("received an inbound message",
		zap.String("messageId", created.ID.String()),
		zap.String("mailboxId", mailbox.ID.String()),
		zap.Int("attachments", len(attachments)),
		zap.Bool("blocked", blocked))

	created.Mailbox = mailbox
	s.markReceived(ctx, tenantInfo, mailbox, created.ReceivedAt)
	s.recordArrival(ctx, mailbox, created, blocked)
	s.notify.MessagesCreated(ctx, created)
	if blocked {
		return &ReceiveWebhookResult{MessageID: created.ID.String()}, nil
	}

	if !mailbox.Provider.FetchesContent() {
		s.stageAttachments(ctx, created, parsed)
	}
	s.answerAwaiting(ctx, created)
	s.startProcessing(ctx, created)

	return &ReceiveWebhookResult{MessageID: created.ID.String()}, nil
}

const blockedSenderNote = "This sender is blocked on this address, so the message was ignored."

func (s *Service) markReceived(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	mailbox *inboundmessage.Mailbox,
	receivedAt int64,
) {
	if err := s.mailboxState.MarkReceived(ctx, inboundmessageports.MailboxReceivedRequest{
		TenantInfo: tenantInfo,
		MailboxID:  mailbox.ID,
		ReceivedAt: receivedAt,
	}); err != nil {
		s.l.Warn("could not record when a mailbox last received mail",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))
	}
}

func (s *Service) recordArrival(
	ctx context.Context,
	mailbox *inboundmessage.Mailbox,
	message *inboundmessage.InboundMessage,
	blocked bool,
) {
	activities := make([]*triage.Activity, 0, 2)
	if blocked {
		activities = append(
			activities,
			triage.NewActivity(message, triage.ActivityIgnored).
				About(message.FromAddress, "blocked"),
		)
	}
	if activity := s.mailbox.NoteTestReceived(ctx, mailbox, message); activity != nil {
		activities = append(activities, activity)
	}
	if len(activities) == 0 {
		return
	}

	if err := s.activityRepo.Insert(ctx, activities); err != nil {
		s.l.Warn("could not record what happened to an arriving message",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}
}

// storeRawBody keeps the whole message where the row cannot.
//
// A failure here never fails the delivery. The bounded text on the row is what
// the classifier and a person both read; the stored copy is for the cases where
// the formatting or a header matters, and losing it costs less than making the
// provider retry a message that was otherwise fine.
func (s *Service) storeRawBody(
	ctx context.Context,
	mailbox *inboundmessage.Mailbox,
	providerMessageID string,
	htmlBody string,
) string {
	if htmlBody == "" {
		return ""
	}

	key := fileutils.GenerateStoragePath(
		mailbox.OrganizationID.String(),
		"inbound-messages/"+mailbox.ID.String(),
		providerMessageID+".html",
	)

	if _, err := s.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		ContentType: "text/html; charset=utf-8",
		Size:        int64(len(htmlBody)),
		Body:        strings.NewReader(htmlBody),
		Metadata: map[string]string{
			"organization-id": mailbox.OrganizationID.String(),
			"mailbox-id":      mailbox.ID.String(),
		},
	}); err != nil {
		s.l.Warn("failed to store an inbound message body",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))

		return ""
	}

	return key
}

// startProcessing hands the message to the workflow that reads it.
//
// A failure here never fails the delivery. The message is already written down,
// which is the part that cannot be recovered; the reading can be retried by a
// sweep or by a person, and making the provider redeliver a message that landed
// correctly would be the worse outcome.
func (s *Service) startProcessing(ctx context.Context, message *inboundmessage.InboundMessage) {
	if s.workflows == nil || !s.workflows.Enabled() {
		s.readContentDetached(ctx, message)

		return
	}

	if err := s.startReading(ctx, message, readingWorkflow{
		// Keyed by the message, so a redelivery that somehow got past the
		// idempotency read cannot start a second reading of the same mail.
		id:    "inbound-message/" + message.ID.String(),
		reuse: enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
	}); err != nil {
		s.l.Warn("failed to start reading an inbound message; it stays at Received",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}
}

type readingWorkflow struct {
	id    string
	reuse enums.WorkflowIdReusePolicy
}

func (s *Service) startReading(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	wf readingWorkflow,
) error {
	_, err := s.workflows.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                    wf.id,
			TaskQueue:             temporaltype.TaskQueueSystem.String(),
			WorkflowIDReusePolicy: wf.reuse,
		},
		temporaltype.ProcessInboundMessageWorkflowName,
		&ProcessInboundMessagePayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: message.OrganizationID,
				BusinessUnitID: message.BusinessUnitID,
			},
			MessageID: message.ID,
		},
	)

	return err
}
