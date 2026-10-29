package receivabletools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
)

const (
	toolLogCollectionActivity      = "receivable_record_collection_activity"
	toolAssignCollector            = "receivable_assign_collector"
	toolScheduleCollectionFollowUp = "receivable_schedule_follow_up"
	toolListCollectionsWorklist    = "receivable_list_collection_worklist"
	toolGetCustomerStatement       = "receivable_get_statement"
	paramCollectionKind            = "kind"
	paramCollectionSummary         = "summary"
	paramCollectionContact         = "contactName"
	paramPromisedAmount            = "promisedAmount"
	paramPromisedOn                = "promisedOn"
	paramFollowUpOn                = "followUpOn"
	paramCollectorID               = "collectorId"
	paramCollectionNote            = "note"
	maxCollectionContactChars      = 200
	collectionTaintHold            = "What a customer said is what a collector acts on " +
		"next, and a promise to pay holds back reminders and moves the follow-up, so a run " +
		"that has read outside text proposes it rather than writing it."
)

var collectionContactKinds = agenttoolschema.Source(
	"collection.contactKind",
	[]collection.ActivityKind{
		collection.ActivityNote,
		collection.ActivityCall,
		collection.ActivityEmail,
		collection.ActivityPromiseToPay,
	},
)

var collectionDateTypes = map[string]assistantartifact.DisplayType{
	paramPromisedOn:  assistantartifact.DisplayDate,
	paramFollowUpOn:  assistantartifact.DisplayDate,
	"nextFollowUpOn": assistantartifact.DisplayDate,
}

var collectionRefs = map[string]permission.Resource{
	paramCollectorID:          permission.ResourceUser,
	toolparams.ParamInvoiceID: permission.ResourceInvoice,
}

var collectionLabels = map[string]string{
	paramCollectionKind:       "Kind",
	paramCollectionSummary:    "What happened",
	paramCollectionContact:    "Spoke to",
	toolparams.ParamInvoiceID: "Invoice",
	paramPromisedAmount:       "Promised amount",
	paramPromisedOn:           "Promised to pay by",
	paramFollowUpOn:           "Follow up on",
	paramCollectorID:          "Collector",
	"nextFollowUpOn":          "Next follow-up",
}

type collectionEntryView struct {
	Kind           string   `json:"kind"`
	Summary        string   `json:"summary,omitempty"`
	ContactName    string   `json:"contactName,omitempty"`
	InvoiceID      pulid.ID `json:"invoiceId,omitempty"`
	PromisedAmount string   `json:"promisedAmount,omitempty"`
	PromisedOn     *int64   `json:"promisedOn,omitempty"`
	FollowUpOn     *int64   `json:"followUpOn,omitempty"`
	CollectorID    pulid.ID `json:"collectorId,omitempty"`
}

type collectionAccountView struct {
	CollectorID    pulid.ID `json:"collectorId,omitempty"`
	NextFollowUpOn *int64   `json:"nextFollowUpOn,omitempty"`
	PromisedAmount string   `json:"promisedAmount,omitempty"`
	PromisedOn     *int64   `json:"promisedOn,omitempty"`
}

func collectionEntryOf(activity *collection.Activity) *collectionEntryView {
	return &collectionEntryView{
		Kind:           string(activity.Kind),
		Summary:        activity.Summary,
		ContactName:    activity.ContactName,
		InvoiceID:      activity.InvoiceID,
		PromisedAmount: minorAmount(activity.PromisedAmountMinor, activity.CurrencyCode),
		PromisedOn:     activity.PromisedOn,
		FollowUpOn:     activity.FollowUpOn,
		CollectorID:    activity.CollectorID,
	}
}

func collectionAccountOf(
	state *collectionports.AccountState,
	currency string,
) *collectionAccountView {
	return &collectionAccountView{
		CollectorID:    state.CollectorID,
		NextFollowUpOn: state.NextFollowUpOn,
		PromisedAmount: minorAmount(state.PromisedAmountMinor, currency),
		PromisedOn:     state.PromisedOn,
	}
}

func minorAmount(amount *int64, currency string) string {
	if amount == nil {
		return ""
	}

	return money.FormatMinor(*amount, currency)
}

