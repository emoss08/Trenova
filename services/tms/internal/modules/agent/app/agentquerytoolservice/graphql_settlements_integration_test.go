//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate/formulatemplatedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype/servicetypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype/shipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/carriersettlementrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driverpayrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/adapters/postgres/driversettlementrepository"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driverpayservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementreviewservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementworkspaceservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payadvance/payadvancedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paycode/paycodedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payescrow/payescrowdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payexpense"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payexpense/payexpensedb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/recurringpay/recurringpaydb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const settlementCurrency = "USD"

var errNoID = errors.New("the row was written without an id")

var (
	voided    = map[string]any{"status": []any{"Voided"}}
	closed    = map[string]any{"status": []any{"Closed"}}
	writtenOf = map[string]any{"status": []any{"WrittenOff"}}
	rejected  = map[string]any{"status": []any{"Rejected"}}
)

// settlementRows writes the records a settlement list reads straight
// through admin, which row-level security does not hold, as SeedTenant
// writes the tenant: each row's worker, and one shipment per tenant for the
// pay events to be earned on.
type settlementRows struct {
	admin *keel.DB

	mu        sync.Mutex
	shipments map[pulid.ID]pulid.ID
}

type creator[T any] interface {
	Create(ctx context.Context, db *keel.DB, row *T) error
}

func (s *settlementRows) worker(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
) (pulid.ID, error) {
	row := testWorker(tenant, "S"+label, false, pulid.Nil)
	row.Profile = nil
	if err := workerdb.Workers.Create(ctx, s.admin, row); err != nil {
		return pulid.Nil, err
	}

	return row.ID, nil
}

// shipment is the tenant's one shipment a pay event is earned on, written
// with what it references the first time a tenant needs it.
func (s *settlementRows) shipment(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
) (pulid.ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	org, bu := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID
	if id, ok := s.shipments[org]; ok {
		return id, nil
	}
	service := &servicetype.ServiceType{
		BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive,
		Code: "PAY", Color: "#112233",
	}
	kind := &shipmenttype.ShipmentType{
		BusinessUnitID: bu, OrganizationID: org, Status: domaintypes.StatusActive,
		Code: "PAY", Color: "#334455",
	}
	buyer := &customer.Customer{
		BusinessUnitID: bu, OrganizationID: org, StateID: tenant.StateID,
		Status: domaintypes.StatusActive, Code: "PAYCUST", Name: "Pay Customer",
		AddressLine1: "1 Main", City: "Dallas", PostalCode: "75001",
	}
	tariff := &formulatemplate.FormulaTemplate{
		BusinessUnitID: bu, OrganizationID: org, Name: "Pay Template",
		Source: "tariff \"Pay Template\"\n\ncharge \"Freight\" = $1\n",
		Status: formulatemplate.StatusActive,
	}
	if err := servicetypedb.ServiceTypes.Create(ctx, s.admin, service); err != nil {
		return pulid.Nil, err
	}
	if err := shipmenttypedb.ShipmentTypes.Create(ctx, s.admin, kind); err != nil {
		return pulid.Nil, err
	}
	if err := customerdb.Customers.Create(ctx, s.admin, buyer); err != nil {
		return pulid.Nil, err
	}
	if err := formulatemplatedb.FormulaTemplates.Create(ctx, s.admin, tariff); err != nil {
		return pulid.Nil, err
	}
	load := &shipment.Shipment{
		ID: pulid.MustNew("shp_"), BusinessUnitID: bu, OrganizationID: org,
		ServiceTypeID: service.ID, ShipmentTypeID: kind.ID, CustomerID: buyer.ID,
		FormulaTemplateID: tariff.ID, Status: shipment.StatusNew,
		ProNumber: "PRO" + org.String()[4:12], BOL: "BOL" + org.String()[4:12],
	}
	if err := shipmentdb.Shipments.Create(ctx, s.admin, load); err != nil {
		return pulid.Nil, err
	}
	s.shipments[org] = load.ID

	return load.ID, nil
}

