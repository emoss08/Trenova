package adjustmentvolume

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment/invoiceadjustmentdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol/accountingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol/invoiceadjustmentcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/seeds/volume/ledgervolume"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	adjAccountWriteOff     = "6940"
	adjWriteOffEvent       = accountingcontrol.JournalSourceEventType("InvoiceWriteOffCreated")
	adjBatchInlineMax      = 25
	adjReconciliationText  = "Adjustment executed against settled or finance-sensitive invoice state"
	adjReconciliationAlert = "Execution will create a reconciliation exception for finance follow-up"
)

type adjOutcome int

const (
	adjOutcomeDraft adjOutcome = iota
	adjOutcomePending
	adjOutcomeExecuted
	adjOutcomeRejected
	adjOutcomeFailed
	adjOutcomeFailedRejected
)

var (
	adjCreditReasons = []string{
		"Fuel surcharge billed at the wrong index week",
		"Detention not approved by the receiver",
		"Lumper receipt did not match the amount billed",
		"Rate adjustment agreed with the account manager",
		"Service failure credit for late delivery",
		"Duplicate accessorial charge",
	}
	adjRebillReasons = []string{
		"Billed at the spot rate; rebill at the contract rate",
		"Customer requires the PO on the invoice; rebill with the corrected reference",
		"Weight corrected from the scale ticket",
		"Wrong bill-to location on the original invoice",
	}
	adjReversalReasons = []string{
		"Shipment billed twice; reversing the duplicate",
		"Invoiced to the wrong customer",
		"Load canceled after billing",
	}
	adjWriteOffReasons = []string{
		"Short-pay under the collection threshold",
		"Customer closed; balance uncollectible",
		"Small balance left after remittance rounding",
		"Disputed accessorial written off per account manager",
	}
	adjRejectReasons = []string{
		"Credit is not supported by the POD",
		"Customer already received this credit on a prior invoice",
		"Rebill amount does not match the rate confirmation",
		"Write-off exceeds what collections agreed to",
		"Needs sign-off from the account manager first",
	}
)

type adjFieldError struct {
	field   string
	message string
}

type adjSetup struct {
	bil         *ledgervolume.Setup
	control     *invoiceadjustmentcontrol.InvoiceAdjustmentControl
	writeOff    pulid.ID
	canWriteOff bool
}

type adjRequest struct {
	rec      *volumegen.ShipmentRecord
	inv      *invoice.Invoice
	kind     invoiceadjustment.Kind
	strategy invoiceadjustment.RebillStrategy
	reason   string
	key      string
	credit   map[pulid.ID]decimal.Decimal
	rebill   map[pulid.ID]decimal.Decimal
}

type adjPreview struct {
	lines            []*invoiceadjustment.InvoiceAdjustmentLine
	creditLines      []*invoice.InvoiceLine
	replacementLines []*invoice.InvoiceLine
	credit           decimal.Decimal
	rebill           decimal.Decimal
	approval         bool
	recon            bool
	unapplied        bool
	review           bool
	accountingDate   int64
	warnings         []string
	errors           []adjFieldError
}

type adjPlan struct {
	req          *adjRequest
	outcome      adjOutcome
	pv           *adjPreview
	submitBy     pulid.ID
	submitAt     int64
	decideBy     pulid.ID
	decideAt     int64
	execAt       int64
	memoNumber   string
	rebillNumber string
	failure      string
	rejectReason string
	adjustment   *invoiceadjustment.InvoiceAdjustment
}

type adjBatchItemPlan struct {
	item *invoiceadjustment.InvoiceAdjustmentBatchItem
	plan *adjPlan
	at   int64
}

type adjBatchPlan struct {
	batch    *invoiceadjustment.InvoiceAdjustmentBatch
	items    []*adjBatchItemPlan
	inline   bool
	startErr bool
}

type adjReversal struct {
	rec    *volumegen.ShipmentRecord
	itemID pulid.ID
	actor  pulid.ID
	at     int64
	reason string
}

type adjRebillSync struct {
	shipmentID pulid.ID
	status     billingqueue.Status
	at         int64
}

type adjRows struct {
	bil         *ledgervolume.Rows
	groups      map[pulid.ID]*invoiceadjustment.InvoiceAdjustmentCorrectionGroup
	groupOrder  []*invoiceadjustment.InvoiceAdjustmentCorrectionGroup
	adjustments []*invoiceadjustment.InvoiceAdjustment
	lines       []*invoiceadjustment.InvoiceAdjustmentLine
	snapshots   []*invoiceadjustment.InvoiceAdjustmentSnapshot
	exceptions  []*invoiceadjustment.InvoiceAdjustmentReconciliationException
	batches     []*invoiceadjustment.InvoiceAdjustmentBatch
	batchItems  []*invoiceadjustment.InvoiceAdjustmentBatchItem
	sources     map[pulid.ID]*invoice.Invoice
	reversals   []*adjReversal
	rebills     []*adjRebillSync
	writeOffs   []pulid.ID
	memos       int
	replacement int
}

type adjPools struct {
	unpaid  []*volumegen.ShipmentRecord
	partial []*volumegen.ShipmentRecord
	paid    []*volumegen.ShipmentRecord
	used    map[pulid.ID]bool
}

func (g *adjustmentPhase) seedAdjustments(ctx context.Context) error {
	pools := g.adjCollect()
	total := len(pools.unpaid) + len(pools.partial) + len(pools.paid)
	if total == 0 {
		return nil
	}

	setup, err := g.adjLoadSetup(ctx)
	if err != nil {
		return err
	}
	if setup == nil {
		g.Opts.Logf("  ! no invoice adjustment control; skipping invoice adjustments")
		return nil
	}

	plans := g.adjPlanIndividual(setup, pools, max(12, total*3/100))
	batches, batchPlans := g.adjPlanBatches(setup, pools, total)
	plans = append(plans, batchPlans...)

	if err = g.adjNumber(ctx, plans); err != nil {
		return err
	}

	rows := &adjRows{
		bil: &ledgervolume.Rows{
			Balances: make(map[ledgervolume.BalanceKey]*ledgervolume.Balance),
		},
		groups:  make(map[pulid.ID]*invoiceadjustment.InvoiceAdjustmentCorrectionGroup),
		sources: make(map[pulid.ID]*invoice.Invoice),
	}
	sort.SliceStable(plans, func(i, j int) bool { return plans[i].submitAt < plans[j].submitAt })
	for _, p := range plans {
		if err = g.adjBuild(setup, rows, p); err != nil {
			return err
		}
	}
	for _, b := range batches {
		g.adjBuildBatch(rows, b)
	}

	if err = g.Materialize(ctx, rows.bil); err != nil {
		return err
	}
	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		return g.adjWrite(ctx, tx, rows)
	}); err != nil {
		return err
	}

	g.adjCount(rows)
	return nil
}

func (g *adjustmentPhase) adjCollect() *adjPools {
	pools := &adjPools{used: make(map[pulid.ID]bool)}
	for _, r := range g.Shipments {
		inv := r.Invoice
		if inv == nil || inv.Status != invoice.StatusPosted || inv.IsCreditMemo() ||
			inv.BillType != billingqueue.BillTypeInvoice || len(inv.Lines) == 0 ||
			inv.FactoringCompanyID != nil {
			continue
		}
		switch inv.SettlementStatus {
		case invoice.SettlementStatusPaid:
			pools.paid = append(pools.paid, r)
		case invoice.SettlementStatusPartiallyPaid:
			pools.partial = append(pools.partial, r)
		default:
			if inv.AppliedAmountMinor == 0 {
				pools.unpaid = append(pools.unpaid, r)
			} else {
				pools.partial = append(pools.partial, r)
			}
		}
	}
	for _, pool := range [][]*volumegen.ShipmentRecord{pools.unpaid, pools.partial, pools.paid} {
		for i := len(pool) - 1; i > 0; i-- {
			j := g.Fake.Intn(i + 1)
			pool[i], pool[j] = pool[j], pool[i]
		}
	}
	return pools
}

func (p *adjPools) take(
	pool []*volumegen.ShipmentRecord,
	accept func(*volumegen.ShipmentRecord) bool,
) *volumegen.ShipmentRecord {
	for _, r := range pool {
		if p.used[r.Invoice.ID] {
			continue
		}
		if accept != nil && !accept(r) {
			continue
		}
		p.used[r.Invoice.ID] = true
		return r
	}
	return nil
}

func (g *adjustmentPhase) adjLoadSetup(ctx context.Context) (*adjSetup, error) {
	control, err := volumegen.FirstOrNil(
		ctx,
		g.DB,
		invoiceadjustmentcontroldb.InvoiceAdjustmentControls.Query(),
	)
	if err != nil {
		return nil, fmt.Errorf("load invoice adjustment control: %w", err)
	}
	if control == nil {
		return nil, nil
	}

	bil, err := g.LoadSetup(ctx)
	if err != nil {
		return nil, err
	}
	setup := &adjSetup{bil: bil, control: control}

	accounting, err := g.AccountingControl(ctx)
	if err != nil {
		return nil, err
	}
	if accounting == nil {
		return setup, nil
	}
	setup.writeOff = accounting.DefaultWriteOffAccountID
	if setup.writeOff.IsNil() {
		byCode, codeErr := g.GLAccountsByCode(ctx, adjAccountWriteOff)
		if codeErr != nil {
			return nil, codeErr
		}
		if id, ok := byCode[adjAccountWriteOff]; ok {
			ac := accountingcontroldb.AccountingControls
			if _, err = ac.Where(ac.DefaultWriteOffAccountID.IsNull()).
				Update(ctx, g.DB, ac.DefaultWriteOffAccountID.To(id)); err != nil {
				return nil, fmt.Errorf("set default write-off account: %w", err)
			}
			setup.writeOff = id
		}
	}
	journalsAllowed := accounting.JournalPostingMode != accountingcontrol.JournalPostingModeManual ||
		accounting.ManualJournalEntryPolicy != accountingcontrol.ManualJournalEntryPolicyDisallow
	setup.canWriteOff = setup.writeOff.IsNotNil() && bil.AR.IsNotNil() && journalsAllowed &&
		len(bil.Periods) > 0

	return setup, nil
}

