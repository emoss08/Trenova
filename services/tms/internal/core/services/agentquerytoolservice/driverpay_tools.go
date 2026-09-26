package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/driverpayservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	absentNotClosed  = "not closed"
	fieldSplit       = "splitPercent"
	fieldTarget      = "targetAmountMinor"
	fieldBalance     = "balanceMinor"
	fieldInterest    = "annualInterestRate"
	fieldRecovered   = "recoveredMinor"
	fieldCap         = "totalCapMinor"
	fieldDeducted    = "deductedToDateMinor"
	fieldPaidToDate  = "paidToDateMinor"
	workerIDFromList = "Only this driver's, by id from search_worker or list_workers."
)

var (
	payCodeDirections = []string{
		string(driverpay.PayCodeDirectionEarning),
		string(driverpay.PayCodeDirectionDeduction),
	}
	payeeClassifications = []string{
		string(driverpay.PayeeClassificationCompanyDriver),
		string(driverpay.PayeeClassificationOwnerOperator),
	}
	escrowStatuses = []string{
		string(driverpay.EscrowAccountStatusActive),
		string(driverpay.EscrowAccountStatusClosed),
	}
	advanceStatuses = []string{
		string(driverpay.AdvanceStatusOutstanding),
		string(driverpay.AdvanceStatusPartiallyRecovered),
		string(driverpay.AdvanceStatusRecovered),
		string(driverpay.AdvanceStatusWrittenOff),
	}
	deductionStatuses = []string{
		string(driverpay.DeductionStatusActive),
		string(driverpay.DeductionStatusPaused),
		string(driverpay.DeductionStatusCompleted),
	}
	earningStatuses = []string{
		string(driverpay.EarningStatusActive),
		string(driverpay.EarningStatusPaused),
		string(driverpay.EarningStatusCompleted),
	}
)

func driverPayToolProviders() []any {
	return []any{
		provideListPayCodesTool,
		provideListPayProfilesTool,
		provideListWorkerPayAssignmentsTool,
		provideListEscrowAccountsTool,
		provideListPayAdvancesTool,
		provideListRecurringDeductionsTool,
		provideListRecurringEarningsTool,
	}
}

type driverPayReader interface {
	ListActivePayCodes(
		ctx context.Context,
		req repositories.ListActivePayCodesRequest,
	) ([]*driverpay.PayCode, error)
	ListProfiles(
		ctx context.Context,
		req *repositories.ListPayProfilesRequest,
	) (*pagination.ListResult[*driverpay.PayProfile], error)
	ListWorkerAssignments(
		ctx context.Context,
		req repositories.ListWorkerPayAssignmentsRequest,
	) ([]*driverpay.WorkerPayAssignment, error)
	ListEscrowAccounts(
		ctx context.Context,
		req *repositories.ListEscrowAccountsRequest,
	) (*pagination.ListResult[*driverpay.EscrowAccount], error)
	ListAdvances(
		ctx context.Context,
		req *repositories.ListPayAdvancesRequest,
	) (*pagination.ListResult[*driverpay.PayAdvance], error)
	ListDeductions(
		ctx context.Context,
		req *repositories.ListRecurringDeductionsRequest,
	) (*pagination.ListResult[*driverpay.RecurringDeduction], error)
	ListEarnings(
		ctx context.Context,
		req *repositories.ListRecurringEarningsRequest,
	) (*pagination.ListResult[*driverpay.RecurringEarning], error)
}

type driverPayTool struct {
	pay    driverPayReader
	access fieldAccess
}

func newDriverPayTool(
	pay driverPayReader,
	permissions serviceports.PermissionEngine,
) driverPayTool {
	return driverPayTool{pay: pay, access: newFieldAccess(permissions)}
}

func provideListPayCodesTool(
	pay *driverpayservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listPayCodesTool{newDriverPayTool(pay, permissions)}
}

func provideListPayProfilesTool(
	pay *driverpayservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listPayProfilesTool{newDriverPayTool(pay, permissions)}
}

