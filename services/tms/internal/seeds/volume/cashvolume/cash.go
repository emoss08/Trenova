package cashvolume

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal/journalreversaldb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt/bankreceiptdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch/bankreceiptbatchdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem/bankreceiptworkitemdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection/collectiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement/customerstatementdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger/factoringledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge"
	"github.com/emoss08/trenova/internal/modules/billing/domain/latecharge/latechargedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder"
	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder/paymentreminderdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol/accountingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol/billingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/internal/seeds/volume/ledgervolume"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	cashMinute          = int64(60)
	cashHour            = int64(3_600)
	cashDay             = int64(86_400)
	cashResolveAfter    = 10 * cashDay
	cashPendingWindow   = 4 * cashDay
	cashJournalPending  = "Pending"
	cashJournalApproved = "Approved"
	cashJournalPosted   = "Posted"
	cashNoReference     = "Bank receipt has no reference number for automatic matching"
	cashNoUniqueMatch   = "No unique customer payment match found for bank receipt"
	cashMatchedNote     = "Resolved by matching bank receipt to customer payment"
	cashPayrollAccrual  = "accrued-payroll"
)

var cashBankSources = []string{
	"Chase BAI2 file",
	"Wells Fargo lockbox",
	"Bank of America CSV export",
}

type cashApplication struct {
	at    int64
	minor int64
}

type cashInvoice struct {
	id          pulid.ID
	number      string
	customer    *volumegen.CustomerRecord
	billType    string
	memoKind    string
	total       int64
	applied     int64
	invoiceDate int64
	due         int64
	disputed    bool
	factored    bool
	nonRecourse bool
	apps        []cashApplication
}

func (c *cashInvoice) openAt(t int64) int64 {
	if t < c.invoiceDate {
		return 0
	}
	open := c.total
	for _, a := range c.apps {
		if a.at <= t {
			open -= a.minor
		}
	}
	return max(open, 0)
}

func (c *cashInvoice) paidOffAt() int64 {
	open := c.total
	for _, a := range c.apps {
		open -= a.minor
		if open <= 0 {
			return a.at
		}
	}
	return 0
}

type cashInvoiceRow struct {
	ID                   pulid.ID
	Number               string
	CustomerID           pulid.ID
	BillType             string
	MemoKind             string
	TotalMinor           int64
	AppliedMinor         int64
	InvoiceDate          int64
	DueDate              *int64
	UpdatedAt            int64
	DisputeStatus        string
	FactoringCompanyID   *pulid.ID
	FactoringAgreementID *pulid.ID
}

type cashApplicationRow struct {
	InvoiceID    pulid.ID
	AppliedMinor int64
	AppliedAt    int64
}

type cashLedgerRow struct {
	CustomerID      pulid.ID
	TransactionDate int64
	AmountMinor     int64
}

type cashProfile struct {
	CustomerID           pulid.ID
	ApplyLateCharges     bool
	LateChargeRate       decimal.NullDecimal
	GracePeriodDays      int
	SendPaymentReminders bool
	SendStatements       bool
}

type cashReversal struct {
	ID         pulid.ID
	ReversedAt *int64
	Reason     string
}

type cashFactoring struct {
	chargedBack map[pulid.ID]struct{}
	nonRecourse map[pulid.ID]bool
	recourse    map[pulid.ID]factoring.Recourse
}

// cashSending is the organization's own setting for each feature that emails
// customers, kept so the seed can put them back once their history is written.
type cashSending struct {
	lateCharges     billingcontrol.LateChargeAssessmentMode
	reminders       bool
	statements      bool
	lateChanged     bool
	remindersChange bool
	statementChange bool
}

type cashState struct {
	sending    cashSending
	billing    *billingcontrol.BillingControl
	accounting *accountingcontrol.AccountingControl
	setup      *ledgervolume.Setup
	loc        *time.Location
	customers  map[pulid.ID]*volumegen.CustomerRecord
	profiles   map[pulid.ID]*cashProfile
	invoices   []*cashInvoice
	byCustomer map[pulid.ID][]*cashInvoice
	ledger     map[pulid.ID][]cashLedgerRow
	contacts   map[pulid.ID]string
}

func (g *cashPhase) seedCash(ctx context.Context) (err error) {
	if len(g.Customers) == 0 {
		return nil
	}

	st, err := g.cashLoad(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if restoreErr := g.cashRestoreSending(context.WithoutCancel(ctx), st); restoreErr != nil {
			err = errors.Join(err, restoreErr)
		}
	}()
	if err = g.cashEnableFeatures(ctx, st); err != nil {
		return err
	}

	steps := []struct {
		name string
		run  func(context.Context, *cashState) error
	}{
		{name: "bank reconciliation", run: g.cashBankReceipts},
		{name: "late charges", run: g.cashLateCharges},
		{name: "payment reminders", run: g.cashReminders},
		{name: "customer statements", run: g.cashStatements},
		{name: "collections", run: g.cashCollections},
		{name: "manual journals and reversals", run: g.cashManualJournals},
		{name: "journals to post", run: g.cashJournalsToPost},
	}
	for _, step := range steps {
		if err = step.run(ctx, st); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	return nil
}

// cashRestoreSending puts back the late-charge, reminder and statement settings
// the seed switched on to write their history. Left on, the worker's next run
// would act on every seeded invoice and email the seeded billing addresses.
func (g *cashPhase) cashRestoreSending(ctx context.Context, st *cashState) error {
	sending := st.sending
	if !sending.lateChanged && !sending.remindersChange && !sending.statementChange {
		return nil
	}
	bc := billingcontroldb.BillingControls
	var sets []keel.Assignment
	if sending.lateChanged {
		sets = append(sets, bc.LateChargeAssessmentMode.To(sending.lateCharges))
	}
	if sending.remindersChange {
		sets = append(sets, bc.PaymentRemindersEnabled.To(sending.reminders))
	}
	if sending.statementChange {
		sets = append(sets, bc.MonthlyStatementsEnabled.To(sending.statements))
	}
	if _, err := bc.Where(keel.True()).Update(ctx, g.DB, sets...); err != nil {
		return fmt.Errorf("restore late charge, reminder and statement settings: %w", err)
	}
	g.Opts.Logf(
		"  • restored the organization's late charge, reminder and statement settings; " +
			"their seeded history stays, and the worker will not act on seeded invoices",
	)
	return nil
}

func (g *cashPhase) cashLoad(ctx context.Context) (*cashState, error) {
	st := &cashState{
		loc:        time.UTC,
		customers:  make(map[pulid.ID]*volumegen.CustomerRecord, len(g.Customers)),
		profiles:   make(map[pulid.ID]*cashProfile, len(g.Customers)),
		byCustomer: make(map[pulid.ID][]*cashInvoice, len(g.Customers)),
		ledger:     make(map[pulid.ID][]cashLedgerRow, len(g.Customers)),
		contacts:   make(map[pulid.ID]string, len(g.Customers)),
	}
	ids := make([]pulid.ID, 0, len(g.Customers))
	for _, c := range g.Customers {
		st.customers[c.ID] = c
		st.contacts[c.ID] = g.Fake.FirstName() + " " + g.Fake.LastName()
		ids = append(ids, c.ID)
	}

	var err error
	if st.billing, err = volumegen.FirstOrNil(ctx, g.DB, billingcontroldb.BillingControls.Query()); err != nil {
		return nil, fmt.Errorf("load billing control: %w", err)
	}
	if st.accounting, err = g.AccountingControl(ctx); err != nil {
		return nil, err
	}

	if st.setup, err = g.LoadSetup(ctx); err != nil {
		return nil, err
	}

	o := tenantdb.Organizations
	org, err := o.Pick(o.Timezone).Where(o.ID.Eq(g.Refs.OrgID)).One(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load organization timezone: %w", err)
	}
	if loc, locErr := time.LoadLocation(org.Timezone); locErr == nil {
		st.loc = loc
	}

	cbp := customerdb.CustomerBillingProfiles
	profiles, err := keel.Into[cashProfile](ctx, g.DB, cbp.Where(cbp.CustomerID.In(ids...)).Pick(
		cbp.CustomerID,
		cbp.ApplyLateCharges,
		cbp.LateChargeRate,
		cbp.GracePeriodDays,
		cbp.SendPaymentReminders,
		cbp.SendStatements,
	))
	if err != nil {
		return nil, fmt.Errorf("load billing profiles: %w", err)
	}
	for i := range profiles {
		p := &profiles[i]
		st.profiles[p.CustomerID] = p
	}

	if err = g.cashLoadInvoices(ctx, st, ids); err != nil {
		return nil, err
	}

	cle := customerledgerdb.CustomerLedgerEntries
	ledger, err := keel.Into[cashLedgerRow](ctx, g.DB, cle.Where(cle.CustomerID.In(ids...)).Pick(
		cle.CustomerID,
		cle.TransactionDate,
		cle.AmountMinor,
	))
	if err != nil {
		return nil, fmt.Errorf("load customer ledger: %w", err)
	}
	for _, row := range ledger {
		st.ledger[row.CustomerID] = append(st.ledger[row.CustomerID], row)
	}

	return st, nil
}

func (g *cashPhase) cashLoadFactoring(ctx context.Context) (*cashFactoring, error) {
	out := &cashFactoring{
		chargedBack: make(map[pulid.ID]struct{}),
		nonRecourse: make(map[pulid.ID]bool),
		recourse:    make(map[pulid.ID]factoring.Recourse),
	}

	fsi := factoringsubmissiondb.SubmissionItems
	charged, err := fsi.Pick(fsi.InvoiceID).
		Where(fsi.Status.Eq(factoringsubmission.SubmissionItemStatusChargedBack)).
		All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load charged back factoring items: %w", err)
	}
	for _, item := range charged {
		out.chargedBack[item.InvoiceID] = struct{}{}
	}

	fc := factoringcompanydb.FactoringCompanies
	companies, err := fc.Pick(fc.ID, fc.NonRecourse).All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load factoring companies: %w", err)
	}
	for _, company := range companies {
		out.nonRecourse[company.ID] = company.NonRecourse
	}

	fa := factoringdb.Agreements
	agreements, err := fa.Pick(fa.ID, fa.Recourse).All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load factoring agreements: %w", err)
	}
	for _, agreement := range agreements {
		out.recourse[agreement.ID] = agreement.Recourse
	}

	return out, nil
}

func (f *cashFactoring) factored(row *cashInvoiceRow) bool {
	if row.FactoringCompanyID == nil || row.FactoringCompanyID.IsNil() {
		return false
	}
	_, charged := f.chargedBack[row.ID]
	return !charged
}

