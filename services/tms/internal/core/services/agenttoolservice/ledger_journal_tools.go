package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/journalreversalservice"
	"github.com/emoss08/trenova/internal/core/services/manualjournalservice"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramManualJournalID       = "manualJournalId"
	paramJournalReversalID     = "journalReversalId"
	paramJournalEntryID        = "journalEntryId"
	paramJournalDescription    = "description"
	paramJournalLines          = "lines"
	paramJournalCurrency       = "currencyCode"
	paramJournalLineDebit      = "debit"
	paramJournalLineCredit     = "credit"
	paramJournalLineLocationID = "locationId"
	paramReversalReasonCode    = "reasonCode"
	paramReversalReasonText    = "reasonText"

	maxManualJournalLines        = 100
	maxManualJournalDescription  = 500
	maxManualJournalReason       = 1000
	maxManualJournalLineText     = 255
	maxJournalCancelReason       = 1000
	maxReversalReasonCode        = 100
	maxReversalReasonText        = 1000
	manualJournalKind            = "manual journal"
	manualJournalRecordEntity    = "manual_journal"
	journalReversalRecordEntity  = "journal_reversal"
	journalReversalKind          = "journal reversal"
	toolListManualJournals       = "list_manual_journals"
	toolGetManualJournal         = "get_manual_journal"
	toolListJournalReversals     = "list_journal_reversals"
	manualJournalSupplier        = "The manual journal, from list_manual_journals or get_manual_journal. Never guess one."
	journalReversalSupplier      = "The journal reversal, from list_journal_reversals or the result of request_journal_reversal. Never guess one."
	manualJournalCurrencyPattern = 3
)

var (
	errJournalLineSide = errors.New(
		"each line takes exactly one of debit or credit, greater than zero",
	)
	errAgentCannotApproveJournal = errors.New(
		"manual journal approval is turned off, so submitting approves the journal; only a " +
			"person submits it",
	)
)

type ledgerAccounts interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetGLAccountsByIDsRequest,
	) ([]*glaccount.GLAccount, error)
}

type manualJournalKeeper interface {
	Get(
		ctx context.Context,
		req *serviceports.GetManualJournalRequest,
	) (*manualjournal.Request, error)
	PlanCreateDraft(
		ctx context.Context,
		req *serviceports.CreateManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*manualjournal.Request, error)
	CreateDraft(
		ctx context.Context,
		req *serviceports.CreateManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*manualjournal.Request, error)
	PlanUpdateDraft(
		ctx context.Context,
		req *serviceports.UpdateManualJournalDraftRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ManualJournalChange, error)
	UpdateDraft(
		ctx context.Context,
		req *serviceports.UpdateManualJournalDraftRequest,
		actor *serviceports.RequestActor,
	) (*manualjournal.Request, error)
	PlanSubmit(
		ctx context.Context,
		req *serviceports.GetManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ManualJournalChange, error)
	Submit(
		ctx context.Context,
		req *serviceports.GetManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*manualjournal.Request, error)
	PlanCancel(
		ctx context.Context,
		req *serviceports.CancelManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ManualJournalChange, error)
	Cancel(
		ctx context.Context,
		req *serviceports.CancelManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*manualjournal.Request, error)
	PlanPost(
		ctx context.Context,
		req *serviceports.GetManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ManualJournalChange, error)
	Post(
		ctx context.Context,
		req *serviceports.GetManualJournalRequest,
		actor *serviceports.RequestActor,
	) (*manualjournal.Request, error)
}

type journalReversalKeeper interface {
	PlanCreate(
		ctx context.Context,
		req *serviceports.CreateJournalReversalRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.JournalReversalChange, error)
	Create(
		ctx context.Context,
		req *serviceports.CreateJournalReversalRequest,
		actor *serviceports.RequestActor,
	) (*journalreversal.Reversal, error)
	PlanCancel(
		ctx context.Context,
		req *serviceports.CancelJournalReversalRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.JournalReversalChange, error)
	Cancel(
		ctx context.Context,
		req *serviceports.CancelJournalReversalRequest,
		actor *serviceports.RequestActor,
	) (*journalreversal.Reversal, error)
	PlanPost(
		ctx context.Context,
		req *serviceports.GetJournalReversalRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.JournalReversalChange, error)
	Post(
		ctx context.Context,
		req *serviceports.GetJournalReversalRequest,
		actor *serviceports.RequestActor,
	) (*journalreversal.Reversal, error)
}

