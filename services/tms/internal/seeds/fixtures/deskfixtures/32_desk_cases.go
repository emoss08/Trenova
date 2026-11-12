package deskfixtures

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowerservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowershipment"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevidence"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan/agentplandb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttemplate"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase/recordcasedb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower/watchtowerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta/stopetadb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity/commoditydb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics/telematicsdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/agentfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/pricingfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	// CaseShipmentPro is the late shipment whose case the Desk opens first.
	CaseShipmentPro = "SEED-CASE-001"

	caseMinute        = int64(60)
	caseHourMs        = int64(3_600_000)
	caseMinuteMs      = int64(60_000)
	caseWindowIn      = int64(435)
	caseWindowLength  = int64(60)
	caseDriveLeft     = int64(130)
	caseDriveToGo     = int64(258)
	caseRest          = int64(630)
	caseMilesToGo     = 215.0
	caseDepartedAgo   = int64(840)
	caseRaisedAgo     = int64(12)
	caseMessageAgo    = int64(15)
	caseDraftedAgo    = int64(5)
	caseNewWindowIn   = int64(870)
	caseRelayArrival  = int64(400)
	caseDisputeMinor  = int64(125_000)
	casePastDueDays   = int64(14)
	caseDisputeOpened = int64(3)
)

// DeskCaseSeed writes one case of each kind the Desk can open, each with the
// evidence a person decides it on.
//
// The shipment is the dispatcher's case: a load in transit whose driver runs
// out of hours short of the door. It carries what the live system would: a
// position from the tractor's telematics, the driver's hours, a projected
// arrival per stop, the customer asking where it is, the condition the
// Watchtower raised, three ways it could go and a plan for the recommended
// one, made of the two writes that carry it out.
//
// The invoice is the collector's case, past due with a reminder and a follow-up
// to approve; the dispute is the biller's, a rate discrepancy with a credit to
// approve. Both are a real invoice from the receivables seed.
//
// Depends on:
//   - Shipment, Receivables: the customers, locations and invoices
//   - AgentDefinition: the agents that propose
//   - Desk: the feed the cases join
type DeskCaseSeed struct{}

type caseRefs struct {
	org         *tenant.Organization
	admin       *tenant.User
	tenant      pagination.TenantInfo
	zone        *time.Location
	now         int64
	origin      *location.Location
	destination *location.Location
	customer    *customer.Customer
	driver      *worker.Worker
	tractor     *tractor.Tractor
	trailer     *trailer.Trailer
	serviceType pulid.ID
	shipType    pulid.ID
	commodity   pulid.ID
	formula     *formulatemplate.FormulaTemplate
	dispatch    *agentdefinition.Definition
	billing     *agentdefinition.Definition
}

func (s *DeskCaseSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	sh := shipmentdb.Shipments
	seeded, err := sh.AllTenants().Where(
		sh.OrganizationID.Eq(org.ID),
		sh.BusinessUnitID.Eq(org.BusinessUnitID),
		sh.ProNumber.Eq(CaseShipmentPro),
	).Exists(ctx, db)
	if err != nil {
		return fmt.Errorf("look for the seeded case shipment: %w", err)
	}
	if seeded {
		return nil
	}

	refs, err := s.loadRefs(ctx, db, org)
	if err != nil {
		return err
	}
	if err = s.shipmentCase(ctx, db, refs); err != nil {
		return err
	}

	return s.invoiceCases(ctx, db, refs)
}