func (f *cashFactoring) carriesRisk(row *cashInvoiceRow) bool {
	if row.FactoringCompanyID == nil {
		return false
	}
	nonRecourse, ok := f.nonRecourse[*row.FactoringCompanyID]
	if !ok {
		return false
	}
	if row.FactoringAgreementID != nil {
		if recourse, found := f.recourse[*row.FactoringAgreementID]; found {
			return recourse == factoring.RecourseNone
		}
	}
	return nonRecourse
}

func (g *cashPhase) cashLoadInvoices(ctx context.Context, st *cashState, ids []pulid.ID) error {
	inv := invoicedb.Invoices
	rows, err := keel.Into[cashInvoiceRow](ctx, g.DB, inv.Where(
		inv.Status.Eq(invoice.StatusPosted),
		inv.CustomerID.In(ids...),
	).Pick(
		inv.ID,
		inv.Number,
		inv.CustomerID,
		keel.CastText(inv.BillType),
		keel.CastText(inv.MemoKind).Coalesce(""),
		inv.TotalAmountMinor,
		inv.AppliedAmountMinor,
		inv.InvoiceDate,
		inv.DueDate,
		inv.UpdatedAt,
		keel.CastText(inv.DisputeStatus).Coalesce(""),
		inv.FactoringCompanyID,
		inv.FactoringAgreementID,
	))
	if err != nil {
		return fmt.Errorf("load posted invoices: %w", err)
	}
	factors, err := g.cashLoadFactoring(ctx)
	if err != nil {
		return err
	}

	byID := make(map[pulid.ID]*cashInvoice, len(rows))
	for i := range rows {
		row := &rows[i]
		cus := st.customers[row.CustomerID]
		if cus == nil {
			continue
		}
		inv := &cashInvoice{
			id:          row.ID,
			number:      row.Number,
			customer:    cus,
			billType:    row.BillType,
			memoKind:    row.MemoKind,
			total:       row.TotalMinor,
			applied:     row.AppliedMinor,
			invoiceDate: row.InvoiceDate,
			disputed:    row.DisputeStatus == string(invoice.DisputeStatusDisputed),
			factored:    factors.factored(row),
			nonRecourse: factors.carriesRisk(row),
		}
		if row.DueDate != nil {
			inv.due = *row.DueDate
		}
		byID[inv.id] = inv
		st.invoices = append(st.invoices, inv)
	}

	cp := customerpaymentdb.Payments
	posted, err := cp.Pick(cp.ID, cp.PaymentDate).Where(
		cp.CustomerID.In(ids...),
		cp.Status.Eq(customerpayment.StatusPosted),
	).All(ctx, g.DB)
	if err != nil {
		return fmt.Errorf("load posted payments: %w", err)
	}
	paidAt := make(map[pulid.ID]int64, len(posted))
	paymentIDs := make([]pulid.ID, 0, len(posted))
	for _, p := range posted {
		paidAt[p.ID] = p.PaymentDate
		paymentIDs = append(paymentIDs, p.ID)
	}
	cpa := customerpaymentdb.Applications
	paymentApps, err := cpa.Pick(cpa.CustomerPaymentID, cpa.InvoiceID, cpa.AppliedAmountMinor).
		Where(cpa.CustomerPaymentID.In(paymentIDs...)).
		All(ctx, g.DB)
	if err != nil {
		return fmt.Errorf("load payment applications: %w", err)
	}
	apps := make([]cashApplicationRow, 0, len(paymentApps))
	for _, app := range paymentApps {
		apps = append(apps, cashApplicationRow{
			InvoiceID:    app.InvoiceID,
			AppliedMinor: app.AppliedAmountMinor,
			AppliedAt:    paidAt[app.CustomerPaymentID],
		})
	}

	invoiceIDs := make([]pulid.ID, 0, len(rows))
	for i := range rows {
		invoiceIDs = append(invoiceIDs, rows[i].ID)
	}
	cma := customerpaymentdb.CreditMemoApplications
	memoApps, err := keel.Into[cashApplicationRow](ctx, g.DB, cma.Where(
		cma.Status.Eq(customerpayment.CreditApplicationStatusApplied),
		cma.InvoiceID.In(invoiceIDs...),
	).Pick(cma.InvoiceID, cma.AppliedAmountMinor, cma.AccountingDate))
	if err != nil {
		return fmt.Errorf("load credit memo applications: %w", err)
	}
	fre := factoringledgerdb.ReserveEntries
	factorApps, err := keel.Into[cashApplicationRow](ctx, g.DB, fre.Where(
		fre.CustomerID.In(ids...),
		fre.ARMinor.Ne(0),
	).Pick(fre.InvoiceID, fre.ARMinor.Neg(), fre.AccountingDate))
	if err != nil {
		return fmt.Errorf("load factoring settlements: %w", err)
	}
	for _, app := range slices.Concat(apps, memoApps, factorApps) {
		if inv := byID[app.InvoiceID]; inv != nil {
			inv.apps = append(inv.apps, cashApplication{at: app.AppliedAt, minor: app.AppliedMinor})
		}
	}

	for i := range rows {
		inv := byID[rows[i].ID]
		if inv == nil {
			continue
		}
		recorded := int64(0)
		for _, a := range inv.apps {
			recorded += a.minor
		}
		if missing := inv.applied - recorded; missing > 0 {
			inv.apps = append(inv.apps, cashApplication{at: rows[i].UpdatedAt, minor: missing})
		}
		sort.SliceStable(inv.apps, func(a, b int) bool { return inv.apps[a].at < inv.apps[b].at })
	}

	sort.SliceStable(st.invoices, func(i, j int) bool {
		return st.invoices[i].invoiceDate < st.invoices[j].invoiceDate
	})
	for _, inv := range st.invoices {
		st.byCustomer[inv.customer.ID] = append(st.byCustomer[inv.customer.ID], inv)
	}

	return nil
}

// cashEnableFeatures switches on the organization features these pages show,
// when they are off: the app writes nothing for a disabled feature, so the
// pages would stay empty no matter how much receivable history exists. The
// features that email customers are switched back by cashRestoreSending.
func (g *cashPhase) cashEnableFeatures(ctx context.Context, st *cashState) error {
	if st.accounting != nil &&
		st.accounting.ReconciliationMode == accountingcontrol.ReconciliationModeDisabled {
		ac := accountingcontroldb.AccountingControls
		if _, err := ac.Where(keel.True()).
			Update(ctx, g.DB, ac.ReconciliationMode.To(accountingcontrol.ReconciliationModeWarnOnly)); err != nil {
			return fmt.Errorf("enable bank reconciliation: %w", err)
		}
		st.accounting.ReconciliationMode = accountingcontrol.ReconciliationModeWarnOnly
		g.Opts.Logf("  • enabled bank reconciliation (WarnOnly) so imported deposits are matched")
	}
	if st.billing == nil {
		return nil
	}

	b := st.billing
	bc := billingcontroldb.BillingControls
	var sets []keel.Assignment
	if b.LateChargeAssessmentMode == billingcontrol.LateChargeAssessmentModeDisabled ||
		b.LateChargeAssessmentMode == "" {
		st.sending.lateCharges, st.sending.lateChanged = b.LateChargeAssessmentMode, true
		if st.sending.lateCharges == "" {
			st.sending.lateCharges = billingcontrol.LateChargeAssessmentModeDisabled
		}
		b.LateChargeAssessmentMode = billingcontrol.LateChargeAssessmentModeAutomatic
		sets = append(sets, bc.LateChargeAssessmentMode.To(b.LateChargeAssessmentMode))
		g.Opts.Logf("  • set late charge assessment to Automatic")
	}
	if !b.PaymentRemindersEnabled || len(b.PaymentReminderDays) == 0 {
		st.sending.reminders, st.sending.remindersChange = b.PaymentRemindersEnabled, true
		b.PaymentRemindersEnabled = true
		if len(b.PaymentReminderDays) == 0 {
			b.PaymentReminderDays = slices.Clone(billingcontrol.DefaultPaymentReminderDays)
		}
		sets = append(
			sets,
			bc.PaymentRemindersEnabled.To(true),
			bc.PaymentReminderDays.To(b.PaymentReminderDays),
		)
		g.Opts.Logf("  • enabled payment reminders")
	}
	if !b.MonthlyStatementsEnabled || b.MonthlyStatementDay < 1 {
		st.sending.statements, st.sending.statementChange = b.MonthlyStatementsEnabled, true
		b.MonthlyStatementsEnabled = true
		if b.MonthlyStatementDay < 1 {
			b.MonthlyStatementDay = 1
		}
		sets = append(
			sets,
			bc.MonthlyStatementsEnabled.To(true),
			bc.MonthlyStatementDay.To(b.MonthlyStatementDay),
		)
		g.Opts.Logf("  • enabled monthly statements")
	}
	if len(sets) == 0 {
		return nil
	}
	if _, err := bc.Where(keel.True()).Update(ctx, g.DB, sets...); err != nil {
		return fmt.Errorf("enable receivables features: %w", err)
	}

	return nil
}

func (g *cashPhase) cashUser() pulid.ID {
	if len(g.Refs.UserIDs) == 0 {
		return g.Refs.AdminID
	}
	return volumegen.Pick(g.Fake, g.Refs.UserIDs)
}

func (g *cashPhase) cashName(st *cashState, id pulid.ID) string {
	if name, ok := st.setup.Names[id]; ok && name != "" {
		return name
	}
	return "a collector"
}

func cashLocalDay(at int64, loc *time.Location) time.Time {
	t := time.Unix(at, 0).In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func cashBusinessDay(day time.Time) time.Time {
	for day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
		day = day.AddDate(0, 0, 1)
	}
	return day
}

func (g *cashPhase) cashBetween(low, high int64) int64 {
	if high <= low {
		return low
	}
	return g.Fake.Between64(low, high)
}

// -- Bank reconciliation --

func (g *cashPhase) cashReversedPayments(ctx context.Context) (map[pulid.ID]*cashReversal, error) {
	out := make(map[pulid.ID]*cashReversal)
	if len(g.Payments) == 0 {
		return out, nil
	}
	ids := make([]pulid.ID, 0, len(g.Payments))
	for _, p := range g.Payments {
		ids = append(ids, p.ID)
	}
	cp := customerpaymentdb.Payments
	rows, err := keel.Into[cashReversal](ctx, g.DB, cp.Where(
		cp.Status.Eq(customerpayment.StatusReversed),
		cp.ID.In(ids...),
	).Pick(cp.ID, cp.ReversedAt, cp.ReversalReason.Coalesce("")))
	if err != nil {
		return nil, fmt.Errorf("load reversed payments: %w", err)
	}
	for i := range rows {
		r := &rows[i]
		out[r.ID] = r
	}
	return out, nil
}

