package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	pageKeyHasMore    = "hasMore"
	pageKeyNextCursor = "nextCursor"
)

const (
	paramFactoringInvoice = "invoiceId"
	paramFactoringFactor  = "factoringCompanyId"
	paramFactoringVerdict = "verdict"
	paramFactoringStatus  = "status"

	factoringStatesDefaultLimit = 25
	factoringStatesMaxLimit     = 50
	factoringNoasDefaultLimit   = 25
	factoringNoasMaxLimit       = 50

	factoringQueuePath = "/billing/factoring"
	minorUnitsExponent = -2
)

type factoringCheckRow struct {
	Code             string `json:"code"`
	Passed           bool   `json:"passed"`
	Message          string `json:"message"`
	Fix              string `json:"fix,omitempty"`
	DocumentTypeID   string `json:"documentTypeId,omitempty"`
	DocumentTypeName string `json:"documentTypeName,omitempty"`
	ShipmentID       string `json:"shipmentId,omitempty"`
	ShipmentPro      string `json:"shipmentPro,omitempty"`
}

type factoringStateRow struct {
	InvoiceID          string              `json:"invoiceId"`
	CustomerID         string              `json:"customerId"`
	FactoringCompanyID string              `json:"factoringCompanyId,omitempty"`
	Verdict            string              `json:"verdict"`
	FailedChecks       int                 `json:"failedChecks"`
	OpenBalance        string              `json:"openBalance,omitempty"`
	ReadySince         optionalDate        `json:"readySince"`
	EvaluatedAt        optionalDate        `json:"evaluatedAt"`
	Checks             []factoringCheckRow `json:"checks,omitempty"`
}

type factoringStatesResult struct {
	Invoices   []factoringStateRow     `json:"invoices"`
	HasMore    bool                    `json:"hasMore"`
	NextCursor string                  `json:"nextCursor,omitempty"`
	Summary    *factoringports.Summary `json:"summary,omitempty"`
	QueuePath  string                  `json:"queuePath"`
	Withheld   []string                `json:"withheldByAccess,omitempty"`
}

func factoringStateRowFrom(
	state *factoring.InvoiceState,
	show amounts,
	withChecks bool,
) factoringStateRow {
	row := factoringStateRow{
		InvoiceID:   state.InvoiceID.String(),
		CustomerID:  state.CustomerID.String(),
		Verdict:     string(state.Verdict),
		ReadySince:  pointerDateOr(state.ReadySince, absentNotReady),
		EvaluatedAt: recordedDate(state.EvaluatedAt),
		OpenBalance: show.of(decimal.New(state.OpenBalanceMinor, minorUnitsExponent)),
	}
	row.FailedChecks = state.FailedChecks
	if state.FactoringCompanyID != nil {
		row.FactoringCompanyID = state.FactoringCompanyID.String()
	}
	if withChecks {
		row.Checks = make([]factoringCheckRow, 0, len(state.Checks))
		for idx := range state.Checks {
			c := state.Checks[idx]
			row.Checks = append(row.Checks, factoringCheckRow{
				Code:             string(c.Code),
				Passed:           c.Passed,
				Message:          c.Message,
				Fix:              string(c.Fix),
				DocumentTypeID:   pulidString(c.DocumentTypeID),
				DocumentTypeName: c.DocumentTypeName,
				ShipmentID:       pulidString(c.ShipmentID),
				ShipmentPro:      c.ShipmentPro,
			})
		}
	}
	return row
}

type getFactoringReadinessTool struct {
	readiness factoringports.ReadinessService
	access    fieldAccess
}

func provideGetFactoringReadinessTool(
	readiness factoringports.ReadinessService,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getFactoringReadinessTool{readiness: readiness, access: newFieldAccess(permissions)}
}

func (t *getFactoringReadinessTool) Name() string { return "get_factoring_readiness" }

