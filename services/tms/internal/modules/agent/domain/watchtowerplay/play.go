package watchtowerplay

import "github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"

// Kind names the first move a kind of item has: the thing a dispatcher or a
// biller does about it before anything is decided. It follows from the kind
// alone, so the person working the item is offered it as one button and no
// model is asked what to do.
type Kind string

const (
	// AskForPOD asks whoever delivered a load for its signed proof of
	// delivery.
	AskForPOD = Kind("AskForPod")
	// AskForStatus asks whoever has a load where it is.
	AskForStatus = Kind("AskForStatus")
	// TenderAgain offers a move down its routing guide again.
	TenderAgain = Kind("TenderAgain")
	// None is a kind with no first move that follows from the record: a
	// person decides, or puts an agent on it.
	None = Kind("")
)

func (k Kind) String() string { return string(k) }

func AllKinds() []Kind {
	return []Kind{AskForPOD, AskForStatus, TenderAgain}
}

// Of is the first move for a kind of item. It is the one place the rule
// "this kind of item starts with this move" is written.
func Of(kind watchtower.SourceKind) Kind {
	switch kind { //nolint:exhaustive // every other kind has no play
	case watchtower.SourcePodMissing:
		return AskForPOD
	case watchtower.SourceMissedPickup,
		watchtower.SourceMissedDelivery,
		watchtower.SourceEtaAtRisk,
		watchtower.SourceNoCheckCall:
		return AskForStatus
	case watchtower.SourceTenderExhausted:
		return TenderAgain
	default:
		return None
	}
}

// Channel is how a play reaches whoever it is for.
type Channel string

const (
	// ChannelDash is a message in the driver's portal.
	ChannelDash = Channel("Dash")
	// ChannelText is a text to the driver's phone.
	ChannelText = Channel("Text")
	// ChannelEmail is an email in the organization's letterhead.
	ChannelEmail = Channel("Email")
	// ChannelTender is an offer to carriers, which carries no message.
	ChannelTender = Channel("Tender")
)

func (c Channel) String() string { return string(c) }

// Play is what a play would do for one item, as the person about to send it
// reads it: who it goes to, how, and the words. The words are theirs to
// change; who it goes to was decided from the record and is not.
type Play struct {
	Kind    Kind
	Channel Channel
	// Recipient is who it goes to, by name. An address or a number is never
	// carried here.
	Recipient string
	// Sender is who an email is from.
	Sender  string
	Subject string
	// Message is the words that would be sent. Empty for a play that sends
	// none.
	Message string
	// MessageLimit is the longest message the channel takes; zero when it
	// has no bound of its own.
	MessageLimit int
	// Summary says what happens, for a play that sends no message.
	Summary string
}

// MaxMessageLength bounds a message on a channel with no bound of its own.
const MaxMessageLength = 4000
