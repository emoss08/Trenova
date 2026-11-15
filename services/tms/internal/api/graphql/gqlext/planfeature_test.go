package gqlext

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/identity/app/fieldsensitivity"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

// partOfCarrier lists the object types with no kind of their own that a
// record of a kind carries under a name that does not share the carrier's
// leading word, each with the reason it is part of every record that carries
// it. The field guard gates such a type as its carrier's kind, so on a plan
// without its carrier's feature it is withheld with it, and on a plan without
// some other feature it is shown with it: a type that is really another
// feature's data must declare that kind with @kind(resource:) instead of
// being listed here.
var partOfCarrier = map[string]string{
	"CarrierAssignment": "who hauls a move and at what cost: base dispatch, read with the move " +
		"that carries it",
	"CarrierAssignmentAccessorial": "an accessorial charge of the move's carrier assignment",
	"ChargeAllocation": "how the shipment's or order's own charges are split between its " +
		"payers",
	"CostCategory": "a cost category of the costing control that carries it, its own " +
		"configuration",
	"CostCategoryGLAccountLink": "the code and name of the general ledger account a cost " +
		"category posts to, Accounting & Cash like the costing control that carries it",
	"CustomerTrackingProfile": "withheld by the field feature on Customer.trackingProfile: " +
		"what the customer's tracking links show, Visibility & Tracking's",
	"DeliveryHourBucket": "an hour of the shipment watchlist's deliveries, an aggregate of the " +
		"organization's shipments",
	"DetentionAccrual": "withheld by the field feature on ShipmentWatchlist.detention: a stop " +
		"accruing detention, Detention's",
	"DriverSettlementLine": "a line of the driver settlement the dispute is about, the " +
		"dispute's own family",
	"FuelIndex": "the published fuel index a costing control or fuel surcharge program " +
		"prices from: reference data with no feature of its own",
	"GeofenceVertex": "a point of the location's own geofence",
	"IFTAJurisdiction": "the jurisdiction a fuel purchase was made in, Fuel & IFTA " +
		"reference data like the purchase",
	"JournalCardLine":    "a line of the manual journal its card shows",
	"JournalCardPosting": "a posting of the manual journal its card shows",
	"JournalEntryLineAccount": "the code and name of the general ledger account a factoring " +
		"agreement or a journal line posts to, Accounting & Cash like every record that " +
		"carries it",
	"LateDelivery": "a late delivery on the shipment watchlist, a shipment's own PRO, city and " +
		"customer name",
	"LocationReference": "where the tractor's or trailer's last completed move left it: base " +
		"equipment continuity, not telematics, so part of the equipment record",
	"NextUncoveredPickup": "the next uncovered pickup on the shipment watchlist, a shipment's own " +
		"window and cities",
	"PayRateOverride":     "a rate override of the pay assignment that carries it",
	"PendingDecisionHold": "why the proposal that carries it waits on a decision",
	"ReadyToBillCustomer": "a customer's count of ready-to-bill shipments on the shipment " +
		"watchlist, an aggregate of shipments",
	"ReviewGoal":   "a goal of the performance review that carries it",
	"ReviewItem":   "an item of the review template that carries it",
	"ReviewRating": "a rating of the performance review that carries it",
	"SettlementException": "an exception raised settling the driver settlement that " +
		"carries it",
	"SettlementLoadShare": "one load's share of the carrier or driver settlement that " +
		"carries it",
	"StopEta": "withheld by the field feature on Shipment.eta: a stop's projected " +
		"arrival, Visibility & Tracking's",
	"StopKey": "the stop key a stored mileage is cached under",
	"TariffBill": "withheld by the field feature on ShipmentRatingDetail.bill: the bill " +
		"the shipment's tariff produced, Rating's",
	"TariffBillExchange": "withheld by the field feature on ShipmentRatingDetail.bill: the " +
		"conversion the tariff's bill was priced through",
	"TariffBillLine": "withheld by the field feature on ShipmentRatingDetail.bill: a line " +
		"of the tariff's bill",
	"TariffChoice": "withheld by the field feature on ShipmentRatingDetail.choices: a best " +
		"or worst block the tariff evaluated",
	"TariffChoiceOption": "withheld by the field feature on ShipmentRatingDetail.choices: an " +
		"option a tariff choice weighed",
	"TariffLineLimit": "withheld by the field feature on ShipmentRatingDetail.bill: the limit " +
		"that changed a tariff line's amount",
	"TimeClockEntry":  "a clock entry of the timesheet that carries it",
	"TriageAction":    "an action the inbound message's triage proposes",
	"TriageConflict":  "a conflict the inbound message's triage found",
	"TriageFact":      "a fact the inbound message's triage read from it",
	"TriageLoadDraft": "the load the inbound message's triage drafted from it",
	"TriageLoadField": "a field of the load the inbound message's triage drafted",
	"UncoveredWindowSummary": "a pickup window's uncovered shipments on the shipment " +
		"watchlist, an aggregate of shipments",
	"UsState": "a state an address names: reference data with no feature of its own",
	"WorkerCredentialDocument": "the metadata of the file the record that carries it holds, " +
		"here the fuel purchase import's own source file",
}

