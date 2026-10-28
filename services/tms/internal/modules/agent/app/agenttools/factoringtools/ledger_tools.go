package factoringtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatatools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/money"
)

const (
	paramItems     = "items"
	paramItemID    = "itemId"
	paramOutcome   = "outcome"
	paramAdvance   = "advance"
	paramFee       = "fee"
	paramFundedOn  = "fundedOn"
	paramMemo      = "memo"
	paramCash      = "cash"
	paramOccurred  = "occurredOn"
	maxMemoLength  = factoring.MaxReserveMemoLength
	maxReasonText  = factoring.MaxFundingReasonLength
	toolGetSubmit  = "factoring_get_submission"
	toolGetBalance = "factoring_get_reserve_balance"
)

type ledgerTool struct {
	ledger   factoringports.LedgerService
	accounts common.LedgerAccounts
}

func ledgerPolicy(name, rationale string) agenttoolports.ToolPolicy {
	return policy(name, &policySpec{
		resource:    permission.ResourceFactoringReserve,
		operation:   permission.OpCreate,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		egress:      agent.EgressMoney,
		idempotent:  true,
		rationale:   rationale,
	})
}

func itemIDParam(description string) map[string]any {
	return agenttoolschema.RecordIDText(permission.ResourceFactoringSubmission, description)
}

func amountParam(description string) map[string]any {
	return common.StringProperty(description+" A decimal such as 1250.00.", 0)
}

func requireApproval(name string, params *agenttoolports.ToolExecuteParams) error {
	if !params.ApprovedFromProposal() {
		return fmt.Errorf("%s: %w", name, common.ErrNeedsAPersonsApproval)
	}
	return nil
}

func optionalAmount(params map[string]any, key string) (*int64, error) {
	value, present, err := common.OptionalDecimal(params, key)
	if err != nil || !present {
		return nil, err
	}
	if value.IsNegative() {
		return nil, fmt.Errorf("parameter %q cannot be negative", key)
	}
	minor := money.MinorUnits(value)
	return &minor, nil
}

func (t *ledgerTool) movementChanges(
	ctx context.Context,
	tenant pagination.TenantInfo,
	currency string,
	movements []*factoringports.PlannedMovement,
) ([]*agent.RecordChange, error) {
	journals := make([]*kernelports.JournalPreview, 0, len(movements))
	sets := make([][]kernelports.JournalLinePreview, 0, len(movements))
	for _, movement := range movements {
		if movement == nil || len(movement.Lines) == 0 {
			continue
		}
		journal := journalOf(movement)
		journals = append(journals, journal)
		sets = append(sets, journal.Lines)
	}
	labels, err := common.AccountLabels(ctx, t.accounts, tenant, sets...)
	if err != nil {
		return nil, err
	}
	changes := make([]*agent.RecordChange, 0, len(journals))
	idx := 0
	for _, movement := range movements {
		if movement == nil || len(movement.Lines) == 0 {
			continue
		}
		change, buildErr := masterdatatools.BookedEntryChange(
			journals[idx],
			currency,
			movementLabel(movement),
			labels,
		)
		if buildErr != nil {
			return nil, buildErr
		}
		changes = append(changes, change)
		idx++
	}
	return changes, nil
}

func journalOf(movement *factoringports.PlannedMovement) *kernelports.JournalPreview {
	lines := make([]kernelports.JournalLinePreview, 0, len(movement.Lines))
	for _, line := range movement.Lines {
		lines = append(lines, kernelports.JournalLinePreview{
			GLAccountID: line.AccountID,
			Description: string(line.Role),
			DebitMinor:  line.Debit,
			CreditMinor: line.Credit,
		})
	}
	return &kernelports.JournalPreview{
		AccountingDate:   movement.AccountingDate,
		FiscalPeriodID:   movement.FiscalPeriodID,
		EntryStatus:      movement.EntryStatus,
		RequiresApproval: movement.RequiresApproval,
		Lines:            lines,
	}
}

