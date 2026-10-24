package volume

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/keelkinds"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	bilChunkSize        = 1_000
	bilMinute           = int64(60)
	bilHour             = int64(3_600)
	bilDay              = int64(86_400)
	bilWeek             = 7 * bilDay
	bilMaxApplications  = 12
	bilFreightLine      = "Freight charge"
	bilAccessorialLine  = "Accessorial charge"
	bilCountry          = "United States"
	bilAccountCash      = "1010"
	bilAccountAR        = "1110"
	bilAccountRevenue   = "4010"
	bilAccountUnapplied = "2160"
	bilStatusPosted     = "Posted"
	bilBatchTypeSystem  = "System"
)

var bilExceptionNotes = map[billingqueue.ExceptionReasonCode]string{
	billingqueue.ExceptionMissingDocumentation:     "Signed POD missing from the document packet",
	billingqueue.ExceptionIncorrectRates:           "Linehaul does not match the customer's contract rate",
	billingqueue.ExceptionWeightDiscrepancy:        "Scale ticket weight differs from the BOL by more than 5%",
	billingqueue.ExceptionAccessorialDispute:       "Customer disputes detention at the consignee",
	billingqueue.ExceptionMissingReferenceNumber:   "Customer requires a PO number on every invoice",
	billingqueue.ExceptionCustomerInformationError: "Bill-to address on file is out of date",
	billingqueue.ExceptionServiceFailure:           "Late delivery reported; customer asked to hold billing",
	billingqueue.ExceptionRateNotOnFile:            "No rate on file for this lane",
}

var bilExceptionCodes = []billingqueue.ExceptionReasonCode{
	billingqueue.ExceptionMissingDocumentation,
	billingqueue.ExceptionIncorrectRates,
	billingqueue.ExceptionWeightDiscrepancy,
	billingqueue.ExceptionAccessorialDispute,
	billingqueue.ExceptionMissingReferenceNumber,
	billingqueue.ExceptionCustomerInformationError,
	billingqueue.ExceptionServiceFailure,
	billingqueue.ExceptionRateNotOnFile,
}

var bilHoldNotes = map[billingqueue.HoldReasonCode]string{
	billingqueue.HoldReasonWaitingOnPaperwork: "Waiting on the signed delivery receipt from the consignee",
	billingqueue.HoldReasonCustomerDispute:    "Customer is reviewing the lumper charge before we bill",
	billingqueue.HoldReasonRateQuestion:       "Confirming the spot rate with the account manager",
}

var bilHoldCodes = []billingqueue.HoldReasonCode{
	billingqueue.HoldReasonWaitingOnPaperwork,
	billingqueue.HoldReasonCustomerDispute,
	billingqueue.HoldReasonRateQuestion,
}

var bilSentBackReasons = []struct {
	code billingqueue.ExceptionReasonCode
	note string
}{
	{billingqueue.ExceptionMissingDocumentation, "Stop times missing on delivery; please update actuals"},
	{billingqueue.ExceptionAccessorialDispute, "Detention not entered though the driver reported three hours"},
	{billingqueue.ExceptionMissingReferenceNumber, "BOL number on the shipment does not match the paperwork"},
	{billingqueue.ExceptionCustomerInformationError, "Delivery appointment shows the wrong consignee"},
}

var bilMemoReasons = []string{
	"Allowance for product damaged in transit",
	"Detention billed in error",
	"Rate adjustment agreed with the customer",
	"Duplicate lumper charge credited",
	"Service failure credit for late delivery",
}

type bilSetup struct {
	postInvoices bool
	postPayments bool
	ar           pulid.ID
	revenue      pulid.ID
	cash         pulid.ID
	unapplied    pulid.ID
	applied      pulid.ID
	periods      []*fiscalperiod.FiscalPeriod
	names        map[pulid.ID]string
}

type bilLeg struct {
	account  pulid.ID
	debit    int64
	credit   int64
	customer pulid.ID
}

type bilJournal struct {
	batchID       pulid.ID
	entryID       pulid.ID
	period        *fiscalperiod.FiscalPeriod
	date          int64
	actorID       pulid.ID
	description   string
	sourceType    string
	sourceID      pulid.ID
	document      string
	event         tenant.JournalSourceEventType
	referenceType string
	idempotency   string
	auto          bool
	legs          []bilLeg
}

type bilBalanceKey struct {
	account pulid.ID
	year    pulid.ID
	period  pulid.ID
}

type bilBalance struct {
	debit     int64
	credit    int64
	net       int64
	lastEntry pulid.ID
	lastDate  int64
}

type bilItemInvoice struct {
	item    pulid.ID
	invoice pulid.ID
}

type bilRows struct {
	items     []*billingqueue.BillingQueueItem
	events    []*billingqueue.ItemEvent
	invoices  []*invoice.Invoice
	lines     []*invoice.InvoiceLine
	links     []bilItemInvoice
	memoApps  []*customerpayment.CreditMemoApplication
	payments  []*customerpayment.Payment
	apps      []*customerpayment.Application
	ledger    []*customerledger.CustomerLedgerEntry
	journals  []*bilJournal
	batches   []*journalentry.JournalBatch
	entries   []*journalentry.JournalEntry
	jlines    []*journalentry.JournalEntryLine
	sources   []*journalsource.Source
	srcLinks  []*journalsource.SourceJournalLink
	comments  []*shipment.ShipmentComment
	balances  map[bilBalanceKey]*bilBalance
	memoCount int
}

type bilPaymentPlan struct {
	invoice *invoice.Invoice
	amount  int64
	date    int64
}

type bilPaymentGroup struct {
	customer *customerRecord
	key      string
	date     int64
	plans    []*bilPaymentPlan
}

func (g *generator) seedBilling(ctx context.Context) error {
	records := make([]*shipmentRecord, 0, len(g.shipments))
	for _, r := range g.shipments {
		if r.Customer == nil {
			continue
		}
		if r.Stage == billingStageQueued || r.Stage == billingStageInvoiced {
			records = append(records, r)
		}
	}
	if len(records) == 0 {
		return nil
	}

	setup, err := g.bilLoadSetup(ctx)
	if err != nil {
		return err
	}

	sort.SliceStable(records, func(i, j int) bool {
		return g.bilTransferAt(records[i]) < g.bilTransferAt(records[j])
	})

	for begin := 0; begin < len(records); begin += bilChunkSize {
		chunk := records[begin:min(begin+bilChunkSize, len(records))]
		rows, buildErr := g.bilBuild(ctx, setup, chunk)
		if buildErr != nil {
			return buildErr
		}
		if err = g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
			return g.bilWrite(ctx, tx, rows)
		}); err != nil {
			return err
		}
		g.bilCount(rows)
		g.progress("billing", min(begin+bilChunkSize, len(records)), len(records))
	}

	return nil
}

