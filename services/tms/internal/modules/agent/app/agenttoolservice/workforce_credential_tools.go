package agenttoolservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/qualificationtools"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
)

var _ qualificationtools.CredentialKeeper = (*workercredentialservice.Service)(nil)