func (s *settlementRows) payCode(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	direction paycode.PayCodeDirection,
) (pulid.ID, error) {
	row := &paycode.PayCode{
		BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
		Status: "Active", Direction: direction, Code: "PC" + label, Name: "Pay code " + label,
	}
	if err := paycodedb.PayCodes.Create(ctx, s.admin, row); err != nil {
		return pulid.Nil, err
	}

	return row.ID, nil
}

// settlementServices are the resolver dependencies the settlement reads
// reach, over the RLS connection.
func settlementServices(conn *postgres.Connection, services *agentread.ResolverParams) {
	logger := zap.NewNop()
	pay := driverpayrepository.Params{DB: conn, Logger: logger}
	stl := driversettlementrepository.Params{DB: conn, Logger: logger}
	settlements := driversettlementrepository.NewSettlement(stl)
	disputes := driversettlementrepository.NewDispute(stl)
	events := driversettlementrepository.NewPayEvent(stl)
	batches := driversettlementrepository.NewBatch(stl)
	expenses := driverpayrepository.NewDriverExpense(pay)
	assignments := driverpayrepository.NewWorkerPayAssignment(pay)
	codes := driverpayrepository.NewPayCode(pay)
	deductions := driverpayrepository.NewRecurringDeduction(pay)
	earnings := driverpayrepository.NewRecurringEarning(pay)
	advances := driverpayrepository.NewPayAdvance(pay)
	escrow := driverpayrepository.NewEscrowAccount(pay)
	profiles := driverpayrepository.NewPayProfile(pay)
	carrierSettlements := carriersettlementrepository.NewSettlement(
		carriersettlementrepository.Params{DB: conn, Logger: logger})

	payService := driverpayservice.New(driverpayservice.Params{
		Logger: logger, DB: conn, ProfileRepo: profiles, AssignmentRepo: assignments,
		DeductionRepo: deductions, PayCodeRepo: codes, EarningRepo: earnings,
		AdvanceRepo: advances, EscrowRepo: escrow,
	})
	services.DriverSettlementRepo = settlements
	services.PayEventRepo = events
	services.SettlementBatchRepo = batches
	services.PayAdvanceRepo = advances
	services.EscrowAccountRepo = escrow
	services.RecurringDeductionRepo = deductions
	services.RecurringEarningRepo = earnings
	services.PayProfileRepo = profiles
	services.CarrierSettlementRepo = carrierSettlements
	services.DriverPayService = payService
	services.DriverSettlementService = driversettlementservice.New(driversettlementservice.Params{
		Logger: logger, DB: conn, SettlementRepo: settlements, BatchRepo: batches,
		PayEventRepo: events, AssignmentRepo: assignments, DeductionRepo: deductions,
		EarningRepo: earnings, PayCodeRepo: codes, AdvanceRepo: advances, EscrowRepo: escrow,
	})
	services.DriverSettlementReviewService = driversettlementreviewservice.New(
		driversettlementreviewservice.Params{
			Logger: logger, Settlements: services.DriverSettlementService,
			SettlementRepo: settlements, DisputeRepo: disputes, ExpenseRepo: expenses,
			PayCodeRepo: codes,
		},
	)
	services.DriverSettlementWorkspaceService = driversettlementworkspaceservice.New(
		driversettlementworkspaceservice.Params{
			Settlements: services.DriverSettlementService, SettlementRepo: settlements,
			BatchRepo: batches, PayEventRepo: events,
		},
	)
	services.CarrierSettlementService = carriersettlementservice.New(
		carriersettlementservice.Params{
			Logger: logger, DB: conn, SettlementRepo: carrierSettlements,
		},
	)
}

