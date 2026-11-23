package permissioncatalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func templateAllows(
	grants []*rbacGrant,
	resource permission.Resource,
	op permission.Operation,
) bool {
	for _, grant := range grants {
		if grant.resource == resource.String() {
			return grant.ops.Has(op)
		}
	}
	return false
}

type rbacGrant struct {
	resource string
	ops      permission.OperationSet
}

func TestJobTemplates_ManagementReportsReachOnlyManagers(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	managers := map[string]bool{"dispatch_supervisor": true, "billing_manager": true}
	for _, template := range JobTemplates() {
		if template.ReadEverything {
			continue
		}
		raw := reg.TemplateGrants(&template)
		grants := make([]*rbacGrant, 0, len(raw))
		for _, grant := range raw {
			grants = append(grants, &rbacGrant{
				resource: grant.Resource,
				ops:      permission.NewOperationSet(permission.Implied(grant.Operations)...),
			})
		}

		want := managers[template.ID]
		for _, need := range []struct {
			resource permission.Resource
			op       permission.Operation
			why      string
		}{
			{permission.ResourceWatchtower, permission.OpManage, "See every case"},
			{permission.ResourceWatchtower, permission.OpExport, "running the case reports"},
			{permission.ResourceAIUsage, permission.OpRead, "AI usage and spend"},
			{permission.ResourceAIUsage, permission.OpExport, "running the usage reports"},
		} {
			assert.Equalf(t, want, templateAllows(grants, need.resource, need.op),
				"%s: %s", template.ID, need.why)
		}
		if want {
			assert.Truef(t, templateAllows(grants, permission.ResourceReport, permission.OpExport),
				"%s: runs and schedules reports", template.ID)
			for _, op := range []permission.Operation{
				permission.OpRead, permission.OpUpdate, permission.OpManage,
			} {
				assert.Falsef(t, templateAllows(grants, permission.ResourceAIProvider, op),
					"%s: seeing spend never opens provider settings", template.ID)
			}
		}
	}
}

func TestAIUsageBundle_GrantsUsageAndNothingOnProviders(t *testing.T) {
	t.Parallel()

	bundle, ok := BundleByID("reporting.ai-usage")
	if !assert.True(t, ok) {
		return
	}
	for _, grant := range bundle.Grants {
		assert.NotEqual(t, permission.ResourceAIProvider, grant.Resource)
	}
	assert.Len(t, bundle.Grants, 1)
	assert.Equal(t, permission.ResourceAIUsage, bundle.Grants[0].Resource)
	assert.ElementsMatch(t,
		[]permission.Operation{permission.OpRead, permission.OpExport},
		bundle.Grants[0].Operations)

	reg := NewRegistry()
	feature, _ := reg.FeatureOf(permission.ResourceAIUsage.String())
	providerFeature, _ := reg.FeatureOf(permission.ResourceAIProvider.String())
	assert.Equal(t, providerFeature, feature, "the plan treats usage as it treats providers")
	assert.ElementsMatch(t,
		[]permission.Operation{permission.OpRead, permission.OpExport},
		reg.GetOperationsForResource(permission.ResourceAIUsage.String()))
}
