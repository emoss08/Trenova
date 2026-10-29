package partyquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func CarrierIntelEventGetSpec() readspec.Get {
	return readspec.Get{
		Name:   "carrier_get_intelligence_event",
		Entity: "carrier intelligence event",
		SearchTerms: []string{
			"carrier risk alert", "authority change", "insurance lapse", "safety rating change",
		},
		Resource: permission.ResourceCarrierIntelligence,
		Kinds:    []permission.RecordKind{permission.KindCarrierIntelligenceEvent},
		Summary: "Retrieve one carrier intelligence event by id: what changed on the " +
			"carrier's authority, insurance or safety record, and its severity. It also " +
			"gives the prior and current values and whether anyone has acknowledged or " +
			"resolved it. Use carrier_list_intelligence_events first when you do not have an id, " +
			"and carrier_get for the carrier itself.",
		ParamName: "eventId",
		IDSource: "from carrier_list_intelligence_events, the page you are on, this run's subject " +
			"or a mentioned record",
	}
}
