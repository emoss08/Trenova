package reference

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
)

const tenantTimeout = 2 * time.Minute

type tenantApply func(ctx context.Context, db *keel.DB, scope tenantbootstrap.Scope) (int, error)

type tenantSeed struct {
	name        string
	description string
	version     string
	fingerprint func() string
	apply       tenantApply
}

func perTenant(s tenantSeed) *seed.Seed {
	return Seeds.Add(seed.Seed{
		Name:        s.name,
		Kind:        seed.Reference,
		Description: s.description,
		PerTenant:   true,
		Version:     s.version,
		Fingerprint: s.fingerprint,
		Timeout:     tenantTimeout,
		Run: func(ctx context.Context, r *seed.Run) error {
			tenant := seedenv.TenantOf(ctx)
			n, err := s.apply(ctx, r.DB, tenantbootstrap.Scope{
				OrganizationID: tenant.OrganizationID,
				BusinessUnitID: tenant.BusinessUnitID,
			})
			r.Count(int64(n))
			return err
		},
	})
}

var ChartOfAccounts = perTenant(tenantSeed{
	name:        "chart_of_accounts",
	description: "The default trucking chart of accounts for a tenant that has none",
	version:     "1",
	apply:       tenantbootstrap.EnsureChartOfAccounts,
})

var DocumentTypes = perTenant(tenantSeed{
	name:        "system_document_types",
	description: "The system document types every tenant needs, added where missing",
	fingerprint: fingerprintOf(func() any { return tenantbootstrap.DocumentTypeCodes() }),
	apply:       tenantbootstrap.CreateDocumentTypes,
})

var ServiceFailureReasonCodes = perTenant(tenantSeed{
	name:        "service_failure_reason_codes",
	description: "The default service failure reason codes, added where missing",
	fingerprint: fingerprintOf(func() any { return tenantbootstrap.ServiceFailureReasonCodes() }),
	apply:       tenantbootstrap.CreateServiceFailureReasonCodes,
})

var DocumentTemplateStarters = perTenant(tenantSeed{
	name:        "document_template_starters",
	description: "An editable draft of every built-in document and message template kind",
	fingerprint: fingerprintOf(func() any { return tenantbootstrap.TemplateStarterKinds() }),
	apply:       tenantbootstrap.CreateDocumentTemplateStarters,
})

var SystemAgentDefinitions = perTenant(tenantSeed{
	name:        "system_agent_definitions",
	description: "The disabled, shadowed system agents every tenant can switch on",
	fingerprint: fingerprintOf(func() any { return tenantbootstrap.SystemAgentKeys() }),
	apply:       tenantbootstrap.CreateSystemAgentDefinitions,
})

var TableChangeAlertAllowlist = perTenant(tenantSeed{
	name:        "table_change_alert_allowlist",
	description: "The tables eligible for table change alerts, for a tenant that has none",
	version:     "1",
	apply:       tenantbootstrap.CreateTCAAllowlist,
})

var AdministratorPermissions = perTenant(tenantSeed{
	name:        "administrator_permissions",
	description: "Grants every registered resource and operation to the administrator role",
	fingerprint: func() string { return permission.NewRegistry().GrantFingerprint() },
	apply:       reconcileAdministratorPermissions,
})

func reconcileAdministratorPermissions(
	ctx context.Context,
	db *keel.DB,
	scope tenantbootstrap.Scope,
) (int, error) {
	result, err := tenantbootstrap.ReconcileAdminRole(ctx, db, scope, permission.NewRegistry())
	return result.GrantsAdded + result.GrantsExtended, err
}
