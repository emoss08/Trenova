package allocationparams

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func AllocationsProperty(description string) map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: description,
		toolschema.KeyMaxItems:    maxChargeAllocations,
		toolschema.KeyItems: map[string]any{
			toolschema.KeyType: toolschema.TypeObject,
			toolschema.KeyProperties: map[string]any{
				ParamBillToCustomer: agenttoolschema.RecordIDText(
					permission.ResourceCustomer,
					"The customer who pays this share, from "+
						"customer_list. Never guess one.",
				),
				ParamAllocMethod: agenttoolschema.Enum("Whether the share is a percent of "+
					"the charge or a fixed amount.", allocationMethods),
				ParamAllocPercent: toolparams.AmountProperty("The share as a percent such as 60, " +
					"when method is Percent."),
				toolparams.ParamAmount: toolparams.AmountProperty(
					"The share as an amount such as 250.00, when " +
						"method is Amount.",
				),
			},
			toolschema.KeyRequired:             []string{ParamBillToCustomer, ParamAllocMethod},
			toolschema.KeyAdditionalProperties: false,
		},
	}
}

type allocationArg struct {
	BillToCustomerID string `json:"billToCustomerId"`
	Method           string `json:"method"`
	Percent          any    `json:"percent"`
	Amount           any    `json:"amount"`
}

// readAllocations reads a charge's payer split. Absent is nil, which leaves the
// split alone; an empty list bills the whole charge to the order's customer.
// A payer already on the charge keeps its row, so the split is edited rather
// than replaced.
func ReadAllocations(
	params map[string]any,
	tenant pagination.TenantInfo,
	kind shipmentvocab.ChargeAllocationKind,
	current []*shipment.ChargeAllocation,
) ([]*shipment.ChargeAllocation, error) {
	if _, given := params[ParamAllocations]; !given {
		return nil, nil
	}

	var args []allocationArg
	if err := toolparams.DecodeParam(params, ParamAllocations, &args); err != nil {
		return nil, err
	}
	if len(args) > maxChargeAllocations {
		return nil, fmt.Errorf("parameter %q holds %d payers; at most %d split one charge",
			ParamAllocations, len(args), maxChargeAllocations)
	}

	existing := make(map[pulid.ID]*shipment.ChargeAllocation, len(current))
	for _, row := range current {
		if row != nil {
			existing[row.BillToCustomerID] = row
		}
	}

	rows := make([]*shipment.ChargeAllocation, 0, len(args))
	for index, arg := range args {
		row, err := allocationRow(index, arg, tenant, kind)
		if err != nil {
			return nil, err
		}
		if kept, ok := existing[row.BillToCustomerID]; ok {
			row.ID = kept.ID
			row.Version = kept.Version
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func allocationRow(
	index int,
	arg allocationArg,
	tenant pagination.TenantInfo,
	kind shipmentvocab.ChargeAllocationKind,
) (*shipment.ChargeAllocation, error) {
	path := fmt.Sprintf("%s[%d]", ParamAllocations, index)
	payer, err := pulid.Parse(strings.TrimSpace(arg.BillToCustomerID))
	if err != nil || payer.IsNil() {
		return nil, fmt.Errorf("%s.%s must be a customer id", path, ParamBillToCustomer)
	}
	method, err := toolparams.RequireEnum(
		map[string]any{ParamAllocMethod: arg.Method},
		ParamAllocMethod,
		allocationMethods.Values,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	row := &shipment.ChargeAllocation{
		OrganizationID:   tenant.OrgID,
		BusinessUnitID:   tenant.BuID,
		ChargeKind:       kind,
		BillToCustomerID: payer,
		Method:           method,
		Sequence:         int16(index), //nolint:gosec // bounded by maxChargeAllocations
	}
	share := map[string]any{ParamAllocPercent: arg.Percent, toolparams.ParamAmount: arg.Amount}
	percent, hasPercent, err := toolparams.OptionalDecimal(share, ParamAllocPercent)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	amount, hasAmount, err := toolparams.OptionalDecimal(share, toolparams.ParamAmount)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	switch method {
	case shipmentvocab.ChargeAllocationMethodPercent:
		if !hasPercent || hasAmount {
			return nil, fmt.Errorf("%s gives a percent and no amount when method is Percent",
				path)
		}
		row.Percent = decimal.NewNullDecimal(percent)
	case shipmentvocab.ChargeAllocationMethodAmount:
		if !hasAmount || hasPercent {
			return nil, fmt.Errorf("%s gives an amount and no percent when method is Amount",
				path)
		}
		row.Amount = decimal.NewNullDecimal(amount)
	}

	return row, nil
}
