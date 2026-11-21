package importer_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/importer"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const keyFixture = "Zml4dHVyZS1hcmNoaXZlLWtleS0zMi1ieXRlcy14eXo="

var storage = values.Storage{
	Endpoint: "https://s3.test", DocumentsBucket: "docs", BackupsPath: "s3://backups/pitr/",
}

func spec() *instanceapi.InstanceSpec {
	id := uuid.New()
	return &instanceapi.InstanceSpec{
		Lifecycle: instanceapi.Suspended,
		Import: &instanceapi.SpecImport{
			Id: id, Bucket: "backups", Prefix: "imports/" + id.String() + "/",
			ExpectSha256: strings.Repeat("ab", 32),
			ArchiveKeys: []instanceapi.ArchiveKey{
				{Id: "import-1", Key: keyFixture, Current: true},
			},
		},
	}
}

func TestTheKeysLiveOnlyInTheirSecretUntilTheImportSettlesAndNeverInALog(t *testing.T) {
	t.Parallel()
	kube := fake.NewClientset()
	logs := &bytes.Buffer{}
	imp, err := importer.New(&importer.Params{
		Kube: kube, Namespace: "trenova", Release: "trenova",
		Logger: slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	require.NoError(t, err)
	s := spec()

	planned, err := imp.Prepare(t.Context(), s, &storage)
	require.NoError(t, err)
	assert.Equal(t, values.ImportKeysSecret("trenova"), planned.KeysSecret)
	secret, err := kube.CoreV1().Secrets("trenova").
		Get(t.Context(), planned.KeysSecret, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Contains(t, string(secret.Data[values.ImportKeysKey]), keyFixture)

	status, reported, err := imp.Observe(t.Context(), s, "trenova-release-1")
	require.NoError(t, err)
	assert.False(t, reported)
	assert.Nil(t, status, "nothing is reported before the release Job exists")

	_, err = kube.BatchV1().Jobs("trenova").Create(t.Context(), &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: "trenova-release-1", Namespace: "trenova",
			Annotations: map[string]string{"trenova.app/import": s.Import.Id.String()},
		},
		Status: batchv1.JobStatus{Conditions: []batchv1.JobCondition{{
			Type: batchv1.JobComplete, Status: corev1.ConditionTrue,
		}}},
	}, metav1.CreateOptions{})
	require.NoError(t, err)
	status, reported, err = imp.Observe(t.Context(), s, "trenova-release-1")
	require.NoError(t, err)
	require.True(t, reported)
	require.NotNil(t, status)
	assert.Equal(t, instanceapi.ImportDone, status.State)
	_, err = kube.CoreV1().
		Secrets("trenova").
		Get(t.Context(), planned.KeysSecret, metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "the keys are deleted when the import settles")

	again, err := imp.Prepare(t.Context(), s, &storage)
	require.NoError(t, err)
	assert.True(t, again.Settled)

	other := spec()
	_, err = imp.Prepare(t.Context(), other, &storage)
	require.ErrorIs(t, err, values.ErrInvalidSpec, "an imported instance is never imported again")

	switched := *s
	switched.Import = nil
	switched.Lifecycle = instanceapi.Active
	after, err := imp.Prepare(t.Context(), &switched, &storage)
	require.NoError(t, err)
	assert.True(t, after.ImportedBefore)

	assert.NotContains(t, logs.String(), keyFixture)
	assert.WithinDuration(t, time.Now(), status.At, time.Minute)
}

func TestTheImportSecretHoldsTheImportsOneKeyAndTwoAreRefused(t *testing.T) {
	t.Parallel()
	kube := fake.NewClientset()
	imp, err := importer.New(&importer.Params{
		Kube: kube, Namespace: "trenova", Release: "trenova",
		Logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
	})
	require.NoError(t, err)

	two := spec()
	two.Import.ArchiveKeys = append(two.Import.ArchiveKeys, instanceapi.ArchiveKey{
		Id: "import-2", Key: keyFixture,
	})
	_, err = imp.Prepare(t.Context(), two, &storage)
	require.ErrorIs(t, err, values.ErrInvalidSpec)
	_, err = kube.CoreV1().Secrets("trenova").
		Get(t.Context(), values.ImportKeysSecret("trenova"), metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "a refused import keeps no key")

	none := spec()
	none.Import.ArchiveKeys = nil
	_, err = imp.Prepare(t.Context(), none, &storage)
	require.ErrorIs(t, err, values.ErrInvalidSpec)

	planned, err := imp.Prepare(t.Context(), spec(), &storage)
	require.NoError(t, err)
	secret, err := kube.CoreV1().Secrets("trenova").
		Get(t.Context(), planned.KeysSecret, metav1.GetOptions{})
	require.NoError(t, err)
	var held []instanceapi.ArchiveKey
	require.NoError(t, sonic.Unmarshal(secret.Data[values.ImportKeysKey], &held))
	require.Len(t, held, 1)
	assert.Equal(t, "import-1", held[0].Id)
	assert.True(t, held[0].Current)
}