func (s *DeskCaseSeed) loadRefs(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (*caseRefs, error) {
	refs := &caseRefs{
		org:    org,
		tenant: pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID},
		zone:   timeutils.LoadLocation(org.Timezone),
		now:    timeutils.NowUnix(),
	}

	var err error
	if refs.admin, err = tenantfixtures.UserByUsername(ctx, db, "admin"); err != nil {
		return nil, fmt.Errorf("get the reader the cases are written for: %w", err)
	}
	l := locationdb.Locations
	if refs.origin, err = l.AllTenants().Where(
		l.OrganizationID.Eq(org.ID), l.BusinessUnitID.Eq(org.BusinessUnitID), l.Code.Eq("WH-DAL"),
	).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get the case's origin: %w", err)
	}
	if refs.destination, err = l.AllTenants().Where(
		l.OrganizationID.Eq(org.ID), l.BusinessUnitID.Eq(org.BusinessUnitID), l.Code.Eq("DC-CHI"),
	).Load(l.State()).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get the case's destination: %w", err)
	}
	refs.zone, _ = timeutils.ResolveZone(refs.destination.Timezone, org.Timezone)

	c := customerdb.Customers
	if refs.customer, err = c.AllTenants().Where(
		c.OrganizationID.Eq(org.ID), c.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(c.Code.Asc()).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get the case's customer: %w", err)
	}

	w := workerdb.Workers
	if refs.driver, err = w.AllTenants().Where(
		w.OrganizationID.Eq(org.ID), w.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(w.CreatedAt.Asc(), w.ID.Asc()).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get the case's driver: %w", err)
	}
	tr := tractordb.Tractors
	if refs.tractor, err = tr.AllTenants().Where(
		tr.OrganizationID.Eq(org.ID), tr.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(tr.Code.Asc()).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get the case's tractor: %w", err)
	}
	tl := trailerdb.Trailers
	if refs.trailer, err = tl.AllTenants().Where(
		tl.OrganizationID.Eq(org.ID), tl.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(tl.Code.Asc()).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get the case's trailer: %w", err)
	}

	st := servicetypedb.ServiceTypes
	service, err := st.AllTenants().Pick(st.ID).Where(
		st.OrganizationID.Eq(org.ID), st.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(st.Code.Asc()).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get a service type: %w", err)
	}
	refs.serviceType = service.ID
	sht := shipmenttypedb.ShipmentTypes
	shipType, err := sht.AllTenants().Pick(sht.ID).Where(
		sht.OrganizationID.Eq(org.ID), sht.BusinessUnitID.Eq(org.BusinessUnitID),
	).Order(sht.Code.Asc()).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get a shipment type: %w", err)
	}
	refs.shipType = shipType.ID
	cm := commoditydb.Commodities
	freight, err := cm.AllTenants().Pick(cm.ID).Where(
		cm.OrganizationID.Eq(org.ID), cm.BusinessUnitID.Eq(org.BusinessUnitID),
		cm.Name.Eq("General Freight"),
	).First(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("get general freight commodity: %w", err)
	}
	refs.commodity = freight.ID
	ft := formulatemplatedb.FormulaTemplates
	if refs.formula, err = ft.AllTenants().Pick(ft.ID, ft.Name).Where(
		ft.OrganizationID.Eq(org.ID), ft.BusinessUnitID.Eq(org.BusinessUnitID),
		ft.Name.Eq("Flat Rate"),
	).First(ctx, db); err != nil {
		return nil, fmt.Errorf("get flat rate formula template: %w", err)
	}

	d := agentdefinitiondb.Definitions
	definitions, err := d.AllTenants().Where(
		d.OrganizationID.Eq(org.ID), d.BusinessUnitID.Eq(org.BusinessUnitID), d.Enabled.IsTrue(),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load agent definitions: %w", err)
	}
	for _, definition := range definitions {
		switch definition.Template {
		case agenttemplate.TemplateDispatchAssistant:
			refs.dispatch = definition
		case agenttemplate.TemplateBillingAssistant:
			refs.billing = definition
		}
	}
	if refs.dispatch == nil || refs.billing == nil {
		return nil, errors.New("the dispatch and billing agents are not seeded")
	}

	return refs, nil
}

func (r *caseRefs) driverName() string {
	return r.driver.FirstName + " " + r.driver.LastName
}

func (r *caseRefs) clock(ts int64) string {
	return time.Unix(ts, 0).In(r.zone).Format("3:04 PM")
}

func (r *caseRefs) stopLocal(stop *location.Location, ts int64) string {
	zone, _ := timeutils.ResolveZone(stop.Timezone, r.org.Timezone)

	return time.Unix(ts, 0).In(zone).Format(toolschema.LocalDateTimeLayout)
}

