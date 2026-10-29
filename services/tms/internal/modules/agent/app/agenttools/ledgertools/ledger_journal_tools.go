package ledgertools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalreversalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/manualjournalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/journalpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const paramManualJournalID = "manualJournalId"

const paramJournalReversalID = "journalReversalId"

const paramJournalEntryID = "journalEntryId"

const paramJournalDescription = "description"

const paramJournalLines = "lines"

const paramJournalCurrency = "currencyCode"

const paramJournalLineDebit = "debit"

const paramJournalLineCredit = "credit"

const paramJournalLineLocationID = "locationId"

const paramReversalReasonCode = "reasonCode"

const paramReversalReasonText = "reasonText"

const maxManualJournalLines = 100

const maxManualJournalDescription = 500

const maxManualJournalReason = 1000

const maxManualJournalLineText = 255

const maxJournalCancelReason = 1000

const maxReversalReasonCode = 100

const maxReversalReasonText = 1000

const manualJournalKind = "manual journal"

const manualJournalRecordEntity = "manual_journal"

const journalReversalRecordEntity = "journal_reversal"

const journalReversalKind = "journal reversal"

const manualJournalSupplier = "The manual journal, from ledger_list_manual_journals or ledger_get_manual_journal. Never guess one."

const journalReversalSupplier = "The journal reversal, from ledger_list_journal_reversals or the result of ledger_create_journal_reversal. Never guess one."

const manualJournalCurrencyPattern = 3

var errJournalLineSide = errors.New(
	"each line takes exactly one of debit or credit, greater than zero",
)

var errAgentCannotApproveJournal = errors.New(
	"manual journal approval is turned off, so submitting approves the journal; only a " +
		"person submits it",
)

type ManualJournalKeeper interface {
	Get(
		ctx context.Context,
		req *manualjournalports.GetManualJournalRequest,
	) (*manualjournal.Request, error)
	PlanCreateDraft(
		ctx context.Context,
		req *manualjournalports.CreateManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournal.Request, error)
	CreateDraft(
		ctx context.Context,
		req *manualjournalports.CreateManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournal.Request, error)
	PlanUpdateDraft(
		ctx context.Context,
		req *manualjournalports.UpdateManualJournalDraftRequest,
		actor *permissionports.RequestActor,
	) (*manualjournalports.ManualJournalChange, error)
	UpdateDraft(
		ctx context.Context,
		req *manualjournalports.UpdateManualJournalDraftRequest,
		actor *permissionports.RequestActor,
	) (*manualjournal.Request, error)
	PlanSubmit(
		ctx context.Context,
		req *manualjournalports.GetManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournalports.ManualJournalChange, error)
	Submit(
		ctx context.Context,
		req *manualjournalports.GetManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournal.Request, error)
	PlanCancel(
		ctx context.Context,
		req *manualjournalports.CancelManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournalports.ManualJournalChange, error)
	Cancel(
		ctx context.Context,
		req *manualjournalports.CancelManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournal.Request, error)
	PlanPost(
		ctx context.Context,
		req *manualjournalports.GetManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournalports.ManualJournalChange, error)
	Post(
		ctx context.Context,
		req *manualjournalports.GetManualJournalRequest,
		actor *permissionports.RequestActor,
	) (*manualjournal.Request, error)
}

type JournalReversalKeeper interface {
	PlanCreate(
		ctx context.Context,
		req *journalreversalports.CreateJournalReversalRequest,
		actor *permissionports.RequestActor,
	) (*journalreversalports.JournalReversalChange, error)
	Create(
		ctx context.Context,
		req *journalreversalports.CreateJournalReversalRequest,
		actor *permissionports.RequestActor,
	) (*journalreversal.Reversal, error)
	PlanCancel(
		ctx context.Context,
		req *journalreversalports.CancelJournalReversalRequest,
		actor *permissionports.RequestActor,
	) (*journalreversalports.JournalReversalChange, error)
	Cancel(
		ctx context.Context,
		req *journalreversalports.CancelJournalReversalRequest,
		actor *permissionports.RequestActor,
	) (*journalreversal.Reversal, error)
	PlanPost(
		ctx context.Context,
		req *journalreversalports.GetJournalReversalRequest,
		actor *permissionports.RequestActor,
	) (*journalreversalports.JournalReversalChange, error)
	Post(
		ctx context.Context,
		req *journalreversalports.GetJournalReversalRequest,
		actor *permissionports.RequestActor,
	) (*journalreversal.Reversal, error)
}

