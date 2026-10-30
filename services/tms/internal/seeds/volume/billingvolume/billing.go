package billingvolume

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueueevent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/seeds/volume/ledgervolume"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	bilChunkSize       = 1_000
	bilWeek            = 7 * volumegen.Day
	bilMaxApplications = 12
	bilFreightLine     = "Freight charge"
	bilAccessorialLine = "Accessorial charge"
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
	{
		billingqueue.ExceptionMissingDocumentation,
		"Stop times missing on delivery; please update actuals",
	},
	{
		billingqueue.ExceptionAccessorialDispute,
		"Detention not entered though the driver reported three hours",
	},
	{
		billingqueue.ExceptionMissingReferenceNumber,
		"BOL number on the shipment does not match the paperwork",
	},
	{
		billingqueue.ExceptionCustomerInformationError,
		"Delivery appointment shows the wrong consignee",
	},
}

var bilMemoReasons = []string{
	"Allowance for product damaged in transit",
	"Detention billed in error",
	"Rate adjustment agreed with the customer",
	"Duplicate lumper charge credited",
	"Service failure credit for late delivery",
}

type bilPaymentGroup struct {
	customer *volumegen.CustomerRecord
	key      string
	date     int64
	plans    []*ledgervolume.PaymentPlan
}

func (g *billingPhase) seedBilling(ctx context.Context) error {
	records := make([]*volumegen.ShipmentRecord, 0, len(g.Shipments))
	for _, r := range g.Shipments {
		if r.Customer == nil {
			continue
		}
		if r.Stage == volumegen.BillingStageQueued || r.Stage == volumegen.BillingStageInvoiced {
			records = append(records, r)
		}
	}
	if len(records) == 0 {
		return nil
	}

	setup, err := g.LoadSetup(ctx)
	if err != nil {
		return err
	}

	sort.SliceStable(records, func(i, j int) bool {
		return g.bilTransferAt(records[i]) < g.bilTransferAt(records[j])
	})

	for begin := 0; begin < len(records); begin += bilChunkSize {
		chunk := records[begin:min(begin+bilChunkSize, len(records))]
		rows, disputes, buildErr := g.bilBuild(ctx, setup, chunk)
		if buildErr != nil {
			return buildErr
		}
		if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
			if writeErr := g.Write(ctx, tx, rows); writeErr != nil {
				return writeErr
			}
			return g.bilWriteDisputes(ctx, tx, disputes)
		}); err != nil {
			return err
		}
		g.bilCount(rows)
		g.Report.Add("Invoice disputes", len(disputes))
		g.Progress("billing", min(begin+bilChunkSize, len(records)), len(records))
	}

	return nil
}

func (g *billingPhase) bilTransferAt(r *volumegen.ShipmentRecord) int64 {
	at := r.TransferredAt
	if at == 0 {
		at = r.ReadyToBillAt
	}
	if at == 0 && r.DeliveredAt > 0 {
		at = r.DeliveredAt + 4*volumegen.Hour
	}
	if r.Stage == volumegen.BillingStageInvoiced && r.InvoiceDate > 0 {
		if at == 0 {
			at = r.InvoiceDate - 12*volumegen.Hour
		}
		at = min(at, r.InvoiceDate-volumegen.Hour)
	}
	if at == 0 {
		at = volumegen.Unix(g.Now) - volumegen.Day
	}

	return at
}

