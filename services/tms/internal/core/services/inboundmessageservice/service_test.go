package inboundmessageservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// The stubs embed their interfaces because this file exercises one path; a stub
// that answered the rest would be noise around the four calls that matter.

type stubMailboxRepo struct {
	repositories.InboundMailboxRepository

	mailbox *inboundmessage.Mailbox
	err     error
	// asked records the hash the service looked up, so the test can prove the
	// plain token never reaches the query.
	asked string
}

func (s *stubMailboxRepo) GetByTokenHash(
	_ context.Context, req repositories.GetMailboxByTokenHashRequest,
) (*inboundmessage.Mailbox, error) {
	s.asked = req.TokenHash
	if s.err != nil {
		return nil, s.err
	}

	return s.mailbox, nil
}

type stubMessageRepo struct {
	repositories.InboundMessageRepository

	existing *inboundmessage.InboundMessage
	created  *inboundmessage.InboundMessage
	attached []*inboundmessage.InboundAttachment
	writes   int
}

func (s *stubMessageRepo) GetByProviderID(
	_ context.Context, _ repositories.GetInboundMessageByProviderIDRequest,
) (*inboundmessage.InboundMessage, error) {
	if s.existing == nil {
		return nil, errortypes.NewNotFoundError("InboundMessage not found")
	}

	return s.existing, nil
}

func (s *stubMessageRepo) Create(
	_ context.Context,
	entity *inboundmessage.InboundMessage,
	attachments []*inboundmessage.InboundAttachment,
) (*inboundmessage.InboundMessage, error) {
	s.writes++
	entity.ID = pulid.MustNew("imsg_")
	s.created = entity
	s.attached = attachments

	return entity, nil
}

type stubStorage struct {
	storage.Client

	uploaded string
	body     []byte
	err      error
}

func (s *stubStorage) Upload(
	_ context.Context, params *storage.UploadParams,
) (*storage.FileInfo, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.uploaded = params.Key
	s.body, _ = io.ReadAll(params.Body)

	return &storage.FileInfo{Key: params.Key}, nil
}

const (
	testToken  = "imbx_live_2f8a91c0"
	testSecret = "c2lnbmluZy1rZXktZm9yLXRoZS10ZXN0cw=="
)

func testMailbox() *inboundmessage.Mailbox {
	return &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Address:        "tenders@acme-logistics.com",
		Provider:       inboundmessage.ProviderResend,
		TokenHash:      hashutils.SHA256Hex(testToken),
		SigningSecret:  testSecret,
		ReviewPolicy:   inboundmessage.ReviewAlways,
		Status:         inboundmessage.MailboxActive,
	}
}

// signed builds a delivery the service should accept, so every refusal test is
// a single deliberate deviation from a known-good request.
func signed(t *testing.T, body []byte, at time.Time) *ReceiveWebhookRequest {
	t.Helper()

	key, err := base64.StdEncoding.DecodeString(testSecret)
	require.NoError(t, err)

	id := "msg_01"
	stamp := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + stamp + "."))
	mac.Write(body)

	return &ReceiveWebhookRequest{
		MailboxToken:       testToken,
		Body:               body,
		SignatureID:        id,
		SignatureTimestamp: stamp,
		Signature:          "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil)),
		ReceivedAt:         at,
	}
}

// resendBody is Resend's email.received event as Resend sends it: metadata
// only. The body, headers and attachment bytes are fetched afterwards.
func resendBody() []byte {
	return []byte(`{
		"type": "email.received",
		"created_at": "2026-07-15T16:00:01.126Z",
		"data": {
			"email_id": "e_9f2b",
			"created_at": "2026-07-15T16:00:00.894Z",
			"from": "Dispatch <dispatch@bigshipper.com>",
			"to": ["Tenders <Tenders@Acme-Logistics.com>"],
			"cc": ["billing@acme-logistics.com"],
			"bcc": [],
			"received_for": ["tenders@acme-logistics.com"],
			"message_id": "<abc@bigshipper.com>",
			"subject": "Load 88213 tender",
			"attachments": [
				{
					"id": "att_1",
					"filename": "tender.pdf",
					"content_type": "application/pdf",
					"content_disposition": "attachment",
					"content_id": null
				}
			]
		}
	}`)
}