func (s *DeskCaseSeed) shipmentCase(ctx context.Context, db *keel.DB, refs *caseRefs) error {
	now := refs.now
	windowStart := now + caseWindowIn*caseMinute
	windowEnd := windowStart + caseWindowLength*caseMinute
	departed := now - caseDepartedAgo*caseMinute
	eta := now + (caseDriveToGo+caseRest)*caseMinute
	slack := (windowEnd - eta) / caseMinute
	raised := now - caseRaisedAgo*caseMinute
	revenue := decimal.NewFromInt(3200)

	shp := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		BusinessUnitID:    refs.org.BusinessUnitID,
		OrganizationID:    refs.org.ID,
		CustomerID:        refs.customer.ID,
		ServiceTypeID:     refs.serviceType,
		ShipmentTypeID:    refs.shipType,
		FormulaTemplateID: refs.formula.ID,
		Status:            shipmentvocab.StatusInTransit,
		ProNumber:         CaseShipmentPro,
		BOL:               "BOL-2026-0901",
		Pieces:            new(int64(22)),
		Weight:            new(int64(38500)),
		ActualShipDate:    new(departed),
		TariffInputs: pricingfixtures.SeededTariffInputs(
			formulatemplate.StandardFlatRate,
			revenue,
		),
		FreightChargeAmount: decimal.NullDecimal{Decimal: revenue, Valid: true},
		TotalChargeAmount:   decimal.NullDecimal{Decimal: revenue, Valid: true},
		EntryMethod:         shipmentvocab.EntryMethodManual,
		FreightTerms:        shipmentvocab.FreightTermsPrepaid,
		FreightLines:        []shipment.FreightLine{},
		OtherChargeAmount:   decimal.NewNullDecimal(decimal.Zero),
	}
	shp.FreightLines = pricingfixtures.SeededFreightLines(shp)
	shp.RatingDetail = pricingfixtures.SeededRatingDetail(shp, refs.formula, now)
	if err := shipmentdb.Shipments.Create(ctx, db, shp); err != nil {
		return fmt.Errorf("insert the case shipment: %w", err)
	}
	if err := shipmentdb.ShipmentCommodities.Create(ctx, db, &shipment.ShipmentCommodity{
		ID:             pulid.MustNew("sc_"),
		BusinessUnitID: refs.org.BusinessUnitID,
		OrganizationID: refs.org.ID,
		ShipmentID:     shp.ID,
		CommodityID:    refs.commodity,
		Pieces:         22,
		Weight:         38500,
	}); err != nil {
		return fmt.Errorf("insert the case shipment's commodity: %w", err)
	}

	move := &shipment.ShipmentMove{
		ID:             pulid.MustNew("sm_"),
		BusinessUnitID: refs.org.BusinessUnitID,
		OrganizationID: refs.org.ID,
		ShipmentID:     shp.ID,
		Status:         shipmentvocab.MoveStatusInTransit,
		Loaded:         true,
		Sequence:       0,
		CoverageType:   shipmentvocab.MoveCoverageTypeDriver,
	}
	if err := shipmentdb.ShipmentMoves.Create(ctx, db, move); err != nil {
		return fmt.Errorf("insert the case move: %w", err)
	}

	pickup := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		BusinessUnitID:       refs.org.BusinessUnitID,
		OrganizationID:       refs.org.ID,
		ShipmentMoveID:       move.ID,
		LocationID:           refs.origin.ID,
		Type:                 shipmentvocab.StopTypePickup,
		ScheduleType:         shipmentvocab.StopScheduleTypeAppointment,
		Handling:             shipmentvocab.StopHandlingLive,
		Status:               shipmentvocab.StopStatusCompleted,
		Sequence:             0,
		ScheduledWindowStart: departed - 90*caseMinute,
		ScheduledWindowEnd:   new(departed - 30*caseMinute),
		ActualArrival:        new(departed - 70*caseMinute),
		ActualDeparture:      new(departed),
	}
	delivery := &shipment.Stop{
		ID:                   pulid.MustNew("stp_"),
		BusinessUnitID:       refs.org.BusinessUnitID,
		OrganizationID:       refs.org.ID,
		ShipmentMoveID:       move.ID,
		LocationID:           refs.destination.ID,
		Type:                 shipmentvocab.StopTypeDelivery,
		ScheduleType:         shipmentvocab.StopScheduleTypeAppointment,
		Handling:             shipmentvocab.StopHandlingLive,
		Status:               shipmentvocab.StopStatusNew,
		Sequence:             1,
		ScheduledWindowStart: windowStart,
		ScheduledWindowEnd:   new(windowEnd),
	}
	if err := shipmentdb.Stops.CreateMany(ctx, db, []*shipment.Stop{pickup, delivery}); err != nil {
		return fmt.Errorf("insert the case stops: %w", err)
	}
	if err := shipmentdb.Assignments.Create(ctx, db, &shipment.Assignment{
		ID:              pulid.MustNew("asn_"),
		BusinessUnitID:  refs.org.BusinessUnitID,
		OrganizationID:  refs.org.ID,
		ShipmentMoveID:  move.ID,
		PrimaryWorkerID: &refs.driver.ID,
		TractorID:       &refs.tractor.ID,
		TrailerID:       &refs.trailer.ID,
		Status:          shipmentvocab.AssignmentStatusInProgress,
		AckStatus:       shipmentvocab.AssignmentAckAccepted,
	}); err != nil {
		return fmt.Errorf("insert the case assignment: %w", err)
	}
	if err := shipmentprojection.Refresh(ctx, db, shp.ID); err != nil {
		return err
	}

	if err := s.telemetry(ctx, db, refs); err != nil {
		return err
	}
	projected, err := s.stopEtas(ctx, db, refs, &caseEtas{
		shipment: shp, move: move, pickup: pickup, delivery: delivery,
		eta: eta, slack: slack, windowEnd: windowEnd,
	})
	if err != nil {
		return err
	}

	run := &agent.AgentRun{
		ID:                pulid.MustNew("ar_"),
		OrganizationID:    refs.org.ID,
		BusinessUnitID:    refs.org.BusinessUnitID,
		AgentDefinitionID: refs.dispatch.ID,
		AgentType:         agent.TypeGeneral,
		Trigger:           agent.RunTriggerEvent,
		SubjectType:       agent.SubjectShipment,
		SubjectID:         shp.ID,
		Status:            agent.RunStatusAwaitingDecision,
		Summary:           CaseShipmentPro + " will miss its delivery window; a plan waits on a person.",
		ModelIdentifier:   agentfixtures.SeedActivityModel,
		PromptVersion:     "seed/v1",
		InputContextHash:  "seed-case-" + shp.ID.String(),
		StartedAt:         raised + 2*caseMinute,
	}
	if err := agentdb.AgentRuns.Create(ctx, db, run); err != nil {
		return fmt.Errorf("insert the case run: %w", err)
	}

	newStart := now + caseNewWindowIn*caseMinute
	newEnd := newStart + caseWindowLength*caseMinute
	plan, err := s.plan(ctx, db, refs, &casePlan{
		run:    run,
		target: shp.ID,
		title:  "Move " + CaseShipmentPro + "'s delivery to " + refs.clock(newStart),
		summary: fmt.Sprintf(
			"%s cannot reach %s inside the window on the hours left. Moving the door to "+
				"after the break keeps the load with one driver.",
			refs.driverName(), refs.destination.Name,
		),
		steps: []*agent.AgentProposal{
			caseProposal("shipment_reschedule_stop", agent.TierActWithApproval,
				"The projected arrival is after the window closes, and the receiver takes "+
					"appointments through the evening.",
				map[string]any{
					"shipmentId":  shp.ID.String(),
					"stopId":      delivery.ID.String(),
					"windowStart": refs.stopLocal(refs.destination, newStart),
					"windowEnd":   refs.stopLocal(refs.destination, newEnd),
				},
				permission.ResourceShipment, shp.ID,
			),
			caseProposal("customer_send_email", agent.TierPropose,
				refs.customer.Name+" asked for the arrival time and is staffing the door.",
				map[string]any{
					"shipmentId": shp.ID.String(),
					"subject":    "New arrival time for " + CaseShipmentPro,
					"body": fmt.Sprintf(
						"%s is running behind: the driver reaches his hours limit before "+
							"%s. We have asked to move the door to %s and expect to arrive about %s.",
						CaseShipmentPro,
						refs.destination.City,
						refs.clock(newStart),
						refs.clock(eta),
					),
				},
				permission.ResourceShipment, shp.ID,
			),
		},
	})
	if err != nil {
		return err
	}

	items := []*watchtower.Item{
		s.etaItem(refs, projected),
		s.messageItem(refs, shp, windowStart),
		s.planItem(refs, plan, refs.dispatch.Name),
	}
	if err = watchtowerdb.Items.CreateMany(ctx, db, items); err != nil {
		return fmt.Errorf("insert the shipment case's items: %w", err)
	}

	late := timeutils.FormatLongDurationMs(-slack * caseMinuteMs)
	options := []*recordcase.Option{
		{
			Key:         "reschedule",
			Kind:        recordcase.OptionKindPlan,
			Label:       "Move the delivery to " + refs.clock(newStart),
			Outcome:     refs.clock(eta),
			OutcomeNote: "inside the new window",
			Tone:        recordcase.ToneGood,
			Cost:        "No fee if " + refs.customer.Name + " confirms",
			Description: fmt.Sprintf(
				"%s keeps the load and delivers after his break; the receiver's door moves to %s.",
				refs.driver.FirstName, refs.window(newStart, newEnd),
			),
			Recommended: true,
			Chosen:      true,
			DraftRunID:  run.ID,
			Projection:  &recordcase.Projection{Window: refs.window(newStart, newEnd)},
		},
		{
			Key:         "relay",
			Kind:        recordcase.OptionKindPlan,
			Label:       "Relay at Bloomington with another driver",
			Outcome:     refs.clock(now + caseRelayArrival*caseMinute),
			OutcomeNote: "on time",
			Tone:        recordcase.ToneGood,
			Cost:        "+$85 deadhead",
			Description: "A second driver takes the load at the Bloomington yard and " +
				refs.driver.FirstName + " takes his break there.",
			Projection: &recordcase.Projection{Step: &recordcase.ProjectedStep{
				Label:     "Relay · Bloomington yard",
				When:      refs.clock(now + (caseDriveLeft-25)*caseMinute),
				Latitude:  new(40.4842),
				Longitude: new(-88.9937),
			}},
		},
		{
			Key:         "leave",
			Kind:        recordcase.OptionKindLeave,
			Label:       "Leave it as it is",
			Outcome:     refs.clock(eta),
			OutcomeNote: late + " late",
			Tone:        recordcase.ToneBad,
			Cost:        "A service failure on " + refs.customer.Name + "'s scorecard",
			Description: "Nothing changes and the load delivers after the window closes.",
		},
	}
	for position, option := range options {
		option.OrganizationID = refs.org.ID
		option.BusinessUnitID = refs.org.BusinessUnitID
		option.RecordKind = recordkind.Kind(permission.ResourceShipment)
		option.RecordID = shp.ID
		option.Position = position
		option.AgentDefinitionID = refs.dispatch.ID
		option.RunID = run.ID
		option.Normalize()
	}
	if err = recordcasedb.Options.CreateMany(ctx, db, options); err != nil {
		return fmt.Errorf("insert the shipment case's options: %w", err)
	}

	return s.reading(ctx, db, refs, shp, items, late, eta)
}

