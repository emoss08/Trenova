package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/ratetools"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
)

var _ ratetools.RateAgreementKeeper = (*rateagreementservice.Service)(nil)

var _ agentports.ToolSelectionResolver = ratetools.RateIncreaseTool{}
