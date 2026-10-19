package agentquerytoolservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const (
	paramFactoringRemittance  = "remittanceId"
	paramRemittanceStatus     = "statuses"
	paramRemittanceLineStatus = "matchStatuses"

	factoringRemittancesDefaultLimit = 25
	factoringRemittancesMaxLimit     = 50
	factoringRemittanceMaxLines      = factoring.MaxRemittanceLines

	absentNotStaged  = "not read yet"
	absentNotApplied = "not applied"
	absentNoDate     = "no date on the line"
)

type factoringRemittanceSuggestionRow struct {
	ItemID        string `json:"itemId"`
	SubmissionID  string `json:"submissionId"`
	InvoiceNumber string `json:"invoiceNumber"`
	Score         int    `json:"score"`
	Reason        string `json:"reason"`
}

type factoringRemittanceLineRow struct {
	LineID           string                             `json:"lineId"`
	RowNumber        int                                `json:"rowNumber"`
	Kind             string                             `json:"kind"`
	InvoiceNumber    string                             `json:"invoiceNumber,omitempty"`
	FactorReference  string                             `json:"factorReference,omitempty"`
	Debtor           string                             `json:"debtor,omitempty"`
	Face             string                             `json:"face,omitempty"`
	Advance          string                             `json:"advance,omitempty"`
	Fee              string                             `json:"fee,omitempty"`
	Reserve          string                             `json:"reserve,omitempty"`
	Cash             string                             `json:"cash,omitempty"`
	OccurredOn       optionalDate                       `json:"occurredOn"`
	MatchStatus      string                             `json:"matchStatus"`
	MatchedBy        string                             `json:"matchedBy,omitempty"`
	ItemID           string                             `json:"itemId,omitempty"`
	MatchedInvoice   string                             `json:"matchedInvoice,omitempty"`
	MatchedSchedule  string                             `json:"matchedSubmissionId,omitempty"`
	Score            int                                `json:"score"`
	Reason           string                             `json:"reason,omitempty"`
	Suggestions      []factoringRemittanceSuggestionRow `json:"suggestions,omitempty"`
	Error            string                             `json:"error,omitempty"`
	IgnoredReason    string                             `json:"ignoredReason,omitempty"`
	BookedAsEntryID  string                             `json:"reserveEntryId,omitempty"`
	CellsInTheReport []string                           `json:"cells,omitempty"`
}

type factoringRemittanceRow struct {
	ID                 string                       `json:"id"`
	FactoringCompanyID string                       `json:"factoringCompanyId"`
	FileName           string                       `json:"fileName,omitempty"`
	Source             string                       `json:"source"`
	Format             string                       `json:"format,omitempty"`
	Status             string                       `json:"status"`
	DefaultKind        string                       `json:"defaultKind"`
	RemittedOn         optionalDate                 `json:"remittedOn"`
	CurrencyCode       string                       `json:"currencyCode"`
	LineCount          int                          `json:"lineCount"`
	MatchedCount       int                          `json:"matchedCount"`
	ExceptionCount     int                          `json:"exceptionCount"`
	AppliedCount       int                          `json:"appliedCount"`
	TotalCash          string                       `json:"totalCash,omitempty"`
	Error              string                       `json:"error,omitempty"`
	StagedAt           optionalDate                 `json:"stagedAt"`
	AppliedAt          optionalDate                 `json:"appliedAt"`
	DiscardReason      string                       `json:"discardReason,omitempty"`
	UnmappedHeaders    []string                     `json:"unmappedHeaders,omitempty"`
	Version            int64                        `json:"version"`
	Lines              []factoringRemittanceLineRow `json:"lines,omitempty"`
}

func optionalMinor(show amounts, minor *int64) string {
	if minor == nil {
		return ""
	}
	return minorAmount(show, *minor)
}

