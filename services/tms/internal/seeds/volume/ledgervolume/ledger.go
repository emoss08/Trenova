package ledgervolume

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource/journalsourcedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueueevent"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueueevent/billingqueueeventdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol/accountingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	bilCountry          = "United States"
	bilAccountCash      = "1010"
	bilAccountAR        = "1110"
	bilAccountRevenue   = "4010"
	bilAccountUnapplied = "2160"
	bilStatusPosted     = "Posted"
	bilBatchTypeSystem  = "System"
)

type Setup struct {
	PostInvoices bool
	PostPayments bool
	AR           pulid.ID
	Revenue      pulid.ID
	Cash         pulid.ID
	unapplied    pulid.ID
	applied      pulid.ID
	Periods      []*fiscalperiod.FiscalPeriod
	Names        map[pulid.ID]string
}

type Leg struct {
	Account  pulid.ID
	Debit    int64
	Credit   int64
	Customer pulid.ID
}

type Journal struct {
	BatchID       pulid.ID
	EntryID       pulid.ID
	period        *fiscalperiod.FiscalPeriod
	Date          int64
	ActorID       pulid.ID
	Description   string
	SourceType    string
	SourceID      pulid.ID
	Document      string
	Event         accountingcontrol.JournalSourceEventType
	ReferenceType string
	Idempotency   string
	Auto          bool
	Legs          []Leg
}

type BalanceKey struct {
	account pulid.ID
	year    pulid.ID
	period  pulid.ID
}

type Balance struct {
	debit     int64
	credit    int64
	net       int64
	lastEntry pulid.ID
	lastDate  int64
}

type ItemInvoice struct {
	Item    pulid.ID
	Invoice pulid.ID
}

type Rows struct {
	Items        []*billingqueue.BillingQueueItem
	Events       []*billingqueueevent.ItemEvent
	Invoices     []*invoice.Invoice
	Lines        []*invoice.InvoiceLine
	Links        []ItemInvoice
	MemoApps     []*customerpayment.CreditMemoApplication
	Payments     []*customerpayment.Payment
	Apps         []*customerpayment.Application
	Ledger       []*customerledger.CustomerLedgerEntry
	Journals     []*Journal
	Batches      []*journalentry.JournalBatch
	Entries      []*journalentry.JournalEntry
	JournalLines []*journalentry.JournalEntryLine
	sources      []*journalsource.Source
	srcLinks     []*journalsource.SourceJournalLink
	Comments     []*shipment.ShipmentComment
	Balances     map[BalanceKey]*Balance
	MemoCount    int
}

type PaymentPlan struct {
	Invoice *invoice.Invoice
	Amount  int64
	Date    int64
}

func (g *Ledger) LoadSetup(ctx context.Context) (*Setup, error) {
	setup := &Setup{Names: make(map[pulid.ID]string, len(g.Refs.UserIDs))}

	users, err := g.UsersByID(ctx)
	if err != nil {
		return nil, err
	}
	for id, u := range users {
		setup.Names[id] = u.Name
	}

	control, err := g.AccountingControl(ctx)
	if err != nil {
		return nil, err
	}
	if control == nil {
		g.Opts.Logf("  ! no accounting control; invoices post without journals or ledger entries")
		return setup, nil
	}

	if err := g.bilConfigureAccounts(ctx, control); err != nil {
		return nil, err
	}
	setup.AR = control.DefaultARAccountID
	setup.Revenue = control.DefaultRevenueAccountID
	setup.Cash = control.DefaultCashAccountID
	setup.unapplied = control.DefaultUnappliedCashAccountID

	if err := g.bilEnsureFiscalYears(ctx); err != nil {
		return nil, err
	}
	fp := fiscalperioddb.FiscalPeriods
	if setup.Periods, err = fp.Where(
		fp.Status.Eq(fiscalperiod.StatusOpen),
		keel.Not(fp.IsAdjusting.IsTrue()),
	).Order(fp.StartDate.Asc()).All(ctx, g.DB); err != nil {
		return nil, fmt.Errorf("load fiscal periods: %w", err)
	}
	if len(setup.Periods) == 0 {
		g.Opts.Logf("  ! no open fiscal periods; invoices post without journals or ledger entries")
		return setup, nil
	}

	accrual := control.AccountingBasis != accountingcontrol.AccountingBasisCash
	onInvoice := control.RevenueRecognitionPolicy == accountingcontrol.RevenueRecognitionOnInvoicePost
	setup.PostInvoices = accrual && onInvoice && setup.AR.IsNotNil() && setup.Revenue.IsNotNil()
	setup.applied = setup.AR
	if !accrual || !onInvoice {
		setup.applied = setup.Revenue
	}
	setup.PostPayments = setup.Cash.IsNotNil() && setup.applied.IsNotNil()

	return setup, nil
}

