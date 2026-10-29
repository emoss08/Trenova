package carrierintelusage

type UsageOutcome string

const (
	UsageOutcomeSuccess     = UsageOutcome("Success")
	UsageOutcomeNotFound    = UsageOutcome("NotFound")
	UsageOutcomeRateLimited = UsageOutcome("RateLimited")
	UsageOutcomeClientError = UsageOutcome("ClientError")
	UsageOutcomeServerError = UsageOutcome("ServerError")
	UsageOutcomeTransport   = UsageOutcome("TransportError")
	UsageOutcomeDenied      = UsageOutcome("Denied")
)

func (o UsageOutcome) String() string { return string(o) }

func (o UsageOutcome) IsValid() bool {
	switch o {
	case UsageOutcomeSuccess, UsageOutcomeNotFound, UsageOutcomeRateLimited,
		UsageOutcomeClientError, UsageOutcomeServerError, UsageOutcomeTransport,
		UsageOutcomeDenied:
		return true
	default:
		return false
	}
}
