package inboundmessageservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentuploadports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/webhooksig"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
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

// secretKeeper is the sliver of the encryption service this needs: sealing a
// mailbox's signing secret when it is set, and opening it to check a delivery.
// Taking the interface rather than the concrete service keeps both paths
// testable without standing up a key manager.
type secretKeeper interface {
	EncryptString(value string) (string, error)
	DecryptString(value string) (string, error)
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	MailboxRepo inboundmessageports.InboundMailboxRepository
	MessageRepo inboundmessageports.InboundMessageRepository
	Storage     kernelports.Client
	Encryption  *encryptionservice.Service
	// Completion is optional so the ingest path still works on an installation
	// with no provider configured. Mail lands and waits for a person instead of
	// being classified, which is the behaviour a mailbox on AlwaysReview has
	// anyway.
	Completion agentports.CompletionService `optional:"true"`
	// Workflows is optional so an installation without a worker still receives
	// mail. It lands at Received and waits, which is what a mailbox on
	// AlwaysReview does with every message anyway.
	Workflows kernelports.WorkflowStarter `optional:"true"`
	// Shipments and Parties resolve what a message is about. Both are optional:
	// an unmatched message is still one a person can read and link by hand.
	Shipments ShipmentFinder `optional:"true"`
	Parties   PartyFinder    `optional:"true"`
	// Uploads and Documents carry attachments through the document pipeline.
	// Without them a message still lands with its attachment rows recorded —
	// the file names, types and sizes the sender attached — and each row says
	// it was never read, which is a truthful inbox rather than an empty one.
	Uploads   documentuploadports.DocumentUploadService `optional:"true"`
	Documents documentports.DocumentRepository          `optional:"true"`
	// Watchtower and Events are how a message that needs a person, or a desk,
	// gets one. Both are optional: without them the inbox still holds every
	// message and its reason, and nothing else is told about it.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
	Events     agentports.AgentEventPublisher      `optional:"true"`
	// Audit records mailbox configuration changes.
	Audit auditports.AuditService `optional:"true"`
	// Realtime keeps an open inbox current, and Notifications tells the people
	// who read the inbox that a message is waiting on them. Both optional: the
	// inbox is still right on the next load without either.
	Realtime      agentports.RealtimeService   `optional:"true"`
	Notifications *notificationservice.Service `optional:"true"`
	Indexer       agentports.RetrievalIndexer  `optional:"true"`
	// ContentFetcher reads what a metadata-only webhook leaves out. Without it
	// a Resend message lands with its sender, recipients and subject and is
	// sent to review saying its content could not be read.
	ContentFetcher inboundmessageports.InboundContentFetcher `optional:"true"`

	DB            kernelports.DBConnection
	QueueRepo     inboundmessageports.TriageQueueRepository
	MailboxState  inboundmessageports.TriageMailboxStateRepository
	ActivityRepo  inboundmessageports.TriageActivityRepository
	ViewRepo      inboundmessageports.TriageViewRepository
	TargetRepo    inboundmessageports.TriageTargetRepository
	SettingsRepo  inboundmessageports.TriageSettingsRepository
	DocumentTypes inboundmessageports.InboundDocumentTypeFinder
	Roles         permissionports.RoleRepository
	Organizations tenantports.OrganizationRepository
	Replies       inboundmessageports.InboundReplyReader        `optional:"true"`
	RuleEvaluator inboundmessageports.TriageRuleEvaluator       `optional:"true"`
	RuleTeacher   inboundmessageports.TriageRuleTeacher         `optional:"true"`
	Hints         inboundmessageports.SenderHintProvider        `optional:"true"`
	Briefing      inboundmessageports.TriageBriefingInvalidator `optional:"true"`
	Wakes         inboundmessageports.TriageWakeScheduler       `optional:"true"`
	Email         emailports.EmailService                       `optional:"true"`
}

// reviewNotifier is the slice of the notification service the inbox uses.
type reviewNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

type Service struct {
	l           *zap.Logger
	mailboxRepo inboundmessageports.InboundMailboxRepository
	messageRepo inboundmessageports.InboundMessageRepository
	storage     kernelports.Client
	encryption  secretKeeper
	completion  agentports.CompletionService
	workflows   kernelports.WorkflowStarter
	shipments   ShipmentFinder
	parties     PartyFinder
	uploads     documentuploadports.DocumentUploadService
	documents   documentports.DocumentRepository
	watchtower  watchtowerports.WatchtowerProjector
	events      agentports.AgentEventPublisher
	audit       auditports.AuditService
	realtime    agentports.RealtimeService
	notifier    reviewNotifier
	indexer     agentports.RetrievalIndexer
	fetcher     inboundmessageports.InboundContentFetcher

	db            kernelports.DBConnection
	queueRepo     inboundmessageports.TriageQueueRepository
	mailboxState  inboundmessageports.TriageMailboxStateRepository
	activityRepo  inboundmessageports.TriageActivityRepository
	viewRepo      inboundmessageports.TriageViewRepository
	targetRepo    inboundmessageports.TriageTargetRepository
	settingsRepo  inboundmessageports.TriageSettingsRepository
	documentTypes inboundmessageports.InboundDocumentTypeFinder
	directory     assigneeDirectory
	organizations organizationReader
	replies       inboundmessageports.InboundReplyReader
	ruleEvaluator inboundmessageports.TriageRuleEvaluator
	ruleTeacher   inboundmessageports.TriageRuleTeacher
	hints         inboundmessageports.SenderHintProvider
	wakes         inboundmessageports.TriageWakeScheduler
	email         emailports.EmailService
	notify        *Notifier
	now           func() int64
}

func New(p Params) *Service {
	var notifier reviewNotifier
	if p.Notifications != nil {
		notifier = p.Notifications
	}

	return &Service{
		l:           p.Logger.Named("service.inbound-message"),
		mailboxRepo: p.MailboxRepo,
		messageRepo: p.MessageRepo,
		storage:     p.Storage,
		encryption:  p.Encryption,
		completion:  p.Completion,
		workflows:   p.Workflows,
		shipments:   p.Shipments,
		parties:     p.Parties,
		uploads:     p.Uploads,
		documents:   p.Documents,
		watchtower:  p.Watchtower,
		events:      p.Events,
		audit:       p.Audit,
		realtime:    p.Realtime,
		notifier:    notifier,
		indexer:     p.Indexer,
		fetcher:     p.ContentFetcher,

		db:            p.DB,
		queueRepo:     p.QueueRepo,
		mailboxState:  p.MailboxState,
		activityRepo:  p.ActivityRepo,
		viewRepo:      p.ViewRepo,
		targetRepo:    p.TargetRepo,
		settingsRepo:  p.SettingsRepo,
		documentTypes: p.DocumentTypes,
		directory:     p.Roles,
		organizations: p.Organizations,
		replies:       p.Replies,
		ruleEvaluator: p.RuleEvaluator,
		ruleTeacher:   p.RuleTeacher,
		hints:         p.Hints,
		wakes:         p.Wakes,
		email:         p.Email,
		notify: newNotifier(notifierDeps{
			logger:     p.Logger,
			realtime:   p.Realtime,
			watchtower: p.Watchtower,
			briefing:   p.Briefing,
		}),
		now: timeutils.NowUnix,
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

	secret, err := s.encryption.DecryptString(mailbox.SigningSecret)
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
	if activity := s.noteTestReceived(ctx, mailbox, message); activity != nil {
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