// carriesOffPlan reports whether a record of the carrier's feature can carry
// a record of the carried feature that the plan leaves out while it includes
// the carrier: the carried feature is an add-on's, and a different add-on's
// than the carrier's. A type with no kind carries for any feature.
func carriesOffPlan(
	carrier platformcatalog.FeatureKey,
	kinded bool,
	carried platformcatalog.FeatureKey,
) bool {
	carriedFeature, ok := platformcatalog.Lookup(carried)
	if !ok || carriedFeature.IsBase() {
		return false
	}
	if !kinded {
		return true
	}

	return !sameAddOn(carrier, carried)
}

// sameAddOn reports whether two features are bought together: both belong
// to one add-on.
func sameAddOn(a, b platformcatalog.FeatureKey) bool {
	first, ok := platformcatalog.Lookup(a)
	if !ok {
		return false
	}
	second, ok := platformcatalog.Lookup(b)

	return ok && first.AddOn == second.AddOn
}

// typeCarriers is what gates a type with no kind: the records of a kind that
// carry it, and the root fields that return it with nothing of a kind in
// between, connections, edges and payloads included. opaque is set when the
// type, or a type with no kind between it and them, is returned by no field
// by name, as a member of a union or an implementation of an interface is,
// so its carriers cannot all be seen.
type typeCarriers struct {
	kinds  []permission.Resource
	roots  []string
	opaque bool
}

type fieldRef struct {
	owner, field string
}

// returners maps every type to the object fields that return it.
func returners(schema *ast.Schema) map[string][]fieldRef {
	out := make(map[string][]fieldRef)
	for name, def := range schema.Types {
		if def.Kind != ast.Object || def.BuiltIn {
			continue
		}
		for _, field := range def.Fields {
			out[field.Type.Name()] = append(out[field.Type.Name()], fieldRef{name, field.Name})
		}
	}

	return out
}

// carriersOf walks up from a type with no kind through the types with no
// kind that return it, to the records of a kind and the root fields above.
func carriersOf(
	refs map[string][]fieldRef,
	kinds map[string]permission.Resource,
	name string,
) typeCarriers {
	var out typeCarriers
	seen := map[string]struct{}{name: {}}
	queue := []string{name}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if len(refs[current]) == 0 {
			out.opaque = true
		}
		for _, ref := range refs[current] {
			if isRoot(ref.owner) {
				out.roots = append(out.roots, ref.owner+"."+ref.field)
				continue
			}
			if kind, kinded := kinds[ref.owner]; kinded {
				out.kinds = append(out.kinds, kind)
				continue
			}
			if _, done := seen[ref.owner]; !done {
				seen[ref.owner] = struct{}{}
				queue = append(queue, ref.owner)
			}
		}
	}

	return out
}

// onlyWithItsAddOn reports whether a type with no kind is reached only where
// the carried feature's add-on is on: every record that carries it is of that
// add-on, and every root field that returns it has a resolver that checks a
// resource of that add-on, so that root is refused before the type is ever
// returned on a plan without it. A mutation's payload is held to the same
// rule through its mutation.
func onlyWithItsAddOn(
	carriers typeCarriers,
	carried platformcatalog.FeatureKey,
	registry *permissioncatalog.Registry,
	checks map[string][]string,
) bool {
	if carriers.opaque || len(carriers.kinds)+len(carriers.roots) == 0 {
		return false
	}
	for _, kind := range carriers.kinds {
		feature, _ := registry.FeatureOf(kind.String())
		if !sameAddOn(feature, carried) {
			return false
		}
	}
	for _, root := range carriers.roots {
		if !slices.ContainsFunc(checks[strings.ToLower(root)], func(resource string) bool {
			feature, ok := registry.FeatureOf(resource)
			return ok && sameAddOn(feature, carried)
		}) {
			return false
		}
	}

	return true
}