func (g *cashPhase) cashDepositLag(method customerpayment.Method) int {
	switch method {
	case customerpayment.MethodWire:
		return 0
	case customerpayment.MethodACH:
		return g.Fake.Between(0, 1)
	case customerpayment.MethodCheck:
		return g.Fake.Between(1, 3)
	case customerpayment.MethodCard:
		return 2
	default:
		return 1
	}
}

func cashDepositMemo(method customerpayment.Method, name, reference string) string {
	name = strings.ToUpper(name)
	switch method {
	case customerpayment.MethodCheck:
		return "LOCKBOX DEPOSIT " + name + " " + reference
	case customerpayment.MethodWire:
		return "WIRE IN " + name + " REF " + reference
	case customerpayment.MethodCard:
		return "MERCHANT SETTLEMENT " + name
	default:
		return "ACH CREDIT " + name + " " + reference
	}
}

type cashBankRows struct {
	batches       []*bankreceiptbatch.BankReceiptBatch
	receipts      []*bankreceipt.BankReceipt
	items         []*bankreceiptworkitem.WorkItem
	batchStamps   []volumegen.Stamp
	receiptStamps []volumegen.Stamp
	itemStamps    []volumegen.Stamp
}

type cashOtherDeposit struct {
	reference string
	memo      string
	low, high float64
	dismiss   bool
	note      string
}

var cashOtherDeposits = []cashOtherDeposit{
	{
		reference: "INTEREST PAYMENT", memo: "INTEREST CREDIT", low: 1.5, high: 85, dismiss: true,
		note: "Monthly interest credit from the bank, not a customer payment",
	},
	{
		reference: "XFER FROM SAV 4471", memo: "ONLINE TRANSFER FROM SAVINGS", low: 5_000, high: 50_000,
		dismiss: true, note: "Internal transfer from the savings account",
	},
	{
		reference: "VENDOR REFUND", memo: "VENDOR REFUND CREDIT", low: 50, high: 900, dismiss: true,
		note: "Refund from a fuel vendor, booked in payables",
	},
	{
		reference: "DEP", memo: "DEPOSIT - NO REMITTANCE DETAIL", low: 400, high: 9_000,
		note: "Researched with the bank; customer asked to send remittance detail",
	},
}

//nolint:gocognit,funlen // one deposit's import and reconciliation story is told in one place
func (g *cashPhase) cashBankReceipts(ctx context.Context, st *cashState) error {
	if len(g.Payments) == 0 {
		return nil
	}
	reversed, err := g.cashReversedPayments(ctx)
	if err != nil {
		return err
	}

	payments := slices.Clone(g.Payments)
	sort.SliceStable(payments, func(i, j int) bool {
		return payments[i].PaymentDate < payments[j].PaymentDate
	})

	nowUnix := volumegen.Unix(g.Now)
	importer := g.Refs.AdminID
	matchesLater := st.accounting != nil &&
		st.accounting.ReconciliationMode != accountingcontrol.ReconciliationModeDisabled
	rows := &cashBankRows{}
	batches := make(map[string]*bankreceiptbatch.BankReceiptBatch)

	batchFor := func(day time.Time) *bankreceiptbatch.BankReceiptBatch {
		key := day.Format("2006-01-02")
		if b, ok := batches[key]; ok {
			return b
		}
		importAt := day.Add(30*time.Hour).Unix() + g.cashBetween(0, 50*cashMinute)
		if importAt > nowUnix-cashHour {
			return nil
		}
		b := &bankreceiptbatch.BankReceiptBatch{
			ID:             volumegen.IDAt("brib_", importAt),
			OrganizationID: g.Refs.OrgID,
			BusinessUnitID: g.Refs.BuID,
			Source:         volumegen.Pick(g.Fake, cashBankSources),
			Reference:      "DEP-" + day.Format("20060102"),
			Status:         bankreceiptbatch.StatusCompleted,
			CreatedByID:    importer,
			UpdatedByID:    importer,
			Version:        1,
			CreatedAt:      importAt,
			UpdatedAt:      importAt,
		}
		batches[key] = b
		rows.batches = append(rows.batches, b)
		rows.batchStamps = append(
			rows.batchStamps,
			volumegen.Stamp{Id: b.ID, Created: importAt, Updated: importAt},
		)
		return b
	}

	addReceipt := func(
		b *bankreceiptbatch.BankReceiptBatch,
		r *bankreceipt.BankReceipt,
		item *bankreceiptworkitem.WorkItem,
		importStatus bankreceipt.Status,
	) {
		b.ImportedCount++
		b.ImportedAmountMinor += r.AmountMinor
		switch importStatus {
		case bankreceipt.StatusMatched:
			b.MatchedCount++
			b.MatchedAmountMinor += r.AmountMinor
		case bankreceipt.StatusException:
			b.ExceptionCount++
			b.ExceptionAmountMinor += r.AmountMinor
		case bankreceipt.StatusImported:
		}
		rows.receipts = append(rows.receipts, r)
		rows.receiptStamps = append(
			rows.receiptStamps,
			volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt},
		)
		if item != nil {
			rows.items = append(rows.items, item)
			rows.itemStamps = append(
				rows.itemStamps,
				volumegen.Stamp{Id: item.ID, Created: item.CreatedAt, Updated: item.UpdatedAt},
			)
		}
	}

	workItem := func(r *bankreceipt.BankReceipt, openedAt int64) *bankreceiptworkitem.WorkItem {
		return &bankreceiptworkitem.WorkItem{
			ID:             volumegen.IDAt("brwi_", openedAt),
			OrganizationID: g.Refs.OrgID,
			BusinessUnitID: g.Refs.BuID,
			BankReceiptID:  r.ID,
			Status:         bankreceiptworkitem.StatusOpen,
			CreatedByID:    importer,
			UpdatedByID:    importer,
			Version:        1,
			CreatedAt:      openedAt,
			UpdatedAt:      openedAt,
		}
	}

	// progress moves an open item along as a person works it: assigned, then
	// in review, then closed by the given step when old enough to be done.
	progress := func(item *bankreceiptworkitem.WorkItem, finish func(by pulid.ID, at int64)) {
		openedAt := item.CreatedAt
		if nowUnix-openedAt > cashResolveAfter && finish != nil {
			worker := g.cashUser()
			assignedAt := openedAt + g.cashBetween(cashHour, 2*cashDay)
			resolvedAt := min(assignedAt+g.cashBetween(cashHour, 5*cashDay), nowUnix-cashMinute)
			item.AssignedToUserID = worker
			item.AssignedAt = volumegen.Ptr(assignedAt)
			item.ResolvedByUserID = worker
			item.ResolvedAt = volumegen.Ptr(resolvedAt)
			item.UpdatedByID = worker
			item.UpdatedAt = resolvedAt
			finish(worker, resolvedAt)
			return
		}
		switch roll := g.Fake.Intn(10); {
		case roll < 5:
		case roll < 8:
			worker := g.cashUser()
			assignedAt := min(openedAt+g.cashBetween(cashHour, 2*cashDay), nowUnix-cashMinute)
			item.Status = bankreceiptworkitem.StatusAssigned
			item.AssignedToUserID = worker
			item.AssignedAt = volumegen.Ptr(assignedAt)
			item.UpdatedByID = worker
			item.UpdatedAt = assignedAt
		default:
			worker := g.cashUser()
			assignedAt := min(openedAt+g.cashBetween(cashHour, cashDay), nowUnix-2*cashMinute)
			reviewAt := min(assignedAt+g.cashBetween(10*cashMinute, cashDay), nowUnix-cashMinute)
			item.Status = bankreceiptworkitem.StatusInReview
			item.AssignedToUserID = worker
			item.AssignedAt = volumegen.Ptr(assignedAt)
			item.UpdatedByID = worker
			item.UpdatedAt = reviewAt
		}
	}

	for _, p := range payments {
		cus := st.customers[p.CustomerID]
		if cus == nil || p.AmountMinor <= 0 {
			continue
		}
		day := cashBusinessDay(
			cashLocalDay(p.PaymentDate, st.loc).AddDate(0, 0, g.cashDepositLag(p.PaymentMethod)),
		)
		b := batchFor(day)
		if b == nil {
			continue
		}
		importAt := b.CreatedAt
		r := &bankreceipt.BankReceipt{
			ID:              volumegen.IDAt("brcpt_", importAt),
			OrganizationID:  g.Refs.OrgID,
			BusinessUnitID:  g.Refs.BuID,
			ReceiptDate:     day.Add(12 * time.Hour).Unix(),
			AmountMinor:     p.AmountMinor,
			ReferenceNumber: p.ReferenceNumber,
			Memo:            cashDepositMemo(p.PaymentMethod, cus.Name, p.ReferenceNumber),
			Status:          bankreceipt.StatusImported,
			ImportBatchID:   b.ID,
			CreatedByID:     importer,
			UpdatedByID:     importer,
			Version:         1,
			CreatedAt:       importAt,
			UpdatedAt:       importAt,
		}
		if !matchesLater {
			addReceipt(b, r, nil, bankreceipt.StatusImported)
			continue
		}

		rev := reversed[p.ID]
		roll := g.Fake.Float()
		if roll < 0.86 && strings.TrimSpace(p.ReferenceNumber) != "" {
			r.Status = bankreceipt.StatusMatched
			r.MatchedCustomerPaymentID = p.ID
			r.MatchedAt = volumegen.Ptr(importAt)
			r.MatchedByID = importer
			var item *bankreceiptworkitem.WorkItem
			if rev != nil && rev.ReversedAt != nil && *rev.ReversedAt > importAt {
				reason := "The customer payment " + p.ReferenceNumber + " matched to this deposit was reversed"
				if rev.Reason != "" {
					reason += ": " + rev.Reason
				}
				r.Status = bankreceipt.StatusException
				r.ExceptionReason = reason
				r.MatchedCustomerPaymentID = pulid.Nil
				r.MatchedAt = nil
				r.MatchedByID = pulid.Nil
				r.UpdatedAt = *rev.ReversedAt
				item = workItem(r, *rev.ReversedAt)
				progress(item, nil)
			}
			addReceipt(b, r, item, bankreceipt.StatusMatched)
			continue
		}

		ref := strings.TrimSpace(p.ReferenceNumber)
		if cut := max(len(ref)/2, 3); roll >= 0.93 && cut < len(ref) {
			r.ReferenceNumber = ref[:cut]
			r.ExceptionReason = cashNoUniqueMatch
		} else {
			r.ReferenceNumber = ""
			r.Memo = cashDepositMemo(p.PaymentMethod, cus.Name, "")
			r.ExceptionReason = cashNoReference
		}
		r.Status = bankreceipt.StatusException
		item := workItem(r, importAt)
		var closer func(by pulid.ID, at int64)
		if rev == nil {
			closer = func(by pulid.ID, at int64) {
				item.Status = bankreceiptworkitem.StatusResolved
				item.ResolutionType = bankreceiptworkitem.ResolutionMatchedToPayment
				item.ResolutionNote = cashMatchedNote
				r.Status = bankreceipt.StatusMatched
				r.MatchedCustomerPaymentID = p.ID
				r.MatchedAt = volumegen.Ptr(at)
				r.MatchedByID = by
				r.UpdatedByID = by
				r.UpdatedAt = at
			}
		}
		progress(item, closer)
		addReceipt(b, r, item, bankreceipt.StatusException)
	}

	if matchesLater {
		for _, b := range slices.Clone(rows.batches) {
			if !g.Fake.Chance(0.18) || b.CreatedAt > nowUnix-cashHour {
				continue
			}
			kind := cashOtherDeposits[g.Fake.Intn(len(cashOtherDeposits))]
			reference := kind.reference
			if reference == "DEP" {
				reference = "DEP" + g.Fake.Digits(8)
			}
			r := &bankreceipt.BankReceipt{
				ID:              volumegen.IDAt("brcpt_", b.CreatedAt),
				OrganizationID:  g.Refs.OrgID,
				BusinessUnitID:  g.Refs.BuID,
				ReceiptDate:     b.CreatedAt - 18*cashHour,
				AmountMinor:     money.MinorUnits(g.Fake.Money(kind.low, kind.high)),
				ReferenceNumber: reference,
				Memo:            kind.memo,
				Status:          bankreceipt.StatusException,
				ExceptionReason: cashNoUniqueMatch,
				ImportBatchID:   b.ID,
				CreatedByID:     importer,
				UpdatedByID:     importer,
				Version:         1,
				CreatedAt:       b.CreatedAt,
				UpdatedAt:       b.CreatedAt,
			}
			if r.AmountMinor <= 0 {
				continue
			}
			item := workItem(r, b.CreatedAt)
			progress(item, func(by pulid.ID, at int64) {
				item.ResolutionNote = kind.note
				if kind.dismiss {
					item.Status = bankreceiptworkitem.StatusDismissed
					item.ResolutionType = bankreceiptworkitem.ResolutionMarkedFalsePositive
					return
				}
				item.Status = bankreceiptworkitem.StatusResolved
				item.ResolutionType = bankreceiptworkitem.ResolutionRequiresExternalFollowUp
			})
			addReceipt(b, r, item, bankreceipt.StatusException)
		}
	}

	if len(rows.receipts) == 0 {
		return nil
	}
	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if txErr := volumegen.InsertRows(ctx, tx, bankreceiptbatchdb.BankReceiptBatches.Handle, rows.batches); txErr != nil {
			return fmt.Errorf("insert bank receipt batches: %w", txErr)
		}
		if txErr := volumegen.InsertRows(ctx, tx, bankreceiptdb.BankReceipts.Handle, rows.receipts); txErr != nil {
			return fmt.Errorf("insert bank receipts: %w", txErr)
		}
		if txErr := volumegen.InsertRows(ctx, tx, bankreceiptworkitemdb.WorkItems.Handle, rows.items); txErr != nil {
			return fmt.Errorf("insert bank receipt work items: %w", txErr)
		}
		if txErr := volumegen.Backdate(ctx, tx, bankreceiptbatchdb.BankReceiptBatches.Handle, rows.batchStamps); txErr != nil {
			return txErr
		}
		if txErr := volumegen.Backdate(ctx, tx, bankreceiptdb.BankReceipts.Handle, rows.receiptStamps); txErr != nil {
			return txErr
		}
		return volumegen.Backdate(ctx, tx, bankreceiptworkitemdb.WorkItems.Handle, rows.itemStamps)
	}); err != nil {
		return err
	}

	g.Report.Add("Bank import batches", len(rows.batches))
	g.Report.Add("Bank receipts", len(rows.receipts))
	g.Report.Add("Reconciliation work items", len(rows.items))
	return nil
}