func provideListWorkerPayAssignmentsTool(
	pay *driverpayservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listWorkerPayAssignmentsTool{newDriverPayTool(pay, permissions)}
}

func provideListEscrowAccountsTool(
	pay *driverpayservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listEscrowAccountsTool{newDriverPayTool(pay, permissions)}
}

func provideListPayAdvancesTool(
	pay *driverpayservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listPayAdvancesTool{newDriverPayTool(pay, permissions)}
}

func provideListRecurringDeductionsTool(
	pay *driverpayservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listRecurringDeductionsTool{newDriverPayTool(pay, permissions)}
}

func provideListRecurringEarningsTool(
	pay *driverpayservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listRecurringEarningsTool{newDriverPayTool(pay, permissions)}
}

type workerFilter struct {
	workerID pulid.ID
	status   string
	window   page
}

func readWorkerFilter(
	params *serviceports.QueryToolParams,
	statuses []string,
) (workerFilter, error) {
	workerID, err := optionalID(params.Params, paramWorkerID)
	if err != nil {
		return workerFilter{}, err
	}
	status, err := validEnum(params.Params, paramStatus, statuses)
	if err != nil {
		return workerFilter{}, err
	}

	return workerFilter{
		workerID: workerID,
		status:   status,
		window:   readPage(params.Params, defaultListLimit, maxListLimit),
	}, nil
}

func (f workerFilter) options(params *serviceports.QueryToolParams) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: tenantOf(params),
		Pagination: pagination.Info{Limit: f.window.fetch(), Offset: f.window.offset},
	}
}

func (f workerFilter) criteria(
	entity string,
	params *serviceports.QueryToolParams,
) *filtercatalog.Criteria {
	criteria := filtercatalog.NewCriteria(entity).At(clockFor(params))
	if f.workerID.IsNotNil() {
		criteria.Field("driver", f.workerID.String())
	}
	if f.status != "" {
		criteria.Field(paramStatus, f.status)
	}

	return criteria
}

func workerFilterSchema(statuses []string, what string) map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramWorkerID: stringParam(workerIDFromList),
		paramStatus:   enumParam("Only "+what+" in this status.", statuses),
	}, defaultListLimit, maxListLimit))
}

type listPayCodesTool struct{ driverPayTool }

func (t *listPayCodesTool) Name() string { return "list_pay_codes" }

func (t *listPayCodesTool) Description() string {
	return "List the organization's active earning and deduction pay codes, each with the " +
		"GL account it maps to. Settlement lines, adjustments and recurring pay post under " +
		"them. Narrow to earnings or deductions."
}

func (t *listPayCodesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		"direction": enumParam("Only earning or only deduction codes.", payCodeDirections),
	})
}

func (t *listPayCodesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourcePayCode})
}

type payCodeRow struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Direction   string `json:"direction"`
	Taxable     bool   `json:"taxable"`
	GLAccountID string `json:"glAccountId,omitempty"`
	System      bool   `json:"system"`
}

func (t *listPayCodesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	direction, err := validEnum(params.Params, "direction", payCodeDirections)
	if err != nil {
		return nil, err
	}

	codes, err := t.pay.ListActivePayCodes(ctx, repositories.ListActivePayCodesRequest{
		TenantInfo: tenantOf(params),
		Direction:  driverpay.PayCodeDirection(direction),
	})
	if err != nil {
		return nil, err
	}

	criteria := filtercatalog.NewCriteria("pay codes").At(clockFor(params))
	if direction != "" {
		criteria.Field("direction", direction)
	}
	rows := make([]payCodeRow, 0, len(codes))
	for _, code := range codes {
		if code == nil {
			continue
		}
		rows = append(rows, payCodeRow{
			ID:          code.ID.String(),
			Code:        code.Code,
			Name:        code.Name,
			Direction:   string(code.Direction),
			Taxable:     code.Taxable,
			GLAccountID: pointerIDString(code.GLAccountID),
			System:      code.IsSystem,
		})
	}
	found := searchResult(criteria, rows, len(rows))

	return &found, nil
}