// plainSecrets stands in for the key manager. The secret this decrypts is
// already the base64 signing key, so the signature maths is exercised for real
// while the envelope encryption is somebody else's tested concern.
type plainSecrets struct{}

// sealedPrefix marks a value the stub has "sealed", so a test can tell a
// secret that went through the keeper from one stored in the clear.
const sealedPrefix = "sealed:"

func (plainSecrets) EncryptString(value string) (string, error) { return sealedPrefix + value, nil }

func (plainSecrets) DecryptString(value string) (string, error) {
	return strings.TrimPrefix(value, sealedPrefix), nil
}

// stubWorkflows is a worker that is there, so a staged message is handed to
// the workflow rather than read on the request path.
type stubWorkflows struct {
	services.WorkflowStarter

	started []client.StartWorkflowOptions
}

func (w *stubWorkflows) Enabled() bool { return true }

func (w *stubWorkflows) StartWorkflow(
	_ context.Context, options client.StartWorkflowOptions, _ any, _ ...any,
) (client.WorkflowRun, error) {
	w.started = append(w.started, options)

	return nil, nil
}

func newService(
	mailboxes *stubMailboxRepo,
	messages *stubMessageRepo,
	store *stubStorage,
) *Service {
	return &Service{
		l:           zap.NewNop(),
		mailboxRepo: mailboxes,
		messageRepo: messages,
		storage:     store,
		encryption:  plainSecrets{},
		workflows:   &stubWorkflows{},
	}
}

func TestReceiveWebhook_StagesAVerifiedDelivery(t *testing.T) {
	t.Parallel()

	mailboxes := &stubMailboxRepo{mailbox: testMailbox()}
	messages := &stubMessageRepo{}
	store := &stubStorage{}
	svc := newService(mailboxes, messages, store)

	now := time.Unix(1784131200, 0)
	result, err := svc.ReceiveWebhook(t.Context(), signed(t, resendBody(), now))
	require.NoError(t, err)
	assert.False(t, result.Duplicate)

	require.NotNil(t, messages.created)
	assert.Equal(t, "e_9f2b", messages.created.ProviderMessageID)
	assert.Equal(t, "<abc@bigshipper.com>", messages.created.MessageID)
	assert.Equal(t, "dispatch@bigshipper.com", messages.created.FromAddress)
	assert.Equal(t, "Dispatch", messages.created.FromName)
	assert.Equal(t, []string{"tenders@acme-logistics.com"}, messages.created.ToAddresses)
	assert.Equal(t, []string{"billing@acme-logistics.com"}, messages.created.CcAddresses)
	assert.Equal(t, "Load 88213 tender", messages.created.Subject)
	assert.Equal(t, int64(1784131200), messages.created.ReceivedAt)
	assert.Empty(t, messages.created.TextBody, "Resend's webhook carries no body")
	assert.Equal(t, inboundmessage.StatusReceived, messages.created.Status)
	require.Len(t, messages.attached, 1)
	assert.Equal(t, "tender.pdf", messages.attached[0].FileName)
	assert.Equal(t, "att_1", messages.attached[0].ProviderAttachmentID)
	assert.Zero(t, messages.attached[0].ByteSize)
	assert.Empty(t, messages.attached[0].FailureText,
		"a file whose bytes are still to be fetched is not refused as empty")
	assert.Empty(t, store.uploaded)
}

/*
Resend posts every message its account receives to each webhook. A mailbox
that staged all of them would fill one organization's inbox with mail meant
for another address — or, with two mailboxes on one account, stage every
message twice. Only mail naming the mailbox's address lands.
*/
func TestReceiveWebhook_IgnoresResendMailForAnotherAddress(t *testing.T) {
	t.Parallel()

	mailbox := testMailbox()
	mailbox.Address = "pods@acme-logistics.com"
	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: mailbox}, messages, &stubStorage{})

	result, err := svc.ReceiveWebhook(t.Context(), signed(t, resendBody(), time.Unix(1784131200, 0)))
	require.NoError(t, err)

	assert.True(t, result.Ignored)
	assert.Zero(t, messages.writes)
}

