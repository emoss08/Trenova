package settlementvolume

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edicarrierinvoice/edicarrierinvoicedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument/edidocumentdb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edipayload"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol/carriersettlementcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/seeds/volume/ledgervolume"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	csMinute         = int64(60)
	csHour           = int64(3_600)
	csDay            = int64(86_400)
	csAccountAP      = "2010"
	csAccountExpense = "5600"
	csSourceType     = "CarrierSettlement"
	csEDIShare       = 0.45
)

const (
	csTablePartners    = "edi_partners"
	csTableChannels    = "carrier_edi_channels"
	csTableMessages    = "edi_messages"
	csTableInvoices    = "edi_carrier_invoices"
	csTableRateCons    = "rate_confirmations"
	csTableBatches     = "carrier_settlement_batches"
	csTableSettlements = "carrier_settlements"
	csTableLines       = "carrier_settlement_lines"
	csTableEvents      = "carrier_cost_events"
	csTableMatches     = "carrier_invoice_matches"
	csTableLedger      = "carrier_ledger_entries"
)

var csConfirmerTitles = []string{"Dispatcher", "Owner", "Operations Manager", "Load Planner"}

var csVoidReasons = []string{
	"Generated before the carrier's corrected invoice arrived",
	"Wrong pay period; regenerating",
	"Carrier remit-to changed; regenerating with the new remit details",
}

var csRejectReasons = []string{
	"Detention line needs the signed in/out times attached",
	"Lumper reimbursement missing the receipt",
	"Fuel surcharge does not match the rate confirmation",
}

var csMatchRejectNotes = []string{
	"Detention not authorized on the rate confirmation; carrier to rebill at the contract rate",
	"Lumper was paid by the receiver; carrier billed it twice",
	"Layover not approved by dispatch",
}

type csAdjustmentKind struct {
	description string
	low, high   int64
	negative    bool
}

var csAdjustmentKinds = []csAdjustmentKind{
	{description: "Detention approved at consignee", low: 7_500, high: 30_000},
	{description: "Lumper reimbursement", low: 15_000, high: 45_000},
	{description: "Layover", low: 15_000, high: 25_000},
	{description: "Cargo claim deduction", low: 5_000, high: 40_000, negative: true},
	{description: "Late delivery chargeback", low: 2_500, high: 10_000, negative: true},
}

type csCarrier struct {
	entity      *carrier.Carrier
	partner     *edi.EDIPartner
	nextInvoice int
	nextControl int
}

type csMove struct {
	rec        *volumegen.ShipmentRecord
	move       *volumegen.MoveRecord
	assignment *shipment.CarrierAssignment
	carrier    *csCarrier
	assignedAt int64
	match      *carriersettlement.InvoiceMatch
}

type csEvent struct {
	entity    *carriersettlement.CostEvent
	createdAt int64
	updatedAt int64
	move      *csMove
	match     *carriersettlement.InvoiceMatch
}

type csSettlement struct {
	entity    *carriersettlement.CarrierSettlement
	events    []*csEvent
	createdAt int64
	updatedAt int64
	replaced  bool
}

type csPeriod struct {
	start int64
	end   int64
	pay   int64
}

type csSetup struct {
	control *carriersettlementcontrol.CarrierSettlementControl
	billing *ledgervolume.Setup
	expense pulid.ID
	payable pulid.ID
	canPost bool
	canPay  bool
}

type csRows struct {
	partners    []*edi.EDIPartner
	channels    []*carrier.CarrierEDIChannel
	messages    []*edimessage.EDIMessage
	invoices    []*edicarrierinvoice.CarrierInvoice
	rateCons    []*rateconfirmation.RateConfirmation
	batches     []*carriersettlement.CarrierSettlementBatch
	settlements []*csSettlement
	lines       []*carriersettlement.CarrierSettlementLine
	events      []*csEvent
	matches     []*carriersettlement.InvoiceMatch
	ledger      []*carriersettlement.LedgerEntry
	stamps      map[string][]volumegen.Stamp
	billing     *ledgervolume.Rows
	touched     map[pulid.ID]struct{}
}

func (r *csRows) stamp(table string, id pulid.ID, created, updated int64) {
	r.stamps[table] = append(
		r.stamps[table],
		volumegen.Stamp{Id: id, Created: created, Updated: updated},
	)
}

func (g *settlementPhase) seedCarrierSettlements(ctx context.Context) error {
	moves := g.csBrokeredMoves()
	if len(moves) == 0 {
		return nil
	}

	setup, err := g.csLoadSetup(ctx)
	if err != nil {
		return err
	}
	carriers, err := g.csLoadCarriers(ctx, moves)
	if err != nil {
		return err
	}

	rows := &csRows{
		stamps: make(map[string][]volumegen.Stamp),
		billing: &ledgervolume.Rows{
			Balances: make(map[ledgervolume.BalanceKey]*ledgervolume.Balance),
		},
		touched: make(map[pulid.ID]struct{}),
	}

	if err = g.csPartners(ctx, rows, carriers); err != nil {
		return err
	}
	for _, mv := range moves {
		g.csRateConfirmations(rows, mv)
	}
	if err = g.csInvoices(ctx, rows, setup, moves); err != nil {
		return err
	}
	events := g.csAccrue(rows, setup, moves)
	if err = g.csSettle(ctx, rows, setup, events); err != nil {
		return err
	}
	if err = g.Materialize(ctx, rows.billing); err != nil {
		return err
	}

	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		return g.csWrite(ctx, tx, rows)
	}); err != nil {
		return err
	}

	g.csReport(rows)
	return nil
}

func (g *settlementPhase) csBrokeredMoves() []*csMove {
	var moves []*csMove
	for _, rec := range g.Shipments {
		for _, mv := range rec.Moves {
			if mv == nil || mv.CarrierAssignment == nil {
				continue
			}
			assignedAt, err := mv.CarrierAssignment.ID.Time()
			if err != nil {
				assignedAt = time.Unix(rec.CreatedAt, 0)
			}
			moves = append(moves, &csMove{
				rec:        rec,
				move:       mv,
				assignment: mv.CarrierAssignment,
				assignedAt: assignedAt.Unix(),
			})
		}
	}
	sort.SliceStable(
		moves,
		func(i, j int) bool { return moves[i].assignedAt < moves[j].assignedAt },
	)
	return moves
}

func (g *settlementPhase) csLoadSetup(ctx context.Context) (*csSetup, error) {
	control, err := volumegen.FirstOrNil(
		ctx,
		g.DB,
		carriersettlementcontroldb.CarrierSettlementControls.Query(),
	)
	switch {
	case err != nil:
		return nil, fmt.Errorf("load carrier settlement control: %w", err)
	case control == nil:
		control = &carriersettlementcontrol.CarrierSettlementControl{
			OrganizationID:     g.Refs.OrgID,
			BusinessUnitID:     g.Refs.BuID,
			PayTrigger:         settlementcontrol.PayTriggerShipmentDelivered,
			PayPeriodFrequency: settlementcontrol.PayPeriodFrequencyWeekly,
			PeriodEndDayOfWeek: 6,
			PayDelayDays:       5,
		}
		if err = carriersettlementcontroldb.CarrierSettlementControls.Create(ctx, g.DB, control); err != nil {
			return nil, fmt.Errorf("create carrier settlement control: %w", err)
		}
	}

	billing, err := g.LoadSetup(ctx)
	if err != nil {
		return nil, err
	}
	setup := &csSetup{control: control, billing: billing}

	accounting, err := g.AccountingControl(ctx)
	if err != nil {
		return nil, err
	}
	if accounting == nil {
		accounting = new(accountingcontrol.AccountingControl)
	}

	setup.expense = pulid.ConvertFromPtr(control.DefaultPurchasedTransportationAccountID)
	if setup.expense.IsNil() {
		setup.expense = accounting.DefaultPurchasedTransportationAccountID
	}
	setup.payable = pulid.ConvertFromPtr(control.DefaultAPAccountID)
	if setup.payable.IsNil() {
		setup.payable = accounting.DefaultAPAccountID
	}
	if setup.expense.IsNil() || setup.payable.IsNil() {
		if err = g.csConfigureAccounts(ctx, setup); err != nil {
			return nil, err
		}
	}

	hasPeriods := len(billing.Periods) > 0
	setup.canPost = hasPeriods && setup.expense.IsNotNil() && setup.payable.IsNotNil()
	setup.canPay = setup.canPost && billing.Cash.IsNotNil()
	if !setup.canPost {
		g.Opts.Logf(
			"  ! carrier settlements stop at Approved: no open fiscal period or AP/expense account",
		)
	}

	return setup, nil
}