// -- Late charges --

type cashLateLine struct {
	inv    *cashInvoice
	index  int
	start  int64
	end    int64
	runAt  int64
	basis  int64
	rate   decimal.Decimal
	charge int64
}

type cashLateMemo struct {
	customer *volumegen.CustomerRecord
	asOf     int64
	lines    []*cashLateLine
}

//nolint:gocognit,funlen // the nightly late charge run is replayed over the history in one pass
func (g *cashPhase) cashLateCharges(ctx context.Context, st *cashState) error {
	if st.billing == nil ||
		st.billing.LateChargeAssessmentMode != billingcontrol.LateChargeAssessmentModeAutomatic {
		return nil
	}
	nowUnix := volumegen.Unix(g.Now)
	minimum := money.MinorUnits(st.billing.LateChargeMinimumAmount)

	var lines []*cashLateLine
	for _, inv := range st.invoices {
		profile := st.profiles[inv.customer.ID]
		if profile == nil || !profile.ApplyLateCharges || !profile.LateChargeRate.Valid ||
			profile.LateChargeRate.Decimal.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if inv.due == 0 || inv.disputed || inv.memoKind == string(invoice.MemoKindLateCharge) ||
			(inv.billType != string(billingqueue.BillTypeInvoice) &&
				inv.billType != string(billingqueue.BillTypeDebitMemo)) {
			continue
		}
		overdueStart := latecharge.OverdueStart(inv.due, profile.GracePeriodDays)
		for idx := 1; ; idx++ {
			start, end := latecharge.PeriodBounds(overdueStart, idx)
			runAt := cashLocalDay(start, st.loc).AddDate(0, 0, 1).Add(2 * time.Hour).Unix()
			if runAt > nowUnix-cashHour {
				break
			}
			open := inv.openAt(runAt)
			charge := latecharge.ChargeMinor(open, profile.LateChargeRate.Decimal)
			if charge <= 0 {
				break
			}
			lines = append(lines, &cashLateLine{
				inv: inv, index: idx, start: start, end: end, runAt: runAt,
				basis: open, rate: profile.LateChargeRate.Decimal, charge: charge,
			})
		}
	}
	if len(lines) == 0 {
		return nil
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].runAt < lines[j].runAt })

	carry := make(map[pulid.ID][]*cashLateLine)
	var memos []*cashLateMemo
	for begin := 0; begin < len(lines); {
		runAt := lines[begin].runAt
		end := begin
		byCustomer := make(map[pulid.ID][]*cashLateLine)
		var order []pulid.ID
		for end < len(lines) && lines[end].runAt == runAt {
			id := lines[end].inv.customer.ID
			if _, ok := byCustomer[id]; !ok {
				order = append(order, id)
			}
			byCustomer[id] = append(byCustomer[id], lines[end])
			end++
		}
		for _, id := range order {
			pending := append(carry[id], byCustomer[id]...)
			total := int64(0)
			for _, l := range pending {
				total += l.charge
			}
			if total < minimum {
				carry[id] = pending
				continue
			}
			delete(carry, id)
			memos = append(memos, &cashLateMemo{
				customer: pending[0].inv.customer,
				asOf:     runAt,
				lines:    pending,
			})
		}
		begin = end
	}
	if len(memos) == 0 {
		return nil
	}

	slots := make([]volumegen.NumberSlot, len(memos))
	for i, m := range memos {
		slots[i] = volumegen.NumberSlot{At: m.asOf}
	}
	numbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeDebitMemo, slots)
	if err != nil {
		return err
	}

	autoPost := st.billing.InvoicePostingMode == billingcontrol.InvoicePostingModeAutomaticWhenNoBlockingExceptions
	rows := &ledgervolume.Rows{Balances: make(map[ledgervolume.BalanceKey]*ledgervolume.Balance)}
	var assessments []*latecharge.LateChargeAssessment
	var assessmentStamps []volumegen.Stamp
	posted, drafts := 0, 0
	for i, m := range memos {
		runKey := volumegen.IDAt("lcr_", m.asOf).String()
		actor := g.Refs.AdminID
		term := m.customer.PaymentTerm
		if term == "" {
			term = invoice.PaymentTermNet30
		}
		item := &billingqueue.BillingQueueItem{
			ID:                    volumegen.IDAt("bqi_", m.asOf),
			OrganizationID:        g.Refs.OrgID,
			BusinessUnitID:        g.Refs.BuID,
			BillToCustomerID:      m.customer.ID,
			Number:                numbers[i],
			Status:                billingqueue.StatusApproved,
			BillType:              billingqueue.BillTypeDebitMemo,
			RerateVariancePercent: decimal.Zero,
			AdjustmentContext: map[string]any{
				"memo":       true,
				"memoKind":   string(invoice.MemoKindLateCharge),
				"raisedById": actor.String(),
			},
			CreatedAt: m.asOf,
			UpdatedAt: m.asOf,
		}
		item.SyncAllocatedMinor()

		memo := &invoice.Invoice{
			ID:                 volumegen.IDAt("inv_", m.asOf),
			OrganizationID:     g.Refs.OrgID,
			BusinessUnitID:     g.Refs.BuID,
			BillingQueueItemID: item.ID,
			CustomerID:         m.customer.ID,
			Scope:              invoice.ScopeMemo,
			Detail:             customer.InvoiceDetailDetailed,
			SectionBy:          customer.InvoiceSectionKeyShipment,
			Number:             numbers[i],
			BillType:           billingqueue.BillTypeDebitMemo,
			Status:             invoice.StatusDraft,
			PaymentTerm:        term,
			CurrencyCode:       money.DefaultCurrencyCode,
			InvoiceDate:        m.asOf,
			DueDate:            invoice.DueDateFromPaymentTerm(m.asOf, term),
			SubtotalAmount:     decimal.Zero,
			OtherAmount:        decimal.Zero,
			AppliedAmount:      decimal.Zero,
			SettlementStatus:   invoice.SettlementStatusUnpaid,
			DisputeStatus:      invoice.DisputeStatusNone,
			SendStatus:         invoice.SendStatusNotSent,
			EDISendStatus:      invoice.EDISendStatusNotSent,
			FactoringOverride:  invoice.FactoringOverrideDefault,
			MemoReason: "Late charges assessed as of " + time.Unix(m.asOf, 0).
				UTC().
				Format("2006-01-02"),
			MemoKind:  invoice.MemoKindLateCharge,
			Memo:      "Late charge run " + runKey,
			CreatedAt: m.asOf,
			UpdatedAt: m.asOf,
		}
		ledgervolume.BillTo(memo, m.customer)

		total := decimal.Zero
		for idx, l := range m.lines {
			amount := money.DecimalFromMinor(l.charge)
			line := &invoice.InvoiceLine{
				ID:             volumegen.IDAt("invl_", m.asOf),
				OrganizationID: g.Refs.OrgID,
				BusinessUnitID: g.Refs.BuID,
				InvoiceID:      memo.ID,
				LineNumber:     idx + 1,
				Type:           invoice.InvoiceLineTypeMemo,
				Description: fmt.Sprintf(
					"Late charge on %s, period %d (%s to %s), %s%% of %s",
					l.inv.number,
					l.index,
					time.Unix(l.start, 0).UTC().Format("2006-01-02"),
					time.Unix(l.end, 0).UTC().Format("2006-01-02"),
					l.rate.String(),
					money.DecimalFromMinor(l.basis).StringFixed(2),
				),
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: amount,
				Amount:    amount,
				CreatedAt: m.asOf,
				UpdatedAt: m.asOf,
			}
			total = total.Add(amount)
			memo.Lines = append(memo.Lines, line)
			rows.Lines = append(rows.Lines, line)

			a := &latecharge.LateChargeAssessment{
				ID:                    volumegen.IDAt("lca_", m.asOf),
				OrganizationID:        g.Refs.OrgID,
				BusinessUnitID:        g.Refs.BuID,
				CustomerID:            m.customer.ID,
				SourceInvoiceID:       l.inv.id,
				PeriodIndex:           l.index,
				PeriodStart:           l.start,
				PeriodEnd:             l.end,
				AsOfDate:              m.asOf,
				BasisOpenBalanceMinor: l.basis,
				RatePercent:           l.rate,
				ChargeMinor:           l.charge,
				DebitMemoInvoiceID:    memo.ID,
				DebitMemoLineID:       line.ID,
				RunKey:                runKey,
				CreatedByID:           actor,
				CreatedAt:             m.asOf,
				UpdatedAt:             m.asOf,
			}
			assessments = append(assessments, a)
			assessmentStamps = append(
				assessmentStamps,
				volumegen.Stamp{Id: a.ID, Created: m.asOf, Updated: m.asOf},
			)
		}
		memo.TotalAmount = total
		memo.SyncMinorAmounts()

		postedAt := int64(0)
		switch {
		case autoPost:
			postedAt = m.asOf
		case nowUnix-m.asOf > 5*cashDay && g.Fake.Chance(0.9):
			postedAt = min(m.asOf+g.cashBetween(4*cashHour, 3*cashDay), nowUnix-cashHour)
		}
		if postedAt > 0 {
			memo.Status = invoice.StatusPosted
			memo.PostedAt = volumegen.Ptr(postedAt)
			memo.UpdatedAt = postedAt
			item.Status = billingqueue.StatusPosted
			item.UpdatedAt = postedAt
			rows.Links = append(
				rows.Links,
				ledgervolume.ItemInvoice{Item: item.ID, Invoice: memo.ID},
			)
			if st.setup.PostInvoices {
				if err = g.PostInvoice(st.setup, rows, memo, pulid.Nil, item); err != nil {
					return err
				}
				j := rows.Journals[len(rows.Journals)-1]
				j.Event = accountingcontrol.JournalSourceEventDebitMemoPosted
				j.ReferenceType = j.Event.String()
				j.Description = "Debit memo posted for " + memo.Number
				entry := rows.Ledger[len(rows.Ledger)-1]
				entry.SourceEventType = j.Event.String()
				st.ledger[m.customer.ID] = append(st.ledger[m.customer.ID], cashLedgerRow{
					CustomerID:      m.customer.ID,
					TransactionDate: entry.TransactionDate,
					AmountMinor:     entry.AmountMinor,
				})
			}
			lateInv := &cashInvoice{
				id:          memo.ID,
				number:      memo.Number,
				customer:    m.customer,
				billType:    string(billingqueue.BillTypeDebitMemo),
				memoKind:    string(invoice.MemoKindLateCharge),
				total:       memo.TotalAmountMinor,
				invoiceDate: postedAt,
			}
			if memo.DueDate != nil {
				lateInv.due = *memo.DueDate
			}
			st.invoices = append(st.invoices, lateInv)
			st.byCustomer[m.customer.ID] = append(st.byCustomer[m.customer.ID], lateInv)
			posted++
		} else {
			drafts++
		}
		rows.Items = append(rows.Items, item)
		rows.Invoices = append(rows.Invoices, memo)
	}

	if err = g.Materialize(ctx, rows); err != nil {
		return err
	}
	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if txErr := g.Write(ctx, tx, rows); txErr != nil {
			return txErr
		}
		if txErr := volumegen.InsertRows(ctx, tx, latechargedb.LateChargeAssessments.Handle, assessments); txErr != nil {
			return fmt.Errorf("insert late charge assessments: %w", txErr)
		}
		return volumegen.Backdate(ctx, tx, latechargedb.LateChargeAssessments.Handle, assessmentStamps)
	}); err != nil {
		return err
	}

	g.Report.Add("Late charge assessments", len(assessments))
	g.Report.Add("Late charge debit memos", posted)
	g.Report.Add("Late charge memos awaiting post", drafts)
	g.Report.Add("Journal entries", len(rows.Entries))
	g.Report.Add("Customer ledger entries", len(rows.Ledger))
	return nil
}

