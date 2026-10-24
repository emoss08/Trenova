package volume

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	payMinute   = int64(60)
	payHour     = int64(3_600)
	payDay      = int64(86_400)
	payCurrency = "USD"

	payProfileCompany       = "Company Driver Per-Mile"
	payProfileOwnerRevenue  = "Owner Operator 75% Revenue"
	payProfileRegional      = "Regional Guaranteed Mixed"
	payProfileLocal         = "Local Flat Plus Stops"
	payProfileTeam          = "Team Split Per-Mile"
	payProfileOwnerLinehaul = "Owner Operator 70% Linehaul"

	payManualAdjustment = "A manual adjustment was added to this settlement"
)

var (
	payMinorFactor = decimal.NewFromInt(100)
	payHundred     = decimal.NewFromInt(100)
	payHalf        = decimal.NewFromInt(50)
)

type payPeriod struct {
	start   int64
	end     int64
	payDate int64
}

type payAccounts struct {
	companyExpense pulid.ID
	ownerExpense   pulid.ID
	reimbursement  pulid.ID
	payable        pulid.ID
	advance        pulid.ID
	escrow         pulid.ID
	cash           pulid.ID
}

type paySetup struct {
	control        *tenant.SettlementControl
	allowDisputes  bool
	requireReceipt bool
	accounts       payAccounts
	codeAccounts   map[pulid.ID]pulid.ID
	codes          map[string]*driverpay.PayCode
	profiles       map[string]*driverpay.PayProfile
	bil            *bilSetup
	journals       bool
}

type payAdvancePlan struct {
	advance    *driverpay.PayAdvance
	writeOffAt int64
}

type payExpensePlan struct {
	expense  *driverpay.Expense
	outcome  driverpay.ExpenseStatus
	attached bool
}

type payDisputePlan struct {
	dispute  *driversettlement.Dispute
	raisedAt int64
}

type paySettlement struct {
	entity      *driversettlement.Settlement
	worker      *payWorker
	generatedAt int64
	updatedAt   int64
	escrowTxs   []*driverpay.EscrowTransaction
	lineTimes   map[*driversettlement.SettlementLine]int64
}

type payWorker struct {
	driver        *driverRecord
	team          bool
	assignments   []*driverpay.WorkerPayAssignment
	deductions    []*driverpay.RecurringDeduction
	earnings      []*driverpay.RecurringEarning
	advances      []*payAdvancePlan
	escrow        *driverpay.EscrowAccount
	events        []*driversettlement.PayEvent
	expenses      []*payExpensePlan
	disputes      []*payDisputePlan
	settlements   []*paySettlement
	nets          []int64
	carryOut      int64
	lastPeriodEnd int64
}

type payRows struct {
	assignments []*driverpay.WorkerPayAssignment
	deductions  []*driverpay.RecurringDeduction
	earnings    []*driverpay.RecurringEarning
	advances    []*driverpay.PayAdvance
	escrows     []*driverpay.EscrowAccount
	escrowTxs   []*driverpay.EscrowTransaction
	events      []*driversettlement.PayEvent
	settlements []*driversettlement.Settlement
	lines       []*driversettlement.SettlementLine
	batches     []*driversettlement.SettlementBatch
	expenses    []*driverpay.Expense
	disputes    []*driversettlement.Dispute
	stamps      map[string][]stamp
}

func (r *payRows) stamp(table string, id pulid.ID, created, updated int64) {
	r.stamps[table] = append(r.stamps[table], stamp{id: id, created: created, updated: updated})
}

func (g *generator) seedPayroll(ctx context.Context) error {
	if len(g.drivers) == 0 {
		return nil
	}

	setup, err := g.payLoadSetup(ctx)
	if err != nil {
		return err
	}

	workers := g.payBuildWorkers(setup)
	g.payAccrueEvents(setup, workers)
	g.payPlanExpenses(setup, workers)

	periods := g.payPeriods(setup.control)
	for _, w := range g.payOrderedWorkers(workers) {
		g.paySettleWorker(setup, w, periods)
	}

	rows := &payRows{stamps: make(map[string][]stamp)}
	if err = g.payCollect(ctx, workers, periods, rows); err != nil {
		return err
	}

	journals := &bilRows{balances: make(map[bilBalanceKey]*bilBalance)}
	if err = g.payJournals(setup, rows, journals); err != nil {
		return err
	}
	if err = g.bilMaterialize(ctx, journals); err != nil {
		return err
	}

	if err = g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		return g.payWrite(ctx, tx, rows, journals)
	}); err != nil {
		return err
	}

	g.report.add("Worker pay assignments", len(rows.assignments))
	g.report.add("Recurring deductions", len(rows.deductions))
	g.report.add("Recurring earnings", len(rows.earnings))
	g.report.add("Pay advances", len(rows.advances))
	g.report.add("Escrow accounts", len(rows.escrows))
	g.report.add("Escrow transactions", len(rows.escrowTxs))
	g.report.add("Pay events", len(rows.events))
	g.report.add("Settlement batches", len(rows.batches))
	g.report.add("Driver settlements", len(rows.settlements))
	g.report.add("Driver settlement lines", len(rows.lines))
	g.report.add("Driver expenses", len(rows.expenses))
	g.report.add("Settlement disputes", len(rows.disputes))
	g.report.add("Payroll journal entries", len(journals.entries))

	return nil
}

// -- Setup --

func (g *generator) payLoadSetup(ctx context.Context) (*paySetup, error) {
	setup := &paySetup{
		codeAccounts:  make(map[pulid.ID]pulid.ID),
		codes:         make(map[string]*driverpay.PayCode),
		profiles:      make(map[string]*driverpay.PayProfile),
		allowDisputes: true,
	}

	control, err := firstOrNil(ctx, g.db, tenantdb.SettlementControls.Query())
	switch {
	case err != nil:
		return nil, fmt.Errorf("load settlement control: %w", err)
	case control == nil:
		control = &tenant.SettlementControl{
			PayPeriodFrequency:            tenant.PayPeriodFrequencyWeekly,
			PeriodEndDayOfWeek:            6,
			PayDelayDays:                  5,
			VarianceThresholdPct:          decimal.NewFromInt(25),
			VarianceLookbackWeeks:         8,
			PayTrigger:                    tenant.PayTriggerShipmentDelivered,
			EscrowInterestFrequencyMonths: 3,
		}
	}
	setup.control = control

	dash, err := firstOrNil(ctx, g.db, tenantdb.DashControls.Query())
	if err != nil {
		return nil, fmt.Errorf("load dash control: %w", err)
	}
	if dash != nil {
		setup.allowDisputes = dash.AllowSettlementDisputes
		setup.requireReceipt = dash.RequireExpenseReceipt
	}

	if err = g.inTx(ctx, func(ctx context.Context, tx *keel.DB) error {
		if txErr := g.payEnsureCodes(ctx, tx, setup); txErr != nil {
			return txErr
		}
		return g.payEnsureProfiles(ctx, tx, setup)
	}); err != nil {
		return nil, err
	}

	if setup.bil, err = g.bilLoadSetup(ctx); err != nil {
		return nil, err
	}

	ac, err := g.accountingControl(ctx)
	if err != nil {
		return nil, err
	}
	if ac != nil {
		setup.accounts = payAccounts{
			companyExpense: ac.DefaultDriverPayExpenseAccountID,
			ownerExpense:   ac.DefaultPurchasedTransportationAccountID,
			reimbursement:  ac.DefaultDriverReimbursementAccountID,
			payable:        ac.DefaultSettlementsPayableAccountID,
			advance:        ac.DefaultDriverAdvanceAccountID,
			escrow:         ac.DefaultEscrowLiabilityAccountID,
			cash:           ac.DefaultCashAccountID,
		}
	}
	setup.journals = len(setup.bil.periods) > 0 && setup.accounts.payable.IsNotNil() &&
		(setup.accounts.companyExpense.IsNotNil() || setup.accounts.ownerExpense.IsNotNil())
	if !setup.journals {
		g.opts.Logf("  ! settlement posting accounts or open fiscal periods are missing; " +
			"driver settlements stop at Approved")
	}

	return setup, nil
}

var payCustomCodes = []struct {
	code string
	name string
}{
	{"TOLLS", "Tolls Reimbursement"},
	{"SCALE", "Scale Ticket Reimbursement"},
	{"LUMPERRMB", "Lumper Reimbursement"},
	{"PARKING", "Parking Reimbursement"},
	{"WASHOUT", "Trailer Washout Reimbursement"},
}

func (g *generator) payEnsureCodes(ctx context.Context, tx *keel.DB, setup *paySetup) error {
	at := unix(g.start) - 400*payDay
	rows := make([]*driverpay.PayCode, 0, len(driverpay.SystemPayCodes())+len(payCustomCodes))
	for _, def := range driverpay.SystemPayCodes() {
		rows = append(rows, &driverpay.PayCode{
			ID:                    idAt("payc_", at),
			BusinessUnitID:        g.refs.buID,
			OrganizationID:        g.refs.orgID,
			Status:                domaintypes.StatusActive,
			Direction:             def.Direction,
			Code:                  def.Code,
			Name:                  def.Name,
			Taxable:               def.Taxable,
			CountsTowardGuarantee: true,
			IsSystem:              true,
		})
	}
	for _, def := range payCustomCodes {
		rows = append(rows, &driverpay.PayCode{
			ID:             idAt("payc_", at),
			BusinessUnitID: g.refs.buID,
			OrganizationID: g.refs.orgID,
			Status:         domaintypes.StatusActive,
			Direction:      driverpay.PayCodeDirectionEarning,
			Code:           def.code,
			Name:           def.name,
			Description:    "Non-taxable reimbursement of a driver-paid trip expense",
		})
	}
	pc := driverpaydb.PayCodes
	inserted, err := pc.CreateOrSkip(ctx, tx, rows, pc.OrganizationID, pc.BusinessUnitID, pc.Direction, pc.Code)
	if err != nil {
		return fmt.Errorf("insert pay codes: %w", err)
	}
	if inserted > 0 {
		g.report.add("Pay codes", int(inserted))
	}

	codes, err := pc.Query().All(ctx, tx)
	if err != nil {
		return fmt.Errorf("load pay codes: %w", err)
	}
	for _, code := range codes {
		setup.codes[string(code.Direction)+"/"+code.Code] = code
		if code.GLAccountID != nil && code.GLAccountID.IsNotNil() {
			setup.codeAccounts[code.ID] = *code.GLAccountID
		}
	}

	stamps := make([]stamp, len(rows))
	for i, r := range rows {
		stamps[i] = stamp{id: r.ID, created: at, updated: at}
	}
	return backdate(ctx, tx, driverpaydb.PayCodes.Handle, stamps)
}

