package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.VerificationKeeper = (*workerdqfservice.Service)(nil)