// -- Payment reminders --

func (g *cashPhase) cashReminders(ctx context.Context, st *cashState) error {
	if st.billing == nil || !st.billing.PaymentRemindersEnabled {
		return nil
	}
	schedule := st.billing.PaymentReminderDays
	if len(schedule) == 0 {
		schedule = billingcontrol.DefaultPaymentReminderDays
	}
	nowUnix := volumegen.Unix(g.Now)

	var rows []*paymentreminder.Reminder
	for _, c := range g.Customers {
		profile := st.profiles[c.ID]
		if profile == nil || !profile.SendPaymentReminders || c.BillingEmail == "" {
			continue
		}
		for _, inv := range st.byCustomer[c.ID] {
			if inv.due == 0 || inv.disputed || inv.factored {
				continue
			}
			for i, days := range schedule {
				sentAt := cashLocalDay(
					inv.due+int64(days)*cashDay,
					st.loc,
				).Add(8 * time.Hour).
					Unix()
				if sentAt > nowUnix {
					break
				}
				open := inv.openAt(sentAt)
				if open <= 0 {
					break
				}
				rows = append(rows, &paymentreminder.Reminder{
					ID:             volumegen.IDAt("ipr_", sentAt),
					OrganizationID: g.Refs.OrgID,
					BusinessUnitID: g.Refs.BuID,
					InvoiceID:      inv.id,
					CustomerID:     c.ID,
					Step: int16(
						i + 1,
					), //nolint:gosec // bounded by MaxPaymentReminderSteps
					DaysPastDue:     paymentreminder.DaysPastDue(inv.due, sentAt),
					OpenAmountMinor: open,
					CurrencyCode:    money.DefaultCurrencyCode,
					SentTo:          c.BillingEmail,
					SentAt:          sentAt,
				})
			}
			if open := inv.openAt(nowUnix); open > 0 && inv.due < nowUnix-5*cashDay &&
				g.Fake.Chance(0.1) {
				sentAt := g.cashBetween(inv.due+5*cashDay, nowUnix-cashHour)
				rows = append(rows, &paymentreminder.Reminder{
					ID:              volumegen.IDAt("ipr_", sentAt),
					OrganizationID:  g.Refs.OrgID,
					BusinessUnitID:  g.Refs.BuID,
					InvoiceID:       inv.id,
					CustomerID:      c.ID,
					Step:            paymentreminder.ManualStep,
					DaysPastDue:     paymentreminder.DaysPastDue(inv.due, sentAt),
					OpenAmountMinor: inv.openAt(sentAt),
					CurrencyCode:    money.DefaultCurrencyCode,
					SentTo:          c.BillingEmail,
					Manual:          true,
					SentByID:        g.cashUser(),
					SentAt:          sentAt,
				})
			}
		}
	}
	rows = slices.DeleteFunc(
		rows,
		func(r *paymentreminder.Reminder) bool { return r.OpenAmountMinor <= 0 },
	)
	if len(rows) == 0 {
		return nil
	}
	if err := g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if txErr := volumegen.InsertRows(ctx, tx, paymentreminderdb.Reminders.Handle, rows); txErr != nil {
			return fmt.Errorf("insert payment reminders: %w", txErr)
		}
		return nil
	}); err != nil {
		return err
	}

	g.Report.Add("Payment reminders", len(rows))
	return nil
}

// -- Customer statements --