type listPayProfilesTool struct{ driverPayTool }

func (t *listPayProfilesTool) Name() string { return "list_pay_profiles" }

func (t *listPayProfilesTool) Description() string {
	return "List the organization's driver pay profiles, the packages that say how a driver " +
		"is paid per mile, per load or by percentage. Each has its classification and status. " +
		"Narrow by name or classification before assigning one."
}

func (t *listPayProfilesTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		paramQuery: stringParam("Words to look for in the profile name."),
		"classification": enumParam("Only company-driver or only owner-operator profiles.",
			payeeClassifications),
	}, defaultListLimit, maxListLimit))
}

func (t *listPayProfilesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceDriverPayProfile})
}

type payProfileRow struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Classification string `json:"classification"`
	Status         string `json:"status"`
	Currency       string `json:"currency"`
	Description    string `json:"description,omitempty"`
}

func (t *listPayProfilesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	classification, err := validEnum(params.Params, "classification", payeeClassifications)
	if err != nil {
		return nil, err
	}
	window := readPage(params.Params, defaultListLimit, maxListLimit)
	query := optionalString(params.Params, paramQuery)

	result, err := t.pay.ListProfiles(ctx, &repositories.ListPayProfilesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: window.fetch(), Offset: window.offset},
			Query:      query,
		},
		Classification: driverpay.PayeeClassification(classification),
	})
	if err != nil {
		return nil, err
	}

	criteria := filtercatalog.NewCriteria("pay profiles").At(clockFor(params))
	criteria.Text(query)
	if classification != "" {
		criteria.Field("classification", classification)
	}
	profiles, more := trim(window, result.Items)
	rows := make([]payProfileRow, 0, len(profiles))
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		rows = append(rows, payProfileRow{
			ID:             profile.ID.String(),
			Name:           profile.Name,
			Classification: string(profile.Classification),
			Status:         string(profile.Status),
			Currency:       profile.CurrencyCode,
			Description:    profile.Description,
		})
	}
	found := searchResult(criteria, rows, len(rows)).paged(window, more)

	return &found, nil
}

type listWorkerPayAssignmentsTool struct{ driverPayTool }

func (t *listWorkerPayAssignmentsTool) Name() string { return "list_worker_pay_assignments" }

func (t *listWorkerPayAssignmentsTool) Description() string {
	return "List a driver's pay profile assignments, newest first: which profile paid them " +
		"from when to when, their share of each load, and which one is in force today."
}

func (t *listWorkerPayAssignmentsTool) ParamSchema() map[string]any {
	return idSchema(paramWorkerID, "The driver, by id from search_worker or list_workers.")
}

func (t *listWorkerPayAssignmentsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceDriverPayProfile})
}

type payAssignmentRow struct {
	ID            string       `json:"id"`
	PayProfileID  string       `json:"payProfileId"`
	PayProfile    string       `json:"payProfile,omitempty"`
	EffectiveFrom optionalDate `json:"effectiveFrom"`
	EffectiveTo   optionalDate `json:"effectiveTo"`
	InForce       bool         `json:"inForce"`
	SplitPercent  string       `json:"splitPercent,omitempty"`
	Notes         string       `json:"notes,omitempty"`
}

type payAssignmentsView struct {
	WorkerID    string             `json:"workerId"`
	Assignments []payAssignmentRow `json:"assignments"`
	Withheld    []string           `json:"withheldByAccess,omitempty"`
}

func (t *listWorkerPayAssignmentsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, paramWorkerID)
	if err != nil {
		return nil, err
	}

	assignments, err := t.pay.ListWorkerAssignments(
		ctx,
		repositories.ListWorkerPayAssignmentsRequest{
			TenantInfo: tenantOf(params),
			WorkerID:   workerID,
		},
	)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceDriverPayProfile)
	showSplit := gate.show(fieldSplit, fieldSplit)
	now := timeutils.NowUnix()
	view := payAssignmentsView{
		WorkerID:    workerID.String(),
		Assignments: make([]payAssignmentRow, 0, len(assignments)),
	}
	for _, assignment := range assignments {
		if assignment == nil {
			continue
		}
		row := payAssignmentRow{
			ID:            assignment.ID.String(),
			PayProfileID:  assignment.PayProfileID.String(),
			EffectiveFrom: recordedDate(assignment.EffectiveFrom),
			EffectiveTo:   expectedDate(derefInt64(assignment.EffectiveTo), absentOpenEnded),
			InForce: assignment.EffectiveFrom <= now &&
				(assignment.EffectiveTo == nil || *assignment.EffectiveTo > now),
			Notes: assignment.Notes,
		}
		if assignment.PayProfile != nil {
			row.PayProfile = assignment.PayProfile.Name
		}
		if showSplit {
			row.SplitPercent = assignment.SplitPercent.String()
		}
		view.Assignments = append(view.Assignments, row)
	}
	view.Withheld = gate.Withheld()

	return view, nil
}

type listEscrowAccountsTool struct{ driverPayTool }

func (t *listEscrowAccountsTool) Name() string { return "list_escrow_accounts" }

func (t *listEscrowAccountsTool) Description() string {
	return "List owner-operator escrow accounts with the driver, status and when each " +
		"opened or closed. The balance, target and interest rate show when your data access " +
		"reaches them. Narrow to one driver or a status."
}

func (t *listEscrowAccountsTool) ParamSchema() map[string]any {
	return workerFilterSchema(escrowStatuses, "accounts")
}

func (t *listEscrowAccountsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceEscrowAccount})
}

type escrowAccountRow struct {
	ID           string       `json:"id"`
	WorkerID     string       `json:"workerId"`
	Worker       string       `json:"worker,omitempty"`
	Status       string       `json:"status"`
	OpenedDate   optionalDate `json:"openedDate"`
	ClosedDate   optionalDate `json:"closedDate"`
	Currency     string       `json:"currency"`
	Balance      string       `json:"balance,omitempty"`
	Target       string       `json:"target,omitempty"`
	InterestRate string       `json:"annualInterestRate,omitempty"`
}

func (t *listEscrowAccountsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	filter, err := readWorkerFilter(params, escrowStatuses)
	if err != nil {
		return nil, err
	}

	result, err := t.pay.ListEscrowAccounts(ctx, &repositories.ListEscrowAccountsRequest{
		Filter:   filter.options(params),
		WorkerID: filter.workerID,
		Status:   driverpay.EscrowAccountStatus(filter.status),
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceEscrowAccount)
	showBalance := gate.show(fieldBalance, "balance")
	showTarget := gate.show(fieldTarget, "target")
	showRate := gate.show(fieldInterest, fieldInterest)
	accounts, more := trim(filter.window, result.Items)
	rows := make([]escrowAccountRow, 0, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		row := escrowAccountRow{
			ID:         account.ID.String(),
			WorkerID:   account.WorkerID.String(),
			Worker:     workerName(account.Worker),
			Status:     string(account.Status),
			OpenedDate: recordedDate(account.OpenedDate),
			ClosedDate: expectedDate(derefInt64(account.ClosedDate), absentNotClosed),
			Currency:   account.CurrencyCode,
		}
		if showBalance {
			row.Balance = minorText(account.BalanceMinor)
		}
		if showTarget {
			row.Target = minorText(account.TargetAmountMinor)
		}
		if showRate {
			row.InterestRate = account.AnnualInterestRate.String()
		}
		rows = append(rows, row)
	}
	found := searchResult(filter.criteria("escrow accounts", params), rows, len(rows)).
		paged(filter.window, more)

	return gatedResult(&found, gate), nil
}

type listPayAdvancesTool struct{ driverPayTool }

func (t *listPayAdvancesTool) Name() string { return "list_pay_advances" }

func (t *listPayAdvancesTool) Description() string {
	return "List driver pay advances, the cash and money codes given against future pay, " +
		"newest first. Each has the driver, how it was given, its reference and status. The " +
		"amount, recovered and still-owed figures show when your data access reaches them."
}

func (t *listPayAdvancesTool) ParamSchema() map[string]any {
	return workerFilterSchema(advanceStatuses, "advances")
}

func (t *listPayAdvancesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourcePayAdvance})
}