// Bcc'd mail and mail forwarded to the provider's address name the mailbox
// only in bcc or received_for, and both still belong to it.
func TestReceiveWebhook_AcceptsResendMailAddressedByBccOrReceivedFor(t *testing.T) {
	t.Parallel()

	for name, address := range map[string]string{
		"bcc":          "ops@acme-logistics.com",
		"received for": "intake@inbound.acme-logistics.com",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body := []byte(`{"type":"email.received","data":{
				"email_id":"e_1","from":"dispatch@bigshipper.com",
				"to":["someone@elsewhere.com"],
				"bcc":["ops@acme-logistics.com"],
				"received_for":["intake@inbound.acme-logistics.com"],
				"subject":"Tender"}}`)
			mailbox := testMailbox()
			mailbox.Address = address
			messages := &stubMessageRepo{}
			svc := newService(&stubMailboxRepo{mailbox: mailbox}, messages, &stubStorage{})

			result, err := svc.ReceiveWebhook(t.Context(), signed(t, body, time.Unix(1784131200, 0)))
			require.NoError(t, err)
			assert.False(t, result.Ignored)
			assert.Equal(t, 1, messages.writes)
		})
	}
}

// The token is hashed before it is used, so the plain value never reaches a
// query, a log or an index — which is the whole reason the column stores a hash.
func TestReceiveWebhook_LooksTheMailboxUpByHashNotToken(t *testing.T) {
	t.Parallel()

	mailboxes := &stubMailboxRepo{mailbox: testMailbox()}
	svc := newService(mailboxes, &stubMessageRepo{}, &stubStorage{})

	_, err := svc.ReceiveWebhook(t.Context(), signed(t, resendBody(), time.Unix(1784131200, 0)))
	require.NoError(t, err)

	assert.Equal(t, hashutils.SHA256Hex(testToken), mailboxes.asked)
	assert.NotEqual(t, testToken, mailboxes.asked)
}

/*
Nothing is written before the signature checks out.

An endpoint that staged first and verified second would let anyone who guessed
a token fill the inbox with messages that were never sent — and each one would
look, to the person reviewing it, exactly like a real one.
*/
func TestReceiveWebhook_WritesNothingWhenTheSignatureIsWrong(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	store := &stubStorage{}
	svc := newService(&stubMailboxRepo{mailbox: testMailbox()}, messages, store)

	req := signed(t, resendBody(), time.Unix(1784131200, 0))
	req.Signature = "v1," + base64.StdEncoding.EncodeToString([]byte("not the signature"))

	_, err := svc.ReceiveWebhook(t.Context(), req)
	require.ErrorIs(t, err, ErrUnverified)
	assert.Zero(t, messages.writes)
	assert.Empty(t, store.uploaded)
}

// A captured delivery replayed later must not land a second time, even though
// its signature is genuine.
func TestReceiveWebhook_RefusesAStaleReplay(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: testMailbox()}, messages, &stubStorage{})

	signedAt := time.Unix(1784131200, 0)
	req := signed(t, resendBody(), signedAt)
	req.ReceivedAt = signedAt.Add(time.Hour)

	_, err := svc.ReceiveWebhook(t.Context(), req)
	require.ErrorIs(t, err, ErrUnverified)
	assert.Zero(t, messages.writes)
}

// A mailbox with no secret can verify nothing, so it accepts nothing. Trusting
// the body instead would make the endpoint an open door for a guessed token.
func TestReceiveWebhook_RefusesAMailboxWithNoSigningSecret(t *testing.T) {
	t.Parallel()

	mailbox := testMailbox()
	mailbox.SigningSecret = ""
	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: mailbox}, messages, &stubStorage{})

	_, err := svc.ReceiveWebhook(t.Context(), signed(t, resendBody(), time.Unix(1784131200, 0)))
	require.ErrorIs(t, err, ErrUnverified)
	assert.Zero(t, messages.writes)
}

