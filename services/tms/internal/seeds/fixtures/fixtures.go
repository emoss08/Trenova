package fixtures

import (
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/seeds/fixtures/accountingfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/agentfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/assistantfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/carrierfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/deskfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/detentionfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/driverpayfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/edifixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/fuelfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/inboundfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/masterdatafixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/pricingfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/ratefixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/shipmentfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/workforcefixtures"
	"github.com/emoss08/trenova/internal/seeds/reference"
)

var Seeds = seed.NewRegistry()

var AdminAccount = Seeds.Add(seed.Seed{
	Name: "admin_account",
	Kind: seed.Fixture,
	Description: "The DEFAULT business unit, the Trenova Logistics and Trenova Transportation " +
		"organizations with their controls, sequences, reference data and administrator role, " +
		"and the admin user (admin / admin123!) who belongs to both",
	After: []*seed.Seed{reference.USStates},
	Run:   tenantfixtures.RunAdminAccount,
})

var FormulaTemplates = Seeds.Add(seed.Seed{
	Name:        "formula_templates",
	Kind:        seed.Fixture,
	Description: "The standard rating method formula templates, approved, in Tariff",
	After:       []*seed.Seed{AdminAccount},
	Run:         pricingfixtures.RunFormulaTemplates,
})

var FormulaLibraries = Seeds.Add(seed.Seed{
	Name:        "formula_libraries",
	Kind:        seed.Fixture,
	Description: "The organisation's shared Tariff libraries, approved",
	After:       []*seed.Seed{AdminAccount},
	Run:         pricingfixtures.RunFormulaLibraries,
})

var Worker = Seeds.Add(seed.Seed{
	Name:        "worker",
	Kind:        seed.Fixture,
	Description: "Creates worker data for development",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&workforcefixtures.WorkerSeed{}).Run,
})

var LocationCategory = Seeds.Add(seed.Seed{
	Name:        "location_category",
	Kind:        seed.Fixture,
	Description: "Creates location category data for development",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&masterdatafixtures.LocationCategorySeed{}).Run,
})

var Location = Seeds.Add(seed.Seed{
	Name:        "location",
	Kind:        seed.Fixture,
	Description: "Creates location data for development",
	After:       []*seed.Seed{LocationCategory, reference.USStates},
	Run:         (&masterdatafixtures.LocationSeed{}).Run,
})

var TestData = Seeds.Add(seed.Seed{
	Name:        "test_data",
	Kind:        seed.Fixture,
	Description: "Creates test data for development environment",
	After:       []*seed.Seed{AdminAccount, Worker},
	Run:         (&masterdatafixtures.TestDataSeed{}).Run,
})

var LookupMatrix = Seeds.Add(seed.Seed{
	Name:        "lookup_matrix",
	Kind:        seed.Fixture,
	Description: "Creates sample single-axis rate tables for development",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&pricingfixtures.LookupMatrixSeed{}).Run,
})

var FuelSurcharge = Seeds.Add(seed.Seed{
	Name:        "fuel_surcharge",
	Kind:        seed.Fixture,
	Description: "Creates all DOE regional diesel indices with recent weekly prices and a demo fuel surcharge program",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&pricingfixtures.FuelSurchargeSeed{}).Run,
})

var Carrier = Seeds.Add(seed.Seed{
	Name:        "carrier",
	Kind:        seed.Fixture,
	Description: "Creates carriers, contacts, and insurance policies covering the coverage-gate matrix",
	After:       []*seed.Seed{AdminAccount, reference.USStates},
	Run:         (&carrierfixtures.CarrierSeed{}).Run,
})

var RoutingGuide = Seeds.Add(seed.Seed{
	Name:        "routing_guide",
	Kind:        seed.Fixture,
	Description: "Creates routing guides on the seeded shipment lanes so waterfall tenders can be demonstrated",
	After:       []*seed.Seed{Carrier, Location},
	Run:         (&carrierfixtures.RoutingGuideSeed{}).Run,
})

var WorkerCredential = Seeds.Add(seed.Seed{
	Name:        "worker_credential",
	Kind:        seed.Fixture,
	Description: "Seeds credential types and per-worker credentials mirrored from worker profiles",
	After:       []*seed.Seed{Worker},
	Run:         (&workforcefixtures.WorkerCredentialSeed{}).Run,
})

var WorkerEmploymentEvent = Seeds.Add(seed.Seed{
	Name:        "worker_employment_event",
	Kind:        seed.Fixture,
	Description: "Seeds hire and informational employment events for seeded workers",
	After:       []*seed.Seed{Worker},
	Run:         (&workforcefixtures.WorkerEmploymentEventSeed{}).Run,
})

var WorkerChecklist = Seeds.Add(seed.Seed{
	Name:        "worker_checklist",
	Kind:        seed.Fixture,
	Description: "Seeds onboarding/offboarding checklist templates and opens checklists for recent hires",
	After:       []*seed.Seed{WorkerCredential, WorkerEmploymentEvent},
	Run:         (&workforcefixtures.WorkerChecklistSeed{}).Run,
})

