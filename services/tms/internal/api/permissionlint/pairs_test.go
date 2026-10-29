package permissionlint_test

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/api/permissionlint"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/stretchr/testify/require"
)

const (
	permissionDir = "../../modules/identity/domain/permission"
	handlersDir   = "../handlers"
	resolverDir   = "../graphql/resolver"
	middlewareDir = "../middleware"

	clientResourcesFile = "../../../../../client/packages/shared/src/types/generated/permission-resources.ts"
)

var clientDirs = []string{
	"../../../../../client/apps/web/src",
	"../../../../../client/apps/dash/src",
	"../../../../../client/packages/shared/src",
}

func TestEveryCheckedPermissionIsDeclared(t *testing.T) {
	t.Parallel()

	consts, err := permissionlint.LoadConstants(permissionDir)
	require.NoError(t, err)
	require.NotEmpty(t, consts.Resources)
	require.NotEmpty(t, consts.Operations)

	result, err := permissionlint.Analyze(consts, handlersDir, resolverDir, middlewareDir)
	require.NoError(t, err)
	pairs := result.Pairs
	require.Greater(t, len(pairs), 500, "the analyzer stopped finding permission checks")

	registry := permissioncatalog.NewRegistry()
	var undeclared []string
	for _, pair := range pairs {
		if !registry.Declares(pair.Resource, permission.Operation(pair.Operation)) {
			undeclared = append(undeclared, pair.String())
		}
	}

	require.Empty(
		t,
		undeclared,
		"these checks name an operation the permission registry does not declare for the "+
			"resource, so no role can ever be granted it and the request is always refused; "+
			"declare the operation in permission/resources_*.go or check one that is declared",
	)
}

func TestAnalyzerFollowsBoundWrappers(t *testing.T) {
	t.Parallel()

	consts, err := permissionlint.LoadConstants(permissionDir)
	require.NoError(t, err)

	result, err := permissionlint.Analyze(consts, handlersDir+"/rolehandler")
	require.NoError(t, err)

	found := make([]string, 0, len(result.Pairs))
	for _, pair := range result.Pairs {
		found = append(found, pair.Resource+":"+pair.Operation)
	}

	require.Contains(t, found, "role:read")
	require.Contains(t, found, "user:unassign")
	require.Contains(t, found, "user:assign")
	require.NotContains(t, found, "role:unassign")
	require.NotContains(t, found, "role:assign")
}

var dynamicChecks = map[string]string{
	"graphql/resolver/aicontrolresolver/tuneups.go:requireApplyPermission":                             "the resource the tune-up changes, from its kind",
	"graphql/resolver/attentionresolver/attentionhelpers.go:attentionSectionRunner":                    "each attention section passes its own resource",
	"graphql/resolver/resolvercore/agentproposalchoices.go:subsetLabels":                               "the resource a proposal field reads",
	"graphql/resolver/bulkeditresolver/bulk_edit.resolvers.go:StartBulkEdit":                           "the resource the bulk editor registry resolves",
	"graphql/resolver/bulkeditresolver/bulk_edit.resolvers.go:UndoMyBulkEdit":                          "the resource the bulk editor registry resolves",
	"graphql/resolver/bulkeditresolver/bulk_edit.resolvers.go:BulkEditFields":                          "the resource the bulk editor registry resolves",
	"graphql/resolver/bulkeditresolver/bulk_edit.resolvers.go:PreviewBulkEdit":                         "the resource the bulk editor registry resolves",
	"graphql/resolver/captureresolver/capturehelpers.go:CaptureRecord":                                 "the kind of record a capture was filed against",
	"graphql/resolver/datatableinsightresolver/data_table_insight.resolvers.go:DataTableFacets":        "the table the insight service resolves",
	"graphql/resolver/datatableinsightresolver/data_table_insight.resolvers.go:DataTableAggregates":    "the table the insight service resolves",
	"graphql/resolver/datatableinsightresolver/data_table_insight.resolvers.go:DataTableInsightFields": "the table the insight service resolves",
	"graphql/resolver/datatableinsightresolver/data_table_insight.resolvers.go:DataTableSeries":        "the table the insight service resolves",
	"graphql/resolver/reportresolver/report.resolvers.go:ExportTableView":                              "the table being exported; an unknown resource has no grant",
	"graphql/resolver/tableconfigurationresolver/access.go:mayReadTable":                               "the table a saved view belongs to; an unknown resource has no grant",
	"graphql/resolver/reportresolver/report.resolvers.go:ScheduleTableView":                            "the table being scheduled; an unknown resource has no grant",
	"handlers/assistanthandler/artifacts.go:registerArtifactRoutes":                                    "the assistant resource, passed by RegisterRoutes",
	"handlers/assistanthandler/cases.go:registerCaseRoutes":                                            "the assistant resource, passed by RegisterRoutes",
	"handlers/assistanthandler/chapters.go:registerChapterRoutes":                                      "the assistant resource, passed by RegisterRoutes",
	"handlers/assistanthandler/queue.go:registerQueueRoutes":                                           "the assistant resource, passed by RegisterRoutes",
	"handlers/assistanthandler/schedules.go:registerScheduleRoutes":                                    "the assistant resource, passed by RegisterRoutes",
	"handlers/assistanthandler/waits.go:registerWaitRoutes":                                            "the assistant resource, passed by RegisterRoutes",
	"middleware/permission.go:RequireParamPermission":                                                  "the resource named in the path, validated against the registry",
}

