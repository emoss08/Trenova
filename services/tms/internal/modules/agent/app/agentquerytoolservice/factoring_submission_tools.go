package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/shopspring/decimal"
)

const (
	paramFactoringSubmission = "submissionId"

	factoringSubmissionsDefaultLimit = 25
	factoringSubmissionsMaxLimit     = 50

	absentNotSentYet = "not sent"
	absentNotFailed  = "not failed"
)

type factoringSubmissionItemRow struct {
	InvoiceID       string `json:"invoiceId"`
	InvoiceNumber   string `json:"invoiceNumber"`
	CustomerID      string `json:"customerId"`
	Face            string `json:"face,omitempty"`
	ExpectedAdvance string `json:"expectedAdvance,omitempty"`
	ExpectedFee     string `json:"expectedFee,omitempty"`
	ExpectedReserve string `json:"expectedReserve,omitempty"`
	ExpectedDays    int    `json:"expectedDays"`
	Status          string `json:"status"`
	HasPacket       bool   `json:"hasPacket"`
}

type factoringSubmissionRow struct {
	ID                 string                       `json:"id"`
	Number             string                       `json:"number"`
	FactoringCompanyID string                       `json:"factoringCompanyId"`
	Status             string                       `json:"status"`
	CurrencyCode       string                       `json:"currencyCode"`
	InvoiceCount       int                          `json:"invoiceCount"`
	Face               string                       `json:"face,omitempty"`
	ExpectedAdvance    string                       `json:"expectedAdvance,omitempty"`
	ExpectedFee        string                       `json:"expectedFee,omitempty"`
	ExpectedReserve    string                       `json:"expectedReserve,omitempty"`
	TransportMethod    string                       `json:"transportMethod"`
	TransportTarget    string                       `json:"transportTarget"`
	SentAt             optionalDate                 `json:"sentAt"`
	FailedAt           optionalDate                 `json:"failedAt"`
	LastError          string                       `json:"lastError,omitempty"`
	Attempts           int                          `json:"attempts"`
	CancelReason       string                       `json:"cancelReason,omitempty"`
	SubmittedAt        optionalDate                 `json:"submittedAt"`
	Version            int64                        `json:"version"`
	Items              []factoringSubmissionItemRow `json:"items,omitempty"`
}

func minorAmount(show amounts, minor int64) string {
	return show.of(decimal.New(minor, minorUnitsExponent))
}

func factoringSubmissionRowFrom(sub *factoring.Submission, show amounts) factoringSubmissionRow {
	row := factoringSubmissionRow{
		ID:                 sub.ID.String(),
		Number:             sub.Number,
		FactoringCompanyID: sub.FactoringCompanyID.String(),
		Status:             string(sub.Status),
		CurrencyCode:       sub.CurrencyCode,
		InvoiceCount:       sub.InvoiceCount,
		Face:               minorAmount(show, sub.FaceMinor),
		ExpectedAdvance:    minorAmount(show, sub.ExpectedAdvanceMinor),
		ExpectedFee:        minorAmount(show, sub.ExpectedFeeMinor),
		ExpectedReserve:    minorAmount(show, sub.ExpectedReserveMinor),
		TransportMethod:    sub.TransportMethod.String(),
		TransportTarget:    sub.TransportTarget,
		SentAt:             pointerDateOr(sub.SentAt, absentNotSentYet),
		FailedAt:           pointerDateOr(sub.FailedAt, absentNotFailed),
		LastError:          sub.LastError,
		Attempts:           sub.Attempts,
		CancelReason:       sub.CancelReason,
		SubmittedAt:        recordedDate(sub.CreatedAt),
		Version:            sub.Version,
	}
	if len(sub.Items) == 0 {
		return row
	}
	row.Items = make([]factoringSubmissionItemRow, 0, len(sub.Items))
	for _, item := range sub.Items {
		if item == nil {
			continue
		}
		row.Items = append(row.Items, factoringSubmissionItemRow{
			InvoiceID:       item.InvoiceID.String(),
			InvoiceNumber:   item.InvoiceNumber,
			CustomerID:      item.CustomerID.String(),
			Face:            minorAmount(show, item.FaceMinor),
			ExpectedAdvance: minorAmount(show, item.ExpectedAdvanceMinor),
			ExpectedFee:     minorAmount(show, item.ExpectedFeeMinor),
			ExpectedReserve: minorAmount(show, item.ExpectedReserveMinor),
			ExpectedDays:    item.ExpectedDays,
			Status:          string(item.Status),
			HasPacket:       !item.PacketDocumentID.IsNil(),
		})
	}
	return row
}

type listFactoringSubmissionsTool struct {
	submissions factoringports.SubmissionService
	access      fieldAccess
}

func provideListFactoringSubmissionsTool(
	submissions factoringports.SubmissionService,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listFactoringSubmissionsTool{
		submissions: submissions,
		access:      newFieldAccess(permissions),
	}
}

