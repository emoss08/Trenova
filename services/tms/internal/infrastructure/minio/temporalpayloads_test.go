package minio_test

import (
	"strings"
	"testing"

	minioadapter "github.com/emoss08/trenova/internal/infrastructure/minio"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

func TestTemporalPayloadStore_RoundTripsAPayloadThroughObjectStorage(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	store, err := minioadapter.NewTemporalPayloadStore(setupTestClient(t, mc))
	require.NoError(t, err)

	original := &commonpb.Payload{
		Metadata: map[string][]byte{"encoding": []byte("json/plain")},
		Data:     []byte(`{"transcript":"` + strings.Repeat("a long assistant turn ", 8192) + `"}`),
	}

	claims, err := store.Store(converter.StorageDriverStoreContext{
		Context: t.Context(),
		Target: converter.StorageDriverWorkflowInfo{
			Namespace:  "default",
			WorkflowID: "assistant-thread/athr_1",
		},
	}, []*commonpb.Payload{original})
	require.NoError(t, err)
	require.Len(t, claims, 1)
	assert.True(t, strings.HasPrefix(claims[0].ClaimData["key"], "temporal-payloads/default/"))

	restored, err := store.Retrieve(
		converter.StorageDriverRetrieveContext{Context: t.Context()},
		claims,
	)
	require.NoError(t, err)
	require.Len(t, restored, 1)
	assert.Equal(t, original.GetData(), restored[0].GetData())
	assert.Equal(t, original.GetMetadata(), restored[0].GetMetadata())
}

// A retried store of the same payload must land on the same object, not leave
// a second copy behind.
func TestTemporalPayloadStore_StoringTheSamePayloadTwiceWritesOneObject(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	store, err := minioadapter.NewTemporalPayloadStore(setupTestClient(t, mc))
	require.NoError(t, err)

	payload := &commonpb.Payload{Data: []byte("the same bytes")}
	storeCtx := converter.StorageDriverStoreContext{
		Context: t.Context(),
		Target:  converter.StorageDriverWorkflowInfo{Namespace: "default", WorkflowID: "wf"},
	}

	first, err := store.Store(storeCtx, []*commonpb.Payload{payload})
	require.NoError(t, err)
	second, err := store.Store(storeCtx, []*commonpb.Payload{payload})
	require.NoError(t, err)

	assert.Equal(t, first[0].ClaimData["key"], second[0].ClaimData["key"])
}

// An object that no longer matches its claim must fail loudly here rather than
// decode into a payload a workflow then chokes on.
func TestTemporalPayloadStore_RefusesAnObjectThatNoLongerMatchesItsClaim(t *testing.T) {
	tc, mc := testutil.SetupTestMinio(t)
	defer tc.Cancel()

	store, err := minioadapter.NewTemporalPayloadStore(setupTestClient(t, mc))
	require.NoError(t, err)

	claims, err := store.Store(converter.StorageDriverStoreContext{
		Context: t.Context(),
		Target:  converter.StorageDriverWorkflowInfo{Namespace: "default", WorkflowID: "wf"},
	}, []*commonpb.Payload{{Data: []byte("original")}})
	require.NoError(t, err)

	tampered := []byte("not what was stored")
	_, err = mc.Client().PutObject(
		t.Context(),
		mc.Bucket(),
		claims[0].ClaimData["key"],
		strings.NewReader(string(tampered)),
		int64(len(tampered)),
		minio.PutObjectOptions{},
	)
	require.NoError(t, err)

	_, err = store.Retrieve(converter.StorageDriverRetrieveContext{Context: t.Context()}, claims)
	require.Error(t, err)
}