var (
	_ manualJournalKeeper   = (*manualjournalservice.Service)(nil)
	_ journalReversalKeeper = (*journalreversalservice.Service)(nil)
)

func targetManualJournal(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramManualJournalID, permission.ResourceManualJournal)
}

func targetJournalReversal(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramJournalReversalID, permission.ResourceJournalReversal)
}

func targetReversedEntry(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramJournalEntryID, permission.ResourceJournalEntry)
}

func manualJournalDraftProperties(forRevision bool) map[string]any {
	lines := journalLinesProperty()
	description := "What the journal records, as the approver will read it."
	date := "The day it is booked to the ledger; it must fall in a fiscal period that takes " +
		"manual journals."
	if forRevision {
		lines[toolschema.KeyDescription] = "Every line of the rewritten journal, replacing " +
			"the draft's lines. Leave it out to keep the lines as they are."
		description += " Leave it out to keep it."
		date += " Leave it out to keep it."
	}

	properties := map[string]any{
		paramJournalDescription: stringProperty(description, maxManualJournalDescription),
		paramReason: stringProperty("Why the entry is needed: the evidence or the "+
			"correction it makes.", maxManualJournalReason),
		paramAccountingDate: agenttoolschema.Date(date),
		paramJournalCurrency: stringProperty("The ISO currency code, such as USD. Leave it "+
			"out for the organization's functional currency.", manualJournalCurrencyPattern),
		paramJournalLines: lines,
	}
	if forRevision {
		properties[paramManualJournalID] = stringProperty(manualJournalSupplier, 0)
	}

	return properties
}

func journalLinesProperty() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeArray,
		toolschema.KeyDescription: "The journal's lines, at least two, whose debits and " +
			"credits must balance before it can be submitted.",
		toolschema.KeyMinItems: 2,
		toolschema.KeyMaxItems: maxManualJournalLines,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				paramGLAccountID: stringProperty("The account the line posts to, from "+
					"list_gl_accounts; it must be active and allow manual journals.", 0),
				paramJournalDescription: stringProperty("What the line is for.",
					maxManualJournalLineText),
				paramJournalLineDebit: stringProperty("The debit as a decimal such as "+
					"125.00. Give a debit or a credit, never both.", 0),
				paramJournalLineCredit: stringProperty("The credit as a decimal such as "+
					"125.00. Give a debit or a credit, never both.", 0),
				paramCustomerID: stringProperty("The customer the line is for, from "+
					"list_customers or get_customer, when it concerns one.", 0),
				paramJournalLineLocationID: stringProperty("The location the line is for, "+
					"from list_locations, when it concerns one.", 0),
			},
			toolschema.KeyRequired: []string{
				paramGLAccountID,
				paramJournalDescription,
			},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func readJournalLines(params map[string]any) ([]*serviceports.ManualJournalLineInput, error) {
	raw, ok := params[paramJournalLines].([]any)
	if !ok || len(raw) < 2 {
		return nil, fmt.Errorf("parameter %q must list at least two lines", paramJournalLines)
	}
	if len(raw) > maxManualJournalLines {
		return nil, fmt.Errorf(
			"parameter %q holds %d lines; a manual journal takes at most %d",
			paramJournalLines, len(raw), maxManualJournalLines,
		)
	}

	lines := make([]*serviceports.ManualJournalLineInput, 0, len(raw))
	for idx, item := range raw {
		fields, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", paramJournalLines, idx)
		}
		line, err := readJournalLine(fields)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramJournalLines, idx, err)
		}
		lines = append(lines, line)
	}

	return lines, nil
}

