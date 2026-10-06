package inboundmessageservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type contentRepo struct {
	repositories.InboundMessageRepository

	message            *inboundmessage.InboundMessage
	saved              *inboundmessage.InboundMessage
	writes             int
	askedForAttachment bool
}

func (r *contentRepo) GetByID(
	_ context.Context, req repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	r.askedForAttachment = r.askedForAttachment || req.IncludeAttachments

	return r.message, nil
}

func (r *contentRepo) Update(
	_ context.Context, entity *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	r.writes++
	copied := *entity
	copied.Attachments = nil
	copied.Mailbox = nil
	r.saved = &copied

	return &copied, nil
}

func (r *contentRepo) ListAttachments(
	_ context.Context, _ pulid.ID, _ pagination.TenantInfo,
) ([]*inboundmessage.InboundAttachment, error) {
	return r.message.Attachments, nil
}

func (r *contentRepo) UpdateAttachment(
	_ context.Context, entity *inboundmessage.InboundAttachment,
) (*inboundmessage.InboundAttachment, error) {
	return entity, nil
}

type stubFetcher struct {
	content *services.InboundContent
	err     error
	asked   *services.InboundContentRequest
}

func (f *stubFetcher) Fetch(
	_ context.Context, req *services.InboundContentRequest,
) (*services.InboundContent, error) {
	f.asked = req
	if f.err != nil {
		return nil, f.err
	}

	return f.content, nil
}

const testAPIKey = "re_live_key"

func resendMessage() *inboundmessage.InboundMessage {
	mailbox := testMailbox()
	mailbox.ProviderAPIKey = sealedPrefix + testAPIKey

	return &inboundmessage.InboundMessage{
		ID:                pulid.MustNew("imsg_"),
		OrganizationID:    mailbox.OrganizationID,
		BusinessUnitID:    mailbox.BusinessUnitID,
		MailboxID:         mailbox.ID,
		ProviderMessageID: "e_9f2b",
		MessageID:         "<abc@bigshipper.com>",
		FromAddress:       "dispatch@bigshipper.com",
		Subject:           "Load 88213 tender",
		Status:            inboundmessage.StatusReceived,
		Mailbox:           mailbox,
		Attachments: []*inboundmessage.InboundAttachment{
			{ID: pulid.MustNew("imsga_"), FileName: "tender.pdf", ProviderAttachmentID: "att_1"},
			{ID: pulid.MustNew("imsga_"), FileName: "huge.zip", ProviderAttachmentID: "att_2"},
			{ID: pulid.MustNew("imsga_"), FileName: "gone.pdf", ProviderAttachmentID: "att_3"},
			{
				ID:                   pulid.MustNew("imsga_"),
				FileName:             "done.pdf",
				ProviderAttachmentID: "att_4",
				UploadSessionID:      pulid.MustNew("dus_"),
			},
		},
	}
}

func fetchedContent() *services.InboundContent {
	return &services.InboundContent{
		HTML:       "<p>Please confirm pickup <b>Thursday</b>.</p>",
		MessageID:  "<ignored@bigshipper.com>",
		InReplyTo:  "<prev@bigshipper.com>",
		References: []string{"<one@bigshipper.com>", "<prev@bigshipper.com>"},
		Attachments: []services.InboundContentAttachment{
			{ProviderID: "att_1", FileName: "tender.pdf", Content: []byte("%PDF-1.7")},
			{ProviderID: "att_2", FileName: "huge.zip", FailureText: "The file is too large."},
			{ProviderID: "att_4", FileName: "done.pdf", Content: []byte("again")},
		},
	}
}

func contentService(
	repo *contentRepo,
	fetcher *stubFetcher,
	store *stubStorage,
	up *uploads,
) *Service {
	svc := &Service{
		l:           zap.NewNop(),
		messageRepo: repo,
		storage:     store,
		encryption:  plainSecrets{},
	}
	if fetcher != nil {
		svc.fetcher = fetcher
	}
	if up != nil {
		svc.uploads = up
	}

	return svc
}

func tenantOfMessage(message *inboundmessage.InboundMessage) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: message.OrganizationID, BuID: message.BusinessUnitID}
}

