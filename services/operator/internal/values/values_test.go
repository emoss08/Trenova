package values_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

var chartDir = filepath.Join("..", "..", "..", "..", "deploy", "helm", "trenova")

const operatorNS = "trenova-system"

var plans = []instanceapi.SpecPlan{
	{Key: "sandbox", Vcpu: 4, MemoryGiB: 16, DiskGiB: 50},
	{Key: "production", Vcpu: 8, MemoryGiB: 32, DiskGiB: 200},
	{Key: "production_plus", Vcpu: 16, MemoryGiB: 64, DiskGiB: 500},
}

func spec(plan instanceapi.SpecPlan, lc instanceapi.InstanceSpecLifecycle, components ...string) instanceapi.InstanceSpec {
	s := controlplanetest.Spec(uuid.New())
	s.Plan = plan
	s.Lifecycle = lc
	s.Components = append([]string{}, components...)
	return s
}

func base() map[string]any {
	return map[string]any{
		"clamav":  map[string]any{"enabled": true},
		"ingress": map[string]any{"enabled": true, "className": "traefik"},
	}
}

func TestEveryPlanAndLifecycleRendersThroughTheChart(t *testing.T) {
	t.Parallel()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	for _, plan := range plans {
		for _, lc := range []instanceapi.InstanceSpecLifecycle{instanceapi.Active, instanceapi.Suspended, instanceapi.Decommissioned} {
			for _, comps := range [][]string{{}, {"docling"}} {
				name := fmt.Sprintf("%s/%s/%v", plan.Key, lc, comps)
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					r, err := values.Render(ptr(spec(plan, lc, comps...)), base(), operatorNS)
					require.NoError(t, err)
					require.Equal(t, "0.11.0", r.Version)

					vals, err := chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
						Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
					}, chartutil.DefaultCapabilities)
					require.NoError(t, err)
					require.NoError(t, chartutil.ValidateAgainstSchema(ch, r.Values))
					out, err := engine.Render(ch, vals)
					require.NoError(t, err)

					job := false
					for name, body := range out {
						if strings.HasSuffix(name, "release-job.yaml") && strings.Contains(body, "kind: Job") {
							job = true
						}
					}
					require.Equal(t, lc == instanceapi.Active, job)
				})
			}
		}
	}
}

func TestRequestsFitThePlan(t *testing.T) {
	t.Parallel()
	for _, plan := range plans {
		r, err := values.Render(ptr(spec(plan, instanceapi.Active, "docling")), base(), operatorNS)
		require.NoError(t, err)
		var mem, cpu int64
		for _, path := range [][]string{
			{"postgres"}, {"tms", "api"}, {"tms", "worker"}, {"imaging"}, {"redis"}, {"meilisearch"},
			{"temporal"}, {"gotenberg"}, {"minio"}, {"gtc"}, {"client"}, {"clamav"}, {"docling"},
		} {
			req := lookup(t, r.Values, append(path, "resources", "requests"))
			m := resource.MustParse(req["memory"].(string))
			c := resource.MustParse(req["cpu"].(string))
			mem += m.Value()
			cpu += c.MilliValue()
		}
		require.LessOrEqual(t, mem, (plan.MemoryGiB-1)<<30, plan.Key)
		require.LessOrEqual(t, cpu, plan.Vcpu*1000, plan.Key)
	}
}

func TestLifecycles(t *testing.T) {
	t.Parallel()
	cases := map[instanceapi.InstanceSpecLifecycle][2]bool{
		instanceapi.Active:         {false, false},
		instanceapi.Suspended:      {true, true},
		instanceapi.Decommissioned: {true, false},
	}
	for lc, want := range cases {
		r, err := values.Render(ptr(spec(plans[0], lc)), base(), operatorNS)
		require.NoError(t, err)
		require.Equal(t, want[0], r.Values["stopped"], lc)
		require.Equal(t, want[1], lookup(t, r.Values, []string{"postgres"})["hibernate"], lc)
	}
}

func TestDerivedValuesOverrideTheBase(t *testing.T) {
	t.Parallel()
	b := base()
	b["domain"] = "wrong.example.com"
	b["tms"] = map[string]any{"image": map[string]any{"tag": "ci"}}
	s := spec(plans[1], instanceapi.Active, "docling")
	r, err := values.Render(&s, b, operatorNS)
	require.NoError(t, err)
	require.Equal(t, s.Hostname, r.Values["domain"])
	require.Equal(t, "k7q2m9x4p1r8t3v6-files.trenova.cloud", r.Values["storageDomain"])
	tms := lookup(t, r.Values, []string{"tms"})
	require.Equal(t, s.InstanceId.String(), tms["instanceId"])
	require.Equal(t, "ci", lookup(t, tms, []string{"image"})["tag"])
	require.Equal(t, true, lookup(t, r.Values, []string{"docling"})["enabled"])
	require.Equal(t, "wrong.example.com", b["domain"], "the base is not modified")
}