func (g *adjustmentPhase) adjPlanIndividual(
	setup *adjSetup,
	pools *adjPools,
	target int,
) []*adjPlan {
	outcomes := []adjOutcome{
		adjOutcomeDraft, adjOutcomePending, adjOutcomeExecuted, adjOutcomeRejected,
		adjOutcomeFailed, adjOutcomeFailedRejected,
	}
	outcomeWeights := []int{8, 26, 46, 12, 5, 3}
	kinds := []invoiceadjustment.Kind{
		invoiceadjustment.KindCreditOnly, invoiceadjustment.KindCreditRebill,
		invoiceadjustment.KindFullReversal, invoiceadjustment.KindWriteOff,
	}
	kindWeights := []int{42, 26, 14, 18}

	plans := make([]*adjPlan, 0, target)
	for attempt := 0; len(plans) < target && attempt < target*4; attempt++ {
		outcome := outcomes[g.Fake.Weighted(outcomeWeights)]
		kind := kinds[g.Fake.Weighted(kindWeights)]
		if outcome == adjOutcomeDraft {
			kind = invoiceadjustment.KindCreditOnly
		}
		if p := g.adjPlanOne(setup, pools, kind, outcome); p != nil {
			plans = append(plans, p)
		}
	}
	return plans
}

func (g *adjustmentPhase) adjPlanOne(
	setup *adjSetup,
	pools *adjPools,
	kind invoiceadjustment.Kind,
	outcome adjOutcome,
) *adjPlan {
	now := volumegen.Unix(g.Now)
	var rec *volumegen.ShipmentRecord
	switch {
	case outcome == adjOutcomeFailed || outcome == adjOutcomeFailedRejected:
		kind = invoiceadjustment.KindWriteOff
		rec = pools.take(pools.paid, nil)
	case kind == invoiceadjustment.KindFullReversal:
		rec = pools.take(pools.unpaid, nil)
	case kind == invoiceadjustment.KindWriteOff:
		if g.Fake.Chance(0.5) {
			rec = pools.take(pools.partial, nil)
		}
		if rec == nil {
			rec = pools.take(pools.unpaid, func(r *volumegen.ShipmentRecord) bool {
				return r.Invoice.DueDate != nil && now-*r.Invoice.DueDate > 60*volumegen.Day
			})
		}
	default:
		pool := [][]*volumegen.ShipmentRecord{pools.unpaid, pools.partial, pools.paid}[g.Fake.Weighted([]int{50, 20, 30})]
		rec = pools.take(pool, nil)
		if rec == nil {
			rec = pools.take(pools.unpaid, nil)
		}
	}
	if rec == nil {
		return nil
	}

	req := g.adjRequestFor(rec, kind)
	if req == nil {
		return nil
	}
	submitAt := g.adjSubmitTime(rec.Invoice, outcome)
	p := &adjPlan{
		req:      req,
		outcome:  outcome,
		submitAt: submitAt,
		submitBy: g.PickUser(g.Refs.UserIDs),
	}

	if outcome == adjOutcomeDraft {
		return p
	}

	state := rec.Invoice
	if outcome == adjOutcomeFailed || outcome == adjOutcomeFailedRejected {
		before := *rec.Invoice
		before.AppliedAmount = decimal.Zero
		before.AppliedAmountMinor = 0
		before.SettlementStatus = invoice.SettlementStatusUnpaid
		state = &before
	}
	p.pv = g.adjPreview(setup, req, state, submitAt)
	if len(p.pv.errors) > 0 {
		return nil
	}
	return g.adjSettleOutcome(setup, p)
}

func (g *adjustmentPhase) adjSettleOutcome(setup *adjSetup, p *adjPlan) *adjPlan {
	now := volumegen.Unix(g.Now)
	failed := p.outcome == adjOutcomeFailed || p.outcome == adjOutcomeFailedRejected
	if !p.pv.approval {
		if failed {
			return nil
		}
		p.outcome = adjOutcomeExecuted
	}
	if p.outcome == adjOutcomeExecuted && p.req.kind == invoiceadjustment.KindWriteOff &&
		!setup.canWriteOff {
		if !p.pv.approval {
			return nil
		}
		p.outcome = adjOutcomePending
	}

	if p.pv.approval && p.outcome != adjOutcomePending {
		approver, ok := g.adjOtherUser(p.submitBy)
		if !ok && setup.control.RequireSeparateApprover {
			if failed {
				return nil
			}
			p.outcome = adjOutcomePending
		} else {
			p.decideBy = approver
			p.decideAt = min(p.submitAt+g.Fake.Between64(30*volumegen.Minute, 4*volumegen.Day), now-20*volumegen.Minute)
			p.decideAt = max(p.decideAt, p.submitAt+volumegen.Minute)
		}
	}

	switch p.outcome {
	case adjOutcomeExecuted:
		p.execAt = p.submitAt
		if p.pv.approval {
			p.execAt = p.decideAt
		}
	case adjOutcomeRejected:
		p.rejectReason = volumegen.Pick(g.Fake, adjRejectReasons)
	case adjOutcomeFailed, adjOutcomeFailedRejected:
		p.failure = "validation failed:\n- creditTotalAmount: Write-off amount cannot exceed the invoice open balance" +
			"\n- invoiceId: Write-offs require a remaining invoice balance"
		if p.outcome == adjOutcomeFailedRejected {
			p.rejectReason = "Invoice was paid in full before the write-off was approved"
		}
	case adjOutcomeDraft, adjOutcomePending:
	}
	return p
}