// bilEnsureFiscalYears opens a calendar fiscal year for every year the history
// reaches back into, so old invoices post on their own dates instead of piling
// onto the first open period.
func (g *Ledger) bilEnsureFiscalYears(ctx context.Context) error {
	fy := fiscalyeardb.FiscalYears
	existing, err := fy.Pick(fy.Year).All(ctx, g.DB)
	if err != nil {
		return fmt.Errorf("load fiscal years: %w", err)
	}
	have := make(map[int]struct{}, len(existing))
	for _, row := range existing {
		have[row.Year] = struct{}{}
	}

	return g.InTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		for year := g.Start.Year(); year <= g.Now.Year(); year++ {
			if _, ok := have[year]; ok {
				continue
			}
			begin := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
			fy := &fiscalyear.FiscalYear{
				OrganizationID: g.Refs.OrgID,
				BusinessUnitID: g.Refs.BuID,
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
			g.Report.Add("Fiscal years", 1)
		}
		return nil
	})
}

func (g *Ledger) bilConfigureAccounts(
	ctx context.Context,
	control *accountingcontrol.AccountingControl,
) error {
	byCode, err := g.GLAccountsByCode(
		ctx,
		bilAccountCash,
		bilAccountAR,
		bilAccountRevenue,
		bilAccountUnapplied,
	)
	if err != nil {
		return err
	}

	ac := accountingcontroldb.AccountingControls
	defaults := []struct {
		column keelkinds.PulidIDCol
		code   string
		field  *pulid.ID
	}{
		{column: ac.DefaultARAccountID, code: bilAccountAR, field: &control.DefaultARAccountID},
		{
			column: ac.DefaultRevenueAccountID,
			code:   bilAccountRevenue,
			field:  &control.DefaultRevenueAccountID,
		},
		{
			column: ac.DefaultCashAccountID,
			code:   bilAccountCash,
			field:  &control.DefaultCashAccountID,
		},
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
		if _, err = ac.Where(def.column.IsNull()).Update(ctx, g.DB, def.column.To(id)); err != nil {
			return fmt.Errorf("set the default %s account: %w", def.code, err)
		}
		*def.field = id
	}

	return nil
}

func Sum(lines []*invoice.InvoiceLine, typ invoice.InvoiceLineType) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines {
		if typ == "" || line.Type == typ {
			total = total.Add(line.Amount)
		}
	}
	return total
}

func BillTo(inv *invoice.Invoice, cus *volumegen.CustomerRecord) {
	inv.BillToName = cus.Name
	inv.BillToCode = cus.Code
	inv.BillToAddressLine1 = cus.AddressLine1
	inv.BillToCity = cus.City
	inv.BillToState = cus.StateAbbr
	inv.BillToPostalCode = cus.PostalCode
	inv.BillToCountry = bilCountry
}