func payComponent(
	kind driverpay.ComponentKind,
	method driverpay.CalcMethod,
	description string,
	rate float64,
	sequence int,
) *driverpay.PayProfileComponent {
	return &driverpay.PayProfileComponent{
		Kind:        kind,
		Method:      method,
		Description: description,
		Rate:        decimal.NewFromFloat(rate),
		Sequence:    sequence,
		IsActive:    true,
	}
}

func payProfileDefs() []*driverpay.PayProfile {
	detention := func(rate float64, free int, seq int) *driverpay.PayProfileComponent {
		c := payComponent(driverpay.ComponentKindDetention, driverpay.CalcMethodPerHour,
			fmt.Sprintf("Detention after %d minutes free time", free), rate, seq)
		c.FreeTimeMinutes = free
		return c
	}
	banded := payComponent(driverpay.ComponentKindLinehaul, driverpay.CalcMethodPerLoadedMile,
		"Loaded miles (banded)", 0.6, 0)
	banded.Bands = []driverpay.MileageBand{
		{MinMiles: 0, MaxMiles: 200, Rate: decimal.NewFromFloat(0.65)},
		{MinMiles: 200, MaxMiles: 500, Rate: decimal.NewFromFloat(0.60)},
		{MinMiles: 500, MaxMiles: 0, Rate: decimal.NewFromFloat(0.56)},
	}
	ownerRevenue := payComponent(driverpay.ComponentKindLinehaul, driverpay.CalcMethodPercentOfRevenue,
		"75% of total revenue", 75, 0)
	ownerRevenue.RevenueBasis = driverpay.RevenueBasisTotalRevenue
	ownerLinehaul := payComponent(driverpay.ComponentKindLinehaul, driverpay.CalcMethodPercentOfRevenue,
		"70% of linehaul revenue", 70, 0)
	ownerLinehaul.RevenueBasis = driverpay.RevenueBasisLinehaul

	return []*driverpay.PayProfile{
		{
			Name:           payProfileCompany,
			Description:    "Banded per-loaded-mile pay with stop pay and detention for company drivers",
			Classification: driverpay.PayeeClassificationCompanyDriver,
			Components: []*driverpay.PayProfileComponent{
				banded,
				payComponent(driverpay.ComponentKindStopPay, driverpay.CalcMethodPerStop,
					"Extra stops beyond pickup and delivery", 25, 1),
				detention(22, 120, 2),
				payComponent(driverpay.ComponentKindHazmat, driverpay.CalcMethodPerEvent,
					"Hazmat load premium", 50, 3),
			},

			Status: domaintypes.StatusActive,

			CurrencyCode: "USD",
		},
		{
			Name:           payProfileOwnerRevenue,
			Description:    "75% of total revenue with stop pay for leased owner-operators",
			Classification: driverpay.PayeeClassificationOwnerOperator,
			Components: []*driverpay.PayProfileComponent{
				ownerRevenue,
				payComponent(driverpay.ComponentKindStopPay, driverpay.CalcMethodPerStop,
					"Extra stops beyond pickup and delivery", 30, 1),
			},

			Status: domaintypes.StatusActive,

			CurrencyCode: "USD",
		},
		{
			Name:                         payProfileRegional,
			Description:                  "Flat per-total-mile with a $900 guaranteed weekly minimum for regional drivers",
			Classification:               driverpay.PayeeClassificationCompanyDriver,
			GuaranteedPeriodMinimumMinor: 90_000,
			Components: []*driverpay.PayProfileComponent{
				payComponent(driverpay.ComponentKindLinehaul, driverpay.CalcMethodPerTotalMile,
					"All dispatched miles", 0.52, 0),
				payComponent(driverpay.ComponentKindStopPay, driverpay.CalcMethodPerStop,
					"Extra stops beyond pickup and delivery", 20, 1),
				payComponent(driverpay.ComponentKindLayover, driverpay.CalcMethodPerDay,
					"Layover per 24 hours away from home terminal", 75, 2),
			},

			Status: domaintypes.StatusActive,

			CurrencyCode: "USD",
		},
		{
			Name:           payProfileLocal,
			Description:    "Flat pay per load with stop pay and short free time for local drivers",
			Classification: driverpay.PayeeClassificationCompanyDriver,
			Components: []*driverpay.PayProfileComponent{
				payComponent(driverpay.ComponentKindLinehaul, driverpay.CalcMethodFlatPerShipment,
					"Flat pay per load", 115, 0),
				payComponent(driverpay.ComponentKindStopPay, driverpay.CalcMethodPerStop,
					"Extra stops beyond pickup and delivery", 30, 1),
				detention(20, 90, 2),
			},

			Status: domaintypes.StatusActive,

			CurrencyCode: "USD",
		},
		{
			Name:           payProfileTeam,
			Description:    "Team per-loaded-mile rate split evenly between both drivers",
			Classification: driverpay.PayeeClassificationCompanyDriver,
			Components: []*driverpay.PayProfileComponent{
				payComponent(driverpay.ComponentKindLinehaul, driverpay.CalcMethodPerLoadedMile,
					"Team loaded miles", 0.78, 0),
				payComponent(driverpay.ComponentKindStopPay, driverpay.CalcMethodPerStop,
					"Extra stops beyond pickup and delivery", 30, 1),
				detention(22, 120, 2),
				payComponent(driverpay.ComponentKindHazmat, driverpay.CalcMethodPerEvent,
					"Hazmat load premium", 60, 3),
			},

			Status: domaintypes.StatusActive,

			CurrencyCode: "USD",
		},
		{
			Name:           payProfileOwnerLinehaul,
			Description:    "70% of linehaul with stop pay and a hazmat premium for owner-operators",
			Classification: driverpay.PayeeClassificationOwnerOperator,
			Components: []*driverpay.PayProfileComponent{
				ownerLinehaul,
				payComponent(driverpay.ComponentKindStopPay, driverpay.CalcMethodPerStop,
					"Extra stops beyond pickup and delivery", 35, 1),
				payComponent(driverpay.ComponentKindHazmat, driverpay.CalcMethodPerEvent,
					"Hazmat load premium", 75, 2),
			},

			Status: domaintypes.StatusActive,

			CurrencyCode: "USD",
		},
	}
}

func (g *generator) payEnsureProfiles(ctx context.Context, tx *keel.DB, setup *paySetup) error {
	pp := driverpaydb.PayProfiles
	existing, err := pp.Load(pp.Components()).All(ctx, tx)
	if err != nil {
		return fmt.Errorf("load pay profiles: %w", err)
	}
	byName := make(map[string]*driverpay.PayProfile, len(existing))
	for _, p := range existing {
		byName[strings.ToLower(p.Name)] = p
	}

	at := unix(g.start) - 380*payDay
	var profiles []*driverpay.PayProfile
	var components []*driverpay.PayProfileComponent
	stamps := make([]stamp, 0)
	compStamps := make([]stamp, 0)
	for _, def := range payProfileDefs() {
		if p, ok := byName[strings.ToLower(def.Name)]; ok {
			setup.profiles[def.Name] = p
			continue
		}
		def.ID = idAt("dpp_", at)
		def.BusinessUnitID = g.refs.buID
		def.OrganizationID = g.refs.orgID
		def.Status = domaintypes.StatusActive
		def.CurrencyCode = payCurrency
		for _, c := range def.Components {
			c.ID = idAt("dppc_", at)
			c.BusinessUnitID = g.refs.buID
			c.OrganizationID = g.refs.orgID
			c.PayProfileID = def.ID
			components = append(components, c)
			compStamps = append(compStamps, stamp{id: c.ID, created: at, updated: at})
		}
		profiles = append(profiles, def)
		stamps = append(stamps, stamp{id: def.ID, created: at, updated: at})
		setup.profiles[def.Name] = def
	}
	if len(profiles) == 0 {
		return nil
	}

	inserting := make([]*driverpay.PayProfile, len(profiles))
	for i, p := range profiles {
		clone := *p
		clone.Components = nil
		inserting[i] = &clone
	}
	if err := insertRows(ctx, tx, driverpaydb.PayProfiles.Handle, inserting); err != nil {
		return fmt.Errorf("insert pay profiles: %w", err)
	}
	if err := insertRows(ctx, tx, driverpaydb.PayProfileComponents.Handle, components); err != nil {
		return fmt.Errorf("insert pay profile components: %w", err)
	}
	if err := backdate(ctx, tx, driverpaydb.PayProfiles.Handle, stamps); err != nil {
		return err
	}
	if err := backdate(ctx, tx, driverpaydb.PayProfileComponents.Handle, compStamps); err != nil {
		return err
	}
	g.report.add("Pay profiles", len(profiles))

	return nil
}

func (s *paySetup) code(direction driverpay.PayCodeDirection, code string) *driverpay.PayCode {
	return s.codes[string(direction)+"/"+code]
}

// -- Workers, assignments and recurring items --

func (g *generator) payBuildWorkers(setup *paySetup) map[pulid.ID]*payWorker {
	teams := make(map[pulid.ID]bool)
	for _, c := range g.crews {
		if c.SecondaryWorkerID != nil {
			teams[c.WorkerID] = true
			teams[*c.SecondaryWorkerID] = true
		}
	}

	workers := make(map[pulid.ID]*payWorker, len(g.drivers))
	for _, d := range g.drivers {
		w := &payWorker{driver: d, team: teams[d.WorkerID]}
		g.payAssign(setup, w)
		g.payRecurring(setup, w)
		g.payAdvances(w)
		workers[d.WorkerID] = w
	}
	return workers
}

func (g *generator) payProfileFor(w *payWorker) (string, decimal.Decimal) {
	owner := w.driver.Type == worker.WorkerTypeContractor
	switch {
	case w.team:
		return payProfileTeam, payHalf
	case owner && g.fake.chance(0.5):
		return payProfileOwnerRevenue, payHundred
	case owner:
		return payProfileOwnerLinehaul, payHundred
	case w.driver.DriverType == worker.DriverTypeLocal:
		return payProfileLocal, payHundred
	case w.driver.DriverType == worker.DriverTypeRegional:
		return payProfileRegional, payHundred
	default:
		return payProfileCompany, payHundred
	}
}

