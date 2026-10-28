package permissioncatalog

import (
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundles_GrantOnlyWhatTheRegistryDeclares(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	seen := make(map[string]struct{}, len(capabilityBundles))
	for _, bundle := range Bundles() {
		_, dup := seen[bundle.ID]
		assert.False(t, dup, "bundle %q is declared twice", bundle.ID)
		seen[bundle.ID] = struct{}{}

		assert.NotEmpty(t, bundle.Label, bundle.ID)
		assert.NotEmpty(t, bundle.Description, bundle.ID)
		require.NotEmpty(t, bundle.Grants, bundle.ID)
		for _, grant := range bundle.Grants {
			for _, op := range grant.Operations {
				assert.True(t, reg.Declares(grant.Resource.String(), op),
					"bundle %q grants %s:%s, which the registry does not declare",
					bundle.ID, grant.Resource, op)
			}
			if grant.Scope != "" {
				assert.True(t, reg.ScopeDeclared(grant.Resource.String(), grant.Scope),
					"bundle %q limits %s to %s, which that kind does not declare",
					bundle.ID, grant.Resource, grant.Scope)
			}
		}
	}
}

func TestJobTemplates_NameBundlesThatExist(t *testing.T) {
	t.Parallel()

	for _, template := range JobTemplates() {
		assert.True(t, template.ReadEverything || len(template.Bundles) > 0, template.ID)
		assert.True(t, template.MaxSensitivity.IsValid(), template.ID)
		for _, id := range template.Bundles {
			_, ok := BundleByID(id)
			assert.True(t, ok, "template %q names unknown bundle %q", template.ID, id)
		}
	}
}

func TestTemplateGrants_MergeBundlesAndKeepTheWidestScope(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	grants := reg.TemplateGrants(JobTemplate{Bundles: []string{"hr.team", "hr.workers"}})

	idx := slices.IndexFunc(grants, func(rp *permission.ResourcePermission) bool {
		return rp.Resource == permission.ResourceWorker.String()
	})
	require.GreaterOrEqual(t, idx, 0)
	worker := grants[idx]
	assert.Equal(t, permission.DataScopeOrganization, worker.DataScope,
		"hr.workers grants workers organization-wide, which outweighs the team grant")
	assert.Equal(
		t,
		[]permission.Operation{
			permission.OpArchive,
			permission.OpCreate,
			permission.OpRead,
			permission.OpRestore,
			permission.OpUpdate,
		},
		worker.Operations,
	)

	teamOnly := reg.TemplateGrants(JobTemplate{Bundles: []string{"hr.team"}})
	idx = slices.IndexFunc(teamOnly, func(rp *permission.ResourcePermission) bool {
		return rp.Resource == permission.ResourceWorker.String()
	})
	require.GreaterOrEqual(t, idx, 0)
	assert.Equal(t, permission.DataScopeTeam, teamOnly[idx].DataScope)
}

func leaveOperations(grants []*permission.ResourcePermission) []permission.Operation {
	idx := slices.IndexFunc(grants, func(rp *permission.ResourcePermission) bool {
		return rp.Resource == permission.ResourceWorkerLeave.String()
	})
	if idx < 0 {
		return nil
	}
	return permission.Implied(grants[idx].Operations)
}

func TestJobTemplates_OnlyHumanResourcesReachLeave(t *testing.T) {
	t.Parallel()

	expected := map[string][]permission.Operation{
		"dispatcher":          nil,
		"dispatch_supervisor": nil,
		"carrier_rep":         nil,
		"billing_clerk":       nil,
		"billing_manager":     nil,
		"accountant":          nil,
		"payroll":             nil,
		"safety":              nil,
		"driver_manager":      {permission.OpApprove, permission.OpRead, permission.OpUpdate},
		"hr": {
			permission.OpApprove,
			permission.OpCreate,
			permission.OpRead,
			permission.OpUpdate,
		},
		"read_only": nil,
	}

	reg := NewRegistry()
	require.Equal(t, permission.ResourceWorkerLeave.String(),
		reg.GetEffectiveResource(permission.ResourceWorkerLeave.String()),
		"a worker grant must not reach leave, or this table says nothing")
	templates := JobTemplates()
	require.Len(t, templates, len(expected), "add every new template to this table")
	for _, template := range templates {
		want, listed := expected[template.ID]
		require.True(t, listed, "template %q is missing from this table", template.ID)
		assert.Equal(t, want, leaveOperations(reg.TemplateGrants(template)),
			"template %q", template.ID)
	}
}

func TestBundles_OnlyHumanResourcesBundlesGrantLeave(t *testing.T) {
	t.Parallel()

	expected := map[string][]permission.Operation{
		"hr.leave": {
			permission.OpApprove,
			permission.OpCreate,
			permission.OpRead,
			permission.OpUpdate,
		},
		"hr.team": {permission.OpApprove, permission.OpRead},
	}

	for _, bundle := range Bundles() {
		var got []permission.Operation
		for _, grant := range bundle.Grants {
			if grant.Resource == permission.ResourceWorkerLeave {
				got = permission.Normalize(append(got, grant.Operations...))
			}
		}
		assert.Equal(t, expected[bundle.ID], got, "bundle %q", bundle.ID)
	}
}

func TestTemplateGrants_ReadEverythingSkipsWhatTheTemplateExcepts(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	grants := reg.TemplateGrants(JobTemplate{
		ReadEverything: true,
		ReadExcept:     []permission.Resource{permission.ResourceWorkerLeave},
	})

	assert.Nil(t, leaveOperations(grants))
	assert.Len(t, grants, len(reg.TemplateGrants(JobTemplate{ReadEverything: true}))-1)
}

func TestTemplateGrants_ReadEverythingReadsEveryReadableKind(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	grants := reg.TemplateGrants(JobTemplate{ReadEverything: true})

	readable := 0
	for _, def := range reg.All() {
		if slices.Contains(reg.GetOperationsForResource(def.Resource), permission.OpRead) {
			readable++
		}
	}
	assert.Len(t, grants, readable)
	for _, rp := range grants {
		assert.Equal(t, []permission.Operation{permission.OpRead}, rp.Operations, rp.Resource)
	}
}