// settlementListCases are the settlement lists, each written as its screen
// would list it; the marked rows are the ones the typed status filter names.
//
//nolint:funlen // one entry per converted list
func settlementListCases(
	conn *postgres.Connection,
	admin *keel.DB,
	services *agentread.ResolverParams,
) []referenceCase {
	settlementServices(conn, services)
	rows := &settlementRows{admin: admin, shipments: make(map[pulid.ID]pulid.ID)}
	now := timeutils.NowUnix()
	day := int64(timeutils.SecondsPerDay)

	return []referenceCase{
		{
			tool: "settlement_list_driver", marked: voided,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return rows.driverSettlement(ctx, tenant, label, marked)
			},
		},
		{
			tool: "settlement_list_pay_events", marked: voided,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				workerID, err := rows.worker(ctx, tenant, "E"+label)
				if err != nil {
					return pulid.Nil, err
				}
				shipmentID, err := rows.shipment(ctx, tenant)
				if err != nil {
					return pulid.Nil, err
				}
				status := driversettlement.PayEventStatusAccrued
				if marked {
					status = driversettlement.PayEventStatusVoided
				}
				row := &driversettlement.PayEvent{
					BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
					WorkerID: workerID, ShipmentID: shipmentID,
					IdempotencyKey: "pe" + label + tenant.Actor.OrganizationID.String(),
					Status:         status, EventDate: now - day, GrossAmountMinor: 125_000,
					TotalMiles: decimal.NewFromInt(500), CurrencyCode: settlementCurrency,
					ProNumber: "PRO" + label,
				}
				return insertWithID(ctx, rows.admin, driversettlementdb.PayEvents, row, &row.ID)
			},
		},
		{
			tool: "settlement_list_carrier", marked: voided,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return rows.carrierSettlement(ctx, tenant, label, marked)
			},
		},
		{
			tool: "settlement_list_driver_expenses", marked: rejected,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				workerID, err := rows.worker(ctx, tenant, "X"+label)
				if err != nil {
					return pulid.Nil, err
				}
				status := payexpense.ExpenseStatusPending
				if marked {
					status = payexpense.ExpenseStatusRejected
				}
				row := &payexpense.Expense{
					BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
					WorkerID: workerID, Status: status, AmountMinor: 4_250,
					CurrencyCode: settlementCurrency, Description: "Lumper fee " + label,
					IncurredDate: now - day, SubmittedByUserID: tenant.Actor.UserID,
				}
				return insertWithID(ctx, rows.admin, payexpensedb.Expenses, row, &row.ID)
			},
		},
		{
			tool: "settlement_list_pay_profiles", marked: inactive,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				return rows.payProfile(ctx, tenant, label, marked)
			},
		},
		{
			tool: "settlement_list_escrow_accounts", marked: closed,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				workerID, err := rows.worker(ctx, tenant, "C"+label)
				if err != nil {
					return pulid.Nil, err
				}
				status, closedDate := payescrow.EscrowAccountStatusActive, (*int64)(nil)
				if marked {
					status, closedDate = payescrow.EscrowAccountStatusClosed, new(now)
				}
				row := &payescrow.EscrowAccount{
					BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
					WorkerID: workerID, Status: status, TargetAmountMinor: 250_000,
					BalanceMinor: 50_000, AnnualInterestRate: decimal.NewFromFloat(0.02),
					OpenedDate: now - 30*day, ClosedDate: closedDate, CurrencyCode: settlementCurrency,
				}
				return insertWithID(ctx, rows.admin, payescrowdb.EscrowAccounts, row, &row.ID)
			},
		},
		{
			tool: "settlement_list_advances", marked: writtenOf,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				workerID, err := rows.worker(ctx, tenant, "V"+label)
				if err != nil {
					return pulid.Nil, err
				}
				status := payadvance.AdvanceStatusOutstanding
				if marked {
					status = payadvance.AdvanceStatusWrittenOff
				}
				row := &payadvance.PayAdvance{
					BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
					WorkerID: workerID, Status: status, Source: payadvance.AdvanceSourceCash,
					Reference: "ADV" + label, IssuedDate: now - day, AmountMinor: 20_000,
					CurrencyCode: settlementCurrency, CreatedByID: tenant.Actor.UserID,
				}
				return insertWithID(ctx, rows.admin, payadvancedb.PayAdvances, row, &row.ID)
			},
		},
		{
			tool: "settlement_list_recurring_deductions", marked: paused,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				workerID, err := rows.worker(ctx, tenant, "D"+label)
				if err != nil {
					return pulid.Nil, err
				}
				code, err := rows.payCode(
					ctx,
					tenant,
					"D"+label,
					paycode.PayCodeDirectionDeduction,
				)
				if err != nil {
					return pulid.Nil, err
				}
				status := recurringpay.DeductionStatusActive
				if marked {
					status = recurringpay.DeductionStatusPaused
				}
				row := &recurringpay.RecurringDeduction{
					BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
					WorkerID: workerID, PayCodeID: code, Status: status,
					Kind: recurringpay.DeductionKindStandard, Frequency: recurringpay.DeductionFrequencyEverySettlement,
					Description: "Insurance " + label, AmountMinor: 7_500, StartDate: now - 30*day,
					CurrencyCode: settlementCurrency, CreatedByID: tenant.Actor.UserID,
				}
				return insertWithID(
					ctx,
					rows.admin,
					recurringpaydb.RecurringDeductions,
					row,
					&row.ID,
				)
			},
		},
		{
			tool: "settlement_list_recurring_earnings", marked: paused,
			write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, marked bool) (pulid.ID, error) {
				workerID, err := rows.worker(ctx, tenant, "R"+label)
				if err != nil {
					return pulid.Nil, err
				}
				code, err := rows.payCode(ctx, tenant, "R"+label, paycode.PayCodeDirectionEarning)
				if err != nil {
					return pulid.Nil, err
				}
				status := recurringpay.EarningStatusActive
				if marked {
					status = recurringpay.EarningStatusPaused
				}
				row := &recurringpay.RecurringEarning{
					BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
					WorkerID: workerID, PayCodeID: code, Status: status,
					Kind: recurringpay.EarningKindStandard, Frequency: recurringpay.EarningFrequencyEverySettlement,
					Description: "Per diem " + label, AmountMinor: 6_000, StartDate: now - 30*day,
					CurrencyCode: settlementCurrency, CreatedByID: tenant.Actor.UserID,
				}
				return insertWithID(ctx, rows.admin, recurringpaydb.RecurringEarnings, row, &row.ID)
			},
		},
	}
}

