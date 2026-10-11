package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/workforcetools"
)

var _ workforcetools.CredentialKeeper = (*workercredentialservice.Service)(nil)