func (g *generator) payAssign(setup *paySetup, w *payWorker) {
	name, split := g.payProfileFor(w)
	profile := setup.profiles[name]
	if profile == nil {
		return
	}
	from := max(w.driver.HireDate, unix(g.start)-g.fake.between64(30, 400)*payDay)
	creator := pick(g.fake, g.refs.userIDs)

	if !w.team && profile.Classification == driverpay.PayeeClassificationCompanyDriver &&
		name != payProfileCompany && g.fake.chance(0.1) {
		if prior := setup.profiles[payProfileCompany]; prior != nil {
			switchAt := unix(g.start) + g.fake.between64(30, 200)*payDay
			if switchAt > from+payDay && switchAt < unix(g.now)-14*payDay {
				end := switchAt
				w.assignments = append(w.assignments, &driverpay.WorkerPayAssignment{
					ID:             idAt("wpa_", from),
					BusinessUnitID: g.refs.buID,
					OrganizationID: g.refs.orgID,
					WorkerID:       w.driver.WorkerID,
					PayProfileID:   prior.ID,
					EffectiveFrom:  from,
					EffectiveTo:    &end,
					SplitPercent:   payHundred,
					Notes:          "Moved to a new pay plan on transfer",
					CreatedByID:    creator,
					CreatedAt:      from,
					UpdatedAt:      switchAt,
				})
				from = switchAt
			}
		}
	}

	assignment := &driverpay.WorkerPayAssignment{
		ID:             idAt("wpa_", from),
		BusinessUnitID: g.refs.buID,
		OrganizationID: g.refs.orgID,
		WorkerID:       w.driver.WorkerID,
		PayProfileID:   profile.ID,
		EffectiveFrom:  from,
		SplitPercent:   split,
		CreatedByID:    creator,
		CreatedAt:      from,
		UpdatedAt:      from,
	}
	if w.team {
		assignment.Notes = "Team pay split evenly with co-driver"
	}
	w.assignments = append(w.assignments, assignment)
}

func (w *payWorker) assignmentAt(at int64) *driverpay.WorkerPayAssignment {
	for i := len(w.assignments) - 1; i >= 0; i-- {
		a := w.assignments[i]
		if a.EffectiveFrom <= at && (a.EffectiveTo == nil || *a.EffectiveTo > at) {
			return a
		}
	}
	return nil
}

func (w *payWorker) owner() bool {
	return w.driver.Type == worker.WorkerTypeContractor
}

func (g *generator) payDeduction(
	setup *paySetup,
	w *payWorker,
	code, description string,
	amountMinor int64,
	frequency driverpay.DeductionFrequency,
	start int64,
) *driverpay.RecurringDeduction {
	pc := setup.code(driverpay.PayCodeDirectionDeduction, code)
	if pc == nil {
		return nil
	}
	d := &driverpay.RecurringDeduction{
		ID:             idAt("rded_", start),
		BusinessUnitID: g.refs.buID,
		OrganizationID: g.refs.orgID,
		WorkerID:       w.driver.WorkerID,
		PayCodeID:      pc.ID,
		Status:         driverpay.DeductionStatusActive,
		Kind:           driverpay.DeductionKindStandard,
		Priority:       100,
		Frequency:      frequency,
		Description:    description,
		AmountMinor:    amountMinor,
		StartDate:      start,
		CurrencyCode:   payCurrency,
		CreatedByID:    pick(g.fake, g.refs.userIDs),
		CreatedAt:      start,
		UpdatedAt:      start,
	}
	w.deductions = append(w.deductions, d)
	return d
}

func (g *generator) payEarning(
	setup *paySetup,
	w *payWorker,
	code, description string,
	amountMinor int64,
	frequency driverpay.EarningFrequency,
	kind driverpay.EarningKind,
	start int64,
) {
	pc := setup.code(driverpay.PayCodeDirectionEarning, code)
	if pc == nil {
		return
	}
	w.earnings = append(w.earnings, &driverpay.RecurringEarning{
		ID:             idAt("rern_", start),
		BusinessUnitID: g.refs.buID,
		OrganizationID: g.refs.orgID,
		WorkerID:       w.driver.WorkerID,
		PayCodeID:      pc.ID,
		Status:         driverpay.EarningStatusActive,
		Kind:           kind,
		Frequency:      frequency,
		Description:    description,
		AmountMinor:    amountMinor,
		StartDate:      start,
		CurrencyCode:   payCurrency,
		CreatedByID:    pick(g.fake, g.refs.userIDs),
		CreatedAt:      start,
		UpdatedAt:      start,
	})
}

func (g *generator) payMinor(low, high int) int64 {
	return int64(g.fake.between(low, high)) * 100
}

//nolint:gocognit // each worker's recurring pay items are decided together
func (g *generator) payRecurring(setup *paySetup, w *payWorker) {
	if len(w.assignments) == 0 {
		return
	}
	start := w.assignments[0].EffectiveFrom
	f := g.fake

	if w.owner() {
		if w.driver.TractorID.IsNotNil() {
			lease := g.payDeduction(setup, w, "TRKLEASE", "Tractor lease-purchase payment",
				g.payMinor(550, 780), driverpay.DeductionFrequencyEverySettlement, start)
			if lease != nil && f.chance(0.4) {
				lease.TotalCapMinor = ptr(g.payMinor(38_000, 62_000))
				lease.DeductedToDateMinor = g.payMinor(4_000, 20_000)
			}
		}
		g.payDeduction(setup, w, "INSUR", "Occupational accident and bobtail insurance",
			g.payMinor(75, 115), driverpay.DeductionFrequencyEverySettlement, start)
		g.payDeduction(setup, w, "ELD", "ELD subscription",
			g.payMinor(35, 49), driverpay.DeductionFrequencyMonthly, start)
		if f.chance(0.3) {
			g.payDeduction(setup, w, "TRLLEASE", "Dry van trailer rental",
				g.payMinor(210, 260), driverpay.DeductionFrequencyEverySettlement, start)
		}
		if f.chance(0.85) {
			target := g.payMinor(2_500, 5_000)
			w.escrow = &driverpay.EscrowAccount{
				ID:                 idAt("escr_", start),
				BusinessUnitID:     g.refs.buID,
				OrganizationID:     g.refs.orgID,
				WorkerID:           w.driver.WorkerID,
				Status:             driverpay.EscrowAccountStatusActive,
				TargetAmountMinor:  target,
				AnnualInterestRate: setup.control.DefaultEscrowInterestRate,
				OpenedDate:         start,
				CurrencyCode:       payCurrency,
				CreatedAt:          start,
				UpdatedAt:          start,
			}
			escrow := g.payDeduction(setup, w, "ESCROW", "Maintenance escrow contribution",
				g.payMinor(75, 150), driverpay.DeductionFrequencyEverySettlement, start)
			if escrow != nil {
				escrow.EscrowAccountID = ptr(w.escrow.ID)
			}
		}
		if f.chance(0.2) {
			g.payEarning(setup, w, "EQUIPRENT", "APU rental for company use of driver-owned unit",
				g.payMinor(50, 75), driverpay.EarningFrequencyEverySettlement,
				driverpay.EarningKindStandard, start)
		}
		return
	}

	switch roll := f.intn(100); {
	case roll < 4:
		loanStart := unix(g.start) + f.between64(10, 240)*payDay
		loan := g.payDeduction(setup, w, "LOAN", "Company loan repayment",
			g.payMinor(25, 75), driverpay.DeductionFrequencyEverySettlement, loanStart)
		if loan != nil {
			loan.TotalCapMinor = ptr(g.payMinor(300, 1_200))
		}
	case roll < 6:
		garnish := g.payDeduction(setup, w, "OTHER", "Child support withholding order",
			g.payMinor(80, 220), driverpay.DeductionFrequencyEverySettlement, start)
		if garnish != nil {
			state := w.driver.Home.State
			garnish.Kind = driverpay.DeductionKindGarnishment
			garnish.Priority = 10
			garnish.CourtOrderNumber = "CO-" + f.digits(8)
			garnish.CaseNumber = state + "-" + f.digits(6)
			garnish.IssuingAgency = state + " Child Support Services"
		}
	case roll < 30:
		g.payDeduction(setup, w, "INSUR", "Voluntary supplemental insurance",
			g.payMinor(18, 45), driverpay.DeductionFrequencyEverySettlement, start)
	}

	if w.driver.DriverType == worker.DriverTypeOTR && f.chance(0.3) {
		g.payEarning(setup, w, "PERDIEM", "OTR per diem: IRS substantiated M&IE",
			g.payMinor(250, 340), driverpay.EarningFrequencyEverySettlement,
			driverpay.EarningKindStandard, start)
	}
	if f.chance(0.15) {
		g.payEarning(setup, w, "SAFETY", "Monthly safety bonus: clean inspections",
			g.payMinor(100, 150), driverpay.EarningFrequencyMonthly, driverpay.EarningKindBonus, start)
	}
	if unix(g.now)-w.driver.HireDate > 3*365*payDay && f.chance(0.4) {
		g.payEarning(setup, w, "LONGEVITY", "Longevity bonus: 3+ years of service",
			g.payMinor(75, 125), driverpay.EarningFrequencyMonthly, driverpay.EarningKindBonus, start)
	}
	if f.chance(0.1) {
		g.payEarning(setup, w, "STIPEND", "Cell phone and ELD data stipend",
			g.payMinor(40, 60), driverpay.EarningFrequencyMonthly, driverpay.EarningKindStandard, start)
	}

	for _, d := range w.deductions {
		if f.chance(0.05) {
			d.Status = driverpay.DeductionStatusPaused
		}
	}
}

var payAdvanceSources = []struct {
	source driverpay.AdvanceSource
	prefix string
}{
	{driverpay.AdvanceSourceEFSMoneyCode, "EFS-"},
	{driverpay.AdvanceSourceComdataCode, "CMD-"},
	{driverpay.AdvanceSourceFuelCard, "FC-"},
	{driverpay.AdvanceSourceCash, ""},
}

var payAdvanceNotes = []string{
	"Fuel and scale money issued en route",
	"Emergency tire replacement on the road",
	"Lumper cash advance at the receiver",
	"Layover meals and lodging",
	"Advance against next settlement",
}

func (g *generator) payAdvances(w *payWorker) {
	f := g.fake
	count := 0
	switch {
	case !w.driver.Active && f.chance(0.25):
		count = 1
	case w.driver.Active && f.chance(0.22):
		count = f.between(1, 4)
	}
	for range count {
		issued := unix(g.start) + f.between64(0, unix(g.now)-unix(g.start)-payHour)
		if len(w.assignments) > 0 {
			issued = max(issued, w.assignments[0].EffectiveFrom+payDay)
		}
		if issued >= unix(g.now) {
			continue
		}
		src := pick(f, payAdvanceSources)
		advance := &driverpay.PayAdvance{
			ID:             idAt("padv_", issued),
			BusinessUnitID: g.refs.buID,
			OrganizationID: g.refs.orgID,
			WorkerID:       w.driver.WorkerID,
			Status:         driverpay.AdvanceStatusOutstanding,
			Source:         src.source,
			IssuedDate:     issued,
			AmountMinor:    g.payMinor(100, 600),
			Notes:          pick(f, payAdvanceNotes),
			CurrencyCode:   payCurrency,
			CreatedByID:    pick(f, g.refs.userIDs),
			CreatedAt:      issued,
			UpdatedAt:      issued,
		}
		if src.prefix != "" {
			advance.Reference = src.prefix + f.digits(6)
		}
		plan := &payAdvancePlan{advance: advance}
		if !w.driver.Active {
			plan.writeOffAt = issued + f.between64(45, 90)*payDay
		}
		w.advances = append(w.advances, plan)
	}
	sort.SliceStable(w.advances, func(i, j int) bool {
		return w.advances[i].advance.IssuedDate < w.advances[j].advance.IssuedDate
	})
}