// insertWithID writes the row and reads back the id keel gave it.
func insertWithID[T any](
	ctx context.Context,
	admin *keel.DB,
	table creator[T],
	row *T,
	id *pulid.ID,
) (pulid.ID, error) {
	if err := table.Create(ctx, admin, row); err != nil {
		return pulid.Nil, err
	}
	if id.IsNil() {
		return pulid.Nil, errNoID
	}

	return *id, nil
}

func (s *settlementRows) driverSettlement(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	marked bool,
) (pulid.ID, error) {
	workerID, err := s.worker(ctx, tenant, "L"+label)
	if err != nil {
		return pulid.Nil, err
	}
	now := timeutils.NowUnix()
	day := int64(timeutils.SecondsPerDay)
	status := driversettlement.StatusDraft
	if marked {
		status = driversettlement.StatusVoided
	}
	row := &driversettlement.Settlement{
		BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
		WorkerID: workerID, SettlementNumber: "DS" + label, Status: status,
		Classification: driverpay.PayeeClassificationCompanyDriver,
		PeriodStart:    now - 14*day, PeriodEnd: now - 7*day, PayDate: now,
		GrossEarningsMinor: 250_000, DeductionsMinor: 20_000, NetPayMinor: 230_000,
		TotalMiles: decimal.NewFromInt(1_800), ShipmentCount: 3, CurrencyCode: settlementCurrency,
	}

	return insertWithID(ctx, s.admin, driversettlementdb.Settlements, row, &row.ID)
}

func (s *settlementRows) carrierSettlement(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	marked bool,
) (pulid.ID, error) {
	hauler := &carrier.Carrier{
		BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
		Status: carrier.Status("Active"), Code: "CS" + label, Name: "Carrier " + label,
	}
	if err := carrierdb.Carriers.Create(ctx, s.admin, hauler); err != nil {
		return pulid.Nil, err
	}
	now := timeutils.NowUnix()
	day := int64(timeutils.SecondsPerDay)
	status := carriersettlement.StatusDraft
	if marked {
		status = carriersettlement.StatusVoided
	}
	row := &carriersettlement.CarrierSettlement{
		BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
		CarrierID: hauler.ID, SettlementNumber: "CS" + label, Status: status,
		PeriodStart: now - 14*day, PeriodEnd: now - 7*day, PayDate: now,
		GrossCostMinor: 180_000, NetPayableMinor: 180_000, ShipmentCount: 2,
		CurrencyCode: settlementCurrency,
	}

	return insertWithID(ctx, s.admin, carriersettlementdb.CarrierSettlements, row, &row.ID)
}