var OrgHoliday = Seeds.Add(seed.Seed{
	Name:        "org_holiday",
	Kind:        seed.Fixture,
	Description: "Seeds the organisation holiday calendar and a peak-season blackout",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&workforcefixtures.OrgHolidaySeed{}).Run,
})

var WorkerTraining = Seeds.Add(seed.Seed{
	Name:        "worker_training",
	Kind:        seed.Fixture,
	Description: "Seeds the training course catalog and per-worker training records",
	After:       []*seed.Seed{Worker},
	Run:         (&workforcefixtures.WorkerTrainingSeed{}).Run,
})

var WorkerSafety = Seeds.Add(seed.Seed{
	Name:        "worker_safety",
	Kind:        seed.Fixture,
	Description: "Seeds safety events, discipline, recognition, a review template and reviews per worker",
	After:       []*seed.Seed{Worker},
	Run:         (&workforcefixtures.WorkerSafetySeed{}).Run,
})

var FuelSpendIFTA = Seeds.Add(seed.Seed{
	Name:        "fuel_spend_ifta",
	Kind:        seed.Fixture,
	Description: "Seeds fuel cards, fuel purchases, IFTA tax rates and jurisdiction mileage for the last two completed quarters",
	After:       []*seed.Seed{reference.IFTAJurisdictions, TestData},
	Run:         (&fuelfixtures.FuelSpendIFTASeed{}).Run,
})

var AIProvider = Seeds.Add(seed.Seed{
	Name:        "ai_provider",
	Kind:        seed.Fixture,
	Description: "Seeds local, self-hosted and hosted AI providers with task routing",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&agentfixtures.AIProviderSeed{}).Run,
})

var AgentDefinition = Seeds.Add(seed.Seed{
	Name:        "agent_definition",
	Kind:        seed.Fixture,
	Description: "Seeds chat, event-driven and scheduled agents for the default organization",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&agentfixtures.AgentDefinitionSeed{}).Run,
})

var HoldReason = Seeds.Add(seed.Seed{
	Name:        "hold_reason",
	Kind:        seed.Fixture,
	Description: "Seeds the hold reasons dispatch, compliance, customer service and billing place holds with",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&masterdatafixtures.HoldReasonSeed{}).Run,
})

var DOTRandomPool = Seeds.Add(seed.Seed{
	Name:        "dot_random_pool",
	Kind:        seed.Fixture,
	Description: "Seeds the default DOT random testing pool at the FMCSA minimum rates",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&workforcefixtures.DOTRandomPoolSeed{}).Run,
})

var LaneDistance = Seeds.Add(seed.Seed{
	Name:        "lane_distance",
	Kind:        seed.Fixture,
	Description: "Seeds road distances between the development locations, so a new load prices its miles without a routing provider",
	After:       []*seed.Seed{AdminAccount, Location},
	Run:         (&masterdatafixtures.LaneDistanceSeed{}).Run,
})

var FiscalYear = Seeds.Add(seed.Seed{
	Name:        "fiscal_year",
	Kind:        seed.Fixture,
	Description: "Opens the current calendar fiscal year with its monthly periods, so invoices and journals post",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&accountingfixtures.FiscalYearSeed{}).Run,
})

var OutboundChannels = Seeds.Add(seed.Seed{
	Name:        "outbound_channels",
	Kind:        seed.Fixture,
	Description: "Adds an email profile with no key and a portal site with no sign-in, so agents can propose sending and portal work",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&deskfixtures.OutboundChannelsSeed{}).Run,
})

var SystemAccount = Seeds.Add(seed.Seed{
	Name:        "system_account",
	Kind:        seed.Fixture,
	Description: "The system user background work runs as, with the configured password",
	After:       []*seed.Seed{AdminAccount},
	Run:         tenantfixtures.RunSystemAccount,
})

var Shipment = Seeds.Add(seed.Seed{
	Name:        "shipment",
	Kind:        seed.Fixture,
	Description: "Creates sample shipments with moves, stops, commodities, and charges for development",
	After:       []*seed.Seed{TestData, FormulaTemplates, Location},
	Run:         (&shipmentfixtures.ShipmentSeed{}).Run,
})

var DriverPay = Seeds.Add(seed.Seed{
	Name:        "driver_pay",
	Kind:        seed.Fixture,
	Description: "Creates pay profiles, completed shipments with assigned moves, pay events, and settlements for development",
	After:       []*seed.Seed{Worker, TestData, Shipment},
	Run:         (&driverpayfixtures.DriverPaySeed{}).Run,
})

var DriverPayLedger = Seeds.Add(seed.Seed{
	Name:        "driver_pay_ledger",
	Kind:        seed.Fixture,
	Description: "Wires driver settlement GL accounts, cost-control sourcing, recurring earnings, and journals for seeded settlements",
	After:       []*seed.Seed{AdminAccount, SystemAccount, DriverPay},
	Run:         (&accountingfixtures.DriverPayLedgerSeed{}).Run,
})

