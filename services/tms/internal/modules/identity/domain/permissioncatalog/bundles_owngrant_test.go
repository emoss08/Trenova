package permissioncatalog

import (
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func effectiveGrant(
	reg *Registry,
	grants []*permission.ResourcePermission,
	kind permission.Resource,
) string {
	idx := slices.IndexFunc(grants, func(rp *permission.ResourcePermission) bool {
		return rp.Resource == kind.String()
	})
	if idx < 0 {
		return ""
	}
	ops := make([]string, 0, 8)
	for _, op := range permission.Implied(grants[idx].Operations) {
		if reg.Declares(kind.String(), op) {
			ops = append(ops, string(op))
		}
	}
	return strings.Join(ops, ",") + "@" + string(grants[idx].DataScope)
}

func TestJobTemplates_KindsGrantedOnTheirOwn(t *testing.T) {
	t.Parallel()

	kinds := []permission.Resource{
		permission.ResourceWorkerInjury,
		permission.ResourceWorkerDOTTest,
		permission.ResourceBenefitPlan,
		permission.ResourceWorkerEmploymentEvent,
		permission.ResourceWorkerDisciplinaryAction,
		permission.ResourcePerformanceReview,
		permission.ResourceTimesheet,
		permission.ResourceProfileChangeRequest,
		permission.ResourceDriverPortal,
		permission.ResourceWorkerLeave,
	}
	none := map[permission.Resource]string{}
	expected := map[string]map[permission.Resource]string{
		"dispatcher":          none,
		"dispatch_supervisor": none,
		"carrier_rep":         none,
		"billing_clerk":       none,
		"billing_manager":     none,
		"accountant":          none,
		"payroll":             none,
		"safety": {
			permission.ResourceWorkerInjury:  "create,read,update@organization",
			permission.ResourceWorkerDOTTest: "create,read,update@organization",
		},
		"driver_manager": {
			permission.ResourceTimesheet:            "approve,read,reject,update@team",
			permission.ResourceProfileChangeRequest: "approve,read,reject@team",
			permission.ResourceDriverPortal:         "cancel,create,read@team",
			permission.ResourceWorkerLeave:          "approve,read,update@team",
		},
		"hr": {
			permission.ResourceBenefitPlan:              "assign,create,read,update@organization",
			permission.ResourceWorkerEmploymentEvent:    "create,read,update@organization",
			permission.ResourceWorkerDisciplinaryAction: "create,read@organization",
			permission.ResourcePerformanceReview:        "create,read,update@organization",
			permission.ResourceTimesheet:                "approve,read,reject,update@team",
			permission.ResourceProfileChangeRequest:     "approve,read,reject@team",
			permission.ResourceDriverPortal:             "cancel,create,read@organization",
			permission.ResourceWorkerLeave:              "approve,create,read,update@organization",
		},
		"read_only": {
			permission.ResourceDriverPortal: "read@organization",
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
		if strings.Contains(effectiveGrant(reg, grants, permission.ResourceWorkerDOTTest), "read") {
			readers = append(readers, template.ID)
		}
	}

	assert.Equal(t, []string{"safety"}, readers,
		"49 CFR 40.321: only the role whose job is DOT compliance may see test results")

	var bundles []string
	for _, bundle := range Bundles() {
		for _, grant := range bundle.Grants {
			if grant.Resource == permission.ResourceWorkerDOTTest {
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
	assert.NotEmpty(t, effectiveGrant(reg, grants, permission.ResourceWorker))
	assert.NotEmpty(t, effectiveGrant(reg, grants, permission.ResourceWorkerCredential))
}

func TestBundles_HumanResourcesAndDriverManagersInviteAndRevokePortalAccess(t *testing.T) {
	t.Parallel()

	expected := map[string][]permission.Operation{
		"hr.workers": permission.Normalize(
			[]permission.Operation{permission.OpRead, permission.OpCreate, permission.OpCancel},
		),
		"hr.team": permission.Normalize(
			[]permission.Operation{permission.OpRead, permission.OpCreate, permission.OpCancel},
		),
	}
	for _, bundle := range Bundles() {
		var got []permission.Operation
		for _, grant := range bundle.Grants {
			if grant.Resource == permission.ResourceDriverPortal {
				got = permission.Normalize(append(got, grant.Operations...))
			}
		}
		assert.Equal(t, expected[bundle.ID], got, "bundle %q", bundle.ID)
	}
}