func movementLabel(movement *factoringports.PlannedMovement) string {
	switch movement.Movement.Kind {
	case factoring.ReserveEntryFunding:
		return "Factor funding of invoice " + movement.InvoiceNumber
	case factoring.ReserveEntryRelease:
		return "Reserve release on invoice " + movement.InvoiceNumber
	case factoring.ReserveEntryChargeback:
		return "Chargeback of invoice " + movement.InvoiceNumber
	case factoring.ReserveEntryChargebackReversal:
		return "Chargeback reversal on invoice " + movement.InvoiceNumber
	default:
		return "Factoring entry for invoice " + movement.InvoiceNumber
	}
}

func bookingSentences(movements []*factoringports.PlannedMovement) []string {
	sentences := make([]string, 0, 2)
	redated, approval := false, false
	for _, movement := range movements {
		if movement == nil || len(movement.Lines) == 0 {
			continue
		}
		redated = redated || movement.Redated()
		approval = approval || movement.RequiresApproval
	}
	if redated {
		sentences = append(sentences, "The day falls in a closed period, so the closed period "+
			"policy books it on the first open day.")
	}
	if approval {
		sentences = append(sentences, "The entries wait for journal approval before they post.")
	}
	return append(sentences, "Only an opposite entry undoes it.")
}

type RecordFactoringFundingTool struct {
	ledgerTool
}

func NewRecordFactoringFundingTool(
	ledger factoringports.LedgerService,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return &RecordFactoringFundingTool{ledgerTool{ledger: ledger, accounts: accounts}}
}

func (t *RecordFactoringFundingTool) Name() string { return "factoring_record_funding" }

func (t *RecordFactoringFundingTool) Description() string {
	return "Record what the factor funded or rejected on a sent schedule, from its funding " +
		"report or remittance. For each invoice, Funded with the advance and fee it paid " +
		"(default: the expected ones from factoring_get_submission) or Rejected with why. " +
		"Each funded invoice " +
		"is booked to the ledger: cash in, the fee expensed, and, under a sale, the reserve the " +
		"factor holds with the invoice settled; under secured borrowing, a liability to the " +
		"factor. A rejected invoice goes back to the queue. A person approves every booking."
}

func (t *RecordFactoringFundingTool) SearchTerms() []string {
	return []string{"factor funding", "funding report", "advance received", "factor rejected"}
}

func (t *RecordFactoringFundingTool) Prerequisites() []string { return []string{toolGetSubmit} }

func (t *RecordFactoringFundingTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramSubmissionID: submissionIDParam("The sent schedule, from factoring_list_submissions."),
		paramVersion:      submissionVersionParam(),
		paramFundedOn:     agenttoolschema.Date("The day the factor funded."),
		paramItems: map[string]any{
			toolschema.KeyType: toolschema.TypeArray,
			toolschema.KeyDescription: "What the factor did with each invoice it decided; " +
				"leave out the ones it has not decided yet.",
			toolschema.KeyMinItems: 1,
			toolschema.KeyMaxItems: factoring.MaxSubmissionInvoices,
			toolschema.KeyItems: map[string]any{
				toolschema.KeyType: toolschema.TypeObject,
				toolschema.KeyProperties: map[string]any{
					paramItemID: itemIDParam(
						"The invoice's line on the schedule (its item id) from " +
							"factoring_get_submission.",
					),
					paramOutcome: agenttoolschema.Enum(
						"Whether the factor funded or rejected it.",
						agenttoolschema.FactoringFundingOutcomes,
					),
					paramAdvance: amountParam("What the factor advanced, when it differs from " +
						"the expected advance."),
					paramFee: amountParam("What the factor kept as its fee, when it differs " +
						"from the expected fee."),
					paramReason: common.StringProperty(
						"Why the factor rejected it.",
						maxReasonText,
					),
				},
				toolschema.KeyRequired:             []string{paramItemID, paramOutcome},
				toolschema.KeyAdditionalProperties: false,
			},
		},
		paramMemo: common.StringProperty("A note kept with the entries, such as the factor's "+
			"funding report number.", maxMemoLength),
	}, paramSubmissionID, paramVersion, paramFundedOn, paramItems)
}