func factoringRemittanceRowFrom(rem *factoring.Remittance, show amounts) factoringRemittanceRow {
	return factoringRemittanceRow{
		ID:                 rem.ID.String(),
		FactoringCompanyID: rem.FactoringCompanyID.String(),
		FileName:           rem.FileName,
		Source:             rem.Source.String(),
		Format:             rem.SourceFormat.String(),
		Status:             rem.Status.String(),
		DefaultKind:        rem.DefaultKind.String(),
		RemittedOn:         pointerDateOr(rem.RemittedOn, absentNoDate),
		CurrencyCode:       rem.CurrencyCode,
		LineCount:          rem.LineCount,
		MatchedCount:       rem.MatchedCount,
		ExceptionCount:     rem.ExceptionCount,
		AppliedCount:       rem.AppliedCount,
		TotalCash:          minorAmount(show, rem.TotalCashMinor),
		Error:              rem.Error,
		StagedAt:           pointerDateOr(rem.StagedAt, absentNotStaged),
		AppliedAt:          pointerDateOr(rem.AppliedAt, absentNotApplied),
		DiscardReason:      rem.DiscardReason,
		UnmappedHeaders:    rem.UnmappedHeaders,
		Version:            rem.Version,
	}
}

func factoringRemittanceLineRowFrom(
	line *factoring.RemittanceLine,
	show amounts,
) factoringRemittanceLineRow {
	row := factoringRemittanceLineRow{
		LineID:          line.ID.String(),
		RowNumber:       line.RowNumber,
		Kind:            line.Kind.String(),
		InvoiceNumber:   line.InvoiceNumber,
		FactorReference: line.FactorReference,
		Debtor:          line.DebtorName,
		Face:            optionalMinor(show, line.FaceMinor),
		Advance:         optionalMinor(show, line.AdvanceMinor),
		Fee:             optionalMinor(show, line.FeeMinor),
		Reserve:         optionalMinor(show, line.ReserveMinor),
		Cash:            optionalMinor(show, line.CashMinor),
		OccurredOn:      pointerDateOr(line.OccurredOn, absentNoDate),
		MatchStatus:     line.MatchStatus.String(),
		MatchedBy:       line.MatchedBy.String(),
		MatchedInvoice:  line.MatchedInvoice,
		Score:           line.Score,
		Reason:          line.Reason.String(),
		Error:           line.Error,
		IgnoredReason:   line.IgnoredReason,
	}
	if line.SubmissionItemID.IsNotNil() {
		row.ItemID = line.SubmissionItemID.String()
	}
	if line.MatchedSubmission.IsNotNil() {
		row.MatchedSchedule = line.MatchedSubmission.String()
	}
	if line.ReserveEntryID.IsNotNil() {
		row.BookedAsEntryID = line.ReserveEntryID.String()
	}
	if line.MatchStatus == factoring.RemittanceLineError {
		row.CellsInTheReport = line.Cells
	}
	for _, suggestion := range line.Suggestions {
		if suggestion == nil {
			continue
		}
		row.Suggestions = append(row.Suggestions, factoringRemittanceSuggestionRow{
			ItemID:        suggestion.ItemID.String(),
			SubmissionID:  suggestion.SubmissionID.String(),
			InvoiceNumber: suggestion.InvoiceNumber,
			Score:         suggestion.Score,
			Reason:        suggestion.Reason.String(),
		})
	}
	return row
}

func factoringRemittancePath(id string) string {
	return factoringQueuePath + "/remittances/" + id
}

type listFactoringRemittancesTool struct {
	remittances factoringports.RemittanceService
	access      fieldAccess
}

func provideListFactoringRemittancesTool(
	remittances factoringports.RemittanceService,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listFactoringRemittancesTool{
		remittances: remittances,
		access:      newFieldAccess(permissions),
	}
}

func (t *listFactoringRemittancesTool) Name() string { return "factoring_list_remittances" }

func (t *listFactoringRemittancesTool) Description() string {
	return "List factor remittances, newest first, with how far each has been read and booked. " +
		"Each has its factor, file, status (Pending, NeedsMapping, Parsed, Failed, " +
		"PartiallyApplied, Applied or Discarded), line, matched, exception and applied " +
		"counts, and why it failed if it did. Pass one's id to factoring_get_remittance for " +
		"its lines."
}

func (t *listFactoringRemittancesTool) SearchTerms() []string {
	return []string{"factor remittance", "funding report", "remittance exceptions"}
}

func (t *listFactoringRemittancesTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringFactor: agenttoolschema.RecordIDText(
			permission.ResourceFactoringCompany,
			"Only remittances from this factor, from factoring_get_remittance or "+
				"factoring_list_readiness.",
		),
		paramRemittanceStatus: jsonschemautils.DescribedArray(
			"Only remittances in these statuses.",
			agenttoolschema.Enum(
				"A remittance status.",
				agenttoolschema.FactoringRemittanceStatuses,
			),
			len(factoring.AllRemittanceStatuses()),
		),
		paramLimit: limitSchema(factoringRemittancesDefaultLimit, factoringRemittancesMaxLimit),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	})
}

