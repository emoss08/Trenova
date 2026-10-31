package reviewprompt_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueassist/reviewprompt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func money(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func evidence() *reviewprompt.Evidence {
	fuelID := pulid.MustNew("ac_")
	docID := pulid.MustNew("doc_")
	pickup := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		Type:                 shipmentvocab.StopTypePickup,
		Sequence:             0,
		ScheduledWindowStart: 1_792_000_000,
		ActualArrival:        ptr(int64(1_792_000_000)),
		ActualDeparture:      ptr(int64(1_792_000_000 + 4*3600 + 20*60)),
		Location: &location.Location{
			Name:  "Acme DC",
			City:  "Dayton",
			State: &usstate.UsState{Abbreviation: "OH"},
		},
	}

	return &reviewprompt.Evidence{
		Item: &billingqueue.BillingQueueItem{
			ID:                   pulid.MustNew("bqi_"),
			Number:               "BQ-1",
			Status:               billingqueue.StatusReadyForReview,
			AllocatedTotalAmount: money("4163.15"),
			BillToCustomer: &customer.Customer{
				Name: "Acme Foods",
				BillingProfile: &customer.CustomerBillingProfile{
					PaymentTerm:       customer.PaymentTermNet30,
					FuelSurchargeMode: customer.FuelSurchargeModeFuelIncluded,
					RequirePONumber:   true,
					BillingNotes:      "Fuel is built into the linehaul.",
					DocumentTypes:     []*documenttype.DocumentType{{Name: "Proof of Delivery"}},
				},
			},
			Shipment: &shipment.Shipment{
				ProNumber:           "S-104",
				BOL:                 "B-77",
				Weight:              ptr(int64(42000)),
				RateOverrideAmount:  decimal.NewNullDecimal(money("3255")),
				RateOverrideReason:  "Matched the broker's spot quote",
				FuelSurchargeLocked: true,
				Moves: []*shipment.ShipmentMove{{
					Sequence: 0,
					Stops:    []*shipment.Stop{pickup},
				}},
				AdditionalCharges: []*shipment.AdditionalCharge{{
					ID:                fuelID,
					IsSystemGenerated: false,
					AccessorialCharge: &accessorialcharge.AccessorialCharge{Code: "FSC"},
					FuelSurchargeDetail: &shipment.FuelSurchargeDetail{
						ProgramName: "DOE weekly",
						IndexCode:   "EIA-US",
						Price:       3.89,
						PriceDate:   "2026-10-26",
						Miles:       ptr(1180.0),
						RatePerMile: ptr(0.77),
						Amount:      908.14,
						Stale:       true,
					},
				}},
			},
			Review: &billingqueue.Review{
				Checks: []*billingqueue.Check{{
					Key:    billingqueue.CheckCharges,
					State:  billingqueue.CheckStateWarn,
					Detail: "One charge is off the rate con",
				}},
				Issues: []*billingqueue.Issue{{Summary: "$908.14 fuel surcharge isn't on the rate con"}},
				Charges: &billingqueue.ChargeReview{
					HasRateCon: true,
					Lines: []*billingqueue.ChargeLine{
						{
							Key:      "freight-0",
							Label:    "Freight",
							Billed:   money("3255"),
							Expected: decimal.NewNullDecimal(money("3255")),
							Source:   billingqueue.ChargeSourceAgreement,
						},
						{
							Key:                fuelID.String(),
							AdditionalChargeID: fuelID,
							Label:              "Fuel surcharge",
							Billed:             money("908.14"),
							Source:             billingqueue.ChargeSourceNone,
						},
					},
				},
			},
		},
		Documents: []*document.Document{{
			ID:              docID,
			OriginalName:    "pod.pdf",
			SignatureStatus: document.SignatureStatusSigned,
			DocumentType:    &documenttype.DocumentType{Name: "Proof of Delivery"},
			DeliveryFacts: &document.DeliveryFacts{
				SignedBy: "Dana",
				OSD:      document.OSDShort,
				OSDNote:  "2 cases",
			},
		}},
		Excerpts: map[pulid.ID]reviewprompt.Excerpt{docID: {Text: "Received by Dana", Parts: 3}},
		Recent: []*billingqueueports.RecentInvoice{{
			Number:      "INV-9",
			InvoiceDate: 1_791_000_000,
			Total:       money("3300"),
			Charges: []*billingqueueports.RecentCharge{
				{Type: invoice.InvoiceLineTypeFreight, Description: "Linehaul", Amount: money("3300")},
			},
		}},
		Comments: []*shipment.ShipmentComment{{
			ID:         pulid.MustNew("shc_"),
			Comment:    "Customer agreed to pay fuel on this one by phone.",
			Type:       shipmentvocab.CommentTypeDispatch,
			Visibility: shipmentvocab.CommentVisibilityOperations,
			CreatedAt:  1_792_100_000,
			User:       &tenant.User{Name: "Riley"},
		}},
	}
}

