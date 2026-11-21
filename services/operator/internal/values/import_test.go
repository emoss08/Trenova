package values_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const importKeyMaterial = "Zml4dHVyZS1hcmNoaXZlLWtleS0zMi1ieXRlcy14eXo="

func importSpec() *instanceapi.SpecImport {
	id := uuid.MustParse("3c9f0a52-6b4e-4f1d-9a7e-1d2c3b4a5f60")
	return &instanceapi.SpecImport{
		Id:           id,
		Bucket:       values.BackupsBucket(&testStorage),
		Prefix:       "imports/" + id.String() + "/",
		ExpectSha256: strings.Repeat("ab", 32),
		ArchiveKeys: []instanceapi.ArchiveKey{
			{Id: "region-1", Key: importKeyMaterial, Current: true},
		},
	}
}

func renderImport(
	t *testing.T,
	lc instanceapi.InstanceSpecLifecycle,
	in *values.Import,
) (values.Rendered, string) {
	t.Helper()
	s := spec(plans[0], lc)
	s.Import = in.Spec
	r, err := values.Render(&s, base(), operatorNS, &testStorage)
	require.NoError(t, err)
	require.NoError(t, values.ApplyImport(r.Values, &testStorage, in))
	out := renderStorageChart(t, r)
	job := ""
	for name, body := range out {
		if strings.HasSuffix(name, "templates/release-job.yaml") {
			job = body
		}
	}
	return r, job
}

func TestAHeldImportRunsTheDatabaseAndTheImportButNoWorkload(t *testing.T) {
	t.Parallel()
	imp := importSpec()
	r, job := renderImport(t, instanceapi.Suspended, &values.Import{
		Spec: imp, KeysSecret: values.ImportKeysSecret("trenova"),
	})

	assert.Equal(t, true, r.Values["stopped"], "nothing that serves a person starts")
	pg, _ := r.Values["postgres"].(map[string]any)
	assert.Equal(t, false, pg["hibernate"], "the database runs for the import")
	require.NotEmpty(t, job, "the release Job runs while the import is held")
	assert.Contains(t, job, "name: import")
	assert.NotContains(t, job, "name: bootstrap")
	assert.Contains(t, job, "trenova.app/import: \""+imp.Id.String()+"\"")
	assert.Contains(t, job, "- --id="+imp.Id.String())
	assert.Contains(t, job, "- --status-file=/dev/termination-log")
	assert.Contains(t, job, "values: [2]", "a refusal fails the Job at once")
	assert.Contains(t, job, "name: trenova-import-keys")
	assert.NotContains(t, job, importKeyMaterial, "the keys are never in the manifest")
	assert.Contains(
		t,
		job,
		"name: trenova-backup-storage\n                  key: ACCESS_SECRET_KEY",
		"the bucket secret is referenced from its Secret, never inlined",
	)
	flat, err := yamlOf(r.Values)
	require.NoError(t, err)
	assert.NotContains(t, flat, importKeyMaterial, "the keys are never a value Helm stores")
}

func TestASettledImportRunsNeitherTheImportNorTheBootstrapAgain(t *testing.T) {
	t.Parallel()
	_, job := renderImport(
		t,
		instanceapi.Suspended,
		&values.Import{Spec: importSpec(), Settled: true},
	)
	assert.NotContains(t, job, "name: import\n")
	assert.NotContains(t, job, "trenova.app/import")
	assert.Contains(t, job, "name: bootstrap")
	assert.NotContains(
		t,
		job,
		"TRENOVA_BOOTSTRAP_ADMIN_EMAIL\n              value: \"dana",
		"the bootstrap has no one to make",
	)

	r, job := renderImport(t, instanceapi.Active, &values.Import{ImportedBefore: true})
	assert.Equal(t, false, r.Values["stopped"])
	boot, _ := r.Values["bootstrap"].(map[string]any)
	assert.Empty(
		t,
		boot["adminEmail"],
		"an imported instance is never bootstrapped after the switch",
	)
	assert.NotContains(t, job, "import-archive")
}

func TestAnImportOutsideTheInstancesBackupsIsRefused(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(*instanceapi.SpecImport){
		"another bucket": func(s *instanceapi.SpecImport) { s.Bucket = "someone-elses" },
		"another prefix": func(s *instanceapi.SpecImport) { s.Prefix = "tenants/x/" },
		"no keys":        func(s *instanceapi.SpecImport) { s.ArchiveKeys = nil },
		"a bad digest":   func(s *instanceapi.SpecImport) { s.ExpectSha256 = "zz" },
	} {
		imp := importSpec()
		change(imp)
		require.ErrorIs(t, values.CheckImport(imp, &testStorage), values.ErrInvalidSpec, name)
	}
	require.ErrorIs(t, values.CheckImport(importSpec(), &values.Storage{}), values.ErrInvalidSpec)
}

func yamlOf(v map[string]any) (string, error) {
	out, err := yaml.Marshal(v)
	return string(out), err
}
