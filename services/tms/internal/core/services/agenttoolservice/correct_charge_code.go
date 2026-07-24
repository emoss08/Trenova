package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

type correctChargeCodeTool struct {
	billing serviceports.BillingQueueService
}

func newCorrectChargeCodeTool(billing serviceports.BillingQueueService) serviceports.AgentTool {
	return &correctChargeCodeTool{billing: billing}
}

func (t *correctChargeCodeTool) Name() string { return "correct_charge_code" }

func (t *correctChargeCodeTool) Description() string {
	return "Correct or normalize the accessorial charge codes on a billing queue item's charges."
}

func (t *correctChargeCodeTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"billingQueueItemId": map[string]any{
				"type":        "string",
				"description": "The id of the billing queue item whose charges are corrected.",
			},
			"additionalCharges": map[string]any{
				"type":        "array",
				"description": "The corrected additional charge set to apply to the item.",
				"items":       map[string]any{"type": "object"},
			},
		},
		"required":             []string{"billingQueueItemId", "additionalCharges"},
		"additionalProperties": false,
	}
}

func (t *correctChargeCodeTool) Reversible() bool { return true }

func (t *correctChargeCodeTool) PermissionResource() permission.Resource {
	return permission.ResourceBillingQueue
}

func (t *correctChargeCodeTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *correctChargeCodeTool) RequiresIdempotencyKey() bool { return false }

func (t *correctChargeCodeTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierPropose
}

func (t *correctChargeCodeTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	itemID, err := requirePulid(params.Params, "billingQueueItemId")
	if err != nil {
		return err
	}

	var charges []*shipment.AdditionalCharge
	if err = decodeParam(params.Params, "additionalCharges", &charges); err != nil {
		return err
	}

	_, err = t.billing.UpdateCharges(ctx, &serviceports.UpdateChargesRequest{
		ItemID:            itemID,
		AdditionalCharges: charges,
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
	}, params.Actor)

	return err
}
