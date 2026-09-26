package agenttoolservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/carriersettlementservice"
	"github.com/emoss08/trenova/internal/core/services/driversettlementservice"
	"github.com/emoss08/trenova/internal/core/services/settlementshared"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	paramAdjustmentAmount      = "amount"
	paramAdjustmentDescription = "description"
	paramAdjustmentQuantity    = "quantity"
	paramAdjustmentRate        = "rate"
	paramPayCodeID             = "payCodeId"
	paramGLAccountID           = "glAccountId"
	paramLineID                = "lineId"
	maxAdjustmentDescription   = 255
	fieldSubmittedAt           = "submittedAt"
	fieldApprovedAt            = "approvedAt"
	fieldPostedAt              = "postedAt"
	fieldPaidAt                = "paidAt"
	fieldVoidedAt              = "voidedAt"
	fieldHasExceptions         = "hasExceptions"
)

type lifecycleText struct {
	noun        string
	payee       string
	getTool     string
	methods     []string
	postEgress  []agent.EgressClass
	paidEgress  []agent.EgressClass
	paidMeaning string
	postMeaning string
	adjustment  func() (map[string]any, []string)
	fillAdjust  func(map[string]any, *settlementshared.ActionRequest) error
}

func (l *lifecycleText) name(verb string) string {
	return verb + "_" + strings.ReplaceAll(l.noun, " ", "_")
}

func submitDecision(text *lifecycleText) *settlementDecision {
	return &settlementDecision{
		name: text.name("submit"),
		description: "Submit a draft " + text.noun + " for approval once its lines are " +
			"right. It moves to pending approval, where a person approves it or sends it " +
			"back. Read it with " + text.getTool + " first; one with exceptions should be " +
			"fixed or explained before it goes.",
		action:     settlementshared.ActionSubmit,
		operation:  permission.OpSubmit,
		egress:     []agent.EgressClass{agent.EgressInternal},
		defaultTo:  agent.TierPropose,
		maxTier:    agent.TierAutoExecute,
		reversible: true,
		rationale: "Moves a draft into the approval queue inside Trenova; nothing is paid and " +
			"a reviewer sends it back to draft.",
		fields:   []string{fieldStatus, fieldSubmittedAt},
		volatile: []string{fieldSubmittedAt},
	}
}

func approveDecision(text *lifecycleText) *settlementDecision {
	return &settlementDecision{
		name: text.name("approve"),
		description: "Propose approving a " + text.noun + " that is pending approval, " +
			"which commits the organization to paying the " + text.payee + " what it comes to. " +
			"A person always decides. Propose it only when " + text.getTool + " shows no " +
			"open exception or dispute you cannot explain.",
		action:     settlementshared.ActionApprove,
		operation:  permission.OpApprove,
		personOnly: true,
		egress:     text.postEgress,
		defaultTo:  agent.TierPropose,
		maxTier:    agent.TierPropose,
		rationale: "Commits the organization to what the " + text.payee + " is paid; only " +
			"a person approves, and hands-off approval is the settlement control's rule.",
		fields:   []string{fieldStatus, fieldApprovedAt},
		volatile: []string{fieldApprovedAt},
	}
}

func rejectDecision(text *lifecycleText) *settlementDecision {
	return &settlementDecision{
		name: text.name("reject"),
		description: "Send a " + text.noun + " that is pending approval back to draft, " +
			"with the reason. Use it when a line, a rate or a deduction is wrong and " +
			"someone must fix it before it is approved; the reason is added to its notes.",
		action:           settlementshared.ActionReject,
		operation:        permission.OpReject,
		egress:           []agent.EgressClass{agent.EgressInternal},
		defaultTo:        agent.TierPropose,
		maxTier:          agent.TierAutoExecute,
		reversible:       true,
		holdsWhenTainted: true,
		rationale: "Returns a settlement to draft with a note inside Trenova; nothing is " +
			"paid and it is submitted again once fixed.",
		properties: map[string]any{
			paramSettlementReason: settlementReasonProperty("What is wrong and what would " +
				"fix it, in a sentence payroll can act on."),
		},
		required: []string{paramSettlementReason},
		fill:     fillSettlementReason,
		fields:   []string{fieldStatus, fieldNotes},
	}
}

func postDecision(text *lifecycleText) *settlementDecision {
	return &settlementDecision{
		name: text.name("post"),
		description: "Propose posting an approved " + text.noun + " to the general " +
			"ledger. " + text.postMeaning + " It cannot be undone except by voiding, so a " +
			"person always decides.",
		action:     settlementshared.ActionPost,
		operation:  permission.OpApprove,
		personOnly: true,
		egress:     text.postEgress,
		defaultTo:  agent.TierPropose,
		maxTier:    agent.TierPropose,
		rationale: "Books the settlement's payable to the ledger and queues it for the " +
			"accounting system; only a person posts.",
		fields:   []string{fieldStatus, fieldPostedAt},
		volatile: []string{fieldPostedAt},
	}
}