func (g *adjustmentPhase) adjOtherUser(not pulid.ID) (pulid.ID, bool) {
	candidates := make([]pulid.ID, 0, len(g.Refs.UserIDs))
	for _, id := range g.Refs.UserIDs {
		if id != not {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return not, false
	}
	return volumegen.Pick(g.Fake, candidates), true
}

func (g *adjustmentPhase) adjSubmitTime(inv *invoice.Invoice, outcome adjOutcome) int64 {
	now := volumegen.Unix(g.Now)
	earliest := inv.InvoiceDate + volumegen.Day
	latest := now - 2*volumegen.Hour
	if outcome == adjOutcomePending || outcome == adjOutcomeDraft {
		earliest = max(earliest, now-25*volumegen.Day)
	}
	if earliest >= latest {
		return latest
	}
	return g.Fake.Between64(earliest, latest)
}

func (g *adjustmentPhase) adjRequestFor(
	rec *volumegen.ShipmentRecord,
	kind invoiceadjustment.Kind,
) *adjRequest {
	inv := rec.Invoice
	req := &adjRequest{
		rec:      rec,
		inv:      inv,
		kind:     kind,
		strategy: invoiceadjustment.RebillStrategyCloneExact,
		key:      "adj-" + inv.Number + "-" + g.Fake.Digits(6),
	}

	switch kind {
	case invoiceadjustment.KindCreditOnly:
		req.reason = volumegen.Pick(g.Fake, adjCreditReasons)
		req.credit = g.adjPartialCredit(inv)
	case invoiceadjustment.KindCreditRebill:
		req.reason = volumegen.Pick(g.Fake, adjRebillReasons)
		if g.Fake.Chance(0.55) {
			req.strategy = invoiceadjustment.RebillStrategyManual
			req.credit = make(map[pulid.ID]decimal.Decimal, len(inv.Lines))
			req.rebill = make(map[pulid.ID]decimal.Decimal, len(inv.Lines))
			for _, line := range inv.Lines {
				amount := line.Amount.Abs()
				req.credit[line.ID] = amount
				if line.Type == invoice.InvoiceLineTypeFreight {
					amount = amount.Mul(decimal.NewFromFloat(0.88 + g.Fake.Float()*0.2)).Round(2)
				}
				req.rebill[line.ID] = amount
			}
		}
	case invoiceadjustment.KindFullReversal:
		req.reason = volumegen.Pick(g.Fake, adjReversalReasons)
	case invoiceadjustment.KindWriteOff:
		req.reason = volumegen.Pick(g.Fake, adjWriteOffReasons)
		open := inv.OpenBalanceAmount()
		if open.LessThanOrEqual(decimal.Zero) {
			open = inv.TotalAmount.Mul(decimal.NewFromFloat(0.05 + g.Fake.Float()*0.1)).Round(2)
		}
		req.credit = make(map[pulid.ID]decimal.Decimal, len(inv.Lines))
		remaining := open
		for _, line := range inv.Lines {
			if remaining.LessThanOrEqual(decimal.Zero) {
				break
			}
			amount := decimal.Min(remaining, line.Amount.Abs())
			req.credit[line.ID] = amount
			remaining = remaining.Sub(amount)
		}
	}
	return req
}

func (g *adjustmentPhase) adjPartialCredit(inv *invoice.Invoice) map[pulid.ID]decimal.Decimal {
	var accessorials []*invoice.InvoiceLine
	var freight *invoice.InvoiceLine
	for _, line := range inv.Lines {
		switch line.Type {
		case invoice.InvoiceLineTypeAccessorial:
			if line.Amount.GreaterThan(decimal.Zero) {
				accessorials = append(accessorials, line)
			}
		case invoice.InvoiceLineTypeFreight:
			if freight == nil {
				freight = line
			}
		}
	}
	credit := make(map[pulid.ID]decimal.Decimal, 1)
	switch {
	case len(accessorials) > 0 && (freight == nil || g.Fake.Chance(0.6)):
		line := volumegen.Pick(g.Fake, accessorials)
		amount := line.Amount
		if g.Fake.Chance(0.4) {
			amount = amount.Mul(decimal.NewFromFloat(0.25 + g.Fake.Float()*0.5)).Round(2)
		}
		credit[line.ID] = amount
	case freight != nil:
		credit[freight.ID] = freight.Amount.Mul(decimal.NewFromFloat(0.04 + g.Fake.Float()*0.16)).
			Round(2)
	default:
		line := inv.Lines[0]
		credit[line.ID] = line.Amount.Abs().Mul(decimal.NewFromFloat(0.1)).Round(2)
	}
	return credit
}

func (g *adjustmentPhase) adjOpenPeriod(setup *adjSetup, date int64) *fiscalperiod.FiscalPeriod {
	for _, p := range setup.bil.Periods {
		if p.StartDate <= date && date <= p.EndDate {
			return p
		}
	}
	return nil
}

func (g *adjustmentPhase) adjAccountingDate(
	setup *adjSetup,
	inv *invoice.Invoice,
	at int64,
	pv *adjPreview,
) int64 {
	next := func() int64 {
		if p := g.adjOpenPeriod(setup, at); p != nil {
			return max(p.StartDate, inv.InvoiceDate)
		}
		return at
	}
	control := setup.control
	if control.AdjustmentAccountingDatePolicy == invoiceadjustmentcontrol.AdjustmentAccountingDateAlwaysNextOpen {
		return next()
	}
	if g.adjOpenPeriod(setup, inv.InvoiceDate) != nil {
		return inv.InvoiceDate
	}
	switch control.ClosedPeriodAdjustmentPolicy {
	case invoiceadjustmentcontrol.ClosedPeriodAdjustmentPolicyDisallow:
		pv.errors = append(pv.errors, adjFieldError{
			"accountingDate", "Closed-period adjustments are disallowed by policy",
		})
	case invoiceadjustmentcontrol.ClosedPeriodAdjustmentPolicyRequireReopen:
		pv.errors = append(pv.errors, adjFieldError{
			"accountingDate",
			"The accounting period must be reopened before this adjustment can be posted",
		})
	case invoiceadjustmentcontrol.ClosedPeriodAdjustmentPolicyPostInNextOpenPeriodWithApproval:
		pv.approval = true
		pv.warnings = append(
			pv.warnings,
			"Adjustment will post in the next open period and requires approval",
		)
	}
	return next()
}

//nolint:gocognit,funlen // mirrors invoiceadjustmentservice.computePreview step for step
func (g *adjustmentPhase) adjPreview(
	setup *adjSetup,
	req *adjRequest,
	inv *invoice.Invoice,
	at int64,
) *adjPreview {
	control := setup.control
	pv := &adjPreview{credit: decimal.Zero, rebill: decimal.Zero}
	fail := func(field, message string) {
		pv.errors = append(pv.errors, adjFieldError{field, message})
	}

	if inv.Status != invoice.StatusPosted {
		fail("invoiceId", "Only posted invoices may be adjusted")
	}
	if req.kind == invoiceadjustment.KindFullReversal && inv.AppliedAmountMinor > 0 {
		fail(
			"invoiceId",
			"Unapply the customer payments and credit memos on this invoice before reversing it in full",
		)
	}
	pv.accountingDate = g.adjAccountingDate(setup, inv, at, pv)

	switch inv.SettlementStatus {
	case invoice.SettlementStatusPaid:
		switch control.PaidInvoiceAdjustmentPolicy {
		case invoiceadjustmentcontrol.AdjustmentEligibilityDisallow:
			fail("invoiceId", "Paid invoices cannot be adjusted by policy")
		case invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithApproval:
			pv.approval = true
		case invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithoutApproval:
		}
	case invoice.SettlementStatusPartiallyPaid:
		switch control.PartiallyPaidInvoiceAdjustmentPolicy {
		case invoiceadjustmentcontrol.AdjustmentEligibilityDisallow:
			fail("invoiceId", "Partially paid invoices cannot be adjusted by policy")
		case invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithApproval:
			pv.approval = true
		case invoiceadjustmentcontrol.AdjustmentEligibilityAllowWithoutApproval:
		}
	case invoice.SettlementStatusUnpaid:
	}
	if inv.SettlementStatus != invoice.SettlementStatusUnpaid {
		pv.recon = true
	}
	if strings.TrimSpace(req.reason) == "" &&
		control.AdjustmentReasonRequirement == invoiceadjustmentcontrol.RequirementPolicyRequired {
		fail("reason", "Adjustment reason is required by policy")
	}

	fullScope := req.credit == nil || req.kind == invoiceadjustment.KindFullReversal
	for _, source := range inv.Lines {
		input, has := req.credit[source.ID]
		if !fullScope && !has {
			continue
		}
		eligible := source.Amount.Abs()
		credit := eligible
		creditQty := source.Quantity
		rebill := decimal.Zero
		rebillQty := decimal.Zero
		if !fullScope {
			if !input.IsZero() {
				credit = input
			}
			if amount, ok := req.rebill[source.ID]; ok && !amount.IsZero() {
				rebill = amount
				rebillQty = source.Quantity
			}
		}
		if req.kind == invoiceadjustment.KindCreditRebill && fullScope &&
			req.strategy != invoiceadjustment.RebillStrategyRerate {
			rebill = source.Amount.Abs()
			rebillQty = source.Quantity
		}
		if credit.GreaterThan(eligible) {
			fail("lines", fmt.Sprintf("Line %d exceeds the remaining eligible amount by %s",
				source.LineNumber, credit.Sub(eligible).StringFixed(4)))
		}

		pv.credit = pv.credit.Add(credit)
		pv.rebill = pv.rebill.Add(rebill)
		pv.lines = append(pv.lines, &invoiceadjustment.InvoiceAdjustmentLine{
			OriginalInvoiceID:       inv.ID,
			OriginalLineID:          source.ID,
			LineNumber:              source.LineNumber,
			Description:             source.Description,
			CreditQuantity:          creditQty,
			CreditAmount:            credit,
			RemainingEligibleAmount: eligible.Sub(credit),
			RebillQuantity:          rebillQty,
			RebillAmount:            rebill,
			ReplacementPayload:      map[string]any{},
		})
		if credit.GreaterThan(decimal.Zero) {
			unit := credit
			if creditQty.GreaterThan(decimal.Zero) {
				unit = credit.Div(creditQty)
			}
			line := &invoice.InvoiceLine{
				LineNumber:  source.LineNumber,
				Type:        source.Type,
				Description: source.Description,
				Quantity:    creditQty,
				UnitPrice:   unit.Neg(),
				Amount:      credit.Neg(),
			}
			line.CopyChargeDetail(source)
			pv.creditLines = append(pv.creditLines, line)
		}
		if rebill.GreaterThan(decimal.Zero) {
			unit := rebill
			if rebillQty.GreaterThan(decimal.Zero) {
				unit = rebill.Div(rebillQty)
			}
			line := &invoice.InvoiceLine{
				LineNumber:  len(pv.replacementLines) + 1,
				Type:        source.Type,
				Description: source.Description,
				Quantity:    decimal.Max(rebillQty, decimal.NewFromInt(1)),
				UnitPrice:   unit,
				Amount:      rebill,
			}
			line.CopyChargeDetail(source)
			pv.replacementLines = append(pv.replacementLines, line)
		}
	}

	open := inv.OpenBalanceAmount()
	pv.unapplied = pv.credit.GreaterThan(open)
	if req.kind == invoiceadjustment.KindWriteOff && pv.credit.GreaterThan(open) {
		fail("creditTotalAmount", "Write-off amount cannot exceed the invoice open balance")
	}

	if pv.unapplied {
		if req.kind == invoiceadjustment.KindWriteOff {
			fail("creditTotalAmount", "Write-offs cannot create unapplied customer credit")
		} else {
			switch control.CustomerCreditBalancePolicy {
			case invoiceadjustmentcontrol.CustomerCreditBalancePolicyDisallow:
				fail("creditTotalAmount", "Policy disallows customer credit balance outcomes")
			case invoiceadjustmentcontrol.CustomerCreditBalancePolicyAllowUnappliedCredit:
				switch control.OverCreditPolicy {
				case invoiceadjustmentcontrol.OverCreditPolicyBlock:
					fail("creditTotalAmount",
						"Adjustment would create unapplied customer credit because of payment state")
				case invoiceadjustmentcontrol.OverCreditPolicyAllowWithApproval:
					pv.approval = true
				}
			}
		}
	}

	amount := pv.credit.Abs()
	if req.kind == invoiceadjustment.KindWriteOff {
		switch control.WriteOffApprovalPolicy {
		case invoiceadjustmentcontrol.WriteOffApprovalPolicyDisallow:
			fail("kind", "Write-offs are disallowed by policy")
		case invoiceadjustmentcontrol.WriteOffApprovalPolicyAlwaysRequireApproval:
			pv.approval = true
		case invoiceadjustmentcontrol.WriteOffApprovalPolicyRequireApprovalAboveThreshold:
			if amount.GreaterThanOrEqual(control.WriteOffApprovalThreshold) {
				pv.approval = true
			}
		}
	} else {
		switch control.StandardAdjustmentApprovalPolicy {
		case invoiceadjustmentcontrol.ApprovalPolicyAlways:
			pv.approval = true
		case invoiceadjustmentcontrol.ApprovalPolicyAmountThreshold:
			if amount.GreaterThanOrEqual(control.StandardAdjustmentApprovalThreshold) {
				pv.approval = true
			}
		case invoiceadjustmentcontrol.ApprovalPolicyNone:
		}
	}

	if req.kind == invoiceadjustment.KindCreditRebill {
		switch control.ReplacementInvoiceReviewPolicy {
		case invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyAlwaysRequireReview:
			pv.review = true
		case invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyRequireReviewWhenEconomicTermsChange:
			pv.review = !pv.rebill.Equal(pv.credit)
		case invoiceadjustmentcontrol.ReplacementInvoiceReviewPolicyNoAdditionalReview:
		}
	}

	if pv.credit.IsZero() {
		fail("lines", "Adjustment must produce a non-zero credit amount")
	}
	if req.kind == invoiceadjustment.KindWriteOff && open.IsZero() {
		fail("invoiceId", "Write-offs require a remaining invoice balance")
	}
	if pv.recon {
		pv.warnings = append(pv.warnings, adjReconciliationAlert)
	}
	return pv
}

func (g *adjustmentPhase) adjPlanBatches(
	setup *adjSetup,
	pools *adjPools,
	total int,
) ([]*adjBatchPlan, []*adjPlan) {
	now := volumegen.Unix(g.Now)
	statuses := []invoiceadjustment.BatchStatus{
		invoiceadjustment.BatchStatusCompleted,
		invoiceadjustment.BatchStatusPartial,
		invoiceadjustment.BatchStatusFailed,
		invoiceadjustment.BatchStatusRunning,
		invoiceadjustment.BatchStatusSubmitted,
		invoiceadjustment.BatchStatusCompleted,
		invoiceadjustment.BatchStatusPartial,
		invoiceadjustment.BatchStatusFailed,
		invoiceadjustment.BatchStatusCompleted,
	}
	count := volumegen.Clamp(total/700, 5, 30)

	byCustomer := make(map[pulid.ID][]*volumegen.ShipmentRecord)
	var customers []pulid.ID
	for _, pool := range [][]*volumegen.ShipmentRecord{pools.unpaid, pools.partial, pools.paid} {
		for _, r := range pool {
			id := r.Invoice.CustomerID
			if _, ok := byCustomer[id]; !ok {
				customers = append(customers, id)
			}
			byCustomer[id] = append(byCustomer[id], r)
		}
	}
	if len(customers) == 0 {
		return nil, nil
	}
	sort.SliceStable(customers, func(i, j int) bool {
		return len(byCustomer[customers[i]]) > len(byCustomer[customers[j]])
	})

	var batches []*adjBatchPlan
	var plans []*adjPlan
	for i := range count {
		status := statuses[i%len(statuses)]
		startErr := status == invoiceadjustment.BatchStatusFailed && i%2 == 1
		temporal := status == invoiceadjustment.BatchStatusRunning ||
			status == invoiceadjustment.BatchStatusSubmitted || startErr || g.Fake.Chance(0.3)
		size := g.Fake.Between(4, adjBatchInlineMax)
		if temporal {
			size = g.Fake.Between(adjBatchInlineMax+1, 48)
		}

		customerID := customers[i%len(customers)]
		candidates := byCustomer[customerID]
		submitBy := g.PickUser(g.Refs.UserIDs)
		submitted := now - g.Fake.Between64(2*volumegen.Day, 150*volumegen.Day)
		if status == invoiceadjustment.BatchStatusRunning ||
			status == invoiceadjustment.BatchStatusSubmitted {
			submitted = now - g.Fake.Between64(5*volumegen.Minute, 50*volumegen.Minute)
		}
		key := "bulk-" + g.Fake.Letters(4) + "-" + g.Fake.Digits(8)
		batch := &invoiceadjustment.InvoiceAdjustmentBatch{
			ID:             volumegen.IDAt("iadjb_", submitted),
			OrganizationID: g.Refs.OrgID,
			BusinessUnitID: g.Refs.BuID,
			IdempotencyKey: key,
			Status:         status,
			SubmittedByID:  submitBy,
			SubmittedAt:    volumegen.Ptr(submitted),
			Metadata:       map[string]any{"processingMode": "inline"},
			Version:        1,
			CreatedAt:      submitted,
			UpdatedAt:      submitted,
		}
		if temporal {
			batch.Metadata["processingMode"] = "temporal"
		}
		bp := &adjBatchPlan{batch: batch, inline: !temporal, startErr: startErr}
		reason := volumegen.Pick(g.Fake, adjCreditReasons)

		var picked []*volumegen.ShipmentRecord
		for _, r := range candidates {
			if len(picked) == size {
				break
			}
			if !pools.used[r.Invoice.ID] && r.Invoice.InvoiceDate < submitted-volumegen.Day {
				pools.used[r.Invoice.ID] = true
				picked = append(picked, r)
			}
		}
		for _, pool := range [][]*volumegen.ShipmentRecord{pools.unpaid, pools.partial, pools.paid} {
			for _, r := range pool {
				if len(picked) == size {
					break
				}
				if !pools.used[r.Invoice.ID] && r.Invoice.InvoiceDate < submitted-volumegen.Day {
					pools.used[r.Invoice.ID] = true
					picked = append(picked, r)
				}
			}
		}
		if len(picked) == 0 {
			continue
		}
		batch.TotalCount = len(picked)

		failShare := 0.0
		switch status {
		case invoiceadjustment.BatchStatusPartial:
			failShare = 0.3
		case invoiceadjustment.BatchStatusFailed:
			failShare = 1
		case invoiceadjustment.BatchStatusRunning:
			failShare = 0.1
		case invoiceadjustment.BatchStatusCompleted, invoiceadjustment.BatchStatusSubmitted,
			invoiceadjustment.BatchStatusQueued, invoiceadjustment.BatchStatusPending:
		}
		processedTo := len(picked)
		if status == invoiceadjustment.BatchStatusRunning {
			processedTo = max(1, len(picked)*g.Fake.Between(25, 70)/100)
		}

		at := submitted
		for idx, r := range picked {
			req := g.adjRequestFor(r, invoiceadjustment.KindCreditOnly)
			req.reason = reason
			req.key = fmt.Sprintf("%s:%s", key, r.Invoice.Number)
			item := &invoiceadjustment.InvoiceAdjustmentBatchItem{
				ID:             volumegen.IDAt("iadjbi_", submitted),
				OrganizationID: g.Refs.OrgID,
				BusinessUnitID: g.Refs.BuID,
				BatchID:        batch.ID,
				InvoiceID:      r.Invoice.ID,
				IdempotencyKey: fmt.Sprintf("%s-%d", key, idx),
				Status:         invoiceadjustment.BatchItemStatusPending,
				ResultPayload:  map[string]any{},
				CreatedAt:      submitted,
				UpdatedAt:      submitted,
			}
			ip := &adjBatchItemPlan{item: item, at: submitted}
			bp.items = append(bp.items, ip)

			switch {
			case startErr || status == invoiceadjustment.BatchStatusSubmitted:
				item.RequestPayload = g.adjRequestPayload(req)
				continue
			case idx >= processedTo:
				item.RequestPayload = g.adjRequestPayload(req)
				if idx == processedTo {
					item.Status = invoiceadjustment.BatchItemStatusExecuting
				}
				continue
			}

			at = min(at+g.Fake.Between64(2, 40), now-volumegen.Minute)
			ip.at = at
			if failShare >= 1 || g.Fake.Chance(failShare) {
				g.adjFailItem(req, item)
				continue
			}
			pv := g.adjPreview(setup, req, r.Invoice, at)
			if len(pv.errors) > 0 {
				item.RequestPayload = g.adjRequestPayload(req)
				item.Status = invoiceadjustment.BatchItemStatusFailed
				item.ErrorMessage = adjErrorText(pv.errors)
				continue
			}
			item.RequestPayload = g.adjRequestPayload(req)
			outcome := adjOutcomeExecuted
			if pv.approval {
				outcome = []adjOutcome{
					adjOutcomePending, adjOutcomeExecuted, adjOutcomeRejected,
				}[g.Fake.Weighted([]int{55, 35, 10})]
			}
			p := &adjPlan{req: req, outcome: outcome, pv: pv, submitAt: at, submitBy: submitBy}
			if settled := g.adjSettleOutcome(setup, p); settled != nil {
				ip.plan = settled
				plans = append(plans, settled)
			}
		}
		batches = append(batches, bp)
	}
	return batches, plans
}

func (g *adjustmentPhase) adjFailItem(
	req *adjRequest,
	item *invoiceadjustment.InvoiceAdjustmentBatchItem,
) {
	inv := req.inv
	if inv.AppliedAmountMinor > 0 {
		req.kind = invoiceadjustment.KindFullReversal
		req.credit = nil
		item.ErrorMessage = adjErrorText([]adjFieldError{{
			"invoiceId",
			"Unapply the customer payments and credit memos on this invoice before reversing it in full",
		}})
	} else {
		line := inv.Lines[0]
		over := line.Amount.Abs().Add(g.Fake.Money(25, 400))
		req.credit = map[pulid.ID]decimal.Decimal{line.ID: over}
		item.ErrorMessage = adjErrorText([]adjFieldError{{
			"lines",
			fmt.Sprintf("Line %d exceeds the remaining eligible amount by %s",
				line.LineNumber, over.Sub(line.Amount.Abs()).StringFixed(4)),
		}})
	}
	item.RequestPayload = g.adjRequestPayload(req)
	item.Status = invoiceadjustment.BatchItemStatusFailed
}

func adjErrorText(errs []adjFieldError) string {
	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		messages = append(messages, e.field+": "+e.message)
	}
	return "validation failed:\n- " + strings.Join(messages, "\n- ")
}