// -- Pay events --

func payToMinor(amount decimal.Decimal) int64 {
	return amount.Mul(payMinorFactor).Round(0).IntPart()
}

func payBandedRate(base decimal.Decimal, bands []driverpay.MileageBand, miles decimal.Decimal) decimal.Decimal {
	if len(bands) == 0 {
		return base
	}
	whole := int(miles.IntPart())
	for _, band := range bands {
		if whole >= band.MinMiles && (band.MaxMiles == 0 || whole < band.MaxMiles) {
			return band.Rate
		}
	}
	return base
}

type payMoveFacts struct {
	shipment   *shipmentRecord
	move       *moveRecord
	hazmat     bool
	detention  decimal.Decimal
	extraStops int
}

func payRevenueShare(facts *payMoveFacts) decimal.Decimal {
	if facts.shipment.Miles > 0 && facts.move.Miles > 0 {
		return decimal.NewFromFloat(facts.move.Miles).Div(decimal.NewFromFloat(facts.shipment.Miles))
	}
	if n := len(facts.shipment.Moves); n > 0 {
		return decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(n)))
	}
	return decimal.NewFromInt(1)
}

func payRevenueBasis(r *shipmentRecord, basis driverpay.RevenueBasis) decimal.Decimal {
	if basis == driverpay.RevenueBasisTotalRevenue {
		return r.Total
	}
	return r.Freight
}

// payComputeMove mirrors driversettlementservice.computeMovePay for the facts
// a generated move carries. Fuel surcharge pass-through pays nothing because
// generated fuel charges carry no fuel surcharge program, and layover per-day
// components accrue nothing at delivery, exactly as in the application.
//
//nolint:gocognit,cyclop // the method dispatch mirrors the application's single switch
func payComputeMove(
	profile *driverpay.PayProfile,
	split decimal.Decimal,
	facts *payMoveFacts,
) (components []driversettlement.PayEventComponent, gross int64) {
	distance := decimal.NewFromFloat(facts.move.Miles)
	splitFactor := split.Div(payHundred)
	for _, comp := range profile.Components {
		if comp == nil || !comp.IsActive {
			continue
		}
		var amount, quantity, rate decimal.Decimal
		switch comp.Method {
		case driverpay.CalcMethodPerLoadedMile, driverpay.CalcMethodPerTotalMile:
			if !distance.IsPositive() {
				continue
			}
			rate = payBandedRate(comp.Rate, comp.Bands, distance)
			amount, quantity = distance.Mul(rate), distance
		case driverpay.CalcMethodPerEmptyMile:
			continue
		case driverpay.CalcMethodPercentOfRevenue:
			if comp.Kind == driverpay.ComponentKindHazmat && !facts.hazmat {
				continue
			}
			basis := payRevenueBasis(facts.shipment, comp.RevenueBasis)
			if comp.Kind == driverpay.ComponentKindFuelSurcharge {
				basis = decimal.Zero
			}
			allocated := basis.Mul(payRevenueShare(facts))
			rate = comp.Rate
			amount, quantity = allocated.Mul(comp.Rate.Div(payHundred)), allocated
		case driverpay.CalcMethodFlatPerShipment:
			if comp.Kind == driverpay.ComponentKindHazmat && !facts.hazmat {
				continue
			}
			share := payRevenueShare(facts)
			rate = comp.Rate
			amount, quantity = comp.Rate.Mul(share), share
		case driverpay.CalcMethodPerStop:
			if facts.extraStops <= 0 {
				continue
			}
			quantity = decimal.NewFromInt(int64(facts.extraStops))
			rate = comp.Rate
			amount = rate.Mul(quantity)
		case driverpay.CalcMethodPerHour:
			if comp.Kind != driverpay.ComponentKindDetention || !facts.detention.IsPositive() {
				continue
			}
			quantity, rate = facts.detention, comp.Rate
			amount = rate.Mul(quantity)
		case driverpay.CalcMethodPerEvent:
			if comp.Kind != driverpay.ComponentKindHazmat || !facts.hazmat {
				continue
			}
			quantity, rate, amount = decimal.NewFromInt(1), comp.Rate, comp.Rate
		default:
			continue
		}
		if comp.MinAmountMinor != nil {
			if floor := decimal.NewFromInt(*comp.MinAmountMinor).Div(payMinorFactor); amount.LessThan(floor) {
				amount = floor
			}
		}
		if comp.MaxAmountMinor != nil {
			if ceiling := decimal.NewFromInt(*comp.MaxAmountMinor).Div(payMinorFactor); amount.GreaterThan(ceiling) {
				amount = ceiling
			}
		}
		amountMinor := payToMinor(amount.Mul(splitFactor))
		if amountMinor == 0 {
			continue
		}
		description := comp.Description
		if description == "" {
			description = string(comp.Kind)
		}
		components = append(components, driversettlement.PayEventComponent{
			Kind:        comp.Kind,
			Method:      comp.Method,
			Description: description,
			Quantity:    quantity.Round(4),
			Rate:        rate,
			AmountMinor: amountMinor,
		})
		gross += amountMinor
	}
	return components, gross
}

var payHoldReasons = []string{
	"Awaiting signed BOL before pay release",
	"Linehaul rate under review with the customer",
	"Detention claim needs the in-and-out times",
	"Damage claim open on this load",
}

func (g *generator) payAccrueEvents(setup *paySetup, workers map[pulid.ID]*payWorker) {
	now := unix(g.now)
	for _, r := range g.shipments {
		if r.DeliveredAt == 0 || r.Status == shipment.StatusCanceled {
			continue
		}
		facts := &payMoveFacts{shipment: r}
		var detention decimal.Decimal
		for _, c := range r.Charges {
			switch strings.ToUpper(c.Code) {
			case "HAZ":
				facts.hazmat = true
			case "DET":
				detention = decimal.NewFromInt(max(c.Unit, 1))
			}
		}
		for i, m := range r.Moves {
			if m.Crew == nil || m.Status == shipment.MoveStatusCanceled {
				continue
			}
			facts.move = m
			facts.extraStops = max(len(m.Cities)-2, 0)
			facts.detention = decimal.Zero
			if i == len(r.Moves)-1 {
				facts.detention = detention
			}
			workerIDs := []pulid.ID{m.Crew.WorkerID}
			if m.Crew.SecondaryWorkerID != nil {
				workerIDs = append(workerIDs, *m.Crew.SecondaryWorkerID)
			}
			for _, workerID := range workerIDs {
				w := workers[workerID]
				if w == nil {
					continue
				}
				g.payAccrue(setup, w, facts, now)
			}
		}
	}
	for _, w := range workers {
		sort.SliceStable(w.events, func(i, j int) bool { return w.events[i].EventDate < w.events[j].EventDate })
	}
}

func (g *generator) payAccrue(setup *paySetup, w *payWorker, facts *payMoveFacts, now int64) {
	r, m := facts.shipment, facts.move
	assignment := w.assignmentAt(r.DeliveredAt)
	if assignment == nil {
		return
	}
	profile := payProfileByID(setup, assignment.PayProfileID)
	if profile == nil {
		return
	}
	components, gross := payComputeMove(profile, assignment.SplitPercent, facts)
	created := min(r.DeliveredAt+g.fake.between64(payMinute, 20*payMinute), now)
	event := &driversettlement.PayEvent{
		ID:               idAt("dpe_", created),
		BusinessUnitID:   g.refs.buID,
		OrganizationID:   g.refs.orgID,
		WorkerID:         w.driver.WorkerID,
		ShipmentID:       r.ID,
		MoveID:           ptr(m.ID),
		AssignmentID:     ptr(assignment.ID),
		PayProfileID:     ptr(profile.ID),
		IdempotencyKey:   "pay:" + w.driver.WorkerID.String() + ":" + m.ID.String(),
		Status:           driversettlement.PayEventStatusAccrued,
		EventDate:        r.DeliveredAt,
		GrossAmountMinor: gross,
		TotalMiles:       decimal.NewFromFloat(m.Miles),
		CurrencyCode:     profile.CurrencyCode,
		Components:       components,
		ProNumber:        r.ProNumber,
		Version:          1,
		CreatedAt:        created,
		UpdatedAt:        created,
	}
	if event.CurrencyCode == "" {
		event.CurrencyCode = payCurrency
	}
	if g.fake.chance(0.015) {
		event.OnHold = true
		event.HoldReason = pick(g.fake, payHoldReasons)
	}
	w.events = append(w.events, event)
}