func (g *cashPhase) cashStatements(ctx context.Context, st *cashState) error {
	if st.billing == nil || !st.billing.MonthlyStatementsEnabled {
		return nil
	}
	nowUnix := volumegen.Unix(g.Now)
	day := int(st.billing.MonthlyStatementDay)
	if day < 1 || day > customerstatement.MaxMonthlyDay {
		day = 1
	}

	first := g.Start.In(st.loc)
	month := time.Date(first.Year(), first.Month(), 1, 0, 0, 0, 0, st.loc).AddDate(0, 1, 0)
	var rows []*customerstatement.Statement
	for ; ; month = month.AddDate(0, 1, 0) {
		sentAt := time.Date(month.Year(), month.Month(), day, 8, 0, 0, 0, st.loc).Unix() +
			g.cashBetween(0, 40*cashMinute)
		if sentAt > nowUnix {
			break
		}
		period, _ := customerstatement.MonthlyPeriod(sentAt, st.loc, day)
		for _, c := range g.Customers {
			profile := st.profiles[c.ID]
			if profile == nil || !profile.SendStatements || c.BillingEmail == "" {
				continue
			}
			ending, active := int64(0), false
			for _, row := range st.ledger[c.ID] {
				if row.TransactionDate <= period.End {
					ending += row.AmountMinor
				}
				if row.TransactionDate >= period.Start && row.TransactionDate <= period.End {
					active = true
				}
			}
			if ending == 0 && !active {
				continue
			}
			pastDue := int64(0)
			for _, inv := range st.byCustomer[c.ID] {
				if inv.due > 0 && inv.due < period.End && inv.invoiceDate <= period.End {
					pastDue += inv.openAt(period.End)
				}
			}
			rows = append(rows, &customerstatement.Statement{
				ID:                 volumegen.IDAt("cst_", sentAt),
				OrganizationID:     g.Refs.OrgID,
				BusinessUnitID:     g.Refs.BuID,
				CustomerID:         c.ID,
				PeriodStart:        period.Start,
				PeriodEnd:          period.End,
				EndingBalanceMinor: ending,
				PastDueMinor:       pastDue,
				CurrencyCode:       money.DefaultCurrencyCode,
				SentTo:             c.BillingEmail,
				Monthly:            true,
				SentAt:             sentAt,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	if err := g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if txErr := volumegen.InsertRows(ctx, tx, customerstatementdb.Statements.Handle, rows); txErr != nil {
			return fmt.Errorf("insert customer statements: %w", txErr)
		}
		return nil
	}); err != nil {
		return err
	}

	g.Report.Add("Customer statements", len(rows))
	return nil
}

// -- Collections --

var cashCallNotes = []string{
	"Called %s in AP about %s; check run is Thursday.",
	"Spoke with %s; %s is approved and waiting on the next payment run.",
	"Left a voicemail for %s about %s.",
	"%s says %s was short-paid by their system; resending the backup.",
}

var cashEmailNotes = []string{
	"Emailed %s the copy of %s with the signed POD.",
	"Sent %s a past-due summary including %s.",
	"%s asked for a W-9 before releasing %s; sent it.",
}

//nolint:gocognit,funlen // one customer's collection history is written in one place
func (g *cashPhase) cashCollections(ctx context.Context, st *cashState) error {
	nowUnix := volumegen.Unix(g.Now)
	today := cashLocalDay(nowUnix, st.loc)

	var accounts []*collection.Account
	var accountStamps []volumegen.Stamp
	var activities []*collection.Activity

	for _, c := range g.Customers {
		invs := slices.DeleteFunc(slices.Clone(st.byCustomer[c.ID]), func(inv *cashInvoice) bool {
			return inv.factored && inv.nonRecourse
		})
		if len(invs) == 0 {
			continue
		}
		contact := st.contacts[c.ID]
		var acts []*collection.Activity
		collector := pulid.Nil
		var lastPromise *collection.Activity

		add := func(kind collection.ActivityKind, at int64, by pulid.ID, summary string, invID pulid.ID) *collection.Activity {
			a := &collection.Activity{
				ID:             volumegen.IDAt("cact_", at),
				OrganizationID: g.Refs.OrgID,
				BusinessUnitID: g.Refs.BuID,
				CustomerID:     c.ID,
				Kind:           kind,
				Summary:        summary,
				InvoiceID:      invID,
				CurrencyCode:   money.DefaultCurrencyCode,
				CreatedByID:    by,
				CreatedAt:      at,
			}
			if kind.IsContact() {
				a.ContactName = contact
			}
			acts = append(acts, a)
			return a
		}
		assign := func(at int64) {
			if collector.IsNotNil() {
				return
			}
			collector = g.cashUser()
			a := add(collection.ActivityCollectorAssigned, at, g.Refs.AdminID,
				"Assigned to "+g.cashName(st, collector), pulid.Nil)
			a.CollectorID = collector
		}
		contactNote := func(inv *cashInvoice, at int64) {
			if g.Fake.Chance(0.6) {
				add(collection.ActivityCall, at, collector,
					fmt.Sprintf(volumegen.Pick(g.Fake, cashCallNotes), contact, inv.number), inv.id)
				return
			}
			add(collection.ActivityEmail, at, collector,
				fmt.Sprintf(volumegen.Pick(g.Fake, cashEmailNotes), contact, inv.number), inv.id)
		}
		promise := func(inv *cashInvoice, at, amount, promisedOn int64) {
			a := add(collection.ActivityPromiseToPay, at, collector,
				"Promised to pay "+inv.number, inv.id)
			a.PromisedAmountMinor = volumegen.Ptr(amount)
			a.PromisedOn = volumegen.Ptr(promisedOn)
			lastPromise = a
		}

		for _, inv := range invs {
			if inv.due == 0 {
				continue
			}
			paidAt := inv.paidOffAt()
			if paidAt == 0 || paidAt <= inv.due+15*cashDay || !g.Fake.Chance(0.5) {
				continue
			}
			callAt := inv.due + g.cashBetween(7*cashDay, min(paidAt-inv.due-cashDay, 25*cashDay))
			assign(callAt - g.cashBetween(cashHour, 2*cashDay))
			contactNote(inv, callAt)
			if g.Fake.Chance(0.6) {
				promisedAt := max(callAt+cashHour, paidAt-g.cashBetween(cashDay, 5*cashDay))
				if promisedAt < paidAt {
					promisedOn := cashLocalDay(
						paidAt,
						st.loc,
					).AddDate(0, 0, g.Fake.Between(0, 2)).
						Unix()
					if owed := inv.openAt(paidAt - 1); owed > 0 {
						promise(inv, promisedAt, owed, promisedOn)
					}
				}
			}
		}

		var overdue []*cashInvoice
		for _, inv := range invs {
			if inv.due > 0 && inv.due < nowUnix-cashDay && inv.openAt(nowUnix) > 0 {
				overdue = append(overdue, inv)
			}
		}
		var nextFollowUp *int64
		if len(overdue) > 0 {
			oldest := overdue[0].due
			for _, inv := range overdue {
				oldest = min(oldest, inv.due)
			}
			assign(min(oldest+g.cashBetween(3*cashDay, 10*cashDay), nowUnix-2*cashHour))
			for range g.Fake.Between(1, 4) {
				inv := volumegen.Pick(g.Fake, overdue)
				at := g.cashBetween(
					max(inv.due+2*cashDay, acts[0].CreatedAt+cashMinute),
					nowUnix-cashHour,
				)
				contactNote(inv, at)
			}
			switch roll := g.Fake.Float(); {
			case roll < 0.35:
				inv := volumegen.Pick(g.Fake, overdue)
				at := g.cashBetween(max(nowUnix-5*cashDay, inv.due), nowUnix-cashHour)
				promisedOn := today.AddDate(0, 0, g.Fake.Between(1, 14)).Unix()
				promise(inv, at, inv.openAt(nowUnix), promisedOn)
				nextFollowUp = volumegen.Ptr(promisedOn + cashDay)
			case roll < 0.55:
				inv := volumegen.Pick(g.Fake, overdue)
				promisedOn := today.AddDate(0, 0, -g.Fake.Between(3, 20)).Unix()
				at := g.cashBetween(max(promisedOn-10*cashDay, inv.due), promisedOn-cashHour)
				if at > inv.due {
					promise(inv, at, inv.openAt(nowUnix), promisedOn)
					add(
						collection.ActivityNote,
						min(promisedOn+2*cashDay, nowUnix-cashMinute),
						collector,
						"Promised payment on "+inv.number+" did not arrive; escalating to the AP manager.",
						inv.id,
					)
				}
			}
			if nextFollowUp == nil {
				nextFollowUp = volumegen.Ptr(today.AddDate(0, 0, g.Fake.Between(1, 10)).Unix())
				a := add(
					collection.ActivityFollowUpScheduled,
					nowUnix-g.cashBetween(cashMinute, 6*cashHour),
					collector,
					"Follow up on the past-due balance",
					pulid.Nil,
				)
				a.FollowUpOn = nextFollowUp
			}
		}
		if len(acts) == 0 {
			continue
		}

		sort.SliceStable(acts, func(i, j int) bool { return acts[i].CreatedAt < acts[j].CreatedAt })
		for _, a := range acts {
			if a.CreatedByID.IsNil() {
				a.CreatedByID = g.Refs.AdminID
			}
		}
		first, last := acts[0].CreatedAt, acts[len(acts)-1].CreatedAt
		account := &collection.Account{
			ID:             volumegen.IDAt("cacc_", first),
			OrganizationID: g.Refs.OrgID,
			BusinessUnitID: g.Refs.BuID,
			CustomerID:     c.ID,
			CollectorID:    collector,
			NextFollowUpOn: nextFollowUp,
			Version:        int64(len(acts)),
			CreatedAt:      first,
			UpdatedAt:      last,
		}
		if lastPromise != nil {
			account.PromiseActivityID = lastPromise.ID
		}
		accounts = append(accounts, account)
		accountStamps = append(
			accountStamps,
			volumegen.Stamp{Id: account.ID, Created: first, Updated: last},
		)
		activities = append(activities, acts...)
	}
	if len(accounts) == 0 {
		return nil
	}

	if err := g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if txErr := volumegen.InsertRows(ctx, tx, collectiondb.Accounts.Handle, accounts); txErr != nil {
			return fmt.Errorf("insert collection accounts: %w", txErr)
		}
		if txErr := volumegen.InsertRows(ctx, tx, collectiondb.Activities.Handle, activities); txErr != nil {
			return fmt.Errorf("insert collection activities: %w", txErr)
		}
		return volumegen.Backdate(ctx, tx, collectiondb.Accounts.Handle, accountStamps)
	}); err != nil {
		return err
	}

	g.Report.Add("Collection accounts", len(accounts))
	g.Report.Add("Collection activities", len(activities))
	return nil
}

// -- Manual journals, reversals and the posting queue --

type cashJournalTemplate struct {
	key         string
	description string
	debit       string
	credit      string
	low, high   float64
	quarterly   bool
}

var cashJournalTemplates = []cashJournalTemplate{
	{
		key:         "dep-tractors",
		description: "Monthly depreciation - tractors",
		debit:       "6710",
		credit:      "1610",
		low:         38_000,
		high:        52_000,
	},
	{
		key:         "dep-trailers",
		description: "Monthly depreciation - trailers",
		debit:       "6710",
		credit:      "1620",
		low:         9_000,
		high:        14_000,
	},
	{
		key:         "insurance",
		description: "Prepaid liability insurance amortization",
		debit:       "5310",
		credit:      "1210",
		low:         21_000,
		high:        27_000,
	},
	{key: cashPayrollAccrual, description: "Accrued office payroll at month end", debit: "6010",
		credit: "2110", low: 31_000, high: 46_000},
	{
		key:         "bank-fees",
		description: "Bank service charges per statement",
		debit:       "6820",
		credit:      "1010",
		low:         180,
		high:        640,
	},
	{key: "bad-debt", description: "Allowance for doubtful accounts adjustment", debit: "6940",
		credit: "1190", low: 2_500, high: 9_500, quarterly: true},
}

var cashRejections = []string{
	"Amount does not tie to the fixed asset register",
	"Wrong period; resubmit for the current month",
	"Missing support for the accrual",
}

type cashManual struct {
	request  *manualjournal.Request
	lines    []*manualjournal.Line
	journal  *ledgervolume.Journal
	template cashJournalTemplate
	postedAt int64
}

func (g *cashPhase) cashAccounts(ctx context.Context) (map[string]pulid.ID, error) {
	codes := make([]string, 0, 2*len(cashJournalTemplates))
	for _, t := range cashJournalTemplates {
		codes = append(codes, t.debit, t.credit)
	}
	ga := glaccountdb.GLAccounts
	rows, err := ga.Pick(ga.ID, ga.AccountCode).Where(
		ga.AccountCode.In(codes...),
		ga.AllowManualJE.IsTrue(),
		ga.Status.Eq(domaintypes.StatusActive),
	).All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load manual journal accounts: %w", err)
	}
	out := make(map[string]pulid.ID, len(rows))
	for _, r := range rows {
		out[r.AccountCode] = r.ID
	}
	return out, nil
}