func (t *getFactoringReadinessTool) Description() string {
	return "Check now whether a posted invoice is ready to sell to its factor, and why not. " +
		"Every check comes back with what clears it: SetUpAgreement, SendNoa, " +
		"RecordNoaAcknowledgement, ResolveDispute, RequestDocument (with the document type " +
		"and shipment) or ReviewShipment. An Excluded verdict means the invoice is not " +
		"something to factor: a draft, voided, paid, a credit memo, kept out by " +
		"set_invoice_factoring_override, or for a customer that is not factored."
}

func (t *getFactoringReadinessTool) SearchTerms() []string {
	return []string{"factor", "factoring", "ready to factor", "notice of assignment", "noa"}
}

func (t *getFactoringReadinessTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringInvoice: agenttoolschema.RecordIDText(
			permission.ResourceInvoice,
			"The invoice, from invoice_list, invoice_get or list_factoring_ready_invoices.",
		),
	}, paramFactoringInvoice)
}

func (t *getFactoringReadinessTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

func (t *getFactoringReadinessTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	invoiceID, err := requirePulid(params.Params, paramFactoringInvoice)
	if err != nil {
		return nil, err
	}
	state, err := t.readiness.GetReadiness(ctx, factoringports.GetReadinessRequest{
		TenantInfo: tenantOf(params),
		InvoiceID:  invoiceID,
	})
	if err != nil {
		return nil, err
	}
	show, gate := amountsFor(ctx, t.access, params)
	result := &factoringStatesResult{
		Invoices:  []factoringStateRow{factoringStateRowFrom(state, show, true)},
		QueuePath: factoringQueuePath,
	}
	if gate != nil {
		result.Withheld = gate.Withheld()
	}
	return result, nil
}

type listFactoringReadyInvoicesTool struct {
	readiness factoringports.ReadinessService
	access    fieldAccess
}

func provideListFactoringReadyInvoicesTool(
	readiness factoringports.ReadinessService,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listFactoringReadyInvoicesTool{
		readiness: readiness,
		access:    newFieldAccess(permissions),
	}
}

func (t *listFactoringReadyInvoicesTool) Name() string { return "list_factoring_ready_invoices" }

func (t *listFactoringReadyInvoicesTool) Description() string {
	return "List factored customers' invoices by readiness, newest first, with the totals. " +
		"The totals are ready, not ready, submitted, and ready for more than a day without " +
		"being submitted. " +
		"Ready is the default; ask for NotReady to see what is holding invoices back, each " +
		"with its failed checks, or Submitted for what is on a schedule. Pass Ready invoices " +
		"to submit_invoices_to_factor, one factor per call, and an invoice to " +
		"get_factoring_readiness for every check."
}

func (t *listFactoringReadyInvoicesTool) SearchTerms() []string {
	return []string{"factoring queue", "ready to factor", "factor"}
}

func (t *listFactoringReadyInvoicesTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringVerdict: agenttoolschema.Enum(
			"Which invoices: Ready (the default), NotReady, or Submitted (on a schedule with the factor).",
			agenttoolschema.FactoringQueueVerdicts,
		),
		paramFactoringFactor: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"Only invoices assigned to this factor, from invoice_get or list_factoring_noas.",
		),
		paramCustomerID: agenttoolschema.RecordIDText(
			permission.ResourceCustomer,
			"Only this customer's invoices, from list_customers or invoice_get.",
		),
		paramLimit: jsonschemautils.Integer(fmt.Sprintf(
			"How many invoices to return, at most %d. Defaults to %d.",
			factoringStatesMaxLimit, factoringStatesDefaultLimit,
		)),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	})
}

func (t *listFactoringReadyInvoicesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceInvoice})
}

func (t *listFactoringReadyInvoicesTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	verdict := factoring.Verdict(optionalString(params.Params, paramFactoringVerdict))
	if verdict == "" {
		verdict = factoring.VerdictReady
	}
	if !slices.Contains(factoring.QueueVerdicts(), verdict) {
		return nil, fmt.Errorf(
			"parameter %q must be Ready, NotReady or Submitted",
			paramFactoringVerdict,
		)
	}
	factorID, err := optionalID(params.Params, paramFactoringFactor)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramFactoringFactor, err)
	}
	customerID, err := optionalID(params.Params, paramCustomerID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramCustomerID, err)
	}
	limit := min(
		max(optionalInt(params.Params, paramLimit, factoringStatesDefaultLimit), 1),
		factoringStatesMaxLimit,
	)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}

	tenant := tenantOf(params)
	page, err := t.readiness.ListStates(ctx, &factoringports.ListInvoiceStateConnectionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: limit},
		},
		Cursor:             cursor,
		Verdicts:           []factoring.Verdict{verdict},
		FactoringCompanyID: factorID,
		CustomerID:         customerID,
	})
	if err != nil {
		return nil, err
	}
	summary, err := t.readiness.Summary(ctx, tenant)
	if err != nil {
		return nil, err
	}

	show, gate := amountsFor(ctx, t.access, params)
	if !show.show {
		summary = nil
	}
	result := &factoringStatesResult{
		Invoices:  make([]factoringStateRow, 0, len(page.Items)),
		HasMore:   page.HasNextPage,
		Summary:   summary,
		QueuePath: factoringQueuePath,
	}
	for _, state := range page.Items {
		if state == nil {
			continue
		}
		row := factoringStateRowFrom(state, show, verdict == factoring.VerdictNotReady)
		if verdict == factoring.VerdictNotReady {
			row.Checks = failedOnly(row.Checks)
		}
		result.Invoices = append(result.Invoices, row)
	}
	if result.NextCursor, err = page.NextCursor(); err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}
	if gate != nil {
		result.Withheld = gate.Withheld()
	}
	return result, nil
}

type factoringFeeTierRow struct {
	DaysFrom  int    `json:"daysFrom"`
	DaysTo    string `json:"daysTo"`
	RateBP    int    `json:"rateBp"`
	FlatMinor int64  `json:"flatMinor"`
}

type factoringAgreementRow struct {
	ID                   string                `json:"id"`
	Status               string                `json:"status"`
	InForce              bool                  `json:"inForce"`
	EffectiveFrom        optionalDate          `json:"effectiveFrom"`
	EffectiveTo          optionalDate          `json:"effectiveTo"`
	AdvanceRateBP        int                   `json:"advanceRateBp"`
	ReserveRateBP        int                   `json:"reserveRateBp"`
	MinimumFeeMinor      int64                 `json:"minimumFeeMinor"`
	ChargebackWindowDays int                   `json:"chargebackWindowDays"`
	Recourse             string                `json:"recourse"`
	AccountingTreatment  string                `json:"accountingTreatment"`
	FundingDays          int                   `json:"fundingDays"`
	SubmissionTime       string                `json:"submissionTime"`
	FeeTiers             []factoringFeeTierRow `json:"feeTiers"`
	RequiredDocuments    []string              `json:"requiredDocuments"`
}

type getFactoringAgreementTool struct {
	agreements factoringports.AgreementService
	now        func() int64
}

func provideGetFactoringAgreementTool(
	agreements factoringports.AgreementService,
) agentports.AgentQueryTool {
	return &getFactoringAgreementTool{agreements: agreements, now: timeutils.NowUnix}
}

func (t *getFactoringAgreementTool) Name() string { return "get_factoring_agreement" }

func (t *getFactoringAgreementTool) Description() string {
	return "Get a factor's agreements, newest first, marking the one in force. Each gives " +
		"the advance and reserve rates, fee tiers by days outstanding, minimum fee, recourse " +
		"and chargeback window, accounting treatment, funding days, the daily submission " +
		"time, and the documents the factor needs with every invoice. A factor with no " +
		"agreement in force has no invoices ready to factor."
}

func (t *getFactoringAgreementTool) SearchTerms() []string {
	return []string{"factoring agreement", "advance rate", "factoring fee", "recourse"}
}

func (t *getFactoringAgreementTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringFactor: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"The factor, from invoice_get, list_factoring_noas or list_factoring_ready_invoices.",
		),
	}, paramFactoringFactor)
}