func targetManualJournal(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramManualJournalID, permission.ResourceManualJournal)
}

func targetJournalReversal(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramJournalReversalID, permission.ResourceJournalReversal)
}

func targetReversedEntry(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramJournalEntryID, permission.ResourceJournalEntry)
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
		paramJournalDescription: toolparams.StringProperty(
			description,
			maxManualJournalDescription,
		),
		toolparams.ParamReason: toolparams.StringProperty(
			"Why the entry is needed: the evidence or the "+
				"correction it makes.",
			maxManualJournalReason,
		),
		toolparams.ParamAccountingDate: agenttoolschema.Date(date),
		paramJournalCurrency: toolparams.StringProperty(
			"The ISO currency code, such as USD. Leave it "+
				"out for the organization's functional currency.",
			manualJournalCurrencyPattern,
		),
		paramJournalLines: lines,
	}
	if forRevision {
		properties[paramManualJournalID] = agenttoolschema.RecordIDText(
			permission.ResourceManualJournal, manualJournalSupplier,
		)
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
				toolparams.ParamGLAccountID: agenttoolschema.RecordIDText(
					permission.ResourceGeneralLedgerAccount,
					"The account the line posts to, from ledger_list_accounts; it must be active "+
						"and allow manual journals.",
				),
				paramJournalDescription: toolparams.StringProperty("What the line is for.",
					maxManualJournalLineText),
				paramJournalLineDebit: toolparams.StringProperty("The debit as a decimal such as "+
					"125.00. Give a debit or a credit, never both.", 0),
				paramJournalLineCredit: toolparams.StringProperty(
					"The credit as a decimal such as "+
						"125.00. Give a debit or a credit, never both.",
					0,
				),
				toolparams.ParamCustomerID: agenttoolschema.RecordIDText(
					permission.ResourceCustomer,
					"The customer the line is for, from customer_list or customer_get, when "+
						"it concerns one.",
				),
				paramJournalLineLocationID: agenttoolschema.RecordIDText(
					permission.ResourceLocation,
					"The location the line is for, from location_list, when it concerns one.",
				),
			},
			toolschema.KeyRequired: []string{
				toolparams.ParamGLAccountID,
				paramJournalDescription,
			},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

