package briefingfacts

// The pages a briefing line opens. They are constants rather than built
// from a route table because a briefing is written by a background job
// with no request to resolve a route against, and a wrong path here is a
// dead link on the first screen of the day.
const (
	watchtowerPath = "/desk/watchtower"
	decisionsPath  = "/desk/decisions"
	dispatchPath   = "/dispatch/console"
	workersPath    = "/hr/workers"
	billingPath    = "/billing/queue"
	paymentsPath   = "/accounting/customer-payments"
)
