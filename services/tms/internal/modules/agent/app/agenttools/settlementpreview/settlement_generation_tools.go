package settlementpreview

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
)

const DriverSettlementEntity = "driver_settlement"

const FieldPayDate = "payDate"

var SettlementDateTypes = map[string]assistantartifact.DisplayType{
	changepreview.FieldPeriodStart: assistantartifact.DisplayDate,
	changepreview.FieldPeriodEnd:   assistantartifact.DisplayDate,
	FieldPayDate:                   assistantartifact.DisplayDate,
}

func SettlementResult(created *driversettlement.Settlement) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: changepreview.ResultCreated,
		Kind:   NounDriverSettlement,
		Name:   created.SettlementNumber,
		IDs:    map[string]string{ParamSettlementID: created.ID.String()},
		Record: &agent.RecordRef{EntityType: DriverSettlementEntity, ID: created.ID.String()},
	}
}

func NewSettlementChange(
	settlement *driversettlement.Settlement,
	label string,
) (*agent.RecordChange, error) {
	change, err := toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceDriverSettlement, Label: label},
		settlement,
		toolpreview.Only(
			toolparams.ParamWorkerID,
			changepreview.FieldStatus,
			changepreview.FieldPeriodStart,
			changepreview.FieldPeriodEnd,
			FieldPayDate,
			"payProfileName",
			"classification",
			"shipmentCount",
			FieldHasExceptions,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			toolparams.ParamWorkerID: permission.ResourceWorker,
		}),
		toolpreview.Types(SettlementDateTypes),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		SettlementMoney(&SettlementFacts{}, DriverSettlementFacts(settlement)),
		toolpreview.SensitiveAs(DriverSettlementLedger().Sensitive...),
	)

	return change, nil
}
