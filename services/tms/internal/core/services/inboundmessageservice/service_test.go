package inboundmessageservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func resendBody() []byte {
	return []byte(`{
		"type": "email.received",
		"data": {
			"email_id": "e_9f2b",
			"from": "Dispatch <dispatch@bigshipper.com>",
			"to": ["tenders@acme-logistics.com"],
			"cc": ["billing@acme-logistics.com"],
			"subject": "Load 88213 tender",
			"text": "Please confirm pickup Thursday.",
			"html": "<p>Please confirm pickup Thursday.</p>",
			"headers": [
				{"name": "Message-ID", "value": "<abc@bigshipper.com>"},
				{"name": "In-Reply-To", "value": "<prev@bigshipper.com>"}
			],
			"attachments": [
				{"filename": "tender.pdf", "content_type": "application/pdf", "content": "SGVsbG8="}
			]
		}
	}`)
}

// plainSecrets stands in for the key manager. The secret this decrypts is
// already the base64 signing key, so the signature maths is exercised for real
// while the envelope encryption is somebody else's tested concern.
type plainSecrets struct{}

func (plainSecrets) DecryptString(value string) (string, error) { return value, nil }

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
	assert.Equal(t, "dispatch@bigshipper.com", messages.created.FromAddress)
	assert.Equal(t, "Dispatch", messages.created.FromName)
	assert.Equal(t, "Load 88213 tender", messages.created.Subject)
	assert.Equal(t, "<prev@bigshipper.com>", messages.created.InReplyTo)
	assert.Equal(t, inboundmessage.StatusReceived, messages.created.Status)
	require.Len(t, messages.attached, 1)
	assert.Equal(t, "tender.pdf", messages.attached[0].FileName)
	assert.Equal(t, int64(5), messages.attached[0].ByteSize)
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
	svc := newService(&stubMailboxRepo{mailbox: testMailbox()}, messages, store)

	_, err := svc.ReceiveWebhook(t.Context(), signed(t, resendBody(), time.Unix(1784131200, 0)))
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
