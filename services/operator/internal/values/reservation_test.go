package values_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

type machine struct {
	name    string
	plan    instanceapi.SpecPlan
	base    func() map[string]any
	storage values.Storage
}

func machines() []machine {
	local := values.Capacity{VCPU: 2, MemoryGiB: 10, DiskGiB: 40}
	localStorage := testStorage
	localStorage.PrivateEndpoint = "http://minio.trenova.internal:9000"
	noScan := func() map[string]any {
		b := base()
		b["clamav"] = map[string]any{"enabled": false}
		return b
	}
	var out []machine
	for _, plan := range plans {
		out = append(out,
			machine{name: plan.Key, plan: plan, base: base},
			machine{name: plan.Key + "/storage", plan: plan, base: base, storage: testStorage},
			machine{name: plan.Key + "/local", plan: local.Fit(plan), base: noScan, storage: localStorage},
			machine{name: plan.Key + "/local-minio", plan: local.Fit(plan), base: noScan},
		)
	}
	return out
}

func TestTurningAComponentOnLeavesEveryOtherWorkloadAsItWas(t *testing.T) {
	t.Parallel()
	for _, component := range []string{"docling", "embedding"} {
		for _, m := range machines() {
			t.Run(component+"/"+m.name, func(t *testing.T) {
				t.Parallel()
				turningOnLeavesTheRestAsItWas(t, m, component)
			})
		}
	}
}

func turningOnLeavesTheRestAsItWas(t *testing.T, m machine, component string) {
	t.Helper()
	without := spec(m.plan, instanceapi.Active)
	with := spec(m.plan, instanceapi.Active, component)
	with.InstanceId = without.InstanceId
	off, err := values.Render(&without, m.base(), operatorNS, m.storage)
	require.NoError(t, err)
	on, err := values.Render(&with, m.base(), operatorNS, m.storage)
	require.NoError(t, err)

	require.Equal(t, false, lookup(t, off.Values, []string{component})["enabled"])
	require.Equal(t, true, lookup(t, on.Values, []string{component})["enabled"])
	require.Equal(t,
		lookup(t, on.Values, []string{component, "resources"}),
		lookup(t, off.Values, []string{component, "resources"}),
		"%s's share is kept for it while it is off", component,
	)
	require.Equal(t,
		lookup(t, off.Values, []string{"postgres", "parameters"}),
		lookup(t, on.Values, []string{"postgres", "parameters"}),
		"PostgreSQL's settings",
	)
	require.Equal(t,
		lookup(t, off.Values, []string{"postgres", "resources"}),
		lookup(t, on.Values, []string{"postgres", "resources"}),
		"PostgreSQL's resources",
	)

	delete(lookup(t, off.Values, []string{component}), "enabled")
	delete(lookup(t, on.Values, []string{component}), "enabled")
	require.Equal(t, off.Values, on.Values, "only %s.enabled differs", component)

	off.Values[component].(map[string]any)["enabled"] = false
	on.Values[component].(map[string]any)["enabled"] = true
	before := renderStorageChart(t, off)
	after := renderStorageChart(t, on)
	for _, file := range []string{"postgres.yaml", "datastores.yaml", "networkpolicy.yaml", "backup.yaml"} {
		require.Equal(t, before["trenova/templates/"+file], after["trenova/templates/"+file], file)
	}
	require.Equal(t,
		withoutComponent(before["trenova/templates/services.yaml"], component),
		withoutComponent(after["trenova/templates/services.yaml"], component),
		"every workload in services.yaml but %s", component,
	)
	require.NotEqual(t,
		before["trenova/templates/services.yaml"],
		after["trenova/templates/services.yaml"],
		"%s itself is added", component,
	)
}

func TestTheEmbeddingServiceReachesTheTMSOnlyThroughManagedConfig(t *testing.T) {
	t.Parallel()
	r, err := values.Render(ptr(spec(plans[0], instanceapi.Active, "embedding")), base(), operatorNS, values.Storage{})
	require.NoError(t, err)
	rendered := renderStorageChart(t, r)

	config := rendered["trenova/templates/configmap.yaml"]
	require.Contains(t, config, "embedding:", "the API and worker are told where the service is")
	require.Contains(t, config, "model: nomic-embed-text:v1.5")
	require.Regexp(t, `url: http://[a-z0-9-]+-embedding:11434`, config)

	policies := rendered["trenova/templates/networkpolicy.yaml"]
	var ingress, egress string
	for _, doc := range strings.Split(policies, "\n---\n") {
		if !strings.Contains(doc, "app.kubernetes.io/component: embedding\n") {
			continue
		}
		if strings.Contains(doc, "policyTypes: [Egress]") || strings.Contains(doc, "- Egress") {
			egress = doc
		} else {
			ingress = doc
		}
	}
	require.NotEmpty(t, ingress, "the service admits only named peers")
	require.Contains(t, ingress, "app.kubernetes.io/component: api")
	require.Contains(t, ingress, "app.kubernetes.io/component: worker")
	require.NotContains(t, ingress, "app.kubernetes.io/component: client")
	require.NotContains(t, ingress, "app.kubernetes.io/component: release")
	require.Contains(t, ingress, "port: 11434")
	require.NotEmpty(t, egress, "the service opens no connections")
	require.Contains(t, egress, "egress: []")
}

func TestAPlanChangeStillDividesTheMachineAgain(t *testing.T) {
	t.Parallel()
	for i := 1; i < len(plans); i++ {
		smaller, err := values.Render(ptr(spec(plans[i-1], instanceapi.Active)), base(), operatorNS, values.Storage{})
		require.NoError(t, err)
		larger, err := values.Render(ptr(spec(plans[i], instanceapi.Active)), base(), operatorNS, values.Storage{})
		require.NoError(t, err)
		name := fmt.Sprintf("%s to %s", plans[i-1].Key, plans[i].Key)
		for _, path := range [][]string{
			{"postgres", "resources"}, {"postgres", "parameters"}, {"tms", "api", "resources"},
			{"tms", "worker", "resources"}, {"docling", "resources"},
		} {
			require.NotEqual(t,
				lookup(t, smaller.Values, path),
				lookup(t, larger.Values, path),
				"%s: %s", name, strings.Join(path, "."),
			)
		}
	}
}

func withoutComponent(manifest, component string) []string {
	var out []string
	for _, doc := range strings.Split(manifest, "\n---\n") {
		if strings.Contains(doc, "app.kubernetes.io/component: "+component+"\n") {
			continue
		}
		out = append(out, strings.TrimSpace(doc))
	}
	return out
}