func section(t *testing.T, sections []llmports.ContextSection, prefix string) llmports.ContextSection {
	t.Helper()
	for _, s := range sections {
		if strings.HasPrefix(s.Title, prefix) {
			return s
		}
	}
	require.Failf(t, "no section", "no section titled %q", prefix)

	return llmports.ContextSection{}
}

func TestSections_ShowEachChargeWithHowItWasPricedAndTheFuelProgramBehindIt(t *testing.T) {
	t.Parallel()

	invoiceText := section(t, reviewprompt.Sections(evidence()), "The invoice").Content

	assert.Contains(t, invoiceText, "BOL: B-77")
	assert.Contains(t, invoiceText,
		"- Freight: billed $3,255.00 · from the rate agreement · its source priced it at $3,255.00")
	assert.Contains(t, invoiceText, "- Fuel surcharge: billed $908.14 · code FSC · entered by hand")
	assert.Contains(t, invoiceText,
		"fuel surcharge: program DOE weekly · index EIA-US at 3.89 on 2026-10-26 · 1180 miles · "+
			"0.77 per mile · computed 908.14 · index price is stale")
}

func TestSections_SetWhatTheChecksFoundApartAsWhatTheBillerAlreadySees(t *testing.T) {
	t.Parallel()

	checks := section(t, reviewprompt.Sections(evidence()), "What the five checks found")

	assert.Equal(t, "What the five checks found (the biller already sees this)", checks.Title)
	assert.Equal(t,
		"charges: warn · One charge is off the rate con\nopen: $908.14 fuel surcharge isn't on the rate con",
		checks.Content)
}

func TestSections_ShowTheStopsWithTimeOnSiteAndTheRatingOverride(t *testing.T) {
	t.Parallel()

	sections := reviewprompt.Sections(evidence())

	assert.Contains(t, section(t, sections, "The stops").Content,
		"Move 1 stop 1 · Pickup · Acme DC, Dayton OH · window 2026-10-14T17:46:40Z · "+
			"arrived 2026-10-14T17:46:40Z · departed 2026-10-14T22:06:40Z · on site 4h 20m")
	shp := section(t, sections, "The shipment").Content
	assert.Contains(t, shp, "Weight: 42000 lb")
	assert.Contains(t, shp, "Rate overridden to $3,255.00: Matched the broker's spot quote")
	assert.Contains(t, shp, "The fuel surcharge is locked.")
}

func TestSections_GiveThePayersTermsAndFenceWhatPeopleWroteAsData(t *testing.T) {
	t.Parallel()

	sections := reviewprompt.Sections(evidence())

	terms := section(t, sections, "Acme Foods's billing terms")
	assert.True(t, terms.Trusted)
	assert.Equal(
		t,
		"Payment terms: Net30\nFuel is included in the rates: a separate fuel charge would bill it twice.\nEvery invoice needs a PO number.\n"+
			"Documents the payer requires: Proof of Delivery",
		terms.Content,
	)

	notes := section(t, sections, "Billing notes kept for Acme Foods")
	assert.False(t, notes.Trusted)
	assert.Equal(t, "Fuel is built into the linehaul.", notes.Content)

	comments := section(t, sections, "Shipment comments")
	assert.False(t, comments.Trusted)
	assert.Equal(t, "2026-10-15T21:33:20Z · Riley · Dispatch, Operations: "+
		"Customer agreed to pay fuel on this one by phone.", comments.Content)
}