//nolint:gocognit,funlen // a month of journal requests and their outcomes is planned in one pass
func (g *cashPhase) cashManualJournals(ctx context.Context, st *cashState) error {
	if st.accounting == nil || len(st.setup.Periods) == 0 {
		return nil
	}
	accounts, err := g.cashAccounts(ctx)
	if err != nil {
		return err
	}
	var templates []cashJournalTemplate
	for _, t := range cashJournalTemplates {
		if accounts[t.debit].IsNotNil() && accounts[t.credit].IsNotNil() {
			templates = append(templates, t)
		}
	}
	if len(templates) == 0 {
		return nil
	}

	nowUnix := volumegen.Unix(g.Now)
	approval := st.accounting.RequireManualJEApproval
	firstPeriod := st.setup.Periods[0].StartDate
	nowLocal := g.Now.In(st.loc)
	currentMonth := time.Date(nowLocal.Year(), nowLocal.Month(), 1, 0, 0, 0, 0, st.loc)
	start := g.Start.In(st.loc)
	month := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, st.loc)

	var manuals []*cashManual
	for ; !month.After(currentMonth); month = month.AddDate(0, 1, 0) {
		monthEnd := month.AddDate(0, 1, 0).Add(-time.Second).Unix()
		if monthEnd < firstPeriod {
			continue
		}
		current := month.Equal(currentMonth)
		for _, t := range templates {
			if t.quarterly && month.Month()%3 != 0 {
				continue
			}
			accountingDate := min(monthEnd-6*cashHour, nowUnix-2*cashHour)
			created := min(accountingDate+g.cashBetween(cashHour, 3*cashDay), nowUnix-cashHour)
			if current {
				created = min(accountingDate-g.cashBetween(cashHour, 2*cashDay), nowUnix-cashHour)
			}
			period, date, ok := g.ResolvePeriod(st.setup, accountingDate)
			if !ok {
				continue
			}
			amount := money.MinorUnits(g.Fake.Money(t.low, t.high))
			author := g.cashUser()
			req := &manualjournal.Request{
				ID:                      volumegen.IDAt("mjr_", created),
				OrganizationID:          g.Refs.OrgID,
				BusinessUnitID:          g.Refs.BuID,
				Status:                  manualjournal.StatusDraft,
				Description:             t.description + " - " + month.Format("January 2006"),
				Reason:                  "Month-end close",
				AccountingDate:          date,
				RequestedFiscalYearID:   period.FiscalYearID,
				RequestedFiscalPeriodID: period.ID,
				CurrencyCode:            money.DefaultCurrencyCode,
				TotalDebit:              amount,
				TotalCredit:             amount,
				CreatedByID:             author,
				UpdatedByID:             author,
				Version:                 1,
				CreatedAt:               created,
				UpdatedAt:               created,
			}
			m := &cashManual{request: req, template: t}
			m.lines = []*manualjournal.Line{
				{
					ID: volumegen.IDAt(
						"mjrl_",
						created,
					), OrganizationID: g.Refs.OrgID, BusinessUnitID: g.Refs.BuID,
					ManualJournalRequestID: req.ID, LineNumber: 1, GLAccountID: accounts[t.debit],
					Description: t.description, DebitAmount: amount, CreatedAt: created, UpdatedAt: created,
				},
				{
					ID: volumegen.IDAt(
						"mjrl_",
						created,
					), OrganizationID: g.Refs.OrgID, BusinessUnitID: g.Refs.BuID,
					ManualJournalRequestID: req.ID, LineNumber: 2, GLAccountID: accounts[t.credit],
					Description: t.description, CreditAmount: amount, CreatedAt: created, UpdatedAt: created,
				},
			}

			approver := g.cashUser()
			approve := func(at int64) {
				req.Status = manualjournal.StatusApproved
				req.ApprovedAt = volumegen.Ptr(at)
				req.ApprovedByID = approver
				req.UpdatedByID = approver
				req.UpdatedAt = at
			}
			submittedAt := min(created+g.cashBetween(10*cashMinute, 6*cashHour), nowUnix-cashMinute)
			roll := g.Fake.Float()
			switch {
			case current && roll < 0.3:
			case current && roll < 0.65:
				if approval {
					req.Status = manualjournal.StatusPendingApproval
					req.UpdatedAt = submittedAt
				} else {
					approve(submittedAt)
				}
			case current:
				approve(min(submittedAt+g.cashBetween(cashHour, cashDay), nowUnix-cashMinute))
			case roll < 0.04:
				rejectedAt := min(
					submittedAt+g.cashBetween(cashHour, 2*cashDay),
					nowUnix-cashMinute,
				)
				req.Status = manualjournal.StatusRejected
				req.RejectedAt = volumegen.Ptr(rejectedAt)
				req.RejectedByID = approver
				req.RejectionReason = volumegen.Pick(g.Fake, cashRejections)
				req.UpdatedAt = rejectedAt
			case roll < 0.06:
				cancelledAt := min(submittedAt+g.cashBetween(cashHour, cashDay), nowUnix-cashMinute)
				req.Status = manualjournal.StatusCancelled
				req.CancelledAt = volumegen.Ptr(cancelledAt)
				req.CancelledByID = author
				req.CancelReason = "Duplicate of another request"
				req.UpdatedAt = cancelledAt
			default:
				approvedAt := min(
					submittedAt+g.cashBetween(cashHour, 2*cashDay),
					nowUnix-2*cashMinute,
				)
				approve(approvedAt)
				m.postedAt = min(
					approvedAt+g.cashBetween(10*cashMinute, cashDay),
					nowUnix-cashMinute,
				)
				journal, jErr := g.NewJournal(st.setup, date)
				if jErr != nil {
					return jErr
				}
				journal.ActorID = approver
				journal.Description = req.Description
				journal.SourceType = "ManualJournalRequest"
				journal.SourceID = req.ID
				journal.Event = accountingcontrol.JournalSourceEventType("ManualJournalPosted")
				journal.ReferenceType = "ManualJournalRequest"
				journal.Idempotency = "manual-journal-posted:" + req.ID.String()
				journal.Legs = []ledgervolume.Leg{
					{Account: accounts[t.debit], Debit: amount},
					{Account: accounts[t.credit], Credit: amount},
				}
				m.journal = journal
				req.Status = manualjournal.StatusPosted
				req.PostedBatchID = journal.BatchID
				req.UpdatedByID = approver
				req.UpdatedAt = m.postedAt
			}
			manuals = append(manuals, m)
		}
	}
	if len(manuals) == 0 {
		return nil
	}

	slots := make([]volumegen.NumberSlot, len(manuals))
	for i, m := range manuals {
		slots[i] = volumegen.NumberSlot{At: m.request.CreatedAt}
	}
	numbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeManualJournalRequest, slots)
	if err != nil {
		return err
	}
	posted := &ledgervolume.Rows{Balances: make(map[ledgervolume.BalanceKey]*ledgervolume.Balance)}
	for i, m := range manuals {
		m.request.RequestNumber = numbers[i]
		if m.journal != nil {
			m.journal.Document = numbers[i]
			posted.Journals = append(posted.Journals, m.journal)
		}
	}
	if err = g.Materialize(ctx, posted); err != nil {
		return err
	}
	manualBatches := make(map[pulid.ID]bool, len(posted.Batches))
	for _, b := range posted.Batches {
		b.BatchType = "Manual"
		manualBatches[b.ID] = true
	}
	for _, e := range posted.Entries {
		e.RequiresApproval = approval
	}

	reversals, reversed, err := g.cashReversals(ctx, st, manuals, posted)
	if err != nil {
		return err
	}

	requests := make([]*manualjournal.Request, 0, len(manuals))
	var lines []*manualjournal.Line
	requestStamps := make([]volumegen.Stamp, 0, len(manuals))
	lineStamps := make([]volumegen.Stamp, 0, 2*len(manuals))
	for _, m := range manuals {
		requests = append(requests, m.request)
		requestStamps = append(
			requestStamps,
			volumegen.Stamp{
				Id:      m.request.ID,
				Created: m.request.CreatedAt,
				Updated: m.request.UpdatedAt,
			},
		)
		for _, l := range m.lines {
			lines = append(lines, l)
			lineStamps = append(
				lineStamps,
				volumegen.Stamp{Id: l.ID, Created: l.CreatedAt, Updated: l.CreatedAt},
			)
		}
	}

	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if txErr := g.Write(ctx, tx, posted); txErr != nil {
			return txErr
		}
		if reversals.rows != nil {
			if txErr := g.Write(ctx, tx, reversals.rows); txErr != nil {
				return txErr
			}
		}
		if txErr := volumegen.InsertRows(ctx, tx, manualjournaldb.Requests.Handle, requests); txErr != nil {
			return fmt.Errorf("insert manual journal requests: %w", txErr)
		}
		if txErr := volumegen.InsertRows(ctx, tx, manualjournaldb.Lines.Handle, lines); txErr != nil {
			return fmt.Errorf("insert manual journal lines: %w", txErr)
		}
		if txErr := volumegen.Backdate(ctx, tx, manualjournaldb.Requests.Handle, requestStamps); txErr != nil {
			return txErr
		}
		if txErr := volumegen.Backdate(ctx, tx, manualjournaldb.Lines.Handle, lineStamps); txErr != nil {
			return txErr
		}
		if len(reversals.entities) > 0 {
			if txErr := volumegen.InsertRows(ctx, tx, journalreversaldb.Reversals.Handle, reversals.entities); txErr != nil {
				return fmt.Errorf("insert journal reversals: %w", txErr)
			}
		}
		return g.cashMarkReversed(ctx, tx, reversed)
	}); err != nil {
		return err
	}

	g.Report.Add("Manual journal requests", len(requests))
	g.Report.Add("Journal reversals", len(reversals.entities))
	g.Report.Add("Journal entries", len(posted.Entries)+reversals.entryCount())
	return nil
}

type cashReversalSet struct {
	rows     *ledgervolume.Rows
	entities []*journalreversal.Reversal
}

func (s *cashReversalSet) entryCount() int {
	if s.rows == nil {
		return 0
	}
	return len(s.rows.Entries)
}

type cashReversedEntry struct {
	original pulid.ID
	reversal pulid.ID
	date     int64
	reason   string
	by       pulid.ID
}