func (t *listFactoringRemittancesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFactoringRemittance})
}

func (t *listFactoringRemittancesTool) Query(
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
		paramRemittanceStatus,
		factoring.RemittanceStatus.IsValid,
	)
	if err != nil {
		return nil, err
	}
	limit := min(
		max(optionalInt(params.Params, paramLimit, factoringRemittancesDefaultLimit), 1),
		factoringRemittancesMaxLimit,
	)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}
	page, err := t.remittances.ListConnection(ctx, &factoringports.ListRemittanceConnectionRequest{
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
	rows := make([]factoringRemittanceRow, 0, len(page.Items))
	for _, rem := range page.Items {
		if rem != nil {
			rows = append(rows, factoringRemittanceRowFrom(rem, show))
		}
	}
	next, err := page.NextCursor()
	if err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}
	result := map[string]any{
		"remittances":     rows,
		pageKeyHasMore:    page.HasNextPage,
		pageKeyNextCursor: next,
		"queuePath":       factoringQueuePath,
	}
	if gate != nil {
		result["withheldByAccess"] = gate.Withheld()
	}
	return result, nil
}

type getFactoringRemittanceTool struct {
	remittances factoringports.RemittanceService
	access      fieldAccess
}

func provideGetFactoringRemittanceTool(
	remittances factoringports.RemittanceService,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getFactoringRemittanceTool{
		remittances: remittances,
		access:      newFieldAccess(permissions),
	}
}

func (t *getFactoringRemittanceTool) Name() string { return "factoring_get_remittance" }

func (t *getFactoringRemittanceTool) Description() string {
	return "Read one factor remittance with its lines and how each was matched. Each line has " +
		"its row, kind, invoice number, amounts and date as the factor printed them, its " +
		"match status, the schedule item it is matched to, and up to three ranked suggestions " +
		"with why. Use its version and a line id with factoring_resolve_remittance_line, and " +
		"its version with factoring_apply_remittance or factoring_discard_remittance."
}

func (t *getFactoringRemittanceTool) SearchTerms() []string {
	return []string{"factor remittance", "remittance lines", "remittance exceptions"}
}

func (t *getFactoringRemittanceTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramFactoringRemittance: agenttoolschema.RecordIDText(
			permission.ResourceFactoringRemittance,
			"The remittance, from factoring_list_remittances.",
		),
		paramRemittanceLineStatus: jsonschemautils.DescribedArray(
			"Only lines in these match statuses; leave out for every line.",
			agenttoolschema.Enum(
				"A line's match status.",
				agenttoolschema.FactoringRemittanceMatchStatuses,
			),
			len(factoring.AllRemittanceMatchStatuses()),
		),
	}, paramFactoringRemittance)
}

func (t *getFactoringRemittanceTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFactoringRemittance})
}

func (t *getFactoringRemittanceTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	id, err := requirePulid(params.Params, paramFactoringRemittance)
	if err != nil {
		return nil, err
	}
	statuses, err := optionalEnums(
		params.Params,
		paramRemittanceLineStatus,
		factoring.RemittanceMatchStatus.IsValid,
	)
	if err != nil {
		return nil, err
	}
	rem, err := t.remittances.Get(ctx, factoringports.GetRemittanceRequest{
		ID:           id,
		TenantInfo:   tenantOf(params),
		IncludeLines: true,
	})
	if err != nil {
		return nil, err
	}
	show, gate := amountsFor(ctx, t.access, params)
	row := factoringRemittanceRowFrom(rem, show)
	row.Lines = make(
		[]factoringRemittanceLineRow,
		0,
		min(len(rem.Lines), factoringRemittanceMaxLines),
	)
	for _, line := range rem.Lines {
		if line == nil || (len(statuses) > 0 && !slices.Contains(statuses, line.MatchStatus)) {
			continue
		}
		row.Lines = append(row.Lines, factoringRemittanceLineRowFrom(line, show))
	}
	result := map[string]any{
		"remittance": row,
		"path":       factoringRemittancePath(rem.ID.String()),
	}
	if gate != nil {
		result["withheldByAccess"] = gate.Withheld()
	}
	return result, nil
}