func payProfileByID(setup *paySetup, id pulid.ID) *driverpay.PayProfile {
	for _, p := range setup.profiles {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// -- Expenses --

var payExpenseKinds = []struct {
	code        string
	description string
	low, high   int
	weight      int
}{
	{"TOLLS", "Toll charges", 12, 95, 35},
	{"SCALE", "Certified scale ticket", 11, 14, 25},
	{"LUMPERRMB", "Lumper paid at the receiver", 120, 380, 15},
	{"PARKING", "Secure overnight parking", 15, 35, 15},
	{"WASHOUT", "Trailer washout after delivery", 45, 90, 10},
}

var payExpenseRejections = []string{
	"Receipt is illegible; please resubmit a clear photo",
	"Toll already paid through the company transponder",
	"Duplicate of an expense already reimbursed",
	"Lumper was paid by the company's Comdata check",
}

func (g *generator) payPlanExpenses(setup *paySetup, workers map[pulid.ID]*payWorker) {
	weights := make([]int, len(payExpenseKinds))
	for i, k := range payExpenseKinds {
		weights[i] = k.weight
	}
	now := unix(g.now)
	byShipment := make(map[pulid.ID]*shipmentRecord, len(g.shipments))
	for _, r := range g.shipments {
		byShipment[r.ID] = r
	}

	for _, w := range workers {
		for _, event := range w.events {
			if !g.fake.chance(0.12) {
				continue
			}
			kind := payExpenseKinds[g.fake.weighted(weights)]
			pc := setup.code(driverpay.PayCodeDirectionEarning, kind.code)
			if pc == nil {
				continue
			}
			incurred := event.EventDate - g.fake.between64(payHour, 30*payHour)
			submitted := min(incurred+g.fake.between64(payHour, 2*payDay), now-payMinute)
			if submitted <= incurred {
				continue
			}
			description := kind.description
			if r := byShipment[event.ShipmentID]; r != nil && len(r.Moves) > 0 {
				cities := r.Moves[len(r.Moves)-1].Cities
				if len(cities) > 0 {
					c := cities[len(cities)-1]
					description = fmt.Sprintf("%s: %s, %s (PRO %s)", kind.description, c.Name, c.State, r.ProNumber)
				}
			}
			expense := &driverpay.Expense{
				ID:                idAt("dexp_", submitted),
				BusinessUnitID:    g.refs.buID,
				OrganizationID:    g.refs.orgID,
				WorkerID:          w.driver.WorkerID,
				ShipmentID:        ptr(event.ShipmentID),
				PayCodeID:         ptr(pc.ID),
				Status:            driverpay.ExpenseStatusPending,
				AmountMinor:       g.payMinor(kind.low, kind.high),
				CurrencyCode:      payCurrency,
				Description:       description,
				IncurredDate:      incurred,
				SubmittedByUserID: pick(g.fake, g.refs.userIDs),
				Version:           1,
				CreatedAt:         submitted,
				UpdatedAt:         submitted,
			}
			plan := &payExpensePlan{expense: expense, outcome: driverpay.ExpenseStatusReimbursed}
			switch roll := g.fake.intn(100); {
			case setup.requireReceipt || roll < 10:
				plan.outcome = driverpay.ExpenseStatusRejected
			case roll < 14:
				plan.outcome = driverpay.ExpenseStatusCancelled
			}
			w.expenses = append(w.expenses, plan)
		}
	}
}

// -- Periods --

func payResolvePeriod(control *tenant.SettlementControl, now int64) payPeriod {
	at := time.Unix(now, 0).UTC()
	endDay := time.Weekday(control.PeriodEndDayOfWeek)
	daysBack := int(at.Weekday() - endDay)
	if daysBack < 0 {
		daysBack += 7
	}
	end := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysBack+1)
	return payPeriod{
		start:   payStepBack(control.PayPeriodFrequency, end).Unix(),
		end:     end.Unix(),
		payDate: end.AddDate(0, 0, control.PayDelayDays).Unix(),
	}
}

func payStepBack(frequency tenant.PayPeriodFrequency, end time.Time) time.Time {
	switch frequency {
	case tenant.PayPeriodFrequencyBiweekly:
		return end.AddDate(0, 0, -14)
	case tenant.PayPeriodFrequencyMonthly:
		return end.AddDate(0, -1, 0)
	default:
		return end.AddDate(0, 0, -7)
	}
}

func (g *generator) payPeriods(control *tenant.SettlementControl) []payPeriod {
	current := payResolvePeriod(control, unix(g.now))
	periods := []payPeriod{current}
	floor := unix(g.start) - 30*payDay
	for p := current; p.start > floor; {
		end := time.Unix(p.start, 0).UTC()
		p = payPeriod{
			start:   payStepBack(control.PayPeriodFrequency, end).Unix(),
			end:     end.Unix(),
			payDate: end.AddDate(0, 0, control.PayDelayDays).Unix(),
		}
		periods = append(periods, p)
	}
	sort.Slice(periods, func(i, j int) bool { return periods[i].end < periods[j].end })
	return periods
}

// -- Settlements --

func (g *generator) payOrderedWorkers(workers map[pulid.ID]*payWorker) []*payWorker {
	ordered := make([]*payWorker, 0, len(workers))
	for _, d := range g.drivers {
		if w := workers[d.WorkerID]; w != nil {
			ordered = append(ordered, w)
		}
	}
	return ordered
}

//nolint:gocognit,cyclop // one worker's pay history is simulated in order
func (g *generator) paySettleWorker(setup *paySetup, w *payWorker, periods []payPeriod) {
	now := unix(g.now)
	next := 0
	for _, period := range periods {
		var events []*driversettlement.PayEvent
		for next < len(w.events) && w.events[next].EventDate < period.end {
			next++
		}
		for _, e := range w.events[:next] {
			if e.Status == driversettlement.PayEventStatusAccrued && !e.OnHold {
				events = append(events, e)
			}
		}
		if len(events) == 0 {
			continue
		}
		generated := period.end + g.fake.between64(payHour, 10*payHour)
		if generated >= now-payMinute {
			continue
		}
		g.payWriteOffs(w, generated)
		s := g.payBuildSettlement(setup, w, period, events, generated)
		g.payAdvanceTimeline(setup, s, period)
		g.payAttachExpenses(w, s)
		g.payResolveDisputes(w, s)
		s.entity.SyncTotals()
		g.payApplySideEffects(s)
		g.payRaiseDispute(setup, w, s)

		w.nets = append(w.nets, s.entity.NetPayMinor)
		w.carryOut = s.entity.CarryForwardOutMinor
		w.lastPeriodEnd = period.end
		w.settlements = append(w.settlements, s)
	}

	g.payWriteOffs(w, now)
	g.payCloseExpenses(w)
	g.payCloseDisputes(w)
}

func (g *generator) payWriteOffs(w *payWorker, at int64) {
	for _, plan := range w.advances {
		a := plan.advance
		if plan.writeOffAt == 0 || plan.writeOffAt > at || a.WrittenOffAt != nil {
			continue
		}
		if outstanding := a.OutstandingMinor(); outstanding > 0 {
			writtenAt := plan.writeOffAt
			a.WrittenOffMinor = outstanding
			a.WriteOffReason = "Driver separated; balance uncollectible"
			a.WrittenOffAt = &writtenAt
			a.WrittenOffByID = pick(g.fake, g.refs.userIDs)
			a.UpdatedAt = writtenAt
			a.SyncStatus()
		}
	}
}

//nolint:funlen // the generated lines follow the application's order
func (g *generator) payBuildSettlement(
	setup *paySetup,
	w *payWorker,
	period payPeriod,
	events []*driversettlement.PayEvent,
	generated int64,
) *paySettlement {
	entity := &driversettlement.Settlement{
		ID:             idAt("dstl_", generated),
		BusinessUnitID: g.refs.buID,
		OrganizationID: g.refs.orgID,
		WorkerID:       w.driver.WorkerID,
		Status:         driversettlement.StatusDraft,
		Classification: driverpay.PayeeClassificationCompanyDriver,
		PeriodStart:    period.start,
		PeriodEnd:      period.end,
		PayDate:        period.payDate,
		CurrencyCode:   payCurrency,
		Version:        1,
	}
	s := &paySettlement{
		entity:      entity,
		worker:      w,
		generatedAt: generated,
		updatedAt:   generated,
		lineTimes:   make(map[*driversettlement.SettlementLine]int64),
	}

	var profile *driverpay.PayProfile
	if assignment := w.assignmentAt(period.end); assignment != nil {
		profile = payProfileByID(setup, assignment.PayProfileID)
	}
	if profile == nil {
		entity.AddException(driversettlement.ExceptionCodeMissingPayProfile,
			driversettlement.ExceptionSeverityCritical,
			"Worker has no effective pay profile assignment for this period")
	} else {
		entity.PayProfileID = ptr(profile.ID)
		entity.PayProfileName = profile.Name
		entity.Classification = profile.Classification
	}

	shipments := make(map[pulid.ID]struct{}, len(events))
	totalMiles := decimal.Zero
	lines := make([]*driversettlement.SettlementLine, 0, len(events)*2+8)
	for _, event := range events {
		event.Status = driversettlement.PayEventStatusSettled
		event.SettlementID = ptr(entity.ID)
		event.UpdatedAt = generated
		shipments[event.ShipmentID] = struct{}{}
		totalMiles = totalMiles.Add(event.TotalMiles)
		for _, comp := range event.Components {
			lines = append(lines, &driversettlement.SettlementLine{
				Category:      driversettlement.LineCategoryEarning,
				ComponentKind: comp.Kind,
				Method:        comp.Method,
				Description:   comp.Description,
				Quantity:      comp.Quantity,
				Rate:          comp.Rate,
				AmountMinor:   comp.AmountMinor,
				ShipmentID:    ptr(event.ShipmentID),
				MoveID:        event.MoveID,
				PayEventID:    ptr(event.ID),
				ProNumber:     event.ProNumber,
			})
		}
	}
	entity.ShipmentCount = len(shipments)
	entity.TotalMiles = totalMiles.Round(2)

	monthly := w.lastPeriodEnd == 0 ||
		time.Unix(w.lastPeriodEnd, 0).UTC().Month() != time.Unix(period.end, 0).UTC().Month() ||
		time.Unix(w.lastPeriodEnd, 0).UTC().Year() != time.Unix(period.end, 0).UTC().Year()

	var guaranteeExcluded int64
	for _, earning := range w.earnings {
		if earning.Status != driverpay.EarningStatusActive || earning.StartDate > period.end ||
			(earning.EndDate != nil && *earning.EndDate <= period.end) {
			continue
		}
		if earning.Frequency == driverpay.EarningFrequencyMonthly && !monthly {
			continue
		}
		amount := earning.NextAmountMinor()
		if amount <= 0 {
			continue
		}
		if remaining := earning.RemainingCapMinor(); remaining != nil && amount == *remaining {
			entity.AddException(driversettlement.ExceptionCodeEarningCapped,
				driversettlement.ExceptionSeverityWarning,
				"One or more recurring earnings reached their cap this period")
		}
		category := driversettlement.LineCategoryEarning
		code := payCodeByID(setup, earning.PayCodeID)
		if code != nil && code.LineIsReimbursement() {
			category = driversettlement.LineCategoryReimbursement
		}
		if category == driversettlement.LineCategoryEarning && code != nil && !code.CountsTowardGuarantee {
			guaranteeExcluded += amount
		}
		lines = append(lines, &driversettlement.SettlementLine{
			Category:           category,
			Description:        earning.Description,
			AmountMinor:        amount,
			RecurringEarningID: ptr(earning.ID),
			PayCodeID:          ptr(earning.PayCodeID),
		})
	}

	var grossSoFar int64
	for _, line := range lines {
		if line.Category == driversettlement.LineCategoryEarning {
			grossSoFar += line.AmountMinor
		}
	}
	if profile != nil && profile.GuaranteedPeriodMinimumMinor > 0 &&
		grossSoFar-guaranteeExcluded < profile.GuaranteedPeriodMinimumMinor {
		topUp := profile.GuaranteedPeriodMinimumMinor - (grossSoFar - guaranteeExcluded)
		lines = append(lines, &driversettlement.SettlementLine{
			Category:    driversettlement.LineCategoryGuaranteeTopUp,
			Description: "Guaranteed minimum pay top-up",
			AmountMinor: topUp,
		})
		entity.AddException(driversettlement.ExceptionCodeGuaranteeApplied,
			driversettlement.ExceptionSeverityWarning,
			"Guaranteed minimum applied: "+money.FormatMinor(topUp, payCurrency)+" top-up")
	}

	if w.carryOut < 0 {
		lines = append(lines, &driversettlement.SettlementLine{
			Category:    driversettlement.LineCategoryCarryForward,
			Description: "Negative balance carried forward from prior settlement",
			AmountMinor: w.carryOut,
		})
	}

	for _, deduction := range w.deductions {
		if deduction.Status != driverpay.DeductionStatusActive || deduction.StartDate > period.end ||
			(deduction.EndDate != nil && *deduction.EndDate <= period.end) {
			continue
		}
		if deduction.Frequency == driverpay.DeductionFrequencyMonthly && !monthly {
			continue
		}
		amount := deduction.NextAmountMinor()
		if amount <= 0 {
			continue
		}
		if remaining := deduction.RemainingCapMinor(); remaining != nil && amount == *remaining {
			entity.AddException(driversettlement.ExceptionCodeDeductionCapped,
				driversettlement.ExceptionSeverityWarning,
				"One or more deductions reached their cap this period")
		}
		category := driversettlement.LineCategoryDeduction
		var escrowID *pulid.ID
		if deduction.IsEscrowContribution() {
			category = driversettlement.LineCategoryEscrowContribution
			escrowID = deduction.EscrowAccountID
			if w.escrow != nil && w.escrow.TargetAmountMinor > 0 {
				remaining := w.escrow.TargetAmountMinor - w.escrow.BalanceMinor
				if remaining <= 0 {
					continue
				}
				amount = min(amount, remaining)
			}
		}
		lines = append(lines, &driversettlement.SettlementLine{
			Category:             category,
			Description:          deduction.Description,
			AmountMinor:          -amount,
			RecurringDeductionID: ptr(deduction.ID),
			EscrowAccountID:      escrowID,
			PayCodeID:            ptr(deduction.PayCodeID),
		})
	}

	lines = append(lines, g.payAdvanceLines(setup, w, generated, grossSoFar, lines)...)

	entity.Lines = lines
	entity.SyncTotals()
	if entity.CarryForwardOutMinor < 0 {
		entity.AddException(driversettlement.ExceptionCodeNegativeNet,
			driversettlement.ExceptionSeverityCritical,
			"Deductions exceed earnings; "+money.FormatMinor(-entity.CarryForwardOutMinor, payCurrency)+
				" carries forward")
	}
	g.payFlagVariance(setup, w, entity)

	return s
}

func payCodeByID(setup *paySetup, id pulid.ID) *driverpay.PayCode {
	for _, code := range setup.codes {
		if code.ID == id {
			return code
		}
	}
	return nil
}

func (g *generator) payAdvanceLines(
	setup *paySetup,
	w *payWorker,
	generated, gross int64,
	existing []*driversettlement.SettlementLine,
) []*driversettlement.SettlementLine {
	available := gross
	for _, line := range existing {
		switch line.Category {
		case driversettlement.LineCategoryEarning, driversettlement.LineCategoryAdvanceRecovery:
		default:
			available += line.AmountMinor
		}
	}

	var lines []*driversettlement.SettlementLine
	for _, plan := range w.advances {
		a := plan.advance
		if a.IssuedDate >= generated || a.WrittenOffAt != nil {
			continue
		}
		outstanding := a.OutstandingMinor()
		if outstanding <= 0 {
			continue
		}
		recovery := outstanding
		if !setup.control.AllowNegativeNet {
			if available <= 0 {
				break
			}
			recovery = min(recovery, available)
		}
		available -= recovery
		description := "Advance recovery"
		if a.Reference != "" {
			description += " (" + a.Reference + ")"
		}
		lines = append(lines, &driversettlement.SettlementLine{
			Category:    driversettlement.LineCategoryAdvanceRecovery,
			Description: description,
			AmountMinor: -recovery,
			AdvanceID:   ptr(a.ID),
		})
	}
	return lines
}

func (g *generator) payFlagVariance(setup *paySetup, w *payWorker, entity *driversettlement.Settlement) {
	threshold := setup.control.VarianceThresholdPct
	lookback := setup.control.VarianceLookbackWeeks
	if threshold.IsZero() || lookback <= 0 || len(w.nets) < 2 {
		return
	}
	trailing := w.nets[max(len(w.nets)-lookback, 0):]
	var sum int64
	for _, net := range trailing {
		sum += net
	}
	average := sum / int64(len(trailing))
	if average <= 0 {
		return
	}
	diff := entity.NetPayMinor - average
	if diff < 0 {
		diff = -diff
	}
	pct := decimal.NewFromInt(diff).Div(decimal.NewFromInt(average)).Mul(payHundred)
	if pct.GreaterThanOrEqual(threshold) {
		entity.AddException(driversettlement.ExceptionCodeHighVariance,
			driversettlement.ExceptionSeverityWarning,
			fmt.Sprintf("Net pay differs from the trailing average by %s%%", pct.Round(0).String()))
	}
}

// payAdvanceTimeline walks a settlement through review, approval, posting and
// payment on realistic delays and stops at whatever stage the present
// reaches. A settlement that cannot post (missing posting accounts) stops at
// Approved, as the application refuses to post it.
func (g *generator) payAdvanceTimeline(setup *paySetup, s *paySettlement, period payPeriod) {
	now := unix(g.now) - payMinute
	f := g.fake
	e := s.entity
	reviewer := pick(f, g.refs.userIDs)
	approver := pick(f, g.refs.userIDs)

	submitted := s.generatedAt + f.between64(2*payHour, 72*payHour)
	if submitted > now {
		return
	}
	if f.chance(0.03) {
		e.Notes = "Rejected: detention on the delivery stop was missing; corrected and resubmitted"
	}
	e.Status = driversettlement.StatusPendingApproval
	e.SubmittedByID = reviewer
	e.SubmittedAt = ptr(submitted)
	s.updatedAt = submitted

	approved := submitted + f.between64(payHour, 36*payHour)
	if approved > now {
		return
	}
	e.Status = driversettlement.StatusApproved
	e.ApprovedByID = approver
	e.ApprovedAt = ptr(approved)
	s.updatedAt = approved

	posted := approved + f.between64(30*payMinute, 24*payHour)
	if posted > now || !g.payPostable(setup, e) {
		return
	}
	e.Status = driversettlement.StatusPosted
	e.PostedByID = approver
	e.PostedAt = ptr(posted)
	e.PostedPayableAccountID = ptr(setup.accounts.payable)
	s.updatedAt = posted

	paid := max(period.payDate+f.between64(6*payHour, 14*payHour), posted+payHour)
	if paid > now || (e.NetPayMinor > 0 && setup.accounts.cash.IsNil()) {
		return
	}
	e.Status = driversettlement.StatusPaid
	e.PaidAt = ptr(paid)
	e.PaidByID = approver
	if f.chance(0.85) {
		e.PaymentMethod, e.PaymentReference = "ACH", "ACH-"+f.digits(8)
	} else {
		e.PaymentMethod, e.PaymentReference = "Check", "CHK "+f.digits(6)
	}
	s.updatedAt = paid
}

func (g *generator) payPostable(setup *paySetup, e *driversettlement.Settlement) bool {
	if !setup.journals {
		return false
	}
	expense := setup.accounts.companyExpense
	if e.Classification == driverpay.PayeeClassificationOwnerOperator {
		expense = setup.accounts.ownerExpense
	}
	if expense.IsNil() {
		return false
	}
	for _, line := range e.Lines {
		switch line.Category {
		case driversettlement.LineCategoryAdvanceRecovery, driversettlement.LineCategoryCarryForward:
			if setup.accounts.advance.IsNil() {
				return false
			}
		case driversettlement.LineCategoryEscrowContribution:
			if setup.accounts.escrow.IsNil() {
				return false
			}
		}
	}
	return e.CarryForwardOutMinor == 0 || setup.accounts.advance.IsNotNil()
}

// payDraftWindow is when an office user could still change the settlement:
// after it was generated and before it was submitted for approval.
func (g *generator) payDraftWindow(s *paySettlement) (from, to int64, ok bool) {
	from = s.generatedAt + 20*payMinute
	to = unix(g.now) - 5*payMinute
	if s.entity.SubmittedAt != nil {
		to = min(to, *s.entity.SubmittedAt-5*payMinute)
	}
	return from, to, to > from
}

func (g *generator) payAddAdjustment(
	s *paySettlement,
	description string,
	amountMinor int64,
	payCodeID *pulid.ID,
	at int64,
) *driversettlement.SettlementLine {
	line := &driversettlement.SettlementLine{
		ID:          idAt("dstll_", at),
		Category:    driversettlement.LineCategoryAdjustment,
		Description: description,
		AmountMinor: amountMinor,
		PayCodeID:   payCodeID,
	}
	s.entity.Lines = append(s.entity.Lines, line)
	s.lineTimes[line] = at
	s.entity.AddException(driversettlement.ExceptionCodeManualAdjustment,
		driversettlement.ExceptionSeverityWarning, payManualAdjustment)
	s.updatedAt = max(s.updatedAt, at)
	return line
}

func (g *generator) payAttachExpenses(w *payWorker, s *paySettlement) {
	from, to, ok := g.payDraftWindow(s)
	if !ok {
		return
	}
	for _, plan := range w.expenses {
		if plan.attached || plan.outcome != driverpay.ExpenseStatusReimbursed ||
			plan.expense.CreatedAt >= from {
			continue
		}
		e := plan.expense
		reviewed := from + g.fake.between64(0, to-from)
		line := g.payAddAdjustment(s, "Expense reimbursement: "+e.Description, e.AmountMinor, e.PayCodeID, reviewed)
		e.Status = driverpay.ExpenseStatusReimbursed
		e.ReviewedByID = ptr(pick(g.fake, g.refs.userIDs))
		e.ReviewedAt = ptr(reviewed)
		e.SettlementLineID = ptr(line.ID)
		e.UpdatedAt = reviewed
		plan.attached = true
	}
}

func (g *generator) payCloseExpenses(w *payWorker) {
	now := unix(g.now)
	for _, plan := range w.expenses {
		e := plan.expense
		if plan.attached {
			continue
		}
		switch {
		case plan.outcome == driverpay.ExpenseStatusCancelled:
			e.Status = driverpay.ExpenseStatusCancelled
			e.UpdatedAt = min(e.CreatedAt+g.fake.between64(payHour, 2*payDay), now)
		case plan.outcome == driverpay.ExpenseStatusRejected ||
			now-e.CreatedAt > 21*payDay:
			reviewed := e.CreatedAt + g.fake.between64(2*payHour, 4*payDay)
			if reviewed >= now {
				continue
			}
			e.Status = driverpay.ExpenseStatusRejected
			e.ReviewNote = pick(g.fake, payExpenseRejections)
			e.ReviewedByID = ptr(pick(g.fake, g.refs.userIDs))
			e.ReviewedAt = ptr(reviewed)
			e.UpdatedAt = reviewed
		}
	}
}

// -- Disputes --

var payDisputeTexts = map[driversettlement.DisputeCategory][]string{
	driversettlement.DisputeCategoryMissingPay: {
		"A load I delivered this week is not on my settlement",
		"Second stop on my relay leg was not paid",
	},
	driversettlement.DisputeCategoryIncorrectRate: {
		"Paid the wrong mileage band; the trip was over 500 miles",
		"My percentage was applied to linehaul only, not total revenue",
	},
	driversettlement.DisputeCategoryIncorrectDeduction: {
		"Insurance was deducted twice this period",
		"Escrow contribution taken after my account reached its target",
	},
	driversettlement.DisputeCategoryMissingReimbursement: {
		"My toll receipts from last week were not reimbursed",
		"Lumper I paid at the receiver is missing",
	},
	driversettlement.DisputeCategoryOther: {
		"Pay date on my statement does not match my bank deposit",
	},
}

var payDisputeCategories = []driversettlement.DisputeCategory{
	driversettlement.DisputeCategoryMissingPay,
	driversettlement.DisputeCategoryIncorrectRate,
	driversettlement.DisputeCategoryIncorrectDeduction,
	driversettlement.DisputeCategoryMissingReimbursement,
	driversettlement.DisputeCategoryOther,
}

func (g *generator) payRaiseDispute(setup *paySetup, w *payWorker, s *paySettlement) {
	e := s.entity
	if !setup.allowDisputes || !g.fake.chance(0.025) {
		return
	}
	var basis int64
	switch e.Status {
	case driversettlement.StatusPaid:
		basis = *e.PaidAt
	case driversettlement.StatusPosted:
		basis = *e.PostedAt
	case driversettlement.StatusApproved:
		basis = *e.ApprovedAt
	default:
		return
	}
	raised := basis + g.fake.between64(4*payHour, 6*payDay)
	if raised >= unix(g.now)-payHour {
		return
	}
	category := pick(g.fake, payDisputeCategories)
	dispute := &driversettlement.Dispute{
		ID:                idAt("dsd_", raised),
		BusinessUnitID:    g.refs.buID,
		OrganizationID:    g.refs.orgID,
		SettlementID:      e.ID,
		WorkerID:          w.driver.WorkerID,
		Status:            driversettlement.DisputeStatusOpen,
		Category:          category,
		Description:       pick(g.fake, payDisputeTexts[category]),
		SubmittedByUserID: pick(g.fake, g.refs.userIDs),
		Version:           1,
		CreatedAt:         raised,
		UpdatedAt:         raised,
	}
	var want driversettlement.LineCategory
	switch category {
	case driversettlement.DisputeCategoryIncorrectRate:
		want = driversettlement.LineCategoryEarning
	case driversettlement.DisputeCategoryIncorrectDeduction:
		want = driversettlement.LineCategoryDeduction
	}
	if want != "" {
		for _, line := range e.Lines {
			if line.Category == want {
				dispute.SettlementLineID = ptr(payLineID(line, s.generatedAt))
				break
			}
		}
	}
	w.disputes = append(w.disputes, &payDisputePlan{dispute: dispute, raisedAt: raised})
}

func payLineID(line *driversettlement.SettlementLine, at int64) pulid.ID {
	if line.ID.IsNil() {
		line.ID = idAt("dstll_", at)
	}
	return line.ID
}

var payDenials = []string{
	"Reviewed the trip; pay matches the profile in effect on the delivery date",
	"Deduction is correct per the signed lease agreement",
	"Receipt was submitted after the reimbursement window",
}

func (g *generator) payResolveDisputes(w *payWorker, s *paySettlement) {
	from, to, ok := g.payDraftWindow(s)
	for _, plan := range w.disputes {
		d := plan.dispute
		if d.Status.IsTerminal() || plan.raisedAt >= s.generatedAt {
			continue
		}
		resolver := pick(g.fake, g.refs.userIDs)
		switch roll := g.fake.intn(100); {
		case roll < 60 && ok:
			resolved := from + g.fake.between64(0, to-from)
			amount := g.payMinor(25, 180)
			if d.Category == driversettlement.DisputeCategoryIncorrectDeduction {
				amount = g.payMinor(18, 115)
			}
			line := g.payAddAdjustment(s, "Dispute correction: "+d.Description, amount, nil, resolved)
			d.Status = driversettlement.DisputeStatusResolved
			d.ResolutionNote = "Confirmed and corrected on the next settlement"
			d.ResolutionLineID = ptr(line.ID)
			d.ResolvedByID = ptr(resolver)
			d.ResolvedAt = ptr(resolved)
			d.UpdatedAt = resolved
		case roll < 85:
			resolved := plan.raisedAt + g.fake.between64(payDay, 5*payDay)
			if resolved >= unix(g.now) {
				continue
			}
			d.Status = driversettlement.DisputeStatusDenied
			d.ResolutionNote = pick(g.fake, payDenials)
			d.ResolvedByID = ptr(resolver)
			d.ResolvedAt = ptr(resolved)
			d.UpdatedAt = resolved
		case roll < 93:
			withdrawn := plan.raisedAt + g.fake.between64(2*payHour, 3*payDay)
			if withdrawn >= unix(g.now) {
				continue
			}
			d.Status = driversettlement.DisputeStatusWithdrawn
			d.ResolvedAt = ptr(withdrawn)
			d.UpdatedAt = withdrawn
		}
	}
}

func (g *generator) payCloseDisputes(w *payWorker) {
	now := unix(g.now)
	for _, plan := range w.disputes {
		d := plan.dispute
		if d.Status.IsTerminal() {
			continue
		}
		if now-plan.raisedAt > 2*payDay {
			d.Status = driversettlement.DisputeStatusInReview
			d.UpdatedAt = plan.raisedAt + g.fake.between64(payHour, 2*payDay)
		}
	}
}

// -- Approval side effects --

func (g *generator) payApplySideEffects(s *paySettlement) {
	e := s.entity
	if e.ApprovedAt == nil {
		return
	}
	at := *e.ApprovedAt
	w := s.worker
	for _, line := range e.Lines {
		amount := -line.AmountMinor
		switch line.Category {
		case driversettlement.LineCategoryDeduction, driversettlement.LineCategoryEscrowContribution:
			d := payDeductionByID(w, line.RecurringDeductionID)
			if d == nil {
				continue
			}
			d.DeductedToDateMinor += amount
			if remaining := d.RemainingCapMinor(); remaining != nil && *remaining == 0 {
				d.Status = driverpay.DeductionStatusCompleted
			}
			d.UpdatedAt = at
			if line.Category == driversettlement.LineCategoryEscrowContribution && w.escrow != nil &&
				line.EscrowAccountID != nil {
				w.escrow.BalanceMinor += amount
				w.escrow.UpdatedAt = at
				s.escrowTxs = append(s.escrowTxs, &driverpay.EscrowTransaction{
					ID:                idAt("esctx_", at),
					BusinessUnitID:    g.refs.buID,
					OrganizationID:    g.refs.orgID,
					EscrowAccountID:   w.escrow.ID,
					Type:              driverpay.EscrowTransactionTypeContribution,
					AmountMinor:       amount,
					BalanceAfterMinor: w.escrow.BalanceMinor,
					OccurredDate:      at,
					SettlementID:      ptr(e.ID),
					CreatedByID:       e.ApprovedByID,
					CreatedAt:         at,
				})
			}
		case driversettlement.LineCategoryAdvanceRecovery:
			for _, plan := range w.advances {
				if line.AdvanceID != nil && plan.advance.ID == *line.AdvanceID {
					plan.advance.RecoveredMinor += amount
					plan.advance.SyncStatus()
					plan.advance.UpdatedAt = at
				}
			}
		case driversettlement.LineCategoryEarning, driversettlement.LineCategoryReimbursement:
			if line.RecurringEarningID == nil {
				continue
			}
			for _, earning := range w.earnings {
				if earning.ID == *line.RecurringEarningID {
					earning.PaidToDateMinor += line.AmountMinor
					if remaining := earning.RemainingCapMinor(); remaining != nil && *remaining == 0 {
						earning.Status = driverpay.EarningStatusCompleted
					}
					earning.UpdatedAt = at
				}
			}
		}
	}
}

func payDeductionByID(w *payWorker, id *pulid.ID) *driverpay.RecurringDeduction {
	if id == nil {
		return nil
	}
	for _, d := range w.deductions {
		if d.ID == *id {
			return d
		}
	}
	return nil
}

// -- Collection, numbering and journals --

//nolint:funlen // every payroll table is gathered in one pass
func (g *generator) payCollect(
	ctx context.Context,
	workers map[pulid.ID]*payWorker,
	periods []payPeriod,
	rows *payRows,
) error {
	ordered := g.payOrderedWorkers(workers)
	var settlements []*paySettlement
	for _, w := range ordered {
		settlements = append(settlements, w.settlements...)
	}
	sort.SliceStable(settlements, func(i, j int) bool {
		return settlements[i].generatedAt < settlements[j].generatedAt
	})

	slots := make([]numberSlot, len(settlements))
	for i, s := range settlements {
		slots[i] = numberSlot{At: s.generatedAt}
	}
	numbers, err := g.numbers.allocate(ctx, tenant.SequenceTypeDriverSettlement, slots)
	if err != nil {
		return err
	}

	batches := make(map[int64]*driversettlement.SettlementBatch, len(periods))
	batchDone := make(map[int64]bool, len(periods))
	for i, s := range settlements {
		e := s.entity
		e.SettlementNumber = numbers[i]
		batch, ok := batches[e.PeriodEnd]
		if !ok {
			batch = &driversettlement.SettlementBatch{
				ID:             idAt("dstlb_", s.generatedAt),
				BusinessUnitID: g.refs.buID,
				OrganizationID: g.refs.orgID,
				Status:         driversettlement.BatchStatusOpen,
				Name: "Pay period ending " +
					time.Unix(e.PeriodEnd, 0).UTC().AddDate(0, 0, -1).Format("Jan 2, 2006"),
				PeriodStart:   e.PeriodStart,
				PeriodEnd:     e.PeriodEnd,
				PayDate:       e.PayDate,
				CurrencyCode:  payCurrency,
				GeneratedByID: g.refs.adminID,
				GeneratedAt:   ptr(s.generatedAt),
				Version:       1,
				CreatedAt:     s.generatedAt,
				UpdatedAt:     s.generatedAt,
			}
			batches[e.PeriodEnd] = batch
			batchDone[e.PeriodEnd] = true
		}
		e.BatchID = ptr(batch.ID)
		batch.SettlementCount++
		if e.HasExceptions {
			batch.ExceptionCount++
		}
		batch.TotalGrossMinor += e.GrossEarningsMinor
		batch.TotalNetMinor += e.NetPayMinor
		batch.UpdatedAt = max(batch.UpdatedAt, s.updatedAt)
		if e.Status != driversettlement.StatusPaid {
			batchDone[e.PeriodEnd] = false
		} else if batch.CompletedAt == nil || *batch.CompletedAt < *e.PaidAt {
			batch.CompletedAt = ptr(*e.PaidAt)
		}

		rows.settlements = append(rows.settlements, e)
		rows.stamp("driver_settlements", e.ID, s.generatedAt, s.updatedAt)
		for _, line := range e.Lines {
			at := s.generatedAt
			if t, ok := s.lineTimes[line]; ok {
				at = t
			}
			if line.ID.IsNil() {
				line.ID = idAt("dstll_", at)
			}
			line.BusinessUnitID = g.refs.buID
			line.OrganizationID = g.refs.orgID
			line.SettlementID = e.ID
			rows.lines = append(rows.lines, line)
			rows.stamp("driver_settlement_lines", line.ID, at, at)
		}
		for _, tx := range s.escrowTxs {
			tx.Description = "Escrow contribution from settlement " + e.SettlementNumber
			rows.escrowTxs = append(rows.escrowTxs, tx)
		}
	}
	periodEnds := make([]int64, 0, len(batches))
	for end := range batches {
		periodEnds = append(periodEnds, end)
	}
	sort.Slice(periodEnds, func(i, j int) bool { return periodEnds[i] < periodEnds[j] })
	for _, end := range periodEnds {
		batch := batches[end]
		if batchDone[end] {
			batch.Status = driversettlement.BatchStatusCompleted
			batch.UpdatedAt = max(batch.UpdatedAt, *batch.CompletedAt)
		} else {
			batch.CompletedAt = nil
		}
		rows.batches = append(rows.batches, batch)
		rows.stamp("driver_settlement_batches", batch.ID, batch.CreatedAt, batch.UpdatedAt)
	}

	for _, w := range ordered {
		for _, a := range w.assignments {
			rows.assignments = append(rows.assignments, a)
			rows.stamp("worker_pay_assignments", a.ID, a.CreatedAt, a.UpdatedAt)
		}
		for _, d := range w.deductions {
			rows.deductions = append(rows.deductions, d)
			rows.stamp("recurring_deductions", d.ID, d.CreatedAt, d.UpdatedAt)
		}
		for _, e := range w.earnings {
			rows.earnings = append(rows.earnings, e)
			rows.stamp("recurring_earnings", e.ID, e.CreatedAt, e.UpdatedAt)
		}
		for _, plan := range w.advances {
			a := plan.advance
			rows.advances = append(rows.advances, a)
			rows.stamp("pay_advances", a.ID, a.CreatedAt, a.UpdatedAt)
		}
		if w.escrow != nil {
			rows.escrows = append(rows.escrows, w.escrow)
			rows.stamp("escrow_accounts", w.escrow.ID, w.escrow.CreatedAt, w.escrow.UpdatedAt)
		}
		for _, event := range w.events {
			rows.events = append(rows.events, event)
			rows.stamp("driver_pay_events", event.ID, event.CreatedAt, event.UpdatedAt)
		}
		for _, plan := range w.expenses {
			e := plan.expense
			rows.expenses = append(rows.expenses, e)
			rows.stamp("driver_expenses", e.ID, e.CreatedAt, e.UpdatedAt)
		}
		for _, plan := range w.disputes {
			d := plan.dispute
			rows.disputes = append(rows.disputes, d)
			rows.stamp("settlement_disputes", d.ID, d.CreatedAt, d.UpdatedAt)
		}
	}

	return nil
}

func (g *generator) payJournals(setup *paySetup, rows *payRows, journals *bilRows) error {
	if !setup.journals {
		return nil
	}
	for _, e := range rows.settlements {
		if e.PostedAt == nil {
			continue
		}
		legs := payPostingLegs(e, setup)
		if len(legs) > 0 {
			journal, err := g.bilNewJournal(setup.bil, *e.PostedAt)
			if err != nil {
				return err
			}
			journal.actorID = e.PostedByID
			journal.description = "Driver settlement " + e.SettlementNumber
			journal.sourceType = "DriverSettlement"
			journal.sourceID = e.ID
			journal.document = e.SettlementNumber
			journal.event = tenant.JournalSourceEventDriverSettlementPosted
			journal.referenceType = journal.event.String()
			journal.idempotency = "driver-settlement-posted:" + e.ID.String()
			journal.auto = true
			journal.legs = legs
			journals.journals = append(journals.journals, journal)
			e.PostedJournalBatchID = ptr(journal.batchID)
		}

		if e.PaidAt == nil || e.NetPayMinor == 0 || e.PostedJournalBatchID == nil {
			continue
		}
		payment, err := g.bilNewJournal(setup.bil, *e.PaidAt)
		if err != nil {
			return err
		}
		payment.actorID = e.PaidByID
		payment.description = "Payment of driver settlement " + e.SettlementNumber
		payment.sourceType = "DriverSettlement"
		payment.sourceID = e.ID
		payment.document = e.SettlementNumber
		payment.event = tenant.JournalSourceEventDriverSettlementPaid
		payment.referenceType = payment.event.String()
		payment.idempotency = "driver-settlement-paid:" + e.ID.String()
		payment.auto = true
		payment.legs = []bilLeg{
			{account: setup.accounts.payable, debit: e.NetPayMinor},
			{account: setup.accounts.cash, credit: e.NetPayMinor},
		}
		journals.journals = append(journals.journals, payment)
		e.PaidJournalBatchID = ptr(payment.batchID)
	}
	return nil
}

// payPostingLegs mirrors driversettlementservice.BuildSettlementPostingLegs:
// earnings and top-ups debit expense (or the pay code's GL account), the net
// credits settlements payable, and recoveries, escrow and deductions credit
// their own accounts.
//
//nolint:gocognit,cyclop // one branch per settlement line category, as in the application
func payPostingLegs(e *driversettlement.Settlement, setup *paySetup) []bilLeg {
	accounts := setup.accounts
	expense := accounts.companyExpense
	if e.Classification == driverpay.PayeeClassificationOwnerOperator {
		expense = accounts.ownerExpense
	}
	reimbursement := accounts.reimbursement
	if reimbursement.IsNil() {
		reimbursement = expense
	}

	var expenseDebit, reimbursements, recovery, escrow, other int64
	codeDebits := make(map[pulid.ID]int64)
	codeCredits := make(map[pulid.ID]int64)
	mapped := func(line *driversettlement.SettlementLine) (pulid.ID, bool) {
		if line.PayCodeID == nil {
			return pulid.Nil, false
		}
		id, ok := setup.codeAccounts[*line.PayCodeID]
		return id, ok && id.IsNotNil()
	}
	for _, line := range e.Lines {
		switch line.Category {
		case driversettlement.LineCategoryEarning, driversettlement.LineCategoryGuaranteeTopUp:
			if id, ok := mapped(line); ok {
				codeDebits[id] += line.AmountMinor
			} else {
				expenseDebit += line.AmountMinor
			}
		case driversettlement.LineCategoryReimbursement:
			if id, ok := mapped(line); ok {
				codeDebits[id] += line.AmountMinor
			} else {
				reimbursements += line.AmountMinor
			}
		case driversettlement.LineCategoryAdvanceRecovery:
			recovery += -line.AmountMinor
		case driversettlement.LineCategoryEscrowContribution:
			escrow += -line.AmountMinor
		case driversettlement.LineCategoryDeduction:
			if id, ok := mapped(line); ok {
				codeCredits[id] += -line.AmountMinor
			} else {
				other += -line.AmountMinor
			}
		case driversettlement.LineCategoryAdjustment:
			id, ok := mapped(line)
			switch {
			case line.AmountMinor >= 0 && ok:
				codeDebits[id] += line.AmountMinor
			case line.AmountMinor >= 0:
				expenseDebit += line.AmountMinor
			case ok:
				codeCredits[id] += -line.AmountMinor
			default:
				other += -line.AmountMinor
			}
		case driversettlement.LineCategoryCarryForward:
		}
	}

	ids := make([]pulid.ID, 0, len(codeDebits)+len(codeCredits))
	for id := range codeDebits {
		ids = append(ids, id)
	}
	for id := range codeCredits {
		if _, ok := codeDebits[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	legs := make([]bilLeg, 0, 8+len(ids))
	for _, id := range ids {
		if debit := codeDebits[id]; debit > 0 {
			legs = append(legs, bilLeg{account: id, debit: debit})
		}
		if credit := codeCredits[id]; credit > 0 {
			legs = append(legs, bilLeg{account: id, credit: credit})
		}
	}
	add := func(account pulid.ID, debit, credit int64) {
		if debit > 0 || credit > 0 {
			legs = append(legs, bilLeg{account: account, debit: debit, credit: credit})
		}
	}
	add(expense, expenseDebit, 0)
	add(reimbursement, reimbursements, 0)
	add(accounts.advance, -e.CarryForwardOutMinor, 0)
	add(accounts.payable, 0, e.NetPayMinor)
	add(accounts.advance, 0, recovery)
	add(accounts.advance, 0, -e.CarryForwardInMinor)
	add(accounts.escrow, 0, escrow)
	add(expense, 0, other)
	return legs
}

// -- Write --

func (g *generator) payWrite(ctx context.Context, tx *keel.DB, rows *payRows, journals *bilRows) error {
	steps := []struct {
		table string
		run   func() error
	}{
		{"worker_pay_assignments", func() error { return insertRows(ctx, tx, driverpaydb.WorkerPayAssignments.Handle, rows.assignments) }},
		{"escrow_accounts", func() error { return insertRows(ctx, tx, driverpaydb.EscrowAccounts.Handle, rows.escrows) }},
		{"recurring_deductions", func() error { return insertRows(ctx, tx, driverpaydb.RecurringDeductions.Handle, rows.deductions) }},
		{"recurring_earnings", func() error { return insertRows(ctx, tx, driverpaydb.RecurringEarnings.Handle, rows.earnings) }},
		{"pay_advances", func() error { return insertRows(ctx, tx, driverpaydb.PayAdvances.Handle, rows.advances) }},
		{"driver_settlement_batches", func() error { return insertRows(ctx, tx, driversettlementdb.SettlementBatches.Handle, rows.batches) }},
		{"driver_settlements", func() error { return insertRows(ctx, tx, driversettlementdb.Settlements.Handle, rows.settlements) }},
		{"driver_settlement_lines", func() error { return insertRows(ctx, tx, driversettlementdb.SettlementLines.Handle, rows.lines) }},
		{"driver_pay_events", func() error { return insertRows(ctx, tx, driversettlementdb.PayEvents.Handle, rows.events) }},
		{"escrow_transactions", func() error { return insertRows(ctx, tx, driverpaydb.EscrowTransactions.Handle, rows.escrowTxs) }},
		{"driver_expenses", func() error { return insertRows(ctx, tx, driverpaydb.Expenses.Handle, rows.expenses) }},
		{"settlement_disputes", func() error { return insertRows(ctx, tx, driversettlementdb.Disputes.Handle, rows.disputes) }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("insert %s: %w", step.table, err)
		}
	}

	tables := make([]string, 0, len(rows.stamps))
	for table := range rows.stamps {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		if err := backdateTable(ctx, tx, table, rows.stamps[table]); err != nil {
			return err
		}
	}
	txStamps := make([]stamp, len(rows.escrowTxs))
	for i, t := range rows.escrowTxs {
		txStamps[i] = stamp{id: t.ID, created: t.CreatedAt}
	}
	if err := backdateTableCreated(ctx, tx, "escrow_transactions", txStamps); err != nil {
		return err
	}

	if len(journals.journals) == 0 {
		return nil
	}
	if err := g.bilWrite(ctx, tx, journals); err != nil {
		return errors.Join(errors.New("write payroll journals"), err)
	}
	return nil
}