func (t *RecordFactoringFundingTool) Policy() agenttoolports.ToolPolicy {
	return ledgerPolicy(t.Name(), "Books the factor's advances and fees to the general ledger "+
		"and settles invoices; only a person's approval runs it.")
}

func (t *RecordFactoringFundingTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramSubmissionID, permission.ResourceFactoringSubmission)
}

func (t *RecordFactoringFundingTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringports.RecordFundingRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	submissionID, err := common.RequirePulid(params.Params, paramSubmissionID)
	if err != nil {
		return nil, err
	}
	version, err := common.RequireIntInRange(params.Params, paramVersion, 0, maxVersion)
	if err != nil {
		return nil, err
	}
	fundedOn, err := common.RequireDay(params.Params, paramFundedOn)
	if err != nil {
		return nil, err
	}
	items, err := fundingItems(params.Params)
	if err != nil {
		return nil, err
	}
	memo, err := common.OptionalBoundedText(params.Params, paramMemo, maxMemoLength)
	if err != nil {
		return nil, err
	}
	req := &factoringports.RecordFundingRequest{
		TenantInfo:   common.TenantFrom(*params),
		SubmissionID: submissionID,
		Version:      int64(version),
		FundedOn:     fundedOn,
		Items:        items,
	}
	if memo != nil {
		req.Memo = *memo
	}
	return req, nil
}

func fundingItems(params map[string]any) ([]factoringports.FundingItemInput, error) {
	raw, ok := params[paramItems].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("parameter %q must list at least one invoice", paramItems)
	}
	if len(raw) > factoring.MaxSubmissionInvoices {
		return nil, fmt.Errorf("parameter %q lists more invoices than a schedule holds", paramItems)
	}
	items := make([]factoringports.FundingItemInput, 0, len(raw))
	for idx, entry := range raw {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s[%d] must be an object", paramItems, idx)
		}
		itemID, err := common.RequirePulid(fields, paramItemID)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramItems, idx, err)
		}
		outcome := factoring.FundingOutcome(
			strings.TrimSpace(common.OptionalString(fields, paramOutcome)),
		)
		if !outcome.IsValid() {
			return nil, fmt.Errorf(
				"%s[%d].%s must be Funded or Rejected",
				paramItems,
				idx,
				paramOutcome,
			)
		}
		advance, err := optionalAmount(fields, paramAdvance)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramItems, idx, err)
		}
		fee, err := optionalAmount(fields, paramFee)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramItems, idx, err)
		}
		reason, err := common.OptionalBoundedText(fields, paramReason, maxReasonText)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", paramItems, idx, err)
		}
		item := factoringports.FundingItemInput{
			ItemID:       itemID,
			Outcome:      outcome,
			AdvanceMinor: advance,
			FeeMinor:     fee,
		}
		if reason != nil {
			item.Reason = *reason
		}
		items = append(items, item)
	}
	return items, nil
}

func (t *RecordFactoringFundingTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.ledger.PlanFunding(ctx, req, params.Actor)
	return err
}