func accountStateChanged(before, after *collectionports.AccountState) bool {
	return before.CollectorID != after.CollectorID ||
		!typeutils.EqualPtr(before.NextFollowUpOn, after.NextFollowUpOn) ||
		!typeutils.EqualPtr(before.PromisedOn, after.PromisedOn) ||
		!typeutils.EqualPtr(before.PromisedAmountMinor, after.PromisedAmountMinor)
}

func collectionCustomerProperty() map[string]any {
	return agenttoolschema.RecordIDText(permission.ResourceCustomer,
		"The customer whose receivables these are, from receivable_list_collection_worklist, "+
			"receivable_get_aging or customer_list. Never guess one.")
}

func collectionTarget(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamCustomerID, permission.ResourceCustomer)
}

func collectionCurrency(plan *collectionports.AccountChangePlan) string {
	if plan.Activity != nil && plan.Activity.CurrencyCode != "" {
		return plan.Activity.CurrencyCode
	}
	if plan.Row != nil && plan.Row.CurrencyCode != "" {
		return plan.Row.CurrencyCode
	}

	return money.DefaultCurrencyCode
}

func collectionCustomerName(plan *collectionports.AccountChangePlan) string {
	if plan.Row == nil {
		return "the customer"
	}

	return plan.Row.CustomerName
}

func renderCollectionPlan(
	customerID pulid.ID,
	plan *collectionports.AccountChangePlan,
	summary string,
) (*agentpreview.ToolPreview, error) {
	name := collectionCustomerName(plan)
	opts := []toolpreview.Option{
		toolpreview.Labels(collectionLabels),
		toolpreview.Types(collectionDateTypes),
		toolpreview.WithRefs(collectionRefs),
	}
	changes := make([]*agentpreview.RecordChange, 0, 2)
	if plan.Activity != nil {
		entry, err := toolpreview.Create(toolpreview.Record{
			Resource: permission.ResourceCustomer,
			ID:       customerID,
			Label:    "Collections history for " + name,
		}, collectionEntryOf(plan.Activity), opts...)
		if err != nil {
			return nil, err
		}
		changes = append(changes, entry)
	}
	if accountStateChanged(&plan.Before, &plan.After) {
		currency := collectionCurrency(plan)
		account, err := toolpreview.Changed(toolpreview.Record{
			Resource: permission.ResourceCustomer,
			ID:       customerID,
			Label:    "Collections account for " + name,
		}, collectionAccountOf(&plan.Before, currency),
			collectionAccountOf(&plan.After, currency), opts...)
		if err != nil {
			return nil, err
		}
		changes = append(changes, account)
	}

	return toolpreview.Build(summary, changes...), nil
}

func collectionResult(
	action string,
	customerID pulid.ID,
	view *collectionports.AccountView,
) *agentrecord.ToolExecutionResult {
	result := &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   "collections account",
		IDs:    map[string]string{toolparams.ParamCustomerID: customerID.String()},
		Record: customerRecord(customerID),
	}
	if view != nil && view.Row != nil {
		result.Name = view.Row.CustomerName
		result.State = string(view.Row.Priority)
	}

	return result
}

func customerRecord(customerID pulid.ID) *agentrecord.RecordRef {
	return &agentrecord.RecordRef{
		EntityType: changepreview.CustomerRecordEntity,
		ID:         customerID.String(),
	}
}