func (g *generator) bilLoadSetup(ctx context.Context) (*bilSetup, error) {
	setup := &bilSetup{names: make(map[pulid.ID]string, len(g.refs.userIDs))}

	users, err := g.usersByID(ctx)
	if err != nil {
		return nil, err
	}
	for id, u := range users {
		setup.names[id] = u.Name
	}

	control, err := g.accountingControl(ctx)
	if err != nil {
		return nil, err
	}
	if control == nil {
		g.opts.Logf("  ! no accounting control; invoices post without journals or ledger entries")
		return setup, nil
	}

	if err := g.bilConfigureAccounts(ctx, control); err != nil {
		return nil, err
	}
	setup.ar = control.DefaultARAccountID
	setup.revenue = control.DefaultRevenueAccountID
	setup.cash = control.DefaultCashAccountID
	setup.unapplied = control.DefaultUnappliedCashAccountID

	if err := g.bilEnsureFiscalYears(ctx); err != nil {
		return nil, err
	}
	fp := fiscalperioddb.FiscalPeriods
	if setup.periods, err = fp.Where(
		fp.Status.Eq(fiscalperiod.StatusOpen),
		keel.Not(fp.IsAdjusting.IsTrue()),
	).Order(fp.StartDate.Asc()).All(ctx, g.db); err != nil {
		return nil, fmt.Errorf("load fiscal periods: %w", err)
	}
	if len(setup.periods) == 0 {
		g.opts.Logf("  ! no open fiscal periods; invoices post without journals or ledger entries")
		return setup, nil
	}

	accrual := control.AccountingBasis != tenant.AccountingBasisCash
	onInvoice := control.RevenueRecognitionPolicy == tenant.RevenueRecognitionOnInvoicePost
	setup.postInvoices = accrual && onInvoice && setup.ar.IsNotNil() && setup.revenue.IsNotNil()
	setup.applied = setup.ar
	if !accrual || !onInvoice {
		setup.applied = setup.revenue
	}
	setup.postPayments = setup.cash.IsNotNil() && setup.applied.IsNotNil()

	return setup, nil
}

// bilEnsureFiscalYears opens a calendar fiscal year for every year the history
// reaches back into, so old invoices post on their own dates instead of piling
// onto the first open period.
func (g *generator) bilEnsureFiscalYears(ctx context.Context) error {
	fy := fiscalyeardb.FiscalYears
	existing, err := fy.Pick(fy.Year).All(ctx, g.db)
	if err != nil {
		return fmt.Errorf("load fiscal years: %w", err)
	}
	have := make(map[int]struct{}, len(existing))
	for _, row := range existing {
		have[row.Year] = struct{}{}
	}

	return g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		for year := g.start.Year(); year <= g.now.Year(); year++ {
			if _, ok := have[year]; ok {
				continue
			}
			begin := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
			fy := &fiscalyear.FiscalYear{
				OrganizationID: g.refs.orgID,
				BusinessUnitID: g.refs.buID,
				Status:         fiscalyear.StatusOpen,
				Year:           year,
				Name:           "FY " + strconv.Itoa(year),
				Description:    "Calendar fiscal year opened for volume seed history",
				StartDate:      begin.Unix(),
				EndDate:        begin.AddDate(1, 0, 0).Add(-time.Second).Unix(),
				IsCalendarYear: true,
			}
			if err := fiscalyeardb.FiscalYears.Create(ctx, tx, fy); err != nil {
				return fmt.Errorf("insert fiscal year %d: %w", year, err)
			}
			if err := fiscalperioddb.FiscalPeriods.CreateMany(ctx, tx, fy.GenerateMonthlyPeriods()); err != nil {
				return fmt.Errorf("insert fiscal periods for %d: %w", year, err)
			}
			g.report.add("Fiscal years", 1)
		}
		return nil
	})
}

func (g *generator) bilConfigureAccounts(
	ctx context.Context,
	control *tenant.AccountingControl,
) error {
	byCode, err := g.glAccountsByCode(ctx, bilAccountCash, bilAccountAR, bilAccountRevenue, bilAccountUnapplied)
	if err != nil {
		return err
	}

	ac := tenantdb.AccountingControls
	defaults := []struct {
		column keelkinds.PulidIDCol
		code   string
		field  *pulid.ID
	}{
		{column: ac.DefaultARAccountID, code: bilAccountAR, field: &control.DefaultARAccountID},
		{column: ac.DefaultRevenueAccountID, code: bilAccountRevenue, field: &control.DefaultRevenueAccountID},
		{column: ac.DefaultCashAccountID, code: bilAccountCash, field: &control.DefaultCashAccountID},
		{
			column: ac.DefaultUnappliedCashAccountID,
			code:   bilAccountUnapplied,
			field:  &control.DefaultUnappliedCashAccountID,
		},
	}
	for _, def := range defaults {
		id, ok := byCode[def.code]
		if def.field.IsNotNil() || !ok {
			continue
		}
		if _, err = ac.Where(def.column.IsNull()).Update(ctx, g.db, def.column.To(id)); err != nil {
			return fmt.Errorf("set the default %s account: %w", def.code, err)
		}
		*def.field = id
	}

	return nil
}

func (g *generator) bilTransferAt(r *shipmentRecord) int64 {
	at := r.TransferredAt
	if at == 0 {
		at = r.ReadyToBillAt
	}
	if at == 0 && r.DeliveredAt > 0 {
		at = r.DeliveredAt + 4*bilHour
	}
	if r.Stage == billingStageInvoiced && r.InvoiceDate > 0 {
		if at == 0 {
			at = r.InvoiceDate - 12*bilHour
		}
		at = min(at, r.InvoiceDate-bilHour)
	}
	if at == 0 {
		at = unix(g.now) - bilDay
	}

	return at
}