func (g *Ledger) ResolvePeriod(
	setup *Setup,
	date int64,
) (*fiscalperiod.FiscalPeriod, int64, bool) {
	periods := setup.Periods
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

func (g *Ledger) NewJournal(
	setup *Setup,
	date int64,
) (*Journal, error) {
	period, accountingDate, ok := g.ResolvePeriod(setup, date)
	if !ok {
		return nil, fmt.Errorf("%w: no open fiscal period", volumegen.ErrMissingSeedData)
	}
	return &Journal{
		BatchID: volumegen.IDAt("jb_", accountingDate),
		EntryID: volumegen.IDAt("je_", accountingDate),
		period:  period,
		Date:    accountingDate,
	}, nil
}

func (g *Ledger) PostInvoice(
	setup *Setup,
	rows *Rows,
	inv *invoice.Invoice,
	related pulid.ID,
	entity *billingqueue.BillingQueueItem,
) error {
	event := accountingcontrol.JournalSourceEventInvoicePosted
	debit, credit := setup.AR, setup.Revenue
	ledgerAmount := inv.TotalAmountMinor
	amount := inv.TotalAmountMinor
	if inv.IsCreditMemo() {
		event = accountingcontrol.JournalSourceEventCreditMemoPosted
		debit, credit = credit, debit
		amount = inv.CreditTotalMinor()
		ledgerAmount = -amount
	}
	if amount == 0 {
		return nil
	}

	journal, err := g.NewJournal(setup, *inv.PostedAt)
	if err != nil {
		return err
	}
	actor := g.Refs.AdminID
	if entity != nil && entity.AssignedBillerID != nil {
		actor = *entity.AssignedBillerID
	}
	journal.ActorID = actor
	journal.Description = "Invoice posted for " + inv.Number
	journal.SourceType = "Invoice"
	journal.SourceID = inv.ID
	journal.Document = inv.Number
	journal.Event = event
	journal.ReferenceType = event.String()
	journal.Idempotency = "invoice-posted:" + inv.ID.String()
	journal.Auto = true
	journal.Legs = []Leg{
		{Account: debit, Debit: amount, Customer: inv.CustomerID},
		{Account: credit, Credit: amount, Customer: inv.CustomerID},
	}
	rows.Journals = append(rows.Journals, journal)

	rows.Ledger = append(rows.Ledger, &customerledger.CustomerLedgerEntry{
		ID:               volumegen.IDAt("cledg_", journal.Date),
		OrganizationID:   g.Refs.OrgID,
		BusinessUnitID:   g.Refs.BuID,
		CustomerID:       inv.CustomerID,
		SourceObjectType: "Invoice",
		SourceObjectID:   inv.ID.String(),
		SourceEventType:  event.String(),
		RelatedInvoiceID: related,
		DocumentNumber:   inv.Number,
		TransactionDate:  journal.Date,
		LineNumber:       1,
		AmountMinor:      ledgerAmount,
		CreatedByID:      actor,
		CreatedAt:        *inv.PostedAt,
	})

	return nil
}

func (g *Ledger) bilPaymentMethod() (customerpayment.Method, string) {
	switch g.Fake.Weighted([]int{55, 30, 10, 3, 2}) {
	case 0:
		return customerpayment.MethodACH, "ACH" + g.Fake.Digits(10)
	case 1:
		return customerpayment.MethodCheck, "CHK " + g.Fake.Digits(6)
	case 2:
		return customerpayment.MethodWire, "FW" + g.Fake.Digits(12)
	case 3:
		return customerpayment.MethodCard, "CC-" + g.Fake.Digits(8)
	default:
		return customerpayment.MethodOther, "LBX-" + g.Fake.Digits(8)
	}
}

func (g *Ledger) Payment(
	setup *Setup,
	rows *Rows,
	cus *volumegen.CustomerRecord,
	date int64,
	plans []*PaymentPlan,
) error {
	journal, err := g.NewJournal(setup, date)
	if err != nil {
		return err
	}
	actor := g.PickUser(g.Refs.UserIDs)
	method, reference := g.bilPaymentMethod()

	payment := &customerpayment.Payment{
		ID:              volumegen.IDAt("cpay_", date),
		OrganizationID:  g.Refs.OrgID,
		BusinessUnitID:  g.Refs.BuID,
		CustomerID:      cus.ID,
		PaymentDate:     date,
		AccountingDate:  journal.Date,
		Status:          customerpayment.StatusPosted,
		PaymentMethod:   method,
		ReferenceNumber: reference,
		CurrencyCode:    money.DefaultCurrencyCode,
		PostedBatchID:   journal.BatchID,
		CreatedByID:     actor,
		UpdatedByID:     actor,
		CreatedAt:       date,
		UpdatedAt:       date,
	}

	numbers := make([]string, 0, len(plans))
	for _, plan := range plans {
		plan.Invoice.ApplyPaymentMinor(plan.Amount)
		plan.Invoice.UpdatedAt = max(plan.Invoice.UpdatedAt, date)
		payment.Applications = append(payment.Applications, &customerpayment.Application{
			ID:                 volumegen.IDAt("cpapp_", date),
			OrganizationID:     g.Refs.OrgID,
			BusinessUnitID:     g.Refs.BuID,
			CustomerPaymentID:  payment.ID,
			InvoiceID:          plan.Invoice.ID,
			AppliedAmountMinor: plan.Amount,
			CreatedAt:          date,
			UpdatedAt:          date,
		})
		payment.AmountMinor += plan.Amount
		numbers = append(numbers, plan.Invoice.Number)
	}
	if setup.unapplied.IsNotNil() && g.Fake.Chance(0.04) {
		payment.AmountMinor += int64(g.Fake.Between(50, 750)) * 100
	}
	payment.SyncAmounts()
	payment.Memo = bilRemittanceMemo(numbers, payment.UnappliedAmountMinor)

	journal.ActorID = actor
	journal.Description = "Customer payment " + payment.DocumentLabel()
	journal.SourceType = "CustomerPayment"
	journal.SourceID = payment.ID
	journal.Document = payment.ReferenceNumber
	journal.Event = accountingcontrol.JournalSourceEventCustomerPaymentPosted
	journal.ReferenceType = accountingcontrol.JournalSourceEventCustomerPaymentPosted.String()
	journal.Idempotency = "customer-payment-posted:" + payment.ID.String()
	journal.Legs = append(journal.Legs, Leg{
		Account: setup.Cash, Debit: payment.AmountMinor, Customer: cus.ID,
	})
	for i, app := range payment.Applications {
		journal.Legs = append(journal.Legs, Leg{
			Account: setup.applied, Credit: app.AppliedAmountMinor, Customer: cus.ID,
		})
		rows.Ledger = append(rows.Ledger, &customerledger.CustomerLedgerEntry{
			ID:               volumegen.IDAt("cledg_", journal.Date),
			OrganizationID:   g.Refs.OrgID,
			BusinessUnitID:   g.Refs.BuID,
			CustomerID:       cus.ID,
			SourceObjectType: "CustomerPayment",
			SourceObjectID:   payment.ID.String(),
			SourceEventType:  journal.Event.String(),
			RelatedInvoiceID: app.InvoiceID,
			DocumentNumber:   payment.ReferenceNumber,
			TransactionDate:  journal.Date,
			LineNumber:       i + 1,
			AmountMinor:      -app.AppliedAmountMinor,
			CreatedByID:      actor,
			CreatedAt:        date,
		})
	}
	if payment.UnappliedAmountMinor > 0 {
		journal.Legs = append(journal.Legs, Leg{
			Account: setup.unapplied, Credit: payment.UnappliedAmountMinor, Customer: cus.ID,
		})
	}

	rows.Journals = append(rows.Journals, journal)
	rows.Payments = append(rows.Payments, payment)
	g.Payments = append(g.Payments, payment)
	rows.Apps = append(rows.Apps, payment.Applications...)

	return nil
}

func bilRemittanceMemo(numbers []string, unappliedMinor int64) string {
	memo := "Remittance for " + numbers[0]
	if len(numbers) > 1 {
		memo = fmt.Sprintf(
			"Remittance for %d invoices: %s",
			len(numbers),
			strings.Join(numbers, ", "),
		)
	}
	if unappliedMinor > 0 {
		memo += "; " + money.DecimalFromMinor(unappliedMinor).
			StringFixed(2) +
			" with no remittance detail"
	}
	return memo
}

func (g *Ledger) Materialize(ctx context.Context, rows *Rows) error {
	if len(rows.Journals) == 0 {
		return nil
	}

	slots := make([]volumegen.NumberSlot, len(rows.Journals))
	for i, j := range rows.Journals {
		slots[i] = volumegen.NumberSlot{At: j.Date}
	}
	batchNumbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeJournalBatch, slots)
	if err != nil {
		return err
	}
	entryNumbers, err := g.Numbers.Allocate(ctx, sequenceconfig.SequenceTypeJournalEntry, slots)
	if err != nil {
		return err
	}

	for i, j := range rows.Journals {
		var totalDebit, totalCredit int64
		lines := make([]*journalentry.JournalEntryLine, 0, len(j.Legs))
		for _, leg := range j.Legs {
			if leg.Debit == 0 && leg.Credit == 0 {
				continue
			}
			totalDebit += leg.Debit
			totalCredit += leg.Credit
			lines = append(lines, &journalentry.JournalEntryLine{
				ID:             volumegen.IDAt("jel_", j.Date),
				BusinessUnitID: g.Refs.BuID,
				OrganizationID: g.Refs.OrgID,
				JournalEntryID: j.EntryID,
				GLAccountID:    leg.Account,
				LineNumber:     int16(len(lines) + 1), //nolint:gosec // bounded by leg count
				Description:    j.Description,
				DebitAmount:    leg.Debit,
				CreditAmount:   leg.Credit,
				NetAmount:      leg.Debit - leg.Credit,
				CustomerID:     leg.Customer,
				CreatedAt:      j.Date,
				UpdatedAt:      j.Date,
			})
		}
		if totalDebit != totalCredit || len(lines) < 2 {
			return fmt.Errorf("journal for %s does not balance: %d debit, %d credit",
				j.Document, totalDebit, totalCredit)
		}

		date := j.Date
		rows.Batches = append(rows.Batches, &journalentry.JournalBatch{
			ID:             j.BatchID,
			OrganizationID: g.Refs.OrgID,
			BusinessUnitID: g.Refs.BuID,
			BatchNumber:    batchNumbers[i],
			BatchType:      bilBatchTypeSystem,
			Status:         bilStatusPosted,
			Description:    j.Description,
			AccountingDate: date,
			FiscalYearID:   j.period.FiscalYearID,
			FiscalPeriodID: j.period.ID,
			EntryCount:     1,
			PostedAt:       &date,
			PostedByID:     j.ActorID,
			CreatedByID:    j.ActorID,
			UpdatedByID:    j.ActorID,
			CreatedAt:      date,
			UpdatedAt:      date,
		})
		rows.Entries = append(rows.Entries, &journalentry.JournalEntry{
			ID:              j.EntryID,
			BusinessUnitID:  g.Refs.BuID,
			OrganizationID:  g.Refs.OrgID,
			BatchID:         j.BatchID,
			FiscalYearID:    j.period.FiscalYearID,
			FiscalPeriodID:  j.period.ID,
			EntryNumber:     entryNumbers[i],
			EntryDate:       date,
			EntryType:       journalentry.EntryTypeStandard,
			AccountingDate:  date,
			Status:          journalentry.StatusPosted,
			ReferenceNumber: j.Document,
			ReferenceType:   j.ReferenceType,
			ReferenceID:     j.SourceID.String(),
			Description:     j.Description,
			TotalDebit:      totalDebit,
			TotalCredit:     totalCredit,
			IsPosted:        true,
			PostedAt:        &date,
			PostedByID:      j.ActorID,
			IsAutoGenerated: j.Auto,
			IsApproved:      true,
			ApprovedAt:      &date,
			ApprovedByID:    j.ActorID,
			CreatedByID:     j.ActorID,
			UpdatedByID:     j.ActorID,
			CreatedAt:       date,
			UpdatedAt:       date,
		})
		rows.JournalLines = append(rows.JournalLines, lines...)

		sourceID := volumegen.IDAt("jsrc_", date)
		rows.sources = append(rows.sources, &journalsource.Source{
			ID:                   sourceID,
			BusinessUnitID:       g.Refs.BuID,
			OrganizationID:       g.Refs.OrgID,
			SourceObjectType:     j.SourceType,
			SourceObjectID:       j.SourceID.String(),
			SourceEventType:      j.Event.String(),
			SourceDocumentNumber: j.Document,
			Status:               bilStatusPosted,
			IdempotencyKey:       j.Idempotency,
			JournalBatchID:       j.BatchID,
			JournalEntryID:       j.EntryID,
			CreatedAt:            date,
		})
		rows.srcLinks = append(rows.srcLinks, &journalsource.SourceJournalLink{
			ID:              sourceID,
			BusinessUnitID:  g.Refs.BuID,
			OrganizationID:  g.Refs.OrgID,
			JournalSourceID: sourceID,
			JournalBatchID:  j.BatchID,
			JournalEntryID:  j.EntryID,
			LinkRole:        "Primary",
			CreatedAt:       date,
		})

		for _, line := range lines {
			key := BalanceKey{
				account: line.GLAccountID,
				year:    j.period.FiscalYearID,
				period:  j.period.ID,
			}
			bal, ok := rows.Balances[key]
			if !ok {
				bal = &Balance{}
				rows.Balances[key] = bal
			}
			bal.debit += line.DebitAmount
			bal.credit += line.CreditAmount
			bal.net += line.NetAmount
			if date >= bal.lastDate {
				bal.lastDate = date
				bal.lastEntry = j.EntryID
			}
		}
	}

	return nil
}