func (s *settlementRows) payProfile(
	ctx context.Context,
	tenant *agentreadtest.Tenant,
	label string,
	marked bool,
) (pulid.ID, error) {
	row := &driverpay.PayProfile{
		BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
		Status: statusOf(marked), Name: "Profile " + label,
		Classification: driverpay.PayeeClassificationCompanyDriver,
		CurrencyCode:   settlementCurrency, PerDiemRatePerMile: decimal.Zero,
	}

	return insertWithID(ctx, s.admin, driverpaydb.PayProfiles, row, &row.ID)
}

// settlementGetsWithheldAtInternal are the fields each settlement get
// withholds at the default agent data access: the pay figures, notes and
// reasons its kind ranks Restricted, as the repository tools gated them.
var settlementGetsWithheldAtInternal = map[string][]string{
	"settlement_get_driver": {
		"grossEarnings", "reimbursements", "deductions", "carryForwardIn", "carryForwardOut",
		"netPay", "totalMiles", "notes", "voidReason",
	},
	"settlement_get_carrier": {
		"grossCost", "adjustments", "netPayable", "paymentMethod", "paymentReference",
		"exchangeRate", "exchangeRateDate", "paidExchangeRate", "paidExchangeRateDate",
		"lines.amount",
	},
	"settlement_get_dispute": {"description", "resolutionNote", "settlement.netPay"},
}

// settlementListsWithheldAtInternal are the fields each settlement list
// withholds at the default agent data access, by the sensitivities its kind
// declares.
var settlementListsWithheldAtInternal = map[string][]string{
	"settlement_list_driver": {
		"carryForwardIn", "carryForwardOut", "deductions", "grossEarnings", "netPay",
		"reimbursements", "totalMiles",
	},
	"settlement_list_pay_events": {"grossAmount", "holdReason", "totalMiles", "voidReason"},
	"settlement_list_carrier": {
		"adjustments", "grossCost", "netPayable", "paymentMethod", "paymentReference",
	},
	"settlement_list_driver_expenses": {"amount", "description", "reviewNote"},
	"settlement_list_pay_profiles": {
		"guaranteedPeriodMinimum", "perDiemDailyCap", "perDiemRatePerMile",
	},
	"settlement_list_escrow_accounts": {"annualInterestRate", "balance", "targetAmount"},
	"settlement_list_advances": {
		"amount", "notes", "outstanding", "recovered", "writeOffReason", "writtenOff",
	},
	"settlement_list_recurring_deductions": {"amount", "deductedToDate", "totalCap"},
	"settlement_list_recurring_earnings":   {"amount", "paidToDate", "totalCap"},
}

// At the default agent data access a settlement list withholds only what its
// kind ranks above Internal: the pay figures, notes and reasons.
func TestConvertedLists_SettlementsWithholdOnlyTheirPayFiguresAtInternal(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	cases, services := convertedListCases(conn, admin)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	mine := agentreadtest.SeedTenant(t, admin, "A")

	for idx := range cases {
		c := &cases[idx]
		if _, ok := settlementListsWithheldAtInternal[c.tool]; !ok {
			continue
		}
		t.Run(c.tool, func(t *testing.T) {
			list, err := agentquerytoolservice.ConvertedListTool(c.tool, reader, engine)
			require.NoError(t, err)
			ours := writeReferenceRows(t, c, mine, "A")
			data, err := list.Query(ours.ctx, internalCall(mine.Actor, map[string]any{"limit": 50}))
			require.NoError(t, err)
			page := asMap(t, data)
			withheld, _ := page["withheldByAccess"].([]any)
			got := make([]string, 0, len(withheld))
			for _, path := range withheld {
				got = append(got, path.(string))
			}
			assert.ElementsMatch(t, settlementListsWithheldAtInternal[c.tool], got)
			items, _ := page["items"].([]any)
			require.Len(t, items, len(ours.rows))
			for _, item := range items {
				row, _ := item.(map[string]any)
				for _, field := range got {
					assert.NotContains(t, row, field)
				}
			}

			_, err = list.Query(ours.ctx, internalCall(mine.Actor, map[string]any{"query": "a"}))
			if _, refused := settlementSearchesRefusedAtInternal[c.tool]; refused {
				require.ErrorContains(t, err, "cannot search by query",
					"its search matches a field withheld at Internal")
				return
			}
			require.NoError(t, err)
		})
	}
}

