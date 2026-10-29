package reference

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/accountingseed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/referenceseed"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
)

const tenantTimeout = 2 * time.Minute

type tenantApply func(ctx context.Context, db *keel.DB, scope tenantscope.Scope) (int, error)

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
			n, err := s.apply(ctx, r.DB, tenantscope.Scope{
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
	apply:       accountingseed.EnsureChartOfAccounts,
})

var DocumentTypes = perTenant(tenantSeed{
	name:        "system_document_types",
	description: "The system document types every tenant needs, added where missing",
	fingerprint: fingerprintOf(func() any { return referenceseed.DocumentTypeCodes() }),
	apply:       referenceseed.CreateDocumentTypes,
})

var ServiceFailureReasonCodes = perTenant(tenantSeed{
	name:        "service_failure_reason_codes",
	description: "The default service failure reason codes, added where missing",
	fingerprint: fingerprintOf(func() any { return referenceseed.ServiceFailureReasonCodes() }),
	apply:       referenceseed.CreateServiceFailureReasonCodes,
})

var DocumentTemplateStarters = perTenant(tenantSeed{
	name:        "document_template_starters",
	description: "An editable draft of every built-in document and message template kind",
	fingerprint: fingerprintOf(func() any { return referenceseed.TemplateStarterKinds() }),
	apply:       referenceseed.CreateDocumentTemplateStarters,
})

var SystemAgentDefinitions = perTenant(tenantSeed{
	name:        "system_agent_definitions",
	description: "The disabled, shadowed system agents every tenant can switch on",
	fingerprint: fingerprintOf(func() any { return referenceseed.SystemAgentKeys() }),
	apply:       referenceseed.CreateSystemAgentDefinitions,
})

var TableChangeAlertAllowlist = perTenant(tenantSeed{
	name:        "table_change_alert_allowlist",
	description: "The tables eligible for table change alerts, for a tenant that has none",
	version:     "1",
	apply:       referenceseed.CreateTCAAllowlist,
})

var AdministratorPermissions = perTenant(tenantSeed{
	name:        "administrator_permissions",
	description: "Grants every registered resource and operation to the administrator role",
	fingerprint: func() string { return permissioncatalog.NewRegistry().GrantFingerprint() },
	apply:       reconcileAdministratorPermissions,
})

func reconcileAdministratorPermissions(
	ctx context.Context,
	db *keel.DB,
	scope tenantscope.Scope,
) (int, error) {
	result, err := tenantbootstrap.ReconcileAdminRole(
		ctx,
		db,
		scope,
		permissioncatalog.NewRegistry(),
	)
	return result.GrantsAdded + result.GrantsExtended, err
}