// A field that returns records of a kind the plan can leave out while it
// includes the record holding the field is answered null without its
// resolver running. A non-null one cannot be: the guard refuses the whole
// operation instead, so the field must be nullable. A type with no kind is
// held to it unless it is reached only where that add-on is on: under
// records of the add-on, or under root fields, payloads included, whose
// resolvers check a resource of it. A field of a kind's own that belongs to
// another feature (FieldFeatures) is the same.
func TestEveryFieldCarryingAnotherFeatureIsNullable(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	registry := fieldsensitivity.DefaultRegistry()
	kinds, err := KindTypes(schema, registry)
	require.NoError(t, err)
	returns := RecordTypes(schema, kinds)
	refs := returners(schema)
	checks := resolverReadChecks(t)

	var offenders []string
	for name, def := range schema.Types {
		if def.Kind != ast.Object || def.BuiltIn || isRoot(name) || wrapperType(def) {
			continue
		}
		kind, kinded := kinds[name]
		carrier, _ := registry.FeatureOf(kind.String())
		for _, field := range def.Fields {
			child, ok := returns[field.Type.Name()]
			if !ok || !field.Type.NonNull {
				continue
			}
			carried, _ := registry.FeatureOf(child.String())
			if !carriesOffPlan(carrier, kinded, carried) {
				continue
			}
			if !kinded &&
				onlyWithItsAddOn(carriersOf(refs, kinds, name), carried, registry, checks) {
				continue
			}
			on := "a type with no kind"
			if kinded {
				on = fmt.Sprintf("%s (%s)", kind, carrier)
			}
			offenders = append(offenders, fmt.Sprintf(
				"%s.%s returns %s, a %s record (%s), on %s",
				name, field.Name, field.Type.String(), child, carried, on))
		}
	}
	offenders = append(offenders, nonNullFieldFeatures(schema, registry, kinds)...)
	slices.Sort(offenders)
	assert.Empty(t, offenders,
		"fields that carry another feature's records or values but are non-null, so the guard "+
			"must refuse the whole operation on a plan without that feature instead of answering "+
			"null. Make each field nullable and draw its withheld state on the screen; a type with "+
			"no kind may instead be reached only under records, or root resolvers that check a "+
			"resource, of the carried add-on:\n  %s",
		strings.Join(offenders, "\n  "))
}

func nonNullFieldFeatures(
	schema *ast.Schema,
	registry *permissioncatalog.Registry,
	kinds map[string]permission.Resource,
) []string {
	seen := make(map[string]struct{})
	var out []string
	for name, kind := range kinds {
		def := schema.Types[name]
		resource, ok := registry.Get(kind.String())
		if def == nil || !ok {
			continue
		}
		holders := []string{name}
		for carried := range carriedTypes(schema, kinds, def) {
			holders = append(holders, carried)
		}
		for _, holder := range holders {
			for field, feature := range resource.FieldFeatures {
				found := schema.Types[holder].Fields.ForName(field)
				if found == nil || !found.Type.NonNull {
					continue
				}
				line := fmt.Sprintf(
					"%s.%s is declared in %s's FieldFeatures as %s, but is non-null",
					holder, field, kind, feature)
				if _, dup := seen[line]; !dup {
					seen[line] = struct{}{}
					out = append(out, line)
				}
			}
		}
	}

	return out
}

// Every object type with no kind is gated as the record that carries it, so
// it must be part of that record. A type that looks like another record's
// data is either declared as its kind or listed in partOfCarrier.
func TestEveryTypeARecordCarriesIsAccountedFor(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	registry := fieldsensitivity.DefaultRegistry()
	kinds, err := KindTypes(schema, registry)
	require.NoError(t, err)

	unaccounted := make(map[string][]string)
	for name, kind := range kinds {
		for carried := range carriedTypes(schema, kinds, schema.Types[name]) {
			_, listed := partOfCarrier[carried]
			if listed || sharesLeadingWord(carried, name, kind) {
				continue
			}
			unaccounted[carried] = append(unaccounted[carried],
				fmt.Sprintf("%s (%s)", name, kind))
		}
	}
	lines := make([]string, 0, len(unaccounted))
	for carried, carriers := range unaccounted {
		slices.Sort(carriers)
		lines = append(lines,
			fmt.Sprintf("%s, carried by %s", carried, strings.Join(carriers, ", ")))
	}
	slices.Sort(lines)
	assert.Empty(t, lines,
		"object types with no kind, carried by records of a kind whose name they do not share, "+
			"so the guard gates them as each carrier's kind. Declare the kind a type shows with "+
			"@kind(resource:), or list it in partOfCarrier with the reason it is part of every "+
			"record that carries it:\n  %s", strings.Join(lines, "\n  "))
	for listed := range partOfCarrier {
		assert.NotContainsf(t, kinds, listed,
			"%s declares a kind; remove it from partOfCarrier", listed)
	}
}