// Every lookup failure reads the same. An endpoint that distinguished "no such
// mailbox" from "the database is down" would let anyone discover which
// addresses exist by posting to guesses.
func TestReceiveWebhook_ReportsEveryLookupFailureIdentically(t *testing.T) {
	t.Parallel()

	for name, repo := range map[string]*stubMailboxRepo{
		"no such token":    {err: errortypes.NewNotFoundError("Mailbox not found")},
		"database is down": {err: errors.New("connection refused")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svc := newService(repo, &stubMessageRepo{}, &stubStorage{})
			_, err := svc.ReceiveWebhook(
				t.Context(), signed(t, resendBody(), time.Unix(1784131200, 0)),
			)
			require.ErrorIs(t, err, ErrUnknownMailbox)
		})
	}
}

func TestReceiveWebhook_RefusesAnEmptyToken(t *testing.T) {
	t.Parallel()

	mailboxes := &stubMailboxRepo{mailbox: testMailbox()}
	svc := newService(mailboxes, &stubMessageRepo{}, &stubStorage{})

	req := signed(t, resendBody(), time.Unix(1784131200, 0))
	req.MailboxToken = "   "

	_, err := svc.ReceiveWebhook(t.Context(), req)
	require.ErrorIs(t, err, ErrUnknownMailbox)
	assert.Empty(t, mailboxes.asked, "an empty token never reaches the repository")
}

func TestReceiveWebhook_RefusesAnInactiveMailbox(t *testing.T) {
	t.Parallel()

	mailbox := testMailbox()
	mailbox.Status = inboundmessage.MailboxInactive
	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: mailbox}, messages, &stubStorage{})

	_, err := svc.ReceiveWebhook(t.Context(), signed(t, resendBody(), time.Unix(1784131200, 0)))
	require.ErrorIs(t, err, ErrMailboxInactive)
	assert.Zero(t, messages.writes)
}

/*
Providers retry aggressively, and every one of them will redeliver a message it
is not sure landed. A retry that created a second row would be a second tender,
a second shipment and a second invoice from one email.
*/
func TestReceiveWebhook_TreatsARedeliveryAsANoOp(t *testing.T) {
	t.Parallel()

	existing := &inboundmessage.InboundMessage{ID: pulid.MustNew("imsg_")}
	messages := &stubMessageRepo{existing: existing}
	store := &stubStorage{}
	svc := newService(&stubMailboxRepo{mailbox: testMailbox()}, messages, store)

	result, err := svc.ReceiveWebhook(
		t.Context(), signed(t, resendBody(), time.Unix(1784131200, 0)),
	)
	require.NoError(t, err)

	assert.True(t, result.Duplicate)
	assert.Equal(t, existing.ID.String(), result.MessageID)
	assert.Zero(t, messages.writes)
	assert.Empty(t, store.uploaded, "a redelivery does not re-upload the body either")
}

// Providers post every webhook kind to one endpoint. A delivery that is not
// mail is not an error — saying so lets the caller answer 200 rather than make
// the provider retry something it will never get right.
func TestReceiveWebhook_IgnoresADeliveryThatIsNotMail(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: testMailbox()}, messages, &stubStorage{})

	body := []byte(`{"type":"email.delivered","data":{"email_id":"e_1"}}`)
	result, err := svc.ReceiveWebhook(t.Context(), signed(t, body, time.Unix(1784131200, 0)))
	require.NoError(t, err)

	assert.True(t, result.Ignored)
	assert.Zero(t, messages.writes)
}

// The stored copy is a convenience; the bounded text on the row is what a
// person and the classifier both read. Losing it costs less than making the
// provider retry a message that was otherwise fine.
func TestReceiveWebhook_StillStagesWhenTheBodyCannotBeStored(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	store := &stubStorage{err: errors.New("bucket unreachable")}
	svc := newService(&stubMailboxRepo{mailbox: postmarkMailbox()}, messages, store)

	req := postmarkDelivery(testPostmarkCredentials)
	req.Body = []byte(`{
		"MessageID": "pm-7c1d",
		"From": "dispatch@bigshipper.com",
		"To": "tenders@acme-logistics.com",
		"Subject": "Load 88213 tender",
		"TextBody": "Please confirm pickup Thursday.",
		"HtmlBody": "<p>Please confirm pickup Thursday.</p>"
	}`)

	_, err := svc.ReceiveWebhook(t.Context(), req)
	require.NoError(t, err)

	require.NotNil(t, messages.created)
	assert.Empty(t, messages.created.HTMLKey)
	assert.Equal(t, "Please confirm pickup Thursday.", messages.created.TextBody)
}

