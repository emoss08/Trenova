package integration

// SupportsTelematics reports whether the integration type is a telematics provider the
// platform can actually drive. It is the single source of truth shared by the provider
// factory and the dispatch scoring gate, so the two can never disagree about which
// organizations have a live ELD feed.
func (t Type) SupportsTelematics() bool {
	return t == TypeSamsara
}

// HasEnabledTelematics reports whether any of the organization's integrations is an
// enabled, supported telematics provider. Organizations without one must not have
// hours-of-service weighed against their drivers at all.
func HasEnabledTelematics(records []*Integration) bool {
	for _, record := range records {
		if record == nil {
			continue
		}
		if record.Category != CategoryTelematics || !record.Enabled {
			continue
		}
		if record.Type.SupportsTelematics() {
			return true
		}
	}
	return false
}