func (t *listFactoringSubmissionsTool) Name() string { return "list_factoring_submissions" }

func (t *listFactoringSubmissionsTool) Description() string {
	return "List the schedules of accounts sent to factors, newest first, with their status " +
		"and totals. Each has its number, factor, status (Sending, Sent, Failed or " +
		"Cancelled), invoice count, expected funding, where it went, and for a failed one " +
		"the last error. Pass a " +
		"schedule's id to get_factoring_submission for its invoices, and its id and version " +
		"to retry_factoring_submission or cancel_factoring_submission."
}

func (t *listFactoringSubmissionsTool) SearchTerms() []string {
	return []string{"factoring schedule", "schedule of accounts", "submitted to factor"}
}

func (t *listFactoringSubmissionsTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringFactor: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"Only schedules sent to this factor, from list_factoring_ready_invoices or get_invoice.",
		),
		paramFactoringStatus: jsonschemautils.DescribedArray(
			"Only schedules in these statuses.",
			agenttoolschema.Enum(
				"A schedule status.",
				agenttoolschema.FactoringSubmissionStatuses,
			),
			len(factoring.AllSubmissionStatuses()),
		),
		paramLimit: jsonschemautils.Integer(fmt.Sprintf(
			"How many schedules to return, at most %d. Defaults to %d.",
			factoringSubmissionsMaxLimit, factoringSubmissionsDefaultLimit,
		)),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	})
}

func (t *listFactoringSubmissionsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFactoringSubmission})
}

func (t *listFactoringSubmissionsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	factorID, err := optionalID(params.Params, paramFactoringFactor)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramFactoringFactor, err)
	}
	statuses, err := optionalEnums(
		params.Params,
		paramFactoringStatus,
		factoring.SubmissionStatus.IsValid,
	)
	if err != nil {
		return nil, err
	}
	limit := min(
		max(optionalInt(params.Params, paramLimit, factoringSubmissionsDefaultLimit), 1),
		factoringSubmissionsMaxLimit,
	)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}

	page, err := t.submissions.List(ctx, &factoringports.ListSubmissionConnectionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: limit},
		},
		Cursor:             cursor,
		FactoringCompanyID: factorID,
		Statuses:           statuses,
	})
	if err != nil {
		return nil, err
	}
	show, gate := amountsFor(ctx, t.access, params)
	rows := make([]factoringSubmissionRow, 0, len(page.Items))
	for _, sub := range page.Items {
		if sub == nil {
			continue
		}
		rows = append(rows, factoringSubmissionRowFrom(sub, show))
	}
	next, err := page.NextCursor()
	if err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}
	result := map[string]any{
		"schedules":  rows,
		"hasMore":    page.HasNextPage,
		"nextCursor": next,
		"queuePath":  factoringQueuePath,
	}
	if gate != nil {
		result["withheldByAccess"] = gate.Withheld()
	}
	return result, nil
}

type getFactoringSubmissionTool struct {
	submissions factoringports.SubmissionService
	access      fieldAccess
}

func provideGetFactoringSubmissionTool(
	submissions factoringports.SubmissionService,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getFactoringSubmissionTool{
		submissions: submissions,
		access:      newFieldAccess(permissions),
	}
}

func (t *getFactoringSubmissionTool) Name() string { return "get_factoring_submission" }

func (t *getFactoringSubmissionTool) Description() string {
	return "Read one schedule of accounts with every invoice on it and why it failed, if it " +
		"did. Each invoice has its amount, expected advance, fee and reserve, and whether its " +
		"packet was built; the schedule has its status, where it went, attempts and last " +
		"error. Use its version with " +
		"retry_factoring_submission or cancel_factoring_submission."
}

func (t *getFactoringSubmissionTool) SearchTerms() []string {
	return []string{"factoring schedule", "schedule of accounts", "failed schedule"}
}

func (t *getFactoringSubmissionTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringSubmission: agenttoolschema.RecordIDText(
			permission.ResourceFactoringSubmission,
			"The schedule, from list_factoring_submissions.",
		),
	}, paramFactoringSubmission)
}

func (t *getFactoringSubmissionTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFactoringSubmission})
}

func (t *getFactoringSubmissionTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	id, err := requirePulid(params.Params, paramFactoringSubmission)
	if err != nil {
		return nil, err
	}
	sub, err := t.submissions.Get(ctx, factoringports.GetSubmissionRequest{
		ID:           id,
		TenantInfo:   tenantOf(params),
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	show, gate := amountsFor(ctx, t.access, params)
	row := factoringSubmissionRowFrom(sub, show)
	result := map[string]any{
		"schedule": row,
		"path":     factoringQueuePath + "/submissions/" + sub.ID.String(),
	}
	if gate != nil {
		result["withheldByAccess"] = gate.Withheld()
	}
	return result, nil
}