func readJournalLine(fields map[string]any) (*serviceports.ManualJournalLineInput, error) {
	accountID, err := requirePulid(fields, paramGLAccountID)
	if err != nil {
		return nil, err
	}
	description, err := requireBoundedText(
		fields,
		paramJournalDescription,
		maxManualJournalLineText,
	)
	if err != nil {
		return nil, err
	}
	debit, hasDebit, err := optionalMoney(fields, paramJournalLineDebit)
	if err != nil {
		return nil, err
	}
	credit, hasCredit, err := optionalMoney(fields, paramJournalLineCredit)
	if err != nil {
		return nil, err
	}
	if hasDebit == hasCredit {
		return nil, errJournalLineSide
	}
	customerID, err := optionalPulidParam(fields, paramCustomerID)
	if err != nil {
		return nil, err
	}
	locationID, err := optionalPulidParam(fields, paramJournalLineLocationID)
	if err != nil {
		return nil, err
	}

	line := &serviceports.ManualJournalLineInput{
		GLAccountID:  accountID,
		Description:  description,
		DebitAmount:  debit,
		CreditAmount: credit,
	}
	if customerID != nil {
		line.CustomerID = *customerID
	}
	if locationID != nil {
		line.LocationID = *locationID
	}

	return line, nil
}

func readJournalCurrency(params map[string]any) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(optionalString(params, paramJournalCurrency)))
	if code != "" && len(code) != manualJournalCurrencyPattern {
		return "", fmt.Errorf("%s must be a three-letter code such as USD", paramJournalCurrency)
	}

	return code, nil
}

func draftManualJournalRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.CreateManualJournalRequest, error) {
	description, err := requireBoundedText(
		params.Params,
		paramJournalDescription,
		maxManualJournalDescription,
	)
	if err != nil {
		return nil, err
	}
	reason, err := boundedText(params.Params, paramReason, maxManualJournalReason)
	if err != nil {
		return nil, err
	}
	accountingDate, err := requireDay(params.Params, paramAccountingDate)
	if err != nil {
		return nil, err
	}
	currency, err := readJournalCurrency(params.Params)
	if err != nil {
		return nil, err
	}
	lines, err := readJournalLines(params.Params)
	if err != nil {
		return nil, err
	}

	return &serviceports.CreateManualJournalRequest{
		Description:    description,
		Reason:         reason,
		AccountingDate: accountingDate,
		CurrencyCode:   currency,
		Lines:          lines,
		TenantInfo:     tenantFrom(*params),
	}, nil
}

type journalRevision struct {
	id          pulid.ID
	description string
	reason      string
	hasReason   bool
	date        int64
	currency    string
	lines       []*serviceports.ManualJournalLineInput
}

func journalRevisionFrom(params *serviceports.ToolExecuteParams) (*journalRevision, error) {
	id, err := requirePulid(params.Params, paramManualJournalID)
	if err != nil {
		return nil, err
	}
	revision := &journalRevision{id: id}
	if revision.description, err = boundedText(
		params.Params,
		paramJournalDescription,
		maxManualJournalDescription,
	); err != nil {
		return nil, err
	}
	_, revision.hasReason = params.Params[paramReason]
	if revision.reason, err = boundedText(
		params.Params,
		paramReason,
		maxManualJournalReason,
	); err != nil {
		return nil, err
	}
	if revision.date, _, err = optionalDay(params.Params, paramAccountingDate); err != nil {
		return nil, err
	}
	if revision.currency, err = readJournalCurrency(params.Params); err != nil {
		return nil, err
	}
	if _, given := params.Params[paramJournalLines]; given {
		if revision.lines, err = readJournalLines(params.Params); err != nil {
			return nil, err
		}
	}

	return revision, nil
}

func (r *journalRevision) over(
	current *manualjournal.Request,
	params *serviceports.ToolExecuteParams,
) *serviceports.UpdateManualJournalDraftRequest {
	req := &serviceports.UpdateManualJournalDraftRequest{
		RequestID:      current.ID,
		Description:    current.Description,
		Reason:         current.Reason,
		AccountingDate: current.AccountingDate,
		CurrencyCode:   current.CurrencyCode,
		Lines:          linesOf(current),
		TenantInfo:     tenantFrom(*params),
	}
	if r.description != "" {
		req.Description = r.description
	}
	if r.hasReason {
		req.Reason = r.reason
	}
	if r.date != 0 {
		req.AccountingDate = r.date
	}
	if r.currency != "" {
		req.CurrencyCode = r.currency
	}
	if r.lines != nil {
		req.Lines = r.lines
	}

	return req
}

