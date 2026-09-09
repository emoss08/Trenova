package repositories

import "context"

// NetworkPulseLane is one origin/destination pair the instance is currently running,
// grouped by state and status rather than listed per shipment. Nothing here identifies a
// load, a customer or a city — the sign-in screen it feeds is served without a session.
type NetworkPulseLane struct {
	OriginState      string
	DestinationState string
	Status           string
	Count            int
}

// NetworkPulseCounts is the raw shape behind the sign-in screen's panel. It spans every
// organization and business unit on the instance — the login page has no tenant to scope
// to — and carries the on-time numerator and denominator rather than a percentage so the
// caller can tell "no deliveries in the window" from "nothing arrived on time".
type NetworkPulseCounts struct {
	LoadsInMotion int
	OnTimeCount   int
	OnTimeTotal   int
	Lanes         []NetworkPulseLane
}

type NetworkPulseRepository interface {
	// GetNetworkPulse counts loads currently in transit, scores completed delivery stops
	// whose actual arrival falls at or after `since`, and returns at most `laneLimit`
	// active lanes ordered by how many loads are on them.
	GetNetworkPulse(ctx context.Context, since int64, laneLimit int) (*NetworkPulseCounts, error)
}