func TestEveryPermissionCheckNamesItsPermission(t *testing.T) {
	t.Parallel()

	consts, err := permissionlint.LoadConstants(permissionDir)
	require.NoError(t, err)

	result, err := permissionlint.Analyze(consts, handlersDir, resolverDir, middlewareDir)
	require.NoError(t, err)

	var unexplained []string
	for _, site := range result.Unresolved {
		if _, ok := dynamicChecks[site]; !ok {
			unexplained = append(unexplained, site)
		}
	}
	require.Empty(
		t,
		unexplained,
		"these checks name a resource or operation the analyzer cannot read, so nothing "+
			"proves the registry declares it; name the permission constants at the call, or "+
			"add the site to dynamicChecks with the reason the resource is chosen at run time",
	)

	var stale []string
	for site := range dynamicChecks {
		if !slices.Contains(result.Unresolved, site) {
			stale = append(stale, site)
		}
	}
	require.Empty(t, stale, "these dynamicChecks entries no longer match a check; remove them")
}

func TestAnalyzerReadsLocalResources(t *testing.T) {
	t.Parallel()

	consts, err := permissionlint.LoadConstants(permissionDir)
	require.NoError(t, err)

	result, err := permissionlint.Analyze(consts, handlersDir+"/iamhandler")
	require.NoError(t, err)
	require.Empty(t, result.Unresolved)

	found := make([]string, 0, len(result.Pairs))
	for _, pair := range result.Pairs {
		found = append(found, pair.Resource+":"+pair.Operation)
	}

	require.Contains(t, found, "identity_provider:delete")
	require.Contains(t, found, "scim_directory:update")
	require.Contains(t, found, "auth_event:read")
	require.NotContains(t, found, "organization:update")
}

func TestEveryClientPermissionGateIsDeclared(t *testing.T) {
	t.Parallel()

	resources, err := permissionlint.LoadClientResources(clientResourcesFile)
	require.NoError(t, err)
	require.NotEmpty(t, resources)

	result, err := permissionlint.AnalyzeClient(resources, clientDirs...)
	require.NoError(t, err)
	require.Greater(t, len(result.Pairs), 50, "the client analyzer stopped finding gates")
	require.Empty(t, result.Unresolved, "these gates name a resource the client does not know")

	registry := permissioncatalog.NewRegistry()
	var undeclared []string
	for _, pair := range result.Pairs {
		if !registry.Declares(pair.Resource, permission.Operation(pair.Operation)) {
			undeclared = append(undeclared, pair.String())
		}
	}

	require.Empty(
		t,
		undeclared,
		"these client gates wait on an operation the registry does not declare for the "+
			"resource, so no role can open them",
	)
}