func (r *caseRefs) window(start, end int64) string {
	from, to := time.Unix(start, 0).In(r.zone), time.Unix(end, 0).In(r.zone)
	if from.Format("PM") == to.Format("PM") {
		return from.Format("3:04") + "–" + to.Format("3:04 PM")
	}

	return from.Format("3:04 PM") + "–" + to.Format("3:04 PM")
}

func (s *DeskCaseSeed) telemetry(ctx context.Context, db *keel.DB, refs *caseRefs) error {
	recorded := refs.now - 4*caseMinute
	vp := telematicsdb.VehiclePositions
	if _, err := vp.AllTenants().Where(
		vp.OrganizationID.Eq(refs.org.ID),
		vp.BusinessUnitID.Eq(refs.org.BusinessUnitID),
		vp.TractorID.Eq(refs.tractor.ID),
	).Delete(ctx, db); err != nil {
		return fmt.Errorf("clear the case tractor's position: %w", err)
	}
	if err := vp.Create(ctx, db, &telematics.VehiclePosition{
		OrganizationID:    refs.org.ID,
		BusinessUnitID:    refs.org.BusinessUnitID,
		TractorID:         refs.tractor.ID,
		Provider:          "Samsara",
		ProviderVehicleID: "seed-case-vehicle",
		Latitude:          39.7817,
		Longitude:         -89.6501,
		HeadingDegrees:    35,
		SpeedMph:          61,
		EngineState:       telematics.EngineStateOn,
		FormattedLocation: "Springfield, IL",
		RecordedAt:        recorded,
		ReceivedAt:        recorded + 8,
	}); err != nil {
		return fmt.Errorf("insert the case tractor's position: %w", err)
	}

	hs := telematicsdb.WorkerHOSStates
	if _, err := hs.AllTenants().Where(
		hs.OrganizationID.Eq(refs.org.ID),
		hs.BusinessUnitID.Eq(refs.org.BusinessUnitID),
		hs.WorkerID.Eq(refs.driver.ID),
	).Delete(ctx, db); err != nil {
		return fmt.Errorf("clear the case driver's hours: %w", err)
	}
	if err := hs.Create(ctx, db, &telematics.WorkerHOSState{
		OrganizationID:      refs.org.ID,
		BusinessUnitID:      refs.org.BusinessUnitID,
		WorkerID:            refs.driver.ID,
		Provider:            "Samsara",
		ProviderDriverID:    "seed-case-driver",
		DutyStatus:          telematics.DutyStatusDriving,
		DriveRemainingMs:    caseDriveLeft * caseMinuteMs,
		ShiftRemainingMs:    245 * caseMinuteMs,
		CycleRemainingMs:    31 * caseHourMs,
		CycleTomorrowMs:     38 * caseHourMs,
		BreakRemainingMs:    caseDriveLeft * caseMinuteMs,
		CurrentVehicleID:    "seed-case-vehicle",
		RulesetCycle:        "USA 70 hour / 8 day",
		RulesetShift:        "US Interstate Property",
		RulesetRestart:      "34-hour Restart",
		RulesetBreak:        "Property (off-duty/sleeper)",
		RulesetJurisdiction: "US",
		CurrentTractorID:    refs.tractor.ID,
		RecordedAt:          recorded,
		ReceivedAt:          recorded + 8,
	}); err != nil {
		return fmt.Errorf("insert the case driver's hours: %w", err)
	}

	return nil
}

