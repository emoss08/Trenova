package inboundprocessing

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueuetest"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func settler(repo *inboundqueuetest.SettleRepo, reply string) *testService {
	svc, _ := withQueue(&testService{
		l:           zap.NewNop(),
		messageRepo: repo,
		completion:  &stubCompletion{reply: reply},
	})

	return svc
}

func TestProcessMessage_SettlesAndStoresTheDecision(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusReceived,
			inboundmessage.ReviewAutoHandle,
		),
	}
	svc := settler(
		repo,
		`{"classification":"Tender","confidence":0.9,"reasoning":"Offers a load."}`,
	)

	result, err := svc.processing().
		ProcessMessage(t.Context(), repo.Message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, inboundmessage.ClassificationTender, result.Classification)
	assert.Equal(t, inboundmessage.StatusClassified, result.Status)
	assert.True(t, result.Handled)

	require.NotNil(t, repo.Saved)
	assert.Equal(t, inboundmessage.ClassificationTender, repo.Saved.Classification)
	assert.InDelta(t, 0.9, repo.Saved.Confidence, 0.0001)
	assert.NotEmpty(t, repo.Saved.ReviewNote, "a settled message says why it settled that way")
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

			repo := &inboundqueuetest.SettleRepo{
				Message: inboundqueuetest.Staged(status, inboundmessage.ReviewAutoHandle),
			}
			svc := settler(repo, `{"classification":"Invoice","confidence":1,"reasoning":"x"}`)

			result, err := svc.processing().
				ProcessMessage(t.Context(), repo.Message.ID, pagination.TenantInfo{})
			require.NoError(t, err)

			assert.Equal(t, status, result.Status)
			assert.Zero(t, repo.Writes, "a settled message must not be written again")
		})
	}
}

// The mailbox is where the policy lives. Without it there is nothing to decide
// against, and guessing would mean guessing about autonomy.
func TestProcessMessage_RefusesAMessageWithNoMailbox(t *testing.T) {
	t.Parallel()

	message := inboundqueuetest.Staged(inboundmessage.StatusReceived, inboundmessage.ReviewAlways)
	message.Mailbox = nil
	repo := &inboundqueuetest.SettleRepo{Message: message}

	_, err := settler(repo, `{"classification":"Tender","confidence":1,"reasoning":"x"}`).
		processing().ProcessMessage(t.Context(), message.ID, pagination.TenantInfo{})

	require.Error(t, err)
	assert.Zero(t, repo.Writes)
}

/*
A message left mid-pipeline is worse than one that failed visibly.

Received reads as "still being worked on", so nobody goes and looks. MarkFailed
moves it to review with the reason attached, which is the state somebody
actually checks.
*/
func TestMarkFailed_PutsAStuckMessageInFrontOfAPerson(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusReceived,
			inboundmessage.ReviewAutoHandle,
		),
	}
	svc := settler(repo, "")

	require.NoError(t, svc.processing().MarkFailed(
		t.Context(), repo.Message.ID, pagination.TenantInfo{},
		"SETTLE_FAILED", "the provider never answered",
	))

	require.NotNil(t, repo.Saved)
	assert.Equal(t, inboundmessage.StatusInReview, repo.Saved.Status)
	assert.Equal(t, "SETTLE_FAILED", repo.Saved.FailureCode)
	assert.Contains(t, repo.Saved.FailureText, "never answered")
}

// A message already finished with is not dragged back into review by a late
// failure from a retry that was overtaken.
func TestMarkFailed_LeavesATerminalMessageAlone(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusActioned,
			inboundmessage.ReviewAlways,
		),
	}
	svc := settler(repo, "")

	require.NoError(t, svc.processing().MarkFailed(
		t.Context(), repo.Message.ID, pagination.TenantInfo{}, "X", "y",
	))
	assert.Zero(t, repo.Writes)
}

func TestProcessMessage_ReportsALookupFailure(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{GetErr: errors.New("database unreachable")}

	_, err := settler(
		repo,
		"",
	).processing().ProcessMessage(t.Context(), pulid.MustNew("imsg_"), pagination.TenantInfo{})
	require.Error(t, err)
}

// An unread message on a trusted mailbox still goes to a person, all the way
// through the pipeline and not just in Settle.
func TestProcessMessage_DoesNotAutoHandleWhatItCouldNotRead(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusReceived,
			inboundmessage.ReviewAutoHandle,
		),
	}
	svc := settler(repo, "not json at all")

	result, err := svc.processing().
		ProcessMessage(t.Context(), repo.Message.ID, pagination.TenantInfo{})
	require.NoError(t, err)

	assert.Equal(t, inboundmessage.StatusInReview, result.Status)
	assert.False(t, result.Handled)
	assert.Equal(t, inboundmessage.ClassificationOther, repo.Saved.Classification)
}

// Inside a workflow, a model that could not be asked is asked again rather
// than sending the message to review: the message is left unsettled and the
// caller told why.
func TestProcessMessage_LeavesTheMessageForAnotherAttemptWhenAsked(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusReceived,
			inboundmessage.ReviewAutoHandle,
		),
	}
	unavailable := errors.New("provider answered 503")
	svc, _ := withQueue(&testService{
		l:           zap.NewNop(),
		messageRepo: repo,
		completion:  &stubCompletion{err: unavailable},
	})

	_, err := svc.processing().ProcessMessage(t.Context(), repo.Message.ID, pagination.TenantInfo{},
		RetryModelFailures(func(error) bool { return true }))
	require.ErrorIs(t, err, ErrClassificationUnavailable)
	require.ErrorIs(t, err, unavailable)
	assert.Zero(t, repo.Writes, "the message waits for the next attempt, unsettled")
}

// On the last attempt, or for a failure no attempt would change, the message
// goes to review as it always has.
func TestProcessMessage_SendsTheMessageToReviewWhenNotRetrying(t *testing.T) {
	t.Parallel()

	repo := &inboundqueuetest.SettleRepo{
		Message: inboundqueuetest.Staged(
			inboundmessage.StatusReceived,
			inboundmessage.ReviewAutoHandle,
		),
	}
	svc, _ := withQueue(&testService{
		l:           zap.NewNop(),
		messageRepo: repo,
		completion:  &stubCompletion{err: errors.New("provider answered 400")},
	})

	result, err := svc.processing().
		ProcessMessage(t.Context(), repo.Message.ID, pagination.TenantInfo{},
			RetryModelFailures(func(error) bool { return false }))
	require.NoError(t, err)
	assert.Equal(t, inboundmessage.ClassificationOther, result.Classification)
	assert.Equal(t, 1, repo.Writes)
}