func (g *settlementPhase) csConfigureAccounts(ctx context.Context, setup *csSetup) error {
	byCode, err := g.GLAccountsByCode(ctx, csAccountAP, csAccountExpense)
	if err != nil {
		return err
	}

	csc := carriersettlementcontroldb.CarrierSettlementControls
	var sets []keel.Assignment
	if id, ok := byCode[csAccountExpense]; ok && setup.expense.IsNil() {
		sets = append(sets, csc.DefaultPurchasedTransportationAccountID.To(id))
		setup.expense = id
	}
	if id, ok := byCode[csAccountAP]; ok && setup.payable.IsNil() {
		sets = append(sets, csc.DefaultAPAccountID.To(id))
		setup.payable = id
	}
	if len(sets) == 0 {
		return nil
	}
	if _, err = csc.Where(keel.True()).Update(ctx, g.DB, sets...); err != nil {
		return fmt.Errorf("configure carrier settlement accounts: %w", err)
	}

	return nil
}

func (g *settlementPhase) csLoadCarriers(
	ctx context.Context,
	moves []*csMove,
) (map[pulid.ID]*csCarrier, error) {
	ids := make([]pulid.ID, 0, len(moves))
	seen := make(map[pulid.ID]struct{}, len(moves))
	for _, mv := range moves {
		if _, ok := seen[mv.assignment.CarrierID]; ok {
			continue
		}
		seen[mv.assignment.CarrierID] = struct{}{}
		ids = append(ids, mv.assignment.CarrierID)
	}

	cr := carrierdb.Carriers
	rows, err := cr.Where(cr.ID.In(ids...)).All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load carriers: %w", err)
	}

	carriers := make(map[pulid.ID]*csCarrier, len(rows))
	for _, c := range rows {
		carriers[c.ID] = &csCarrier{
			entity:      c,
			nextInvoice: g.Fake.Between(100_000, 800_000),
			nextControl: g.Fake.Between(100_000, 800_000),
		}
	}
	for _, mv := range moves {
		mv.carrier = carriers[mv.assignment.CarrierID]
	}

	return carriers, nil
}

func (g *settlementPhase) csPartners(
	ctx context.Context,
	rows *csRows,
	carriers map[pulid.ID]*csCarrier,
) error {
	codes, err := volumegen.Taken(ctx, g.DB, edidb.EDIPartners.Handle, edidb.EDIPartners.Code)
	if err != nil {
		return err
	}
	names, err := volumegen.Taken(ctx, g.DB, edidb.EDIPartners.Handle, edidb.EDIPartners.Name)
	if err != nil {
		return err
	}

	ids := make([]pulid.ID, 0, len(carriers))
	for id := range carriers {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		c := carriers[id]
		if !g.Fake.Chance(csEDIShare) {
			continue
		}
		created := volumegen.Unix(g.Start) - g.Fake.Between64(10*csDay, 200*csDay)
		name := volumegen.UniqueCode(c.entity.Name+" EDI", names)
		slug := volumegen.Slug(c.entity.Name)
		partner := &edi.EDIPartner{
			ID:                 volumegen.IDAt("edip_", created),
			BusinessUnitID:     g.Refs.BuID,
			OrganizationID:     g.Refs.OrgID,
			Kind:               edi.PartnerKindExternal,
			Status:             domaintypes.StatusActive,
			Code:               volumegen.UniqueCode(c.entity.Code+"-EDI", codes),
			Name:               name,
			Description:        c.entity.Name + " freight invoices (210) in",
			Timezone:           "America/Chicago",
			Country:            "US",
			ContactName:        g.Fake.FirstName() + " " + g.Fake.LastName(),
			ContactEmail:       "edi@" + slug + ".example.com",
			ContactPhone:       g.Fake.Phone(),
			EnabledForInbound:  true,
			EnabledForOutbound: g.Fake.Chance(0.6),
			Settings:           map[string]any{"carrier": map[string]any{"scac": c.entity.SCAC}},
			Version:            1,
		}
		channel := &carrier.CarrierEDIChannel{
			ID:             volumegen.IDAt("cec_", created),
			BusinessUnitID: g.Refs.BuID,
			OrganizationID: g.Refs.OrgID,
			CarrierID:      c.entity.ID,
			EDIPartnerID:   partner.ID,
			IsDefault:      true,
			Version:        1,
		}
		c.partner = partner
		rows.partners = append(rows.partners, partner)
		rows.channels = append(rows.channels, channel)
		rows.stamp(csTablePartners, partner.ID, created, created)
		rows.stamp(csTableChannels, channel.ID, created, created)
	}

	return nil
}

func (g *settlementPhase) csRateConfirmations(rows *csRows, mv *csMove) {
	ca := mv.assignment
	now := volumegen.Unix(g.Now)
	created := mv.assignedAt
	email := mv.carrier.entity.Email
	if email == "" {
		email = "dispatch@" + volumegen.Slug(mv.carrier.entity.Name) + ".example.com"
	}

	newRateCon := func(revision int64, at int64) *rateconfirmation.RateConfirmation {
		return &rateconfirmation.RateConfirmation{
			ID:                  volumegen.IDAt("ratecon_", at),
			BusinessUnitID:      g.Refs.BuID,
			OrganizationID:      g.Refs.OrgID,
			CarrierAssignmentID: ca.ID,
			CarrierID:           ca.CarrierID,
			ShipmentID:          mv.rec.ID,
			ShipmentMoveID:      mv.move.ID,
			Revision:            revision,
			Status:              rateconfirmation.StatusGenerated,
			GeneratedByID:       ca.AssignedByID,
			GeneratedVia:        rateconfirmation.ViaDispatcher,
			Version:             1,
		}
	}
	send := func(rc *rateconfirmation.RateConfirmation, at int64) int64 {
		sentAt := min(at+g.Fake.Between64(2*csMinute, 40*csMinute), now)
		rc.Status = rateconfirmation.StatusSent
		rc.SentAt = &sentAt
		rc.SentToEmails = email
		return sentAt
	}

	first := newRateCon(1, created)
	updated := created
	if ca.ConfirmedAt == nil && g.Fake.Chance(0.2) {
		rows.rateCons = append(rows.rateCons, first)
		rows.stamp(csTableRateCons, first.ID, created, updated)
		return
	}
	updated = send(first, created)
	if ca.ConfirmedAt == nil {
		rows.rateCons = append(rows.rateCons, first)
		rows.stamp(csTableRateCons, first.ID, created, updated)
		return
	}

	confirmedAt := max(*ca.ConfirmedAt, updated)
	active := first
	activeCreated := created
	if confirmedAt-updated > 20*csMinute && g.Fake.Chance(0.06) {
		voidAt := g.Fake.Between64(updated+5*csMinute, confirmedAt-10*csMinute)
		first.Status = rateconfirmation.StatusVoided
		first.VoidedAt = &voidAt
		first.VoidReason = "Rate revised before the carrier signed"
		rows.rateCons = append(rows.rateCons, first)
		rows.stamp(csTableRateCons, first.ID, created, voidAt)

		active = newRateCon(2, voidAt)
		activeCreated = voidAt
		send(active, voidAt)
	}

	active.Status = rateconfirmation.StatusConfirmed
	active.ConfirmedAt = &confirmedAt
	active.ConfirmedByName = g.Fake.FirstName() + " " + g.Fake.LastName()
	active.ConfirmedByTitle = volumegen.Pick(g.Fake, csConfirmerTitles)
	active.ConfirmedVia = rateconfirmation.ViaPublicSignature
	if g.Fake.Chance(0.25) {
		active.ConfirmedVia = rateconfirmation.ViaDispatcher
	}
	rows.rateCons = append(rows.rateCons, active)
	rows.stamp(csTableRateCons, active.ID, activeCreated, confirmedAt)
}