func (g *adjustmentPhase) adjRequestPayload(req *adjRequest) map[string]any {
	request := &invoiceadjustmentports.InvoiceAdjustmentRequest{
		InvoiceID:      req.inv.ID,
		Kind:           req.kind,
		RebillStrategy: req.strategy,
		Reason:         req.reason,
		IdempotencyKey: req.key,
		TenantInfo:     pagination.TenantInfo{OrgID: g.Refs.OrgID, BuID: g.Refs.BuID},
	}
	for _, line := range req.inv.Lines {
		credit, ok := req.credit[line.ID]
		if !ok {
			continue
		}
		input := &invoiceadjustmentports.InvoiceAdjustmentLineInput{
			OriginalLineID: line.ID,
			CreditAmount:   credit,
			Description:    line.Description,
		}
		if rebill, has := req.rebill[line.ID]; has {
			input.RebillAmount = rebill
		}
		request.Lines = append(request.Lines, input)
	}
	return jsonutils.MustToJSON(request)
}

func (g *adjustmentPhase) adjNumber(ctx context.Context, plans []*adjPlan) error {
	var memoSlots, rebillSlots []volumegen.NumberSlot
	var memoPlans, rebillPlans []*adjPlan
	for _, p := range plans {
		if p.outcome != adjOutcomeExecuted {
			continue
		}
		memoSlots = append(memoSlots, volumegen.NumberSlot{At: p.execAt})
		memoPlans = append(memoPlans, p)
		if p.req.kind == invoiceadjustment.KindCreditRebill {
			rebillSlots = append(rebillSlots, volumegen.NumberSlot{At: p.execAt})
			rebillPlans = append(rebillPlans, p)
		}
	}
	memoNumbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeCreditMemo, memoSlots)
	if err != nil {
		return err
	}
	for i, p := range memoPlans {
		p.memoNumber = memoNumbers[i]
	}
	rebillNumbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeInvoice, rebillSlots)
	if err != nil {
		return err
	}
	for i, p := range rebillPlans {
		p.rebillNumber = rebillNumbers[i]
	}
	return nil
}

func (g *adjustmentPhase) adjGroup(
	rows *adjRows,
	inv *invoice.Invoice,
	at int64,
) *invoiceadjustment.InvoiceAdjustmentCorrectionGroup {
	if group, ok := rows.groups[inv.ID]; ok {
		return group
	}
	group := &invoiceadjustment.InvoiceAdjustmentCorrectionGroup{
		ID:               volumegen.IDAt("icg_", at),
		OrganizationID:   g.Refs.OrgID,
		BusinessUnitID:   g.Refs.BuID,
		RootInvoiceID:    inv.ID,
		CurrentInvoiceID: inv.ID,
		Metadata:         map[string]any{},
		CreatedAt:        at,
		UpdatedAt:        at,
	}
	rows.groups[inv.ID] = group
	rows.groupOrder = append(rows.groupOrder, group)
	inv.CorrectionGroupID = group.ID
	inv.UpdatedAt = max(inv.UpdatedAt, at)
	rows.sources[inv.ID] = inv
	return group
}

func adjSnapshotPayload(inv *invoice.Invoice) map[string]any {
	payload := jsonutils.MustToJSON(inv)
	payload["paymentSummary"] = map[string]any{
		"appliedAmount":     inv.AppliedAmount,
		"openBalanceAmount": inv.OpenBalanceAmount(),
		"settlementStatus":  inv.SettlementStatus,
		"disputeStatus":     inv.DisputeStatus,
	}
	return payload
}