func linesOf(current *manualjournal.Request) []*serviceports.ManualJournalLineInput {
	lines := make([]*serviceports.ManualJournalLineInput, 0, len(current.Lines))
	for _, line := range current.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, &serviceports.ManualJournalLineInput{
			GLAccountID:  line.GLAccountID,
			Description:  line.Description,
			DebitAmount:  line.DebitAmount,
			CreditAmount: line.CreditAmount,
			CustomerID:   line.CustomerID,
			LocationID:   line.LocationID,
		})
	}

	return lines
}

func revisedDraft(
	ctx context.Context,
	journals manualJournalKeeper,
	revision *journalRevision,
	params *serviceports.ToolExecuteParams,
) (*serviceports.UpdateManualJournalDraftRequest, error) {
	current, err := journals.Get(ctx, &serviceports.GetManualJournalRequest{
		RequestID:  revision.id,
		TenantInfo: tenantFrom(*params),
	})
	if err != nil {
		return nil, err
	}

	return revision.over(current, params), nil
}

func manualJournalRequestFrom(
	params *serviceports.ToolExecuteParams,
) (*serviceports.GetManualJournalRequest, error) {
	id, err := requirePulid(params.Params, paramManualJournalID)
	if err != nil {
		return nil, err
	}

	return &serviceports.GetManualJournalRequest{
		RequestID:  id,
		TenantInfo: tenantFrom(*params),
	}, nil
}

func manualJournalResult(action string, entity *manualjournal.Request) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   manualJournalKind,
		Name:   entity.RequestNumber,
		IDs:    map[string]string{paramManualJournalID: entity.ID.String()},
		Record: recordOf(manualJournalRecordEntity, entity.ID),
	}
}

func journalDraftSpec(spec *receivableSpec) *receivableSpec {
	spec.resource = permission.ResourceManualJournal
	spec.artifact = manualJournalRecordEntity
	spec.egress = agent.EgressInternal
	spec.defaultTier = agent.TierPropose
	spec.maxTier = agent.TierActWithApproval
	spec.reversible = true

	return spec
}

func ledgerMoneySpec(spec *receivableSpec) *receivableSpec {
	spec.egress = agent.EgressMoney
	spec.defaultTier = agent.TierPropose
	spec.maxTier = agent.TierPropose
	spec.personOnly = true

	return spec
}

func labelledJournal(
	ctx context.Context,
	accounts ledgerAccounts,
	params *serviceports.ToolExecuteParams,
	change *serviceports.ManualJournalChange,
	err error,
) (*manualJournalPlan, error) {
	if err != nil {
		return nil, err
	}

	return planManualJournalChange(ctx, accounts, tenantFrom(*params), change)
}

func labelledReversal(
	ctx context.Context,
	accounts ledgerAccounts,
	params *serviceports.ToolExecuteParams,
	change *serviceports.JournalReversalChange,
	err error,
) (*reversalPlan, error) {
	if err != nil {
		return nil, err
	}

	return planReversal(ctx, accounts, tenantFrom(*params), change)
}

