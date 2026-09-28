package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

type correctChargeCodeTool struct {
	billing serviceports.BillingQueueService
}

func newCorrectChargeCodeTool(billing serviceports.BillingQueueService) serviceports.AgentTool {
	return &correctChargeCodeTool{billing: billing}
}

func (t *correctChargeCodeTool) Name() string { return "correct_charge_code" }

func (t *correctChargeCodeTool) Description() string {
	return "Correct or normalize the accessorial charge codes on a billing queue item's charges. " +
		"Use it when a charge carries the wrong accessorial; the set you send replaces the " +
		"item's additional charges, so include every charge that should remain, each with " +
		"an accessorialChargeId from list_accessorial_charges. The item must be InReview " +
		"first (transition_item_to_in_review)."
}

func (t *correctChargeCodeTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"billingQueueItemId": map[string]any{
				"type": "string",
				"description": "The billing queue item whose charges are corrected: this run's " +
					"subject or the record on the page. No tool lists queue items, so never " +
					"guess one.",
			},
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

const (
	chargeLineID          = "id"
	chargeLineAccessorial = "accessorialChargeId"
	chargeLineMethod      = "method"
	chargeLineUnit        = "unit"
)

// chargeLineSchema is one additional charge as the charge plan reads it:
// the fields shipment.AdditionalCharge takes from a caller, and nothing the
// system owns.
func chargeLineSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			chargeLineID: stringProperty("The existing charge this line keeps or corrects, "+
				"from the item's charges; leave it out for a charge that is new.", 0),
			chargeLineAccessorial: stringProperty(
				"The accessorial, from list_accessorial_charges.", 0,
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
			paramAmount: map[string]any{
				toolschema.KeyType: []string{toolschema.TypeString, "number"},
				toolschema.KeyDescription: "The charge's amount in major units, as a decimal " +
					"such as 125.00.",
			},
		},
		toolschema.KeyRequired: []string{
			chargeLineAccessorial, chargeLineMethod, chargeLineUnit, paramAmount,
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *correctChargeCodeTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceBillingQueue,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Rewrites the accessorial charges a customer will be invoiced, so it " +
			"moves money.",
	}
}

func (t *correctChargeCodeTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
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
func (t *correctChargeCodeTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return previewValidates(ctx, t, &params)
}

// request is the charge edit the preview and the write both make: the
// item's additional charges replaced by the set the call sends.
func (t *correctChargeCodeTool) request(
	params *serviceports.ToolExecuteParams,
) (*serviceports.UpdateChargesRequest, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}

	itemID, err := requirePulid(params.Params, "billingQueueItemId")
	if err != nil {
		return nil, err
	}

	var charges []*shipment.AdditionalCharge
	if err = decodeParam(params.Params, "additionalCharges", &charges); err != nil {
		return nil, err
	}

	return &serviceports.UpdateChargesRequest{
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
func (t *correctChargeCodeTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "billingQueueItemId", permission.ResourceBillingQueue)
}
