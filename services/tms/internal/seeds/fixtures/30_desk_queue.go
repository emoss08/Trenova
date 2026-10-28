package fixtures

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
)

const deskSeedMinute = int64(60)

type queueSeedSubject struct {
	kind agent.SubjectType
	id   pulid.ID
	path string
}

type queueSeedAgent struct {
	template agenttemplate.Template
	status   agent.RunStatus
	summary  string
	runOn    queueSeedSubject
}

type queueSeedSpec struct {
	key       string
	kind      watchtower.SourceKind
	severity  watchtower.Severity
	title     string
	summary   string
	ago       int64
	record    *watchtower.RecordRef
	facts     []watchtower.Fact
	due       *int64
	dueLabel  string
	impact    []string
	trail     []string
	subject   queueSeedSubject
	eventKind agent.EventKind
	seen      bool
	agent     *queueSeedAgent
}

type queueSeed struct {
	items []*watchtower.Item
	runs  []*agent.AgentRun
	reads []*watchtower.Read
}

func minutes(value int64) *int64 { return &value }

func (r *deskSeedRefs) loadQueueRefs(ctx context.Context, db *keel.DB) error {
	shipments, err := shipmentdb.Shipments.AllTenants().Where(
		shipmentdb.Shipments.OrganizationID.Eq(r.org.ID),
		shipmentdb.Shipments.BusinessUnitID.Eq(r.org.BusinessUnitID),
		shipmentdb.Shipments.ProNumber.In([]string{
			"SEED-SHP-001", "SEED-SHP-002", "SEED-SHP-004", "SEED-SHP-007", "SEED-SHP-008",
		}...),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load seeded shipments: %w", err)
	}
	r.shipments = make(map[string]*shipment.Shipment, len(shipments))
	for _, entity := range shipments {
		r.shipments[entity.ProNumber] = entity
	}

	w := workerdb.Workers
	workers, err := w.AllTenants().Where(
		w.OrganizationID.Eq(r.org.ID),
		w.BusinessUnitID.Eq(r.org.BusinessUnitID),
	).Order(w.LastName.Asc(), w.FirstName.Asc()).Limit(2).List(ctx, db)
	if err != nil {
		return fmt.Errorf("load seeded workers: %w", err)
	}
	r.workers = workers

	return nil
}

func (r *deskSeedRefs) shipmentSubject(pro string) queueSeedSubject {
	entity, ok := r.shipments[pro]
	if !ok {
		return queueSeedSubject{}
	}
	path, _ := productguide.RecordPath("shipment", entity.ID.String())
	return queueSeedSubject{kind: agent.SubjectShipment, id: entity.ID, path: path}
}

func (r *deskSeedRefs) workerAt(index int, fallback string) (string, queueSeedSubject) {
	if index >= len(r.workers) {
		return fallback, queueSeedSubject{}
	}
	entity := r.workers[index]
	name := strings.TrimSpace(entity.FirstName + " " + entity.LastName)
	if name == "" {
		name = fallback
	}
	path, _ := productguide.RecordPath("worker", entity.ID.String())
	return name, queueSeedSubject{kind: agent.SubjectWorker, id: entity.ID, path: path}
}

func shipmentRef(pro string) *watchtower.RecordRef {
	kind, _ := permission.ResourceShipment.Kind()
	return &watchtower.RecordRef{Kind: kind, Label: pro}
}

func recordRef(resource permission.Resource, label string) *watchtower.RecordRef {
	kind, ok := resource.Kind()
	if !ok {
		return &watchtower.RecordRef{Label: label}
	}
	return &watchtower.RecordRef{Kind: kind, Label: label}
}

func seedFacts(pairs ...string) []watchtower.Fact {
	facts := make([]watchtower.Fact, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		facts = append(facts, watchtower.Fact{Label: pairs[i], Value: pairs[i+1]})
	}
	return facts
}