type caseEtas struct {
	shipment  *shipment.Shipment
	move      *shipment.ShipmentMove
	pickup    *shipment.Stop
	delivery  *shipment.Stop
	eta       int64
	slack     int64
	windowEnd int64
}

func (s *DeskCaseSeed) stopEtas(
	ctx context.Context,
	db *keel.DB,
	refs *caseRefs,
	p *caseEtas,
) (*stopeta.StopETA, error) {
	computed := refs.now - caseRaisedAgo*caseMinute
	input := computed - caseMinute
	rows := []*stopeta.StopETA{
		{
			ID:               pulid.MustNew("seta_"),
			OrganizationID:   refs.org.ID,
			BusinessUnitID:   refs.org.BusinessUnitID,
			ShipmentID:       p.shipment.ID,
			ShipmentMoveID:   p.move.ID,
			StopID:           p.pickup.ID,
			MoveSequence:     0,
			StopSequence:     0,
			Arrived:          true,
			Verdict:          stopeta.VerdictUnknown,
			Basis:            stopeta.BasisAtStop,
			DriveMethod:      stopeta.DriveMethodNone,
			ThresholdMinutes: 30,
			ComputedAt:       computed,
		},
		{
			ID:               pulid.MustNew("seta_"),
			OrganizationID:   refs.org.ID,
			BusinessUnitID:   refs.org.BusinessUnitID,
			ShipmentID:       p.shipment.ID,
			ShipmentMoveID:   p.move.ID,
			StopID:           p.delivery.ID,
			MoveSequence:     0,
			StopSequence:     1,
			EstimatedArrival: new(p.eta),
			WindowEnd:        new(p.windowEnd),
			SlackMinutes:     new(p.slack),
			Verdict:          stopeta.VerdictLate,
			Basis:            stopeta.BasisPosition,
			DriveMethod:      stopeta.DriveMethodRouted,
			MilesRemaining:   caseMilesToGo,
			DriveMinutes:     caseDriveToGo,
			RestMinutes:      caseRest,
			InputAt:          new(input),
			ThresholdMinutes: 30,
			Reason: fmt.Sprintf(
				"%.0f miles of routed driving from the tractor's last position, with a "+
					"10-hour break and a 30-minute break when the driver's hours run out.",
				caseMilesToGo,
			),
			ComputedAt: computed,
		},
	}
	if err := stopetadb.StopETAs.CreateMany(ctx, db, rows); err != nil {
		return nil, fmt.Errorf("insert the case's projected arrivals: %w", err)
	}

	return rows[1], nil
}

