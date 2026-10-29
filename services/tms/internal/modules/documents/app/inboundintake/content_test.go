package inboundintake

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubFetcher struct {
	content *inboundmessageports.InboundContent
	err     error
	asked   *inboundmessageports.InboundContentRequest
}

func (f *stubFetcher) Fetch(
	_ context.Context, req *inboundmessageports.InboundContentRequest,
) (*inboundmessageports.InboundContent, error) {
	f.asked = req
	if f.err != nil {
		return nil, f.err
	}

	return f.content, nil
}

const testAPIKey = "re_live_key"

func resendMessage() *inboundmessage.InboundMessage {
	mailbox := inboundqueuetest.TestMailbox()
	mailbox.ProviderAPIKey = inboundqueuetest.SealedPrefix + testAPIKey

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

func fetchedContent() *inboundmessageports.InboundContent {
	return &inboundmessageports.InboundContent{
		HTML:       "<p>Please confirm pickup <b>Thursday</b>.</p>",
		MessageID:  "<ignored@bigshipper.com>",
		InReplyTo:  "<prev@bigshipper.com>",
		References: []string{"<one@bigshipper.com>", "<prev@bigshipper.com>"},
		Attachments: []inboundmessageports.InboundContentAttachment{
			{ProviderID: "att_1", FileName: "tender.pdf", Content: []byte("%PDF-1.7")},
			{ProviderID: "att_2", FileName: "huge.zip", FailureText: "The file is too large."},
			{ProviderID: "att_4", FileName: "done.pdf", Content: []byte("again")},
		},
	}
}

func contentService(
	repo *inboundqueuetest.ContentRepo,
	fetcher *stubFetcher,
	store *stubStorage,
	up *uploads,
) *testService {
	svc, _ := withQueue(&testService{
		l:           zap.NewNop(),
		messageRepo: repo,
		storage:     store,
		encryption:  inboundqueuetest.PlainSecrets{},
	})
	if fetcher != nil {
		svc.fetcher = fetcher
	}
	if up != nil {
		svc.uploads = up
	}

	return svc
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
	repo := &inboundqueuetest.ContentRepo{Message: message}
	fetcher := &stubFetcher{content: fetchedContent()}
	store := &stubStorage{}
	up := &uploads{sessionID: pulid.MustNew("dus_")}
	svc := contentService(repo, fetcher, store, up)

	require.NoError(
		t,
		svc.intake().FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message)),
	)

	assert.True(t, repo.AskedForAttachment)
	require.NotNil(t, fetcher.asked)
	assert.Equal(t, testAPIKey, fetcher.asked.APIKey, "the key is opened, never sent sealed")
	assert.Equal(t, inboundmessage.ProviderResend, fetcher.asked.Provider)
	assert.Equal(t, "e_9f2b", fetcher.asked.ProviderMessageID)

	require.Equal(t, 1, repo.Writes)
	assert.Equal(t, "Please confirm pickup Thursday.", repo.Saved.TextBody,
		"an HTML-only message is read into the text the classifier reads")
	assert.NotEmpty(t, repo.Saved.HTMLKey)
	assert.Equal(t, "<p>Please confirm pickup <b>Thursday</b>.</p>", string(store.body))
	assert.Equal(t, "<abc@bigshipper.com>", repo.Saved.MessageID,
		"the id from the webhook is kept")
	assert.Equal(t, "<prev@bigshipper.com>", repo.Saved.InReplyTo)
	assert.Equal(
		t,
		[]string{"<one@bigshipper.com>", "<prev@bigshipper.com>"},
		repo.Saved.References,
	)

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
	repo := &inboundqueuetest.ContentRepo{Message: message}
	content := fetchedContent()
	content.Text = "Plain text as sent."
	svc := contentService(repo, &stubFetcher{content: content}, &stubStorage{}, nil)

	require.NoError(
		t,
		svc.intake().FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message)),
	)
	assert.Equal(t, "Plain text as sent.", repo.Saved.TextBody)
}

func TestFetchContent_RecordsFilesItCannotStore(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &inboundqueuetest.ContentRepo{Message: message}
	svc := contentService(repo, &stubFetcher{content: fetchedContent()}, &stubStorage{}, nil)

	require.NoError(
		t,
		svc.intake().FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message)),
	)
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
	repo := &inboundqueuetest.ContentRepo{Message: message}
	fetcher := &stubFetcher{content: fetchedContent()}
	svc := contentService(repo, fetcher, &stubStorage{}, nil)

	err := svc.intake().FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message))
	require.ErrorIs(t, err, inboundmessageports.ErrInboundContentRejected)
	assert.Contains(t, err.Error(), "no Resend API key")
	assert.Nil(t, fetcher.asked)
	assert.Zero(t, repo.Writes)
}

func TestFetchContent_RejectsWhenNothingCanFetch(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	svc := contentService(&inboundqueuetest.ContentRepo{Message: message}, nil, &stubStorage{}, nil)

	err := svc.intake().FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message))
	require.ErrorIs(t, err, inboundmessageports.ErrInboundContentRejected)
}

func TestFetchContent_PassesTheProvidersFailureThrough(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &inboundqueuetest.ContentRepo{Message: message}
	cause := fmt.Errorf("%w: Resend answered 503", inboundmessageports.ErrInboundContentUnavailable)
	svc := contentService(repo, &stubFetcher{err: cause}, &stubStorage{}, nil)

	err := svc.intake().FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message))
	require.ErrorIs(t, err, inboundmessageports.ErrInboundContentUnavailable)
	assert.Zero(t, repo.Writes)
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
			repo := &inboundqueuetest.ContentRepo{Message: message}
			fetcher := &stubFetcher{content: fetchedContent()}
			svc := contentService(repo, fetcher, &stubStorage{}, nil)

			require.NoError(
				t,
				svc.intake().
					FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message)),
			)
			assert.Nil(t, fetcher.asked)
			assert.Zero(t, repo.Writes)
		})
	}
}