func newDraftManualJournalTool(
	journals manualJournalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReportingReceivableTool(journalDraftSpec(&receivableSpec{
		name: "draft_manual_journal",
		description: "Draft a manual journal entry for the general ledger: its lines, the " +
			"accounts they post to and the day it is booked. It is saved as a draft and books " +
			"nothing; submit_manual_journal sends it for approval. Take accounts from " +
			"list_gl_accounts and make the debits equal the credits.",
		operation: permission.OpCreate,
		taintHold: "A journal drafted from what someone outside wrote is proposed, since its " +
			"lines are what an approver books.",
		rationale: "Saves a draft journal inside Trenova; nothing reaches the ledger until a " +
			"person submits, approves and posts it, and a draft is cancelled the same way.",
		properties: manualJournalDraftProperties(false),
		required: []string{
			paramJournalDescription,
			paramAccountingDate,
			paramJournalLines,
		},
	}), receivablePlan[*serviceports.CreateManualJournalRequest, *manualJournalPlan]{
		request: draftManualJournalRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.CreateManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*manualJournalPlan, error) {
			planned, err := journals.PlanCreateDraft(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return planManualJournalDraft(ctx, accounts, req.TenantInfo, planned)
		},
		refused: func(req *serviceports.CreateManualJournalRequest) string {
			return fmt.Sprintf("Would draft the manual journal %q with %s.",
				req.Description, countOf(len(req.Lines), "line"))
		},
		render: renderManualJournalDraft,
		run: func(
			ctx context.Context,
			req *serviceports.CreateManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := journals.CreateDraft(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return manualJournalResult("drafted", created), nil
		},
	})
}

func newReviseManualJournalDraftTool(
	journals manualJournalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReportingReceivableTool(journalDraftSpec(&receivableSpec{
		name:        "revise_manual_journal_draft",
		searchTerms: []string{"change journal lines", "edit draft journal", "fix draft journal"},
		description: "Rewrite a draft manual journal before it is submitted: its " +
			"description, reason, day, currency or lines. Fields left out keep their value; " +
			"lines, when given, replace every line. Only a draft can be revised; a journal " +
			"sent back by its approver is drafted again with draft_manual_journal.",
		operation: permission.OpUpdate,
		taintHold: "A journal rewritten from what someone outside wrote is proposed, since its " +
			"lines are what an approver books.",
		rationale: "Changes a draft journal inside Trenova; nothing reaches the ledger until a " +
			"person submits, approves and posts it, and a later revision changes it back.",
		properties: manualJournalDraftProperties(true),
		required:   []string{paramManualJournalID},
		target:     targetManualJournal,
	}), receivablePlan[*journalRevision, *manualJournalPlan]{
		request: journalRevisionFrom,
		plan: func(
			ctx context.Context,
			revision *journalRevision,
			params *serviceports.ToolExecuteParams,
		) (*manualJournalPlan, error) {
			req, err := revisedDraft(ctx, journals, revision, params)
			if err != nil {
				return nil, err
			}
			change, err := journals.PlanUpdateDraft(ctx, req, params.Actor)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		refused: func(*journalRevision) string {
			return "Would revise a draft manual journal."
		},
		render: renderManualJournalRevision,
		run: func(
			ctx context.Context,
			revision *journalRevision,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := revisedDraft(ctx, journals, revision, params)
			if err != nil {
				return nil, err
			}
			updated, err := journals.UpdateDraft(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return manualJournalResult("revised", updated), nil
		},
	})
}

func plannedSubmit(
	ctx context.Context,
	journals manualJournalKeeper,
	req *serviceports.GetManualJournalRequest,
	params *serviceports.ToolExecuteParams,
) (*serviceports.ManualJournalChange, error) {
	change, err := journals.PlanSubmit(ctx, req, params.Actor)
	if err != nil {
		return nil, err
	}
	if params.Actor.IsAgent() && change.After.Status == manualjournal.StatusApproved {
		return nil, errAgentCannotApproveJournal
	}

	return change, nil
}

func newSubmitManualJournalTool(
	journals manualJournalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReceivableTool(journalDraftSpec(&receivableSpec{
		name: "submit_manual_journal",
		description: "Submit a balanced draft manual journal for approval. It waits for an " +
			"approver; where the organization has manual journal approval turned off, " +
			"submitting approves it, so only a person does that. Read it with " +
			"get_manual_journal first.",
		operation: permission.OpSubmit,
		rationale: "Moves a draft journal into the approval queue inside Trenova; an approver " +
			"decides and it books nothing until a person posts it.",
		properties: map[string]any{
			paramManualJournalID: stringProperty(manualJournalSupplier, 0),
		},
		required: []string{paramManualJournalID},
		target:   targetManualJournal,
	}), receivablePlan[*serviceports.GetManualJournalRequest, *manualJournalPlan]{
		request: manualJournalRequestFrom,
		plan: func(
			ctx context.Context,
			req *serviceports.GetManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*manualJournalPlan, error) {
			change, err := plannedSubmit(ctx, journals, req, params)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		refused: func(*serviceports.GetManualJournalRequest) string {
			return "Would submit a manual journal for approval."
		},
		render: renderManualJournalSubmit,
		run: func(
			ctx context.Context,
			req *serviceports.GetManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if _, err := plannedSubmit(ctx, journals, req, params); err != nil {
				return nil, err
			}
			_, err := journals.Submit(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func cancelManualJournalRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.CancelManualJournalRequest, error) {
	id, err := requirePulid(params.Params, paramManualJournalID)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxJournalCancelReason)
	if err != nil {
		return nil, err
	}

	return &serviceports.CancelManualJournalRequest{
		RequestID:  id,
		Reason:     reason,
		TenantInfo: tenantFrom(*params),
	}, nil
}

func newCancelManualJournalTool(
	journals manualJournalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReceivableTool(journalDraftSpec(&receivableSpec{
		name: "cancel_manual_journal",
		description: "Cancel a manual journal that is a draft, waiting for approval or " +
			"approved but not posted, with the reason. It is kept, cancelled, and never " +
			"posted. A posted journal is undone with request_journal_reversal instead.",
		operation: permission.OpCancel,
		taintHold: "The reason is what the journal's author reads next, so a run that has read " +
			"outside text proposes it.",
		rationale: "Withdraws a journal before it reaches the ledger; nothing is booked, and " +
			"the entry is drafted again if it was needed after all.",
		properties: map[string]any{
			paramManualJournalID: stringProperty(manualJournalSupplier, 0),
			paramReason: stringProperty("Why it must not be posted, in a sentence its author "+
				"can act on.", maxJournalCancelReason),
		},
		required: []string{paramManualJournalID, paramReason},
		target:   targetManualJournal,
	}), receivablePlan[*serviceports.CancelManualJournalRequest, *manualJournalPlan]{
		request: cancelManualJournalRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.CancelManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*manualJournalPlan, error) {
			change, err := journals.PlanCancel(ctx, req, params.Actor)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		refused: func(*serviceports.CancelManualJournalRequest) string {
			return "Would cancel a manual journal."
		},
		render: renderManualJournalCancel,
		run: func(
			ctx context.Context,
			req *serviceports.CancelManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := journals.Cancel(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func newPostManualJournalTool(
	journals manualJournalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReceivableTool(ledgerMoneySpec(&receivableSpec{
		name: "post_manual_journal",
		description: "Propose posting an approved manual journal to the general ledger, in " +
			"its fiscal period or the next open one the closed period policy allows. It " +
			"changes account balances and is undone only by a reversal, so a person always " +
			"decides.",
		resource:  permission.ResourceManualJournal,
		artifact:  manualJournalRecordEntity,
		operation: permission.OpApprove,
		rationale: "Books the journal's lines to the general ledger; only a person posts a " +
			"manual journal.",
		properties: map[string]any{
			paramManualJournalID: stringProperty(manualJournalSupplier, 0),
		},
		required: []string{paramManualJournalID},
		target:   targetManualJournal,
	}), receivablePlan[*serviceports.GetManualJournalRequest, *manualJournalPlan]{
		request: manualJournalRequestFrom,
		plan: func(
			ctx context.Context,
			req *serviceports.GetManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*manualJournalPlan, error) {
			change, err := journals.PlanPost(ctx, req, params.Actor)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		refused: func(*serviceports.GetManualJournalRequest) string {
			return "Would post a manual journal to the general ledger."
		},
		render: renderManualJournalPost,
		run: func(
			ctx context.Context,
			req *serviceports.GetManualJournalRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := journals.Post(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func journalReversalRequestFrom(
	params *serviceports.ToolExecuteParams,
) (*serviceports.CreateJournalReversalRequest, error) {
	entryID, err := requirePulid(params.Params, paramJournalEntryID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := requireDay(params.Params, paramAccountingDate)
	if err != nil {
		return nil, err
	}
	code, err := requireBoundedText(params.Params, paramReversalReasonCode, maxReversalReasonCode)
	if err != nil {
		return nil, err
	}
	text, err := requireBoundedText(params.Params, paramReversalReasonText, maxReversalReasonText)
	if err != nil {
		return nil, err
	}

	return &serviceports.CreateJournalReversalRequest{
		OriginalJournalEntryID:  entryID,
		RequestedAccountingDate: accountingDate,
		ReasonCode:              code,
		ReasonText:              text,
		TenantInfo:              tenantFrom(*params),
	}, nil
}

func newRequestJournalReversalTool(
	reversals journalReversalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReportingReceivableTool(ledgerMoneySpec(&receivableSpec{
		name: "request_journal_reversal",
		description: "Propose reversing a posted journal entry that was booked in error, with " +
			"every debit and credit swapped. It is dated the day you name, or the next open " +
			"period's first day, waits for approval where the organization requires it, and " +
			"post_journal_reversal books it.",
		resource:  permission.ResourceJournalReversal,
		artifact:  journalReversalRecordEntity,
		operation: permission.OpCreate,
		rationale: "Commits the organization to taking a posted entry back out of the " +
			"ledger, approved at once where approval is off; only a person requests a reversal.",
		properties: map[string]any{
			paramJournalEntryID: stringProperty("The posted journal entry to reverse, from "+
				"list_journal_entries or get_journal_entry. Never guess one.", 0),
			paramAccountingDate: agenttoolschema.Date(
				"The day the reversal is booked; a day in a " +
					"closed period moves to the next open one where policy allows.",
			),
			paramReversalReasonCode: stringProperty("A short code for the kind of error, such "+
				"as WrongAccount or Duplicate.", maxReversalReasonCode),
			paramReversalReasonText: stringProperty("What was wrong with the entry, as the "+
				"approver and the auditor will read it.", maxReversalReasonText),
		},
		required: []string{
			paramJournalEntryID,
			paramAccountingDate,
			paramReversalReasonCode,
			paramReversalReasonText,
		},
		target:      targetReversedEntry,
		searchTerms: []string{"reverse", "undo", "wrong", "entry", "error"},
	}), receivablePlan[*serviceports.CreateJournalReversalRequest, *reversalPlan]{
		request: journalReversalRequestFrom,
		plan: func(
			ctx context.Context,
			req *serviceports.CreateJournalReversalRequest,
			params *serviceports.ToolExecuteParams,
		) (*reversalPlan, error) {
			change, err := reversals.PlanCreate(ctx, req, params.Actor)

			return labelledReversal(ctx, accounts, params, change, err)
		},
		refused: func(*serviceports.CreateJournalReversalRequest) string {
			return "Would request a reversal of a posted journal entry."
		},
		render: renderReversalRequest,
		run: func(
			ctx context.Context,
			req *serviceports.CreateJournalReversalRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := reversals.Create(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "requested",
				Kind:   journalReversalKind,
				IDs:    map[string]string{paramJournalReversalID: created.ID.String()},
				Record: recordOf(journalReversalRecordEntity, created.ID),
			}, nil
		},
	})
}

func cancelJournalReversalRequest(
	params *serviceports.ToolExecuteParams,
) (*serviceports.CancelJournalReversalRequest, error) {
	id, err := requirePulid(params.Params, paramJournalReversalID)
	if err != nil {
		return nil, err
	}
	reason, err := requireBoundedText(params.Params, paramReason, maxJournalCancelReason)
	if err != nil {
		return nil, err
	}

	return &serviceports.CancelJournalReversalRequest{
		ReversalID: id,
		Reason:     reason,
		TenantInfo: tenantFrom(*params),
	}, nil
}

func newCancelJournalReversalTool(
	reversals journalReversalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReceivableTool(&receivableSpec{
		name: "cancel_journal_reversal",
		description: "Cancel a journal reversal that has not been posted, with the reason, " +
			"when the entry should stand after all. The original entry is left as it is.",
		resource:    permission.ResourceJournalReversal,
		artifact:    journalReversalRecordEntity,
		operation:   permission.OpCancel,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		taintHold: "The reason is what the reversal's requester reads next, so a run that has " +
			"read outside text proposes it.",
		rationale: "Withdraws a reversal before it reaches the ledger; nothing is booked, and " +
			"it is requested again if it was needed after all.",
		properties: map[string]any{
			paramJournalReversalID: stringProperty(journalReversalSupplier, 0),
			paramReason: stringProperty("Why the entry should stand, in a sentence the "+
				"requester can act on.", maxJournalCancelReason),
		},
		required: []string{paramJournalReversalID, paramReason},
		target:   targetJournalReversal,
	}, receivablePlan[*serviceports.CancelJournalReversalRequest, *reversalPlan]{
		request: cancelJournalReversalRequest,
		plan: func(
			ctx context.Context,
			req *serviceports.CancelJournalReversalRequest,
			params *serviceports.ToolExecuteParams,
		) (*reversalPlan, error) {
			change, err := reversals.PlanCancel(ctx, req, params.Actor)

			return labelledReversal(ctx, accounts, params, change, err)
		},
		refused: func(*serviceports.CancelJournalReversalRequest) string {
			return "Would cancel a journal reversal."
		},
		render: renderReversalCancel,
		run: func(
			ctx context.Context,
			req *serviceports.CancelJournalReversalRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := reversals.Cancel(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func newPostJournalReversalTool(
	reversals journalReversalKeeper,
	accounts ledgerAccounts,
) serviceports.AgentTool {
	return newReceivableTool(ledgerMoneySpec(&receivableSpec{
		name: "post_journal_reversal",
		description: "Propose posting an approved journal reversal: it books the original " +
			"entry's lines with debits and credits swapped and marks the original reversed. " +
			"It changes account balances for good, so a person always decides.",
		resource:  permission.ResourceJournalReversal,
		artifact:  journalReversalRecordEntity,
		operation: permission.OpApprove,
		rationale: "Books the reversing entry to the general ledger; only a person posts a " +
			"reversal.",
		properties: map[string]any{
			paramJournalReversalID: stringProperty(journalReversalSupplier, 0),
		},
		required: []string{paramJournalReversalID},
		target:   targetJournalReversal,
	}), receivablePlan[*serviceports.GetJournalReversalRequest, *reversalPlan]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*serviceports.GetJournalReversalRequest, error) {
			id, err := requirePulid(params.Params, paramJournalReversalID)
			if err != nil {
				return nil, err
			}

			return &serviceports.GetJournalReversalRequest{
				ReversalID: id,
				TenantInfo: tenantFrom(*params),
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *serviceports.GetJournalReversalRequest,
			params *serviceports.ToolExecuteParams,
		) (*reversalPlan, error) {
			change, err := reversals.PlanPost(ctx, req, params.Actor)

			return labelledReversal(ctx, accounts, params, change, err)
		},
		refused: func(*serviceports.GetJournalReversalRequest) string {
			return "Would post a journal reversal to the general ledger."
		},
		render: renderReversalPost,
		run: func(
			ctx context.Context,
			req *serviceports.GetJournalReversalRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := reversals.Post(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func provideDraftManualJournalTool(
	journals *manualjournalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newDraftManualJournalTool(journals, accounts)
}

func provideReviseManualJournalDraftTool(
	journals *manualjournalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newReviseManualJournalDraftTool(journals, accounts)
}

func provideSubmitManualJournalTool(
	journals *manualjournalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newSubmitManualJournalTool(journals, accounts)
}

func provideCancelManualJournalTool(
	journals *manualjournalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newCancelManualJournalTool(journals, accounts)
}

func providePostManualJournalTool(
	journals *manualjournalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newPostManualJournalTool(journals, accounts)
}

func provideRequestJournalReversalTool(
	reversals *journalreversalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newRequestJournalReversalTool(reversals, accounts)
}

func provideCancelJournalReversalTool(
	reversals *journalreversalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newCancelJournalReversalTool(reversals, accounts)
}

func providePostJournalReversalTool(
	reversals *journalreversalservice.Service,
	accounts repositories.GLAccountRepository,
) serviceports.AgentTool {
	return newPostJournalReversalTool(reversals, accounts)
}