type casePlan struct {
	run     *agent.AgentRun
	target  pulid.ID
	title   string
	summary string
	steps   []*agent.AgentProposal
}

func caseProposal(
	tool string,
	tier agent.AutonomyTier,
	rationale string,
	params map[string]any,
	resource permission.Resource,
	target pulid.ID,
) *agent.AgentProposal {
	return &agent.AgentProposal{
		ToolName:     tool,
		ToolParams:   params,
		Confidence:   decimal.NewFromFloat(0.88),
		Rationale:    rationale,
		Evidence:     []agentevidence.EvidenceRef{{Type: string(resource), ID: target.String()}},
		AutonomyTier: tier,
		Status:       agent.ProposalStatusPending,
		HeldBy:       []string{},
	}
}

func (s *DeskCaseSeed) plan(
	ctx context.Context,
	db *keel.DB,
	refs *caseRefs,
	p *casePlan,
) (*agentplan.AgentPlan, error) {
	plan := &agentplan.AgentPlan{
		ID:             pulid.MustNew("apl_"),
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		RunID:          p.run.ID,
		Title:          p.title,
		Summary:        p.summary,
		Status:         agentplan.PlanStatusPending,
		TargetID:       p.target,
		StepCount:      len(p.steps),
		ExpiresAt:      refs.now + 3*deskSeedDay,
		CreatedAt:      refs.now - caseDraftedAgo*caseMinute,
	}
	if err := agentplandb.AgentPlans.Create(ctx, db, plan); err != nil {
		return nil, fmt.Errorf("insert the case plan: %w", err)
	}
	for i, step := range p.steps {
		step.OrganizationID = refs.org.ID
		step.BusinessUnitID = refs.org.BusinessUnitID
		step.RunID = p.run.ID
		step.PlanID = &plan.ID
		step.PlanStep = i + 1
		step.ExpiresAt = plan.ExpiresAt
		if err := agentdb.AgentProposals.Create(ctx, db, step); err != nil {
			return nil, fmt.Errorf("insert the case plan's step %d: %w", i+1, err)
		}
	}

	return plan, nil
}

func (s *DeskCaseSeed) planItem(
	refs *caseRefs,
	plan *agentplan.AgentPlan,
	agentName string,
) *watchtower.Item {
	input := watchtowersources.DescribePlan(plan, agentName)
	input.TenantInfo = refs.tenant
	item := watchtowerservice.ItemFromInput(input)
	item.ID = pulid.MustNew("wt_")
	item.Status = watchtower.StatusOpen

	return item
}

func (s *DeskCaseSeed) etaItem(
	refs *caseRefs,
	projected *stopeta.StopETA,
) *watchtower.Item {
	stop := &stopetaports.AttentionStop{
		ETA:          projected,
		ProNumber:    CaseShipmentPro,
		StopType:     shipmentvocab.StopTypeDelivery,
		LocationName: refs.destination.Name,
		City:         refs.destination.City,
	}
	if refs.destination.State != nil {
		stop.StateCode = refs.destination.State.Abbreviation
	}
	item := watchtowerservice.ItemFromInput(watchtowershipment.DescribeEtaRisk(refs.tenant, stop))
	item.ID = pulid.MustNew("wt_")
	item.Status = watchtower.StatusOpen

	return item
}

func (s *DeskCaseSeed) messageItem(
	refs *caseRefs,
	shp *shipment.Shipment,
	windowStart int64,
) *watchtower.Item {
	received := refs.now - caseMessageAgo*caseMinute
	item := &watchtower.Item{
		ID:             pulid.MustNew("wt_"),
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		SourceKind:     watchtower.SourceInboundMessage,
		SourceID:       "seed-case-message-" + shp.ID.String(),
		Severity:       watchtower.SeverityWarning,
		Title:          "ETA for " + CaseShipmentPro + "?",
		Summary: fmt.Sprintf(
			"Can you confirm the ETA for %s? We need to staff the %s door.",
			CaseShipmentPro, refs.clock(windowStart),
		),
		OccurredAt:  received,
		Status:      watchtower.StatusOpen,
		RecordKind:  recordkind.Kind(permission.ResourceInboundMessage),
		RecordLabel: "ETA for " + CaseShipmentPro + "?",
		Facts: []watchtower.Fact{
			{Label: "From", Value: "receiving@" + caseDomain(refs.customer.Code)},
			{Label: "Kind", Value: "StatusRequest"},
			{Label: "Received", At: received},
		},
		DueSource: watchtower.DueSourcePerson,
	}
	item.SetAbout(shp.ID)
	item.Normalize()

	return item
}

func caseDomain(code string) string {
	domain := make([]rune, 0, len(code))
	for _, r := range code {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			domain = append(domain, r)
		} else if r >= 'A' && r <= 'Z' {
			domain = append(domain, r+('a'-'A'))
		}
	}
	if len(domain) == 0 {
		return "customer.example"
	}

	return string(domain) + ".example"
}