func (g *billingPhase) bilBuild(
	ctx context.Context,
	setup *ledgervolume.Setup,
	chunk []*volumegen.ShipmentRecord,
) (*ledgervolume.Rows, []*invoicedispute.InvoiceDispute, error) {
	rows := &ledgervolume.Rows{Balances: make(map[ledgervolume.BalanceKey]*ledgervolume.Balance)}

	slots := make([]volumegen.NumberSlot, len(chunk))
	for i, r := range chunk {
		slots[i] = volumegen.NumberSlot{At: g.bilTransferAt(r)}
	}
	numbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeInvoice, slots)
	if err != nil {
		return nil, nil, err
	}

	invoices := make([]*invoice.Invoice, 0, len(chunk))
	owners := make(map[pulid.ID]*volumegen.CustomerRecord, len(chunk))
	queued := make(map[pulid.ID]*billingqueue.BillingQueueItem, len(chunk))
	for i, r := range chunk {
		lines := g.bilLines(r)
		total := ledgervolume.Sum(lines, "")
		entity := g.bilQueueItem(setup, rows, r, numbers[i], total)
		if r.Stage != volumegen.BillingStageInvoiced {
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

	if setup.PostInvoices {
		for _, inv := range invoices {
			if err = g.PostInvoice(setup, rows, inv, pulid.Nil, queued[inv.ID]); err != nil {
				return nil, nil, err
			}
		}
		if err = g.bilCreditMemos(ctx, setup, rows, invoices, owners); err != nil {
			return nil, nil, err
		}
	}
	var disputes []*invoicedispute.InvoiceDispute
	if setup.PostPayments {
		if disputes, err = g.bilPayments(setup, rows, invoices, owners); err != nil {
			return nil, nil, err
		}
	}

	if err = g.Materialize(ctx, rows); err != nil {
		return nil, nil, err
	}

	return rows, disputes, nil
}

func (g *billingPhase) bilClamp(at, floor int64) int64 {
	return max(floor, min(at, volumegen.Unix(g.Now)-volumegen.Minute))
}

func (g *billingPhase) bilQueueItem(
	setup *ledgervolume.Setup,
	rows *ledgervolume.Rows,
	r *volumegen.ShipmentRecord,
	number string,
	total decimal.Decimal,
) *billingqueue.BillingQueueItem {
	transfer := g.bilTransferAt(r)
	status := r.QueueStatus
	if r.Stage == volumegen.BillingStageInvoiced {
		status = billingqueue.StatusPosted
	}
	if status == "" {
		status = billingqueue.StatusReadyForReview
	}

	entity := &billingqueue.BillingQueueItem{
		ID:                    volumegen.IDAt("bqi_", transfer),
		OrganizationID:        g.Refs.OrgID,
		BusinessUnitID:        g.Refs.BuID,
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
	rows.Items = append(rows.Items, entity)

	g.bilEvent(setup, rows, entity, billingqueueevent.EventTransferred,
		"Queued when "+bilShipmentRef(r)+" was transferred to billing", transfer, nil, nil)

	if status == billingqueue.StatusReadyForReview && g.Fake.Chance(0.35) {
		return entity
	}

	ceiling := volumegen.Unix(g.Now) - volumegen.Minute
	if r.Stage == volumegen.BillingStageInvoiced {
		ceiling = r.InvoiceDate - 10*volumegen.Minute
	}
	ceiling = max(ceiling, transfer)

	biller := g.PickUser(g.Refs.UserIDs)
	entity.AssignedBillerID = &biller
	assignedAt := g.bilClamp(
		transfer+g.Fake.Between64(5*volumegen.Minute, 90*volumegen.Minute),
		transfer,
	)
	assignedAt = max(transfer, min(assignedAt, ceiling))
	assigner := g.Refs.AdminID
	g.bilEvent(setup, rows, entity, billingqueueevent.EventAssigned,
		"Assigned "+bilName(setup, biller)+" as biller", assignedAt, &assigner,
		map[string]any{"billerId": biller.String()})
	entity.UpdatedAt = assignedAt

	if status == billingqueue.StatusReadyForReview {
		return entity
	}

	reviewStart := g.bilClamp(
		assignedAt+g.Fake.Between64(10*volumegen.Minute, 30*volumegen.Hour),
		assignedAt,
	)
	reviewStart = max(assignedAt, min(reviewStart, ceiling))
	entity.ReviewStartedAt = &reviewStart
	entity.UpdatedAt = reviewStart
	actionAt := g.bilClamp(
		reviewStart+g.Fake.Between64(5*volumegen.Minute, 6*volumegen.Hour),
		reviewStart,
	)
	actionAt = max(reviewStart, min(actionAt, ceiling))

	switch status { //nolint:exhaustive // ReadyForReview and Canceled are not seeded here
	case billingqueue.StatusOnHold:
		code := volumegen.Pick(g.Fake, bilHoldCodes)
		before := billingqueue.StatusInReview
		entity.HoldReasonCode = &code
		entity.HeldAt = &actionAt
		entity.HeldByID = &biller
		entity.StatusBeforeHold = &before
		entity.ReviewNotes = bilHoldNotes[code]
		g.bilEvent(setup, rows, entity, billingqueueevent.EventHeld,
			"Put on hold · "+code.Phrase(), actionAt, &biller, nil)
		entity.UpdatedAt = actionAt
	case billingqueue.StatusException:
		code := volumegen.Pick(g.Fake, bilExceptionCodes)
		entity.ExceptionReasonCode = &code
		entity.ExceptionNotes = bilExceptionNotes[code]
		g.bilEvent(setup, rows, entity, billingqueueevent.EventStatusChanged,
			"Moved to exception", actionAt, &biller, nil)
		entity.UpdatedAt = actionAt
	case billingqueue.StatusSentBackToOps:
		reason := volumegen.Pick(g.Fake, bilSentBackReasons)
		code := reason.code
		entity.ExceptionReasonCode = &code
		entity.ExceptionNotes = reason.note
		g.bilEvent(setup, rows, entity, billingqueueevent.EventStatusChanged,
			"Moved to sent back to operations", actionAt, &biller, nil)
		rows.Comments = append(rows.Comments, g.bilSendBackComment(entity, biller, actionAt))
		entity.UpdatedAt = actionAt
	case billingqueue.StatusApproved, billingqueue.StatusPosted:
		entity.ReviewCompletedAt = &actionAt
		g.bilEvent(
			setup,
			rows,
			entity,
			billingqueueevent.EventApproved,
			"Approved the invoice for "+billingqueuereview.FormatMoney(
				total,
			),
			actionAt,
			&biller,
			nil,
		)
		entity.UpdatedAt = actionAt
	}

	return entity
}

// bilSendBackComment is the operations note billingqueueservice.SendBackComment
// posts on the shipment when an item is returned to operations.
func (g *billingPhase) bilSendBackComment(
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
		ID:             volumegen.IDAt("shc_", at),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ShipmentID:     entity.ShipmentID,
		UserID:         userID,
		Comment:        text,
		Type:           shipmentvocab.CommentTypeBilling,
		Visibility:     shipmentvocab.CommentVisibilityOperations,
		Priority:       shipmentvocab.CommentPriorityHigh,
		Source:         shipmentvocab.CommentSourceSystem,
		Metadata:       map[string]any{},
		Version:        1,
		CreatedAt:      at,
	}
}

func (g *billingPhase) bilEvent(
	setup *ledgervolume.Setup,
	rows *ledgervolume.Rows,
	entity *billingqueue.BillingQueueItem,
	kind billingqueueevent.EventKind,
	text string,
	at int64,
	actor *pulid.ID,
	payload map[string]any,
) {
	if payload == nil {
		payload = map[string]any{}
	}
	event := &billingqueueevent.ItemEvent{
		ID:             volumegen.IDAt("bqe_", at),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ItemID:         entity.ID,
		Kind:           kind,
		Text:           text,
		ActorType:      billingqueueevent.EventActorSystem,
		Payload:        payload,
		At:             at,
		CreatedAt:      at,
	}
	if actor != nil {
		id := *actor
		event.ActorType = billingqueueevent.EventActorUser
		event.ActorID = &id
		event.ActorName = bilName(setup, id)
	}
	rows.Events = append(rows.Events, event)
}

func (g *billingPhase) bilLines(r *volumegen.ShipmentRecord) []*invoice.InvoiceLine {
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
				fl.Rate.Mul(*fl.Quantity).Round(chargeallocation.SharePlaces).Equal(fl.Amount) {
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
		if def := g.Refs.Accessorial(c.Code); def != nil {
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

func (g *billingPhase) bilInvoice(
	rows *ledgervolume.Rows,
	r *volumegen.ShipmentRecord,
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
		ID:                 volumegen.IDAt("inv_", invoiceDate),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
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
		inv.ServiceDate = volumegen.Ptr(r.DeliveredAt)
	}
	ledgervolume.BillTo(inv, r.Customer)
	inv.SubtotalAmount = ledgervolume.Sum(lines, invoice.InvoiceLineTypeFreight)
	inv.OtherAmount = ledgervolume.Sum(lines, invoice.InvoiceLineTypeAccessorial)
	inv.TotalAmount = ledgervolume.Sum(lines, "")

	biller := g.Refs.AdminID
	if entity.AssignedBillerID != nil {
		biller = *entity.AssignedBillerID
	}
	if posted {
		inv.Status = invoice.StatusPosted
		inv.PostedAt = volumegen.Ptr(invoiceDate)
		g.facStamp(inv, r.Customer)
	}
	switch roll := g.Fake.Float(); {
	case !posted || r.Customer.BillingEmail == "":
	case roll < 0.92:
		sentAt := g.bilClamp(
			invoiceDate+g.Fake.Between64(volumegen.Minute, 20*volumegen.Minute),
			invoiceDate,
		)
		inv.SendStatus = invoice.SendStatusSent
		inv.SentAt = &sentAt
		inv.SentByID = biller
		inv.EmailToSnapshot = []string{r.Customer.BillingEmail}
		inv.EmailSubjectSnapshot = fmt.Sprintf("Invoice %s from %s", inv.Number, g.Refs.OrgName)
	case roll < 0.95:
		inv.SendStatus = invoice.SendStatusFailed
		inv.LastSendError = "550 5.1.1 Recipient address rejected: mailbox unavailable"
		inv.EmailToSnapshot = []string{r.Customer.BillingEmail}
	}

	for _, line := range lines {
		line.ID = volumegen.IDAt("invl_", invoiceDate)
		line.OrganizationID = g.Refs.OrgID
		line.BusinessUnitID = g.Refs.BuID
		line.InvoiceID = inv.ID
		line.CreatedAt = invoiceDate
		line.UpdatedAt = invoiceDate
	}
	inv.SyncMinorAmounts()

	rows.Invoices = append(rows.Invoices, inv)
	rows.Lines = append(rows.Lines, lines...)
	if posted {
		rows.Links = append(rows.Links, ledgervolume.ItemInvoice{Item: entity.ID, Invoice: inv.ID})
	}

	return inv
}

func (g *billingPhase) bilPostedEvents(
	setup *ledgervolume.Setup,
	rows *ledgervolume.Rows,
	entity *billingqueue.BillingQueueItem,
	inv *invoice.Invoice,
) {
	text := "Posted as " + inv.Number
	if inv.SendStatus == invoice.SendStatusSent && len(inv.EmailToSnapshot) > 0 {
		text += " · emailed to " + inv.EmailToSnapshot[0]
	}
	actor := g.Refs.AdminID
	if entity.AssignedBillerID != nil {
		actor = *entity.AssignedBillerID
	}
	g.bilEvent(setup, rows, entity, billingqueueevent.EventPosted, text, inv.InvoiceDate, &actor,
		map[string]any{"invoiceId": inv.ID.String(), "invoiceNumber": inv.Number})
	entity.UpdatedAt = max(entity.UpdatedAt, inv.InvoiceDate)
}

func bilShipmentRef(r *volumegen.ShipmentRecord) string {
	switch {
	case r.ProNumber != "":
		return r.ProNumber
	case r.BOL != "":
		return r.BOL
	default:
		return "this shipment"
	}
}

func bilName(setup *ledgervolume.Setup, id pulid.ID) string {
	if name, ok := setup.Names[id]; ok && name != "" {
		return name
	}
	return "a biller"
}

func (g *billingPhase) bilCreditMemos(
	ctx context.Context,
	setup *ledgervolume.Setup,
	rows *ledgervolume.Rows,
	invoices []*invoice.Invoice,
	owners map[pulid.ID]*volumegen.CustomerRecord,
) error {
	targets := make([]*invoice.Invoice, 0)
	dates := make([]int64, 0)
	nowUnix := volumegen.Unix(g.Now)
	for _, inv := range invoices {
		if inv.FactoringCompanyID != nil || !g.Fake.Chance(0.012) {
			continue
		}
		memoDate := inv.InvoiceDate + g.Fake.Between64(3*volumegen.Day, 40*volumegen.Day)
		if memoDate >= nowUnix-volumegen.Hour {
			continue
		}
		targets = append(targets, inv)
		dates = append(dates, memoDate)
	}
	if len(targets) == 0 {
		return nil
	}

	slots := make([]volumegen.NumberSlot, len(targets))
	for i, at := range dates {
		slots[i] = volumegen.NumberSlot{At: at}
	}
	numbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeCreditMemo, slots)
	if err != nil {
		return err
	}

	for i, target := range targets {
		open := target.OpenBalanceMinor()
		creditMinor := money.MinorUnits(
			target.TotalAmount.Mul(decimal.NewFromFloat(0.04 + g.Fake.Float()*0.11)),
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

func (g *billingPhase) bilCreditMemo(
	setup *ledgervolume.Setup,
	rows *ledgervolume.Rows,
	target *invoice.Invoice,
	cus *volumegen.CustomerRecord,
	number string,
	memoDate int64,
	creditMinor int64,
) error {
	raisedBy := g.PickUser(g.Refs.UserIDs)
	reason := volumegen.Pick(g.Fake, bilMemoReasons)
	entity := &billingqueue.BillingQueueItem{
		ID:                    volumegen.IDAt("bqi_", memoDate),
		OrganizationID:        g.Refs.OrgID,
		BusinessUnitID:        g.Refs.BuID,
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
	rows.Items = append(rows.Items, entity)

	amount := money.DecimalFromMinor(creditMinor).Neg()
	line := &invoice.InvoiceLine{
		ID:             volumegen.IDAt("invl_", memoDate),
		OrganizationID: g.Refs.OrgID,
		BusinessUnitID: g.Refs.BuID,
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
		ID:                 volumegen.IDAt("inv_", memoDate),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
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
		DueDate:            volumegen.Ptr(memoDate),
		PostedAt:           volumegen.Ptr(memoDate),
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
		ledgervolume.BillTo(memo, cus)
	} else {
		memo.BillToName = target.BillToName
	}
	line.InvoiceID = memo.ID
	memo.SyncMinorAmounts()

	rows.Invoices = append(rows.Invoices, memo)
	rows.Lines = append(rows.Lines, line)
	rows.Links = append(rows.Links, ledgervolume.ItemInvoice{Item: entity.ID, Invoice: memo.ID})
	rows.MemoCount++

	g.bilEvent(setup, rows, entity, billingqueueevent.EventPosted, "Posted as "+number, memoDate,
		&raisedBy, map[string]any{"invoiceId": memo.ID.String(), "invoiceNumber": number})

	if err := g.PostInvoice(setup, rows, memo, target.ID, entity); err != nil {
		return err
	}

	_, accountingDate, _ := g.ResolvePeriod(setup, memoDate)
	target.ApplyPaymentMinor(creditMinor)
	memo.ApplyCreditMinor(creditMinor)
	target.UpdatedAt = max(target.UpdatedAt, memoDate)
	rows.MemoApps = append(rows.MemoApps, &customerpayment.CreditMemoApplication{
		ID:                  volumegen.IDAt("cma_", memoDate),
		OrganizationID:      g.Refs.OrgID,
		BusinessUnitID:      g.Refs.BuID,
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

func (g *billingPhase) bilPlanPayment(inv *invoice.Invoice) *ledgervolume.PaymentPlan {
	open := inv.OpenBalanceMinor()
	if open <= 0 || inv.IsCreditMemo() || inv.FactoringCompanyID != nil {
		return nil
	}
	nowUnix := volumegen.Unix(g.Now)
	due := inv.InvoiceDate
	if inv.DueDate != nil {
		due = *inv.DueDate
	}
	overdueDays := (nowUnix - due) / volumegen.Day

	var full, partial float64
	switch {
	case overdueDays < 0:
		full, partial = 0.22, 0.03
	case overdueDays <= 30:
		full, partial = 0.72, 0.08
	case overdueDays <= 60:
		full, partial = 0.90, 0.04
	case overdueDays <= 90:
		full, partial = 0.95, 0.02
	default:
		full, partial = 0.985, 0.01
	}

	roll := g.Fake.Float()
	if roll >= full+partial {
		return nil
	}

	earliest := inv.InvoiceDate + volumegen.Day
	latest := nowUnix - volumegen.Hour
	if earliest > latest {
		return nil
	}
	date := due + g.Fake.Between64(-12*volumegen.Day, 21*volumegen.Day)
	if overdueDays < 0 {
		date = earliest + g.Fake.Between64(0, latest-earliest)
	}
	date = max(earliest, min(date, latest))

	amount := open
	if roll >= full {
		amount = open * int64(g.Fake.Between(30, 80)) / 100
	}
	if amount <= 0 {
		return nil
	}

	return &ledgervolume.PaymentPlan{Invoice: inv, Amount: amount, Date: date}
}

func (g *billingPhase) bilPayments(
	setup *ledgervolume.Setup,
	rows *ledgervolume.Rows,
	invoices []*invoice.Invoice,
	owners map[pulid.ID]*volumegen.CustomerRecord,
) ([]*invoicedispute.InvoiceDispute, error) {
	groups := make(map[string]*bilPaymentGroup)
	var disputes []*invoicedispute.InvoiceDispute
	for _, inv := range invoices {
		cus := owners[inv.ID]
		if cus == nil {
			continue
		}
		plan := g.bilPlanPayment(inv)
		if dispute := g.bilDispute(inv, plan); dispute != nil {
			disputes = append(disputes, dispute)
		}
		if plan == nil {
			continue
		}
		key := fmt.Sprintf("%s|%d", cus.ID, plan.Date/bilWeek)
		group, ok := groups[key]
		if !ok {
			group = &bilPaymentGroup{customer: cus, key: key}
			groups[key] = group
		}
		group.plans = append(group.plans, plan)
		group.date = max(group.date, plan.Date)
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
			if err := g.Payment(setup, rows, group.customer, group.date, plans); err != nil {
				return nil, err
			}
		}
	}

	return disputes, nil
}

func (g *billingPhase) bilCount(rows *ledgervolume.Rows) {
	invoices, drafts := 0, 0
	for _, inv := range rows.Invoices {
		switch {
		case inv.IsCreditMemo():
		case inv.Status == invoice.StatusDraft:
			drafts++
		default:
			invoices++
		}
	}
	g.Report.Add("Billing queue items", len(rows.Items))
	g.Report.Add("Billing queue events", len(rows.Events))
	g.Report.Add("Shipment comments", len(rows.Comments))
	g.Report.Add("Invoices", invoices)
	g.Report.Add("Draft invoices", drafts)
	g.Report.Add("Credit memos", rows.MemoCount)
	g.Report.Add("Invoice lines", len(rows.Lines))
	g.Report.Add("Credit memo applications", len(rows.MemoApps))
	g.Report.Add("Customer payments", len(rows.Payments))
	g.Report.Add("Payment applications", len(rows.Apps))
	g.Report.Add("Customer ledger entries", len(rows.Ledger))
	g.Report.Add("Journal entries", len(rows.Entries))
	g.Report.Add("Journal entry lines", len(rows.JournalLines))
}

var facExcludeReasons = []string{
	"Customer pays this lane directly under a separate PO",
	"Spot load billed outside the factoring agreement",
	"Customer asked to pay this invoice to us directly",
}

// facStamp records on a posting invoice the factor its customer was assigned
// to at that moment and the agreement then in force, unless the invoice is
// kept out of factoring.
func (g *billingPhase) facStamp(inv *invoice.Invoice, cus *volumegen.CustomerRecord) {
	if cus == nil || inv.PostedAt == nil {
		return
	}
	at := *inv.PostedAt
	for _, p := range cus.Factoring {
		if !p.Covers(at) {
			continue
		}
		if g.Fake.Chance(0.025) {
			inv.FactoringOverride = invoice.FactoringOverrideExclude
			inv.FactoringOverrideReason = volumegen.Pick(g.Fake, facExcludeReasons)
			return
		}
		factorID := p.Factor.Company.ID
		inv.FactoringCompanyID = &factorID
		if p.Factor.Agreement.InForce(at) {
			agreementID := p.Factor.Agreement.ID
			inv.FactoringAgreementID = &agreementID
		}
		return
	}
}

type billingPhase struct {
	*ledgervolume.Ledger
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&billingPhase{Ledger: &ledgervolume.Ledger{Generator: g}}).seedBilling(ctx)
}
