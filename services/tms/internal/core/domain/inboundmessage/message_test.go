package inboundmessage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func message() *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		MailboxID:         pulid.MustNew("imbx_"),
		ProviderMessageID: "pm_1",
		FromAddress:       "dispatch@shipper.example",
		Status:            inboundmessage.StatusReceived,
	}
}

func validateMessage(m *inboundmessage.InboundMessage) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	m.Validate(multiErr)

	return multiErr
}

/*
A message is kept whether or not anything could be made of it.

The one thing it may never be missing is where it came from and what the
provider called it: the first is who to reply to, and the second is what makes
a redelivered webhook a no-op rather than a second shipment off one tender.
*/
func TestMessage_RefusesOneItCouldNotTraceBack(t *testing.T) {
	t.Parallel()

	noSender := message()
	noSender.FromAddress = ""
	assert.True(t, validateMessage(noSender).HasErrors())

	noProviderID := message()
	noProviderID.ProviderMessageID = ""
	assert.True(t, validateMessage(noProviderID).HasErrors())

	noMailbox := message()
	noMailbox.MailboxID = pulid.Nil
	assert.True(t, validateMessage(noMailbox).HasErrors())
}

// Nothing classified yet is the ordinary state of a message that just arrived,
// so an absent classification is not an error.
func TestMessage_AcceptsOneNothingHasReadYet(t *testing.T) {
	t.Parallel()

	assert.False(t, validateMessage(message()).HasErrors())
}

func TestMessage_RefusesAClassificationNothingActsOn(t *testing.T) {
	t.Parallel()

	m := message()
	m.Classification = inboundmessage.Classification("Complaint")

	assert.True(t, validateMessage(m).HasErrors())
}

// A confidence outside zero and one is a number the policy would compare
// against and silently always pass or always fail.
func TestMessage_RefusesAConfidenceThatIsNotOne(t *testing.T) {
	t.Parallel()

	for _, confidence := range []float64{-0.1, 1.2} {
		m := message()
		m.Confidence = confidence

		assert.True(t, validateMessage(m).HasErrors(), "confidence %v", confidence)
	}
}

/*
The inbox splits on one question, so it is answered in one place.

Quarantined counts as waiting on a person: a message held back for failing a
signature check or scoring as spam is one somebody has to look at, not one the
system has finished with.
*/
func TestMessage_NeedsReviewIsTheLaneTheInboxSplitsOn(t *testing.T) {
	t.Parallel()

	waiting := map[inboundmessage.Status]bool{
		inboundmessage.StatusReceived:    false,
		inboundmessage.StatusProcessing:  false,
		inboundmessage.StatusClassified:  false,
		inboundmessage.StatusInReview:    true,
		inboundmessage.StatusQuarantined: true,
		inboundmessage.StatusActioned:    false,
		inboundmessage.StatusIgnored:     false,
	}

	for status, expected := range waiting {
		m := message()
		m.Status = status

		assert.Equal(t, expected, m.NeedsReview(), "%s", status)
	}
}

// A sweep has to know what it may leave alone, and the three finished states
// are the ones nothing further happens to.
func TestStatus_TerminalNamesWhatIsFinishedWith(t *testing.T) {
	t.Parallel()

	assert.True(t, inboundmessage.StatusActioned.Terminal())
	assert.True(t, inboundmessage.StatusIgnored.Terminal())
	assert.True(t, inboundmessage.StatusQuarantined.Terminal())

	assert.False(t, inboundmessage.StatusReceived.Terminal())
	assert.False(t, inboundmessage.StatusInReview.Terminal())
}