func (s *DeskCaseSeed) reading(
	ctx context.Context,
	db *keel.DB,
	refs *caseRefs,
	shp *shipment.Shipment,
	items []*watchtower.Item,
	late string,
	eta int64,
) error {
	signals := make([]recordcase.Signal, 0, len(items))
	for _, item := range items {
		signals = append(signals, recordcase.Signal{
			Kind:     string(item.SourceKind),
			SourceID: item.SourceID,
			Severity: string(item.Severity),
			Title:    item.Title,
			DueAt:    item.DueAt,
		})
	}
	verdict := fmt.Sprintf(
		"%s has ==%s of driving left==[2] and is %.0f miles from %s[1]. He takes his "+
			"10-hour break on the way, so the load reaches the door around %s, %s after it "+
			"closes[4]. %s is already asking[3].",
		refs.driverName(),
		timeutils.FormatLongDurationMs(caseDriveLeft*caseMinuteMs),
		caseMilesToGo,
		refs.destination.City,
		refs.clock(eta),
		late,
		refs.customer.Name,
	)
	written := refs.now - caseDraftedAgo*caseMinute
	if err := recordcasedb.Readings.Create(ctx, db, &recordcase.Reading{
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		RecordKind:     recordkind.Kind(permission.ResourceShipment),
		RecordID:       shp.ID,
		Fingerprint:    recordcase.ReadingFingerprint("In transit · projected late", signals),
		Status:         recordcase.ReadingStatusReady,
		Segments:       recordcase.ParseVerdict(verdict, 4),
		WhyItMatters: refs.customer.Name +
			" is staffing the door for this load, and a missed appointment counts as a service failure.",
		IfNothingChanges: fmt.Sprintf(
			"Delivered about %s, %s after the window closes.", refs.clock(eta), late,
		),
		ModelIdentifier: agentfixtures.SeedActivityModel,
		RequestedAt:     written,
		WrittenAt:       written,
	}); err != nil {
		return fmt.Errorf("insert the shipment case's reading: %w", err)
	}

	return nil
}

func (s *DeskCaseSeed) invoiceCases(ctx context.Context, db *keel.DB, refs *caseRefs) error {
	inv := invoicedb.Invoices
	invoices, err := inv.AllTenants().Where(
		inv.OrganizationID.Eq(refs.org.ID),
		inv.BusinessUnitID.Eq(refs.org.BusinessUnitID),
		inv.Status.Eq(invoice.StatusPosted),
		inv.SettlementStatus.Eq(invoice.SettlementStatusUnpaid),
		inv.DisputeStatus.Eq(invoice.DisputeStatusNone),
		inv.BillType.Eq(billingvocab.BillTypeInvoice),
	).Order(inv.InvoiceDate.Asc(), inv.ID.Asc()).Limit(2).List(ctx, db)
	if err != nil {
		return fmt.Errorf("load the invoices for the billing cases: %w", err)
	}
	if len(invoices) < 2 {
		return nil
	}

	if err = s.pastDueCase(ctx, db, refs, invoices[0]); err != nil {
		return err
	}

	return s.disputeCase(ctx, db, refs, invoices[1])
}

func (s *DeskCaseSeed) billingRun(
	ctx context.Context,
	db *keel.DB,
	refs *caseRefs,
	subject agent.SubjectType,
	id pulid.ID,
	summary string,
) (*agent.AgentRun, error) {
	run := &agent.AgentRun{
		ID:                pulid.MustNew("ar_"),
		OrganizationID:    refs.org.ID,
		BusinessUnitID:    refs.org.BusinessUnitID,
		AgentDefinitionID: refs.billing.ID,
		AgentType:         agent.TypeGeneral,
		Trigger:           agent.RunTriggerScheduled,
		SubjectType:       subject,
		SubjectID:         id,
		Status:            agent.RunStatusAwaitingDecision,
		Summary:           summary,
		ModelIdentifier:   agentfixtures.SeedActivityModel,
		PromptVersion:     "seed/v1",
		InputContextHash:  "seed-case-" + id.String(),
		StartedAt:         refs.now - 40*caseMinute,
	}
	if err := agentdb.AgentRuns.Create(ctx, db, run); err != nil {
		return nil, fmt.Errorf("insert a billing case run: %w", err)
	}

	return run, nil
}

