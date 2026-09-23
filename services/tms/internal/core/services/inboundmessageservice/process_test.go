package inboundmessageservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type settleRepo struct {
	repositories.InboundMessageRepository

	message *inboundmessage.InboundMessage
	getErr  error
	saved   *inboundmessage.InboundMessage
	writes  int
}

func (s *settleRepo) GetByID(
	_ context.Context, _ repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	return s.message, nil
}

func (s *settleRepo) Update(
	_ context.Context, entity *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	s.writes++
	s.saved = entity

	return entity, nil
}

func staged(
	status inboundmessage.Status,
	policy inboundmessage.ReviewPolicy,
) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:          pulid.MustNew("imsg_"),
		Status:      status,
		FromAddress: "dispatch@bigshipper.com",
		Subject:     "Load 88213 tender",
		TextBody:    "Please confirm pickup.",
		Mailbox: &inboundmessage.Mailbox{
			ID:           pulid.MustNew("imbx_"),
			ReviewPolicy: policy,
			Status:       inboundmessage.MailboxActive,
		},
	}
}

func settler(repo *settleRepo, reply string) *Service {
	return &Service{
		l:           zap.NewNop(),
		messageRepo: repo,
		completion:  &stubCompletion{reply: reply},
	}
}

func TestProcessMessage_SettlesAndStoresTheDecision(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{
		message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle),
	}
	svc := settler(
		repo,
		`{"classification":"Tender","confidence":0.9,"reasoning":"Offers a load."}`,
	)

	result, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, inboundmessage.ClassificationTender, result.Classification)
	assert.Equal(t, inboundmessage.StatusClassified, result.Status)
	assert.True(t, result.Handled)

	require.NotNil(t, repo.saved)
	assert.Equal(t, inboundmessage.ClassificationTender, repo.saved.Classification)
	assert.InDelta(t, 0.9, repo.saved.Confidence, 0.0001)
	assert.NotEmpty(t, repo.saved.ReviewNote, "a settled message says why it settled that way")
}

/*
A workflow retried after a crash must not reclassify a message somebody has
already dealt with.

Temporal retries on its own schedule, and a message a person reviewed an hour
ago would otherwise be read again and have their decision written over by a
model — silently, because the retry looks like ordinary progress.
*/
func TestProcessMessage_LeavesAMessageThatWasAlreadyDealtWith(t *testing.T) {
	t.Parallel()

	for _, status := range []inboundmessage.Status{
		inboundmessage.StatusActioned,
		inboundmessage.StatusIgnored,
		inboundmessage.StatusInReview,
		inboundmessage.StatusClassified,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			repo := &settleRepo{message: staged(status, inboundmessage.ReviewAutoHandle)}
			svc := settler(repo, `{"classification":"Invoice","confidence":1,"reasoning":"x"}`)

			result, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{})
			require.NoError(t, err)

			assert.Equal(t, status, result.Status)
			assert.Zero(t, repo.writes, "a settled message must not be written again")
		})
	}
}

// The mailbox is where the policy lives. Without it there is nothing to decide
// against, and guessing would mean guessing about autonomy.
func TestProcessMessage_RefusesAMessageWithNoMailbox(t *testing.T) {
	t.Parallel()

	message := staged(inboundmessage.StatusReceived, inboundmessage.ReviewAlways)
	message.Mailbox = nil
	repo := &settleRepo{message: message}

	_, err := settler(repo, `{"classification":"Tender","confidence":1,"reasoning":"x"}`).
		ProcessMessage(t.Context(), message.ID, pagination.TenantInfo{})

	require.Error(t, err)
	assert.Zero(t, repo.writes)
}

/*
A message left mid-pipeline is worse than one that failed visibly.

Received reads as "still being worked on", so nobody goes and looks. MarkFailed
moves it to review with the reason attached, which is the state somebody
actually checks.
*/
func TestMarkFailed_PutsAStuckMessageInFrontOfAPerson(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{
		message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle),
	}
	svc := settler(repo, "")

	require.NoError(t, svc.MarkFailed(
		t.Context(), repo.message.ID, pagination.TenantInfo{},
		"SETTLE_FAILED", "the provider never answered",
	))

	require.NotNil(t, repo.saved)
	assert.Equal(t, inboundmessage.StatusInReview, repo.saved.Status)
	assert.Equal(t, "SETTLE_FAILED", repo.saved.FailureCode)
	assert.Contains(t, repo.saved.FailureText, "never answered")
}

// A message already finished with is not dragged back into review by a late
// failure from a retry that was overtaken.
func TestMarkFailed_LeavesATerminalMessageAlone(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{message: staged(inboundmessage.StatusActioned, inboundmessage.ReviewAlways)}
	svc := settler(repo, "")

	require.NoError(t, svc.MarkFailed(
		t.Context(), repo.message.ID, pagination.TenantInfo{}, "X", "y",
	))
	assert.Zero(t, repo.writes)
}

func TestProcessMessage_ReportsALookupFailure(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{getErr: errors.New("database unreachable")}

	_, err := settler(
		repo,
		"",
	).ProcessMessage(t.Context(), pulid.MustNew("imsg_"), pagination.TenantInfo{})
	require.Error(t, err)
}

// An unread message on a trusted mailbox still goes to a person, all the way
// through the pipeline and not just in Settle.
func TestProcessMessage_DoesNotAutoHandleWhatItCouldNotRead(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{
		message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle),
	}
	svc := settler(repo, "not json at all")

	result, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, inboundmessage.StatusInReview, result.Status)
	assert.False(t, result.Handled)
	assert.Equal(t, inboundmessage.ClassificationOther, repo.saved.Classification)
}

// Inside a workflow, a model that could not be asked is asked again rather
// than sending the message to review: the message is left unsettled and the
// caller told why.
func TestProcessMessage_LeavesTheMessageForAnotherAttemptWhenAsked(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{
		message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle),
	}
	unavailable := errors.New("provider answered 503")
	svc := &Service{
		l:           zap.NewNop(),
		messageRepo: repo,
		completion:  &stubCompletion{err: unavailable},
	}

	_, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{},
		RetryModelFailures(func(error) bool { return true }))
	require.ErrorIs(t, err, ErrClassificationUnavailable)
	require.ErrorIs(t, err, unavailable)
	assert.Zero(t, repo.writes, "the message waits for the next attempt, unsettled")
}

// On the last attempt, or for a failure no attempt would change, the message
// goes to review as it always has.
func TestProcessMessage_SendsTheMessageToReviewWhenNotRetrying(t *testing.T) {
	t.Parallel()

	repo := &settleRepo{
		message: staged(inboundmessage.StatusReceived, inboundmessage.ReviewAutoHandle),
	}
	svc := &Service{
		l:           zap.NewNop(),
		messageRepo: repo,
		completion:  &stubCompletion{err: errors.New("provider answered 400")},
	}

	result, err := svc.ProcessMessage(t.Context(), repo.message.ID, pagination.TenantInfo{},
		RetryModelFailures(func(error) bool { return false }))
	require.NoError(t, err)
	assert.Equal(t, inboundmessage.ClassificationOther, result.Classification)
	assert.Equal(t, 1, repo.writes)
}