func NewLogCollectionActivityTool(collections collectionports.Service) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name: toolLogCollectionActivity,
			Recipe: []string{
				toolListCollectionsWorklist,
				toolGetCustomerStatement,
				toolLogCollectionActivity,
			},
			Description: "Log a call, email, note or promise to pay against a customer's " +
				"overdue receivables, so the next person chasing them knows what was said. " +
				"Record what the customer said in their words; a promise to pay takes the amount " +
				"and the date they gave, holds back payment reminders until then, and moves the " +
				"follow-up to the day after unless you give one. Never infer a promise from a " +
				"payment that has not arrived.",
			Resource:    permission.ResourceAccountsReceivable,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			TaintHold:   collectionTaintHold,
			Rationale: "Adds an entry to a customer's collections history inside Trenova; a " +
				"promise moves the follow-up and holds back reminders until its date, and " +
				"nothing is sent.",
			Target:   collectionTarget,
			Artifact: changepreview.CustomerRecordEntity,
			Properties: map[string]any{
				toolparams.ParamCustomerID: collectionCustomerProperty(),
				paramCollectionKind: agenttoolschema.Enum(
					"Call or Email for a contact with the customer, Note for anything else "+
						"worth knowing, PromiseToPay when they said they would pay an amount by "+
						"a date.",
					collectionContactKinds,
				),
				paramCollectionSummary: toolparams.StringProperty(
					"What was said, in the customer's words where you have them. Required "+
						"except for a promise to pay.",
					collection.MaxSummaryLength,
				),
				paramCollectionContact: toolparams.StringProperty(
					"Who you or the customer spoke to, if you know.",
					maxCollectionContactChars,
				),
				toolparams.ParamInvoiceID: agenttoolschema.RecordIDText(
					permission.ResourceInvoice,
					"The invoice it is about, when it is about one, from receivable_list_open_items or "+
						"receivable_get_statement.",
				),
				paramPromisedAmount: toolparams.AmountProperty(
					"For a promise to pay: the amount promised as a decimal, such as 1250.00.",
				),
				paramPromisedOn: agenttoolschema.Date(
					"For a promise to pay: the day they said they would pay by.",
				),
				paramFollowUpOn: agenttoolschema.Date(
					"When to chase the account next, today or later. Leave it out to keep the " +
						"current follow-up, or for a promise, the day after it falls due.",
				),
			},
			Required:    []string{toolparams.ParamCustomerID, paramCollectionKind},
			SearchTerms: []string{"promise to pay", "log a contact", "dunning"},
		},
		toolspec.ReceivablePlan[
			*collectionports.LogActivityRequest,
			*collectionports.AccountChangePlan,
		]{
			Request: logActivityRequest,
			Plan: func(
				ctx context.Context,
				req *collectionports.LogActivityRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*collectionports.AccountChangePlan, error) {
				return collections.PlanLogActivity(ctx, req, params.Actor)
			},
			Refused: func(*collectionports.LogActivityRequest) string {
				return "Would log an entry in the customer's collections history."
			},
			Render: func(
				req *collectionports.LogActivityRequest,
				plan *collectionports.AccountChangePlan,
			) (*agentpreview.ToolPreview, error) {
				summary := fmt.Sprintf("Would log a %s with %s.",
					activityNoun(req.Kind), collectionCustomerName(plan))
				return renderCollectionPlan(req.CustomerID, plan, summary)
			},
			Run: func(
				ctx context.Context,
				req *collectionports.LogActivityRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				view, err := collections.LogActivity(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return collectionResult("logged", req.CustomerID, view), nil
			},
		},
	)
}

func logActivityRequest(
	params *agenttoolports.ToolExecuteParams,
) (*collectionports.LogActivityRequest, error) {
	customerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamCustomerID)
	if err != nil {
		return nil, err
	}
	kind, err := toolparams.RequireEnum(
		params.Params,
		paramCollectionKind,
		collectionContactKinds.Values,
	)
	if err != nil {
		return nil, err
	}
	summary, err := toolparams.BoundedText(
		params.Params,
		paramCollectionSummary,
		collection.MaxSummaryLength,
	)
	if err != nil {
		return nil, err
	}
	contact, err := toolparams.BoundedText(
		params.Params,
		paramCollectionContact,
		maxCollectionContactChars,
	)
	if err != nil {
		return nil, err
	}
	invoiceID, _, err := toolparams.OptionalPulid(params.Params, toolparams.ParamInvoiceID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", toolparams.ParamInvoiceID, err)
	}

	req := &collectionports.LogActivityRequest{
		TenantInfo:  toolparams.TenantFrom(*params),
		CustomerID:  customerID,
		Kind:        kind,
		Summary:     summary,
		ContactName: contact,
		InvoiceID:   invoiceID,
	}
	if kind == collection.ActivityPromiseToPay {
		amount, given, amountErr := toolparams.OptionalDecimal(params.Params, paramPromisedAmount)
		if amountErr != nil {
			return nil, amountErr
		}
		if given {
			req.PromisedAmount = decimal.NewNullDecimal(amount)
		}
		if req.PromisedOn, err = optionalDayParam(params.Params, paramPromisedOn); err != nil {
			return nil, err
		}
	}
	if req.FollowUpOn, err = optionalDayParam(params.Params, paramFollowUpOn); err != nil {
		return nil, err
	}

	return req, nil
}

