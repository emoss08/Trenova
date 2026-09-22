package inboundmessageservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/webhooksig"
	"go.uber.org/fx"
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

// secretDecryptor is the sliver of the encryption service this needs. Taking
// the interface rather than the concrete service keeps the ingest path
// testable without standing up a key manager to check a signature.
type secretDecryptor interface {
	DecryptString(value string) (string, error)
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	MailboxRepo repositories.InboundMailboxRepository
	MessageRepo repositories.InboundMessageRepository
	Storage     storage.Client
	Encryption  *encryptionservice.Service
	// Completion is optional so the ingest path still works on an installation
	// with no provider configured. Mail lands and waits for a person instead of
	// being classified, which is the behaviour a mailbox on AlwaysReview has
	// anyway.
	Completion services.CompletionService `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	mailboxRepo repositories.InboundMailboxRepository
	messageRepo repositories.InboundMessageRepository
	storage     storage.Client
	encryption  secretDecryptor
	completion  services.CompletionService
}

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.inbound-message"),
		mailboxRepo: p.MailboxRepo,
		messageRepo: p.MessageRepo,
		storage:     p.Storage,
		encryption:  p.Encryption,
		completion:  p.Completion,
	}
}

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
		repositories.GetMailboxByTokenHashRequest{
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

	secret, err := s.encryption.DecryptString(mailbox.SigningSecret)
	if err != nil {
		s.l.Error("failed to decrypt a mailbox signing secret",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))

		return fmt.Errorf("%w: the signing secret could not be read", ErrUnverified)
	}

	if err = webhooksig.VerifySvix(webhooksig.SvixParams{
		Secret:    secret,
		ID:        req.SignatureID,
		Timestamp: req.SignatureTimestamp,
		Signature: req.Signature,
		Body:      req.Body,
		Now:       req.ReceivedAt,
	}); err != nil {
		s.l.Warn("rejected an unverified inbound delivery",
			zap.String("mailboxId", mailbox.ID.String()), zap.Error(err))

		return fmt.Errorf("%w: %w", ErrUnverified, err)
	}

	return nil
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
		repositories.GetInboundMessageByProviderIDRequest{
			MailboxID:         mailbox.ID,
			ProviderMessageID: parsed.ProviderMessageID,
		},
	)
	if err == nil && existing != nil {
		return &ReceiveWebhookResult{MessageID: existing.ID.String(), Duplicate: true}, nil
	}

	htmlKey := s.storeRawBody(ctx, mailbox, parsed)

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

	attachments := make([]*inboundmessage.InboundAttachment, 0, len(parsed.Attachments))
	for _, attachment := range parsed.Attachments {
		attachments = append(attachments, &inboundmessage.InboundAttachment{
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
			ByteSize:    int64(len(attachment.Content)),
			Kind:        inboundmessage.AttachmentUnknown,
		})
	}

	created, err := s.messageRepo.Create(ctx, entity, attachments)
	if err != nil {
		return nil, err
	}

	s.l.Info("received an inbound message",
		zap.String("messageId", created.ID.String()),
		zap.String("mailboxId", mailbox.ID.String()),
		zap.Int("attachments", len(attachments)))

	return &ReceiveWebhookResult{MessageID: created.ID.String()}, nil
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
	parsed *providerMessage,
) string {
	if parsed.HTMLBody == "" {
		return ""
	}

	key := fileutils.GenerateStoragePath(
		mailbox.OrganizationID.String(),
		"inbound-messages/"+mailbox.ID.String(),
		parsed.ProviderMessageID+".html",
	)

	if _, err := s.storage.Upload(ctx, &storage.UploadParams{
		Key:         key,
		ContentType: "text/html; charset=utf-8",
		Size:        int64(len(parsed.HTMLBody)),
		Body:        bytes.NewReader([]byte(parsed.HTMLBody)),
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
