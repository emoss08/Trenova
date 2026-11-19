package values_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

var testStorage = values.Storage{
	Endpoint:        "https://s3.bhs.io.cloud.ovh.net",
	Region:          "bhs",
	DocumentsBucket: "trenova-5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10-docs",
	BackupsPath:     "s3://trenova-5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10-backups/pitr/",
}

func renderStorageChart(t *testing.T, r values.Rendered) map[string]string {
	t.Helper()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	require.NoError(t, chartutil.ValidateAgainstSchema(ch, r.Values))
	vals, err := chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)
	return out
}

func TestAPlansSizeNeverResizesTheVolumesOfARunningDatabase(t *testing.T) {
	t.Parallel()
	for _, plan := range plans {
		r, err := values.Render(ptr(spec(plan, instanceapi.Active)), base(), operatorNS, &testStorage)
		require.NoError(t, err)
		require.Equal(t, false, lookup(t, r.Values, []string{"postgres", "storage"})["resizeInUseVolumes"])
		cluster := renderStorageChart(t, r)["trenova/templates/postgres.yaml"]
		require.Equal(t, 1, countLines(cluster, "    resizeInUseVolumes: false"),
			"local-path cannot grow a claim, and CloudNativePG applies nothing after a refused resize")
	}

	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	vals, err := chartutil.ToRenderValues(ch, map[string]any{"domain": "tms.example.com", "storageDomain": "files.example.com"}, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)
	require.Equal(t, 1, countLines(out["trenova/templates/postgres.yaml"], "    resizeInUseVolumes: true"),
		"the chart's own default grows a claim, for a storage class that can")
}

func TestTheInstancesBucketsReplaceMinIOAndCarryItsBackups(t *testing.T) {
	t.Parallel()
	for _, lc := range []instanceapi.InstanceSpecLifecycle{instanceapi.Active, instanceapi.Suspended, instanceapi.Decommissioned} {
		r, err := values.Render(ptr(spec(plans[1], lc)), base(), operatorNS, &testStorage)
		require.NoError(t, err)
		require.Equal(t, false, lookup(t, r.Values, []string{"minio"})["enabled"])
		st := lookup(t, r.Values, []string{"storage"})
		require.Equal(t, testStorage.DocumentsBucket, st["bucket"])
		require.Equal(t, "bhs", st["region"])
		require.Equal(t, false, st["autoCreateBucket"])
		ext := lookup(t, r.Values, []string{"storage", "external"})
		require.Equal(t, "s3.bhs.io.cloud.ovh.net", ext["endpoint"])
		require.Equal(t, testStorage.Endpoint, ext["publicEndpoint"])
		require.Equal(t, values.DocumentsSecret, ext["existingSecret"])
		backup := lookup(t, r.Values, []string{"backup"})
		require.Equal(t, true, backup["enabled"])
		require.Equal(t, testStorage.BackupsPath, backup["destinationPath"])
		require.Equal(t, values.BackupsSecret, backup["existingSecret"])

		out := renderStorageChart(t, r)
		require.NotContains(t, out["trenova/templates/datastores.yaml"], "- name: minio",
			"MinIO is still rendered for a managed instance")
		store := out["trenova/templates/backup.yaml"]
		require.Contains(t, store, "kind: ObjectStore")
		require.Contains(t, store, "destinationPath: \""+testStorage.BackupsPath+"\"")
		require.Contains(t, store, "name: "+values.BackupsSecret)
		require.Contains(t, store, "retentionPolicy: \"30d\"")
		require.Contains(t, store, "kind: ScheduledBackup")
		require.Contains(t, store, "suspend: "+map[bool]string{true: "true", false: "false"}[lc == instanceapi.Suspended])
		cluster := out["trenova/templates/postgres.yaml"]
		require.Contains(t, cluster, "isWALArchiver: true")
		require.Contains(t, cluster, "serverName: trenova-postgres")

		// The backups credential reaches no application pod: only the
		// object store names it.
		for name, body := range out {
			if name != "trenova/templates/backup.yaml" {
				require.NotContains(t, body, values.BackupsSecret, name)
			}
		}
		require.Contains(t, out["trenova/templates/tms.yaml"], "name: "+values.DocumentsSecret)
	}
}

