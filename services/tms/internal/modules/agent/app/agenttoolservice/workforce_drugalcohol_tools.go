package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/drugalcoholtools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdrugalcoholservice"
)

var _ drugalcoholtools.DrugAlcoholKeeper = (*workerdrugalcoholservice.Service)(nil)
