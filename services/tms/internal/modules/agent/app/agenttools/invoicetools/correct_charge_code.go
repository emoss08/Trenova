package invoicetools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

type CorrectChargeCodeTool struct {
	billing  billingqueueports.BillingQueueService
	profiles billingProfileReader
}

type billingProfileReader interface {
	GetBillingProfile(
		ctx context.Context,
		req customerports.GetCustomerBillingProfileRequest,
	) (*customer.CustomerBillingProfile, error)
}

func NewCorrectChargeCodeTool(
	billing billingqueueports.BillingQueueService,
	profiles billingProfileReader,
) agenttoolports.AgentTool {
	return &CorrectChargeCodeTool{billing: billing, profiles: profiles}
}

func (t *CorrectChargeCodeTool) Name() string { return "billing_correct_charge_codes" }

func (t *CorrectChargeCodeTool) Description() string {
	return "Correct or normalize the accessorial charge codes on a billing queue item's charges. " +
		"Use it when a charge carries the wrong accessorial; the set you send replaces the " +
		"item's additional charges, so include every charge that should remain, each with " +
		"an accessorialChargeId from masterdata_list_accessorial_charges. The item must be InReview " +
		"first (billing_start_review)."
}

func (t *CorrectChargeCodeTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"billingQueueItemId": agenttoolschema.KindID(
				"The billing queue item whose charges are corrected, from "+
					"billing_list_queue_items, billing_get_queue_item, this run's subject or "+
					"the record on the page. Never guess one.",
				recordkind.BillingQueueItem,
			),
			"additionalCharges": map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The corrected additional charge set to apply to the item.",
				toolschema.KeyItems:       chargeLineSchema(),
			},
		},
		"required":             []string{"billingQueueItemId", "additionalCharges"},
		"additionalProperties": false,
	}
}

const chargeLineID = "id"

const chargeLineAccessorial = "accessorialChargeId"

const chargeLineMethod = "method"

const chargeLineUnit = "unit"

// chargeLineSchema is one additional charge as the charge plan reads it:
// the fields shipment.AdditionalCharge takes from a caller, and nothing the
// system owns.
func chargeLineSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			chargeLineID: agenttoolschema.KindID("The existing charge this line keeps or "+
				"corrects, from the item's charges; leave it out for a charge that is new.",
				recordkind.AdditionalCharge),
			chargeLineAccessorial: agenttoolschema.RecordIDText(
				permission.ResourceAccessorialCharge,
				"The accessorial, from masterdata_list_accessorial_charges.",
			),
			chargeLineMethod: agenttoolschema.Enum(
				"How the amount is applied: Flat once, PerUnit times the unit, "+
					"Percentage of the linehaul.",
				agenttoolschema.AccessorialMethods,
			),
			chargeLineUnit: map[string]any{
				toolschema.KeyType:        toolschema.TypeInteger,
				toolschema.KeyMinimum:     1,
				toolschema.KeyDescription: "How many units the charge covers; 1 for a flat charge.",
			},
			toolparams.ParamAmount: map[string]any{
				toolschema.KeyType: []string{toolschema.TypeString, "number"},
				toolschema.KeyDescription: "The charge's amount in major units, as a decimal " +
					"such as 125.00.",
			},
		},
		toolschema.KeyRequired: []string{
			chargeLineAccessorial, chargeLineMethod, chargeLineUnit, toolparams.ParamAmount,
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *CorrectChargeCodeTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:        t.Name(),
		Kind:        agenttooltrust.ToolKindAction,
		Resource:    permission.ResourceBillingQueue,
		Operation:   permission.OpUpdate,
		Scope:       agenttooltrust.ToolScopeTenant,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierAutoExecute,
		Egress:      []agent.EgressClass{agent.EgressMoney},
		Condition: &agenttoolports.TierCondition{
			Description: correctChargeBoundText,
			Bounds:      t.withinBounds,
		},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Rewrites the accessorial charges a customer will be invoiced, so it " +
			"moves money.",
	}
}

func (t *CorrectChargeCodeTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.billing.UpdateCharges(ctx, request, params.Actor)

	return err
}

// Validate runs the charge plan the preview runs, so an item that is not in
// review, or a set of charges the service would refuse, is refused to the
// model before it is proposed.
func (t *CorrectChargeCodeTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

// request is the charge edit the preview and the write both make: the
// item's additional charges replaced by the set the call sends.
func (t *CorrectChargeCodeTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*billingqueueports.UpdateChargesRequest, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	itemID, err := toolparams.RequirePulid(params.Params, "billingQueueItemId")
	if err != nil {
		return nil, err
	}

	var charges []*shipment.AdditionalCharge
	if err = toolparams.DecodeParam(params.Params, "additionalCharges", &charges); err != nil {
		return nil, err
	}

	return &billingqueueports.UpdateChargesRequest{
		ItemID:            itemID,
		AdditionalCharges: charges,
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *CorrectChargeCodeTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, "billingQueueItemId", permission.ResourceBillingQueue)
}