func TestFetchContent_RefusesAMessageWithNoMailbox(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	message.Mailbox = nil
	svc := contentService(
		&inboundqueuetest.ContentRepo{Message: message},
		&stubFetcher{},
		&stubStorage{},
		nil,
	)

	require.Error(
		t,
		svc.intake().FetchContent(t.Context(), message.ID, inboundqueue.TenantOfMessage(message)),
	)
}

/*
A message whose content could not be read is not left looking as though it is
still being worked on: it goes to a person with the reason, and every file
still waiting says it was never read.
*/
func TestMarkContentUnavailable_SendsTheMessageToReviewWithTheReason(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &inboundqueuetest.ContentRepo{Message: message}
	svc := contentService(repo, nil, &stubStorage{}, nil)

	require.NoError(t, svc.intake().MarkContentUnavailable(
		t.Context(), message.ID, inboundqueue.TenantOfMessage(message), "Resend refused the key.",
	))

	require.NotNil(t, repo.Saved)
	assert.Equal(t, inboundmessage.StatusInReview, repo.Saved.Status)
	assert.Equal(t, ContentUnavailableCode, repo.Saved.FailureCode)
	assert.Equal(t, "Resend refused the key.", repo.Saved.FailureText)
	for _, row := range message.Attachments[:3] {
		assert.Equal(t, "The file could not be read from the provider with the message.",
			row.FailureText)
	}
	assert.Empty(t, message.Attachments[3].FailureText, "a staged file keeps its session")
}

func TestContentFailureText(t *testing.T) {
	t.Parallel()

	rejected := ContentFailureText(
		fmt.Errorf(
			"%w: the mailbox has no Resend API key",
			inboundmessageports.ErrInboundContentRejected,
		),
	)
	assert.Equal(t,
		"The body and attachments of this message could not be read from the provider: "+
			"the mailbox has no Resend API key. Set a full access API key on the mailbox in "+
			"Inbound mailboxes; mail that arrives afterwards is read in full.",
		rejected)

	unavailable := ContentFailureText(
		fmt.Errorf("%w: Resend answered 503", inboundmessageports.ErrInboundContentUnavailable),
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
	repo := &inboundqueuetest.ContentRepo{Message: message}
	svc := contentService(repo, &stubFetcher{}, &stubStorage{}, nil)

	svc.intake().fetchContentInline(t.Context(), message)

	require.NotNil(t, repo.Saved)
	assert.Equal(t, ContentUnavailableCode, repo.Saved.FailureCode)
	assert.Contains(t, repo.Saved.FailureText, "no Resend API key")
}

type blockingFetcher struct {
	release chan struct{}
	asked   chan struct{}
}

func (f *blockingFetcher) Fetch(
	ctx context.Context, _ *inboundmessageports.InboundContentRequest,
) (*inboundmessageports.InboundContent, error) {
	close(f.asked)
	select {
	case <-f.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &inboundmessageports.InboundContent{Text: "Read after the provider was answered."}, nil
}

/*
Without a worker the fetch still happens, but off the request path: downloading
a message's attachments while the provider waits would time the webhook out
and make the provider redeliver mail that already landed. The fetch outlives
the request, so it must not inherit the request's cancellation.
*/
func TestReadContentDetached_AnswersFirstAndOutlivesTheRequest(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	message.Attachments = nil
	repo := &inboundqueuetest.ContentRepo{Message: message}
	fetcher := &blockingFetcher{release: make(chan struct{}), asked: make(chan struct{})}
	svc := contentService(repo, nil, &stubStorage{}, nil)
	svc.fetcher = fetcher

	requestCtx, cancelRequest := context.WithCancel(t.Context())
	svc.intake().readContentDetached(requestCtx, message)
	cancelRequest()

	select {
	case <-fetcher.asked:
	case <-time.After(5 * time.Second):
		t.Fatal("the detached fetch never ran")
	}
	writes, _ := repo.Snapshot()
	assert.Zero(t, writes, "nothing is written before the provider answers")

	close(fetcher.release)
	require.Eventually(t, func() bool {
		writes, _ := repo.Snapshot()
		return writes == 1
	}, 5*time.Second, 10*time.Millisecond)
	_, saved := repo.Snapshot()
	assert.Equal(t, "Read after the provider was answered.", saved.TextBody)
}

// A fetch that ran out of time has a dead context. Writing the failure down on
// that same context would fail too, leaving the message at Received with no
// worker ever coming back to it, so the failure is written on its own.
func TestFetchContentInline_RecordsATimedOutFetchOnAFreshContext(t *testing.T) {
	t.Parallel()

	message := resendMessage()
	repo := &inboundqueuetest.ContentRepo{Message: message}
	cause := fmt.Errorf("%w: Resend could not be reached: %w",
		inboundmessageports.ErrInboundContentUnavailable, context.DeadlineExceeded)
	svc := contentService(repo, &stubFetcher{err: cause}, &stubStorage{}, nil)

	expired, cancel := context.WithCancel(t.Context())
	cancel()
	svc.intake().fetchContentInline(expired, message)

	_, saved := repo.Snapshot()
	require.NotNil(t, saved, "the failure is written even though the fetch's context is gone")
	assert.Equal(t, ContentUnavailableCode, saved.FailureCode)
	assert.Equal(t, inboundmessage.StatusInReview, saved.Status)
}