// cashReversals reverses each posted month-end payroll accrual on the first
// day of the next month, the way an accountant reverses an accrual.
//
//nolint:funlen // a reversal request, its journal and its link to the original are built together
func (g *cashPhase) cashReversals(
	ctx context.Context,
	st *cashState,
	manuals []*cashManual,
	posted *ledgervolume.Rows,
) (*cashReversalSet, []cashReversedEntry, error) {
	set := &cashReversalSet{}
	if st.accounting.JournalReversalPolicy == accountingcontrol.JournalReversalPolicyDisallow {
		return set, nil, nil
	}
	nowUnix := volumegen.Unix(g.Now)
	approval := st.accounting.RequireManualJEApproval
	entries := make(map[pulid.ID]int, len(posted.Entries))
	for i, e := range posted.Entries {
		entries[e.ID] = i
	}

	set.rows = &ledgervolume.Rows{Balances: make(map[ledgervolume.BalanceKey]*ledgervolume.Balance)}
	type pending struct {
		entity   *journalreversal.Reversal
		journal  *ledgervolume.Journal
		original pulid.ID
		number   string
	}
	var postedReversals []*pending
	const reason = "Reverse the month-end payroll accrual"
	for _, m := range manuals {
		if m.template.key != cashPayrollAccrual || m.journal == nil {
			continue
		}
		idx, ok := entries[m.journal.EntryID]
		if !ok {
			continue
		}
		original := posted.Entries[idx]
		reverseOn := cashLocalDay(original.AccountingDate, st.loc).AddDate(0, 0, 1).Unix()
		requestedAt := reverseOn + g.cashBetween(cashHour, 20*cashHour)
		if requestedAt > nowUnix-cashHour {
			continue
		}
		period, date, ok := g.ResolvePeriod(st.setup, reverseOn)
		if !ok {
			continue
		}
		requester := g.cashUser()
		entity := &journalreversal.Reversal{
			ID:                      volumegen.IDAt(journalreversal.IDPrefix, requestedAt),
			OrganizationID:          g.Refs.OrgID,
			BusinessUnitID:          g.Refs.BuID,
			OriginalJournalEntryID:  original.ID,
			Status:                  journalreversal.StatusApproved,
			RequestedAccountingDate: date,
			ResolvedFiscalYearID:    period.FiscalYearID,
			ResolvedFiscalPeriodID:  period.ID,
			ReasonCode:              "AccrualReversal",
			ReasonText:              reason,
			RequestedByID:           requester,
			ApprovedByID:            requester,
			ApprovedAt:              volumegen.Ptr(requestedAt),
			Version:                 1,
			CreatedAt:               requestedAt,
			UpdatedAt:               requestedAt,
		}
		if approval {
			entity.Status = journalreversal.StatusPendingApproval
			entity.ApprovedByID = pulid.Nil
			entity.ApprovedAt = nil
		}
		set.entities = append(set.entities, entity)
		if nowUnix-requestedAt < 3*cashDay {
			continue
		}

		approver := g.cashUser()
		approvedAt := min(requestedAt+g.cashBetween(cashHour, cashDay), nowUnix-2*cashMinute)
		postedAt := min(approvedAt+g.cashBetween(10*cashMinute, cashDay), nowUnix-cashMinute)
		if approval {
			entity.ApprovedByID = approver
			entity.ApprovedAt = volumegen.Ptr(approvedAt)
		}
		journal, err := g.NewJournal(st.setup, date)
		if err != nil {
			return nil, nil, err
		}
		journal.ActorID = approver
		journal.Description = original.Description
		journal.SourceType = "JournalEntry"
		journal.SourceID = original.ID
		journal.Document = original.EntryNumber
		journal.Event = accountingcontrol.JournalSourceEventType("JournalReversalCreated")
		journal.ReferenceType = "JournalReversal"
		journal.Idempotency = "journal-reversal:" + entity.ID.String()
		journal.Auto = true
		for _, leg := range m.journal.Legs {
			journal.Legs = append(journal.Legs, ledgervolume.Leg{
				Account: leg.Account, Debit: leg.Credit, Credit: leg.Debit, Customer: leg.Customer,
			})
		}
		set.rows.Journals = append(set.rows.Journals, journal)
		entity.Status = journalreversal.StatusPosted
		entity.PostedByID = approver
		entity.PostedAt = volumegen.Ptr(postedAt)
		entity.ReversalJournalEntryID = journal.EntryID
		entity.PostedBatchID = journal.BatchID
		entity.UpdatedAt = postedAt
		postedReversals = append(postedReversals, &pending{
			entity: entity, journal: journal, original: original.ID, number: original.EntryNumber,
		})
	}
	if len(set.rows.Journals) == 0 {
		set.rows = nil
		return set, nil, nil
	}
	if err := g.Materialize(ctx, set.rows); err != nil {
		return nil, nil, err
	}

	byEntry := make(map[pulid.ID]*pending, len(postedReversals))
	for _, p := range postedReversals {
		byEntry[p.journal.EntryID] = p
	}
	for _, b := range set.rows.Batches {
		b.BatchType = "Reversal"
		b.Description = "Journal reversal"
	}
	reversed := make([]cashReversedEntry, 0, len(postedReversals))
	for _, e := range set.rows.Entries {
		p := byEntry[e.ID]
		if p == nil {
			continue
		}
		e.EntryType = "Reversal"
		e.IsReversal = true
		e.ReversalOfID = p.original
		e.ReversalDate = volumegen.Ptr(e.AccountingDate)
		e.ReversalReason = reason
		e.ReferenceNumber = p.number
		e.ReferenceID = p.entity.ID.String()
		reversed = append(reversed, cashReversedEntry{
			original: p.original, reversal: e.ID, date: p.entity.RequestedAccountingDate,
			reason: reason, by: p.entity.PostedByID,
		})
	}

	return set, reversed, nil
}

func (g *cashPhase) cashMarkReversed(
	ctx context.Context,
	tx *keel.DB,
	reversed []cashReversedEntry,
) error {
	if len(reversed) == 0 {
		return nil
	}
	originals := make([]string, len(reversed))
	reversals := make([]string, len(reversed))
	dates := make([]int64, len(reversed))
	users := make([]string, len(reversed))
	for i, r := range reversed {
		originals[i] = r.original.String()
		reversals[i] = r.reversal.String()
		dates[i] = r.date
		users[i] = r.by.String()
	}
	je := journalentrydb.JournalEntries
	d := keel.Unnest("d", []string{"id", "reversal_id", "reversal_date", "user_id"},
		originals, reversals, dates, users)
	if _, err := je.With(d).Join(d, d.Col("id").EqCol(je.ID)).Where(keel.True()).Update(
		ctx,
		tx,
		je.Status.To(journalentry.StatusReversed),
		je.ReversedByID.ToExpr(d.Col("reversal_id")),
		je.ReversalDate.ToExpr(d.Col("reversal_date")),
		je.ReversalReason.To(reversed[0].reason),
		je.UpdatedByID.ToExpr(d.Col("user_id")),
		volumegen.KeepUpdated(je.Handle),
	); err != nil {
		return fmt.Errorf("mark reversed journal entries: %w", err)
	}
	return nil
}

// cashJournalsToPost puts the most recent system journals back in the posting
// queue when the organization posts journals by hand. Under manual posting the
// application writes every system journal Pending (or Approved when no
// approval is required) until an accountant posts it, so the last few days'
// journals have not been posted yet; their amounts come back off the balances.
//
//nolint:funlen // the queue state and the balance rollback must change together
func (g *cashPhase) cashJournalsToPost(ctx context.Context, st *cashState) error {
	if st.accounting == nil ||
		st.accounting.JournalPostingMode != accountingcontrol.JournalPostingModeManual {
		return nil
	}
	cutoff := volumegen.Unix(g.Now) - cashPendingWindow

	je := journalentrydb.JournalEntries
	targets, err := je.Pick(je.ID, je.BatchID, je.FiscalYearID, je.FiscalPeriodID).Where(
		je.Status.Eq(journalentry.Status(cashJournalPosted)),
		je.IsAutoGenerated.IsTrue(),
		je.IsReversal.IsFalse(),
		je.ReversedByID.IsNull(),
		je.AccountingDate.Gte(cutoff),
	).All(ctx, g.DB)
	if err != nil {
		return fmt.Errorf("load recent system journals: %w", err)
	}
	if len(targets) == 0 {
		return nil
	}
	entryIDs := make([]pulid.ID, 0, len(targets))
	batchIDs := make([]pulid.ID, 0, len(targets))
	byEntry := make(map[pulid.ID]*journalentry.JournalEntry, len(targets))
	for _, t := range targets {
		entryIDs = append(entryIDs, t.ID)
		byEntry[t.ID] = t
		if t.BatchID.IsNotNil() {
			batchIDs = append(batchIDs, t.BatchID)
		}
	}

	status := cashJournalApproved
	if st.accounting.RequireManualJEApproval {
		status = cashJournalPending
	}

	jel := journalentrydb.JournalEntryLines
	lines, err := jel.Pick(jel.JournalEntryID, jel.GLAccountID, jel.DebitAmount, jel.CreditAmount, jel.NetAmount).
		Where(jel.JournalEntryID.In(entryIDs...)).
		All(ctx, g.DB)
	if err != nil {
		return fmt.Errorf("load recent system journal lines: %w", err)
	}

	if err = g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if err := volumegen.RollBackBalances(ctx, tx, lines, byEntry); err != nil {
			return err
		}

		approved := status == cashJournalApproved
		sets := []keel.Assignment{
			je.Status.To(journalentry.Status(status)),
			je.IsPosted.To(false),
			je.PostedAt.ToNull(),
			je.PostedByID.ToNull(),
			je.RequiresApproval.To(!approved),
			je.IsApproved.To(approved),
			volumegen.KeepUpdated(je.Handle),
		}
		if !approved {
			sets = append(sets, je.ApprovedAt.ToNull(), je.ApprovedByID.ToNull())
		}
		if _, err := je.Where(je.ID.In(entryIDs...)).Update(ctx, tx, sets...); err != nil {
			return fmt.Errorf("queue journal entries: %w", err)
		}
		if len(batchIDs) > 0 {
			jb := journalentrydb.JournalBatches
			if _, err := jb.Where(jb.ID.In(batchIDs...)).Update(
				ctx,
				tx,
				jb.Status.To(status),
				jb.PostedAt.ToNull(),
				jb.PostedByID.ToNull(),
				volumegen.KeepUpdated(jb.Handle),
			); err != nil {
				return fmt.Errorf("queue journal batches: %w", err)
			}
		}
		js := journalsourcedb.Sources
		if _, err := js.Where(js.JournalEntryID.In(entryIDs...)).
			Update(ctx, tx, js.Status.To(status)); err != nil {
			return fmt.Errorf("queue journal sources: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	g.Report.Add("Journals waiting to post", len(targets))
	return nil
}

type cashPhase struct {
	*ledgervolume.Ledger
}

func Seed(ctx context.Context, g *volumegen.Generator) error {
	return (&cashPhase{Ledger: &ledgervolume.Ledger{Generator: g}}).seedCash(ctx)
}