func optionalDayParam(params map[string]any, key string) (*int64, error) {
	day, given, err := toolparams.OptionalDay(params, key)
	if err != nil || !given {
		return nil, err
	}

	return &day, nil
}

func activityNoun(kind collection.ActivityKind) string {
	switch kind { //nolint:exhaustive // the kinds a person logs
	case collection.ActivityCall:
		return "call"
	case collection.ActivityEmail:
		return "email"
	case collection.ActivityPromiseToPay:
		return "promise to pay"
	default:
		return "note"
	}
}

func NewAssignCollectorTool(collections collectionports.Service) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:   toolAssignCollector,
			Recipe: []string{toolListCollectionsWorklist, toolAssignCollector},
			Description: "Give a customer's overdue receivables to a collector, or take them " +
				"off whoever has them. The collector must be a member of the organization and " +
				"is told the account is theirs. Use a collectorId receivable_list_collection_worklist " +
				"shows or the person who asked; leave it out only to leave the account with " +
				"nobody.",
			Resource:    permission.ResourceAccountsReceivable,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierActWithApproval,
			Reversible:  true,
			Rationale: "Changes who chases a customer inside Trenova and tells the new " +
				"collector; nothing is sent to the customer and assigning someone else changes " +
				"it back.",
			Target:   collectionTarget,
			Artifact: changepreview.CustomerRecordEntity,
			Properties: map[string]any{
				toolparams.ParamCustomerID: collectionCustomerProperty(),
				paramCollectorID: agenttoolschema.RecordIDText(
					permission.ResourceUser,
					"The collector's user id: a collectorId receivable_list_collection_worklist shows, or "+
						"the person who asked. Leave it out to take the account off whoever has "+
						"it. Never guess one.",
				),
			},
			Required:    []string{toolparams.ParamCustomerID},
			SearchTerms: []string{"collector", "assign collections", "who chases"},
		},
		toolspec.ReceivablePlan[
			*collectionports.AssignCollectorRequest,
			*collectionports.AccountChangePlan,
		]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*collectionports.AssignCollectorRequest, error) {
				customerID, err := toolparams.RequirePulid(
					params.Params,
					toolparams.ParamCustomerID,
				)
				if err != nil {
					return nil, err
				}
				req := &collectionports.AssignCollectorRequest{
					TenantInfo: toolparams.TenantFrom(*params),
					CustomerID: customerID,
				}
				collector, err := toolparams.OptionalPulidParam(params.Params, paramCollectorID)
				if err != nil {
					return nil, err
				}
				if collector != nil {
					req.CollectorID = *collector
				}
				return req, nil
			},
			Plan: func(
				ctx context.Context,
				req *collectionports.AssignCollectorRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*collectionports.AccountChangePlan, error) {
				return collections.PlanAssignCollector(ctx, req, params.Actor)
			},
			Refused: func(*collectionports.AssignCollectorRequest) string {
				return "Would change who collects from the customer."
			},
			Render: func(
				req *collectionports.AssignCollectorRequest,
				plan *collectionports.AccountChangePlan,
			) (*agentpreview.ToolPreview, error) {
				name := collectionCustomerName(plan)
				summary := fmt.Sprintf("Would give %s's overdue receivables to %s.",
					name, plan.After.CollectorName)
				switch {
				case !plan.Changed:
					summary = fmt.Sprintf(
						"%s already has this collector; nothing would change.",
						name,
					)
				case req.CollectorID.IsNil():
					summary = fmt.Sprintf("Would take %s's overdue receivables off %s.",
						name, plan.Before.CollectorName)
				case plan.NotifiesCollector:
					summary = fmt.Sprintf("Would give %s's overdue receivables to %s, who is told "+
						"the account is theirs.", name, plan.After.CollectorName)
				}
				return renderCollectionPlan(req.CustomerID, plan, summary)
			},
			Run: func(
				ctx context.Context,
				req *collectionports.AssignCollectorRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				view, err := collections.AssignCollector(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return collectionResult("assigned", req.CustomerID, view), nil
			},
		},
	)
}