func (t *getFactoringAgreementTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFactoringAgreement})
}

func (t *getFactoringAgreementTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	factorID, err := requirePulid(params.Params, paramFactoringFactor)
	if err != nil {
		return nil, err
	}
	agreements, err := t.agreements.List(ctx, factoringports.ListAgreementsRequest{
		TenantInfo:         tenantOf(params),
		FactoringCompanyID: factorID,
	})
	if err != nil {
		return nil, err
	}
	now := t.now()
	rows := make([]factoringAgreementRow, 0, len(agreements))
	for _, a := range agreements {
		if a == nil {
			continue
		}
		rows = append(rows, factoringAgreementRowFrom(a, now))
	}
	return map[string]any{"agreements": rows, "count": len(rows)}, nil
}

func factoringAgreementRowFrom(a *factoring.Agreement, now int64) factoringAgreementRow {
	row := factoringAgreementRow{
		ID:                   a.ID.String(),
		Status:               string(a.Status),
		InForce:              a.InForce(now),
		EffectiveFrom:        recordedDate(a.EffectiveFrom),
		EffectiveTo:          pointerDateOr(a.EffectiveTo, absentOpenEnded),
		AdvanceRateBP:        a.AdvanceRateBP,
		ReserveRateBP:        a.ReserveRateBP,
		MinimumFeeMinor:      a.MinimumFeeMinor,
		ChargebackWindowDays: a.ChargebackWindowDays,
		Recourse:             string(a.Recourse),
		AccountingTreatment:  string(a.AccountingTreatment),
		FundingDays:          a.FundingDays,
		SubmissionTime:       clockOf(a.SubmissionTime),
		FeeTiers:             make([]factoringFeeTierRow, 0, len(a.FeeTiers)),
		RequiredDocuments:    make([]string, 0, len(a.DocumentTypes)),
	}
	for _, tier := range factoring.SortedFeeTiers(a.FeeTiers) {
		daysTo := absentOpenEnded
		if tier.DaysTo != nil {
			daysTo = strconv.Itoa(*tier.DaysTo)
		}
		row.FeeTiers = append(row.FeeTiers, factoringFeeTierRow{
			DaysFrom:  tier.DaysFrom,
			DaysTo:    daysTo,
			RateBP:    tier.RateBP,
			FlatMinor: tier.FlatMinor,
		})
	}
	for _, dt := range a.DocumentTypes {
		if dt != nil {
			row.RequiredDocuments = append(row.RequiredDocuments, dt.Name)
		}
	}
	return row
}

type factoringNoaRow struct {
	ID                 string       `json:"id"`
	CustomerID         string       `json:"customerId"`
	FactoringCompanyID string       `json:"factoringCompanyId"`
	Status             string       `json:"status"`
	Effective          bool         `json:"effective"`
	SentAt             optionalDate `json:"sentAt"`
	SentTo             string       `json:"sentTo,omitempty"`
	AcknowledgedAt     optionalDate `json:"acknowledgedAt"`
	ExpiresAt          optionalDate `json:"expiresAt"`
	ReleasedAt         optionalDate `json:"releasedAt"`
	ReleaseReason      string       `json:"releaseReason,omitempty"`
	Version            int64        `json:"version"`
}

type listFactoringNoasTool struct {
	noas factoringports.NoaService
	now  func() int64
}

func provideListFactoringNoasTool(noas factoringports.NoaService) agentports.AgentQueryTool {
	return &listFactoringNoasTool{noas: noas, now: timeutils.NowUnix}
}

func (t *listFactoringNoasTool) Name() string { return "list_factoring_noas" }

func (t *listFactoringNoasTool) Description() string {
	return "List notices of assignment, the letters telling customers to pay a factor. Each " +
		"says who it is for, whether it was sent, acknowledged with a signed copy, or " +
		"released, and when it expires. Effective is true only for an acknowledged, unexpired notice; an " +
		"invoice is not ready to factor without one. Pass id and version to " +
		"send_factoring_noa, record_factoring_noa_acknowledgement or release_factoring_noa."
}

