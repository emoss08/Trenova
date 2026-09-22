package inboundmessageservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mailboxOn(policy inboundmessage.ReviewPolicy, bar float64) *inboundmessage.Mailbox {
	return &inboundmessage.Mailbox{ReviewPolicy: policy, MinConfidence: bar}
}

func read(class inboundmessage.Classification, confidence float64) inboundmessageservice.Classification {
	return inboundmessageservice.Classification{
		Class:      class,
		Confidence: confidence,
		Narrated:   true,
	}
}

func TestSettle_AutoHandlesAConfidentReadOnATrustedMailbox(t *testing.T) {
	t.Parallel()

	got := inboundmessageservice.Settle(
		mailboxOn(inboundmessage.ReviewAutoHandle, 0),
		read(inboundmessage.ClassificationStatusRequest, 0.95),
	)

	assert.Equal(t, inboundmessage.StatusClassified, got.Status)
	assert.True(t, got.Handle)
}

/*
A message nothing read is a message nobody has read.

The fallback classification is what every failure produces — the provider was
down, the reply was junk, no provider is configured at all. Whatever a mailbox
is trusted to do, it is not trusted to do it blind, and AutoHandle is exactly
the setting where getting this wrong is silent.
*/
func TestSettle_NeverActsOnAMessageNothingRead(t *testing.T) {
	t.Parallel()

	unread := inboundmessageservice.Classification{
		Class:      inboundmessage.ClassificationTender,
		Confidence: 0,
		Narrated:   false,
	}

	got := inboundmessageservice.Settle(mailboxOn(inboundmessage.ReviewAutoHandle, 0), unread)

	assert.Equal(t, inboundmessage.StatusInReview, got.Status)
	assert.False(t, got.Handle)
	assert.Contains(t, got.Note, "Nothing could read")
}

/*
Other is the classifier saying it does not know.

This is the one case where high confidence is worse than low: a model that is
0.99 sure the message is Other is confidently telling us it is unsure, and an
AutoHandle mailbox that acted on it would be acting on nothing.
*/
func TestSettle_NeverActsOnOtherHoweverConfident(t *testing.T) {
	t.Parallel()

	got := inboundmessageservice.Settle(
		mailboxOn(inboundmessage.ReviewAutoHandle, 0),
		read(inboundmessage.ClassificationOther, 0.99),
	)

	assert.Equal(t, inboundmessage.StatusInReview, got.Status)
	assert.False(t, got.Handle)
}

func TestSettle_HonoursTheConfidenceBar(t *testing.T) {
	t.Parallel()

	mailbox := mailboxOn(inboundmessage.ReviewBelowConfidence, 0.8)

	above := inboundmessageservice.Settle(mailbox, read(inboundmessage.ClassificationInvoice, 0.85))
	assert.True(t, above.Handle)

	below := inboundmessageservice.Settle(mailbox, read(inboundmessage.ClassificationInvoice, 0.75))
	assert.False(t, below.Handle)
	assert.Equal(t, inboundmessage.StatusInReview, below.Status)

	// Exactly at the bar is at or above it, which is what the domain's own
	// comparison says and the only reading that makes the number a threshold
	// rather than a gap.
	at := inboundmessageservice.Settle(mailbox, read(inboundmessage.ClassificationInvoice, 0.8))
	assert.True(t, at.Handle)
}

func TestSettle_AlwaysReviewSendsEverythingToAPerson(t *testing.T) {
	t.Parallel()

	got := inboundmessageservice.Settle(
		mailboxOn(inboundmessage.ReviewAlways, 0),
		read(inboundmessage.ClassificationTender, 1),
	)

	assert.Equal(t, inboundmessage.StatusInReview, got.Status)
	assert.False(t, got.Handle)
}

// "Waiting for review" is not actionable. Which of the two reasons stopped it
// is: one is a setting somebody chose, the other is the message itself.
func TestSettle_SaysWhichReasonStoppedIt(t *testing.T) {
	t.Parallel()

	always := inboundmessageservice.Settle(
		mailboxOn(inboundmessage.ReviewAlways, 0),
		read(inboundmessage.ClassificationTender, 0.99),
	)
	require.NotEmpty(t, always.Note)
	assert.Contains(t, always.Note, "sends everything to a person")

	unsure := inboundmessageservice.Settle(
		mailboxOn(inboundmessage.ReviewBelowConfidence, 0.9),
		read(inboundmessage.ClassificationTender, 0.5),
	)
	assert.Contains(t, unsure.Note, "not confidently enough")
}

// Whatever it decides, it decides one of the states the inbox splits on.
func TestSettle_OnlyProducesStatusesTheInboxUnderstands(t *testing.T) {
	t.Parallel()

	policies := inboundmessage.AllReviewPolicies()
	for _, policy := range policies {
		for _, class := range inboundmessage.AllClassifications() {
			for _, confidence := range []float64{0, 0.5, 1} {
				got := inboundmessageservice.Settle(
					mailboxOn(policy, 0.7), read(class, confidence),
				)
				assert.Truef(t, got.Status.IsValid(),
					"policy %s, class %s produced %q", policy, class, got.Status)
				assert.NotEmptyf(t, got.Note,
					"policy %s, class %s produced no explanation", policy, class)
			}
		}
	}
}
