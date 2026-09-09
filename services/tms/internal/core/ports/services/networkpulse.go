package services

// NetworkPulseLane is one origin/destination pair the instance is currently running,
// grouped by state and shipment status. There is no load, customer or city in it.
type NetworkPulseLane struct {
	From string `json:"from"`
	To   string `json:"to"`
	// Status is the raw shipment status ("InTransit", "Assigned", ...). It is left
	// unhumanised so the wording stays with the UI that renders it.
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// NetworkPulse is the instance-wide snapshot the sign-in screen renders. It carries no
// organization, customer, lane detail below state level, or shipment identity — only
// aggregates and the sample they were drawn from — because it is served without a
// session.
type NetworkPulse struct {
	LoadsInMotion int     `json:"loadsInMotion"`
	OnTimePercent float64 `json:"onTimePercent"`
	// SampleSize is the number of completed deliveries the percentage was scored over.
	// Zero means the window held no deliveries, and the percentage means nothing;
	// clients must suppress the figure rather than render 0%.
	SampleSize int                `json:"sampleSize"`
	WindowDays int                `json:"windowDays"`
	Lanes      []NetworkPulseLane `json:"lanes"`
}