func (r *deskSeedRefs) queueSpecs() []queueSeedSpec {
	dana, danaSubject := r.workerAt(0, "Dana Ortiz")
	tom, tomSubject := r.workerAt(1, "Tom Reyes")

	return []queueSeedSpec{
		{
			key:      "late-007",
			kind:     watchtower.SourceEtaAtRisk,
			severity: watchtower.SeverityCritical,
			title:    "SEED-SHP-007 will miss its delivery window",
			summary: "Stuck behind the I-35 storm near Ames. Minneapolis dock closes at 7 PM; " +
				"the driver is 3h 10m out.",
			ago:    4,
			record: shipmentRef("SEED-SHP-007"),
			facts: seedFacts(
				"Customer", "Northline Foods",
				"Appointment", "Today 6:00 PM",
				"New ETA", "Today 9:10 PM",
				"Driver", "Will Grant",
			),
			due:      minutes(190),
			dueLabel: "Minneapolis dock closes",
			impact:   []string{"Northline Foods", "1 load", "Late fee $250"},
			trail: []string{
				"Picked up at Ames · 11:40 AM",
				"Storm warning on I-35 · 1:15 PM",
				"Speed under 20 mph for 40 min",
				"ETA moved past 6:00 PM appointment",
			},
			subject: r.shipmentSubject("SEED-SHP-007"),
		},
		{
			key:      "insurance-lakeshore",
			kind:     watchtower.SourceCarrierIntelEvent,
			severity: watchtower.SeverityCritical,
			title:    "Lakeshore Logistics' cargo insurance lapses in 9 days",
			summary:  "They're on 4 tendered loads next week. Their certificate on file expires Oct 11.",
			ago:      22,
			record:   recordRef(permission.ResourceCarrier, "Lakeshore Logistics"),
			facts: seedFacts(
				"MC", "771203",
				"Expires", "Oct 11, 2026",
				"Loads affected", "4",
			),
			due:      minutes(9 * 24 * 60),
			dueLabel: "Insurance lapses",
			impact:   []string{"4 tendered loads", "$18,400 freight"},
			trail: []string{
				"Certificate on file expires Oct 11",
				"4 loads tendered for Oct 6–10",
			},
			agent: &queueSeedAgent{
				template: agenttemplate.TemplateCarrierRiskDesk,
				status:   agent.RunStatusGatheringContext,
				runOn:    r.shipmentSubject("SEED-SHP-008"),
			},
		},
		{
			key:      "cash-lockbox",
			kind:     watchtower.SourceAgentRunFailed,
			severity: watchtower.SeverityCritical,
			title:    "Cash Application stopped partway through a lockbox file",
			summary:  "Matched 212 of 340 payments, then lost access to the bank feed.",
			ago:      38,
			record:   &watchtower.RecordRef{Label: "Lockbox · Oct 2"},
			facts: seedFacts(
				"Matched", "212 of 340",
				"Error", "Bank feed token expired",
				"Run", "run_01J9…4QX",
			),
			due:      minutes(0),
			dueLabel: "Payments are unmatched now",
			impact:   []string{"128 payments", "$61,240 unapplied"},
			trail: []string{
				"Lockbox file received · 1:02 PM",
				"212 of 340 matched",
				"Bank feed token expired",
			},
		},
		{
			key:      "detention-001",
			kind:     watchtower.SourceDetentionOccurrence,
			severity: watchtower.SeverityWarning,
			title:    "Detention clock passed 2 hours at Acme Plant 2",
			summary: "SEED-SHP-001 has been at the dock since 1:40 PM. Billable after 2 free " +
				"hours at $65/h.",
			ago:      51,
			record:   shipmentRef("SEED-SHP-001"),
			facts:    seedFacts("Arrived", "1:40 PM", "Free time", "2h", "Accrued", "$32.50"),
			due:      minutes(-51),
			dueLabel: "Detention is accruing",
			impact:   []string{"$32.50 so far", "$65/h"},
			trail: []string{
				"Arrived at Acme Plant 2 · 1:40 PM",
				"Free time ended · 3:40 PM",
				"Still checked in",
			},
			seen: true,
			agent: &queueSeedAgent{
				template: agenttemplate.TemplateDetentionDesk,
				status:   agent.RunStatusDiagnosing,
				runOn:    r.shipmentSubject("SEED-SHP-001"),
			},
		},
		{
			key:      "edi-bluewater",
			kind:     watchtower.SourceEDIInboundQuarantined,
			severity: watchtower.SeverityWarning,
			title:    "3 load tenders quarantined from Bluewater",
			summary:  "The 204s reference a location code we don't have mapped (BW-DC-17).",
			ago:      60,
			record:   recordRef(permission.ResourceCustomer, "Bluewater Retail"),
			facts:    seedFacts("Messages", "3 × 204", "Unknown code", "BW-DC-17"),
			due:      minutes(120),
			dueLabel: "Tenders expire",
			impact:   []string{"3 loads", "$7,900"},
		},
		{
			key:      "credential-medical",
			kind:     watchtower.SourceWorkerCredential,
			severity: watchtower.SeverityWarning,
			title:    dana + "'s medical card expires in 12 days",
			summary:  "Assigned to SEED-SHP-004 next week.",
			ago:      120,
			record:   recordRef(permission.ResourceWorker, dana),
			facts:    seedFacts("Expires", "in 12 days", "Next load", "SEED-SHP-004"),
			due:      minutes(12 * 24 * 60),
			dueLabel: "Medical card expires",
			impact:   []string{"SEED-SHP-004"},
			subject:  danaSubject,
			seen:     true,
			agent: &queueSeedAgent{
				template: agenttemplate.TemplateCredentialDesk,
				status:   agent.RunStatusAwaitingDecision,
				summary:  "Waiting for you to approve the reminder",
				runOn:    danaSubject,
			},
		},
		{
			key:      "billing-biller",
			kind:     watchtower.SourceBillingException,
			severity: watchtower.SeverityWarning,
			title:    "11 invoices can't post without a biller",
			summary:  "Ready for review, but nobody is assigned as biller.",
			ago:      125,
			record:   &watchtower.RecordRef{Label: "Billing queue"},
			facts:    seedFacts("Invoices", "11", "Amount", "$34,120.50"),
			due:      minutes(360),
			dueLabel: "Billing cutoff",
			impact:   []string{"11 invoices", "$34,120.50"},
			seen:     true,
		},
		{
			key:       "service-harbor",
			kind:      watchtower.SourceServiceFailure,
			severity:  watchtower.SeverityWarning,
			title:     "Two late deliveries for Harbor Supply this week",
			summary:   "A second service failure in 5 days. Their contract has a 95% on-time clause.",
			ago:       180,
			record:    recordRef(permission.ResourceCustomer, "Harbor Supply Co."),
			facts:     seedFacts("On-time this month", "91%", "Contract minimum", "95%"),
			impact:    []string{"Harbor Supply", "95% on-time clause"},
			subject:   r.shipmentSubject("SEED-SHP-004"),
			eventKind: agent.EventServiceFailureDetected,
			seen:      true,
		},
		{
			key:      "late-002",
			kind:     watchtower.SourceEtaAtRisk,
			severity: watchtower.SeverityInfo,
			title:    "SEED-SHP-002 running about 2 hours late",
			summary:  "Weather near Des Moines. Still inside the delivery window.",
			ago:      185,
			record:   shipmentRef("SEED-SHP-002"),
			facts:    seedFacts("New ETA", "6:45 PM", "Window", "until 8:00 PM"),
			due:      minutes(105),
			dueLabel: "Delivery window closes",
			impact:   []string{"Bluewater dock notified"},
			subject:  r.shipmentSubject("SEED-SHP-002"),
			seen:     true,
			agent: &queueSeedAgent{
				template: agenttemplate.TemplateLoadMonitor,
				status:   agent.RunStatusCompleted,
				summary:  "Customer notified · watching",
				runOn:    r.shipmentSubject("SEED-SHP-002"),
			},
		},
		{
			key:       "credential-hazmat",
			kind:      watchtower.SourceWorkerCredential,
			severity:  watchtower.SeverityInfo,
			title:     tom + "'s hazmat endorsement renews next month",
			summary:   "No loads need hazmat before then.",
			ago:       300,
			record:    recordRef(permission.ResourceWorker, tom),
			facts:     seedFacts("Expires", "in 46 days"),
			due:       minutes(46 * 24 * 60),
			dueLabel:  "Endorsement renews",
			subject:   tomSubject,
			eventKind: agent.EventWorkerCredentialExpiring,
			seen:      true,
		},
		{
			key:      "edi-acme-214",
			kind:     watchtower.SourceEDIInboundQuarantined,
			severity: watchtower.SeverityInfo,
			title:    "Acme sent 6 status updates we didn't expect",
			summary:  "214s for loads already delivered. Probably a resend.",
			ago:      360,
			record:   recordRef(permission.ResourceCustomer, "Acme Manufacturing"),
			facts:    seedFacts("Messages", "6 × 214"),
			seen:     true,
		},
	}
}