func (g *settlementPhase) csInvoices(
	ctx context.Context,
	rows *csRows,
	setup *csSetup,
	moves []*csMove,
) error {
	dt := edidocumentdb.EDIDocumentTypes
	docType, err := volumegen.FirstOrNil(ctx, g.DB, dt.Where(
		dt.Standard.Eq(edidocument.EDIStandardX12),
		dt.TransactionSet.Eq(edidocument.TransactionSet210),
		dt.Direction.Eq(edidocument.DocumentDirectionInbound),
	))
	if err != nil {
		return fmt.Errorf("load inbound 210 document type: %w", err)
	}
	if docType == nil {
		g.Opts.Logf("  ! no inbound X12 210 document type; carrier invoices skipped")
		return nil
	}

	now := volumegen.Unix(g.Now)
	tolerance := setup.control.VarianceToleranceMinor
	for _, mv := range moves {
		if mv.carrier == nil || mv.carrier.partner == nil ||
			mv.move.Status != shipment.MoveStatusCompleted || mv.move.CompletedAt == 0 ||
			!mv.assignment.IsActive() {
			continue
		}
		received := mv.move.CompletedAt + g.Fake.Between64(6*csHour, 10*csDay)
		if received > now-csHour {
			continue
		}
		g.csInvoice(rows, mv, docType, received, tolerance)
	}

	return nil
}

func (g *settlementPhase) csInvoice(
	rows *csRows,
	mv *csMove,
	docType *edidocument.EDIDocumentType,
	received int64,
	tolerance int64,
) {
	now := volumegen.Unix(g.Now)
	c := mv.carrier
	ca := mv.assignment
	expected := money.MinorUnits(ca.TotalCost)
	linehaul := money.MinorUnits(ca.BaseAmount)
	fuel := money.MinorUnits(ca.FuelSurcharge)

	var variance int64
	extra := ""
	switch roll := g.Fake.Float(); {
	case roll < 0.14:
		variance = g.Fake.Between64(2_500, 35_000)
		extra = volumegen.Pick(g.Fake, []string{"Detention", "Lumper", "Layover", "Driver assist"})
	case roll < 0.18:
		variance = -g.Fake.Between64(1_000, min(5_000, max(linehaul/10, 1_001)))
	}
	total := expected + variance

	prefix := c.entity.SCAC
	if prefix == "" {
		prefix = volumegen.CodeFrom(c.entity.Name, 4)
	}
	c.nextInvoice++
	number := fmt.Sprintf("%s%07d", prefix, c.nextInvoice)
	c.nextControl++
	control := c.nextControl

	charges := []edipayload.FreightInvoiceCharge{{
		Sequence:    1,
		Code:        "400",
		Description: "Linehaul",
		Amount:      money.DecimalFromMinor(linehaul + min(variance, 0)),
	}}
	if fuel != 0 {
		charges = append(charges, edipayload.FreightInvoiceCharge{
			Sequence:    2,
			Code:        "405",
			Description: "Fuel surcharge",
			Amount:      money.DecimalFromMinor(fuel),
		})
	}
	if variance > 0 {
		charges = append(charges, edipayload.FreightInvoiceCharge{
			Sequence:    int64(len(charges) + 1),
			Code:        strings.ToUpper(extra[:3]),
			Description: extra,
			Amount:      money.DecimalFromMinor(variance),
		})
	}

	deliveredAt := mv.move.CompletedAt
	invoiceDate := received - g.Fake.Between64(csHour, 20*csHour)
	totalAmount := decimal.NewNullDecimal(money.DecimalFromMinor(total))
	refs := map[string]string{"scac": c.entity.SCAC}
	payload := &edipayload.FreightInvoicePayload{
		InvoiceNumber:    number,
		InvoiceDate:      invoiceDate,
		DeliveryDate:     deliveredAt,
		BOL:              mv.rec.BOL,
		ProNumber:        ca.ProNumber,
		BillToName:       g.Refs.OrgName,
		CurrencyCode:     money.DefaultCurrencyCode,
		TotalAmount:      totalAmount,
		LineCharges:      charges,
		ReferenceNumbers: refs,
	}
	raw, segments := csRaw210(c, g.Refs.OrgName, payload, control, received)

	message := &edimessage.EDIMessage{
		ID:                       volumegen.IDAt("edimsg_", received),
		BusinessUnitID:           g.Refs.BuID,
		OrganizationID:           g.Refs.OrgID,
		EDIPartnerID:             c.partner.ID,
		DocumentTypeID:           docType.ID,
		Direction:                edidocument.DocumentDirectionInbound,
		Standard:                 edidocument.EDIStandardX12,
		TransactionSet:           edidocument.TransactionSet210,
		X12Version:               docType.DefaultVersion,
		Status:                   edimessage.MessageStatusGenerated,
		InterchangeControlNumber: fmt.Sprintf("%09d", control),
		GroupControlNumber:       fmt.Sprintf("%d", control),
		TransactionControlNumber: "0001",
		SegmentCount:             int64(segments),
		RawX12:                   raw,
		PayloadSnapshot: edipayload.DocumentPayload{
			TransactionSet: edidocument.TransactionSet210,
			FreightInvoice: payload,
		},
		AckStatus:   edimessage.MessageAcknowledgmentStatusNotExpected,
		GeneratedAt: received,
		Version:     1,
	}
	invoice := &edicarrierinvoice.CarrierInvoice{
		ID:                   volumegen.IDAt("edici_", received),
		BusinessUnitID:       g.Refs.BuID,
		OrganizationID:       g.Refs.OrgID,
		EDIPartnerID:         c.partner.ID,
		InboundMessageID:     message.ID,
		InvoiceNumber:        number,
		InvoiceDate:          &invoiceDate,
		DeliveryDate:         &deliveredAt,
		BOL:                  mv.rec.BOL,
		ProNumber:            ca.ProNumber,
		BillToName:           g.Refs.OrgName,
		CurrencyCode:         money.DefaultCurrencyCode,
		TotalAmount:          totalAmount,
		LineCharges:          charges,
		ReferenceNumbers:     refs,
		ReconciliationStatus: edicarrierinvoice.CarrierInvoiceReconciliationStatusUnmatched,
		ReconciliationNotes: fmt.Sprintf(
			"carrier invoice %s could not be matched to an outbound tender",
			number,
		),
		Version: 1,
	}
	rows.messages = append(rows.messages, message)
	rows.invoices = append(rows.invoices, invoice)
	rows.stamp(csTableMessages, message.ID, received, received)
	invoiceUpdated := received

	matchAt := received + g.Fake.Between64(csHour, 2*csDay)
	if matchAt > now-10*csMinute || (received > now-4*csDay && g.Fake.Chance(0.35)) {
		if g.Fake.Chance(0.6) {
			invoice.CarrierID = c.entity.ID
		}
		rows.stamp(csTableInvoices, invoice.ID, received, invoiceUpdated)
		return
	}

	invoice.CarrierID = c.entity.ID
	matcher := g.PickUser(g.Refs.UserIDs)
	invoiceID := invoice.ID
	match := &carriersettlement.InvoiceMatch{
		ID:                  volumegen.IDAt("cim_", matchAt),
		BusinessUnitID:      g.Refs.BuID,
		OrganizationID:      g.Refs.OrgID,
		EDICarrierInvoiceID: &invoiceID,
		CarrierID:           c.entity.ID,
		CarrierAssignmentID: ca.ID,
		Status:              carriersettlement.InvoiceMatchStatusMatched,
		MatchedVia:          carriersettlement.MatchViaManual,
		InvoiceNumber:       number,
		InvoiceTotalMinor:   total,
		ExpectedTotalMinor:  expected,
		VarianceMinor:       variance,
		CurrencyCode:        ca.CurrencyCode,
		Version:             1,
	}
	if abs64(variance) > tolerance {
		match.Status = carriersettlement.InvoiceMatchStatusVariance
	}
	mv.match = match

	invoice.ExpectedAmount = decimal.NewNullDecimal(money.DecimalFromMinor(expected))
	invoice.VarianceAmount = decimal.NewNullDecimal(money.DecimalFromMinor(variance))
	invoice.ReconciliationStatus = edicarrierinvoice.CarrierInvoiceReconciliationStatusMatched
	invoice.ReconciliationNotes = ""
	if match.Status == carriersettlement.InvoiceMatchStatusVariance {
		invoice.ReconciliationStatus = edicarrierinvoice.CarrierInvoiceReconciliationStatusVariance
		invoice.ReconciliationNotes = fmt.Sprintf(
			"carrier invoice %s differs from the tendered amount by %s",
			number,
			money.DecimalFromMinor(variance).String(),
		)
	}
	invoiceUpdated = matchAt
	matchUpdated := matchAt

	age := now - matchAt
	resolveAt := matchAt + g.Fake.Between64(csHour, 3*csDay)
	if resolveAt < now {
		switch match.Status { //nolint:exhaustive // only open match states are resolved here
		case carriersettlement.InvoiceMatchStatusMatched:
			if (age > 5*csDay && g.Fake.Chance(0.92)) || g.Fake.Chance(0.45) {
				g.csResolveMatch(
					match,
					carriersettlement.InvoiceMatchStatusResolved,
					"",
					matcher,
					resolveAt,
				)
				invoice.ReconciliationStatus = edicarrierinvoice.CarrierInvoiceReconciliationStatusMatched
				invoiceUpdated, matchUpdated = resolveAt, resolveAt
			}
		case carriersettlement.InvoiceMatchStatusVariance:
			roll := g.Fake.Float()
			acceptOdds, rejectOdds := 0.3, 0.05
			if age > 5*csDay {
				acceptOdds, rejectOdds = 0.7, 0.2
			}
			switch {
			case roll < acceptOdds:
				g.csResolveMatch(match, carriersettlement.InvoiceMatchStatusResolved,
					"Accepted per the carrier's supporting paperwork", matcher, resolveAt)
				invoice.ReconciliationStatus = edicarrierinvoice.CarrierInvoiceReconciliationStatusMatched
				invoiceUpdated, matchUpdated = resolveAt, resolveAt
				rows.events = append(rows.events, g.csVarianceEvent(mv, match, resolveAt))
			case roll < acceptOdds+rejectOdds:
				g.csResolveMatch(match, carriersettlement.InvoiceMatchStatusRejected,
					volumegen.Pick(g.Fake, csMatchRejectNotes), matcher, resolveAt)
				matchUpdated = resolveAt
			}
		}
	}

	rows.matches = append(rows.matches, match)
	rows.stamp(csTableMatches, match.ID, matchAt, matchUpdated)
	rows.stamp(csTableInvoices, invoice.ID, received, invoiceUpdated)
}