type payAdvanceRow struct {
	ID         string       `json:"id"`
	WorkerID   string       `json:"workerId"`
	Worker     string       `json:"worker,omitempty"`
	Status     string       `json:"status"`
	Source     string       `json:"source"`
	Reference  string       `json:"reference,omitempty"`
	IssuedDate optionalDate `json:"issuedDate"`
	Currency   string       `json:"currency"`
	Amount     string       `json:"amount,omitempty"`
	Recovered  string       `json:"recovered,omitempty"`
	StillOwed  string       `json:"stillOwed,omitempty"`
}

func (t *listPayAdvancesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	filter, err := readWorkerFilter(params, advanceStatuses)
	if err != nil {
		return nil, err
	}

	result, err := t.pay.ListAdvances(ctx, &repositories.ListPayAdvancesRequest{
		Filter:   filter.options(params),
		WorkerID: filter.workerID,
		Status:   driverpay.AdvanceStatus(filter.status),
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourcePayAdvance)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)
	showRecovered := showAmounts && gate.show(fieldRecovered, withheldAmounts)
	advances, more := trim(filter.window, result.Items)
	rows := make([]payAdvanceRow, 0, len(advances))
	for _, advance := range advances {
		if advance == nil {
			continue
		}
		row := payAdvanceRow{
			ID:         advance.ID.String(),
			WorkerID:   advance.WorkerID.String(),
			Worker:     workerName(advance.Worker),
			Status:     string(advance.Status),
			Source:     string(advance.Source),
			Reference:  advance.Reference,
			IssuedDate: recordedDate(advance.IssuedDate),
			Currency:   advance.CurrencyCode,
		}
		if showRecovered {
			row.Amount = minorText(advance.AmountMinor)
			row.Recovered = minorText(advance.RecoveredMinor)
			row.StillOwed = minorText(advance.OutstandingMinor())
		}
		rows = append(rows, row)
	}
	found := searchResult(filter.criteria("pay advances", params), rows, len(rows)).
		paged(filter.window, more)

	return gatedResult(&found, gate), nil
}

type recurringRow struct {
	ID              string       `json:"id"`
	WorkerID        string       `json:"workerId"`
	Worker          string       `json:"worker,omitempty"`
	Status          string       `json:"status"`
	Kind            string       `json:"kind,omitempty"`
	Frequency       string       `json:"frequency"`
	Description     string       `json:"description"`
	PayCodeID       string       `json:"payCodeId"`
	EscrowAccountID string       `json:"escrowAccountId,omitempty"`
	StartDate       optionalDate `json:"startDate"`
	EndDate         optionalDate `json:"endDate"`
	Currency        string       `json:"currency"`
	Amount          string       `json:"amount,omitempty"`
	Cap             string       `json:"totalCap,omitempty"`
	ToDate          string       `json:"toDate,omitempty"`
}

type listRecurringDeductionsTool struct{ driverPayTool }

func (t *listRecurringDeductionsTool) Name() string { return "list_recurring_deductions" }

func (t *listRecurringDeductionsTool) Description() string {
	return "List drivers' recurring deductions, such as insurance, equipment lease or escrow " +
		"contributions. Each has the driver, schedule, dates and status. The amount, cap and " +
		"what has been taken so far show when your data access reaches them."
}

func (t *listRecurringDeductionsTool) ParamSchema() map[string]any {
	return workerFilterSchema(deductionStatuses, "deductions")
}

func (t *listRecurringDeductionsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRecurringDeduction})
}