func (s *DeskSeed) queueItems(refs *deskSeedRefs, readerID pulid.ID) *queueSeed {
	specs := refs.queueSpecs()
	out := &queueSeed{
		items: make([]*watchtower.Item, 0, len(specs)),
		runs:  make([]*agent.AgentRun, 0, 4),
		reads: make([]*watchtower.Read, 0, len(specs)),
	}

	for _, spec := range specs {
		occurredAt := refs.now - spec.ago*deskSeedMinute
		item := &watchtower.Item{
			ID:             pulid.MustNew("wt_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			SourceKind:     spec.kind,
			SourceID:       "seed-watchtower-" + spec.key,
			Severity:       spec.severity,
			Title:          spec.title,
			Summary:        spec.summary,
			Path:           spec.subject.path,
			OccurredAt:     occurredAt,
			Status:         watchtower.StatusOpen,
			Facts:          spec.facts,
			Impact:         spec.impact,
			Trail:          seedTrail(spec.trail),
		}
		if spec.record != nil {
			item.RecordKind = spec.record.Kind
			item.RecordLabel = spec.record.Label
		}
		if spec.subject.id.IsNotNil() {
			item.SubjectType = spec.subject.kind
			item.SubjectID = spec.subject.id
			item.EventKind = spec.eventKind
		}
		if spec.due != nil {
			item.ApplyDeadline(&watchtower.Deadline{
				At:    refs.now + *spec.due*deskSeedMinute,
				Label: spec.dueLabel,
			})
		} else {
			item.DueSource = watchtower.DueSourcePerson
		}
		item.Normalize()

		if spec.agent != nil {
			if run := refs.seedRun(spec.agent, occurredAt); run != nil {
				if err := item.HandTo(run.AgentDefinitionID, run.ID, readerID, occurredAt); err == nil {
					out.runs = append(out.runs, run)
				}
			}
		}
		if spec.seen && readerID.IsNotNil() {
			out.reads = append(out.reads, &watchtower.Read{
				ItemID:         item.ID,
				UserID:         readerID,
				OrganizationID: refs.org.ID,
				BusinessUnitID: refs.org.BusinessUnitID,
				SeenAt:         refs.now,
			})
		}
		out.items = append(out.items, item)
	}

	return out
}

func seedTrail(lines []string) []watchtower.TrailEvent {
	trail := make([]watchtower.TrailEvent, 0, len(lines))
	for _, line := range lines {
		trail = append(trail, watchtower.TrailEvent{Text: line})
	}
	return trail
}

func (r *deskSeedRefs) seedRun(spec *queueSeedAgent, handedAt int64) *agent.AgentRun {
	definition, ok := r.definitions[spec.template]
	if !ok || spec.runOn.id.IsNil() {
		return nil
	}
	run := &agent.AgentRun{
		ID:                pulid.MustNew("ar_"),
		OrganizationID:    r.org.ID,
		BusinessUnitID:    r.org.BusinessUnitID,
		AgentDefinitionID: definition.ID,
		AgentType:         agent.TypeGeneral,
		Trigger:           agent.RunTriggerManual,
		SubjectType:       spec.runOn.kind,
		SubjectID:         spec.runOn.id,
		Status:            spec.status,
		Summary:           spec.summary,
		ModelIdentifier:   seedActivityModel,
		PromptVersion:     "seed/v1",
		InputContextHash:  "seed-" + spec.runOn.id.String(),
		StartedAt:         handedAt,
	}
	if spec.status == agent.RunStatusCompleted {
		completedAt := handedAt + 5*deskSeedMinute
		run.CompletedAt = &completedAt
	}
	return run
}