func TestReceiveWebhook_RefusesAMalformedBody(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: testMailbox()}, messages, &stubStorage{})

	_, err := svc.ReceiveWebhook(
		t.Context(), signed(t, []byte("not json"), time.Unix(1784131200, 0)),
	)
	require.ErrorIs(t, err, ErrMalformedPayload)
	assert.Zero(t, messages.writes)
}

// The message is what a sender believes was received. A body too long for the
// column is truncated, never dropped — and truncated on a rune boundary,
// because the column is text and half a rune makes the row unreadable rather
// than merely shortened.
func TestBoundedText_CutsOnARuneBoundary(t *testing.T) {
	t.Parallel()

	body := ""
	for len(body) < inboundmessage.MaxBodyBytes+16 {
		body += "é"
	}

	cut := boundedText(body)
	assert.LessOrEqual(t, len(cut), inboundmessage.MaxBodyBytes)
	assert.True(t, utf8ValidString(cut), "a truncated body is still readable text")
}

func utf8ValidString(s string) bool {
	for _, r := range s {
		if r == 0xFFFD {
			return false
		}
	}

	return true
}

const testPostmarkCredentials = "postmark:a-long-random-webhook-password"

func postmarkMailbox() *inboundmessage.Mailbox {
	mailbox := testMailbox()
	mailbox.Provider = inboundmessage.ProviderPostmark
	mailbox.SigningSecret = testPostmarkCredentials

	return mailbox
}

func postmarkBody() []byte {
	return []byte(`{
		"MessageID": "pm-7c1d",
		"From": "dispatch@bigshipper.com",
		"FromName": "Dispatch",
		"To": "tenders@acme-logistics.com",
		"Subject": "Load 88213 tender",
		"TextBody": "Please confirm pickup Thursday."
	}`)
}

func postmarkDelivery(credentials string) *ReceiveWebhookRequest {
	return &ReceiveWebhookRequest{
		MailboxToken:  testToken,
		Body:          postmarkBody(),
		Authorization: "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials)),
		ReceivedAt:    time.Unix(1784131200, 0),
	}
}

/*
Postmark does not sign inbound webhooks, so a Postmark mailbox used to be
checked for a Svix signature Postmark never sends — every delivery was refused.
It is verified by the basic-auth credentials Postmark sends from the webhook
URL instead, which are the protection Postmark offers.
*/
func TestReceiveWebhook_AcceptsAPostmarkDeliveryCarryingTheConfiguredCredentials(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: postmarkMailbox()}, messages, &stubStorage{})

	result, err := svc.ReceiveWebhook(t.Context(), postmarkDelivery(testPostmarkCredentials))
	require.NoError(t, err)
	assert.False(t, result.Duplicate)
	require.NotNil(t, messages.created)
	assert.Equal(t, "pm-7c1d", messages.created.ProviderMessageID)
}

func TestReceiveWebhook_RefusesAPostmarkDeliveryWithTheWrongCredentials(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: postmarkMailbox()}, messages, &stubStorage{})

	_, err := svc.ReceiveWebhook(t.Context(), postmarkDelivery("postmark:guessed"))
	require.ErrorIs(t, err, ErrUnverified)
	assert.Zero(t, messages.writes)
}

// The scheme is the mailbox's, never the request's. A delivery to a Postmark
// mailbox that brings a Svix signature instead of credentials is refused, or
// anyone could pick whichever check they found easier to satisfy.
func TestReceiveWebhook_ChecksTheMailboxesSchemeNotTheOneTheRequestBrings(t *testing.T) {
	t.Parallel()

	messages := &stubMessageRepo{}
	svc := newService(&stubMailboxRepo{mailbox: postmarkMailbox()}, messages, &stubStorage{})

	req := signed(t, postmarkBody(), time.Unix(1784131200, 0))
	_, err := svc.ReceiveWebhook(t.Context(), req)
	require.ErrorIs(t, err, ErrUnverified)
	assert.Zero(t, messages.writes)
}