func (g *settlementPhase) csResolveMatch(
	match *carriersettlement.InvoiceMatch,
	status carriersettlement.InvoiceMatchStatus,
	note string,
	userID pulid.ID,
	at int64,
) {
	match.Status = status
	match.ResolutionNote = note
	match.ResolvedByID = userID
	match.ResolvedAt = &at
}

func (g *settlementPhase) csVarianceEvent(
	mv *csMove,
	match *carriersettlement.InvoiceMatch,
	at int64,
) *csEvent {
	assignmentID := match.CarrierAssignmentID
	moveID := mv.move.ID
	shipmentID := mv.rec.ID
	event := &carriersettlement.CostEvent{
		ID:                  volumegen.IDAt("cacc_", at),
		OrganizationID:      g.Refs.OrgID,
		BusinessUnitID:      g.Refs.BuID,
		CarrierID:           match.CarrierID,
		CarrierAssignmentID: &assignmentID,
		ShipmentID:          &shipmentID,
		MoveID:              &moveID,
		EventType:           carriersettlement.CostEventTypeAdjustment,
		Status:              carriersettlement.CostEventStatusPending,
		IdempotencyKey:      "carrier-invoice-variance:" + match.ID.String(),
		EventDate:           at,
		AmountMinor:         match.VarianceMinor,
		CurrencyCode:        match.CurrencyCode,
		Description:         "Invoice variance (" + match.InvoiceNumber + ")",
		Version:             1,
	}
	adjustmentID := event.ID
	match.AdjustmentCostEventID = &adjustmentID

	return &csEvent{entity: event, createdAt: at, updatedAt: at, move: mv, match: match}
}

func (g *settlementPhase) csAccrue(rows *csRows, setup *csSetup, moves []*csMove) []*csEvent {
	for _, mv := range moves {
		ca := mv.assignment
		if !ca.IsActive() || mv.move.Status == shipment.MoveStatusCanceled {
			continue
		}
		accruedAt, eventDate, ok := csTrigger(setup.control.PayTrigger, mv.rec, mv.move)
		if !ok {
			continue
		}
		assignmentID := ca.ID
		moveID := mv.move.ID
		shipmentID := mv.rec.ID
		proNumber := ca.ProNumber
		if proNumber == "" {
			proNumber = mv.rec.ProNumber
		}
		add := func(typ carriersettlement.CostEventType, amount int64, description string) {
			event := &carriersettlement.CostEvent{
				ID:                  volumegen.IDAt("cacc_", accruedAt),
				OrganizationID:      g.Refs.OrgID,
				BusinessUnitID:      g.Refs.BuID,
				CarrierID:           ca.CarrierID,
				CarrierAssignmentID: &assignmentID,
				ShipmentID:          &shipmentID,
				MoveID:              &moveID,
				EventType:           typ,
				Status:              carriersettlement.CostEventStatusPending,
				IdempotencyKey:      "carrier-cost:" + ca.ID.String() + ":" + typ.String() + ":0",
				EventDate:           eventDate,
				AmountMinor:         amount,
				CurrencyCode:        ca.CurrencyCode,
				Description:         description,
				ProNumber:           proNumber,
				AssignmentVersion:   ca.Version,
				Version:             1,
			}
			rows.events = append(rows.events, &csEvent{
				entity:    event,
				createdAt: accruedAt,
				updatedAt: accruedAt,
				move:      mv,
			})
		}
		add(carriersettlement.CostEventTypeLinehaulCost, money.MinorUnits(ca.BaseAmount),
			"Linehaul ("+ca.RateMethod.String()+")")
		if !ca.FuelSurcharge.IsZero() {
			add(carriersettlement.CostEventTypeFuelSurcharge, money.MinorUnits(ca.FuelSurcharge),
				"Fuel surcharge")
		}
	}

	events := rows.events
	sort.SliceStable(
		events,
		func(i, j int) bool { return events[i].createdAt < events[j].createdAt },
	)
	return events
}

