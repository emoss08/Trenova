package carrierintel

import "slices"

type Capability uint32

const (
	CapabilityLookupFull Capability = 1 << iota
	CapabilityLookupLite
	CapabilityLookupFMCSA
	CapabilitySearch
	CapabilityAutocomplete
	CapabilityNativeMonitoring
	CapabilitySnapshotMonitoring
	CapabilityEquipmentLookup
	CapabilityNetworkSignals
	CapabilityLanes
	CapabilityBrokerAuthority
	CapabilityInsuranceHistory
	CapabilityRiskScore
)

var capabilityNames = []struct {
	cap  Capability
	name string
}{
	{CapabilityLookupFull, "LookupFull"},
	{CapabilityLookupLite, "LookupLite"},
	{CapabilityLookupFMCSA, "LookupFMCSA"},
	{CapabilitySearch, "Search"},
	{CapabilityAutocomplete, "Autocomplete"},
	{CapabilityNativeMonitoring, "NativeMonitoring"},
	{CapabilitySnapshotMonitoring, "SnapshotMonitoring"},
	{CapabilityEquipmentLookup, "EquipmentLookup"},
	{CapabilityNetworkSignals, "NetworkSignals"},
	{CapabilityLanes, "Lanes"},
	{CapabilityBrokerAuthority, "BrokerAuthority"},
	{CapabilityInsuranceHistory, "InsuranceHistory"},
	{CapabilityRiskScore, "RiskScore"},
}

type CapabilitySet uint32

func NewCapabilitySet(caps ...Capability) CapabilitySet {
	var set CapabilitySet
	for _, c := range caps {
		set |= CapabilitySet(c)
	}
	return set
}

func (s CapabilitySet) Has(c Capability) bool {
	return s&CapabilitySet(c) != 0
}

func (s CapabilitySet) Names() []string {
	names := make([]string, 0, len(capabilityNames))
	for _, entry := range capabilityNames {
		if s.Has(entry.cap) {
			names = append(names, entry.name)
		}
	}
	return names
}

func (s CapabilitySet) SupportsMonitoring() bool {
	return s.Has(CapabilityNativeMonitoring) || s.Has(CapabilitySnapshotMonitoring)
}

func (s CapabilitySet) MonitoringMode() EnrollmentMode {
	if s.Has(CapabilityNativeMonitoring) {
		return EnrollmentModeNative
	}
	return EnrollmentModeSnapshotDiff
}

func (s CapabilitySet) BestDepth(requested LookupDepth) (LookupDepth, bool) {
	order := []struct {
		depth LookupDepth
		cap   Capability
	}{
		{LookupDepthFull, CapabilityLookupFull},
		{LookupDepthLite, CapabilityLookupLite},
		{LookupDepthFMCSA, CapabilityLookupFMCSA},
	}

	for _, entry := range order {
		if entry.depth == requested && s.Has(entry.cap) {
			return entry.depth, true
		}
	}
	for _, entry := range slices.Backward(order) {
		if entry.depth.Rank() >= requested.Rank() && s.Has(entry.cap) {
			return entry.depth, true
		}
	}
	for _, entry := range order {
		if s.Has(entry.cap) {
			return entry.depth, true
		}
	}
	return "", false
}

type ProviderDescriptor struct {
	Capabilities CapabilitySet
	Sections     []Section
}

func (d ProviderDescriptor) CoversSections(required []Section) bool {
	for _, section := range required {
		if !slices.Contains(d.Sections, section) {
			return false
		}
	}
	return true
}