func (t *listRecurringDeductionsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	filter, err := readWorkerFilter(params, deductionStatuses)
	if err != nil {
		return nil, err
	}

	result, err := t.pay.ListDeductions(ctx, &repositories.ListRecurringDeductionsRequest{
		Filter:   filter.options(params),
		WorkerID: filter.workerID,
		Status:   driverpay.DeductionStatus(filter.status),
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceRecurringDeduction)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)
	deductions, more := trim(filter.window, result.Items)
	rows := make([]recurringRow, 0, len(deductions))
	for _, deduction := range deductions {
		if deduction == nil {
			continue
		}
		row := recurringRow{
			ID:              deduction.ID.String(),
			WorkerID:        deduction.WorkerID.String(),
			Worker:          workerName(deduction.Worker),
			Status:          string(deduction.Status),
			Kind:            string(deduction.Kind),
			Frequency:       string(deduction.Frequency),
			Description:     deduction.Description,
			PayCodeID:       deduction.PayCodeID.String(),
			EscrowAccountID: pointerIDString(deduction.EscrowAccountID),
			StartDate:       recordedDate(deduction.StartDate),
			EndDate:         expectedDate(derefInt64(deduction.EndDate), absentOpenEnded),
			Currency:        deduction.CurrencyCode,
		}
		if showAmounts && gate.show(fieldCap, withheldAmounts) &&
			gate.show(fieldDeducted, withheldAmounts) {
			row.Amount = minorText(deduction.AmountMinor)
			row.Cap = optionalMinorText(deduction.TotalCapMinor)
			row.ToDate = minorText(deduction.DeductedToDateMinor)
		}
		rows = append(rows, row)
	}
	found := searchResult(filter.criteria("recurring deductions", params), rows, len(rows)).
		paged(filter.window, more)

	return gatedResult(&found, gate), nil
}

type listRecurringEarningsTool struct{ driverPayTool }

func (t *listRecurringEarningsTool) Name() string { return "list_recurring_earnings" }

func (t *listRecurringEarningsTool) Description() string {
	return "List drivers' recurring earnings, such as per diem, a guaranteed stipend or a " +
		"bonus paid over time. Each has the driver, schedule, dates and status. The amount, " +
		"cap and what has been paid so far show when your data access reaches them."
}

func (t *listRecurringEarningsTool) ParamSchema() map[string]any {
	return workerFilterSchema(earningStatuses, "earnings")
}

func (t *listRecurringEarningsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRecurringEarning})
}

func (t *listRecurringEarningsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	filter, err := readWorkerFilter(params, earningStatuses)
	if err != nil {
		return nil, err
	}

	result, err := t.pay.ListEarnings(ctx, &repositories.ListRecurringEarningsRequest{
		Filter:   filter.options(params),
		WorkerID: filter.workerID,
		Status:   driverpay.EarningStatus(filter.status),
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceRecurringEarning)
	showAmounts := gate.show(fieldAmountMinor, withheldAmounts)
	earnings, more := trim(filter.window, result.Items)
	rows := make([]recurringRow, 0, len(earnings))
	for _, earning := range earnings {
		if earning == nil {
			continue
		}
		row := recurringRow{
			ID:          earning.ID.String(),
			WorkerID:    earning.WorkerID.String(),
			Worker:      workerName(earning.Worker),
			Status:      string(earning.Status),
			Kind:        string(earning.Kind),
			Frequency:   string(earning.Frequency),
			Description: earning.Description,
			PayCodeID:   earning.PayCodeID.String(),
			StartDate:   recordedDate(earning.StartDate),
			EndDate:     expectedDate(derefInt64(earning.EndDate), absentOpenEnded),
			Currency:    earning.CurrencyCode,
		}
		if showAmounts && gate.show(fieldCap, withheldAmounts) &&
			gate.show(fieldPaidToDate, withheldAmounts) {
			row.Amount = minorText(earning.AmountMinor)
			row.Cap = optionalMinorText(earning.TotalCapMinor)
			row.ToDate = minorText(earning.PaidToDateMinor)
		}
		rows = append(rows, row)
	}
	found := searchResult(filter.criteria("recurring earnings", params), rows, len(rows)).
		paged(filter.window, more)

	return gatedResult(&found, gate), nil
}

func optionalMinorText(minor *int64) string {
	if minor == nil {
		return ""
	}

	return minorText(*minor)
}