func (t *listFactoringNoasTool) SearchTerms() []string {
	return []string{"notice of assignment", "noa", "factoring letter"}
}

func (t *listFactoringNoasTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramCustomerID: agenttoolschema.RecordIDText(
			permission.ResourceCustomer,
			"Only this customer's notices, from list_customers or invoice_get.",
		),
		paramFactoringFactor: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"Only notices for this factor, from invoice_get or list_factoring_ready_invoices.",
		),
		paramFactoringStatus: jsonschemautils.DescribedArray(
			"Only notices in these statuses.",
			agenttoolschema.Enum("A notice status.", agenttoolschema.FactoringNoaStatuses),
			len(factoring.AllNoaStatuses()),
		),
		paramLimit: jsonschemautils.Integer(fmt.Sprintf(
			"How many notices to return, at most %d. Defaults to %d.",
			factoringNoasMaxLimit, factoringNoasDefaultLimit,
		)),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	})
}

func (t *listFactoringNoasTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFactoringNoa})
}

func (t *listFactoringNoasTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	customerID, err := optionalID(params.Params, paramCustomerID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramCustomerID, err)
	}
	factorID, err := optionalID(params.Params, paramFactoringFactor)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramFactoringFactor, err)
	}
	statuses, err := optionalEnums(params.Params, paramFactoringStatus, factoring.NoaStatus.IsValid)
	if err != nil {
		return nil, err
	}
	limit := min(
		max(optionalInt(params.Params, paramLimit, factoringNoasDefaultLimit), 1),
		factoringNoasMaxLimit,
	)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}

	page, err := t.noas.List(ctx, &factoringports.ListNoaConnectionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: limit},
		},
		Cursor:             cursor,
		CustomerID:         customerID,
		FactoringCompanyID: factorID,
		Statuses:           statuses,
	})
	if err != nil {
		return nil, err
	}
	now := t.now()
	rows := make([]factoringNoaRow, 0, len(page.Items))
	for _, noa := range page.Items {
		if noa == nil {
			continue
		}
		rows = append(rows, factoringNoaRowFrom(noa, now))
	}
	next, err := page.NextCursor()
	if err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}
	return map[string]any{
		"notices":         rows,
		pageKeyHasMore:    page.HasNextPage,
		pageKeyNextCursor: next,
	}, nil
}

func factoringNoaRowFrom(noa *factoring.Noa, now int64) factoringNoaRow {
	return factoringNoaRow{
		ID:                 noa.ID.String(),
		CustomerID:         noa.CustomerID.String(),
		FactoringCompanyID: noa.FactoringCompanyID.String(),
		Status:             string(noa.Status),
		Effective:          noa.IsEffective(now),
		SentAt:             pointerDateOr(noa.SentAt, absentNotSent),
		SentTo:             noa.SentTo,
		AcknowledgedAt:     pointerDateOr(noa.AcknowledgedAt, absentNotAcknowledged),
		ExpiresAt:          pointerDateOr(noa.ExpiresAt, absentNoExpiry),
		ReleasedAt:         pointerDateOr(noa.ReleasedAt, absentNotReleased),
		ReleaseReason:      noa.ReleaseReason,
		Version:            noa.Version,
	}
}

func failedOnly(checks []factoringCheckRow) []factoringCheckRow {
	out := make([]factoringCheckRow, 0, len(checks))
	for _, c := range checks {
		if !c.Passed {
			out = append(out, c)
		}
	}
	return out
}

func clockOf(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

func factoringToolProviders() []any {
	return []any{
		provideGetFactoringReadinessTool,
		provideListFactoringReadyInvoicesTool,
		provideGetFactoringAgreementTool,
		provideListFactoringNoasTool,
		provideListFactoringSubmissionsTool,
		provideGetFactoringSubmissionTool,
		provideGetFactoringReserveBalanceTool,
		provideListFactoringRemittancesTool,
		provideGetFactoringRemittanceTool,
	}
}
