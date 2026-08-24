package tender

type Mode string

const (
	ModeWaterfall      = Mode("Waterfall")
	ModeSpotBroadcast  = Mode("SpotBroadcast")
	ModeSpotSequential = Mode("SpotSequential")
)

func (m Mode) String() string {
	return string(m)
}

func (m Mode) IsValid() bool {
	switch m {
	case ModeWaterfall, ModeSpotBroadcast, ModeSpotSequential:
		return true
	}
	return false
}

func (m Mode) IsSequential() bool {
	return m == ModeWaterfall || m == ModeSpotSequential
}

type Status string

const (
	StatusActive      = Status("Active")
	StatusAccepted    = Status("Accepted")
	StatusExhausted   = Status("Exhausted")
	StatusCanceled    = Status("Canceled")
	StatusNeedsReview = Status("NeedsReview")
)

func (s Status) String() string {
	return string(s)
}

func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusAccepted, StatusExhausted, StatusCanceled, StatusNeedsReview:
		return true
	}
	return false
}

// IsTerminal reports whether the tender has fully finished. NeedsReview is
// NOT terminal: the workflow has stopped driving it, but it is still live for
// a dispatcher and still occupies the move's one-live-tender slot, matching
// Tender.IsLive.
func (s Status) IsTerminal() bool {
	return s == StatusAccepted || s == StatusExhausted || s == StatusCanceled
}

type OfferStatus string

const (
	OfferStatusPending        = OfferStatus("Pending")
	OfferStatusSent           = OfferStatus("Sent")
	OfferStatusAccepted       = OfferStatus("Accepted")
	OfferStatusDeclined       = OfferStatus("Declined")
	OfferStatusExpired        = OfferStatus("Expired")
	OfferStatusWithdrawn      = OfferStatus("Withdrawn")
	OfferStatusSuperseded     = OfferStatus("Superseded")
	OfferStatusSkipped        = OfferStatus("Skipped")
	OfferStatusDeliveryFailed = OfferStatus("DeliveryFailed")
)

func (s OfferStatus) String() string {
	return string(s)
}

func (s OfferStatus) IsValid() bool {
	switch s {
	case OfferStatusPending,
		OfferStatusSent,
		OfferStatusAccepted,
		OfferStatusDeclined,
		OfferStatusExpired,
		OfferStatusWithdrawn,
		OfferStatusSuperseded,
		OfferStatusSkipped,
		OfferStatusDeliveryFailed:
		return true
	}
	return false
}

func (s OfferStatus) IsOutstanding() bool {
	return s == OfferStatusPending || s == OfferStatusSent
}

func (s OfferStatus) IsTerminal() bool {
	return !s.IsOutstanding()
}

type Channel string

const (
	ChannelEmail = Channel("Email")
	ChannelEDI   = Channel("EDI")
)

func (c Channel) String() string {
	return string(c)
}

func (c Channel) IsValid() bool {
	switch c {
	case ChannelEmail, ChannelEDI:
		return true
	}
	return false
}

type ResponseAction string

const (
	ResponseActionAccept  = ResponseAction("Accept")
	ResponseActionDecline = ResponseAction("Decline")
)

func (a ResponseAction) String() string {
	return string(a)
}

func (a ResponseAction) IsValid() bool {
	switch a {
	case ResponseActionAccept, ResponseActionDecline:
		return true
	}
	return false
}

type ResponseSource string

const (
	ResponseSourceEmail  = ResponseSource("Email")
	ResponseSourceEDI    = ResponseSource("EDI")
	ResponseSourceManual = ResponseSource("Manual")
)

func (s ResponseSource) String() string {
	return string(s)
}

func (s ResponseSource) IsValid() bool {
	switch s {
	case ResponseSourceEmail, ResponseSourceEDI, ResponseSourceManual:
		return true
	}
	return false
}