func NewScheduleCollectionFollowUpTool(
	collections collectionports.Service,
) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(
		&toolspec.ReceivableSpec{
			Name:   toolScheduleCollectionFollowUp,
			Recipe: []string{toolListCollectionsWorklist, toolScheduleCollectionFollowUp},
			Description: "Set the day a customer's overdue receivables are next chased, or " +
				"clear the follow-up, with a note saying why. The account comes back up the " +
				"worklist on that day. Leave followUpOn out only to clear it.",
			Resource:    permission.ResourceAccountsReceivable,
			Operation:   permission.OpUpdate,
			Egress:      agent.EgressInternal,
			DefaultTier: agent.TierPropose,
			MaxTier:     agent.TierAutoExecute,
			Reversible:  true,
			TaintHold:   collectionTaintHold,
			Rationale: "Changes when a customer is next chased inside Trenova and notes why; " +
				"nothing is sent and another follow-up replaces it.",
			Target:   collectionTarget,
			Artifact: changepreview.CustomerRecordEntity,
			Properties: map[string]any{
				toolparams.ParamCustomerID: collectionCustomerProperty(),
				paramFollowUpOn: agenttoolschema.Date(
					"The day to chase the account next, today or later. Leave it out to clear " +
						"the follow-up.",
				),
				paramCollectionNote: toolparams.StringProperty(
					"Why then, kept in the account's history.",
					collection.MaxSummaryLength,
				),
			},
			Required:    []string{toolparams.ParamCustomerID},
			SearchTerms: []string{"follow up", "chase later", "collections reminder date"},
		},
		toolspec.ReceivablePlan[
			*collectionports.ScheduleFollowUpRequest,
			*collectionports.AccountChangePlan,
		]{
			Request: func(
				params *agenttoolports.ToolExecuteParams,
			) (*collectionports.ScheduleFollowUpRequest, error) {
				customerID, err := toolparams.RequirePulid(
					params.Params,
					toolparams.ParamCustomerID,
				)
				if err != nil {
					return nil, err
				}
				note, err := toolparams.BoundedText(
					params.Params,
					paramCollectionNote,
					collection.MaxSummaryLength,
				)
				if err != nil {
					return nil, err
				}
				followUpOn, err := optionalDayParam(params.Params, paramFollowUpOn)
				if err != nil {
					return nil, err
				}
				return &collectionports.ScheduleFollowUpRequest{
					TenantInfo: toolparams.TenantFrom(*params),
					CustomerID: customerID,
					FollowUpOn: followUpOn,
					Note:       note,
				}, nil
			},
			Plan: func(
				ctx context.Context,
				req *collectionports.ScheduleFollowUpRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*collectionports.AccountChangePlan, error) {
				return collections.PlanScheduleFollowUp(ctx, req, params.Actor)
			},
			Refused: func(*collectionports.ScheduleFollowUpRequest) string {
				return "Would change when the customer is next chased."
			},
			Render: func(
				req *collectionports.ScheduleFollowUpRequest,
				plan *collectionports.AccountChangePlan,
			) (*agentpreview.ToolPreview, error) {
				summary := fmt.Sprintf("Would schedule when %s is next chased.",
					collectionCustomerName(plan))
				if req.FollowUpOn == nil {
					summary = fmt.Sprintf("Would clear the follow-up on %s.",
						collectionCustomerName(plan))
				}
				return renderCollectionPlan(req.CustomerID, plan, summary)
			},
			Run: func(
				ctx context.Context,
				req *collectionports.ScheduleFollowUpRequest,
				params *agenttoolports.ToolExecuteParams,
			) (*agentrecord.ToolExecutionResult, error) {
				view, err := collections.ScheduleFollowUp(ctx, req, params.Actor)
				if err != nil {
					return nil, err
				}
				return collectionResult("scheduled", req.CustomerID, view), nil
			},
		},
	)
}