//nolint:funlen // one adjustment's rows are built together so their links agree
func (g *adjustmentPhase) adjBuild(setup *adjSetup, rows *adjRows, p *adjPlan) error {
	req := p.req
	inv := req.inv
	group := g.adjGroup(rows, inv, p.submitAt)
	at := p.submitAt

	adj := &invoiceadjustment.InvoiceAdjustment{
		ID:                      volumegen.IDAt("iadj_", at),
		OrganizationID:          g.Refs.OrgID,
		BusinessUnitID:          g.Refs.BuID,
		CorrectionGroupID:       group.ID,
		OriginalInvoiceID:       inv.ID,
		Kind:                    req.kind,
		Status:                  invoiceadjustment.StatusDraft,
		ApprovalStatus:          invoiceadjustment.ApprovalStatusNotRequired,
		ReplacementReviewStatus: invoiceadjustment.ReplacementReviewStatusNotRequired,
		RebillStrategy:          req.strategy,
		IdempotencyKey:          req.key,
		AccountingDate:          at,
		CreditTotalAmount:       decimal.Zero,
		RebillTotalAmount:       decimal.Zero,
		NetDeltaAmount:          decimal.Zero,
		RerateVariancePercent:   decimal.Zero,
		Metadata:                map[string]any{},
		Version:                 1,
		CreatedAt:               at,
		UpdatedAt:               at,
	}
	p.adjustment = adj

	if p.outcome == adjOutcomeDraft {
		adj.Kind = invoiceadjustment.KindCreditOnly
		adj.RebillStrategy = invoiceadjustment.RebillStrategyCloneExact
		adj.IdempotencyKey = volumegen.IDAt("iadjkey_", at).String()
		adj.Metadata["draft"] = true
		for _, line := range inv.Lines {
			rows.lines = append(rows.lines, &invoiceadjustment.InvoiceAdjustmentLine{
				ID:                      volumegen.IDAt("iadjl_", at),
				OrganizationID:          g.Refs.OrgID,
				BusinessUnitID:          g.Refs.BuID,
				AdjustmentID:            adj.ID,
				OriginalInvoiceID:       inv.ID,
				OriginalLineID:          line.ID,
				LineNumber:              line.LineNumber,
				Description:             line.Description,
				CreditQuantity:          line.Quantity,
				CreditAmount:            line.Amount.Abs(),
				RemainingEligibleAmount: line.Amount.Abs(),
				RebillQuantity:          line.Quantity,
				RebillAmount:            line.Amount.Abs(),
				ReplacementPayload:      map[string]any{},
				CreatedAt:               at,
				UpdatedAt:               at,
			})
		}
		adj.SyncMinorAmounts()
		rows.adjustments = append(rows.adjustments, adj)
		return nil
	}

	pv := p.pv
	adj.Reason = req.reason
	adj.PolicyReason = strings.Join(pv.warnings, " | ")
	adj.AccountingDate = pv.accountingDate
	adj.CreditTotalAmount = pv.credit
	adj.RebillTotalAmount = pv.rebill
	adj.NetDeltaAmount = pv.rebill.Sub(pv.credit)
	adj.WouldCreateUnappliedCredit = pv.unapplied
	adj.RequiresReconciliationException = pv.recon
	adj.ApprovalRequired = pv.approval
	if pv.review {
		adj.ReplacementReviewStatus = invoiceadjustment.ReplacementReviewStatusRequired
	}
	adj.SubmittedByID = p.submitBy
	adj.SubmittedAt = volumegen.Ptr(at)
	adj.Metadata["attachmentIds"] = []any{}
	adj.Status = invoiceadjustment.StatusApproved
	if pv.approval {
		adj.Status = invoiceadjustment.StatusPendingApproval
		adj.ApprovalStatus = invoiceadjustment.ApprovalStatusPending
	}

	for _, line := range pv.lines {
		line.ID = volumegen.IDAt("iadjl_", at)
		line.OrganizationID = g.Refs.OrgID
		line.BusinessUnitID = g.Refs.BuID
		line.AdjustmentID = adj.ID
		line.CreatedAt = at
		line.UpdatedAt = at
		rows.lines = append(rows.lines, line)
	}
	rows.snapshots = append(rows.snapshots, &invoiceadjustment.InvoiceAdjustmentSnapshot{
		ID:             volumegen.IDAt("iadjs_", at),
		OrganizationID: g.Refs.OrgID,
		BusinessUnitID: g.Refs.BuID,
		AdjustmentID:   adj.ID,
		InvoiceID:      inv.ID,
		Kind:           invoiceadjustment.SnapshotKindSubmission,
		Payload:        adjSnapshotPayload(inv),
		CreatedByID:    p.submitBy,
		CreatedAt:      at,
	})

	switch p.outcome {
	case adjOutcomeExecuted:
		if err := g.adjExecute(setup, rows, p, group); err != nil {
			return err
		}
	case adjOutcomeRejected:
		adj.Reject(p.decideBy, p.decideAt, p.rejectReason)
		adj.UpdatedAt = p.decideAt
	case adjOutcomeFailed, adjOutcomeFailedRejected:
		adj.Status = invoiceadjustment.StatusExecutionFailed
		adj.ExecutionError = p.failure
		adj.UpdatedAt = p.decideAt
		if p.outcome == adjOutcomeFailedRejected {
			rejectAt := min(
				p.decideAt+g.Fake.Between64(volumegen.Hour, 3*volumegen.Day),
				volumegen.Unix(g.Now)-volumegen.Minute,
			)
			adj.Reject(p.decideBy, max(rejectAt, p.decideAt), p.rejectReason)
			adj.UpdatedAt = max(rejectAt, p.decideAt)
		}
	case adjOutcomeDraft, adjOutcomePending:
	}

	adj.SyncMinorAmounts()
	rows.adjustments = append(rows.adjustments, adj)
	return nil
}

//nolint:funlen // mirrors invoiceadjustmentservice.executeApprovedAdjustment
func (g *adjustmentPhase) adjExecute(
	setup *adjSetup,
	rows *adjRows,
	p *adjPlan,
	group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
) error {
	adj := p.adjustment
	req := p.req
	source := req.inv
	pv := p.pv
	at := p.execAt
	actor := p.submitBy
	if adj.ApprovalRequired {
		actor = p.decideBy
		adj.ApprovedByID = p.decideBy
		adj.ApprovedAt = volumegen.Ptr(at)
		adj.ApprovalStatus = invoiceadjustment.ApprovalStatusApproved
	}

	memoItem := &billingqueue.BillingQueueItem{
		ID:                        volumegen.IDAt("bqi_", at),
		OrganizationID:            g.Refs.OrgID,
		BusinessUnitID:            g.Refs.BuID,
		ShipmentID:                source.ShipmentID,
		OrderID:                   source.OrderID,
		BillToCustomerID:          source.CustomerID,
		AllocatedTotalAmount:      source.TotalAmount.Abs(),
		Number:                    p.memoNumber,
		Status:                    billingqueue.StatusPosted,
		BillType:                  billingqueue.BillTypeCreditMemo,
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           volumegen.Ptr(source.ID),
		SourceInvoiceAdjustmentID: volumegen.Ptr(adj.ID),
		CorrectionGroupID:         volumegen.Ptr(group.ID),
		RequiresReplacementReview: pv.review,
		RerateVariancePercent:     decimal.Zero,
		AdjustmentContext:         map[string]any{"kind": req.kind},
		CreatedAt:                 at,
		UpdatedAt:                 at,
	}
	memoItem.SyncAllocatedMinor()
	rows.bil.Items = append(rows.bil.Items, memoItem)

	subtotal, other := decimal.Zero, decimal.Zero
	for _, line := range pv.creditLines {
		if line.Type == invoice.InvoiceLineTypeFreight {
			subtotal = subtotal.Add(line.Amount)
		} else {
			other = other.Add(line.Amount)
		}
	}
	memo := &invoice.Invoice{
		ID:                        volumegen.IDAt("inv_", at),
		OrganizationID:            g.Refs.OrgID,
		BusinessUnitID:            g.Refs.BuID,
		BillingQueueItemID:        memoItem.ID,
		Scope:                     invoice.ScopeAdjustment,
		ShipmentID:                source.ShipmentID,
		OrderID:                   source.OrderID,
		OrderNumber:               source.OrderNumber,
		CustomerID:                source.CustomerID,
		Detail:                    source.Detail,
		SectionBy:                 source.SectionBy,
		Number:                    memoItem.Number,
		BillType:                  billingqueue.BillTypeCreditMemo,
		Status:                    invoice.StatusPosted,
		PaymentTerm:               source.PaymentTerm,
		CurrencyCode:              source.CurrencyCode,
		InvoiceDate:               pv.accountingDate,
		DueDate:                   volumegen.Ptr(pv.accountingDate),
		PostedAt:                  volumegen.Ptr(at),
		ShipmentProNumber:         source.ShipmentProNumber,
		ShipmentBOL:               source.ShipmentBOL,
		ServiceDate:               source.ServiceDate,
		BillToName:                source.BillToName,
		BillToCode:                source.BillToCode,
		BillToAddressLine1:        source.BillToAddressLine1,
		BillToAddressLine2:        source.BillToAddressLine2,
		BillToCity:                source.BillToCity,
		BillToState:               source.BillToState,
		BillToPostalCode:          source.BillToPostalCode,
		BillToCountry:             source.BillToCountry,
		ShipperCustomerID:         source.ShipperCustomerID,
		IsSplitBill:               source.IsSplitBill,
		SubtotalAmount:            subtotal,
		OtherAmount:               other,
		TotalAmount:               pv.credit.Neg(),
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		SendStatus:                invoice.SendStatusNotSent,
		EDISendStatus:             invoice.EDISendStatusNotSent,
		FactoringOverride:         invoice.FactoringOverrideDefault,
		CorrectionGroupID:         group.ID,
		SourceInvoiceAdjustmentID: adj.ID,
		IsAdjustmentArtifact:      true,
		Lines:                     pv.creditLines,
		CreatedAt:                 at,
		UpdatedAt:                 at,
	}
	g.adjAttachLines(rows, memo, at)
	memo.SyncMinorAmounts()
	rows.bil.Invoices = append(rows.bil.Invoices, memo)
	rows.memos++
	adj.CreditMemoInvoiceID = memo.ID

	if entryID, err := g.adjPostCreditMemo(setup, rows, adj, memo, source, actor); err != nil {
		return err
	} else if entryID.IsNotNil() {
		adj.Metadata["creditMemoJournalEntryId"] = entryID.String()
	}

	if applied := min(memo.CreditRemainingMinor(), source.OpenBalanceMinor()); applied > 0 {
		rows.bil.MemoApps = append(rows.bil.MemoApps, &customerpayment.CreditMemoApplication{
			ID:                  volumegen.IDAt("cma_", at),
			OrganizationID:      g.Refs.OrgID,
			BusinessUnitID:      g.Refs.BuID,
			CreditMemoInvoiceID: memo.ID,
			InvoiceID:           source.ID,
			AppliedAmountMinor:  applied,
			AccountingDate:      memo.InvoiceDate,
			LineNumber:          1,
			Status:              customerpayment.CreditApplicationStatusApplied,
			CreatedByID:         actor,
			CreatedAt:           at,
			UpdatedAt:           at,
		})
		source.ApplyPaymentMinor(applied)
		memo.ApplyCreditMinor(applied)
	}
	source.UpdatedAt = max(source.UpdatedAt, at)
	rows.sources[source.ID] = source

	var rebillItemID, replacementID any
	if req.kind == invoiceadjustment.KindCreditRebill {
		replacement := g.adjRebill(rows, p, memo, group)
		rebillItemID = adj.RebillQueueItemID.String()
		replacementID = replacement.ID.String()
		group.CurrentInvoiceID = replacement.ID
	}

	var writeOffID any
	if req.kind == invoiceadjustment.KindWriteOff {
		entryID, err := g.adjWriteOffJournal(setup, rows, adj, source, actor)
		if err != nil {
			return err
		}
		adj.Metadata["writeOffJournalEntryId"] = entryID.String()
		writeOffID = entryID.String()
	}

	if pv.recon {
		rows.exceptions = append(
			rows.exceptions,
			&invoiceadjustment.InvoiceAdjustmentReconciliationException{
				ID:                  volumegen.IDAt("irex_", at),
				OrganizationID:      g.Refs.OrgID,
				BusinessUnitID:      g.Refs.BuID,
				AdjustmentID:        adj.ID,
				InvoiceID:           source.ID,
				CreditMemoInvoiceID: memo.ID,
				Status:              invoiceadjustment.ExceptionStatusOpen,
				Reason:              adjReconciliationText,
				Amount:              pv.credit,
				Metadata: map[string]any{
					"settlementStatus": source.SettlementStatus,
					"disputeStatus":    source.DisputeStatus,
				},
				CreatedAt: at,
				UpdatedAt: at,
			},
		)
	}

	rows.snapshots = append(rows.snapshots, &invoiceadjustment.InvoiceAdjustmentSnapshot{
		ID:             volumegen.IDAt("iadjs_", at),
		OrganizationID: g.Refs.OrgID,
		BusinessUnitID: g.Refs.BuID,
		AdjustmentID:   adj.ID,
		InvoiceID:      source.ID,
		Kind:           invoiceadjustment.SnapshotKindExecution,
		CreatedByID:    actor,
		Payload: map[string]any{
			"sourceInvoice":          adjSnapshotPayload(source),
			"creditMemoId":           memo.ID.String(),
			"rebillQueueItemId":      rebillItemID,
			"replacementInvoiceId":   replacementID,
			"writeOffJournalEntryId": writeOffID,
		},
		CreatedAt: at,
	})

	if req.kind == invoiceadjustment.KindFullReversal {
		g.adjVoid(rows, p, source, actor)
	}

	adj.Status = invoiceadjustment.StatusExecuted
	adj.UpdatedAt = at
	group.UpdatedAt = max(group.UpdatedAt, at)
	return nil
}

