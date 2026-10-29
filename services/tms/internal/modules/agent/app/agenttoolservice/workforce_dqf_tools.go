package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/qualificationtools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
)

var _ qualificationtools.VerificationKeeper = (*workerdqfservice.Service)(nil)