func (t *RecordFactoringFundingTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := requireApproval(t.Name(), &params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.ledger.RecordFunding(ctx, req, params.Actor)
	return err
}

func (t *RecordFactoringFundingTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	const summary = "Would record the factor's funding of a schedule and book it to the ledger."
	req, err := t.request(&params)
	if err != nil {
		return refused(summary, err)
	}
	plan, err := t.ledger.PlanFunding(ctx, req, params.Actor)
	if err != nil {
		return refused(summary, err)
	}
	scheduleChange, err := toolpreview.Changed(
		submissionRecord(plan.Before),
		plan.Before,
		plan.After,
		submissionPreviewOptions()...,
	)
	if err != nil {
		return nil, err
	}
	entries, err := t.movementChanges(ctx, req.TenantInfo, plan.After.CurrencyCode, plan.Movements)
	if err != nil {
		return nil, err
	}

	currency := plan.After.CurrencyCode
	sentences := []string{fmt.Sprintf(
		"Would record schedule %s's funding on %s: %s funded (advance %s, fee %s, reserve %s) "+
			"and %s rejected, booking %s.",
		plan.After.Number,
		common.DayLabel(req.FundedOn),
		common.CountOf(plan.FundedCount, "invoice"),
		formatMinor(currency, plan.AdvanceMinor),
		formatMinor(currency, plan.FeeMinor),
		formatMinor(currency, plan.ReserveMinor),
		common.CountOf(plan.RejectCount, "invoice"),
		common.CountOf(len(entries), "journal entry"),
	)}
	sentences = append(sentences, bookingSentences(plan.Movements)...)
	return toolpreview.Build(
		strings.Join(sentences, " "),
		append([]*agent.RecordChange{scheduleChange}, entries...)...,
	), nil
}

type movementSpec struct {
	name        string
	description string
	rationale   string
	searchTerms []string
	dateHelp    string
	cashHelp    string
	reasonHelp  string
	needsReason bool
	refusal     string
	plan        func(
		factoringports.LedgerService,
		context.Context,
		*factoringports.ReserveMovementRequest,
		*permissionports.RequestActor,
	) (*factoringports.ReserveMovementPlan, error)
	run func(
		factoringports.LedgerService,
		context.Context,
		*factoringports.ReserveMovementRequest,
		*permissionports.RequestActor,
	) (*factoringports.ReserveMovementResult, error)
	summary func(*factoringports.ReserveMovementPlan, string) string
}

type ReserveMovementTool struct {
	ledgerTool
	spec movementSpec
}

func (t *ReserveMovementTool) Name() string            { return t.spec.name }
func (t *ReserveMovementTool) Description() string     { return t.spec.description }
func (t *ReserveMovementTool) SearchTerms() []string   { return t.spec.searchTerms }
func (t *ReserveMovementTool) Prerequisites() []string { return []string{toolGetSubmit} }

func (t *ReserveMovementTool) ParamSchema() map[string]any {
	properties := map[string]any{
		paramSubmissionID: submissionIDParam("The schedule, from factoring_list_submissions."),
		paramItemID: itemIDParam(
			"The invoice's line on the schedule (its item id) from factoring_get_submission.",
		),
		paramVersion:  submissionVersionParam(),
		paramOccurred: agenttoolschema.Date(t.spec.dateHelp),
		paramReason:   common.StringProperty(t.spec.reasonHelp, maxReasonText),
	}
	required := []string{paramSubmissionID, paramItemID, paramVersion, paramOccurred}
	if t.spec.cashHelp != "" {
		properties[paramCash] = amountParam(t.spec.cashHelp)
	}
	if t.spec.needsReason {
		required = append(required, paramReason)
	}
	return jsonschemautils.Object(properties, required...)
}

func (t *ReserveMovementTool) Policy() agenttoolports.ToolPolicy {
	return ledgerPolicy(t.spec.name, t.spec.rationale)
}

func (t *ReserveMovementTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramSubmissionID, permission.ResourceFactoringSubmission)
}

func (t *ReserveMovementTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringports.ReserveMovementRequest, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, err
	}
	submissionID, err := common.RequirePulid(params.Params, paramSubmissionID)
	if err != nil {
		return nil, err
	}
	itemID, err := common.RequirePulid(params.Params, paramItemID)
	if err != nil {
		return nil, err
	}
	version, err := common.RequireIntInRange(params.Params, paramVersion, 0, maxVersion)
	if err != nil {
		return nil, err
	}
	occurredOn, err := common.RequireDay(params.Params, paramOccurred)
	if err != nil {
		return nil, err
	}
	req := &factoringports.ReserveMovementRequest{
		TenantInfo:   common.TenantFrom(*params),
		SubmissionID: submissionID,
		ItemID:       itemID,
		Version:      int64(version),
		OccurredOn:   occurredOn,
	}
	if t.spec.cashHelp != "" {
		if req.CashMinor, err = optionalAmount(params.Params, paramCash); err != nil {
			return nil, err
		}
	}
	if t.spec.needsReason {
		reason, reasonErr := common.RequireBoundedText(params.Params, paramReason, maxReasonText)
		if reasonErr != nil {
			return nil, reasonErr
		}
		req.Reason = reason
		return req, nil
	}
	reason, err := common.OptionalBoundedText(params.Params, paramReason, maxReasonText)
	if err != nil {
		return nil, err
	}
	if reason != nil {
		req.Reason = *reason
	}
	return req, nil
}

