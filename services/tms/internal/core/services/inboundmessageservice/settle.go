package inboundmessageservice

import (
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
)

// Outcome is what a classified message becomes: the status it lands in, and
// whether a desk is allowed to act on it without a person.
type Outcome struct {
	Status inboundmessage.Status
	// Handle is whether an agent may act. It is separate from the status
	// because InReview and Classified are both states a person can see, and the
	// difference that matters is whether anything happens next on its own.
	Handle bool
	// Note is why, in the words the inbox shows. A message a person is being
	// asked to look at should say what it was about the message that stopped it.
	Note string
}

// Settle decides what happens to a message once it has been read.
//
// The mailbox is the only thing that grants autonomy, and Mailbox.HandlesWithoutReview
// is the only place that policy is read — so a message cannot be interpreted one
// way here and another in the inbox.
func Settle(
	mailbox *inboundmessage.Mailbox,
	classification Classification,
) Outcome {
	review := Outcome{Status: inboundmessage.StatusInReview}

	// A message nothing read is a message nobody has read. Whatever the mailbox
	// is trusted to do, it is not trusted to do it blind.
	if !classification.Narrated {
		review.Note = "Nothing could read this message, so it is waiting on a person."

		return review
	}

	// Other is the classifier saying it does not know. Acting on "I am not sure
	// what this is" is the one case where high confidence is worse than low,
	// because it would be confident about being unsure.
	if classification.Class == inboundmessage.ClassificationOther {
		review.Note = "This did not look like any of the kinds the desk handles."

		return review
	}

	if !mailbox.HandlesWithoutReview(classification.Confidence) {
		review.Note = reviewReason(mailbox, classification)

		return review
	}

	return Outcome{
		Status: inboundmessage.StatusClassified,
		Handle: true,
		Note:   "Read as " + string(classification.Class) + " and handled without review.",
	}
}

// reviewReason says which of the two reasons stopped it, because "waiting for
// review" is not actionable and "the mailbox reviews everything" is.
func reviewReason(
	mailbox *inboundmessage.Mailbox,
	classification Classification,
) string {
	if mailbox.ReviewPolicy == inboundmessage.ReviewAlways {
		return "Read as " + string(classification.Class) +
			". This mailbox sends everything to a person."
	}

	return "Read as " + string(classification.Class) +
		", but not confidently enough for this mailbox to act on its own."
}