// settlementSearchesRefusedAtInternal are the settlement lists whose
// search matches a field withheld at Internal: an expense's description and
// an advance's notes.
var settlementSearchesRefusedAtInternal = map[string]struct{}{
	"settlement_list_driver_expenses": {},
	"settlement_list_advances":        {},
}

// A settlement get reads one record of the tenant's with only the
// documented pay figures withheld at Internal, refuses another tenant's,
// and withholds what a Public role may not see.
func TestConvertedGets_SettlementsReadOneRecordUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	lists, services := convertedListCases(conn, admin)
	rows := &settlementRows{admin: admin, shipments: make(map[pulid.ID]pulid.ID)}
	byList := map[string]string{
		"settlement_list_driver":  "settlement_get_driver",
		"settlement_list_carrier": "settlement_get_carrier",
	}
	gets := make(map[string]*referenceCase, 3)
	for idx := range lists {
		if name, ok := byList[lists[idx].tool]; ok {
			gets[name] = &lists[idx]
		}
	}
	gets["settlement_get_dispute"] = &referenceCase{
		tool: "settlement_get_dispute",
		write: func(ctx context.Context, tenant *agentreadtest.Tenant, label string, _ bool) (pulid.ID, error) {
			settlementID, err := rows.driverSettlement(ctx, tenant, "Q"+label, false)
			if err != nil {
				return pulid.Nil, err
			}
			settlements := driversettlementdb.Settlements
			settlement, err := settlements.Pick(settlements.WorkerID).
				Where(settlements.ID.Eq(settlementID)).One(ctx, admin)
			if err != nil {
				return pulid.Nil, err
			}
			workerID := settlement.WorkerID
			row := &driversettlement.Dispute{
				BusinessUnitID: tenant.Actor.BusinessUnitID, OrganizationID: tenant.Actor.OrganizationID,
				SettlementID: settlementID, WorkerID: workerID,
				Status: driversettlement.DisputeStatusOpen, Category: driversettlement.DisputeCategoryMissingPay,
				Description: "The Tuesday load is missing", SubmittedByUserID: tenant.Actor.UserID,
			}
			return insertWithID(ctx, rows.admin, driversettlementdb.Disputes, row, &row.ID)
		},
	}
	require.Len(t, gets, 3)

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	public := agentreadtest.NewReader(
		t,
		agentreadtest.Options{Engine: &publicRole{}, Resolvers: services},
	)
	restricted := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &restrictedRole{}, Resolvers: services,
	})
	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")

	for name, c := range gets {
		t.Run(name, func(t *testing.T) {
			get, err := agentquerytoolservice.ConvertedGetTool(name, reader, engine)
			require.NoError(t, err)
			ours := writeReferenceRows(t, c, mine, "A")
			other := writeReferenceRows(t, c, theirs, "B")
			actor := mine.Actor
			param := get.ParamSchema()["required"].([]string)[0]

			t.Run("our record, as the operation reads it", func(t *testing.T) {
				for _, id := range ours.rows {
					record := callTool(t, ours.ctx, get, actor, map[string]any{param: id})
					assert.Equal(t, id, record["id"])
				}
			})

			t.Run("nothing is withheld where access reaches Restricted", func(t *testing.T) {
				open, err := agentquerytoolservice.ConvertedGetTool(
					name,
					restricted,
					&restrictedRole{},
				)
				require.NoError(t, err)
				params := internalCall(actor, map[string]any{param: ours.rows[0]})
				params.DataAccessCeiling = permission.SensitivityRestricted
				data, err := open.Query(ours.ctx, params)
				require.NoError(t, err)
				assert.NotContains(t, asMap(t, data), "withheldByAccess")
			})

			t.Run("only the documented fields are withheld at Internal", func(t *testing.T) {
				data, err := get.Query(
					ours.ctx,
					internalCall(actor, map[string]any{param: ours.rows[0]}),
				)
				require.NoError(t, err)
				withheld, _ := asMap(t, data)["withheldByAccess"].([]any)
				for _, path := range withheld {
					assert.Contains(t, settlementGetsWithheldAtInternal[name], path)
				}
			})

			t.Run("another tenant's record is not found", func(t *testing.T) {
				_, err := get.Query(
					ours.ctx,
					internalCall(actor, map[string]any{param: other.rows[0]}),
				)
				require.Error(t, err)
			})

			t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
				hidden, err := agentquerytoolservice.ConvertedGetTool(name, public, &publicRole{})
				require.NoError(t, err)
				record := callTool(t, ours.ctx, hidden, actor, map[string]any{param: ours.rows[0]})
				withheld, _ := record["withheldByAccess"].([]any)
				require.NotEmpty(t, withheld)
				for _, field := range withheld {
					assert.NotContains(t, record, field)
				}
			})
		})
	}
}

