package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.DrugAlcoholKeeper = (*workerdrugalcoholservice.Service)(nil)