func (t *ReserveMovementTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.spec.plan(t.ledger, ctx, req, params.Actor)
	return err
}

func (t *ReserveMovementTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := requireApproval(t.Name(), &params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.spec.run(t.ledger, ctx, req, params.Actor)
	return err
}

type itemView struct {
	Status           factoring.SubmissionItemStatus `json:"status"`
	SettledOn        *int64                         `json:"settledOn"`
	ChargedBackOn    *int64                         `json:"chargedBackOn"`
	ChargebackReason string                         `json:"chargebackReason"`
}

func itemViewOf(item *factoring.SubmissionItem) *itemView {
	if item == nil {
		return &itemView{}
	}
	return &itemView{
		Status:           item.Status,
		SettledOn:        item.SettledOn,
		ChargedBackOn:    item.ChargedBackOn,
		ChargebackReason: item.ChargebackReason,
	}
}

func (t *ReserveMovementTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	req, err := t.request(&params)
	if err != nil {
		return refused(t.spec.refusal, err)
	}
	plan, err := t.spec.plan(t.ledger, ctx, req, params.Actor)
	if err != nil {
		return refused(t.spec.refusal, err)
	}
	itemChange, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceFactoringSubmission,
			ID:       plan.Before.ID,
			Label:    "Invoice " + plan.Item.InvoiceNumber + " on schedule " + plan.Before.Number,
			Version:  common.PreviewVersion(plan.Before.Version),
		},
		itemViewOf(plan.Before.Item(plan.Item.ID)),
		itemViewOf(plan.Item),
		toolpreview.Labels(map[string]string{
			"status":           "Status",
			"settledOn":        "Settled",
			"chargedBackOn":    "Charged back",
			"chargebackReason": "Why it was charged back",
		}),
	)
	if err != nil {
		return nil, err
	}
	changes := []*agent.RecordChange{itemChange}
	movements := []*factoringports.PlannedMovement{plan.Movement}
	entries, err := t.movementChanges(ctx, req.TenantInfo, plan.After.CurrencyCode, movements)
	if err != nil {
		return nil, err
	}
	changes = append(changes, entries...)
	sentences := []string{t.spec.summary(plan, plan.After.CurrencyCode)}
	if len(entries) == 0 {
		sentences = append(
			sentences,
			"Nothing is booked: the factor held no reserve and paid nothing.",
		)
	} else {
		sentences = append(sentences, bookingSentences(movements)...)
	}
	return toolpreview.Build(strings.Join(sentences, " "), changes...), nil
}

func movementCash(plan *factoringports.ReserveMovementPlan) int64 {
	if plan.Movement == nil {
		return 0
	}
	return plan.Movement.Movement.CashMinor
}

func NewReleaseFactoringReserveTool(
	ledger factoringports.LedgerService,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return &ReserveMovementTool{
		ledgerTool: ledgerTool{ledger: ledger, accounts: accounts},
		spec: movementSpec{
			name: "factoring_release_reserve",
			description: "Book the factor's release of a funded invoice's reserve, once the " +
				"customer paid the factor and the factor remitted what it held back. The " +
				"cash defaults to the reserve held; anything the factor kept beyond that is " +
				"booked as its fee. The invoice's factoring is then settled. A person " +
				"approves every booking.",
			rationale: "Books the factor's reserve remittance to the general ledger; only a " +
				"person's approval runs it.",
			searchTerms: []string{"reserve release", "factor remittance", "rebate", "reserve paid"},
			dateHelp:    "The day the factor paid the reserve.",
			cashHelp: "What the factor paid, when it differs from the reserve it held; the " +
				"difference is booked to the fee account.",
			reasonHelp: "A note kept with the entry, such as the remittance number.",
			refusal:    "Would book the factor's release of an invoice's reserve.",
			plan:       factoringports.LedgerService.PlanRelease,
			run:        factoringports.LedgerService.ReleaseReserve,
			summary: func(plan *factoringports.ReserveMovementPlan, currency string) string {
				_, _, reserve := plan.Item.Actuals()
				return fmt.Sprintf(
					"Would book the factor's release of invoice %s's reserve of %s: %s in cash. "+
						"The invoice's factoring is then settled.",
					plan.Item.InvoiceNumber,
					formatMinor(currency, reserve),
					formatMinor(currency, movementCash(plan)),
				)
			},
		},
	}
}

