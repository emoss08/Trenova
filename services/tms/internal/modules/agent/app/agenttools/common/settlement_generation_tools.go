package common

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
)

const ParamWorkerID = "workerId"

const SecondsPerDay = int64(24 * time.Hour / time.Second)

const DriverSettlementEntity = "driver_settlement"

const FieldPeriodStart = "periodStart"

const FieldPeriodEnd = "periodEnd"

const FieldPayDate = "payDate"

var SettlementDateTypes = map[string]assistantartifact.DisplayType{
	FieldPeriodStart: assistantartifact.DisplayDate,
	FieldPeriodEnd:   assistantartifact.DisplayDate,
	FieldPayDate:     assistantartifact.DisplayDate,
}

func SettlementResult(created *driversettlement.Settlement) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: ResultCreated,
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
			ParamWorkerID, FieldStatus, FieldPeriodStart, FieldPeriodEnd, FieldPayDate,
			"payProfileName", "classification", "shipmentCount", FieldHasExceptions,
		),
		toolpreview.WithRefs(map[string]permission.Resource{
			ParamWorkerID: permission.ResourceWorker,
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

func DayText(seconds int64) string {
	return time.Unix(seconds, 0).UTC().Format("Jan 2, 2006")
}