// A driver's unsettled pay is read by the worker's id: our driver's accrued
// pay event is counted with its gross withheld at Internal, and another
// tenant's driver reads as having earned nothing here.
func TestConvertedGets_DriverEarningsAreReadByTheWorker(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	lists, services := convertedListCases(conn, admin)
	var events *referenceCase
	for idx := range lists {
		if lists[idx].tool == "settlement_list_pay_events" {
			events = &lists[idx]
		}
	}
	require.NotNil(t, events)

	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	mine := agentreadtest.SeedTenant(t, admin, "A")
	theirs := agentreadtest.SeedTenant(t, admin, "B")
	ours := writeReferenceRows(t, events, mine, "A")
	other := writeReferenceRows(t, events, theirs, "B")
	workerOf := func(eventID string) string {
		payEvents := driversettlementdb.PayEvents
		event, err := payEvents.Pick(payEvents.WorkerID).
			Where(payEvents.ID.Eq(pulid.ID(eventID))).AllTenants().
			One(dbscope.WithSystem(t.Context(), "test"), admin)
		require.NoError(t, err)
		return event.WorkerID.String()
	}

	get, err := agentquerytoolservice.ConvertedGetTool(
		"settlement_get_driver_earnings",
		reader,
		engine,
	)
	require.NoError(t, err)

	mineID := workerOf(ours.rows[0])
	data, err := get.Query(ours.ctx, internalCall(mine.Actor, map[string]any{"workerId": mineID}))
	require.NoError(t, err)
	summary := asMap(t, data)
	assert.Equal(t, mineID, summary["workerId"])
	assert.EqualValues(t, 1, summary["accruedEventCount"])
	assert.NotContains(t, summary, "accruedGross")
	assert.ElementsMatch(t,
		[]any{"accruedGross", "outstandingAdvances", "escrowBalance"}, summary["withheldByAccess"])

	restricted := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &restrictedRole{}, Resolvers: services,
	})
	open, err := agentquerytoolservice.ConvertedGetTool(
		"settlement_get_driver_earnings", restricted, &restrictedRole{})
	require.NoError(t, err)
	params := internalCall(mine.Actor, map[string]any{"workerId": mineID})
	params.DataAccessCeiling = permission.SensitivityRestricted
	data, err = open.Query(ours.ctx, params)
	require.NoError(t, err)
	shown := asMap(t, data)
	assert.Equal(t, "1250.00", shown["accruedGross"], "minor units read as the decimal")
	assert.Equal(t, "0.00", shown["outstandingAdvances"])
	assert.NotContains(t, shown, "accruedGrossMinor")

	theirsID := workerOf(other.rows[0])
	foreign := callTool(t, ours.ctx, get, mine.Actor, map[string]any{"workerId": theirsID})
	assert.EqualValues(t, 0, foreign["accruedEventCount"])
}