func NewRecordFactoringChargebackTool(
	ledger factoringports.LedgerService,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return &ReserveMovementTool{
		ledgerTool: ledgerTool{ledger: ledger, accounts: accounts},
		spec: movementSpec{
			name: "factoring_record_chargeback",
			description: "Book a recourse factor's chargeback of a funded invoice the customer " +
				"did not pay. The organization repays the factor (by default the advance and " +
				"fee) and the factor gives up the reserve. Under a sale the invoice goes back on " +
				"receivables and the organization collects it again. Refused under a " +
				"non-recourse agreement. Say why. A person approves every booking.",
			rationale: "Books a repayment to the factor and reopens an invoice; only a " +
				"person's approval runs it.",
			searchTerms: []string{"chargeback", "charge back", "recourse", "repurchase invoice"},
			dateHelp:    "The day the factor charged it back.",
			cashHelp: "What the organization paid or owes the factor, when it differs from the " +
				"advance and fee; the difference is booked to the fee account.",
			reasonHelp:  "Why the factor charged it back.",
			needsReason: true,
			refusal:     "Would book the factor's chargeback of an invoice.",
			plan:        factoringports.LedgerService.PlanChargeback,
			run:         factoringports.LedgerService.RecordChargeback,
			summary: func(plan *factoringports.ReserveMovementPlan, currency string) string {
				return fmt.Sprintf(
					"Would book the factor's chargeback of invoice %s: %s paid back to the "+
						"factor, and the invoice is the organization's to collect again.",
					plan.Item.InvoiceNumber,
					formatMinor(currency, -movementCash(plan)),
				)
			},
		},
	}
}

func NewReverseFactoringChargebackTool(
	ledger factoringports.LedgerService,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return &ReserveMovementTool{
		ledgerTool: ledgerTool{ledger: ledger, accounts: accounts},
		spec: movementSpec{
			name: "factoring_reverse_chargeback",
			description: "Undo an invoice's last chargeback with the opposite entry, when the " +
				"factor withdrew it or it was recorded by mistake. The invoice goes back to " +
				"funded with the factor. Say why. A person approves every booking.",
			rationale: "Books the opposite of a chargeback to the general ledger; only a " +
				"person's approval runs it.",
			searchTerms: []string{"reverse chargeback", "undo chargeback", "chargeback withdrawn"},
			dateHelp:    "The day the chargeback was withdrawn.",
			reasonHelp:  "Why the chargeback is reversed.",
			needsReason: true,
			refusal:     "Would reverse an invoice's chargeback.",
			plan:        factoringports.LedgerService.PlanChargebackReversal,
			run:         factoringports.LedgerService.ReverseChargeback,
			summary: func(plan *factoringports.ReserveMovementPlan, currency string) string {
				return fmt.Sprintf(
					"Would reverse invoice %s's chargeback of %s, booking its lines swapped; "+
						"the invoice is funded with the factor again.",
					plan.Item.InvoiceNumber,
					formatMinor(currency, chargebackCash(plan)),
				)
			},
		},
	}
}

func chargebackCash(plan *factoringports.ReserveMovementPlan) int64 {
	if plan.Reverses == nil {
		return 0
	}
	return -plan.Reverses.CashMinor
}
