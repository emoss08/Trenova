package reportaccess_test

import (
	"testing"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/reportengine"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var caseEntities = []string{"item", "resolution", "owner"}

func ownCasesOnly() permtest.Grant {
	return permtest.Grant{
		Operations:     []permission.Operation{permission.OpRead, permission.OpUpdate},
		DataScope:      permission.DataScopeOrganization,
		MaxSensitivity: permission.SensitivityConfidential,
	}
}

func everyCase() permtest.Grant {
	grant := ownCasesOnly()
	grant.Operations = append(grant.Operations, permission.OpManage, permission.OpExport)
	return grant
}

func TestRequiredOperation_WithoutSeeEveryCaseTheCaseDatasetsAreRefused(t *testing.T) {
	t.Parallel()

	f := newFixture(t, protectedfield.AllGroups())
	f.engine.Overrides[permission.ResourceWatchtower.String()] = ownCasesOnly()
	access := f.access(t, reportaccess.ModeRead)

	for _, key := range caseEntities {
		entity, ok := f.catalog.Entity(key)
		require.Truef(t, ok, "%s is not in the catalog", key)

		use := access.EntityUse(entity)
		assert.Equalf(t, reportaccess.NeedsOperation, use.Decision, "%s", key)
		assert.Equal(t, permission.ResourceWatchtower, use.Kind)
		assert.Equal(t, permission.OpManage, use.Operation)
		assert.Truef(t, access.LacksOperation(entity), "%s", key)

		for _, field := range entity.Fields {
			assert.Equalf(
				t,
				reportaccess.NeedsOperation,
				access.FieldUse(t.Context(), field).Decision,
				"%s.%s: a reference view does not open a case dataset",
				key,
				field.Key,
			)
		}
		for _, edge := range entity.Edges {
			assert.Equalf(
				t,
				reportaccess.NeedsOperation,
				access.EdgeUse(t.Context(), edge).Decision,
				"%s.%s",
				key,
				edge.Name,
			)
		}
	}

	assert.Equal(t, reportaccess.NeedsOperation,
		access.EdgeUse(t.Context(), f.edge(t, "item", "caseOwner")).Decision,
		"an edge into a case dataset needs the operation too")
}

func TestRequiredOperation_TheRefusalNamesThePermission(t *testing.T) {
	t.Parallel()

	f := newFixture(t, protectedfield.AllGroups())
	f.engine.Overrides[permission.ResourceWatchtower.String()] = ownCasesOnly()
	access := f.access(t, reportaccess.ModeRead)

	entity, _ := f.catalog.Entity("resolution")
	access.Entity(t.Context(), entity)

	err := access.Refusal(
		t.Context(),
		&analytics.Error{Code: analytics.CodeDenied, Message: "denied"},
	)
	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.Contains(
		t,
		err.Error(),
		"Watchtower Resolutions need the \"See every case\" permission on Watchtower",
	)
	assert.Equal(t, "watchtower manage", f.resolver.Need(reportaccess.Use{
		Decision:  reportaccess.NeedsOperation,
		Kind:      permission.ResourceWatchtower,
		Operation: permission.OpManage,
	}))
}

func TestRequiredOperation_SeeEveryCaseOpensTheCaseDatasets(t *testing.T) {
	t.Parallel()

	f := newFixture(t, protectedfield.AllGroups())
	f.engine.Overrides[permission.ResourceWatchtower.String()] = everyCase()
	access := f.access(t, reportaccess.ModeExport)

	for _, key := range caseEntities {
		entity, _ := f.catalog.Entity(key)
		assert.Equalf(t, reportaccess.Available, access.EntityUse(entity).Decision, "%s", key)
		assert.False(t, access.LacksOperation(entity))
	}
}

func TestPerson_ALinkedPersonShowsOnlyTheirNameWithoutUserRead(t *testing.T) {
	t.Parallel()

	f := newFixture(t, protectedfield.AllGroups())
	f.engine.Overrides[permission.ResourceWatchtower.String()] = everyCase()
	f.engine.Overrides[permission.ResourceUser.String()] = permtest.Grant{}
	access := f.access(t, reportaccess.ModeRead)

	assert.Equal(t, reportaccess.Available,
		access.EdgeUse(t.Context(), f.edge(t, "resolution", "resolvedBy")).Decision)

	name := access.FieldUse(t.Context(), f.field(t, "resolution", "resolvedBy.name"))
	assert.True(t, name.Reference, "a person's name is in the user reference view")
	assert.Equal(t, reportaccess.Available, name.Decision)

	title := access.FieldUse(t.Context(), f.field(t, "resolution", "resolvedBy.jobTitle"))
	assert.Equal(t, reportaccess.NeedsKindRead, title.Decision)

	person, ok := f.catalog.Entity("person")
	require.True(t, ok)
	assert.Equal(t, reportaccess.NeedsKindRead, access.EntityUse(person).Decision,
		"a roster of people needs user read")
	for _, field := range person.Fields {
		assert.NotContainsf(t,
			[]string{"emailAddress", "username", "password", "phoneNumber", "profilePicUrl"},
			field.Key, "person carries %s", field.Key)
	}
}

func TestUsage_TheUsageDatasetFollowsAIUsageNotAIProvider(t *testing.T) {
	t.Parallel()

	usageOnly := newFixture(t, protectedfield.AllGroups())
	usageOnly.engine.Overrides[permission.ResourceAIProvider.String()] = permtest.Grant{}
	usageOnly.engine.Overrides[permission.ResourceAIUsage.String()] = permtest.Grant{
		Operations:     []permission.Operation{permission.OpRead, permission.OpExport},
		DataScope:      permission.DataScopeOrganization,
		MaxSensitivity: permission.SensitivityRestricted,
	}
	usage, ok := usageOnly.catalog.Entity("daily_usage")
	require.True(t, ok)
	assert.Equal(t, reportaccess.Available,
		usageOnly.access(t, reportaccess.ModeExport).EntityUse(usage).Decision)

	providerOnly := newFixture(t, protectedfield.AllGroups())
	providerOnly.engine.Overrides[permission.ResourceAIUsage.String()] = permtest.Grant{}
	providerOnly.engine.Overrides[permission.ResourceAIProvider.String()] = permtest.Grant{
		Operations:     []permission.Operation{permission.OpRead},
		DataScope:      permission.DataScopeOrganization,
		MaxSensitivity: permission.SensitivityConfidential,
	}
	use := providerOnly.access(t, reportaccess.ModeRead).EntityUse(usage)
	assert.Equal(t, reportaccess.NeedsKindRead, use.Decision)
	assert.Equal(t, permission.ResourceAIUsage, use.Kind)
}

func TestMissingOperation_FindsTheCannedReportsThatNeedSeeEveryCase(t *testing.T) {
	t.Parallel()

	f := newFixture(t, protectedfield.AllGroups())
	f.engine.Overrides[permission.ResourceWatchtower.String()] = ownCasesOnly()
	access := f.access(t, reportaccess.ModeRead)

	registry, err := canned.Default()
	require.NoError(t, err)

	needsEveryCase := map[string]bool{
		"cases-handled-by-user":             true,
		"watchtower-open-items-by-user":     true,
		"watchtower-backlog-by-kind":        true,
		"watchtower-resolution-reasons":     true,
		"watchtower-response-times-by-week": true,
		"open-case-load-by-owner":           true,
	}
	for _, entry := range registry.All() {
		reach, reachErr := f.resolver.ReachOf(entry.Definition)
		require.NoError(t, reachErr, entry.Key)
		use, missing := access.MissingOperation(reach)
		assert.Equalf(t, needsEveryCase[entry.Key], missing, "%s", entry.Key)
		if missing {
			assert.Equal(t, permission.OpManage, use.Operation)
		}
	}
}

func templateEngine(
	t *testing.T,
	registry *permissioncatalog.Registry,
	id string,
) *permtest.Engine {
	t.Helper()
	for _, template := range permissioncatalog.JobTemplates() {
		if template.ID != id {
			continue
		}
		engine := &permtest.Engine{Overrides: make(map[string]permtest.Grant)}
		for _, grant := range registry.TemplateGrants(&template) {
			engine.Overrides[grant.Resource] = permtest.Grant{
				Operations:     permission.Implied(grant.Operations),
				DataScope:      grant.DataScope,
				MaxSensitivity: template.MaxSensitivity,
			}
		}
		return engine
	}
	require.FailNowf(t, "no job template", "%s", id)
	return nil
}

func compileDeskReports(
	t *testing.T,
	templateID string,
	mode reportaccess.Mode,
) map[string]error {
	t.Helper()
	catalog, err := sharedCatalog()
	require.NoError(t, err)
	registry := permissioncatalog.NewRegistry()
	engine := templateEngine(t, registry, templateID)
	resolver, err := reportaccess.New(reportaccess.Params{
		Catalog:  catalog,
		Engine:   engine,
		Registry: registry,
		Views:    refview.NewPort(plantest.Views(registry), engine),
		Fields:   groupAccess{visible: protectedfield.AllGroups()},
	})
	require.NoError(t, err)
	tenant := pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
	library, err := canned.Default()
	require.NoError(t, err)
	limits := reportengine.DefinitionLimits(&reportingconfig.Config{})

	results := make(map[string]error)
	for _, entry := range library.All() {
		if entry.Category != "Desk" {
			continue
		}
		access, accessErr := resolver.For(t.Context(), tenant, mode)
		require.NoError(t, accessErr)
		ctx := dbscope.WithTenant(t.Context(), tenant.DBTenant())
		_, results[entry.Key] = analytics.Compile(
			ctx, catalog, access, entry.Definition, nil,
			analytics.CompileOptions{Limits: limits},
		)
	}
	require.Len(t, results, 9, "the nine Desk reports")
	return results
}

func TestManagerTemplatesRunEveryDeskReportInExportMode(t *testing.T) {
	t.Parallel()

	for _, templateID := range []string{"dispatch_supervisor", "billing_manager"} {
		for _, mode := range []reportaccess.Mode{reportaccess.ModeRead, reportaccess.ModeExport} {
			for key, err := range compileDeskReports(t, templateID, mode) {
				assert.NoErrorf(t, err, "%s cannot open %s in mode %d", templateID, key, mode)
			}
		}
	}
}

func TestTemplatesBelowManagerAreRefusedTheDeskReports(t *testing.T) {
	t.Parallel()

	for _, templateID := range []string{"dispatcher", "billing_clerk"} {
		for key, err := range compileDeskReports(t, templateID, reportaccess.ModeRead) {
			assert.Equalf(t, analytics.CodeDenied, analytics.ErrorCode(err),
				"%s opens %s", templateID, key)
		}
	}
}