func bilStampOf[T any](rows []T, fn func(T) volumegen.Stamp) []volumegen.Stamp {
	stamps := make([]volumegen.Stamp, len(rows))
	for i, row := range rows {
		stamps[i] = fn(row)
	}
	return stamps
}

func Insert[T any](
	ctx context.Context,
	tx *keel.DB,
	h keel.Handle[*T],
	rows []*T,
	fn func(*T) volumegen.Stamp,
	createdOnly bool,
) error {
	if len(rows) == 0 {
		return nil
	}
	stamps := bilStampOf(rows, fn)
	if err := volumegen.InsertRows(ctx, tx, h, rows); err != nil {
		return err
	}

	return volumegen.BackdateColumns(ctx, tx, h, stamps, createdOnly)
}

func (g *Ledger) Write(ctx context.Context, tx *keel.DB, rows *Rows) error {
	if err := Insert(ctx, tx, billingqueuedb.BillingQueueItems.Handle, rows.Items,
		func(r *billingqueue.BillingQueueItem) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, invoicedb.Invoices.Handle, rows.Invoices,
		func(r *invoice.Invoice) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, invoicedb.InvoiceLines.Handle, rows.Lines,
		func(r *invoice.InvoiceLine) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := g.bilLinkInvoices(ctx, tx, rows.Links); err != nil {
		return err
	}
	if err := Insert(ctx, tx, shipmentdb.ShipmentComments.Handle, rows.Comments,
		func(r *shipment.ShipmentComment) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.CreatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, billingqueueeventdb.ItemEvents.Handle, rows.Events,
		func(r *billingqueueevent.ItemEvent) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.At}
		}, true); err != nil {
		return err
	}
	if err := Insert(ctx, tx, customerpaymentdb.CreditMemoApplications.Handle, rows.MemoApps,
		func(r *customerpayment.CreditMemoApplication) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, journalentrydb.JournalBatches.Handle, rows.Batches,
		func(r *journalentry.JournalBatch) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, journalentrydb.JournalEntries.Handle, rows.Entries,
		func(r *journalentry.JournalEntry) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, journalentrydb.JournalEntryLines.Handle, rows.JournalLines,
		func(r *journalentry.JournalEntryLine) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := volumegen.InsertRows(ctx, tx, journalsourcedb.Sources.Handle, rows.sources); err != nil {
		return fmt.Errorf("insert journal_sources: %w", err)
	}
	if err := volumegen.InsertRows(ctx, tx, journalsourcedb.SourceJournalLinks.Handle, rows.srcLinks); err != nil {
		return fmt.Errorf("insert source_journal_links: %w", err)
	}
	if err := Insert(ctx, tx, customerpaymentdb.Payments.Handle, rows.Payments,
		func(r *customerpayment.Payment) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, customerpaymentdb.Applications.Handle, rows.Apps,
		func(r *customerpayment.Application) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt, Updated: r.UpdatedAt}
		}, false); err != nil {
		return err
	}
	if err := Insert(ctx, tx, customerledgerdb.CustomerLedgerEntries.Handle, rows.Ledger,
		func(r *customerledger.CustomerLedgerEntry) volumegen.Stamp {
			return volumegen.Stamp{Id: r.ID, Created: r.CreatedAt}
		}, true); err != nil {
		return err
	}

	return g.bilApplyBalances(ctx, tx, rows.Balances)
}