func csTrigger(
	trigger settlementcontrol.PayTrigger,
	rec *volumegen.ShipmentRecord,
	mv *volumegen.MoveRecord,
) (accruedAt, eventDate int64, ok bool) {
	delivered := rec.DeliveredAt > 0 && (rec.Status == shipment.StatusCompleted ||
		rec.Status == shipment.StatusReadyToInvoice || rec.Status == shipment.StatusInvoiced)
	switch trigger {
	case settlementcontrol.PayTriggerMoveCompleted:
		if mv.Status == shipment.MoveStatusCompleted && mv.CompletedAt > 0 {
			return mv.CompletedAt, mv.CompletedAt, true
		}
		return 0, 0, false
	case settlementcontrol.PayTriggerPODReceived:
		if !delivered || rec.Status == shipment.StatusCompleted {
			return 0, 0, false
		}
		return max(rec.ReadyToBillAt, rec.DeliveredAt), rec.DeliveredAt, true
	case settlementcontrol.PayTriggerShipmentInvoiced:
		if !delivered || rec.Status != shipment.StatusInvoiced || rec.InvoiceDate == 0 {
			return 0, 0, false
		}
		return rec.InvoiceDate, rec.DeliveredAt, true
	default:
		if !delivered {
			return 0, 0, false
		}
		return rec.DeliveredAt, rec.DeliveredAt, true
	}
}

func (g *settlementPhase) csPeriods(
	control *carriersettlementcontrol.CarrierSettlementControl,
	earliest int64,
) []csPeriod {
	now := g.Now.UTC()
	endDay := time.Weekday(control.PeriodEndDayOfWeek)
	daysBack := int(now.Weekday() - endDay)
	if daysBack < 0 {
		daysBack += 7
	}
	boundary := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, -daysBack+1)

	step := func(t time.Time, back bool) time.Time {
		sign := 1
		if back {
			sign = -1
		}
		switch control.PayPeriodFrequency {
		case settlementcontrol.PayPeriodFrequencyBiweekly:
			return t.AddDate(0, 0, 14*sign)
		case settlementcontrol.PayPeriodFrequencyMonthly:
			return t.AddDate(0, sign, 0)
		default:
			return t.AddDate(0, 0, 7*sign)
		}
	}

	var periods []csPeriod
	for end := boundary; ; end = step(end, true) {
		start := step(end, true)
		periods = append(periods, csPeriod{
			start: start.Unix(),
			end:   end.Unix(),
			pay:   end.AddDate(0, 0, control.PayDelayDays).Unix(),
		})
		if start.Unix() <= earliest {
			break
		}
	}
	sort.Slice(periods, func(i, j int) bool { return periods[i].start < periods[j].start })
	return periods
}

func (g *settlementPhase) csSettle(
	ctx context.Context,
	rows *csRows,
	setup *csSetup,
	events []*csEvent,
) error {
	if len(events) == 0 {
		return nil
	}
	now := volumegen.Unix(g.Now)
	earliest := events[0].entity.EventDate
	for _, ev := range events {
		earliest = min(earliest, ev.entity.EventDate)
	}

	batches, err := g.csOpenBatches(ctx)
	if err != nil {
		return err
	}

	pending := make(map[pulid.ID][]*csEvent)
	for _, ev := range events {
		pending[ev.entity.CarrierID] = append(pending[ev.entity.CarrierID], ev)
	}
	carrierIDs := make([]pulid.ID, 0, len(pending))
	for id := range pending {
		carrierIDs = append(carrierIDs, id)
	}
	sort.Slice(carrierIDs, func(i, j int) bool { return carrierIDs[i] < carrierIDs[j] })

	for _, period := range g.csPeriods(setup.control, earliest) {
		genAt := period.end + g.Fake.Between64(4*csHour, 40*csHour)
		if genAt > now-30*csMinute {
			break
		}
		var batch *carriersettlement.CarrierSettlementBatch
		for _, carrierID := range carrierIDs {
			pool := g.csTake(
				pending,
				carrierID,
				period.end,
				genAt,
				setup.control.HoldUntilInvoiceMatched,
			)
			if len(pool) == 0 {
				continue
			}
			if batch == nil {
				batch = g.csBatch(rows, batches, period, genAt)
			}
			at := genAt
			for {
				settlement := g.csNewSettlement(rows, batch, period, carrierID, pool, at)
				voidAt := g.csLifecycle(settlement, at, period.pay, setup)
				if voidAt == 0 {
					break
				}
				at = voidAt + g.Fake.Between64(csHour, 8*csHour)
				if at > now-10*csMinute {
					for _, ev := range pool {
						ev.entity.Status = carriersettlement.CostEventStatusPending
						ev.entity.SettlementID = nil
						ev.updatedAt = voidAt
						if ev.match != nil {
							ev.match.CarrierSettlementID = nil
						}
					}
					pending[carrierID] = append(pool, pending[carrierID]...)
					break
				}
			}
		}
	}

	return g.csNumberAndPost(ctx, rows, setup)
}

func (g *settlementPhase) csTake(
	pending map[pulid.ID][]*csEvent,
	carrierID pulid.ID,
	periodEnd, genAt int64,
	holdUntilMatched bool,
) []*csEvent {
	var take, keep []*csEvent
	for _, ev := range pending[carrierID] {
		eligible := ev.entity.EventDate < periodEnd && ev.createdAt <= genAt
		if eligible && holdUntilMatched &&
			ev.entity.EventType != carriersettlement.CostEventTypeAdjustment {
			match := ev.move.match
			eligible = match != nil &&
				match.Status == carriersettlement.InvoiceMatchStatusResolved &&
				match.ResolvedAt != nil &&
				*match.ResolvedAt <= genAt
		}
		if eligible {
			take = append(take, ev)
		} else {
			keep = append(keep, ev)
		}
	}
	pending[carrierID] = keep
	return take
}

func (g *settlementPhase) csOpenBatches(
	ctx context.Context,
) (map[string]*carriersettlement.CarrierSettlementBatch, error) {
	csb := carriersettlementdb.CarrierSettlementBatches
	existing, err := csb.Where(csb.Status.Eq(carriersettlement.BatchStatusOpen)).All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load open carrier settlement batches: %w", err)
	}
	batches := make(map[string]*carriersettlement.CarrierSettlementBatch, len(existing))
	for _, b := range existing {
		batches[csPeriodKey(b.PeriodStart, b.PeriodEnd)] = b
	}
	return batches, nil
}

func csPeriodKey(start, end int64) string {
	return fmt.Sprintf("%d|%d", start, end)
}

func (g *settlementPhase) csBatch(
	rows *csRows,
	batches map[string]*carriersettlement.CarrierSettlementBatch,
	period csPeriod,
	genAt int64,
) *carriersettlement.CarrierSettlementBatch {
	key := csPeriodKey(period.start, period.end)
	if batch, ok := batches[key]; ok {
		rows.touched[batch.ID] = struct{}{}
		return batch
	}
	generatedAt := genAt
	batch := &carriersettlement.CarrierSettlementBatch{
		ID:             volumegen.IDAt("carstlb_", genAt),
		BusinessUnitID: g.Refs.BuID,
		OrganizationID: g.Refs.OrgID,
		Status:         carriersettlement.BatchStatusOpen,
		Name: "Carrier pay period ending " +
			time.Unix(period.end, 0).UTC().AddDate(0, 0, -1).Format("Jan 2, 2006"),
		PeriodStart:   period.start,
		PeriodEnd:     period.end,
		PayDate:       period.pay,
		CurrencyCode:  money.DefaultCurrencyCode,
		GeneratedByID: g.PickUser(g.Refs.UserIDs),
		GeneratedAt:   &generatedAt,
		Version:       1,
	}
	batches[key] = batch
	rows.batches = append(rows.batches, batch)
	rows.touched[batch.ID] = struct{}{}
	rows.stamp(csTableBatches, batch.ID, genAt, genAt)
	return batch
}