var Detention = Seeds.Add(seed.Seed{
	Name:        "detention",
	Kind:        seed.Fixture,
	Description: "Creates detention policies, notice recipients, and worked detention occurrences with evidence chains",
	After:       []*seed.Seed{TestData, Shipment, DriverPay},
	Run:         (&detentionfixtures.DetentionSeed{}).Run,
})

var RateAgreement = Seeds.Add(seed.Seed{
	Name:        "rate_agreement",
	Kind:        seed.Fixture,
	Description: "Creates rate zones, a zone rate table, and a customer rate agreement with overlapping lanes",
	After:       []*seed.Seed{Shipment, TestData, FuelSurcharge, FormulaTemplates},
	Run:         (&ratefixtures.RateAgreementSeed{}).Run,
})

var PTOPolicy = Seeds.Add(seed.Seed{
	Name:        "pto_policy",
	Kind:        seed.Fixture,
	Description: "Creates PTO accrual policies, enrols seeded workers, and posts opening balances and accruals",
	After:       []*seed.Seed{Worker, DriverPay},
	Run:         (&driverpayfixtures.PTOPolicySeed{}).Run,
})

var AssistantConversation = Seeds.Add(seed.Seed{
	Name:        "assistant_conversation",
	Kind:        seed.Fixture,
	Description: "Seeds assistant conversations with tool traffic and a pending proposal",
	After:       []*seed.Seed{AgentDefinition, Shipment},
	Run:         (&assistantfixtures.AssistantConversationSeed{}).Run,
})

var Insight = Seeds.Add(seed.Seed{
	Name:        "insight",
	Kind:        seed.Fixture,
	Description: "Seeds operational insights with history, a dismissal and a resolution",
	After:       []*seed.Seed{Shipment, Location, AIProvider},
	Run:         (&assistantfixtures.InsightSeed{}).Run,
})

var InboundMessage = Seeds.Add(seed.Seed{
	Name:        "inbound_message",
	Kind:        seed.Fixture,
	Description: "Seeds three monitored addresses and a Triage queue in every state the design shows",
	After: []*seed.Seed{
		Shipment,
		DriverPay,
		AdminAccount,
		Carrier,
		Worker,
		AIProvider,
		Detention,
		HoldReason,
	},
	Run: (&inboundfixtures.InboundMessageSeed{}).Run,
})

var AgentActivity = Seeds.Add(seed.Seed{
	Name:        "agent_activity",
	Kind:        seed.Fixture,
	Description: "Seeds agent runs, decisions, a plan, exceptions and the trust ledger",
	After:       []*seed.Seed{AgentDefinition, Shipment, Worker},
	Run:         (&agentfixtures.AgentActivitySeed{}).Run,
})

var Desk = Seeds.Add(seed.Seed{
	Name:        "desk",
	Kind:        seed.Fixture,
	Description: "Seeds the watchtower feed and today's briefing",
	After:       []*seed.Seed{InboundMessage, AgentActivity, Insight},
	Run:         (&deskfixtures.DeskSeed{}).Run,
})

var AssistantArtifact = Seeds.Add(seed.Seed{
	Name:        "assistant_artifact",
	Kind:        seed.Fixture,
	Description: "Seeds Desk artifacts of every kind the pane can render",
	After:       []*seed.Seed{AssistantConversation, AgentActivity},
	Run:         (&assistantfixtures.AssistantArtifactSeed{}).Run,
})

var CarrierRateAgreement = Seeds.Add(seed.Seed{
	Name:        "carrier_rate_agreement",
	Kind:        seed.Fixture,
	Description: "Gives each active carrier an approved buy-side contract, so a carrier shop prices a lane",
	After:       []*seed.Seed{RateAgreement},
	Run:         (&ratefixtures.CarrierRateAgreementSeed{}).Run,
})

var AIAnalytics = Seeds.Add(seed.Seed{
	Name:        "ai_analytics",
	Kind:        seed.Fixture,
	Description: "Seeds a month of AI analytics across local, self-hosted and hosted providers",
	After:       []*seed.Seed{AIProvider, AgentDefinition, AgentActivity},
	Run:         (&agentfixtures.AIAnalyticsSeed{}).Run,
})

var EDITradingPartner = Seeds.Add(seed.Seed{
	Name:        "edi_trading_partner",
	Kind:        seed.Fixture,
	Description: "Seeds an outside EDI trading partner for Acme Manufacturing with an active Lading project from the carrier starter and 214s in every acknowledgment state",
	After:       []*seed.Seed{AdminAccount, Shipment},
	Run:         (&edifixtures.EDITradingPartnerSeed{}).Run,
})

var Receivables = Seeds.Add(seed.Seed{
	Name:        "receivables",
	Kind:        seed.Fixture,
	Description: "Posts aged invoices, customer payments and a credit memo so receivables read a real ledger",
	After:       []*seed.Seed{AdminAccount, SystemAccount, Shipment, FiscalYear},
	Run:         (&accountingfixtures.ReceivablesSeed{}).Run,
})