func (g *adjustmentPhase) adjAttachLines(rows *adjRows, inv *invoice.Invoice, at int64) {
	for _, line := range inv.Lines {
		line.ID = volumegen.IDAt("invl_", at)
		line.OrganizationID = g.Refs.OrgID
		line.BusinessUnitID = g.Refs.BuID
		line.InvoiceID = inv.ID
		line.ShipmentID = inv.ShipmentID
		line.ShipmentProNumber = inv.ShipmentProNumber
		line.ShipmentBOL = inv.ShipmentBOL
		line.CreatedAt = at
		line.UpdatedAt = at
	}
	rows.bil.Lines = append(rows.bil.Lines, inv.Lines...)
}

func (g *adjustmentPhase) adjRebill(
	rows *adjRows,
	p *adjPlan,
	memo *invoice.Invoice,
	group *invoiceadjustment.InvoiceAdjustmentCorrectionGroup,
) *invoice.Invoice {
	adj := p.adjustment
	source := p.req.inv
	pv := p.pv
	at := p.execAt

	status := billingqueue.StatusApproved
	if pv.review {
		status = billingqueue.StatusReadyForReview
	}
	item := &billingqueue.BillingQueueItem{
		ID:                        volumegen.IDAt("bqi_", at),
		OrganizationID:            g.Refs.OrgID,
		BusinessUnitID:            g.Refs.BuID,
		ShipmentID:                source.ShipmentID,
		OrderID:                   source.OrderID,
		BillToCustomerID:          source.CustomerID,
		AllocatedTotalAmount:      pv.rebill.Abs(),
		Number:                    p.rebillNumber,
		Status:                    status,
		BillType:                  billingqueue.BillTypeInvoice,
		IsAdjustmentOrigin:        true,
		SourceInvoiceID:           volumegen.Ptr(source.ID),
		SourceInvoiceAdjustmentID: volumegen.Ptr(adj.ID),
		SourceCreditMemoInvoiceID: volumegen.Ptr(memo.ID),
		CorrectionGroupID:         volumegen.Ptr(group.ID),
		RebillStrategy:            string(adj.RebillStrategy),
		RequiresReplacementReview: pv.review,
		RerateVariancePercent:     decimal.Zero,
		AdjustmentContext: map[string]any{
			"replacementLines": jsonutils.MustToJSON(map[string]any{"lines": pv.replacementLines})["lines"],
			"subtotalAmount": ledgervolume.Sum(
				pv.replacementLines,
				invoice.InvoiceLineTypeFreight,
			),
			"otherAmount": ledgervolume.Sum(
				pv.replacementLines,
				invoice.InvoiceLineTypeAccessorial,
			),
			"totalAmount":        pv.rebill,
			"accountingDate":     pv.accountingDate,
			"sourceInvoiceId":    source.ID,
			"correctionGroupId":  group.ID,
			"sourceAdjustmentId": adj.ID,
		},
		CreatedAt: at,
		UpdatedAt: at,
	}
	item.SyncAllocatedMinor()
	rows.bil.Items = append(rows.bil.Items, item)

	replacement := &invoice.Invoice{
		ID:                 volumegen.IDAt("inv_", at),
		OrganizationID:     g.Refs.OrgID,
		BusinessUnitID:     g.Refs.BuID,
		BillingQueueItemID: item.ID,
		Scope:              invoice.ScopeAdjustment,
		ShipmentID:         source.ShipmentID,
		CustomerID:         source.CustomerID,
		Detail:             source.Detail,
		SectionBy:          source.SectionBy,
		Number:             item.Number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusDraft,
		PaymentTerm:        source.PaymentTerm,
		CurrencyCode:       source.CurrencyCode,
		InvoiceDate:        pv.accountingDate,
		DueDate: invoice.DueDateFromPaymentTerm(
			pv.accountingDate,
			source.PaymentTerm,
		),
		ShipmentProNumber:  source.ShipmentProNumber,
		ShipmentBOL:        source.ShipmentBOL,
		ServiceDate:        source.ServiceDate,
		BillToName:         source.BillToName,
		BillToCode:         source.BillToCode,
		BillToAddressLine1: source.BillToAddressLine1,
		BillToAddressLine2: source.BillToAddressLine2,
		BillToCity:         source.BillToCity,
		BillToState:        source.BillToState,
		BillToPostalCode:   source.BillToPostalCode,
		BillToCountry:      source.BillToCountry,
		ShipperCustomerID:  source.ShipperCustomerID,
		IsSplitBill:        source.IsSplitBill,
		SubtotalAmount: ledgervolume.Sum(
			pv.replacementLines,
			invoice.InvoiceLineTypeFreight,
		),
		OtherAmount: ledgervolume.Sum(
			pv.replacementLines,
			invoice.InvoiceLineTypeAccessorial,
		),
		TotalAmount:               pv.rebill,
		AppliedAmount:             decimal.Zero,
		SettlementStatus:          invoice.SettlementStatusUnpaid,
		DisputeStatus:             invoice.DisputeStatusNone,
		SendStatus:                invoice.SendStatusNotSent,
		EDISendStatus:             invoice.EDISendStatusNotSent,
		FactoringOverride:         invoice.FactoringOverrideDefault,
		CorrectionGroupID:         group.ID,
		SupersedesInvoiceID:       source.ID,
		SourceInvoiceAdjustmentID: adj.ID,
		IsAdjustmentArtifact:      true,
		Lines:                     pv.replacementLines,
		CreatedAt:                 at,
		UpdatedAt:                 at,
	}
	g.adjAttachLines(rows, replacement, at)
	replacement.SyncMinorAmounts()
	rows.bil.Invoices = append(rows.bil.Invoices, replacement)
	rows.replacement++

	source.SupersededByInvoiceID = replacement.ID
	adj.ReplacementInvoiceID = replacement.ID
	adj.RebillQueueItemID = item.ID
	rows.rebills = append(rows.rebills, &adjRebillSync{
		shipmentID: source.ShipmentID,
		status:     status,
		at:         at,
	})
	return replacement
}

func (g *adjustmentPhase) adjPostCreditMemo(
	setup *adjSetup,
	rows *adjRows,
	adj *invoiceadjustment.InvoiceAdjustment,
	memo, source *invoice.Invoice,
	actor pulid.ID,
) (pulid.ID, error) {
	if !setup.bil.PostInvoices {
		return pulid.Nil, nil
	}
	amount := memo.CreditTotalMinor()
	if amount == 0 {
		return pulid.Nil, nil
	}
	event := accountingcontrol.JournalSourceEventCreditMemoPosted
	ledgerDate := memo.InvoiceDate
	entryID := pulid.Nil

	if adj.Kind != invoiceadjustment.KindWriteOff {
		journal, err := g.NewJournal(setup.bil, memo.InvoiceDate)
		if err != nil {
			return pulid.Nil, err
		}
		journal.ActorID = actor
		journal.Description = "Credit memo posted for " + memo.Number
		journal.SourceType = "Invoice"
		journal.SourceID = memo.ID
		journal.Document = memo.Number
		journal.Event = event
		journal.ReferenceType = event.String()
		journal.Idempotency = "invoice-posted:" + memo.ID.String()
		journal.Auto = true
		journal.Legs = []ledgervolume.Leg{
			{Account: setup.bil.Revenue, Debit: amount, Customer: memo.CustomerID},
			{Account: setup.bil.AR, Credit: amount, Customer: memo.CustomerID},
		}
		rows.bil.Journals = append(rows.bil.Journals, journal)
		ledgerDate = journal.Date
		entryID = journal.EntryID
	}

	rows.bil.Ledger = append(rows.bil.Ledger, &customerledger.CustomerLedgerEntry{
		ID:               volumegen.IDAt("cledg_", *memo.PostedAt),
		OrganizationID:   g.Refs.OrgID,
		BusinessUnitID:   g.Refs.BuID,
		CustomerID:       memo.CustomerID,
		SourceObjectType: "Invoice",
		SourceObjectID:   memo.ID.String(),
		SourceEventType:  event.String(),
		RelatedInvoiceID: source.ID,
		DocumentNumber:   memo.Number,
		TransactionDate:  ledgerDate,
		LineNumber:       1,
		AmountMinor:      -amount,
		CreatedByID:      actor,
		CreatedAt:        *memo.PostedAt,
	})
	return entryID, nil
}