func TestSections_ListPastInvoicesWithWhatEachBilled(t *testing.T) {
	t.Parallel()

	recent := section(t, reviewprompt.Sections(evidence()), "Recent invoices to Acme Foods")

	assert.Equal(t, "INV-9 · 2026-10-03 · total $3,300.00: Linehaul $3,300.00", recent.Content)
}

func TestSections_HeadADocumentWithWhatWasRecordedAndSayWhenItHasMoreToRead(t *testing.T) {
	t.Parallel()

	e := evidence()
	doc := section(t, reviewprompt.Sections(e), "Document ")

	assert.Equal(t, "Document "+e.Documents[0].ID.String()+" · Proof of Delivery · signature Signed"+
		" · recorded: signed by Dana, over/short/damaged Short (2 cases)", doc.Title)
	assert.False(t, doc.Trusted)
	assert.Equal(t, "Received by Dana\n(part 1 of 3; read_document reads the rest)", doc.Content)
}

func TestParts_SplitsTextIntoExcerptSizedPartsByCharacter(t *testing.T) {
	t.Parallel()

	text := strings.Repeat("é", reviewprompt.ExcerptRunes) + "end"
	parts := reviewprompt.Parts(text)

	require.Len(t, parts, 2)
	assert.Equal(t, reviewprompt.ExcerptRunes, len([]rune(parts[0])))
	assert.Equal(t, "end", parts[1])
	assert.Empty(t, reviewprompt.Parts(""))
}

func TestHash_MovesOnlyWhenWhatTheReviewIsShownChanges(t *testing.T) {
	t.Parallel()

	e := evidence()
	before := reviewprompt.Hash(e)

	e.Item.Shipment.Version++
	e.Comments[0].UpdatedAt++
	assert.Equal(t, before, reviewprompt.Hash(e), "a save that changes nothing shown is not a new review")

	e.Comments[0].Comment = "Customer refused the fuel surcharge."
	afterComment := reviewprompt.Hash(e)
	assert.NotEqual(t, before, afterComment)

	*e.Item.Shipment.Moves[0].Stops[0].ActualDeparture += 3600
	afterStop := reviewprompt.Hash(e)
	assert.NotEqual(t, afterComment, afterStop)

	e.Documents[0].DeliveryFacts.Seal = "SL-4"
	assert.NotEqual(t, afterStop, reviewprompt.Hash(e), "what Nova recorded from a document is part of it")
}

func TestValidate_TakesAnEmptySummaryOnlyWhenTheChecksAlreadySaySomething(t *testing.T) {
	t.Parallel()

	flagged := evidence().Item
	clear := evidence().Item
	clear.Review.Issues = nil

	require.Error(t, reviewprompt.Validate(&reviewprompt.Answer{Verdict: "Maybe"}, flagged))
	require.NoError(t, reviewprompt.Validate(&reviewprompt.Answer{Verdict: "Agrees"}, clear))
	require.NoError(t, reviewprompt.Validate(&reviewprompt.Answer{Verdict: "Concern"}, flagged),
		"a concern the checks already explain needs no words of its own")
	require.Error(t, reviewprompt.Validate(&reviewprompt.Answer{Verdict: "Concern", Summary: " "}, clear),
		"a concern no check found says why")
	require.NoError(t, reviewprompt.Validate(
		&reviewprompt.Answer{Verdict: "Concern", Summary: "Fuel is twice this payer's usual."}, clear))
}
