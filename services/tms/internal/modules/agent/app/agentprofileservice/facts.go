package agentprofileservice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

// Facts are said in rounded words on purpose. A count to two figures and a
// share to the nearest five percent stay put while the work barely moves, so
// the digest, and with it the write-up and the cached prompt, changes only
// when the organization's work does.
const (
	significantFigures = 2
	shareStep          = 5
	countStep          = 5
	exactBelow         = 10
	stepBelow          = 100
)

// OrganizationInputs is everything an organization's facts are made from.
type OrganizationInputs struct {
	Organization    *tenant.Organization
	Activity        *agentprofileports.OrganizationActivity
	DispatchControl *dispatchcontrol.DispatchControl
	BillingControl  *billingcontrol.BillingControl
	ShipmentControl *shipmentcontrol.ShipmentControl
	WindowDays      int
}

// OrganizationFacts says what the organization's records show, one line per
// thing measured, leaving out what there is nothing to say about.
func OrganizationFacts(in *OrganizationInputs) agentprofile.Facts {
	facts := make(agentprofile.Facts, 0, 16)
	add := func(key, text string) {
		if text = strings.TrimSpace(text); text != "" {
			facts = append(facts, agentprofile.Fact{Key: key, Text: text})
		}
	}

	if org := in.Organization; org != nil {
		add("organization.operations", operationsLine(org))
		add("organization.home", homeLine(org))
	}

	if a := in.Activity; a != nil {
		days := in.WindowDays
		if days <= 0 {
			days = agentprofileports.OrganizationWindowDays
		}
		add("shipments.volume", volumeLine(a.Volume, days))
		add("shipments.coverage", coverageLine(a.Volume, a.CarriersUsed))
		add("customers.top", topCustomersLine(a.TopCustomers, a.Volume))
		add("lanes.top", topLanesLine(a.TopLanes, a.Volume))
		add("locations.pickups", namesLine("Most frequent pickup locations", a.TopPickupLocations))
		add("fleet.tractors", fleetLine("tractor", a.Tractors))
		add("fleet.trailers", fleetLine("trailer", a.Trailers))
		add("workforce.drivers", workforceLine(a.Workers))
		add("integrations.enabled", integrationsLine(a.Integrations))
		add("integrations.accounting", accountingLine(a.AccountingConnections))
		add("integrations.edi", ediLine(a.EDIPartners))
	}

	add("settings.dispatch", dispatchLine(in.DispatchControl))
	add("settings.billing", billingLine(in.BillingControl))
	add("settings.shipments", shipmentControlLine(in.ShipmentControl))

	return facts.Normalized()
}

// UserFacts says what a person's records show they work on. A person whose
// records show nothing gets no facts, and so no profile.
func UserFacts(a *agentprofileports.UserActivity, windowDays int) agentprofile.Facts {
	facts := make(agentprofile.Facts, 0, 10)
	if a == nil {
		return facts
	}
	if windowDays <= 0 {
		windowDays = agentprofileports.UserWindowDays
	}
	add := func(key, text string) {
		if text = strings.TrimSpace(text); text != "" {
			facts = append(facts, agentprofile.Fact{Key: key, Text: text})
		}
	}

	if title := strings.TrimSpace(a.JobTitle); title != "" {
		add("person.title", "Job title: "+title+".")
	}
	if a.Driver {
		add("person.driver", "Is set up as a driver in this organization.")
	}
	add("work.shipments.owned", ownedShipmentsLine(a.OwnedShipments, windowDays))
	if a.EnteredShipments > 0 {
		add("work.shipments.entered", fmt.Sprintf("Entered %s shipments in the last %d days.",
			about(a.EnteredShipments), windowDays))
	}
	add("work.customers", userCustomersLine(a.Customers))
	add("work.records", changedRecordsLine(a.ChangedRecords, windowDays))
	add(
		"work.conversations",
		conversationsLine(a.ConversationCount, a.ConversationAbout, windowDays),
	)
	add("work.tools", toolsLine(a.ToolsUsed))
	if len(a.FavoritePages) > 0 {
		add("work.favorites", "Pages pinned as favorites: "+strings.Join(a.FavoritePages, ", ")+".")
	}

	return facts.Normalized()
}

// HasActivity reports whether a person's facts say anything about their work
// beyond who they are.
func HasActivity(facts agentprofile.Facts) bool {
	for _, fact := range facts {
		if strings.HasPrefix(fact.Key, "work.") {
			return true
		}
	}

	return false
}

func operationsLine(org *tenant.Organization) string {
	switch {
	case org.AssetOperationsEnabled && org.BrokerageEnabled:
		return "Runs its own fleet and also brokers freight to outside carriers."
	case org.AssetOperationsEnabled:
		return "Runs its own fleet; brokerage is not turned on."
	case org.BrokerageEnabled:
		return "Brokers freight to outside carriers; it does not run a fleet of its own."
	default:
		return ""
	}
}