func (g *generator) bilBuild(
	ctx context.Context,
	setup *bilSetup,
	chunk []*shipmentRecord,
) (*bilRows, error) {
	rows := &bilRows{balances: make(map[bilBalanceKey]*bilBalance)}

	slots := make([]numberSlot, len(chunk))
	for i, r := range chunk {
		slots[i] = numberSlot{At: g.bilTransferAt(r)}
	}
	numbers, err := g.numbers.allocate(ctx, tenant.SequenceTypeInvoice, slots)
	if err != nil {
		return nil, err
	}

	invoices := make([]*invoice.Invoice, 0, len(chunk))
	owners := make(map[pulid.ID]*customerRecord, len(chunk))
	queued := make(map[pulid.ID]*billingqueue.BillingQueueItem, len(chunk))
	for i, r := range chunk {
		lines := g.bilLines(r)
		total := bilSum(lines, "")
		entity := g.bilQueueItem(setup, rows, r, numbers[i], total)
		if r.Stage != billingStageInvoiced {
			if entity.Status == billingqueue.StatusApproved && entity.ReviewCompletedAt != nil {
				r.Invoice = g.bilInvoice(rows, r, entity, lines, *entity.ReviewCompletedAt, false)
			}
			continue
		}
		inv := g.bilInvoice(rows, r, entity, lines, r.InvoiceDate, true)
		r.Invoice = inv
		invoices = append(invoices, inv)
		owners[inv.ID] = r.Customer
		queued[inv.ID] = entity
		g.bilPostedEvents(setup, rows, entity, inv)
	}

	if setup.postInvoices {
		for _, inv := range invoices {
			if err = g.bilPostInvoice(setup, rows, inv, pulid.Nil, queued[inv.ID]); err != nil {
				return nil, err
			}
		}
		if err = g.bilCreditMemos(ctx, setup, rows, invoices, owners); err != nil {
			return nil, err
		}
	}
	if setup.postPayments {
		if err = g.bilPayments(setup, rows, invoices, owners); err != nil {
			return nil, err
		}
	}

	if err = g.bilMaterialize(ctx, rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func (g *generator) bilPick(users []pulid.ID) pulid.ID {
	if len(users) == 0 {
		return g.refs.adminID
	}
	return pick(g.fake, users)
}

func (g *generator) bilClamp(at, floor int64) int64 {
	return max(floor, min(at, unix(g.now)-bilMinute))
}

func (g *generator) bilQueueItem(
	setup *bilSetup,
	rows *bilRows,
	r *shipmentRecord,
	number string,
	total decimal.Decimal,
) *billingqueue.BillingQueueItem {
	transfer := g.bilTransferAt(r)
	status := r.QueueStatus
	if r.Stage == billingStageInvoiced {
		status = billingqueue.StatusPosted
	}
	if status == "" {
		status = billingqueue.StatusReadyForReview
	}

	entity := &billingqueue.BillingQueueItem{
		ID:                    idAt("bqi_", transfer),
		OrganizationID:        g.refs.orgID,
		BusinessUnitID:        g.refs.buID,
		ShipmentID:            r.ID,
		OrderID:               r.OrderID,
		BillToCustomerID:      r.Customer.ID,
		AllocatedTotalAmount:  total,
		Number:                number,
		Status:                status,
		BillType:              billingqueue.BillTypeInvoice,
		RerateVariancePercent: decimal.Zero,
		AdjustmentContext:     map[string]any{},
		CreatedAt:             transfer,
		UpdatedAt:             transfer,
	}
	entity.SyncAllocatedMinor()
	rows.items = append(rows.items, entity)

	g.bilEvent(setup, rows, entity, billingqueue.EventTransferred,
		"Queued when "+bilShipmentRef(r)+" was transferred to billing", transfer, nil, nil)

	if status == billingqueue.StatusReadyForReview && g.fake.chance(0.35) {
		return entity
	}

	ceiling := unix(g.now) - bilMinute
	if r.Stage == billingStageInvoiced {
		ceiling = r.InvoiceDate - 10*bilMinute
	}
	ceiling = max(ceiling, transfer)

	biller := g.bilPick(g.refs.userIDs)
	entity.AssignedBillerID = &biller
	assignedAt := g.bilClamp(transfer+g.fake.between64(5*bilMinute, 90*bilMinute), transfer)
	assignedAt = max(transfer, min(assignedAt, ceiling))
	assigner := g.refs.adminID
	g.bilEvent(setup, rows, entity, billingqueue.EventAssigned,
		"Assigned "+bilName(setup, biller)+" as biller", assignedAt, &assigner,
		map[string]any{"billerId": biller.String()})
	entity.UpdatedAt = assignedAt

	if status == billingqueue.StatusReadyForReview {
		return entity
	}

	reviewStart := g.bilClamp(assignedAt+g.fake.between64(10*bilMinute, 30*bilHour), assignedAt)
	reviewStart = max(assignedAt, min(reviewStart, ceiling))
	entity.ReviewStartedAt = &reviewStart
	entity.UpdatedAt = reviewStart
	actionAt := g.bilClamp(reviewStart+g.fake.between64(5*bilMinute, 6*bilHour), reviewStart)
	actionAt = max(reviewStart, min(actionAt, ceiling))

	switch status { //nolint:exhaustive // ReadyForReview and Canceled are not seeded here
	case billingqueue.StatusOnHold:
		code := pick(g.fake, bilHoldCodes)
		before := billingqueue.StatusInReview
		entity.HoldReasonCode = &code
		entity.HeldAt = &actionAt
		entity.HeldByID = &biller
		entity.StatusBeforeHold = &before
		entity.ReviewNotes = bilHoldNotes[code]
		g.bilEvent(setup, rows, entity, billingqueue.EventHeld,
			"Put on hold · "+code.Phrase(), actionAt, &biller, nil)
		entity.UpdatedAt = actionAt
	case billingqueue.StatusException:
		code := pick(g.fake, bilExceptionCodes)
		entity.ExceptionReasonCode = &code
		entity.ExceptionNotes = bilExceptionNotes[code]
		g.bilEvent(setup, rows, entity, billingqueue.EventStatusChanged,
			"Moved to exception", actionAt, &biller, nil)
		entity.UpdatedAt = actionAt
	case billingqueue.StatusSentBackToOps:
		reason := pick(g.fake, bilSentBackReasons)
		code := reason.code
		entity.ExceptionReasonCode = &code
		entity.ExceptionNotes = reason.note
		g.bilEvent(setup, rows, entity, billingqueue.EventStatusChanged,
			"Moved to sent back to operations", actionAt, &biller, nil)
		rows.comments = append(rows.comments, g.bilSendBackComment(entity, biller, actionAt))
		entity.UpdatedAt = actionAt
	case billingqueue.StatusApproved, billingqueue.StatusPosted:
		entity.ReviewCompletedAt = &actionAt
		g.bilEvent(setup, rows, entity, billingqueue.EventApproved,
			"Approved the invoice for "+billingqueue.FormatMoney(total), actionAt, &biller, nil)
		entity.UpdatedAt = actionAt
	}

	return entity
}

// bilSendBackComment is the operations note billingqueueservice.SendBackComment
// posts on the shipment when an item is returned to operations.
func (g *generator) bilSendBackComment(
	entity *billingqueue.BillingQueueItem,
	userID pulid.ID,
	at int64,
) *shipment.ShipmentComment {
	text := "Sent back from billing"
	if entity.ExceptionReasonCode != nil {
		text += ": " + string(*entity.ExceptionReasonCode)
	}
	if entity.ExceptionNotes != "" {
		text += "\n\n" + entity.ExceptionNotes
	}

	return &shipment.ShipmentComment{
		ID:             idAt("shc_", at),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ShipmentID,
		UserID:         userID,
		Comment:        text,
		Type:           shipment.CommentTypeBilling,
		Visibility:     shipment.CommentVisibilityOperations,
		Priority:       shipment.CommentPriorityHigh,
		Source:         shipment.CommentSourceSystem,
		Metadata:       map[string]any{},
		Version:        1,
		CreatedAt:      at,
	}
}

func (g *generator) bilEvent(
	setup *bilSetup,
	rows *bilRows,
	entity *billingqueue.BillingQueueItem,
	kind billingqueue.EventKind,
	text string,
	at int64,
	actor *pulid.ID,
	payload map[string]any,
) {
	if payload == nil {
		payload = map[string]any{}
	}
	event := &billingqueue.ItemEvent{
		ID:             idAt("bqe_", at),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ItemID:         entity.ID,
		Kind:           kind,
		Text:           text,
		ActorType:      billingqueue.EventActorSystem,
		Payload:        payload,
		At:             at,
		CreatedAt:      at,
	}
	if actor != nil {
		id := *actor
		event.ActorType = billingqueue.EventActorUser
		event.ActorID = &id
		event.ActorName = bilName(setup, id)
	}
	rows.events = append(rows.events, event)
}

func (g *generator) bilLines(r *shipmentRecord) []*invoice.InvoiceLine {
	freightLines := r.FreightLines
	if len(freightLines) == 0 {
		freightLines = []shipment.FreightLine{{Label: bilFreightLine, Amount: r.Freight}}
	}
	lines := make([]*invoice.InvoiceLine, 0, len(r.Charges)+len(freightLines))
	for i := range freightLines {
		fl := &freightLines[i]
		description := strings.TrimSpace(fl.Label)
		if description == "" {
			description = bilFreightLine
		}
		freight := &invoice.InvoiceLine{
			ShipmentID:          r.ID,
			ShipmentProNumber:   r.ProNumber,
			ShipmentBOL:         r.BOL,
			LineNumber:          len(lines) + 1,
			Type:                invoice.InvoiceLineTypeFreight,
			Description:         description,
			Quantity:            decimal.NewFromInt(1),
			UnitPrice:           fl.Amount,
			Amount:              fl.Amount,
			FormulaTemplateID:   r.FormulaTemplateID,
			FormulaTemplateName: r.FormulaName,
			RateUnit:            fl.RateUnit,
		}
		// Itemized the way invoicelines.itemized does: quantity at the tariff's
		// rate when the rate times the quantity is the amount billed.
		if fl.Rate != nil {
			freight.Rate = decimal.NewNullDecimal(*fl.Rate)
			if fl.Quantity != nil && fl.Quantity.IsPositive() &&
				fl.Rate.Mul(*fl.Quantity).Round(shipment.SharePlaces).Equal(fl.Amount) {
				freight.Quantity = *fl.Quantity
				freight.UnitPrice = *fl.Rate
			}
		}
		lines = append(lines, freight)
	}

	for i := range r.Charges {
		c := &r.Charges[i]
		quantity := decimal.NewFromInt(max(c.Unit, 1))
		line := &invoice.InvoiceLine{
			ShipmentID:          r.ID,
			ShipmentProNumber:   r.ProNumber,
			ShipmentBOL:         r.BOL,
			LineNumber:          len(lines) + 1,
			Type:                invoice.InvoiceLineTypeAccessorial,
			Description:         bilAccessorialLine,
			Quantity:            quantity,
			UnitPrice:           c.Amount.Div(quantity).Round(4),
			Amount:              c.Amount,
			AccessorialChargeID: c.AccessorialChargeID,
			AdditionalChargeID:  c.AdditionalChargeID,
			ChargeCode:          c.Code,
			ChargeMethod:        c.Method,
			Rate:                decimal.NewNullDecimal(c.UnitPrice),
		}
		if c.Description != "" {
			line.Description = c.Description
		}
		if def := g.refs.accessorial(c.Code); def != nil {
			if d := strings.TrimSpace(def.Description); d != "" {
				line.Description = d
			}
		}
		if c.Method == accessorialcharge.MethodPercentage {
			line.RateBasisAmount = decimal.NewNullDecimal(r.Freight)
		}
		lines = append(lines, line)
	}

	return lines
}

func bilSum(lines []*invoice.InvoiceLine, typ invoice.InvoiceLineType) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines {
		if typ == "" || line.Type == typ {
			total = total.Add(line.Amount)
		}
	}
	return total
}

func (g *generator) bilInvoice(
	rows *bilRows,
	r *shipmentRecord,
	entity *billingqueue.BillingQueueItem,
	lines []*invoice.InvoiceLine,
	invoiceDate int64,
	posted bool,
) *invoice.Invoice {
	term := r.Customer.PaymentTerm
	if term == "" {
		term = invoice.PaymentTermNet30
	}
	inv := &invoice.Invoice{
		ID:                 idAt("inv_", invoiceDate),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		BillingQueueItemID: entity.ID,
		ShipmentID:         r.ID,
		OrderID:            r.OrderID,
		CustomerID:         r.Customer.ID,
		Scope:              invoice.ScopeShipment,
		ShipmentCount:      1,
		Detail:             customer.InvoiceDetailDetailed,
		SectionBy:          customer.InvoiceSectionKeyShipment,
		Number:             entity.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        term,
		CurrencyCode:       money.DefaultCurrencyCode,
		InvoiceDate:        invoiceDate,
		DueDate:            invoice.DueDateFromPaymentTerm(invoiceDate, term),
		ShipmentProNumber:  r.ProNumber,
		ShipmentBOL:        r.BOL,
		OrderNumber:        r.OrderNumber,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		SendStatus:         invoice.SendStatusNotSent,
		EDISendStatus:      invoice.EDISendStatusNotSent,
		FactoringOverride:  invoice.FactoringOverrideDefault,
		Lines:              lines,
		CreatedAt:          invoiceDate,
		UpdatedAt:          invoiceDate,
	}
	if r.DeliveredAt > 0 {
		inv.ServiceDate = ptr(r.DeliveredAt)
	}
	bilBillTo(inv, r.Customer)
	inv.SubtotalAmount = bilSum(lines, invoice.InvoiceLineTypeFreight)
	inv.OtherAmount = bilSum(lines, invoice.InvoiceLineTypeAccessorial)
	inv.TotalAmount = bilSum(lines, "")

	biller := g.refs.adminID
	if entity.AssignedBillerID != nil {
		biller = *entity.AssignedBillerID
	}
	if posted {
		inv.Status = invoice.StatusPosted
		inv.PostedAt = ptr(invoiceDate)
		g.facStamp(inv, r.Customer)
	}
	switch roll := g.fake.float(); {
	case !posted || r.Customer.BillingEmail == "":
	case roll < 0.92:
		sentAt := g.bilClamp(invoiceDate+g.fake.between64(bilMinute, 20*bilMinute), invoiceDate)
		inv.SendStatus = invoice.SendStatusSent
		inv.SentAt = &sentAt
		inv.SentByID = biller
		inv.EmailToSnapshot = []string{r.Customer.BillingEmail}
		inv.EmailSubjectSnapshot = fmt.Sprintf("Invoice %s from %s", inv.Number, g.refs.orgName)
	case roll < 0.95:
		inv.SendStatus = invoice.SendStatusFailed
		inv.LastSendError = "550 5.1.1 Recipient address rejected: mailbox unavailable"
		inv.EmailToSnapshot = []string{r.Customer.BillingEmail}
	}

	for _, line := range lines {
		line.ID = idAt("invl_", invoiceDate)
		line.OrganizationID = g.refs.orgID
		line.BusinessUnitID = g.refs.buID
		line.InvoiceID = inv.ID
		line.CreatedAt = invoiceDate
		line.UpdatedAt = invoiceDate
	}
	inv.SyncMinorAmounts()

	rows.invoices = append(rows.invoices, inv)
	rows.lines = append(rows.lines, lines...)
	if posted {
		rows.links = append(rows.links, bilItemInvoice{item: entity.ID, invoice: inv.ID})
	}

	return inv
}

func (g *generator) bilPostedEvents(
	setup *bilSetup,
	rows *bilRows,
	entity *billingqueue.BillingQueueItem,
	inv *invoice.Invoice,
) {
	text := "Posted as " + inv.Number
	if inv.SendStatus == invoice.SendStatusSent && len(inv.EmailToSnapshot) > 0 {
		text += " · emailed to " + inv.EmailToSnapshot[0]
	}
	actor := g.refs.adminID
	if entity.AssignedBillerID != nil {
		actor = *entity.AssignedBillerID
	}
	g.bilEvent(setup, rows, entity, billingqueue.EventPosted, text, inv.InvoiceDate, &actor,
		map[string]any{"invoiceId": inv.ID.String(), "invoiceNumber": inv.Number})
	entity.UpdatedAt = max(entity.UpdatedAt, inv.InvoiceDate)
}

func bilBillTo(inv *invoice.Invoice, cus *customerRecord) {
	inv.BillToName = cus.Name
	inv.BillToCode = cus.Code
	inv.BillToAddressLine1 = cus.AddressLine1
	inv.BillToCity = cus.City
	inv.BillToState = cus.StateAbbr
	inv.BillToPostalCode = cus.PostalCode
	inv.BillToCountry = bilCountry
}

func bilShipmentRef(r *shipmentRecord) string {
	switch {
	case r.ProNumber != "":
		return r.ProNumber
	case r.BOL != "":
		return r.BOL
	default:
		return "this shipment"
	}
}

func bilName(setup *bilSetup, id pulid.ID) string {
	if name, ok := setup.names[id]; ok && name != "" {
		return name
	}
	return "a biller"
}

func (g *generator) bilResolvePeriod(
	setup *bilSetup,
	date int64,
) (*fiscalperiod.FiscalPeriod, int64, bool) {
	periods := setup.periods
	if len(periods) == 0 {
		return nil, 0, false
	}
	idx := sort.Search(len(periods), func(i int) bool { return periods[i].EndDate >= date })
	if idx == len(periods) {
		last := periods[len(periods)-1]
		return last, last.EndDate, true
	}
	p := periods[idx]
	if date < p.StartDate {
		return p, p.StartDate, true
	}
	return p, date, true
}

func (g *generator) bilNewJournal(
	setup *bilSetup,
	date int64,
) (*bilJournal, error) {
	period, accountingDate, ok := g.bilResolvePeriod(setup, date)
	if !ok {
		return nil, fmt.Errorf("%w: no open fiscal period", ErrMissingSeedData)
	}
	return &bilJournal{
		batchID: idAt("jb_", accountingDate),
		entryID: idAt("je_", accountingDate),
		period:  period,
		date:    accountingDate,
	}, nil
}

func (g *generator) bilPostInvoice(
	setup *bilSetup,
	rows *bilRows,
	inv *invoice.Invoice,
	related pulid.ID,
	entity *billingqueue.BillingQueueItem,
) error {
	event := tenant.JournalSourceEventInvoicePosted
	debit, credit := setup.ar, setup.revenue
	ledgerAmount := inv.TotalAmountMinor
	amount := inv.TotalAmountMinor
	if inv.IsCreditMemo() {
		event = tenant.JournalSourceEventCreditMemoPosted
		debit, credit = credit, debit
		amount = inv.CreditTotalMinor()
		ledgerAmount = -amount
	}
	if amount == 0 {
		return nil
	}

	journal, err := g.bilNewJournal(setup, *inv.PostedAt)
	if err != nil {
		return err
	}
	actor := g.refs.adminID
	if entity != nil && entity.AssignedBillerID != nil {
		actor = *entity.AssignedBillerID
	}
	journal.actorID = actor
	journal.description = "Invoice posted for " + inv.Number
	journal.sourceType = "Invoice"
	journal.sourceID = inv.ID
	journal.document = inv.Number
	journal.event = event
	journal.referenceType = event.String()
	journal.idempotency = "invoice-posted:" + inv.ID.String()
	journal.auto = true
	journal.legs = []bilLeg{
		{account: debit, debit: amount, customer: inv.CustomerID},
		{account: credit, credit: amount, customer: inv.CustomerID},
	}
	rows.journals = append(rows.journals, journal)

	rows.ledger = append(rows.ledger, &customerledger.CustomerLedgerEntry{
		ID:               idAt("cledg_", journal.date),
		OrganizationID:   g.refs.orgID,
		BusinessUnitID:   g.refs.buID,
		CustomerID:       inv.CustomerID,
		SourceObjectType: "Invoice",
		SourceObjectID:   inv.ID.String(),
		SourceEventType:  event.String(),
		RelatedInvoiceID: related,
		DocumentNumber:   inv.Number,
		TransactionDate:  journal.date,
		LineNumber:       1,
		AmountMinor:      ledgerAmount,
		CreatedByID:      actor,
		CreatedAt:        *inv.PostedAt,
	})

	return nil
}

func (g *generator) bilCreditMemos(
	ctx context.Context,
	setup *bilSetup,
	rows *bilRows,
	invoices []*invoice.Invoice,
	owners map[pulid.ID]*customerRecord,
) error {
	targets := make([]*invoice.Invoice, 0)
	dates := make([]int64, 0)
	nowUnix := unix(g.now)
	for _, inv := range invoices {
		if inv.FactoringCompanyID != nil || !g.fake.chance(0.012) {
			continue
		}
		memoDate := inv.InvoiceDate + g.fake.between64(3*bilDay, 40*bilDay)
		if memoDate >= nowUnix-bilHour {
			continue
		}
		targets = append(targets, inv)
		dates = append(dates, memoDate)
	}
	if len(targets) == 0 {
		return nil
	}

	slots := make([]numberSlot, len(targets))
	for i, at := range dates {
		slots[i] = numberSlot{At: at}
	}
	numbers, err := g.numbers.allocate(ctx, tenant.SequenceTypeCreditMemo, slots)
	if err != nil {
		return err
	}

	for i, target := range targets {
		open := target.OpenBalanceMinor()
		creditMinor := money.MinorUnits(
			target.TotalAmount.Mul(decimal.NewFromFloat(0.04 + g.fake.float()*0.11)),
		)
		creditMinor = min(creditMinor, open)
		if creditMinor <= 0 {
			continue
		}
		if err = g.bilCreditMemo(setup, rows, target, owners[target.ID], numbers[i], dates[i],
			creditMinor); err != nil {
			return err
		}
	}

	return nil
}

func (g *generator) bilCreditMemo(
	setup *bilSetup,
	rows *bilRows,
	target *invoice.Invoice,
	cus *customerRecord,
	number string,
	memoDate int64,
	creditMinor int64,
) error {
	raisedBy := g.bilPick(g.refs.userIDs)
	reason := pick(g.fake, bilMemoReasons)
	entity := &billingqueue.BillingQueueItem{
		ID:                    idAt("bqi_", memoDate),
		OrganizationID:        g.refs.orgID,
		BusinessUnitID:        g.refs.buID,
		BillToCustomerID:      target.CustomerID,
		AssignedBillerID:      &raisedBy,
		Number:                number,
		Status:                billingqueue.StatusPosted,
		BillType:              billingqueue.BillTypeCreditMemo,
		RerateVariancePercent: decimal.Zero,
		AdjustmentContext: map[string]any{
			"memo":       true,
			"memoKind":   string(invoice.MemoKindManual),
			"raisedById": raisedBy.String(),
		},
		CreatedAt: memoDate,
		UpdatedAt: memoDate,
	}
	entity.SyncAllocatedMinor()
	rows.items = append(rows.items, entity)

	amount := money.DecimalFromMinor(creditMinor).Neg()
	line := &invoice.InvoiceLine{
		ID:             idAt("invl_", memoDate),
		OrganizationID: g.refs.orgID,
		BusinessUnitID: g.refs.buID,
		LineNumber:     1,
		Type:           invoice.InvoiceLineTypeMemo,
		Description:    reason,
		Quantity:       decimal.NewFromInt(1),
		UnitPrice:      amount,
		Amount:         amount,
		CreatedAt:      memoDate,
		UpdatedAt:      memoDate,
	}
	memo := &invoice.Invoice{
		ID:                 idAt("inv_", memoDate),
		OrganizationID:     g.refs.orgID,
		BusinessUnitID:     g.refs.buID,
		BillingQueueItemID: entity.ID,
		CustomerID:         target.CustomerID,
		Scope:              invoice.ScopeMemo,
		Detail:             customer.InvoiceDetailDetailed,
		SectionBy:          customer.InvoiceSectionKeyShipment,
		Number:             number,
		BillType:           billingqueue.BillTypeCreditMemo,
		Status:             invoice.StatusPosted,
		PaymentTerm:        target.PaymentTerm,
		CurrencyCode:       money.DefaultCurrencyCode,
		InvoiceDate:        memoDate,
		DueDate:            ptr(memoDate),
		PostedAt:           ptr(memoDate),
		SubtotalAmount:     decimal.Zero,
		OtherAmount:        decimal.Zero,
		TotalAmount:        amount,
		AppliedAmount:      decimal.Zero,
		SettlementStatus:   invoice.SettlementStatusUnpaid,
		DisputeStatus:      invoice.DisputeStatusNone,
		SendStatus:         invoice.SendStatusNotSent,
		EDISendStatus:      invoice.EDISendStatusNotSent,
		FactoringOverride:  invoice.FactoringOverrideDefault,
		ReferenceInvoiceID: target.ID,
		MemoReason:         reason,
		MemoKind:           invoice.MemoKindManual,
		Lines:              []*invoice.InvoiceLine{line},
		CreatedAt:          memoDate,
		UpdatedAt:          memoDate,
	}
	if cus != nil {
		bilBillTo(memo, cus)
	} else {
		memo.BillToName = target.BillToName
	}
	line.InvoiceID = memo.ID
	memo.SyncMinorAmounts()

	rows.invoices = append(rows.invoices, memo)
	rows.lines = append(rows.lines, line)
	rows.links = append(rows.links, bilItemInvoice{item: entity.ID, invoice: memo.ID})
	rows.memoCount++

	g.bilEvent(setup, rows, entity, billingqueue.EventPosted, "Posted as "+number, memoDate,
		&raisedBy, map[string]any{"invoiceId": memo.ID.String(), "invoiceNumber": number})

	if err := g.bilPostInvoice(setup, rows, memo, target.ID, entity); err != nil {
		return err
	}

	_, accountingDate, _ := g.bilResolvePeriod(setup, memoDate)
	target.ApplyPaymentMinor(creditMinor)
	memo.ApplyCreditMinor(creditMinor)
	target.UpdatedAt = max(target.UpdatedAt, memoDate)
	rows.memoApps = append(rows.memoApps, &customerpayment.CreditMemoApplication{
		ID:                  idAt("cma_", memoDate),
		OrganizationID:      g.refs.orgID,
		BusinessUnitID:      g.refs.buID,
		CreditMemoInvoiceID: memo.ID,
		InvoiceID:           target.ID,
		AppliedAmountMinor:  creditMinor,
		AccountingDate:      accountingDate,
		LineNumber:          1,
		Status:              customerpayment.CreditApplicationStatusApplied,
		CreatedByID:         raisedBy,
		CreatedAt:           memoDate,
		UpdatedAt:           memoDate,
	})

	return nil
}

func (g *generator) bilPlanPayment(inv *invoice.Invoice) *bilPaymentPlan {
	open := inv.OpenBalanceMinor()
	if open <= 0 || inv.IsCreditMemo() || inv.FactoringCompanyID != nil {
		return nil
	}
	nowUnix := unix(g.now)
	due := inv.InvoiceDate
	if inv.DueDate != nil {
		due = *inv.DueDate
	}
	overdueDays := (nowUnix - due) / bilDay

	var full, partial float64
	switch {
	case overdueDays < 0:
		full, partial = 0.18, 0.02
	case overdueDays <= 30:
		full, partial = 0.55, 0.08
	case overdueDays <= 60:
		full, partial = 0.74, 0.08
	case overdueDays <= 90:
		full, partial = 0.84, 0.06
	default:
		full, partial = 0.93, 0.03
	}

	roll := g.fake.float()
	if roll >= full+partial {
		return nil
	}

	earliest := inv.InvoiceDate + bilDay
	latest := nowUnix - bilHour
	if earliest > latest {
		return nil
	}
	date := due + g.fake.between64(-12*bilDay, 21*bilDay)
	if overdueDays < 0 {
		date = earliest + g.fake.between64(0, latest-earliest)
	}
	date = max(earliest, min(date, latest))

	amount := open
	if roll >= full {
		amount = open * int64(g.fake.between(30, 80)) / 100
	}
	if amount <= 0 {
		return nil
	}

	return &bilPaymentPlan{invoice: inv, amount: amount, date: date}
}

func (g *generator) bilPayments(
	setup *bilSetup,
	rows *bilRows,
	invoices []*invoice.Invoice,
	owners map[pulid.ID]*customerRecord,
) error {
	groups := make(map[string]*bilPaymentGroup)
	for _, inv := range invoices {
		plan := g.bilPlanPayment(inv)
		if plan == nil {
			continue
		}
		cus := owners[inv.ID]
		if cus == nil {
			continue
		}
		key := fmt.Sprintf("%s|%d", cus.ID, plan.date/bilWeek)
		group, ok := groups[key]
		if !ok {
			group = &bilPaymentGroup{customer: cus, key: key}
			groups[key] = group
		}
		group.plans = append(group.plans, plan)
		group.date = max(group.date, plan.date)
	}

	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		group := groups[key]
		for begin := 0; begin < len(group.plans); begin += bilMaxApplications {
			plans := group.plans[begin:min(begin+bilMaxApplications, len(group.plans))]
			if err := g.bilPayment(setup, rows, group.customer, group.date, plans); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *generator) bilPaymentMethod() (customerpayment.Method, string) {
	switch g.fake.weighted([]int{55, 30, 10, 3, 2}) {
	case 0:
		return customerpayment.MethodACH, "ACH" + g.fake.digits(10)
	case 1:
		return customerpayment.MethodCheck, "CHK " + g.fake.digits(6)
	case 2:
		return customerpayment.MethodWire, "FW" + g.fake.digits(12)
	case 3:
		return customerpayment.MethodCard, "CC-" + g.fake.digits(8)
	default:
		return customerpayment.MethodOther, "LBX-" + g.fake.digits(8)
	}
}

func (g *generator) bilPayment(
	setup *bilSetup,
	rows *bilRows,
	cus *customerRecord,
	date int64,
	plans []*bilPaymentPlan,
) error {
	journal, err := g.bilNewJournal(setup, date)
	if err != nil {
		return err
	}
	actor := g.bilPick(g.refs.userIDs)
	method, reference := g.bilPaymentMethod()

	payment := &customerpayment.Payment{
		ID:              idAt("cpay_", date),
		OrganizationID:  g.refs.orgID,
		BusinessUnitID:  g.refs.buID,
		CustomerID:      cus.ID,
		PaymentDate:     date,
		AccountingDate:  journal.date,
		Status:          customerpayment.StatusPosted,
		PaymentMethod:   method,
		ReferenceNumber: reference,
		CurrencyCode:    money.DefaultCurrencyCode,
		PostedBatchID:   journal.batchID,
		CreatedByID:     actor,
		UpdatedByID:     actor,
		CreatedAt:       date,
		UpdatedAt:       date,
	}

	numbers := make([]string, 0, len(plans))
	for _, plan := range plans {
		plan.invoice.ApplyPaymentMinor(plan.amount)
		plan.invoice.UpdatedAt = max(plan.invoice.UpdatedAt, date)
		payment.Applications = append(payment.Applications, &customerpayment.Application{
			ID:                 idAt("cpapp_", date),
			OrganizationID:     g.refs.orgID,
			BusinessUnitID:     g.refs.buID,
			CustomerPaymentID:  payment.ID,
			InvoiceID:          plan.invoice.ID,
			AppliedAmountMinor: plan.amount,
			CreatedAt:          date,
			UpdatedAt:          date,
		})
		payment.AmountMinor += plan.amount
		numbers = append(numbers, plan.invoice.Number)
	}
	if setup.unapplied.IsNotNil() && g.fake.chance(0.04) {
		payment.AmountMinor += int64(g.fake.between(50, 750)) * 100
	}
	payment.SyncAmounts()
	payment.Memo = bilRemittanceMemo(numbers, payment.UnappliedAmountMinor)

	journal.actorID = actor
	journal.description = "Customer payment " + payment.DocumentLabel()
	journal.sourceType = "CustomerPayment"
	journal.sourceID = payment.ID
	journal.document = payment.ReferenceNumber
	journal.event = tenant.JournalSourceEventCustomerPaymentPosted
	journal.referenceType = tenant.JournalSourceEventCustomerPaymentPosted.String()
	journal.idempotency = "customer-payment-posted:" + payment.ID.String()
	journal.legs = append(journal.legs, bilLeg{
		account: setup.cash, debit: payment.AmountMinor, customer: cus.ID,
	})
	for i, app := range payment.Applications {
		journal.legs = append(journal.legs, bilLeg{
			account: setup.applied, credit: app.AppliedAmountMinor, customer: cus.ID,
		})
		rows.ledger = append(rows.ledger, &customerledger.CustomerLedgerEntry{
			ID:               idAt("cledg_", journal.date),
			OrganizationID:   g.refs.orgID,
			BusinessUnitID:   g.refs.buID,
			CustomerID:       cus.ID,
			SourceObjectType: "CustomerPayment",
			SourceObjectID:   payment.ID.String(),
			SourceEventType:  journal.event.String(),
			RelatedInvoiceID: app.InvoiceID,
			DocumentNumber:   payment.ReferenceNumber,
			TransactionDate:  journal.date,
			LineNumber:       i + 1,
			AmountMinor:      -app.AppliedAmountMinor,
			CreatedByID:      actor,
			CreatedAt:        date,
		})
	}
	if payment.UnappliedAmountMinor > 0 {
		journal.legs = append(journal.legs, bilLeg{
			account: setup.unapplied, credit: payment.UnappliedAmountMinor, customer: cus.ID,
		})
	}

	rows.journals = append(rows.journals, journal)
	rows.payments = append(rows.payments, payment)
	g.payments = append(g.payments, payment)
	rows.apps = append(rows.apps, payment.Applications...)

	return nil
}

func bilRemittanceMemo(numbers []string, unappliedMinor int64) string {
	memo := "Remittance for " + numbers[0]
	if len(numbers) > 1 {
		memo = fmt.Sprintf("Remittance for %d invoices: %s", len(numbers), strings.Join(numbers, ", "))
	}
	if unappliedMinor > 0 {
		memo += "; " + money.DecimalFromMinor(unappliedMinor).StringFixed(2) + " with no remittance detail"
	}
	return memo
}

func (g *generator) bilMaterialize(ctx context.Context, rows *bilRows) error {
	if len(rows.journals) == 0 {
		return nil
	}

	slots := make([]numberSlot, len(rows.journals))
	for i, j := range rows.journals {
		slots[i] = numberSlot{At: j.date}
	}
	batchNumbers, err := g.numbers.allocate(ctx, tenant.SequenceTypeJournalBatch, slots)
	if err != nil {
		return err
	}
	entryNumbers, err := g.numbers.allocate(ctx, tenant.SequenceTypeJournalEntry, slots)
	if err != nil {
		return err
	}

	for i, j := range rows.journals {
		var totalDebit, totalCredit int64
		lines := make([]*journalentry.JournalEntryLine, 0, len(j.legs))
		for _, leg := range j.legs {
			if leg.debit == 0 && leg.credit == 0 {
				continue
			}
			totalDebit += leg.debit
			totalCredit += leg.credit
			lines = append(lines, &journalentry.JournalEntryLine{
				ID:             idAt("jel_", j.date),
				BusinessUnitID: g.refs.buID,
				OrganizationID: g.refs.orgID,
				JournalEntryID: j.entryID,
				GLAccountID:    leg.account,
				LineNumber:     int16(len(lines) + 1), //nolint:gosec // bounded by leg count
				Description:    j.description,
				DebitAmount:    leg.debit,
				CreditAmount:   leg.credit,
				NetAmount:      leg.debit - leg.credit,
				CustomerID:     leg.customer,
				CreatedAt:      j.date,
				UpdatedAt:      j.date,
			})
		}
		if totalDebit != totalCredit || len(lines) < 2 {
			return fmt.Errorf("journal for %s does not balance: %d debit, %d credit",
				j.document, totalDebit, totalCredit)
		}

		date := j.date
		rows.batches = append(rows.batches, &journalentry.JournalBatch{
			ID:             j.batchID,
			OrganizationID: g.refs.orgID,
			BusinessUnitID: g.refs.buID,
			BatchNumber:    batchNumbers[i],
			BatchType:      bilBatchTypeSystem,
			Status:         bilStatusPosted,
			Description:    j.description,
			AccountingDate: date,
			FiscalYearID:   j.period.FiscalYearID,
			FiscalPeriodID: j.period.ID,
			EntryCount:     1,
			PostedAt:       &date,
			PostedByID:     j.actorID,
			CreatedByID:    j.actorID,
			UpdatedByID:    j.actorID,
			CreatedAt:      date,
			UpdatedAt:      date,
		})
		rows.entries = append(rows.entries, &journalentry.JournalEntry{
			ID:              j.entryID,
			BusinessUnitID:  g.refs.buID,
			OrganizationID:  g.refs.orgID,
			BatchID:         j.batchID,
			FiscalYearID:    j.period.FiscalYearID,
			FiscalPeriodID:  j.period.ID,
			EntryNumber:     entryNumbers[i],
			EntryDate:       date,
			EntryType:       journalentry.EntryTypeStandard,
			AccountingDate:  date,
			Status:          journalentry.StatusPosted,
			ReferenceNumber: j.document,
			ReferenceType:   j.referenceType,
			ReferenceID:     j.sourceID.String(),
			Description:     j.description,
			TotalDebit:      totalDebit,
			TotalCredit:     totalCredit,
			IsPosted:        true,
			PostedAt:        &date,
			PostedByID:      j.actorID,
			IsAutoGenerated: j.auto,
			IsApproved:      true,
			ApprovedAt:      &date,
			ApprovedByID:    j.actorID,
			CreatedByID:     j.actorID,
			UpdatedByID:     j.actorID,
			CreatedAt:       date,
			UpdatedAt:       date,
		})
		rows.jlines = append(rows.jlines, lines...)

		sourceID := idAt("jsrc_", date)
		rows.sources = append(rows.sources, &journalsource.Source{
			ID:                   sourceID,
			BusinessUnitID:       g.refs.buID,
			OrganizationID:       g.refs.orgID,
			SourceObjectType:     j.sourceType,
			SourceObjectID:       j.sourceID.String(),
			SourceEventType:      j.event.String(),
			SourceDocumentNumber: j.document,
			Status:               bilStatusPosted,
			IdempotencyKey:       j.idempotency,
			JournalBatchID:       j.batchID,
			JournalEntryID:       j.entryID,
			CreatedAt:            date,
		})
		rows.srcLinks = append(rows.srcLinks, &journalsource.SourceJournalLink{
			ID:              sourceID,
			BusinessUnitID:  g.refs.buID,
			OrganizationID:  g.refs.orgID,
			JournalSourceID: sourceID,
			JournalBatchID:  j.batchID,
			JournalEntryID:  j.entryID,
			LinkRole:        "Primary",
			CreatedAt:       date,
		})

		for _, line := range lines {
			key := bilBalanceKey{
				account: line.GLAccountID,
				year:    j.period.FiscalYearID,
				period:  j.period.ID,
			}
			bal, ok := rows.balances[key]
			if !ok {
				bal = &bilBalance{}
				rows.balances[key] = bal
			}
			bal.debit += line.DebitAmount
			bal.credit += line.CreditAmount
			bal.net += line.NetAmount
			if date >= bal.lastDate {
				bal.lastDate = date
				bal.lastEntry = j.entryID
			}
		}
	}

	return nil
}

func bilStampOf[T any](rows []T, fn func(T) stamp) []stamp {
	stamps := make([]stamp, len(rows))
	for i, row := range rows {
		stamps[i] = fn(row)
	}
	return stamps
}

func bilInsert[T any](
	ctx context.Context,
	tx *keel.DB,
	h keel.Handle[T],
	rows []*T,
	fn func(*T) stamp,
	createdOnly bool,
) error {
	if len(rows) == 0 {
		return nil
	}
	stamps := bilStampOf(rows, fn)
	if err := insertRows(ctx, tx, h, rows); err != nil {
		return err
	}

	return backdateColumns(ctx, tx, h, stamps, createdOnly)
}

func (g *generator) bilWrite(ctx context.Context, tx *keel.DB, rows *bilRows) error {

	if err := bilInsert(ctx, tx, billingqueuedb.BillingQueueItems.Handle, rows.items,
		func(r *billingqueue.BillingQueueItem) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, invoicedb.Invoices.Handle, rows.invoices,
		func(r *invoice.Invoice) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, invoicedb.InvoiceLines.Handle, rows.lines,
		func(r *invoice.InvoiceLine) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := g.bilLinkInvoices(ctx, tx, rows.links); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, shipmentdb.ShipmentComments.Handle, rows.comments,
		func(r *shipment.ShipmentComment) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.CreatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, billingqueuedb.ItemEvents.Handle, rows.events,
		func(r *billingqueue.ItemEvent) stamp {
			return stamp{id: r.ID, created: r.At}
		}, true); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, customerpaymentdb.CreditMemoApplications.Handle, rows.memoApps,
		func(r *customerpayment.CreditMemoApplication) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, journalentrydb.JournalBatches.Handle, rows.batches,
		func(r *journalentry.JournalBatch) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, journalentrydb.JournalEntries.Handle, rows.entries,
		func(r *journalentry.JournalEntry) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, journalentrydb.JournalEntryLines.Handle, rows.jlines,
		func(r *journalentry.JournalEntryLine) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := insertRows(ctx, tx, journalsourcedb.Sources.Handle, rows.sources); err != nil {
		return fmt.Errorf("insert journal_sources: %w", err)
	}
	if err := insertRows(ctx, tx, journalsourcedb.SourceJournalLinks.Handle, rows.srcLinks); err != nil {
		return fmt.Errorf("insert source_journal_links: %w", err)
	}
	if err := bilInsert(ctx, tx, customerpaymentdb.Payments.Handle, rows.payments,
		func(r *customerpayment.Payment) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, customerpaymentdb.Applications.Handle, rows.apps,
		func(r *customerpayment.Application) stamp {
			return stamp{id: r.ID, created: r.CreatedAt, updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := bilInsert(ctx, tx, customerledgerdb.CustomerLedgerEntries.Handle, rows.ledger,
		func(r *customerledger.CustomerLedgerEntry) stamp {
			return stamp{id: r.ID, created: r.CreatedAt}
		}, true); err != nil {
		return err
	}

	return g.bilApplyBalances(ctx, tx, rows.balances)
}

func (g *generator) bilLinkInvoices(ctx context.Context, tx *keel.DB, links []bilItemInvoice) error {
	for begin := 0; begin < len(links); begin += backdateChunk {
		chunk := links[begin:min(begin+backdateChunk, len(links))]
		items := make([]string, len(chunk))
		invoices := make([]string, len(chunk))
		for i, link := range chunk {
			items[i] = link.item.String()
			invoices[i] = link.invoice.String()
		}
		bq := billingqueuedb.BillingQueueItems
		d := keel.Unnest("d", []string{"id", "invoice_id"}, items, invoices)
		if _, err := bq.With(d).Join(d, d.Col("id").EqCol(bq.ID)).Where(keel.True()).
			Update(ctx, tx, bq.InvoiceID.ToExpr(d.Col("invoice_id")), keepUpdated(bq.Handle)); err != nil {
			return fmt.Errorf("link billing queue items to invoices: %w", err)
		}
	}

	return nil
}

func (g *generator) bilApplyBalances(
	ctx context.Context,
	tx *keel.DB,
	balances map[bilBalanceKey]*bilBalance,
) error {
	keys := make([]bilBalanceKey, 0, len(balances))
	for key := range balances {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].account != keys[j].account {
			return keys[i].account < keys[j].account
		}
		return keys[i].period < keys[j].period
	})

	rows := make([]*glbalance.GLAccountPeriodBalance, 0, len(keys))
	for _, key := range keys {
		bal := balances[key]
		rows = append(rows, &glbalance.GLAccountPeriodBalance{
			OrganizationID:     g.refs.orgID,
			BusinessUnitID:     g.refs.buID,
			GLAccountID:        key.account,
			FiscalYearID:       key.year,
			FiscalPeriodID:     key.period,
			PeriodDebitMinor:   bal.debit,
			PeriodCreditMinor:  bal.credit,
			NetChangeMinor:     bal.net,
			LastJournalEntryID: bal.lastEntry,
		})
	}

	return postBalances(ctx, tx, rows)
}

func (g *generator) bilCount(rows *bilRows) {
	invoices, drafts := 0, 0
	for _, inv := range rows.invoices {
		switch {
		case inv.IsCreditMemo():
		case inv.Status == invoice.StatusDraft:
			drafts++
		default:
			invoices++
		}
	}
	g.report.add("Billing queue items", len(rows.items))
	g.report.add("Billing queue events", len(rows.events))
	g.report.add("Shipment comments", len(rows.comments))
	g.report.add("Invoices", invoices)
	g.report.add("Draft invoices", drafts)
	g.report.add("Credit memos", rows.memoCount)
	g.report.add("Invoice lines", len(rows.lines))
	g.report.add("Credit memo applications", len(rows.memoApps))
	g.report.add("Customer payments", len(rows.payments))
	g.report.add("Payment applications", len(rows.apps))
	g.report.add("Customer ledger entries", len(rows.ledger))
	g.report.add("Journal entries", len(rows.entries))
	g.report.add("Journal entry lines", len(rows.jlines))
}