// A driver's own words on a dispute or an expense mark the result as read
// from outside the organization, and only where they are shown.
func TestConvertedReads_MarkTheDriversWordsOnlyWhenShown(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	lists, services := convertedListCases(conn, admin)
	var expenses *referenceCase
	for idx := range lists {
		if lists[idx].tool == "settlement_list_driver_expenses" {
			expenses = &lists[idx]
		}
	}
	require.NotNil(t, expenses)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	mine := agentreadtest.SeedTenant(t, admin, "A")
	ours := writeReferenceRows(t, expenses, mine, "A")

	list, err := agentquerytoolservice.ConvertedListTool(
		"settlement_list_driver_expenses",
		reader,
		engine,
	)
	require.NoError(t, err)
	params := internalCall(mine.Actor, map[string]any{"limit": 50})

	hidden, err := list.Query(ours.ctx, params)
	require.NoError(t, err)
	assert.Empty(t, hidden.(agent.TaintCarrier).TaintedRecords(),
		"the description is withheld at Internal, so nothing outside was read")

	restricted := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &restrictedRole{}, Resolvers: services,
	})
	open, err := agentquerytoolservice.ConvertedListTool(
		"settlement_list_driver_expenses", restricted, &restrictedRole{})
	require.NoError(t, err)
	params.DataAccessCeiling = permission.SensitivityRestricted
	shown, err := open.Query(ours.ctx, params)
	require.NoError(t, err)
	marked := shown.(agent.TaintCarrier).TaintedRecords()
	require.Len(t, marked, len(ours.rows))
	for _, ref := range marked {
		assert.Equal(t, "driver_expense", ref.EntityType)
		assert.Contains(t, ours.rows, ref.ID)
	}
}

func internalCall(
	actor *permissionports.RequestActor,
	args map[string]any,
) *agenttoolports.QueryToolParams {
	return &agenttoolports.QueryToolParams{
		OrganizationID:    actor.OrganizationID,
		BusinessUnitID:    actor.BusinessUnitID,
		Actor:             actor,
		Timezone:          "America/Chicago",
		Params:            args,
		DataAccessCeiling: permission.SensitivityInternal,
		RunID:             pulid.MustNew("ar_"),
	}
}

// An escrow account's query matches what identifies an account without its
// money, its status and currency, through the real database: a reader whose
// access reaches Restricted finds the account, and so does one at Internal,
// since nothing the query matches is withheld there.
func TestConvertedLists_EscrowAccountsSearchWithoutTheirMoney(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	cases, services := convertedListCases(conn, admin)
	idx := slices.IndexFunc(cases, func(c referenceCase) bool {
		return c.tool == "settlement_list_escrow_accounts"
	})
	require.GreaterOrEqual(t, idx, 0)
	mine := agentreadtest.SeedTenant(t, admin, "A")
	ours := writeReferenceRows(t, &cases[idx], mine, "A")

	restrictedEngine := &tcaRestrictedRole{}
	reached, err := agentquerytoolservice.ConvertedListTool(cases[idx].tool,
		agentreadtest.NewReader(t, agentreadtest.Options{
			Engine: restrictedEngine, Resolvers: services,
		}), restrictedEngine)
	require.NoError(t, err)
	data, err := reached.Query(ours.ctx, &agenttoolports.QueryToolParams{
		OrganizationID: mine.Actor.OrganizationID, BusinessUnitID: mine.Actor.BusinessUnitID,
		Actor: mine.Actor, Timezone: "America/Chicago",
		Params:            map[string]any{"query": "Closed"},
		DataAccessCeiling: permission.SensitivityRestricted, RunID: pulid.MustNew("ar_"),
	})
	require.NoError(t, err, "a Restricted reader's search reads")
	closed := rowsByID(asMap(t, data))
	require.Len(t, closed, 1, "the one closed account matches its status")

	engine := &mocks.DenyResourcesPermissionEngine{}
	internal, err := agentquerytoolservice.ConvertedListTool(cases[idx].tool,
		agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services}),
		engine)
	require.NoError(t, err)
	data, err = internal.Query(ours.ctx, internalCall(mine.Actor, map[string]any{"query": "usd"}))
	require.NoError(t, err, "nothing the query matches is withheld at Internal")
	assert.Len(t, rowsByID(asMap(t, data)), len(ours.rows), "every account is in dollars")
}
