package storedobject_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks/contentscanportsmock"
	"github.com/emoss08/trenova/internal/testutil/objectstoretest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encryption() *encryptionservice.Service {
	return encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("storedobject-test-key-material"),
	)
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func write(
	t *testing.T,
	store *objectstoretest.Store,
	enc *encryptionservice.Service,
	key string,
	aad encryptionservice.AAD,
	body []byte,
) *storedobject.Written {
	t.Helper()
	written, err := storedobject.Write(t.Context(), store, enc, &storedobject.WriteRequest{
		Key:         key,
		ContentType: "application/pdf",
		AAD:         aad,
		Body:        bytes.NewReader(body),
		Size:        int64(len(body)),
	})
	require.NoError(t, err)
	return written
}

func TestAnObjectIsStoredSealedAndOpensAsItStreams(t *testing.T) {
	t.Parallel()

	store := objectstoretest.New()
	enc := encryption()
	key := "org/documents/ratecon.pdf"
	aad := storedobject.AAD(tenant(), key)
	body := bytes.Repeat([]byte("Load 48213 Rate $2,450 "), 20_000)

	written := write(t, store, enc, key, aad, body)
	sum := sha256.Sum256(body)
	assert.Equal(t, hex.EncodeToString(sum[:]), written.SHA256)

	stored, ok := store.Object(key)
	require.True(t, ok)
	assert.True(t, encryptionservice.IsBlob(stored))
	assert.NotContains(t, string(stored), "Load 48213")

	object, err := storedobject.Open(t.Context(), store, enc, key, aad)
	require.NoError(t, err)
	defer object.Body.Close()
	assert.Equal(t, int64(len(body)), object.Size)
	got, err := io.ReadAll(object.Body)
	require.NoError(t, err)
	assert.Equal(t, body, got)

	_, err = storedobject.Open(t.Context(), store, enc, key, storedobject.AAD(tenant(), key))
	require.Error(t, err, "another tenant's associated data does not open it")
}

func TestAnObjectInTheEnvelopeFormatIsRefused(t *testing.T) {
	t.Parallel()

	store := objectstoretest.New()
	enc := encryption()
	key := "org/documents/old.pdf"
	aad := storedobject.AAD(tenant(), key)
	envelope, err := enc.EncryptBytesWithAAD([]byte("%PDF-1.7"), aad)
	require.NoError(t, err)
	store.Put(key, []byte(envelope))

	_, err = storedobject.Open(t.Context(), store, enc, key, aad)
	require.ErrorIs(t, err, storedobject.ErrNotSealed)
}

func TestAnInfectedUploadIsNeverStored(t *testing.T) {
	t.Parallel()

	store := objectstoretest.New()
	gate := contentscanportsmock.NewInfectedGate("Win.Test.EICAR_HDB-1")
	body := bytes.Repeat([]byte("x"), 3*encryptionservice.BlobChunkSize)
	key := "org/documents/bad.pdf"

	_, err := storedobject.Write(t.Context(), store, encryption(), &storedobject.WriteRequest{
		Key:  key,
		AAD:  storedobject.AAD(tenant(), key),
		Body: bytes.NewReader(body),
		Size: int64(len(body)),
		Gate: gate,
		Scan: &contentscanports.StreamRequest{
			Source:   contentscanports.SourceUpload,
			FileName: "bad.pdf",
		},
	})

	require.ErrorIs(t, err, contentscanports.ErrInfected)
	_, stored := store.Object(key)
	assert.False(t, stored, "the last chunk waited for the verdict, so the upload never completed")
	assert.Equal(t, 1, store.Attempts())
	require.Len(t, gate.Requests(), 1)
	assert.Equal(t, body, gate.Requests()[0].Content)
}

func TestABodyThatIsNotItsDeclaredSizeIsRefused(t *testing.T) {
	t.Parallel()

	store := objectstoretest.New()
	key := "org/documents/short.pdf"
	_, err := storedobject.Write(t.Context(), store, encryption(), &storedobject.WriteRequest{
		Key:  key,
		AAD:  storedobject.AAD(tenant(), key),
		Body: bytes.NewReader([]byte("short")),
		Size: 50,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))
	assert.Empty(t, store.Keys())
}

func TestReadRefusesAnObjectOverItsLimit(t *testing.T) {
	t.Parallel()

	store := objectstoretest.New()
	enc := encryption()
	key := "org/renders/page.webp"
	aad := storedobject.AAD(tenant(), key)
	write(t, store, enc, key, aad, bytes.Repeat([]byte("p"), 1000))

	_, err := storedobject.Read(t.Context(), store, enc, key, aad, 999)
	require.ErrorIs(t, err, storedobject.ErrTooLarge)
	data, err := storedobject.Read(t.Context(), store, enc, key, aad, 1000)
	require.NoError(t, err)
	assert.Len(t, data, 1000)

	_, found, err := storedobject.ReadOptional(t.Context(), store, enc, "missing", aad, 1000)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestASourceStreamsTheObjectAgainForEachAttempt(t *testing.T) {
	t.Parallel()

	store := objectstoretest.New()
	enc := encryption()
	key := "org/documents/bol.pdf"
	aad := storedobject.AAD(tenant(), key)
	body := []byte("%PDF-1.7 bill of lading")
	write(t, store, enc, key, aad, body)

	source := storedobject.Source(store, enc, key, aad, "application/pdf", int64(len(body)))
	assert.Equal(t, int64(len(body)), source.Size)
	for range 2 {
		file, err := source.Open(t.Context())
		require.NoError(t, err)
		got, err := io.ReadAll(file)
		require.NoError(t, err)
		require.NoError(t, file.Close())
		assert.Equal(t, body, got)
	}

	wrong := storedobject.Source(store, enc, key, aad, "application/pdf", 3)
	_, err := wrong.Open(t.Context())
	require.ErrorIs(t, err, kernelports.ErrImagingInvalidDocument)
}