/*
Resend's webhook carries no body and no files. Reading them afterwards is the
difference between a tender the desk can act on and a subject line with
nothing under it, so the fetched body, headers and every file land on the
message the webhook staged — each file on its own row, matched by Resend's id.
*/
func TestFetchContent_WritesTheBodyHeadersAndFilesOntoTheMessage(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &contentRepo{message: message}
	fetcher := &stubFetcher{content: fetchedContent()}
	store := &stubStorage{}
	up := &uploads{sessionID: pulid.MustNew("dus_")}
	svc := contentService(repo, fetcher, store, up)

	require.NoError(t, svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message)))

	assert.True(t, repo.askedForAttachment)
	require.NotNil(t, fetcher.asked)
	assert.Equal(t, testAPIKey, fetcher.asked.APIKey, "the key is opened, never sent sealed")
	assert.Equal(t, inboundmessage.ProviderResend, fetcher.asked.Provider)
	assert.Equal(t, "e_9f2b", fetcher.asked.ProviderMessageID)

	require.Equal(t, 1, repo.writes)
	assert.Equal(t, "Please confirm pickup Thursday.", repo.saved.TextBody,
		"an HTML-only message is read into the text the classifier reads")
	assert.NotEmpty(t, repo.saved.HTMLKey)
	assert.Equal(t, "<p>Please confirm pickup <b>Thursday</b>.</p>", string(store.body))
	assert.Equal(t, "<abc@bigshipper.com>", repo.saved.MessageID,
		"the id from the webhook is kept")
	assert.Equal(t, "<prev@bigshipper.com>", repo.saved.InReplyTo)
	assert.Equal(t, []string{"<one@bigshipper.com>", "<prev@bigshipper.com>"}, repo.saved.References)

	tender, huge, gone, done := message.Attachments[0], message.Attachments[1],
		message.Attachments[2], message.Attachments[3]
	assert.Equal(t, up.sessionID, tender.UploadSessionID)
	assert.Equal(t, int64(len("%PDF-1.7")), tender.ByteSize)
	assert.Empty(t, tender.FailureText)
	assert.Equal(t, "The file is too large.", huge.FailureText)
	assert.Equal(t, "The provider did not return this file.", gone.FailureText)
	assert.NotEqual(t, up.sessionID, done.UploadSessionID, "a staged file is not uploaded again")
	assert.Equal(t, 1, up.parts)
}

func TestFetchContent_PrefersTheProvidersOwnText(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	message.Attachments = nil
	repo := &contentRepo{message: message}
	content := fetchedContent()
	content.Text = "Plain text as sent."
	svc := contentService(repo, &stubFetcher{content: content}, &stubStorage{}, nil)

	require.NoError(t, svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message)))
	assert.Equal(t, "Plain text as sent.", repo.saved.TextBody)
}

func TestFetchContent_RecordsFilesItCannotStore(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &contentRepo{message: message}
	svc := contentService(repo, &stubFetcher{content: fetchedContent()}, &stubStorage{}, nil)

	require.NoError(t, svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message)))
	assert.Equal(t,
		"This installation cannot store attachments, so the file was not kept.",
		message.Attachments[0].FailureText)
}

// No number of retries supplies a missing key, so the fetch is refused as a
// rejection — the activity sends the message to review at once instead of
// spending its retries asking.
func TestFetchContent_RejectsAMailboxWithNoAPIKey(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	message.Mailbox.ProviderAPIKey = ""
	repo := &contentRepo{message: message}
	fetcher := &stubFetcher{content: fetchedContent()}
	svc := contentService(repo, fetcher, &stubStorage{}, nil)

	err := svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message))
	require.ErrorIs(t, err, services.ErrInboundContentRejected)
	assert.Contains(t, err.Error(), "no Resend API key")
	assert.Nil(t, fetcher.asked)
	assert.Zero(t, repo.writes)
}

func TestFetchContent_RejectsWhenNothingCanFetch(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	svc := contentService(&contentRepo{message: message}, nil, &stubStorage{}, nil)

	err := svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message))
	require.ErrorIs(t, err, services.ErrInboundContentRejected)
}

