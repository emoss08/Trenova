package extensionportalservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensiontest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type fixture struct {
	svc       *Service
	core      *agentextensionservice.Service
	repo      *agentextensiontest.MemoryRepo
	usage     *agentextensiontest.MemoryUsage
	browser   *fakeBrowser
	sites     *memorySites
	runs      *memoryRuns
	audit     *agentextensiontest.RecordingAudit
	workflows *recordingWorkflows
	tenant    pagination.TenantInfo
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		repo:    agentextensiontest.NewMemoryRepo(),
		usage:   &agentextensiontest.MemoryUsage{Cost: decimal.Zero},
		browser: &fakeBrowser{},
		sites:   newMemorySites(),
		runs:    newMemoryRuns(),
		audit:   &agentextensiontest.RecordingAudit{},
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
	encryption := agentextensiontest.EncryptionService()
	connectors := []agentextensionports.ExtensionConnector{
		&agentextensiontest.FakeWeb{Kind: agentextension.TypeExa},
		&agentextensiontest.FakeWeb{Kind: agentextension.TypeParallel},
		f.browser,
	}
	f.core = agentextensionservice.New(agentextensionservice.Params{
		Logger:       zap.NewNop(),
		Repo:         f.repo,
		Usage:        f.usage,
		Encryption:   encryption,
		AuditService: f.audit,
		Connectors:   connectors,
	})
	f.svc = New(Params{
		Logger:       zap.NewNop(),
		Extensions:   f.core,
		Encryption:   encryption,
		AuditService: f.audit,
		Connectors:   connectors,
		Sites:        f.sites,
		Runs:         f.runs,
	})
	f.svc.now = func() int64 { return agentextensiontest.Now }

	return f
}

func (f *fixture) enableType(
	t *testing.T,
	typ agentextension.Type,
	configuration map[string]string,
) *agentextensionports.AgentExtensionConfigResponse {
	t.Helper()

	return agentextensiontest.EnableType(t, f.core, f.tenant, typ, configuration)
}