func markPaidDecision(text *lifecycleText) *settlementDecision {
	properties := settlementPaymentProperties(text.methods)

	return &settlementDecision{
		name: text.name("record") + "_payment",
		description: "Propose recording that a posted " + text.noun + " was paid, with how " +
			"and the reference. " + text.paidMeaning + " A person always decides, and only " +
			"once the money has actually gone out.",
		action:     settlementshared.ActionMarkPaid,
		operation:  permission.OpUpdate,
		personOnly: true,
		egress:     text.paidEgress,
		defaultTo:  agent.TierPropose,
		maxTier:    agent.TierPropose,
		rationale: "Records a payment to the " + text.payee + " and queues it for the " +
			"accounting system; only a person says money went out.",
		properties: properties,
		required:   []string{paramPaymentMethod},
		fill:       fillSettlementPayment(text.methods),
		fields: []string{
			fieldStatus, fieldPaidAt, paramPaymentMethod, paramPaymentReference,
		},
		volatile: []string{fieldPaidAt},
	}
}

func voidDecision(text *lifecycleText) *settlementDecision {
	return &settlementDecision{
		name: text.name("void"),
		description: "Propose voiding a " + text.noun + " that is not yet paid, with the " +
			"reason. A posted one has its journal entry reversed. Its pay returns to the " +
			"pool to be settled again, so a person always decides.",
		action:     settlementshared.ActionVoid,
		operation:  permission.OpCancel,
		personOnly: true,
		egress:     []agent.EgressClass{agent.EgressMoney},
		defaultTo:  agent.TierPropose,
		maxTier:    agent.TierPropose,
		rationale: "Cancels a settlement for good and reverses any posting; only a person " +
			"voids.",
		properties: map[string]any{
			paramSettlementReason: settlementReasonProperty("Why it must not be paid, such " +
				"as a duplicate of another settlement."),
		},
		required: []string{paramSettlementReason},
		fill:     fillSettlementReason,
		fields:   []string{fieldStatus, fieldVoidedAt, "voidReason"},
		volatile: []string{fieldVoidedAt},
	}
}

func recalculateDecision(text *lifecycleText) *settlementDecision {
	return &settlementDecision{
		name: text.name("recalculate"),
		description: "Recalculate a draft " + text.noun + " from what has accrued for its " +
			"period, keeping manual adjustments. Use it after a late load, a corrected " +
			"rate or a released hold changed what the " + text.payee + " is owed.",
		action:    settlementshared.ActionRecalculate,
		operation: permission.OpUpdate,
		egress:    []agent.EgressClass{agent.EgressInternal},
		defaultTo: agent.TierPropose,
		maxTier:   agent.TierAutoExecute,
		rationale: "Rebuilds a draft from the records it is computed from inside Trenova; " +
			"nothing is paid until a person approves it.",
		fields: []string{
			fieldHasExceptions, "shipmentCount", "payProfileName", "classification",
		},
	}
}

func addAdjustmentDecision(text *lifecycleText) *settlementDecision {
	properties, required := text.adjustment()

	return &settlementDecision{
		name: text.name("add") + "_adjustment",
		description: "Add a manual adjustment line to a draft or pending " + text.noun +
			": a positive amount pays the " + text.payee + " more, one below zero takes " +
			"money back. The settlement is flagged as manually adjusted for whoever " +
			"approves it. Say why in the description.",
		action:           settlementshared.ActionAddAdjustment,
		operation:        permission.OpUpdate,
		egress:           []agent.EgressClass{agent.EgressMoney},
		defaultTo:        agent.TierPropose,
		maxTier:          agent.TierAutoExecute,
		reversible:       true,
		holdsWhenTainted: true,
		rationale: "Changes what the " + text.payee + " will be paid on a settlement a " +
			"person still approves, so it moves money; the line is removed the same way.",
		properties: properties,
		required:   required,
		fill:       text.fillAdjust,
		fields:     []string{fieldHasExceptions},
	}
}

func removeAdjustmentDecision(text *lifecycleText) *settlementDecision {
	return &settlementDecision{
		name: text.name("remove") + "_adjustment",
		description: "Remove a manual adjustment line from a draft or pending " + text.noun +
			". Only a line someone added by hand can be removed; earnings and deductions " +
			"computed from pay records are corrected at their source and recalculated.",
		action:     settlementshared.ActionRemoveAdjustment,
		operation:  permission.OpUpdate,
		egress:     []agent.EgressClass{agent.EgressMoney},
		defaultTo:  agent.TierPropose,
		maxTier:    agent.TierAutoExecute,
		reversible: true,
		rationale: "Changes what the " + text.payee + " will be paid on a settlement a " +
			"person still approves, so it moves money; the line is added back the same way.",
		properties: map[string]any{
			paramLineID: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The adjustment line's id, from the lines " +
					text.getTool + " lists. Never guess one.",
			},
		},
		required: []string{paramLineID},
		fill:     fillAdjustmentLine,
		fields:   []string{fieldHasExceptions},
	}
}

