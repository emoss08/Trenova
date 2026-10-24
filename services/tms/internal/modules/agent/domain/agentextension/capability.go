package agentextension

import "slices"

const (
	ToolWebSearch       = "web_search"
	ToolWebRead         = "web_read"
	ToolListPortalSites = "web_list_portal_sites"
	ToolReadPortal      = "web_read_portal"
	ToolActOnPortal     = "web_use_portal"
	ToolAnalyzeData     = "report_analyze_data"
	ToolWatchWeb        = "web_watch"
	ToolListWebWatches  = "web_list_watches"
	ToolStopWebWatch    = "web_stop_watch"
)

// WebWatchTools are the web research tools that watch the web over time,
// which a vendor without scheduled searches leaves out.
var WebWatchTools = []string{ToolWatchWeb, ToolListWebWatches, ToolStopWebWatch}

type Capability string

const (
	CapabilityWebResearch      = Capability("WebResearch")
	CapabilityPortalAutomation = Capability("PortalAutomation")
	CapabilityDataAnalysis     = Capability("DataAnalysis")
)

func AllCapabilities() []Capability {
	return []Capability{
		CapabilityWebResearch,
		CapabilityPortalAutomation,
		CapabilityDataAnalysis,
	}
}

func (c Capability) String() string { return string(c) }

func (c Capability) IsValid() bool {
	_, ok := capabilities[c]
	return ok
}

type CapabilitySpec struct {
	Tools               []string
	OutsideContentTools []string
	Unchecked           string
	RequestNoun         string
}

func (s CapabilitySpec) HasTool(name string) bool {
	return slices.Contains(s.Tools, name)
}

func (s CapabilitySpec) ReturnsOutsideContent(name string) bool {
	return slices.Contains(s.OutsideContentTools, name)
}

var capabilities = map[Capability]CapabilitySpec{
	CapabilityWebResearch: {
		Tools: []string{
			ToolWebSearch, ToolWebRead, ToolWatchWeb, ToolListWebWatches, ToolStopWebWatch,
		},
		OutsideContentTools: []string{ToolWebSearch, ToolWebRead},
		Unchecked:           "the web could not be checked",
		RequestNoun:         "web requests",
	},
	CapabilityPortalAutomation: {
		Tools:               []string{ToolListPortalSites, ToolReadPortal, ToolActOnPortal},
		OutsideContentTools: []string{ToolReadPortal},
		Unchecked:           "the portal could not be checked",
		RequestNoun:         "portal tasks",
	},
	CapabilityDataAnalysis: {
		Tools:       []string{ToolAnalyzeData},
		Unchecked:   "the analysis could not run",
		RequestNoun: "analyses",
	},
}

func CapabilitySpecFor(c Capability) (CapabilitySpec, bool) {
	spec, ok := capabilities[c]
	return spec, ok
}

func CapabilityForTool(name string) (Capability, bool) {
	for _, c := range AllCapabilities() {
		if capabilities[c].HasTool(name) {
			return c, true
		}
	}

	return "", false
}

func IsExtensionTool(name string) bool {
	_, ok := CapabilityForTool(name)
	return ok
}

func ReturnsExternalContent(name string) bool {
	c, ok := CapabilityForTool(name)
	return ok && capabilities[c].ReturnsOutsideContent(name)
}

type ActiveExtension struct {
	Type         Type
	Availability Availability
}

type Active map[Capability]ActiveExtension

func ActiveFrom(records []*Extension) Active {
	byType := make(map[Type]*Extension, len(records))
	for _, record := range records {
		byType[record.Type] = record
	}

	active := make(Active, len(capabilities))
	for _, typ := range AllTypes() {
		spec, ok := SpecFor(typ)
		if !ok {
			continue
		}
		record := byType[typ]
		if !record.Ready(spec) {
			continue
		}
		if _, taken := active[spec.Capability]; taken {
			continue
		}
		active[spec.Capability] = ActiveExtension{Type: typ, Availability: record.Availability}
	}

	return active
}

// Serving is the extension that answers a tool, when one is on for its
// capability and its vendor can serve that tool.
func (a Active) Serving(name string) (ActiveExtension, bool) {
	c, isExtension := CapabilityForTool(name)
	if !isExtension {
		return ActiveExtension{}, false
	}
	served, ok := a[c]
	if !ok {
		return ActiveExtension{}, false
	}
	if spec, known := SpecFor(served.Type); known && spec.Omits(name) {
		return ActiveExtension{}, false
	}

	return served, true
}

func (a Active) Offers(name string) bool {
	if !IsExtensionTool(name) {
		return true
	}
	_, served := a.Serving(name)

	return served
}

func (a Active) Grants() []string {
	granted := make([]string, 0, len(a)*2)
	for _, c := range AllCapabilities() {
		served, ok := a[c]
		if !ok || served.Availability != AvailabilityAllAgents {
			continue
		}
		spec, _ := SpecFor(served.Type)
		granted = append(granted, spec.ToolsOf(c)...)
	}

	return granted
}

func (a Active) GrantsEveryAgent(name string) bool {
	served, ok := a.Serving(name)
	return ok && served.Availability == AvailabilityAllAgents
}