func readJournalLines(params map[string]any) ([]*manualjournalports.ManualJournalLineInput, error) {
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

	lines := make([]*manualjournalports.ManualJournalLineInput, 0, len(raw))
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

func readJournalLine(fields map[string]any) (*manualjournalports.ManualJournalLineInput, error) {
	accountID, err := toolparams.RequirePulid(fields, toolparams.ParamGLAccountID)
	if err != nil {
		return nil, err
	}
	description, err := toolparams.RequireBoundedText(
		fields,
		paramJournalDescription,
		maxManualJournalLineText,
	)
	if err != nil {
		return nil, err
	}
	debit, hasDebit, err := toolparams.OptionalMoney(fields, paramJournalLineDebit)
	if err != nil {
		return nil, err
	}
	credit, hasCredit, err := toolparams.OptionalMoney(fields, paramJournalLineCredit)
	if err != nil {
		return nil, err
	}
	if hasDebit == hasCredit {
		return nil, errJournalLineSide
	}
	customerID, err := toolparams.OptionalPulidParam(fields, toolparams.ParamCustomerID)
	if err != nil {
		return nil, err
	}
	locationID, err := toolparams.OptionalPulidParam(fields, paramJournalLineLocationID)
	if err != nil {
		return nil, err
	}

	line := &manualjournalports.ManualJournalLineInput{
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
	code := strings.ToUpper(
		strings.TrimSpace(toolparams.OptionalString(params, paramJournalCurrency)),
	)
	if code != "" && len(code) != manualJournalCurrencyPattern {
		return "", fmt.Errorf("%s must be a three-letter code such as USD", paramJournalCurrency)
	}

	return code, nil
}

func draftManualJournalRequest(
	params *agenttoolports.ToolExecuteParams,
) (*manualjournalports.CreateManualJournalRequest, error) {
	description, err := toolparams.RequireBoundedText(
		params.Params,
		paramJournalDescription,
		maxManualJournalDescription,
	)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.BoundedText(
		params.Params,
		toolparams.ParamReason,
		maxManualJournalReason,
	)
	if err != nil {
		return nil, err
	}
	accountingDate, err := toolparams.RequireDay(params.Params, toolparams.ParamAccountingDate)
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

	return &manualjournalports.CreateManualJournalRequest{
		Description:    description,
		Reason:         reason,
		AccountingDate: accountingDate,
		CurrencyCode:   currency,
		Lines:          lines,
		TenantInfo:     toolparams.TenantFrom(*params),
	}, nil
}

func journalRevisionFrom(
	params *agenttoolports.ToolExecuteParams,
) (*journalpreview.JournalRevision, error) {
	id, err := toolparams.RequirePulid(params.Params, paramManualJournalID)
	if err != nil {
		return nil, err
	}
	revision := &journalpreview.JournalRevision{ID: id}
	if revision.Description, err = toolparams.BoundedText(
		params.Params,
		paramJournalDescription,
		maxManualJournalDescription,
	); err != nil {
		return nil, err
	}
	_, revision.HasReason = params.Params[toolparams.ParamReason]
	if revision.Reason, err = toolparams.BoundedText(
		params.Params,
		toolparams.ParamReason,
		maxManualJournalReason,
	); err != nil {
		return nil, err
	}
	if revision.Date, _, err = toolparams.OptionalDay(
		params.Params,
		toolparams.ParamAccountingDate,
	); err != nil {
		return nil, err
	}
	if revision.Currency, err = readJournalCurrency(params.Params); err != nil {
		return nil, err
	}
	if _, given := params.Params[paramJournalLines]; given {
		if revision.Lines, err = readJournalLines(params.Params); err != nil {
			return nil, err
		}
	}

	return revision, nil
}

func revisedDraft(
	ctx context.Context,
	journals ManualJournalKeeper,
	revision *journalpreview.JournalRevision,
	params *agenttoolports.ToolExecuteParams,
) (*manualjournalports.UpdateManualJournalDraftRequest, error) {
	current, err := journals.Get(ctx, &manualjournalports.GetManualJournalRequest{
		RequestID:  revision.ID,
		TenantInfo: toolparams.TenantFrom(*params),
	})
	if err != nil {
		return nil, err
	}

	return revision.Over(current, params), nil
}

func manualJournalRequestFrom(
	params *agenttoolports.ToolExecuteParams,
) (*manualjournalports.GetManualJournalRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, paramManualJournalID)
	if err != nil {
		return nil, err
	}

	return &manualjournalports.GetManualJournalRequest{
		RequestID:  id,
		TenantInfo: toolparams.TenantFrom(*params),
	}, nil
}

func manualJournalResult(
	action string,
	entity *manualjournal.Request,
) *agentrecord.ToolExecutionResult {
	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   manualJournalKind,
		Name:   entity.RequestNumber,
		IDs:    map[string]string{paramManualJournalID: entity.ID.String()},
		Record: changepreview.RecordOf(manualJournalRecordEntity, entity.ID),
	}
}

func journalDraftSpec(spec *toolspec.ReceivableSpec) *toolspec.ReceivableSpec {
	spec.Resource = permission.ResourceManualJournal
	spec.Artifact = manualJournalRecordEntity
	spec.Egress = agent.EgressInternal
	spec.DefaultTier = agent.TierPropose
	spec.MaxTier = agent.TierActWithApproval
	spec.Reversible = true

	return spec
}

