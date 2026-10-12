package common

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/shared/pulid"
)

// detentionDecisionPreview is an approval or a waiver: the occurrence's
// values before and after, and what the shipment's detention charge comes to
// on either side.
func DetentionDecisionPreview(
	summary string,
	change *detentionservice.OccurrenceChange,
	volatile ...string,
) (*agent.ToolPreview, error) {
	occurrence, err := toolpreview.Changed(
		DetentionRecord(change.Before),
		change.Before,
		change.After,
		toolpreview.Volatile(volatile...),
		toolpreview.WithRefs(map[string]permission.Resource{
			"approvedById": permission.ResourceUser,
			"waivedById":   permission.ResourceUser,
		}),
		toolpreview.Ignore("evidence", "notices", "evidenceHead"),
	)
	if err != nil {
		return nil, err
	}

	toolpreview.AttachMoney(
		occurrence,
		shipmentDetentionMoney(change),
		toolpreview.SensitiveAs("billableAmount"),
	)

	return toolpreview.Build(summary, occurrence), nil
}

// shipmentDetentionMoney is the shipment's detention charge, occurrence by
// occurrence, with the decided one as it stands and as it would stand.
func shipmentDetentionMoney(change *detentionservice.OccurrenceChange) *agent.MoneyPreview {
	occurrences := change.Shipment
	if !containsOccurrence(occurrences, change.Before.ID) {
		occurrences = append([]*detention.DetentionOccurrence{change.Before}, occurrences...)
	}

	lines := make([]agent.MoneyLine, 0, len(occurrences))
	for _, occurrence := range occurrences {
		if occurrence == nil {
			continue
		}
		before, after := occurrence, occurrence
		if occurrence.ID == change.Before.ID {
			before, after = change.Before, change.After
		}
		lines = append(lines, agent.MoneyLine{
			Label:  detentionLineLabel(occurrence),
			Before: KnownAmount(before.ChargedAmount()),
			After:  KnownAmount(after.ChargedAmount()),
		})
	}

	return toolpreview.MoneyBlock(change.Before.Currency, lines...)
}

func containsOccurrence(occurrences []*detention.DetentionOccurrence, id pulid.ID) bool {
	for _, occurrence := range occurrences {
		if occurrence != nil && occurrence.ID == id {
			return true
		}
	}

	return false
}

func DetentionRecord(occurrence *detention.DetentionOccurrence) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceDetentionPolicy,
		ID:       occurrence.ID,
		Label:    "Detention " + DetentionPlace(occurrence),
		Version:  PinnedVersion(occurrence.Version),
	}
}

func DetentionPlace(occurrence *detention.DetentionOccurrence) string {
	parts := make([]string, 0, 2)
	if name := strings.TrimSpace(occurrence.LocationName); name != "" {
		parts = append(parts, "at "+name)
	}
	if pro := strings.TrimSpace(occurrence.ShipmentProNumber); pro != "" {
		parts = append(parts, "on "+pro)
	}
	if len(parts) == 0 {
		return "on its shipment"
	}

	return strings.Join(parts, " ")
}

func detentionLineLabel(occurrence *detention.DetentionOccurrence) string {
	name := strings.TrimSpace(occurrence.LocationName)
	if name == "" {
		name = "Stop"
	}
	if occurrence.StopType == "" {
		return "Detention at " + name
	}

	return fmt.Sprintf("Detention at %s (%s)", name, strings.ToLower(string(occurrence.StopType)))
}