func TestWithoutStorageTheChartKeepsMinIOAndTakesNoBackups(t *testing.T) {
	t.Parallel()
	r, err := values.Render(ptr(spec(plans[1], instanceapi.Active)), base(), operatorNS, &values.Storage{})
	require.NoError(t, err)
	_, hasBackup := r.Values["backup"]
	require.False(t, hasBackup)
	out := renderStorageChart(t, r)
	require.Empty(t, strings.TrimSpace(out["trenova/templates/backup.yaml"]))
}

func TestARestoreRendersBesideTheLiveDatabase(t *testing.T) {
	t.Parallel()
	target := time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC)
	s := spec(plans[1], instanceapi.Active)
	s.Restore = &instanceapi.SpecRestore{Id: uuid.New(), Generation: "r1a2b3c4d", SourceGeneration: "", TargetTime: target}

	r, err := values.Render(&s, base(), operatorNS, &testStorage)
	require.NoError(t, err)
	pg := lookup(t, r.Values, []string{"postgres"})
	_, switched := pg["generation"]
	require.False(t, switched, "the live database changed before the switch")
	out := renderStorageChart(t, r)
	clusters := out["trenova/templates/postgres.yaml"]
	// The live database, unchanged, and the new one recovering beside it.
	require.Equal(t, 2, strings.Count(clusters, "kind: Cluster\n"))
	require.Contains(t, clusters, "name: trenova-postgres\n")
	require.Contains(t, clusters, "name: trenova-postgres-r1a2b3c4d\n")
	require.Contains(t, clusters, "targetTime: \"2026-11-14T09:30:00Z\"")
	require.Contains(t, clusters, "source: trenova-postgres\n")
	require.Contains(t, out["trenova/templates/configmap.yaml"], "trenova-postgres-rw")
	require.NotContains(t, out["trenova/templates/configmap.yaml"], "trenova-postgres-r1a2b3c4d-rw")

	// Switched: the instance runs on the new database, which renders the
	// same, and no second copy.
	s.Database = &instanceapi.SpecDatabase{Generation: "r1a2b3c4d", SourceGeneration: "", TargetTime: target}
	r, err = values.Render(&s, base(), operatorNS, &testStorage)
	require.NoError(t, err)
	out = renderStorageChart(t, r)
	clusters = out["trenova/templates/postgres.yaml"]
	require.Equal(t, 1, strings.Count(clusters, "kind: Cluster\n"))
	require.Contains(t, clusters, "name: trenova-postgres-r1a2b3c4d\n")
	require.Contains(t, out["trenova/templates/configmap.yaml"], "trenova-postgres-r1a2b3c4d-rw")
	require.Equal(t, "trenova-postgres-r1a2b3c4d", values.PostgresCluster("trenova", "r1a2b3c4d"))

	// A restore needs the backups: without storage it is refused.
	s.Database = nil
	_, err = values.Render(&s, base(), operatorNS, &values.Storage{})
	require.ErrorIs(t, err, values.ErrInvalidSpec)
	// And a malformed generation never reaches the chart.
	s.Restore.Generation = "r1; drop"
	_, err = values.Render(&s, base(), operatorNS, &testStorage)
	require.ErrorIs(t, err, values.ErrInvalidSpec)
}

func TestAPrivateEndpointCarriesThePodsTrafficAndBrowsersKeepThePublicOne(t *testing.T) {
	t.Parallel()
	st := testStorage
	st.Endpoint = "https://files.instances.localhost"
	st.PrivateEndpoint = "http://minio.trenova.internal:9000"
	b := base()
	b["clamav"] = map[string]any{"enabled": false}
	r, err := values.Render(ptr(spec(plans[1], instanceapi.Active)), b, operatorNS, &st)
	require.NoError(t, err)
	ext := lookup(t, r.Values, []string{"storage", "external"})
	require.Equal(t, "minio.trenova.internal:9000", ext["endpoint"])
	require.Equal(t, false, ext["useSSL"])
	require.Equal(t, "https://files.instances.localhost", ext["publicEndpoint"])
	require.Equal(t, "http://minio.trenova.internal:9000", lookup(t, r.Values, []string{"backup"})["endpointURL"])
	out := renderStorageChart(t, r)
	require.Contains(t, out["trenova/templates/backup.yaml"], "endpointURL: \"http://minio.trenova.internal:9000\"")
}