func (s *DeskCaseSeed) pastDueCase(
	ctx context.Context,
	db *keel.DB,
	refs *caseRefs,
	entity *invoice.Invoice,
) error {
	due := refs.now - casePastDueDays*deskSeedDay
	sent := due - 30*deskSeedDay
	inv := invoicedb.Invoices
	if _, err := inv.AllTenants().Where(
		inv.OrganizationID.Eq(refs.org.ID),
		inv.BusinessUnitID.Eq(refs.org.BusinessUnitID),
		inv.ID.Eq(entity.ID),
	).Update(ctx, db,
		inv.DueDate.To(due),
		inv.InvoiceDate.To(sent),
		inv.SentAt.To(sent),
		inv.SendStatus.To(invoice.SendStatusSent),
	); err != nil {
		return fmt.Errorf("age the past-due case's invoice: %w", err)
	}

	run, err := s.billingRun(ctx, db, refs, agent.SubjectInvoice, entity.ID,
		"Invoice "+entity.Number+" is past due and nobody has followed up.")
	if err != nil {
		return err
	}
	plan, err := s.plan(ctx, db, refs, &casePlan{
		run:    run,
		target: entity.ID,
		title:  "Chase invoice " + entity.Number + " with " + entity.BillToName,
		summary: fmt.Sprintf(
			"Invoice %s is %d days past due and no reminder has gone out since it was sent.",
			entity.Number, casePastDueDays,
		),
		steps: []*agent.AgentProposal{
			caseProposal("receivable_send_payment_reminder", agent.TierPropose,
				"The invoice is past its terms and the customer has not been reminded.",
				map[string]any{
					"customerId": entity.CustomerID.String(),
					"note":       "Invoice " + entity.Number + " is past its terms.",
				},
				permission.ResourceInvoice, entity.ID,
			),
			caseProposal("receivable_schedule_follow_up", agent.TierActWithApproval,
				"A reminder nobody checks on is a reminder that did nothing.",
				map[string]any{
					"customerId": entity.CustomerID.String(),
					"followUpOn": time.Unix(refs.now+3*deskSeedDay, 0).
						In(refs.zone).
						Format(time.DateOnly),
					"note": "Call accounts payable if " + entity.Number + " is still open.",
				},
				permission.ResourceInvoice, entity.ID,
			),
		},
	})
	if err != nil {
		return err
	}

	return watchtowerdb.Items.Create(ctx, db, s.planItem(refs, plan, refs.billing.Name))
}

func (s *DeskCaseSeed) disputeCase(
	ctx context.Context,
	db *keel.DB,
	refs *caseRefs,
	entity *invoice.Invoice,
) error {
	var err error
	amount := min(caseDisputeMinor, entity.TotalAmountMinor)
	dispute := &invoicedispute.InvoiceDispute{
		ID:                  pulid.MustNew("idsp_"),
		OrganizationID:      refs.org.ID,
		BusinessUnitID:      refs.org.BusinessUnitID,
		InvoiceID:           entity.ID,
		CustomerID:          entity.CustomerID,
		Status:              invoicedispute.DisputeCaseStatusOpen,
		ReasonCode:          invoicedispute.DisputeReasonRateDiscrepancy,
		DisputedAmount:      decimal.New(amount, -2),
		DisputedAmountMinor: amount,
		Notes: "The linehaul on this invoice is higher than the rate we agreed for the lane. " +
			"Please correct it to the contract rate.",
		OpenedByID: refs.admin.ID,
		OpenedAt:   refs.now - caseDisputeOpened*deskSeedDay,
	}
	if err = invoicedisputedb.InvoiceDisputes.Create(ctx, db, dispute); err != nil {
		return fmt.Errorf("insert the seeded dispute: %w", err)
	}
	inv := invoicedb.Invoices
	if _, err = inv.AllTenants().Where(
		inv.OrganizationID.Eq(refs.org.ID),
		inv.BusinessUnitID.Eq(refs.org.BusinessUnitID),
		inv.ID.Eq(entity.ID),
	).Update(ctx, db, inv.DisputeStatus.To(invoice.DisputeStatusDisputed)); err != nil {
		return fmt.Errorf("mark the seeded invoice disputed: %w", err)
	}

	run, err := s.billingRun(ctx, db, refs, agent.SubjectInvoiceDispute, dispute.ID,
		entity.BillToName+" disputes the rate on invoice "+entity.Number+".")
	if err != nil {
		return err
	}
	plan, err := s.plan(ctx, db, refs, &casePlan{
		run:    run,
		target: dispute.ID,
		title:  "Rebill invoice " + entity.Number + " at the contract rate",
		summary: "The billed linehaul is above the agreement for the lane; the difference " +
			"is the amount disputed.",
		steps: []*agent.AgentProposal{
			caseProposal(
				"invoice_resolve_dispute",
				agent.TierPropose,
				"The agreement on file has the lower rate, so the customer is right about the difference.",
				map[string]any{
					"disputeId":       dispute.ID.String(),
					"resolution":      string(invoicedispute.DisputeResolutionRebilled),
					"resolutionNotes": "Rate corrected to the agreement; rebilled at the contract rate.",
				},
				permission.ResourceInvoiceDispute,
				dispute.ID,
			),
			caseProposal("customer_send_email", agent.TierPropose,
				"The customer should hear that the credit is on its way before they chase it.",
				map[string]any{
					"shipmentId": entity.ShipmentID.String(),
					"subject":    "Corrected invoice " + entity.Number,
					"body": "You were right about the rate on invoice " + entity.Number +
						". We have corrected it to the contract rate and a corrected invoice follows.",
				},
				permission.ResourceInvoiceDispute, dispute.ID,
			),
		},
	})
	if err != nil {
		return err
	}

	return watchtowerdb.Items.Create(ctx, db, s.planItem(refs, plan, refs.billing.Name))
}