func (g *adjustmentPhase) adjWriteOffJournal(
	setup *adjSetup,
	rows *adjRows,
	adj *invoiceadjustment.InvoiceAdjustment,
	source *invoice.Invoice,
	actor pulid.ID,
) (pulid.ID, error) {
	amount := money.MinorUnits(adj.CreditTotalAmount.Abs())
	journal, err := g.NewJournal(setup.bil, adj.AccountingDate)
	if err != nil {
		return pulid.Nil, err
	}
	journal.ActorID = actor
	journal.Description = "Invoice write-off for " + source.Number
	journal.SourceType = "InvoiceAdjustment"
	journal.SourceID = adj.ID
	journal.Document = source.Number
	journal.Event = adjWriteOffEvent
	journal.ReferenceType = "InvoiceAdjustmentWriteOff"
	journal.Idempotency = "invoice-writeoff:" + adj.ID.String()
	journal.Auto = true
	journal.Legs = []ledgervolume.Leg{
		{Account: setup.writeOff, Debit: amount, Customer: source.CustomerID},
		{Account: setup.bil.AR, Credit: amount, Customer: source.CustomerID},
	}
	rows.bil.Journals = append(rows.bil.Journals, journal)
	rows.writeOffs = append(rows.writeOffs, journal.EntryID)
	return journal.EntryID, nil
}

func (g *adjustmentPhase) adjVoid(
	rows *adjRows,
	p *adjPlan,
	source *invoice.Invoice,
	actor pulid.ID,
) {
	at := p.execAt
	reason := strings.TrimSpace(p.req.reason)
	if reason == "" {
		reason = "Reversed in full by adjustment " + p.adjustment.ID.String()
	}
	source.Status = invoice.StatusVoided
	source.VoidedAt = volumegen.Ptr(at)
	source.VoidedByID = actor
	source.VoidReason = reason
	source.VoidDisposition = invoice.VoidDispositionDoNotRebill
	source.VoidedByAdjustmentID = p.adjustment.ID
	source.UpdatedAt = max(source.UpdatedAt, at)

	rows.reversals = append(rows.reversals, &adjReversal{
		rec:    p.req.rec,
		itemID: source.BillingQueueItemID,
		actor:  actor,
		at:     at,
		reason: adjCancelReason(reason),
	})
	p.req.rec.Status = shipmentvocab.StatusCompleted
}

func adjCancelReason(reason string) string {
	const limit = 100
	prefix := "Invoice voided: "
	if len(prefix)+len(reason) <= limit {
		return prefix + reason
	}
	return prefix + reason[:limit-len(prefix)]
}

func (g *adjustmentPhase) adjBuildBatch(rows *adjRows, b *adjBatchPlan) {
	batch := b.batch
	now := volumegen.Unix(g.Now)
	last := batch.CreatedAt
	for _, ip := range b.items {
		item := ip.item
		if ip.plan != nil && ip.plan.adjustment != nil {
			adj := ip.plan.adjustment
			item.AdjustmentID = adj.ID
			item.Status = invoiceadjustment.BatchItemStatusExecuted
			if ip.plan.pv.approval {
				item.Status = invoiceadjustment.BatchItemStatusPendingApproval
			}
			if b.inline {
				item.ResultPayload = jsonutils.MustToJSON(adj)
			} else {
				status := invoiceadjustment.StatusExecuted
				if ip.plan.pv.approval {
					status = invoiceadjustment.StatusPendingApproval
				}
				item.ResultPayload = map[string]any{"adjustmentId": adj.ID, "status": status}
			}
		}
		if item.Status != invoiceadjustment.BatchItemStatusPending {
			item.UpdatedAt = min(ip.at, now)
			last = max(last, item.UpdatedAt)
		}
		rows.batchItems = append(rows.batchItems, item)
	}

	processed, succeeded, failed := 0, 0, 0
	for _, ip := range b.items {
		switch ip.item.Status {
		case invoiceadjustment.BatchItemStatusExecuted,
			invoiceadjustment.BatchItemStatusPendingApproval,
			invoiceadjustment.BatchItemStatusRejected:
			processed++
			succeeded++
		case invoiceadjustment.BatchItemStatusFailed:
			processed++
			failed++
		case invoiceadjustment.BatchItemStatusPending, invoiceadjustment.BatchItemStatusPreviewed,
			invoiceadjustment.BatchItemStatusSubmitted, invoiceadjustment.BatchItemStatusExecuting:
		}
	}
	batch.ProcessedCount = processed
	batch.SucceededCount = succeeded
	batch.FailedCount = failed

	workflowID := fmt.Sprintf("invoice-adjustment-batch-%s-%s-%s",
		g.Refs.OrgID.String(), g.Refs.BuID.String(), batch.ID.String())
	switch {
	case b.startErr:
		batch.Status = invoiceadjustment.BatchStatusFailed
		batch.Metadata["workflowError"] = kernelports.ErrWorkflowStarterDisabled.Error()
	case batch.Status == invoiceadjustment.BatchStatusSubmitted:
		batch.Metadata["workflowId"] = workflowID
		batch.Metadata["runId"] = adjRunID(g.Generator)
	default:
		if !b.inline {
			batch.Metadata["workflowId"] = workflowID
			batch.Metadata["runId"] = adjRunID(g.Generator)
		}
		switch {
		case processed == 0, processed < batch.TotalCount:
			batch.Status = invoiceadjustment.BatchStatusRunning
		case failed == 0:
			batch.Status = invoiceadjustment.BatchStatusCompleted
		case succeeded == 0:
			batch.Status = invoiceadjustment.BatchStatusFailed
		default:
			batch.Status = invoiceadjustment.BatchStatusPartial
		}
	}
	batch.UpdatedAt = last
	rows.batches = append(rows.batches, batch)
}

func adjRunID(g *volumegen.Generator) string {
	const hex = "0123456789abcdef"
	var b strings.Builder
	for i := range 36 {
		switch i {
		case 8, 13, 18, 23:
			b.WriteByte('-')
		default:
			b.WriteByte(hex[g.Fake.Intn(len(hex))])
		}
	}
	return b.String()
}