func TestRefusesWhatItCannotApply(t *testing.T) {
	t.Parallel()
	s := spec(plans[0], instanceapi.Active)
	s.SchemaVersion = 2
	_, err := values.Render(&s, base(), operatorNS)
	require.ErrorIs(t, err, values.ErrSchemaVersion)

	for name, mutate := range map[string]func(*instanceapi.InstanceSpec){
		"version":   func(s *instanceapi.InstanceSpec) { s.Release.Version = "latest" },
		"v-prefix":  func(s *instanceapi.InstanceSpec) { s.Release.Version = "v0.11.0" },
		"hostname":  func(s *instanceapi.InstanceSpec) { s.Hostname = "Not A Host" },
		"lifecycle": func(s *instanceapi.InstanceSpec) { s.Lifecycle = "paused" },
		"tiny plan": func(s *instanceapi.InstanceSpec) { s.Plan.MemoryGiB = 4 },
	} {
		s := spec(plans[0], instanceapi.Active)
		mutate(&s)
		_, err := values.Render(&s, base(), operatorNS)
		require.ErrorIs(t, err, values.ErrInvalidSpec, name)
	}
}

func TestManagedModeAdmitsOnlyTheOperatorsNamespace(t *testing.T) {
	t.Parallel()
	s := spec(plans[0], instanceapi.Active)
	r, err := values.Render(&s, base(), operatorNS)
	require.NoError(t, err)
	managed := lookup(t, r.Values, []string{"managed"})
	require.Equal(t, true, managed["enabled"])
	require.Equal(t, []any{map[string]any{
		"namespaceSelector": map[string]any{
			"matchLabels": map[string]any{"kubernetes.io/metadata.name": operatorNS},
		},
	}}, managed["from"])

	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	vals, err := chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)
	policies := out["trenova/templates/networkpolicy.yaml"]
	require.Contains(t, policies, "port: 8090")
	require.Contains(t, policies, "kubernetes.io/metadata.name: "+operatorNS)
	require.Contains(t, out["trenova/templates/tms.yaml"], "name: managed")
	require.Contains(t, out["trenova/templates/secrets.yaml"], "managedToken:")
	require.Contains(t, out["trenova/templates/configmap.yaml"], "port: 8090")

	_, err = values.Render(&s, base(), "")
	require.Error(t, err)
}

func TestNamesFollowTheChart(t *testing.T) {
	t.Parallel()
	cases := []struct {
		release string
		base    map[string]any
		secret  string
		api     string
	}{
		{release: "trenova", secret: "trenova-secrets", api: "trenova-api"},
		{release: "trenova-prod", secret: "trenova-prod-secrets", api: "trenova-prod-api"},
		{release: "acme", secret: "acme-trenova-secrets", api: "acme-trenova-api"},
		{
			release: "trenova",
			base:    map[string]any{"secrets": map[string]any{"existingSecret": "app"}},
			secret:  "app",
			api:     "trenova-api",
		},
		{
			release: "a-very-long-release-name-that-goes-on-and-on",
			secret:  "a-very-long-release-name-that-goes-on-an-secrets",
			api:     "a-very-long-release-name-that-goes-on-an-api",
		},
		{
			release: "abcdefghijklmnopqrstuvwxyz-0123456789ab",
			secret:  "abcdefghijklmnopqrstuvwxyz-0123456789ab-secrets",
			api:     "abcdefghijklmnopqrstuvwxyz-0123456789ab-api",
		},
	}
	for _, tc := range cases {
		t.Run(tc.release, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.secret, values.AppSecret(tc.release, tc.base))
			require.Equal(t, tc.api, values.APIService(tc.release))
		})
	}
}

func lookup(t *testing.T, m map[string]any, path []string) map[string]any {
	t.Helper()
	cur := m
	for _, k := range path {
		next, ok := cur[k].(map[string]any)
		require.True(t, ok, "missing %s", strings.Join(path, "."))
		cur = next
	}
	return cur
}

func ptr[T any](v T) *T { return &v }
