package aiauditprojector

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditredact"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiaudittest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
)

func pruneThrough(s *scenario, seq int64) aiauditports.PruneAIAuditEventsRequest {
	return aiauditports.PruneAIAuditEventsRequest{TenantInfo: s.tenant, ThroughSeq: seq}
}

func testRedactor() *aiauditredact.Redactor {
	return aiauditredact.New(
		permissioncatalog.NewRegistry(),
		aiaudittest.TestPolicies(),
		auditservice.NewSensitiveDataManager(&securityconfig.EncryptionConfig{}),
	)
}
