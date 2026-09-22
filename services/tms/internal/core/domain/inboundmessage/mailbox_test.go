package inboundmessage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mailbox(policy inboundmessage.ReviewPolicy, confidence float64) *inboundmessage.Mailbox {
	return &inboundmessage.Mailbox{
		Name:          "Tenders",
		Address:       "tenders@carrier.example",
		Provider:      inboundmessage.ProviderPostmark,
		ReviewPolicy:  policy,
		MinConfidence: confidence,
		Status:        inboundmessage.MailboxActive,
	}
}

func validate(m *inboundmessage.Mailbox) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	m.Validate(multiErr)

	return multiErr
}

/*
A mailbox somebody created and did not finish configuring reviews everything.

The alternative is a mailbox that starts acting on its own the moment it is
saved, which is the one default nobody would have chosen and everybody would
find out about afterwards.
*/
func TestMailbox_DefaultsToReviewingEverything(t *testing.T) {
	t.Parallel()

	m := &inboundmessage.Mailbox{}
	m.ApplyDefaults()

	assert.Equal(t, inboundmessage.ReviewAlways, m.ReviewPolicy)
	assert.Equal(t, inboundmessage.MailboxActive, m.Status)
	assert.False(t, m.HandlesWithoutReview(1.0), "a fresh mailbox acts on nothing")
}

/*
Below the floor a confidence bar is not a bar.

A mailbox set to act above 0.1 is an auto-handling mailbox with a number in
front of it, and it would read to whoever configured it as the careful option.
*/
func TestMailbox_RefusesAConfidenceBarThatIsNotOne(t *testing.T) {
	t.Parallel()

	for _, confidence := range []float64{0, 0.1, 0.49, 1.5} {
		multiErr := validate(mailbox(inboundmessage.ReviewBelowConfidence, confidence))

		assert.True(t, multiErr.HasErrors(), "confidence %v was accepted", confidence)
	}
}

func TestMailbox_AcceptsABarItWillActuallyHold(t *testing.T) {
	t.Parallel()

	multiErr := validate(mailbox(inboundmessage.ReviewBelowConfidence, 0.8))

	require.False(t, multiErr.HasErrors(), "%v", multiErr)
}

// The bar is only read by the policy that has one, so a leftover value on
// another policy is not an error to fix.
func TestMailbox_IgnoresTheBarOnAPolicyThatDoesNotReadIt(t *testing.T) {
	t.Parallel()

	multiErr := validate(mailbox(inboundmessage.ReviewAlways, 0))

	assert.False(t, multiErr.HasErrors(), "%v", multiErr)
}

/*
The policy is read in one place, so a mailbox cannot mean one thing to the
workflow that ingests and another to the inbox that lists.
*/
func TestMailbox_HandlesWithoutReviewFollowsThePolicy(t *testing.T) {
	t.Parallel()

	always := mailbox(inboundmessage.ReviewAlways, 0)
	assert.False(t, always.HandlesWithoutReview(0.99))

	auto := mailbox(inboundmessage.ReviewAutoHandle, 0)
	assert.True(t, auto.HandlesWithoutReview(0))

	graded := mailbox(inboundmessage.ReviewBelowConfidence, 0.8)
	assert.True(t, graded.HandlesWithoutReview(0.8), "the bar is inclusive")
	assert.False(t, graded.HandlesWithoutReview(0.79))
}

func TestMailbox_RefusesAnAddressThatIsNotOne(t *testing.T) {
	t.Parallel()

	m := mailbox(inboundmessage.ReviewAlways, 0)
	m.Address = "the tender inbox"

	assert.True(t, validate(m).HasErrors())
}

func TestMailbox_RefusesAProviderNobodyDeliversFrom(t *testing.T) {
	t.Parallel()

	m := mailbox(inboundmessage.ReviewAlways, 0)
	m.Provider = inboundmessage.Provider("Mailgun")

	assert.True(t, validate(m).HasErrors())
}