func (g *Ledger) bilLinkInvoices(
	ctx context.Context,
	tx *keel.DB,
	links []ItemInvoice,
) error {
	for begin := 0; begin < len(links); begin += volumegen.BackdateChunk {
		chunk := links[begin:min(begin+volumegen.BackdateChunk, len(links))]
		items := make([]string, len(chunk))
		invoices := make([]string, len(chunk))
		for i, link := range chunk {
			items[i] = link.Item.String()
			invoices[i] = link.Invoice.String()
		}
		bq := billingqueuedb.BillingQueueItems
		d := keel.Unnest("d", []string{"id", "invoice_id"}, items, invoices)
		if _, err := bq.With(d).Join(d, d.Col("id").EqCol(bq.ID)).Where(keel.True()).
			Update(ctx, tx, bq.InvoiceID.ToExpr(d.Col("invoice_id")), volumegen.KeepUpdated(bq.Handle)); err != nil {
			return fmt.Errorf("link billing queue items to invoices: %w", err)
		}
	}

	return nil
}

func (g *Ledger) bilApplyBalances(
	ctx context.Context,
	tx *keel.DB,
	balances map[BalanceKey]*Balance,
) error {
	keys := make([]BalanceKey, 0, len(balances))
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
			OrganizationID:     g.Refs.OrgID,
			BusinessUnitID:     g.Refs.BuID,
			GLAccountID:        key.account,
			FiscalYearID:       key.year,
			FiscalPeriodID:     key.period,
			PeriodDebitMinor:   bal.debit,
			PeriodCreditMinor:  bal.credit,
			NetChangeMinor:     bal.net,
			LastJournalEntryID: bal.lastEntry,
		})
	}

	return volumegen.PostBalances(ctx, tx, rows)
}

type Ledger struct {
	*volumegen.Generator
}