func (g *settlementPhase) csNewSettlement(
	rows *csRows,
	batch *carriersettlement.CarrierSettlementBatch,
	period csPeriod,
	carrierID pulid.ID,
	pool []*csEvent,
	genAt int64,
) *csSettlement {
	batchID := batch.ID
	entity := &carriersettlement.CarrierSettlement{
		ID:             volumegen.IDAt("carstl_", genAt),
		BusinessUnitID: g.Refs.BuID,
		OrganizationID: g.Refs.OrgID,
		CarrierID:      carrierID,
		BatchID:        &batchID,
		Status:         carriersettlement.StatusDraft,
		PeriodStart:    period.start,
		PeriodEnd:      period.end,
		PayDate:        period.pay,
		CurrencyCode:   money.DefaultCurrencyCode,
		Version:        1,
	}

	sort.SliceStable(
		pool,
		func(i, j int) bool { return pool[i].entity.EventDate < pool[j].entity.EventDate },
	)
	shipments := make(map[pulid.ID]struct{}, len(pool))
	for _, ev := range pool {
		e := ev.entity
		eventID := e.ID
		if e.ShipmentID != nil {
			shipments[*e.ShipmentID] = struct{}{}
		}
		description := e.Description
		if description == "" {
			description = e.EventType.String()
		}
		entity.Lines = append(entity.Lines, &carriersettlement.CarrierSettlementLine{
			ID:             volumegen.IDAt("carstll_", genAt),
			BusinessUnitID: g.Refs.BuID,
			OrganizationID: g.Refs.OrgID,
			SettlementID:   entity.ID,
			EventType:      e.EventType,
			Description:    description,
			AmountMinor:    e.AmountMinor,
			CostEventID:    &eventID,
			ShipmentID:     e.ShipmentID,
			MoveID:         e.MoveID,
			ProNumber:      e.ProNumber,
		})
		settlementID := entity.ID
		e.Status = carriersettlement.CostEventStatusAttached
		e.SettlementID = &settlementID
		ev.updatedAt = genAt
		if ev.match != nil {
			ev.match.CarrierSettlementID = &settlementID
		}
	}
	entity.ShipmentCount = len(shipments)
	entity.SyncTotals()

	if g.Fake.Chance(0.08) {
		kind := volumegen.Pick(g.Fake, csAdjustmentKinds)
		amount := g.Fake.Between64(kind.low, kind.high)
		if kind.negative {
			amount = -min(amount, entity.GrossCostMinor/3)
		}
		if amount != 0 {
			entity.Lines = append(entity.Lines, &carriersettlement.CarrierSettlementLine{
				ID:             volumegen.IDAt("carstll_", genAt),
				BusinessUnitID: g.Refs.BuID,
				OrganizationID: g.Refs.OrgID,
				SettlementID:   entity.ID,
				EventType:      carriersettlement.CostEventTypeAdjustment,
				Description:    kind.description,
				AmountMinor:    amount,
			})
			entity.SyncTotals()
		}
	}

	settlement := &csSettlement{entity: entity, events: pool, createdAt: genAt, updatedAt: genAt}
	rows.settlements = append(rows.settlements, settlement)
	for _, line := range entity.Lines {
		line.SettlementID = entity.ID
		rows.lines = append(rows.lines, line)
	}
	return settlement
}

// csLifecycle walks a generated settlement through review, posting and
// payment as far as the clock allows, and returns the void time when the
// settlement is voided so its cost events can be regenerated.
func (g *settlementPhase) csLifecycle(
	s *csSettlement,
	genAt, payDate int64,
	setup *csSetup,
) int64 {
	now := volumegen.Unix(g.Now)
	entity := s.entity
	users := g.Refs.UserIDs

	if g.Fake.Chance(0.025) {
		voidAt := genAt + g.Fake.Between64(csHour, 30*csHour)
		if voidAt <= now-5*csMinute {
			if g.Fake.Chance(0.5) {
				submitAt := g.Fake.Between64(
					genAt+10*csMinute,
					max(voidAt-5*csMinute, genAt+10*csMinute),
				)
				entity.SubmittedByID = g.PickUser(users)
				entity.SubmittedAt = &submitAt
			}
			entity.Status = carriersettlement.StatusVoided
			entity.VoidedByID = g.PickUser(users)
			entity.VoidedAt = &voidAt
			entity.VoidReason = volumegen.Pick(g.Fake, csVoidReasons)
			s.updatedAt = voidAt
			s.replaced = true
			return voidAt
		}
	}

	submitAt := genAt + g.Fake.Between64(2*csHour, 30*csHour)
	if g.Fake.Chance(0.05) {
		rejectAt := submitAt + g.Fake.Between64(csHour, 20*csHour)
		if rejectAt < now {
			entity.Notes = "Rejected: " + volumegen.Pick(g.Fake, csRejectReasons)
			s.updatedAt = rejectAt
			submitAt = rejectAt + g.Fake.Between64(2*csHour, 24*csHour)
		}
	}
	if submitAt > now {
		return 0
	}
	entity.Status = carriersettlement.StatusPendingApproval
	entity.SubmittedByID = g.PickUser(users)
	entity.SubmittedAt = &submitAt
	s.updatedAt = submitAt

	approveAt := submitAt + g.Fake.Between64(csHour, 36*csHour)
	if approveAt > now {
		return 0
	}
	approver := g.PickUser(users)
	entity.Status = carriersettlement.StatusApproved
	entity.ApprovedByID = approver
	entity.ApprovedAt = &approveAt
	s.updatedAt = approveAt

	if !setup.canPost {
		return 0
	}
	postAt := approveAt
	if !setup.control.AutoPostOnApprove {
		postAt += g.Fake.Between64(10*csMinute, 20*csHour)
	}
	if postAt > now {
		return 0
	}
	entity.Status = carriersettlement.StatusPosted
	entity.PostedByID = approver
	entity.PostedAt = &postAt
	expense, payable := setup.expense, setup.payable
	entity.PostedExpenseAccountID = &expense
	entity.PostedAPAccountID = &payable
	s.updatedAt = postAt

	if !setup.canPay || g.Fake.Chance(0.03) {
		return 0
	}
	paidAt := max(payDate+g.Fake.Between64(-12*csHour, 2*csDay), postAt+2*csHour)
	if paidAt > now {
		return 0
	}
	entity.Status = carriersettlement.StatusPaid
	entity.PaidAt = &paidAt
	entity.PaidByID = g.PickUser(users)
	s.updatedAt = paidAt

	return 0
}