func TestFetchContent_PassesTheProvidersFailureThrough(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &contentRepo{message: message}
	cause := fmt.Errorf("%w: Resend answered 503", services.ErrInboundContentUnavailable)
	svc := contentService(repo, &stubFetcher{err: cause}, &stubStorage{}, nil)

	err := svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message))
	require.ErrorIs(t, err, services.ErrInboundContentUnavailable)
	assert.Zero(t, repo.writes)
}

func TestFetchContent_LeavesWhatNeedsNoFetchAlone(t *testing.T) {
	t.Parallel()

	for name, mutate := range map[string]func(*inboundmessage.InboundMessage){
		"postmark sends the whole message": func(m *inboundmessage.InboundMessage) {
			m.Mailbox.Provider = inboundmessage.ProviderPostmark
		},
		"a person already dealt with it": func(m *inboundmessage.InboundMessage) {
			m.Status = inboundmessage.StatusActioned
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			message := resendMessage()
			mutate(message)
			repo := &contentRepo{message: message}
			fetcher := &stubFetcher{content: fetchedContent()}
			svc := contentService(repo, fetcher, &stubStorage{}, nil)

			require.NoError(t, svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message)))
			assert.Nil(t, fetcher.asked)
			assert.Zero(t, repo.writes)
		})
	}
}

func TestFetchContent_RefusesAMessageWithNoMailbox(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	message.Mailbox = nil
	svc := contentService(&contentRepo{message: message}, &stubFetcher{}, &stubStorage{}, nil)

	require.Error(t, svc.FetchContent(t.Context(), message.ID, tenantOfMessage(message)))
}

/*
A message whose content could not be read is not left looking as though it is
still being worked on: it goes to a person with the reason, and every file
still waiting says it was never read.
*/
func TestMarkContentUnavailable_SendsTheMessageToReviewWithTheReason(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &contentRepo{message: message}
	svc := contentService(repo, nil, &stubStorage{}, nil)

	require.NoError(t, svc.MarkContentUnavailable(
		t.Context(), message.ID, tenantOfMessage(message), "Resend refused the key.",
	))

	require.NotNil(t, repo.saved)
	assert.Equal(t, inboundmessage.StatusInReview, repo.saved.Status)
	assert.Equal(t, ContentUnavailableCode, repo.saved.FailureCode)
	assert.Equal(t, "Resend refused the key.", repo.saved.FailureText)
	for _, row := range message.Attachments[:3] {
		assert.Equal(t, "The file could not be read from the provider with the message.",
			row.FailureText)
	}
	assert.Empty(t, message.Attachments[3].FailureText, "a staged file keeps its session")
}

func TestContentFailureText(t *testing.T) {
	t.Parallel()

	rejected := ContentFailureText(
		fmt.Errorf("%w: the mailbox has no Resend API key", services.ErrInboundContentRejected),
	)
	assert.Equal(t,
		"The body and attachments of this message could not be read from the provider: "+
			"the mailbox has no Resend API key. Set a full access API key on the mailbox in "+
			"Inbound mailboxes; mail that arrives afterwards is read in full.",
		rejected)

	unavailable := ContentFailureText(
		fmt.Errorf("%w: Resend answered 503", services.ErrInboundContentUnavailable),
	)
	assert.Equal(t,
		"The body and attachments of this message could not be read from the provider: "+
			"Resend answered 503.",
		unavailable)
}

// With no worker nothing else would ever read the message, so the webhook
// reads it on the request path and writes a failure onto the message rather
// than failing a delivery that landed.
func TestFetchContentInline_RecordsAFailureOnTheMessage(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	message.Mailbox.ProviderAPIKey = ""
	repo := &contentRepo{message: message}
	svc := contentService(repo, &stubFetcher{}, &stubStorage{}, nil)

	svc.fetchContentInline(t.Context(), message)

	require.NotNil(t, repo.saved)
	assert.Equal(t, ContentUnavailableCode, repo.saved.FailureCode)
	assert.Contains(t, repo.saved.FailureText, "no Resend API key")
}
