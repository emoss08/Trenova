package agentextension

const (
	ToolWebSearch = "web_search"
	ToolWebRead   = "web_read"
)

type Capability string

const (
	CapabilityWebResearch = Capability("WebResearch")
)

func AllCapabilities() []Capability {
	return []Capability{CapabilityWebResearch}
}

func (c Capability) String() string { return string(c) }

func (c Capability) IsValid() bool {
	_, ok := capabilities[c]
	return ok
}

type CapabilitySpec struct {
	Tools                  []string
	ReturnsExternalContent bool
	Unchecked              string
	RequestNoun            string
}

func (s CapabilitySpec) HasTool(name string) bool {
	for _, tool := range s.Tools {
		if tool == name {
			return true
		}
	}

	return false
}

var capabilities = map[Capability]CapabilitySpec{
	CapabilityWebResearch: {
		Tools:                  []string{ToolWebSearch, ToolWebRead},
		ReturnsExternalContent: true,
		Unchecked:              "the web could not be checked",
		RequestNoun:            "web requests",
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
	return ok && capabilities[c].ReturnsExternalContent
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
		if _, taken := active[spec.Capability]; taken {
			continue
		}
		record := byType[typ]
		if !record.Ready(spec) {
			continue
		}
		active[spec.Capability] = ActiveExtension{Type: typ, Availability: record.Availability}
	}

	return active
}

func (a Active) Serving(name string) (ActiveExtension, bool) {
	c, isExtension := CapabilityForTool(name)
	if !isExtension {
		return ActiveExtension{}, false
	}
	served, ok := a[c]

	return served, ok
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
		granted = append(granted, capabilities[c].Tools...)
	}

	return granted
}

func (a Active) GrantsEveryAgent(name string) bool {
	served, ok := a.Serving(name)
	return ok && served.Availability == AvailabilityAllAgents
}