func (g *settlementPhase) csNumberAndPost(ctx context.Context, rows *csRows, setup *csSetup) error {
	if len(rows.settlements) == 0 {
		return nil
	}
	slots := make([]volumegen.NumberSlot, len(rows.settlements))
	for i, s := range rows.settlements {
		slots[i] = volumegen.NumberSlot{At: s.createdAt}
	}
	numbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeCarrierSettlement, slots)
	if err != nil {
		return err
	}

	carriers := make(map[pulid.ID]*carrier.Carrier)
	for _, ev := range rows.events {
		if ev.move != nil && ev.move.carrier != nil {
			carriers[ev.move.carrier.entity.ID] = ev.move.carrier.entity
		}
	}

	for i, s := range rows.settlements {
		entity := s.entity
		entity.SettlementNumber = numbers[i]
		for _, line := range entity.Lines {
			rows.stamp(csTableLines, line.ID, s.createdAt, s.createdAt)
		}

		if !s.replaced {
			for _, ev := range s.events {
				if entity.PostedAt != nil {
					ev.entity.Status = carriersettlement.CostEventStatusSettled
					ev.updatedAt = *entity.PostedAt
				}
			}
		}

		if entity.PostedAt != nil && entity.NetPayableMinor > 0 {
			if err = g.csPostJournal(rows, setup, s); err != nil {
				return err
			}
		}
		if entity.PaidAt != nil && entity.NetPayableMinor > 0 {
			if err = g.csPayJournal(rows, setup, s, carriers[entity.CarrierID]); err != nil {
				return err
			}
		}
		rows.stamp(csTableSettlements, entity.ID, s.createdAt, s.updatedAt)
	}

	return nil
}

func (g *settlementPhase) csPostJournal(rows *csRows, setup *csSetup, s *csSettlement) error {
	entity := s.entity
	postAt := *entity.PostedAt
	journal, err := g.NewJournal(setup.billing, postAt)
	if err != nil {
		return err
	}
	event := accountingcontrol.JournalSourceEventCarrierSettlementPosted
	journal.ActorID = entity.PostedByID
	journal.Description = "Carrier settlement " + entity.SettlementNumber
	journal.SourceType = csSourceType
	journal.SourceID = entity.ID
	journal.Document = entity.SettlementNumber
	journal.Event = event
	journal.ReferenceType = event.String()
	journal.Idempotency = "carrier-settlement-posted:" + entity.ID.String()
	journal.Auto = true
	journal.Legs = []ledgervolume.Leg{
		{Account: setup.expense, Debit: entity.NetPayableMinor},
		{Account: setup.payable, Credit: entity.NetPayableMinor},
	}
	rows.billing.Journals = append(rows.billing.Journals, journal)
	batchID := journal.BatchID
	entity.PostedJournalBatchID = &batchID

	g.csLedger(rows, entity, carriersettlement.LedgerEntryTypeBill, event, &batchID,
		entity.NetPayableMinor, postAt, entity.PostedByID)
	return nil
}

func (g *settlementPhase) csPayJournal(
	rows *csRows,
	setup *csSetup,
	s *csSettlement,
	c *carrier.Carrier,
) error {
	entity := s.entity
	paidAt := *entity.PaidAt
	journal, err := g.NewJournal(setup.billing, paidAt)
	if err != nil {
		return err
	}
	method := carrier.PaymentMethodCheck
	if c != nil && c.PaymentMethod != "" {
		method = c.PaymentMethod
	}
	entity.PaymentMethod = string(method)
	if method == carrier.PaymentMethodACHManual {
		entity.PaymentReference = "ACH" + g.Fake.Digits(15)
	} else {
		entity.PaymentReference = "CHK " + g.Fake.Digits(6)
	}

	event := accountingcontrol.JournalSourceEventCarrierSettlementPaid
	journal.ActorID = entity.PaidByID
	journal.Description = "Payment of carrier settlement " + entity.SettlementNumber
	journal.SourceType = csSourceType
	journal.SourceID = entity.ID
	journal.Document = entity.SettlementNumber
	journal.Event = event
	journal.ReferenceType = event.String()
	journal.Idempotency = "carrier-settlement-paid:" + entity.ID.String()
	journal.Auto = true
	journal.Legs = []ledgervolume.Leg{
		{Account: setup.payable, Debit: entity.NetPayableMinor},
		{Account: setup.billing.Cash, Credit: entity.NetPayableMinor},
	}
	rows.billing.Journals = append(rows.billing.Journals, journal)
	batchID := journal.BatchID
	entity.PaidJournalBatchID = &batchID

	g.csLedger(rows, entity, carriersettlement.LedgerEntryTypePayment, event, &batchID,
		-entity.NetPayableMinor, paidAt, entity.PaidByID)
	return nil
}

func (g *settlementPhase) csLedger(
	rows *csRows,
	entity *carriersettlement.CarrierSettlement,
	entryType carriersettlement.LedgerEntryType,
	event accountingcontrol.JournalSourceEventType,
	batchID *pulid.ID,
	amount, at int64,
	actor pulid.ID,
) {
	settlementID := entity.ID
	if actor.IsNil() {
		actor = g.Refs.AdminID
	}
	entry := &carriersettlement.LedgerEntry{
		ID:                  volumegen.IDAt("carldg_", at),
		BusinessUnitID:      g.Refs.BuID,
		OrganizationID:      g.Refs.OrgID,
		CarrierID:           entity.CarrierID,
		EntryType:           entryType,
		SourceObjectType:    csSourceType,
		SourceObjectID:      entity.ID.String(),
		SourceEventType:     event.String(),
		RelatedSettlementID: &settlementID,
		JournalBatchID:      batchID,
		DocumentNumber:      entity.SettlementNumber,
		TransactionDate:     at,
		LineNumber:          1,
		AmountMinor:         amount,
		CreatedByID:         actor,
		CreatedAt:           at,
	}
	rows.ledger = append(rows.ledger, entry)
	rows.stamp(csTableLedger, entry.ID, at, at)
}

//nolint:gocognit,funlen // the writes follow the foreign keys between these tables
func (g *settlementPhase) csWrite(ctx context.Context, tx *keel.DB, rows *csRows) error {
	settlements := make([]*carriersettlement.CarrierSettlement, len(rows.settlements))
	for i, s := range rows.settlements {
		settlements[i] = s.entity
	}
	events := make([]*carriersettlement.CostEvent, len(rows.events))
	for i, ev := range rows.events {
		events[i] = ev.entity
		rows.stamp(csTableEvents, ev.entity.ID, ev.createdAt, max(ev.updatedAt, ev.createdAt))
	}

	steps := []struct {
		table string
		run   func() error
	}{
		{
			csTablePartners,
			func() error { return volumegen.InsertRows(ctx, tx, edidb.EDIPartners.Handle, rows.partners) },
		},
		{
			csTableChannels,
			func() error { return volumegen.InsertRows(ctx, tx, carrierdb.CarrierEDIChannels.Handle, rows.channels) },
		},
		{
			csTableMessages,
			func() error { return volumegen.InsertRows(ctx, tx, edimessagedb.EDIMessages.Handle, rows.messages) },
		},
		{
			csTableInvoices,
			func() error {
				return volumegen.InsertRows(
					ctx,
					tx,
					edicarrierinvoicedb.CarrierInvoices.Handle,
					rows.invoices,
				)
			},
		},
		{
			csTableRateCons,
			func() error {
				return volumegen.InsertRows(
					ctx,
					tx,
					rateconfirmationdb.RateConfirmations.Handle,
					rows.rateCons,
				)
			},
		},
		{csTableBatches, func() error {
			return volumegen.InsertRows(
				ctx,
				tx,
				carriersettlementdb.CarrierSettlementBatches.Handle,
				rows.batches,
			)
		}},
		{
			csTableSettlements,
			func() error {
				return volumegen.InsertRows(
					ctx,
					tx,
					carriersettlementdb.CarrierSettlements.Handle,
					settlements,
				)
			},
		},
		{
			csTableEvents,
			func() error { return volumegen.InsertRows(ctx, tx, carriersettlementdb.CostEvents.Handle, events) },
		},
		{csTableLines, func() error {
			return volumegen.InsertRows(
				ctx,
				tx,
				carriersettlementdb.CarrierSettlementLines.Handle,
				rows.lines,
			)
		}},
		{
			csTableMatches,
			func() error {
				return volumegen.InsertRows(
					ctx,
					tx,
					carriersettlementdb.InvoiceMatches.Handle,
					rows.matches,
				)
			},
		},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("insert %s: %w", step.table, err)
		}
	}

	if err := g.Write(ctx, tx, rows.billing); err != nil {
		return err
	}
	if err := volumegen.InsertRows(
		ctx,
		tx,
		carriersettlementdb.LedgerEntries.Handle,
		rows.ledger,
	); err != nil {
		return fmt.Errorf("insert %s: %w", csTableLedger, err)
	}

	for _, table := range []string{
		csTablePartners, csTableChannels, csTableMessages, csTableInvoices, csTableRateCons,
		csTableBatches, csTableSettlements, csTableEvents, csTableLines, csTableMatches,
	} {
		if err := volumegen.BackdateTable(ctx, tx, table, rows.stamps[table]); err != nil {
			return err
		}
	}
	if err := volumegen.BackdateTableCreated(ctx, tx, csTableLedger, rows.stamps[csTableLedger]); err != nil {
		return err
	}

	return g.csBatchAggregates(ctx, tx, rows.touched)
}