//nolint:funlen // writes run in foreign-key order
func (g *adjustmentPhase) adjWrite(ctx context.Context, tx *keel.DB, rows *adjRows) error {
	// A group's current invoice is often a credit memo or replacement written
	// below, so the group starts at its root invoice and is re-pointed after.
	current := make([]pulid.ID, len(rows.groupOrder))
	for i, group := range rows.groupOrder {
		current[i] = group.CurrentInvoiceID
		group.CurrentInvoiceID = group.RootInvoiceID
	}
	err := ledgervolume.Insert(
		ctx,
		tx,
		invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups.Handle,
		rows.groupOrder,
		func(r *invoiceadjustment.InvoiceAdjustmentCorrectionGroup) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.CreatedAt}
		},
		false,
	)
	for i, group := range rows.groupOrder {
		group.CurrentInvoiceID = current[i]
	}
	if err != nil {
		return err
	}
	if err := g.adjUpdateGroupLinks(ctx, tx, rows, true); err != nil {
		return err
	}
	if err := g.Write(ctx, tx, rows.bil); err != nil {
		return err
	}
	if err := ledgervolume.Insert(ctx, tx, invoiceadjustmentdb.InvoiceAdjustments.Handle, rows.adjustments,
		func(r *invoiceadjustment.InvoiceAdjustment) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := ledgervolume.Insert(ctx, tx, invoiceadjustmentdb.InvoiceAdjustmentLines.Handle, rows.lines,
		func(r *invoiceadjustment.InvoiceAdjustmentLine) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := ledgervolume.Insert(ctx, tx, invoiceadjustmentdb.InvoiceAdjustmentSnapshots.Handle, rows.snapshots,
		func(r *invoiceadjustment.InvoiceAdjustmentSnapshot) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt}
		}, true); err != nil {
		return err
	}
	if err := ledgervolume.Insert(ctx, tx, invoiceadjustmentdb.InvoiceAdjustmentReconciliationExceptions.Handle, rows.exceptions,
		func(r *invoiceadjustment.InvoiceAdjustmentReconciliationException) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := ledgervolume.Insert(ctx, tx, invoiceadjustmentdb.InvoiceAdjustmentBatches.Handle, rows.batches,
		func(r *invoiceadjustment.InvoiceAdjustmentBatch) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := ledgervolume.Insert(ctx, tx, invoiceadjustmentdb.InvoiceAdjustmentBatchItems.Handle, rows.batchItems,
		func(r *invoiceadjustment.InvoiceAdjustmentBatchItem) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := g.adjUpdateSources(ctx, tx, rows); err != nil {
		return err
	}
	if err := g.adjUpdateGroupLinks(ctx, tx, rows, false); err != nil {
		return err
	}
	if err := g.adjApplyReversals(ctx, tx, rows); err != nil {
		return err
	}
	if err := g.adjSyncRebills(ctx, tx, rows); err != nil {
		return err
	}
	if len(rows.writeOffs) > 0 {
		je := journalentrydb.JournalEntries
		if _, err := je.Where(je.ID.In(rows.writeOffs...)).
			Update(ctx, tx, je.EntryType.To(journalentry.EntryTypeAdjusting), volumegen.KeepUpdated(je.Handle)); err != nil {
			return fmt.Errorf("mark write-off journals adjusting: %w", err)
		}
	}
	return nil
}

func (g *adjustmentPhase) adjUpdateGroupLinks(
	ctx context.Context,
	tx *keel.DB,
	rows *adjRows,
	onInvoices bool,
) error {
	if len(rows.groupOrder) == 0 {
		return nil
	}
	ids := make([]string, len(rows.groupOrder))
	values := make([]string, len(rows.groupOrder))
	for i, group := range rows.groupOrder {
		if onInvoices {
			ids[i] = group.RootInvoiceID.String()
			values[i] = group.ID.String()
		} else {
			ids[i] = group.ID.String()
			values[i] = group.CurrentInvoiceID.String()
		}
	}
	d := keel.Unnest("d", []string{"id", "value"}, ids, values)
	var err error
	if onInvoices {
		inv := invoicedb.Invoices
		_, err = inv.With(d).Join(d, d.Col("id").EqCol(inv.ID)).Where(keel.True()).
			Update(ctx, tx, inv.CorrectionGroupID.ToExpr(d.Col("value")), volumegen.KeepUpdated(inv.Handle))
	} else {
		cg := invoiceadjustmentdb.InvoiceAdjustmentCorrectionGroups
		_, err = cg.With(d).Join(d, d.Col("id").EqCol(cg.ID)).Where(keel.True()).
			Update(ctx, tx, cg.CurrentInvoiceID.ToExpr(d.Col("value")), volumegen.KeepUpdated(cg.Handle))
	}
	if err != nil {
		return fmt.Errorf("link correction groups: %w", err)
	}
	return nil
}

func (g *adjustmentPhase) adjUpdateSources(ctx context.Context, tx *keel.DB, rows *adjRows) error {
	n := len(rows.sources)
	if n == 0 {
		return nil
	}
	ids := make([]string, 0, n)
	superseded := make([]string, 0, n)
	applied := make([]string, 0, n)
	appliedMinor := make([]int64, 0, n)
	settlement := make([]string, 0, n)
	status := make([]string, 0, n)
	voidedAt := make([]int64, 0, n)
	voidedBy := make([]string, 0, n)
	voidReason := make([]string, 0, n)
	disposition := make([]string, 0, n)
	voidedByAdj := make([]string, 0, n)
	updated := make([]int64, 0, n)
	for _, inv := range rows.sources {
		ids = append(ids, inv.ID.String())
		superseded = append(superseded, inv.SupersededByInvoiceID.String())
		applied = append(applied, inv.AppliedAmount.StringFixed(4))
		appliedMinor = append(appliedMinor, inv.AppliedAmountMinor)
		settlement = append(settlement, string(inv.SettlementStatus))
		status = append(status, string(inv.Status))
		var at int64
		if inv.VoidedAt != nil {
			at = *inv.VoidedAt
		}
		voidedAt = append(voidedAt, at)
		voidedBy = append(voidedBy, inv.VoidedByID.String())
		voidReason = append(voidReason, inv.VoidReason)
		disposition = append(disposition, string(inv.VoidDisposition))
		voidedByAdj = append(voidedByAdj, inv.VoidedByAdjustmentID.String())
		updated = append(updated, inv.UpdatedAt)
	}
	d := keel.Unnest("d", []string{
		"id", "superseded", "applied", "applied_minor", "settlement", "status",
		"voided_at", "voided_by", "void_reason", "disposition", "voided_by_adj", "updated",
	}, ids, superseded, applied, appliedMinor, settlement, status,
		voidedAt, voidedBy, voidReason, disposition, voidedByAdj, updated)
	inv := invoicedb.Invoices
	if _, err := inv.With(d).Join(d, d.Col("id").EqCol(inv.ID)).Where(keel.True()).Update(
		ctx,
		tx,
		inv.SupersededByInvoiceID.ToExpr(volumegen.NullIfEmpty(d.Col("superseded"))),
		inv.AppliedAmount.ToExpr(keel.CastNumeric(d.Col("applied"))),
		inv.AppliedAmountMinor.ToExpr(d.Col("applied_minor")),
		inv.SettlementStatus.ToExpr(d.Col("settlement")),
		inv.Status.ToExpr(d.Col("status")),
		inv.VoidedAt.ToExpr(keel.Int64Of(d.Col("voided_at")).NullIf(0)),
		inv.VoidedByID.ToExpr(volumegen.NullIfEmpty(d.Col("voided_by"))),
		inv.VoidReason.ToExpr(volumegen.NullIfEmpty(d.Col("void_reason"))),
		inv.VoidDisposition.ToExpr(volumegen.NullIfEmpty(d.Col("disposition"))),
		inv.VoidedByAdjustmentID.ToExpr(volumegen.NullIfEmpty(d.Col("voided_by_adj"))),
		inv.UpdatedAt.ToExpr(d.Col("updated")),
	); err != nil {
		return fmt.Errorf("update adjusted invoices: %w", err)
	}
	return nil
}

func (g *adjustmentPhase) adjApplyReversals(ctx context.Context, tx *keel.DB, rows *adjRows) error {
	if len(rows.reversals) == 0 {
		return nil
	}
	itemIDs := make([]string, 0, len(rows.reversals))
	actors := make([]string, 0, len(rows.reversals))
	ats := make([]int64, 0, len(rows.reversals))
	reasons := make([]string, 0, len(rows.reversals))
	shipments := make([]string, 0, len(rows.reversals))
	orders := make([]string, 0, len(rows.reversals))
	for _, r := range rows.reversals {
		itemIDs = append(itemIDs, r.itemID.String())
		actors = append(actors, r.actor.String())
		ats = append(ats, r.at)
		reasons = append(reasons, r.reason)
		shipments = append(shipments, r.rec.ID.String())
		orders = append(orders, r.rec.OrderID.String())
	}
	bq := billingqueuedb.BillingQueueItems
	items := keel.Unnest(
		"d",
		[]string{"id", "actor", "at", "reason"},
		itemIDs,
		actors,
		ats,
		reasons,
	)
	if _, err := bq.With(items).Join(items, items.Col("id").EqCol(bq.ID)).Where(keel.True()).Update(
		ctx,
		tx,
		bq.Status.To(billingqueue.StatusCanceled),
		bq.InvoiceID.ToNull(),
		bq.CanceledByID.ToExpr(items.Col("actor")),
		bq.CanceledAt.ToExpr(items.Col("at")),
		bq.CancelReason.ToExpr(items.Col("reason")),
		bq.UpdatedAt.ToExpr(items.Col("at")),
	); err != nil {
		return fmt.Errorf("cancel reversed billing queue items: %w", err)
	}
	sh := shipmentdb.Shipments
	restored := keel.Unnest("d", []string{"id", "at"}, shipments, ats)
	if _, err := sh.With(restored).Join(restored, restored.Col("id").EqCol(sh.ID)).Where(keel.True()).Update(
		ctx,
		tx,
		sh.Status.To(shipmentvocab.StatusCompleted),
		sh.BilledAt.ToNull(),
		sh.RateLocked.To(false),
		sh.BillingTransferStatus.To(shipmentvocab.BillingTransferStatus(billingqueue.StatusCanceled)),
		sh.UpdatedAt.ToExpr(restored.Col("at")),
	); err != nil {
		return fmt.Errorf("restore reversed shipments: %w", err)
	}
	o := orderdb.Orders
	reopened := keel.Unnest("d", []string{"id", "at"}, orders, ats)
	if _, err := o.With(reopened).Join(reopened, reopened.Col("id").EqCol(o.ID)).Where(keel.True()).Update(
		ctx,
		tx,
		o.Status.To(order.StatusCompleted),
		o.UpdatedAt.ToExpr(reopened.Col("at")),
	); err != nil {
		return fmt.Errorf("restore reversed orders: %w", err)
	}
	return nil
}

func (g *adjustmentPhase) adjSyncRebills(ctx context.Context, tx *keel.DB, rows *adjRows) error {
	if len(rows.rebills) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rows.rebills))
	statuses := make([]string, 0, len(rows.rebills))
	ats := make([]int64, 0, len(rows.rebills))
	for _, r := range rows.rebills {
		ids = append(ids, r.shipmentID.String())
		statuses = append(statuses, string(r.status))
		ats = append(ats, r.at)
	}
	sh := shipmentdb.Shipments
	d := keel.Unnest("d", []string{"id", "status", "at"}, ids, statuses, ats)
	if _, err := sh.With(d).Join(d, d.Col("id").EqCol(sh.ID)).Where(keel.True()).Update(
		ctx,
		tx,
		sh.BillingTransferStatus.ToExpr(d.Col("status")),
		sh.TransferredToBillingAt.ToExpr(d.Col("at")),
		sh.UpdatedAt.ToExpr(sh.UpdatedAt.Greatest(keel.Int64Of(d.Col("at")))),
	); err != nil {
		return fmt.Errorf("sync rebilled shipments: %w", err)
	}
	return nil
}

func (g *adjustmentPhase) adjCount(rows *adjRows) {
	statuses := make(map[invoiceadjustment.Status]int)
	for _, adj := range rows.adjustments {
		statuses[adj.Status]++
	}
	g.Report.Add("Invoice adjustments", len(rows.adjustments))
	g.Report.Add("Invoice adjustment lines", len(rows.lines))
	g.Report.Add("Invoice adjustment snapshots", len(rows.snapshots))
	g.Report.Add("Invoice correction groups", len(rows.groupOrder))
	g.Report.Add("Reconciliation exceptions", len(rows.exceptions))
	g.Report.Add("Adjustment batches", len(rows.batches))
	g.Report.Add("Adjustment batch items", len(rows.batchItems))
	g.Report.Add("Adjustment credit memos", rows.memos)
	g.Report.Add("Replacement invoices", rows.replacement)
	g.Report.Add("Billing queue items", len(rows.bil.Items))
	g.Report.Add("Credit memo applications", len(rows.bil.MemoApps))
	g.Report.Add("Customer ledger entries", len(rows.bil.Ledger))
	g.Report.Add("Journal entries", len(rows.bil.Entries))
	g.Report.Add("Journal entry lines", len(rows.bil.JournalLines))
	for _, status := range []invoiceadjustment.Status{
		invoiceadjustment.StatusDraft, invoiceadjustment.StatusPendingApproval,
		invoiceadjustment.StatusExecuted, invoiceadjustment.StatusRejected,
		invoiceadjustment.StatusExecutionFailed,
	} {
		g.Opts.Logf("  • adjustments %s: %d", status, statuses[status])
	}
}

type adjustmentPhase struct {
	*ledgervolume.Ledger
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&adjustmentPhase{Ledger: &ledgervolume.Ledger{Generator: g}}).seedAdjustments(ctx)
}