func homeLine(org *tenant.Organization) string {
	parts := make([]string, 0, 2)
	if city := strings.TrimSpace(org.City); city != "" {
		if org.State != nil && strings.TrimSpace(org.State.Abbreviation) != "" {
			city += ", " + strings.TrimSpace(org.State.Abbreviation)
		}
		parts = append(parts, "Based in "+city)
	}
	if tz := strings.TrimSpace(org.Timezone); tz != "" {
		parts = append(parts, "keeps time in "+tz)
	}
	if len(parts) == 0 {
		return ""
	}

	return capitalize(strings.Join(parts, "; ")) + "."
}

func volumeLine(v agentprofileports.ShipmentVolume, days int) string {
	if v.Total == 0 {
		return fmt.Sprintf("No shipments were entered in the last %d days.", days)
	}

	line := fmt.Sprintf("Entered %s shipments in the last %d days", about(v.Total), days)
	if weeks := int64(days / 7); weeks > 0 && v.Total >= weeks {
		line += fmt.Sprintf(" (%s a week)", about(v.Total/weeks))
	}
	line += "."
	if share := shareText(v.Canceled, v.Total); share != "" {
		line += " " + capitalize(share) + " were canceled."
	}
	if v.Customers > 0 {
		line += fmt.Sprintf(" %s customers shipped.", capitalize(about(v.Customers)))
	}

	return line
}

func coverageLine(v agentprofileports.ShipmentVolume, carriers int64) string {
	if v.Moves == 0 || v.Brokered == 0 {
		return ""
	}

	line := fmt.Sprintf("%s of moves were covered by outside carriers",
		capitalize(shareText(v.Brokered, v.Moves)))
	if carriers > 0 {
		line += fmt.Sprintf(", using %s carriers", about(carriers))
	}

	return line + "."
}

func topCustomersLine(
	rows []agentprofileports.NamedCount,
	v agentprofileports.ShipmentVolume,
) string {
	shipped := v.Total - v.Canceled
	if len(rows) == 0 || shipped <= 0 {
		return ""
	}

	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		parts = append(parts, name+" ("+shareText(row.Count, shipped)+")")
	}
	if len(parts) == 0 {
		return ""
	}

	return "Busiest customers by shipments: " + strings.Join(parts, ", ") + "."
}

func topLanesLine(rows []agentprofileports.LaneCount, v agentprofileports.ShipmentVolume) string {
	shipped := v.Total - v.Canceled
	if len(rows) == 0 || shipped <= 0 {
		return ""
	}

	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		origin, destination := strings.TrimSpace(row.Origin), strings.TrimSpace(row.Destination)
		if origin == "" || destination == "" {
			continue
		}
		parts = append(parts, origin+" to "+destination+" ("+shareText(row.Count, shipped)+")")
	}
	if len(parts) == 0 {
		return ""
	}

	return "Busiest lanes, state to state: " + strings.Join(parts, ", ") + "."
}

func namesLine(label string, rows []agentprofileports.NamedCount) string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if name := strings.TrimSpace(row.Name); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}

	return label + ": " + strings.Join(names, "; ") + "."
}

func fleetLine(kind string, f agentprofileports.FleetCounts) string {
	if f.Active == 0 {
		return ""
	}

	line := fmt.Sprintf("%s %ss", capitalize(about(f.Active)), kind)
	if f.OutOfService > 0 {
		line += fmt.Sprintf(", %s of them out of service or in the shop", about(f.OutOfService))
	}
	if types := typeShares(f.ByType, f.Active); types != "" {
		line += "; mostly " + types
	}

	return line + "."
}

func workforceLine(w agentprofileports.WorkforceCounts) string {
	if w.Active == 0 {
		return ""
	}

	line := fmt.Sprintf("%s active drivers", capitalize(about(w.Active)))
	if types := typeShares(w.ByDriverType, w.Active); types != "" {
		line += " (" + types + ")"
	}
	if w.Contractors > 0 {
		line += fmt.Sprintf("; %s are contractors", shareText(w.Contractors, w.Active))
	}

	return line + "."
}

func typeShares(rows []agentprofileports.NamedCount, total int64) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" || row.Count <= 0 {
			continue
		}
		parts = append(parts, name+" "+shareText(row.Count, total))
	}

	return strings.Join(parts, ", ")
}

func integrationsLine(types []string) string {
	names := make([]string, 0, len(types))
	for _, kind := range types {
		if kind = strings.TrimSpace(kind); kind != "" {
			names = append(names, kind)
		}
	}
	if len(names) == 0 {
		return ""
	}

	return "Integrations turned on: " + strings.Join(names, ", ") + "."
}

func accountingLine(rows []agentprofileports.NamedStatus) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			continue
		}
		part := name
		if status := strings.TrimSpace(row.Status); status != "" && status != "Connected" {
			part += " (" + strings.ToLower(status) + ")"
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return ""
	}

	return "Accounting system: " + strings.Join(parts, ", ") + "."
}

func ediLine(partners int64) string {
	switch {
	case partners <= 0:
		return ""
	case partners == 1:
		return "Exchanges EDI with one trading partner."
	default:
		return fmt.Sprintf("Exchanges EDI with %s trading partners.", about(partners))
	}
}

