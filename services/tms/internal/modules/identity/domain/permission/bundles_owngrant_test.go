package permission

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func effectiveGrant(reg *Registry, grants []*ResourcePermission, kind Resource) string {
	idx := slices.IndexFunc(grants, func(rp *ResourcePermission) bool {
		return rp.Resource == kind.String()
	})
	if idx < 0 {
		return ""
	}
	ops := make([]string, 0, 8)
	for _, op := range Implied(grants[idx].Operations) {
		if reg.Declares(kind.String(), op) {
			ops = append(ops, string(op))
		}
	}
	return strings.Join(ops, ",") + "@" + string(grants[idx].DataScope)
}

func TestJobTemplates_KindsGrantedOnTheirOwn(t *testing.T) {
	t.Parallel()

	kinds := []Resource{
		ResourceWorkerInjury,
		ResourceWorkerDOTTest,
		ResourceBenefitPlan,
		ResourceWorkerEmploymentEvent,
		ResourceWorkerDisciplinaryAction,
		ResourcePerformanceReview,
		ResourceTimesheet,
		ResourceProfileChangeRequest,
		ResourceDriverPortal,
		ResourceWorkerLeave,
	}
	none := map[Resource]string{}
	expected := map[string]map[Resource]string{
		"dispatcher":          none,
		"dispatch_supervisor": none,
		"carrier_rep":         none,
		"billing_clerk":       none,
		"billing_manager":     none,
		"accountant":          none,
		"payroll":             none,
		"safety": {
			ResourceWorkerInjury:  "create,read,update@organization",
			ResourceWorkerDOTTest: "create,read,update@organization",
		},
		"driver_manager": {
			ResourceTimesheet:            "approve,read,reject,update@team",
			ResourceProfileChangeRequest: "approve,read,reject@team",
			ResourceDriverPortal:         "cancel,create,read@team",
			ResourceWorkerLeave:          "approve,read,update@team",
		},
		"hr": {
			ResourceBenefitPlan:              "assign,create,read,update@organization",
			ResourceWorkerEmploymentEvent:    "create,read,update@organization",
			ResourceWorkerDisciplinaryAction: "create,read@organization",
			ResourcePerformanceReview:        "create,read,update@organization",
			ResourceTimesheet:                "approve,read,reject,update@team",
			ResourceProfileChangeRequest:     "approve,read,reject@team",
			ResourceDriverPortal:             "cancel,create,read@organization",
			ResourceWorkerLeave:              "approve,create,read,update@organization",
		},
		"read_only": {
			ResourceDriverPortal: "read@organization",
		},
	}

	reg := NewRegistry()
	for _, kind := range kinds {
		require.Equal(t, kind.String(), reg.GetEffectiveResource(kind.String()),
			"a worker grant must not reach %s, or this table says nothing", kind)
	}
	templates := JobTemplates()
	require.Len(t, templates, len(expected), "add every new template to this table")
	for _, template := range templates {
		want, listed := expected[template.ID]
		require.True(t, listed, "template %q is missing from this table", template.ID)
		grants := reg.TemplateGrants(template)
		for _, kind := range kinds {
			assert.Equal(t, want[kind], effectiveGrant(reg, grants, kind),
				"template %q on %s", template.ID, kind)
		}
	}
}

func TestJobTemplates_OnlyDOTComplianceReadsDrugAndAlcoholResults(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	var readers []string
	for _, template := range JobTemplates() {
		grants := reg.TemplateGrants(template)
		if strings.Contains(effectiveGrant(reg, grants, ResourceWorkerDOTTest), "read") {
			readers = append(readers, template.ID)
		}
	}

	assert.Equal(t, []string{"safety"}, readers,
		"49 CFR 40.321: only the role whose job is DOT compliance may see test results")

	var bundles []string
	for _, bundle := range Bundles() {
		for _, grant := range bundle.Grants {
			if grant.Resource == ResourceWorkerDOTTest {
				bundles = append(bundles, bundle.ID)
			}
		}
	}
	assert.Equal(t, []string{"safety.drivers"}, bundles)
}

func TestTemplateGrants_ReadOnlyReadsNoKindGrantedOnItsOwn(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	idx := slices.IndexFunc(JobTemplates(), func(template JobTemplate) bool {
		return template.ID == "read_only"
	})
	require.GreaterOrEqual(t, idx, 0)
	grants := reg.TemplateGrants(JobTemplates()[idx])

	for _, kind := range ownGrantKinds {
		assert.Empty(t, effectiveGrant(reg, grants, kind), "read_only on %s", kind)
		assert.Equal(t, kind.String(), reg.GetEffectiveResource(kind.String()),
			"%s is excepted from read-only because nothing else grants it; "+
				"a kind with a parent is reached through that parent instead", kind)
	}
	assert.NotEmpty(t, effectiveGrant(reg, grants, ResourceWorker))
	assert.NotEmpty(t, effectiveGrant(reg, grants, ResourceWorkerCredential))
}

func TestBundles_HumanResourcesAndDriverManagersInviteAndRevokePortalAccess(t *testing.T) {
	t.Parallel()

	expected := map[string][]Operation{
		"hr.workers": Normalize([]Operation{OpRead, OpCreate, OpCancel}),
		"hr.team":    Normalize([]Operation{OpRead, OpCreate, OpCancel}),
	}
	for _, bundle := range Bundles() {
		var got []Operation
		for _, grant := range bundle.Grants {
			if grant.Resource == ResourceDriverPortal {
				got = Normalize(append(got, grant.Operations...))
			}
		}
		assert.Equal(t, expected[bundle.ID], got, "bundle %q", bundle.ID)
	}
}