func (g *settlementPhase) csBatchAggregates(
	ctx context.Context,
	tx *keel.DB,
	touched map[pulid.ID]struct{},
) error {
	if len(touched) == 0 {
		return nil
	}
	batchIDs := make([]pulid.ID, 0, len(touched))
	for id := range touched {
		batchIDs = append(batchIDs, id)
	}

	type batchTotals struct {
		BatchID pulid.ID
		Count   int64
		Gross   int64
		Net     int64
	}
	cs := carriersettlementdb.CarrierSettlements
	totals, err := keel.Into[batchTotals](ctx, tx, cs.Where(
		cs.BatchID.In(batchIDs...),
		cs.Status.Ne(carriersettlement.StatusVoided),
	).Group(cs.BatchID).Pick(
		cs.BatchID,
		keel.Count(),
		keel.CastInt64(keel.Int64Of(cs.GrossCostMinor.Sum()).Coalesce(0)),
		keel.CastInt64(keel.Int64Of(cs.NetPayableMinor.Sum()).Coalesce(0)),
	))
	if err != nil {
		return fmt.Errorf("total carrier settlement batches: %w", err)
	}
	byBatch := make(map[pulid.ID]batchTotals, len(totals))
	for _, t := range totals {
		byBatch[t.BatchID] = t
	}

	ids := make([]string, len(batchIDs))
	counts := make([]int64, len(batchIDs))
	gross := make([]int64, len(batchIDs))
	net := make([]int64, len(batchIDs))
	for i, id := range batchIDs {
		t := byBatch[id]
		ids[i], counts[i], gross[i], net[i] = id.String(), t.Count, t.Gross, t.Net
	}

	csb := carriersettlementdb.CarrierSettlementBatches
	d := keel.Unnest("d", []string{"id", "settlements", "gross", "net"}, ids, counts, gross, net)
	if _, err = csb.With(d).Join(d, d.Col("id").EqCol(csb.ID)).Where(keel.True()).Update(
		ctx,
		tx,
		csb.SettlementCount.ToExpr(d.Col("settlements")),
		csb.TotalGrossMinor.ToExpr(d.Col("gross")),
		csb.TotalNetMinor.ToExpr(d.Col("net")),
		volumegen.KeepUpdated(csb.Handle),
	); err != nil {
		return fmt.Errorf("recalculate carrier settlement batch aggregates: %w", err)
	}

	return nil
}

func (g *settlementPhase) csReport(rows *csRows) {
	statuses := make(map[carriersettlement.Status]int)
	for _, s := range rows.settlements {
		statuses[s.entity.Status]++
	}
	eventStatuses := make(map[carriersettlement.CostEventStatus]int)
	for _, ev := range rows.events {
		eventStatuses[ev.entity.Status]++
	}

	g.Report.Add("Carrier EDI partners", len(rows.partners))
	g.Report.Add("Carrier EDI 210 messages", len(rows.messages))
	g.Report.Add("Carrier invoices (EDI 210)", len(rows.invoices))
	g.Report.Add("Rate confirmations", len(rows.rateCons))
	g.Report.Add("Carrier cost events", len(rows.events))
	g.Report.Add("Carrier invoice matches", len(rows.matches))
	g.Report.Add("Carrier settlement batches", len(rows.batches))
	g.Report.Add("Carrier settlements", len(rows.settlements))
	g.Report.Add("Carrier settlement lines", len(rows.lines))
	g.Report.Add("Carrier ledger entries", len(rows.ledger))
	g.Report.Add("Carrier settlement journals", len(rows.billing.Entries))

	for _, status := range []carriersettlement.Status{
		carriersettlement.StatusDraft, carriersettlement.StatusPendingApproval,
		carriersettlement.StatusApproved, carriersettlement.StatusPosted,
		carriersettlement.StatusPaid, carriersettlement.StatusVoided,
	} {
		g.Opts.Logf("  • carrier settlements %s: %d", status, statuses[status])
	}
	for _, status := range []carriersettlement.CostEventStatus{
		carriersettlement.CostEventStatusPending, carriersettlement.CostEventStatusAttached,
		carriersettlement.CostEventStatusSettled,
	} {
		g.Opts.Logf("  • carrier cost events %s: %d", status, eventStatuses[status])
	}
}

func csRaw210(
	c *csCarrier,
	receiver string,
	p *edipayload.FreightInvoicePayload,
	control int,
	received int64,
) (string, int) {
	at := time.Unix(received, 0).UTC()
	sender := c.entity.SCAC
	if sender == "" {
		sender = volumegen.CodeFrom(c.entity.Name, 4)
	}
	receiverID := volumegen.CodeFrom(receiver, 8)
	totalCents := money.MinorUnits(p.TotalAmount.Decimal)

	body := []string{
		"ST*210*0001",
		fmt.Sprintf("B3**%s*%s*PP**%s*%d**%s*035*%s",
			p.InvoiceNumber, p.ProNumber, at.Format("20060102"), totalCents,
			time.Unix(p.DeliveryDate, 0).UTC().Format("20060102"), sender),
		"N9*BM*" + p.BOL,
		"N1*BT*" + p.BillToName,
	}
	for _, charge := range p.LineCharges {
		body = append(body,
			fmt.Sprintf("LX*%d", charge.Sequence),
			fmt.Sprintf("L1*%d**%s*%d*%s",
				charge.Sequence, charge.Code, money.MinorUnits(charge.Amount), charge.Description),
		)
	}
	body = append(body, fmt.Sprintf("L3*****%d", totalCents))
	body = append(body, fmt.Sprintf("SE*%d*0001", len(body)+1))

	segments := []string{
		fmt.Sprintf("ISA*00*          *00*          *02*%-15s*ZZ*%-15s*%s*%s*U*00401*%09d*0*P*>",
			sender, receiverID, at.Format("060102"), at.Format("1504"), control),
		fmt.Sprintf("GS*IM*%s*%s*%s*%s*%d*X*004010",
			sender, receiverID, at.Format("20060102"), at.Format("1504"), control),
	}
	segments = append(segments, body...)
	segments = append(segments,
		fmt.Sprintf("GE*1*%d", control),
		fmt.Sprintf("IEA*1*%09d", control),
	)

	return strings.Join(segments, "~\n") + "~", len(body)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

type settlementPhase struct {
	*ledgervolume.Ledger
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&settlementPhase{Ledger: &ledgervolume.Ledger{Generator: g}}).seedCarrierSettlements(
		ctx,
	)
}