func dispatchLine(dc *dispatchcontrol.DispatchControl) string {
	if dc == nil {
		return ""
	}

	parts := []string{"planning mode " + string(dc.ResolvedPlanningMode())}
	if dc.EnableAutoAssignment {
		parts = append(parts, "automatic assignment on")
	}
	if dc.EnforceHOSCompliance {
		parts = append(parts, "hours of service enforced")
	}
	if dc.EnableCheckCallMonitoring {
		parts = append(parts, "check-call monitoring on")
	}
	if level := strings.TrimSpace(string(dc.ComplianceEnforcementLevel)); level != "" {
		parts = append(parts, "compliance problems are a "+strings.ToLower(level))
	}

	return "Dispatch settings: " + strings.Join(parts, "; ") + "."
}

func billingLine(bc *billingcontrol.BillingControl) string {
	if bc == nil {
		return ""
	}

	return fmt.Sprintf(
		"Billing settings: moving ready freight to billing is %s; invoice drafts are %s; posting is %s.",
		splitWords(string(bc.BillingQueueTransferMode)),
		splitWords(string(bc.InvoiceDraftCreationMode)),
		splitWords(string(bc.InvoicePostingMode)),
	)
}

func shipmentControlLine(sc *shipmentcontrol.ShipmentControl) string {
	if sc == nil {
		return ""
	}

	parts := make([]string, 0, 3)
	if sc.TrackDetentionTime {
		parts = append(parts, "detention time is tracked")
	}
	if sc.AutoDelayShipments {
		parts = append(parts, "late shipments are marked delayed automatically")
	}
	if sc.CheckForDuplicateBOLs {
		parts = append(parts, "duplicate BOLs are checked")
	}
	if len(parts) == 0 {
		return ""
	}

	return "Shipment settings: " + strings.Join(parts, "; ") + "."
}

func ownedShipmentsLine(rows []agentprofileports.NamedCount, days int) string {
	var total int64
	for _, row := range rows {
		total += row.Count
	}
	if total == 0 {
		return ""
	}

	line := fmt.Sprintf("Owns %s shipments entered in the last %d days", about(total), days)
	if statuses := typeShares(rows, total); statuses != "" {
		line += " (" + statuses + ")"
	}

	return line + "."
}

func userCustomersLine(rows []agentprofileports.NamedCount) string {
	var total int64
	for _, row := range rows {
		total += row.Count
	}
	if total == 0 {
		return ""
	}

	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if name := strings.TrimSpace(row.Name); name != "" {
			names = append(names, name)
		}
	}

	return "Works most with these customers: " + strings.Join(names, ", ") + "."
}

func changedRecordsLine(rows []agentprofileports.NamedCount, days int) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" || row.Count <= 0 {
			continue
		}
		parts = append(parts, splitWords(name)+" ("+about(row.Count)+")")
	}
	if len(parts) == 0 {
		return ""
	}

	return fmt.Sprintf("Changes these kinds of records most, over the last %d days: %s.",
		days, strings.Join(parts, ", "))
}

func conversationsLine(count int64, subjects []agentprofileports.NamedCount, days int) string {
	if count == 0 {
		return ""
	}

	line := fmt.Sprintf("Had %s conversations with agents in the last %d days", about(count), days)
	kinds := make([]string, 0, len(subjects))
	for _, row := range subjects {
		if name := strings.TrimSpace(row.Name); name != "" {
			kinds = append(kinds, splitWords(name))
		}
	}
	if len(kinds) > 0 {
		line += ", mostly about a " + strings.Join(kinds, ", ")
	}

	return line + "."
}

func toolsLine(rows []agentprofileports.NamedCount) string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if name := strings.TrimSpace(row.Name); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}

	return "The tools agents most often run for them: " + strings.Join(names, ", ") + "."
}

// about says a count in rounded words: "7", "about 45", "about 1,200". Under
// ten a count is exact; under a hundred it is said to the nearest five; above
// that, to two figures.
func about(n int64) string {
	rounded := roundCount(n)
	text := intutils.FormatWithCommas(rounded)
	if rounded != n {
		return "about " + text
	}

	return text
}

func roundCount(n int64) int64 {
	switch {
	case n < exactBelow:
		return n
	case n < stepBelow:
		return intutils.RoundToStep(n, countStep)
	default:
		return intutils.RoundSignificant(n, significantFigures)
	}
}

// shareText says part of whole as a rounded percentage, "under 5%" when it is
// small but not nothing.
func shareText(part, whole int64) string {
	if part <= 0 || whole <= 0 {
		return ""
	}
	share := intutils.RoundedShare(part, whole, shareStep)
	if share == 0 {
		return "under " + strconv.Itoa(shareStep) + "%"
	}

	return "about " + strconv.Itoa(share) + "%"
}

func capitalize(text string) string {
	return stringutils.CapitalizeFirst(text)
}

// splitWords reads a camel-cased enum or resource name as words:
// "ManualReviewRequired" is "manual review required".
func splitWords(name string) string {
	return strings.ReplaceAll(stringutils.ConvertCamelToSnake(name), "_", " ")
}