func labelledJournal(
	ctx context.Context,
	accounts journalpreview.LedgerAccounts,
	params *agenttoolports.ToolExecuteParams,
	change *manualjournalports.ManualJournalChange,
	err error,
) (*journalpreview.ManualJournalPlan, error) {
	if err != nil {
		return nil, err
	}

	return journalpreview.PlanManualJournalChange(
		ctx,
		accounts,
		toolparams.TenantFrom(*params),
		change,
	)
}

func labelledReversal(
	ctx context.Context,
	accounts journalpreview.LedgerAccounts,
	params *agenttoolports.ToolExecuteParams,
	change *journalreversalports.JournalReversalChange,
	err error,
) (*journalpreview.ReversalPlan, error) {
	if err != nil {
		return nil, err
	}

	return journalpreview.PlanReversal(ctx, accounts, toolparams.TenantFrom(*params), change)
}

func newDraftManualJournalTool(
	journals ManualJournalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(journalDraftSpec(&toolspec.ReceivableSpec{
		Name:        "ledger_create_manual_journal",
		SearchTerms: []string{"draft manual journal"},
		Description: "Draft a manual journal entry for the general ledger: its lines, the " +
			"accounts they post to and the day it is booked. It is saved as a draft and books " +
			"nothing; ledger_submit_manual_journal sends it for approval. Take accounts from " +
			"ledger_list_accounts and make the debits equal the credits.",
		Operation: permission.OpCreate,
		TaintHold: "A journal drafted from what someone outside wrote is proposed, since its " +
			"lines are what an approver books.",
		Rationale: "Saves a draft journal inside Trenova; nothing reaches the ledger until a " +
			"person submits, approves and posts it, and a draft is cancelled the same way.",
		Properties: manualJournalDraftProperties(false),
		Required: []string{
			paramJournalDescription,
			toolparams.ParamAccountingDate,
			paramJournalLines,
		},
	}), toolspec.ReceivablePlan[*manualjournalports.CreateManualJournalRequest, *journalpreview.ManualJournalPlan]{
		Request: draftManualJournalRequest,
		Plan: func(
			ctx context.Context,
			req *manualjournalports.CreateManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*journalpreview.ManualJournalPlan, error) {
			planned, err := journals.PlanCreateDraft(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return journalpreview.PlanManualJournalDraft(ctx, accounts, req.TenantInfo, planned)
		},
		Refused: func(req *manualjournalports.CreateManualJournalRequest) string {
			return fmt.Sprintf("Would draft the manual journal %q with %s.",
				req.Description, changepreview.CountOf(len(req.Lines), "line"))
		},
		Render: journalpreview.RenderManualJournalDraft,
		Run: func(
			ctx context.Context,
			req *manualjournalports.CreateManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := journals.CreateDraft(ctx, req, params.Actor)
			if err != nil {
				return nil, err
			}

			return manualJournalResult("drafted", created), nil
		},
	})
}

func newReviseManualJournalDraftTool(
	journals ManualJournalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(journalDraftSpec(&toolspec.ReceivableSpec{
		Name:        "ledger_update_manual_journal",
		SearchTerms: []string{"change journal lines", "edit draft journal", "fix draft journal"},
		Description: "Rewrite a draft manual journal before it is submitted: its " +
			"description, reason, day, currency or lines. Fields left out keep their value; " +
			"lines, when given, replace every line. Only a draft can be revised; a journal " +
			"sent back by its approver is drafted again with ledger_create_manual_journal.",
		Operation: permission.OpUpdate,
		TaintHold: "A journal rewritten from what someone outside wrote is proposed, since its " +
			"lines are what an approver books.",
		Rationale: "Changes a draft journal inside Trenova; nothing reaches the ledger until a " +
			"person submits, approves and posts it, and a later revision changes it back.",
		Properties: manualJournalDraftProperties(true),
		Required:   []string{paramManualJournalID},
		Target:     targetManualJournal,
	}), toolspec.ReceivablePlan[*journalpreview.JournalRevision, *journalpreview.ManualJournalPlan]{
		Request: journalRevisionFrom,
		Plan: func(
			ctx context.Context,
			revision *journalpreview.JournalRevision,
			params *agenttoolports.ToolExecuteParams,
		) (*journalpreview.ManualJournalPlan, error) {
			req, err := revisedDraft(ctx, journals, revision, params)
			if err != nil {
				return nil, err
			}
			change, err := journals.PlanUpdateDraft(ctx, req, params.Actor)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		Refused: func(*journalpreview.JournalRevision) string {
			return "Would revise a draft manual journal."
		},
		Render: journalpreview.RenderManualJournalRevision,
		Run: func(
			ctx context.Context,
			revision *journalpreview.JournalRevision,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
	journals ManualJournalKeeper,
	req *manualjournalports.GetManualJournalRequest,
	params *agenttoolports.ToolExecuteParams,
) (*manualjournalports.ManualJournalChange, error) {
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
	journals ManualJournalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(journalDraftSpec(&toolspec.ReceivableSpec{
		Name: "ledger_submit_manual_journal",
		Description: "Submit a balanced draft manual journal for approval. It waits for an " +
			"approver; where the organization has manual journal approval turned off, " +
			"submitting approves it, so only a person does that. Read it with " +
			"ledger_get_manual_journal first.",
		Operation: permission.OpSubmit,
		Rationale: "Moves a draft journal into the approval queue inside Trenova; an approver " +
			"decides and it books nothing until a person posts it.",
		Properties: map[string]any{
			paramManualJournalID: agenttoolschema.RecordIDText(permission.ResourceManualJournal,
				manualJournalSupplier),
		},
		Required: []string{paramManualJournalID},
		Target:   targetManualJournal,
	}), toolspec.ReceivablePlan[*manualjournalports.GetManualJournalRequest, *journalpreview.ManualJournalPlan]{
		Request: manualJournalRequestFrom,
		Plan: func(
			ctx context.Context,
			req *manualjournalports.GetManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*journalpreview.ManualJournalPlan, error) {
			change, err := plannedSubmit(ctx, journals, req, params)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		Refused: func(*manualjournalports.GetManualJournalRequest) string {
			return "Would submit a manual journal for approval."
		},
		Render: journalpreview.RenderManualJournalSubmit,
		Run: func(
			ctx context.Context,
			req *manualjournalports.GetManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if _, err := plannedSubmit(ctx, journals, req, params); err != nil {
				return nil, err
			}
			_, err := journals.Submit(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func cancelManualJournalRequest(
	params *agenttoolports.ToolExecuteParams,
) (*manualjournalports.CancelManualJournalRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, paramManualJournalID)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(
		params.Params,
		toolparams.ParamReason,
		maxJournalCancelReason,
	)
	if err != nil {
		return nil, err
	}

	return &manualjournalports.CancelManualJournalRequest{
		RequestID:  id,
		Reason:     reason,
		TenantInfo: toolparams.TenantFrom(*params),
	}, nil
}

func newCancelManualJournalTool(
	journals ManualJournalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(journalDraftSpec(&toolspec.ReceivableSpec{
		Name: "ledger_cancel_manual_journal",
		Description: "Cancel a manual journal that is a draft, waiting for approval or " +
			"approved but not posted, with the reason. It is kept, cancelled, and never " +
			"posted. A posted journal is undone with ledger_create_journal_reversal instead.",
		Operation: permission.OpCancel,
		TaintHold: "The reason is what the journal's author reads next, so a run that has read " +
			"outside text proposes it.",
		Rationale: "Withdraws a journal before it reaches the ledger; nothing is booked, and " +
			"the entry is drafted again if it was needed after all.",
		Properties: map[string]any{
			paramManualJournalID: agenttoolschema.RecordIDText(permission.ResourceManualJournal,
				manualJournalSupplier),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why it must not be posted, in a sentence its author "+
					"can act on.",
				maxJournalCancelReason,
			),
		},
		Required: []string{paramManualJournalID, toolparams.ParamReason},
		Target:   targetManualJournal,
	}), toolspec.ReceivablePlan[*manualjournalports.CancelManualJournalRequest, *journalpreview.ManualJournalPlan]{
		Request: cancelManualJournalRequest,
		Plan: func(
			ctx context.Context,
			req *manualjournalports.CancelManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*journalpreview.ManualJournalPlan, error) {
			change, err := journals.PlanCancel(ctx, req, params.Actor)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		Refused: func(*manualjournalports.CancelManualJournalRequest) string {
			return "Would cancel a manual journal."
		},
		Render: journalpreview.RenderManualJournalCancel,
		Run: func(
			ctx context.Context,
			req *manualjournalports.CancelManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := journals.Cancel(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func newPostManualJournalTool(
	journals ManualJournalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(journalpreview.LedgerMoneySpec(&toolspec.ReceivableSpec{
		Name: "ledger_post_manual_journal",
		Description: "Propose posting an approved manual journal to the general ledger, in " +
			"its fiscal period or the next open one the closed period policy allows. It " +
			"changes account balances and is undone only by a reversal, so a person always " +
			"decides.",
		Resource:  permission.ResourceManualJournal,
		Artifact:  manualJournalRecordEntity,
		Operation: permission.OpApprove,
		Rationale: "Books the journal's lines to the general ledger; only a person posts a " +
			"manual journal.",
		Properties: map[string]any{
			paramManualJournalID: agenttoolschema.RecordIDText(permission.ResourceManualJournal,
				manualJournalSupplier),
		},
		Required: []string{paramManualJournalID},
		Target:   targetManualJournal,
	}), toolspec.ReceivablePlan[*manualjournalports.GetManualJournalRequest, *journalpreview.ManualJournalPlan]{
		Request: manualJournalRequestFrom,
		Plan: func(
			ctx context.Context,
			req *manualjournalports.GetManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*journalpreview.ManualJournalPlan, error) {
			change, err := journals.PlanPost(ctx, req, params.Actor)

			return labelledJournal(ctx, accounts, params, change, err)
		},
		Refused: func(*manualjournalports.GetManualJournalRequest) string {
			return "Would post a manual journal to the general ledger."
		},
		Render: journalpreview.RenderManualJournalPost,
		Run: func(
			ctx context.Context,
			req *manualjournalports.GetManualJournalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := journals.Post(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func journalReversalRequestFrom(
	params *agenttoolports.ToolExecuteParams,
) (*journalreversalports.CreateJournalReversalRequest, error) {
	entryID, err := toolparams.RequirePulid(params.Params, paramJournalEntryID)
	if err != nil {
		return nil, err
	}
	accountingDate, err := toolparams.RequireDay(params.Params, toolparams.ParamAccountingDate)
	if err != nil {
		return nil, err
	}
	code, err := toolparams.RequireBoundedText(
		params.Params,
		paramReversalReasonCode,
		maxReversalReasonCode,
	)
	if err != nil {
		return nil, err
	}
	text, err := toolparams.RequireBoundedText(
		params.Params,
		paramReversalReasonText,
		maxReversalReasonText,
	)
	if err != nil {
		return nil, err
	}

	return &journalreversalports.CreateJournalReversalRequest{
		OriginalJournalEntryID:  entryID,
		RequestedAccountingDate: accountingDate,
		ReasonCode:              code,
		ReasonText:              text,
		TenantInfo:              toolparams.TenantFrom(*params),
	}, nil
}

func newRequestJournalReversalTool(
	reversals JournalReversalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		journalpreview.LedgerMoneySpec(&toolspec.ReceivableSpec{
			Name: "ledger_create_journal_reversal",
			Description: "Propose reversing a posted journal entry that was booked in error, with " +
				"every debit and credit swapped. It is dated the day you name, or the next open " +
				"period's first day, waits for approval where the organization requires it, and " +
				"ledger_post_journal_reversal books it.",
			Resource:  permission.ResourceJournalReversal,
			Artifact:  journalReversalRecordEntity,
			Operation: permission.OpCreate,
			Rationale: "Commits the organization to taking a posted entry back out of the " +
				"ledger, approved at once where approval is off; only a person requests a reversal.",
			Properties: map[string]any{
				paramJournalEntryID: agenttoolschema.RecordIDText(permission.ResourceJournalEntry,
					"The posted journal entry to reverse, from ledger_list_journal_entries or "+
						"ledger_get_journal_entry. Never guess one."),
				toolparams.ParamAccountingDate: agenttoolschema.Date(
					"The day the reversal is booked; a day in a " +
						"closed period moves to the next open one where policy allows.",
				),
				paramReversalReasonCode: toolparams.StringProperty(
					"A short code for the kind of error, such "+
						"as WrongAccount or Duplicate.",
					maxReversalReasonCode,
				),
				paramReversalReasonText: toolparams.StringProperty(
					"What was wrong with the entry, as the "+
						"approver and the auditor will read it.",
					maxReversalReasonText,
				),
			},
			Required: []string{
				paramJournalEntryID,
				toolparams.ParamAccountingDate,
				paramReversalReasonCode,
				paramReversalReasonText,
			},
			Target:      targetReversedEntry,
			SearchTerms: []string{"reverse", "undo", "wrong", "entry", "error"},
		}),
		toolspec.ReceivablePlan[*journalreversalports.CreateJournalReversalRequest, *journalpreview.ReversalPlan]{
			Request: journalReversalRequestFrom,
			Plan: func(
				ctx context.Context,
				req *journalreversalports.CreateJournalReversalRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*journalpreview.ReversalPlan, error) {
				change, err := reversals.PlanCreate(ctx, req, params.Actor)

				return labelledReversal(ctx, accounts, params, change, err)
			},
			Refused: func(*journalreversalports.CreateJournalReversalRequest) string {
				return "Would request a reversal of a posted journal entry."
			},
			Render: journalpreview.RenderReversalRequest,
			Run: func(
				ctx context.Context,
				req *journalreversalports.CreateJournalReversalRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				created, err := reversals.Create(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}

				return &agentrecord.ToolExecutionResult{
					Action: "requested",
					Kind:   journalReversalKind,
					IDs:    map[string]string{paramJournalReversalID: created.ID.String()},
					Record: changepreview.RecordOf(journalReversalRecordEntity, created.ID),
				}, nil
			},
		},
	)
}

func cancelJournalReversalRequest(
	params *agenttoolports.ToolExecuteParams,
) (*journalreversalports.CancelJournalReversalRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, paramJournalReversalID)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(
		params.Params,
		toolparams.ParamReason,
		maxJournalCancelReason,
	)
	if err != nil {
		return nil, err
	}

	return &journalreversalports.CancelJournalReversalRequest{
		ReversalID: id,
		Reason:     reason,
		TenantInfo: toolparams.TenantFrom(*params),
	}, nil
}

func newCancelJournalReversalTool(
	reversals JournalReversalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(&toolspec.ReceivableSpec{
		Name: "ledger_cancel_journal_reversal",
		Description: "Cancel a journal reversal that has not been posted, with the reason, " +
			"when the entry should stand after all. The original entry is left as it is.",
		Resource:    permission.ResourceJournalReversal,
		Artifact:    journalReversalRecordEntity,
		Operation:   permission.OpCancel,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		TaintHold: "The reason is what the reversal's requester reads next, so a run that has " +
			"read outside text proposes it.",
		Rationale: "Withdraws a reversal before it reaches the ledger; nothing is booked, and " +
			"it is requested again if it was needed after all.",
		Properties: map[string]any{
			paramJournalReversalID: agenttoolschema.RecordIDText(
				permission.ResourceJournalReversal, journalReversalSupplier,
			),
			toolparams.ParamReason: toolparams.StringProperty(
				"Why the entry should stand, in a sentence the "+
					"requester can act on.",
				maxJournalCancelReason,
			),
		},
		Required: []string{paramJournalReversalID, toolparams.ParamReason},
		Target:   targetJournalReversal,
	}, toolspec.ReceivablePlan[*journalreversalports.CancelJournalReversalRequest, *journalpreview.ReversalPlan]{
		Request: cancelJournalReversalRequest,
		Plan: func(
			ctx context.Context,
			req *journalreversalports.CancelJournalReversalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*journalpreview.ReversalPlan, error) {
			change, err := reversals.PlanCancel(ctx, req, params.Actor)

			return labelledReversal(ctx, accounts, params, change, err)
		},
		Refused: func(*journalreversalports.CancelJournalReversalRequest) string {
			return "Would cancel a journal reversal."
		},
		Render: journalpreview.RenderReversalCancel,
		Run: func(
			ctx context.Context,
			req *journalreversalports.CancelJournalReversalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := reversals.Cancel(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func newPostJournalReversalTool(
	reversals JournalReversalKeeper,
	accounts journalpreview.LedgerAccounts,
) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(journalpreview.LedgerMoneySpec(&toolspec.ReceivableSpec{
		Name: "ledger_post_journal_reversal",
		Description: "Propose posting an approved journal reversal: it books the original " +
			"entry's lines with debits and credits swapped and marks the original reversed. " +
			"It changes account balances for good, so a person always decides.",
		Resource:  permission.ResourceJournalReversal,
		Artifact:  journalReversalRecordEntity,
		Operation: permission.OpApprove,
		Rationale: "Books the reversing entry to the general ledger; only a person posts a " +
			"reversal.",
		Properties: map[string]any{
			paramJournalReversalID: agenttoolschema.RecordIDText(
				permission.ResourceJournalReversal, journalReversalSupplier,
			),
		},
		Required: []string{paramJournalReversalID},
		Target:   targetJournalReversal,
	}), toolspec.ReceivablePlan[*journalreversalports.GetJournalReversalRequest, *journalpreview.ReversalPlan]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*journalreversalports.GetJournalReversalRequest, error) {
			id, err := toolparams.RequirePulid(params.Params, paramJournalReversalID)
			if err != nil {
				return nil, err
			}

			return &journalreversalports.GetJournalReversalRequest{
				ReversalID: id,
				TenantInfo: toolparams.TenantFrom(*params),
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *journalreversalports.GetJournalReversalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*journalpreview.ReversalPlan, error) {
			change, err := reversals.PlanPost(ctx, req, params.Actor)

			return labelledReversal(ctx, accounts, params, change, err)
		},
		Refused: func(*journalreversalports.GetJournalReversalRequest) string {
			return "Would post a journal reversal to the general ledger."
		},
		Render: journalpreview.RenderReversalPost,
		Run: func(
			ctx context.Context,
			req *journalreversalports.GetJournalReversalRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := reversals.Post(ctx, req, params.Actor)

			return nil, err
		},
	})
}

func ProvideDraftManualJournalTool(
	journals *manualjournalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newDraftManualJournalTool(journals, accounts)
}

func ProvideReviseManualJournalDraftTool(
	journals *manualjournalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newReviseManualJournalDraftTool(journals, accounts)
}

func ProvideSubmitManualJournalTool(
	journals *manualjournalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newSubmitManualJournalTool(journals, accounts)
}

func ProvideCancelManualJournalTool(
	journals *manualjournalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newCancelManualJournalTool(journals, accounts)
}

func ProvidePostManualJournalTool(
	journals *manualjournalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newPostManualJournalTool(journals, accounts)
}

func ProvideRequestJournalReversalTool(
	reversals *journalreversalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newRequestJournalReversalTool(reversals, accounts)
}

func ProvideCancelJournalReversalTool(
	reversals *journalreversalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newCancelJournalReversalTool(reversals, accounts)
}

func ProvidePostJournalReversalTool(
	reversals *journalreversalservice.Service,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return newPostJournalReversalTool(reversals, accounts)
}