// carriedTypes is every object type with no kind that a type of a kind
// reaches without passing through another type of a kind or a connection,
// edge or page info wrapper.
func carriedTypes(
	schema *ast.Schema,
	kinds map[string]permission.Resource,
	from *ast.Definition,
) map[string]struct{} {
	out := make(map[string]struct{})
	if from == nil {
		return out
	}
	queue := []*ast.Definition{from}
	for len(queue) > 0 {
		def := queue[0]
		queue = queue[1:]
		for _, field := range def.Fields {
			child := schema.Types[field.Type.Name()]
			if child == nil || child.Kind != ast.Object || child.BuiltIn ||
				wrapperType(child) || child.Name == "PageInfo" {
				continue
			}
			if _, kinded := kinds[child.Name]; kinded {
				continue
			}
			if _, seen := out[child.Name]; seen {
				continue
			}
			out[child.Name] = struct{}{}
			queue = append(queue, child)
		}
	}

	return out
}

func sharesLeadingWord(carried, carrier string, kind permission.Resource) bool {
	word := leadingWord(carried)
	if strings.EqualFold(word, leadingWord(carrier)) {
		return true
	}
	first, _, _ := strings.Cut(kind.String(), "_")

	return strings.EqualFold(word, first)
}

func leadingWord(name string) string {
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			return name[:i]
		}
	}

	return name
}

func TestLeadingWordsTellAPartFromAnotherRecordsData(t *testing.T) {
	t.Parallel()

	assert.True(t, sharesLeadingWord("InvoiceLine", "Invoice", "invoice"))
	assert.True(t, sharesLeadingWord("AccountingAppSettings", "AccountingIntegration",
		"accounting_integration"))
	assert.True(t, sharesLeadingWord("WorkerProfile", "Driver", "worker"),
		"the resource's name counts as well as the type's")
	assert.False(t, sharesLeadingWord("FactoringSummary", "Invoice", "invoice"))
	assert.Equal(t, "Edi", leadingWord("EdiMessageDetail"))
	assert.Equal(t, "Lane", leadingWord("Lane"))
}

func TestCarriesOffPlanAsksOnlyAboutAnotherAddOn(t *testing.T) {
	t.Parallel()

	assert.True(t, carriesOffPlan(platformcatalog.FeatureBilling, true,
		platformcatalog.FeatureFactoring), "a base invoice carrying a factoring company")
	assert.False(t, carriesOffPlan(platformcatalog.FeatureAccounting, true,
		platformcatalog.FeatureFactoring), "the same add-on is bought together")
	assert.False(t, carriesOffPlan(platformcatalog.FeatureFactoring, true,
		platformcatalog.FeatureBilling), "a base feature is always on")
	assert.True(t, carriesOffPlan("", false, platformcatalog.FeatureDetention),
		"a type with no kind carries for any feature")
}

func TestATypeReachedOnlyWhereItsAddOnIsOnNeedNotBeNullable(t *testing.T) {
	t.Parallel()

	schema := parseSchema(t, `
type Query { card: Card  board: Board  health: EdiAckHealth }
type Mutation { refileCard: Card  saveBoard: Board }
type Card { message: EdiMessage! }
type Board { row: Row }
type Row { message: EdiMessage! }
type EdiAckHealth { overdue: Overdue }
type Overdue { message: EdiMessage! }
type EdiMessage { id: ID }
`)
	kinds := map[string]permission.Resource{
		"EdiMessage":   permission.ResourceEDI,
		"EdiAckHealth": permission.ResourceEDI,
	}
	refs := returners(schema)
	registry := fieldsensitivity.DefaultRegistry()
	checks := map[string][]string{
		"query.card":          {permission.ResourceEDI.String()},
		"mutation.refilecard": {permission.ResourceEDI.String()},
		"query.board":         {permission.ResourceShipment.String()},
		"mutation.saveboard":  {permission.ResourceShipment.String()},
	}
	reachedOnlyWithEDI := func(name string) bool {
		return onlyWithItsAddOn(carriersOf(refs, kinds, name),
			platformcatalog.FeatureEDIIntegration, registry, checks)
	}

	assert.True(t, reachedOnlyWithEDI("Card"),
		"a card whose root and whose mutation both check an EDI resource")
	assert.True(t, reachedOnlyWithEDI("Overdue"), "a part of an EDI record")
	assert.False(t, reachedOnlyWithEDI("Row"),
		"a row of a board whose root and payload check only a base resource")
}
