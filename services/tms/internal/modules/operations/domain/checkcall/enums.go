package checkcall

type Source string

const (
	SourceDispatcher = Source("Dispatcher")
	SourceCarrier    = Source("Carrier")
	SourceAgent      = Source("Agent")
)

func (s Source) IsValid() bool {
	switch s {
	case SourceDispatcher, SourceCarrier, SourceAgent:
		return true
	default:
		return false
	}
}

type Kind string

const (
	KindUpdate    = Kind("Update")
	KindArrival   = Kind("Arrival")
	KindDeparture = Kind("Departure")
)

func (k Kind) IsValid() bool {
	switch k {
	case KindUpdate, KindArrival, KindDeparture:
		return true
	default:
		return false
	}
}

func (k Kind) IsStopVisit() bool {
	return k == KindArrival || k == KindDeparture
}

const (
	MaxSourceKeyLength     = 255
	MaxCityLength          = 100
	MaxStateCodeLength     = 10
	MaxDelayReasonLength   = 255
	MaxNoteLength          = 1000
	MaxContactNameLength   = 100
	MaxRecipientLength     = 320
	DefaultLinkTTLSeconds  = int64(21 * 24 * 3600)
	MaxReportBackdateHours = int64(7 * 24)
)