func fillAdjustmentLine(params map[string]any, req *settlementshared.ActionRequest) error {
	lineID, err := requirePulid(params, paramLineID)
	if err != nil {
		return err
	}
	req.LineID = lineID

	return nil
}

func adjustmentBase(payee string) map[string]any {
	return map[string]any{
		paramAdjustmentDescription: stringProperty("What the adjustment is for, as the "+
			payee+" and payroll will read it on the statement.", maxAdjustmentDescription),
		paramAdjustmentAmount: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "The amount as a decimal such as 125.00; negative " +
				"to take money back. Never zero.",
		},
	}
}

func fillAdjustmentBase(
	params map[string]any,
	req *settlementshared.ActionRequest,
) (*settlementshared.AdjustmentInput, error) {
	description, err := boundedString(
		params,
		paramAdjustmentDescription,
		maxAdjustmentDescription,
		true,
	)
	if err != nil {
		return nil, err
	}
	amount, err := requireSignedMoney(params, paramAdjustmentAmount)
	if err != nil {
		return nil, err
	}
	req.Adjustment = &settlementshared.AdjustmentInput{
		Description: description,
		AmountMinor: amount,
	}

	return req.Adjustment, nil
}

func driverAdjustmentSchema() (properties map[string]any, required []string) {
	properties = adjustmentBase("driver")
	properties[paramAdjustmentQuantity] = map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "How many units the amount is for, such as hours of " +
			"detention, when it is a rate times a quantity.",
	}
	properties[paramAdjustmentRate] = map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: "The rate per unit, when it is a rate times a quantity.",
	}
	properties[paramPayCodeID] = map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The pay code it posts under, from list_pay_codes, so " +
			"it reaches that code's GL account. Leave it out for the default account.",
	}

	return properties, []string{paramAdjustmentDescription, paramAdjustmentAmount}
}

func fillDriverAdjustment(params map[string]any, req *settlementshared.ActionRequest) error {
	input, err := fillAdjustmentBase(params, req)
	if err != nil {
		return err
	}
	if input.Quantity, _, err = optionalDecimal(params, paramAdjustmentQuantity); err != nil {
		return err
	}
	if input.Rate, _, err = optionalDecimal(params, paramAdjustmentRate); err != nil {
		return err
	}
	payCodeID, err := optionalPulidParam(params, paramPayCodeID)
	if err != nil {
		return err
	}
	input.PayCodeID = payCodeID

	return nil
}

func carrierAdjustmentSchema() (properties map[string]any, required []string) {
	properties = adjustmentBase("carrier")
	properties[paramGLAccountID] = map[string]any{
		toolschema.KeyType: toolschema.TypeString,
		toolschema.KeyDescription: "The expense account it posts to, from list_gl_accounts, " +
			"when it is not the default purchased transportation account.",
	}

	return properties, []string{paramAdjustmentDescription, paramAdjustmentAmount}
}

func fillCarrierAdjustment(params map[string]any, req *settlementshared.ActionRequest) error {
	input, err := fillAdjustmentBase(params, req)
	if err != nil {
		return err
	}
	glAccountID, err := optionalPulidParam(params, paramGLAccountID)
	if err != nil {
		return err
	}
	input.GLAccountID = glAccountID

	return nil
}

func driverLifecycle() *lifecycleText {
	return &lifecycleText{
		noun:    nounDriverSettlement,
		payee:   "driver",
		getTool: "get_driver_settlement",
		methods: driverPaymentMethods,
		postEgress: []agent.EgressClass{
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		paidEgress: []agent.EgressClass{
			agent.EgressDriverVisible,
			agent.EgressMoney,
		},
		postMeaning: "Posting books the driver pay expense and settlements payable, queues " +
			"it for the accounting system and tells the driver in the driver portal.",
		paidMeaning: "The driver is told in the driver portal.",
		adjustment:  driverAdjustmentSchema,
		fillAdjust:  fillDriverAdjustment,
	}
}

func carrierLifecycle() *lifecycleText {
	return &lifecycleText{
		noun:       nounCarrierSettlement,
		payee:      "carrier",
		getTool:    "get_carrier_settlement",
		methods:    carrierPaymentMethods,
		postEgress: []agent.EgressClass{agent.EgressMoney},
		paidEgress: []agent.EgressClass{agent.EgressMoney},
		postMeaning: "Posting books purchased transportation against accounts payable, " +
			"records the bill on the carrier's ledger and queues it for the accounting system.",
		paidMeaning: "Recording it books the payment against accounts payable and cash.",
		adjustment:  carrierAdjustmentSchema,
		fillAdjust:  fillCarrierAdjustment,
	}
}

func newSettlementDecisionTool[E any](
	ledger settlementLedger[E],
	decision *settlementDecision,
	book settlementBook[E],
) serviceports.AgentTool {
	return &settlementDecisionTool[E]{decision: *decision, ledger: ledger, book: book}
}

func driverDecisionTool(
	decision func(*lifecycleText) *settlementDecision,
	book settlementBook[driversettlement.Settlement],
) serviceports.AgentTool {
	return newSettlementDecisionTool(driverSettlementLedger(), decision(driverLifecycle()), book)
}

func carrierDecisionTool(
	decision func(*lifecycleText) *settlementDecision,
	book settlementBook[carriersettlement.CarrierSettlement],
) serviceports.AgentTool {
	return newSettlementDecisionTool(
		carrierSettlementLedger(),
		decision(carrierLifecycle()),
		book,
	)
}

func provideSubmitDriverSettlementTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return driverDecisionTool(submitDecision, s)
}

func provideApproveDriverSettlementTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return driverDecisionTool(approveDecision, s)
}

func provideRejectDriverSettlementTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return driverDecisionTool(rejectDecision, s)
}

func providePostDriverSettlementTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return driverDecisionTool(postDecision, s)
}

func provideMarkDriverSettlementPaidTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return driverDecisionTool(markPaidDecision, s)
}

func provideVoidDriverSettlementTool(s *driversettlementservice.Service) serviceports.AgentTool {
	return driverDecisionTool(voidDecision, s)
}

func provideRecalculateDriverSettlementTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return driverDecisionTool(recalculateDecision, s)
}

func provideAddDriverSettlementAdjustmentTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return driverDecisionTool(addAdjustmentDecision, s)
}

func provideRemoveDriverSettlementAdjustmentTool(
	s *driversettlementservice.Service,
) serviceports.AgentTool {
	return driverDecisionTool(removeAdjustmentDecision, s)
}

func provideSubmitCarrierSettlementTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return carrierDecisionTool(submitDecision, s)
}

func provideApproveCarrierSettlementTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return carrierDecisionTool(approveDecision, s)
}

func provideRejectCarrierSettlementTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return carrierDecisionTool(rejectDecision, s)
}

func providePostCarrierSettlementTool(s *carriersettlementservice.Service) serviceports.AgentTool {
	return carrierDecisionTool(postDecision, s)
}

func provideMarkCarrierSettlementPaidTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return carrierDecisionTool(markPaidDecision, s)
}

func provideVoidCarrierSettlementTool(s *carriersettlementservice.Service) serviceports.AgentTool {
	return carrierDecisionTool(voidDecision, s)
}

func provideRecalculateCarrierSettlementTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return carrierDecisionTool(recalculateDecision, s)
}

func provideAddCarrierSettlementAdjustmentTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return carrierDecisionTool(addAdjustmentDecision, s)
}

func provideRemoveCarrierSettlementAdjustmentTool(
	s *carriersettlementservice.Service,
) serviceports.AgentTool {
	return carrierDecisionTool(removeAdjustmentDecision, s)
}

func settlementDecisionProviders() []any {
	return []any{
		provideSubmitDriverSettlementTool,
		provideApproveDriverSettlementTool,
		provideRejectDriverSettlementTool,
		providePostDriverSettlementTool,
		provideMarkDriverSettlementPaidTool,
		provideVoidDriverSettlementTool,
		provideRecalculateDriverSettlementTool,
		provideAddDriverSettlementAdjustmentTool,
		provideRemoveDriverSettlementAdjustmentTool,
		provideSubmitCarrierSettlementTool,
		provideApproveCarrierSettlementTool,
		provideRejectCarrierSettlementTool,
		providePostCarrierSettlementTool,
		provideMarkCarrierSettlementPaidTool,
		provideVoidCarrierSettlementTool,
		provideRecalculateCarrierSettlementTool,
		provideAddCarrierSettlementAdjustmentTool,
		provideRemoveCarrierSettlementAdjustmentTool,
	}
}

var (
	_ settlementBook[driversettlement.Settlement]         = (*driversettlementservice.Service)(nil)
	_ settlementBook[carriersettlement.CarrierSettlement] = (*carriersettlementservice.Service)(nil)
	_ serviceports.ToolPreviewer                          = (*settlementDecisionTool[driversettlement.Settlement])(
		nil,
	)
	_ serviceports.ToolValidator = (*settlementDecisionTool[driversettlement.Settlement])(
		nil,
	)
	_ serviceports.TargetedTool = (*settlementDecisionTool[driversettlement.Settlement])(
		nil,
	)
)
