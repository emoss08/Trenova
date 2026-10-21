package encryptionservice

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func blobAAD(resource string) AAD {
	return AAD{
		Purpose:        PurposeDocument,
		OrganizationID: pulid.ID("org_01J0000000000000000000000A"),
		BusinessUnitID: pulid.ID("bu_01J0000000000000000000000B"),
		ResourceID:     resource,
	}
}

func blobService(t *testing.T, active string, retired ...string) *Service {
	t.Helper()
	keyring, err := NewLocalKeyring(active, retired)
	require.NoError(t, err)
	return NewWithKeyManager(keyring)
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	data := make([]byte, n)
	_, err := rand.Read(data)
	require.NoError(t, err)
	return data
}

func seal(t *testing.T, svc *Service, plaintext []byte, aad AAD) []byte {
	t.Helper()
	sealed, err := svc.SealBlob(t.Context(), bytes.NewReader(plaintext), int64(len(plaintext)), aad)
	require.NoError(t, err)
	out, err := io.ReadAll(sealed)
	require.NoError(t, err)
	require.Equal(t, sealed.Size(), int64(len(out)), "Size is the exact sealed length")
	return out
}

func open(t *testing.T, svc *Service, sealed []byte, aad AAD) ([]byte, error) {
	t.Helper()
	opened, err := svc.OpenBlob(t.Context(), bytes.NewReader(sealed), int64(len(sealed)), aad)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(opened)
}

func headerLength(t *testing.T, sealed []byte) int {
	t.Helper()
	header, err := readBlobHeader(bytes.NewReader(sealed))
	require.NoError(t, err)
	return int(header.length)
}

func TestABlobRoundTripsAtEverySizeAroundAChunk(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	aad := blobAAD("documents/a.pdf")
	for _, size := range []int{
		0, 1, BlobChunkSize - 1, BlobChunkSize, BlobChunkSize + 1, 3*BlobChunkSize + 5,
	} {
		plaintext := randomBytes(t, size)
		sealed := seal(t, svc, plaintext, aad)
		require.True(t, IsBlob(sealed))

		opened, err := svc.OpenBlob(t.Context(), bytes.NewReader(sealed), int64(len(sealed)), aad)
		require.NoError(t, err)
		assert.Equal(t, int64(size), opened.Size(), "the plaintext size follows from the sealed size")
		got, err := io.ReadAll(opened)
		require.NoError(t, err, "size %d", size)
		assert.Equal(t, plaintext, got, "size %d", size)
		assert.Equal(t, SealedBlobSize(int64(headerLength(t, sealed)), int64(size)), int64(len(sealed)))
	}
}

func TestTruncationAppendingAndReorderingFailAuthentication(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	aad := blobAAD("documents/b.pdf")
	plaintext := randomBytes(t, 3*BlobChunkSize+100)
	sealed := seal(t, svc, plaintext, aad)
	start := headerLength(t, sealed)
	chunk := BlobChunkSize + blobTagSize

	cases := map[string][]byte{
		"cut at a chunk boundary": sealed[:start+2*chunk],
		"cut inside a chunk":      sealed[:start+2*chunk+10],
		"last chunk dropped":      sealed[:start+3*chunk],
		"a chunk appended":        append(bytes.Clone(sealed), sealed[start:start+chunk]...),
		"two chunks swapped": func() []byte {
			out := bytes.Clone(sealed[:start])
			out = append(out, sealed[start+chunk:start+2*chunk]...)
			out = append(out, sealed[start:start+chunk]...)
			return append(out, sealed[start+2*chunk:]...)
		}(),
		"a byte flipped": func() []byte {
			out := bytes.Clone(sealed)
			out[start+chunk+7] ^= 0x01
			return out
		}(),
	}
	for name, tampered := range cases {
		got, err := open(t, svc, tampered, aad)
		require.ErrorIs(t, err, ErrBlobTampered, name)
		assert.Less(t, len(got), len(plaintext), name)
	}
}

func TestChunksAndKeysFromAnotherBlobDoNotOpen(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	aad := blobAAD("documents/c.pdf")
	first := seal(t, svc, randomBytes(t, 2*BlobChunkSize), aad)
	second := seal(t, svc, randomBytes(t, 2*BlobChunkSize), aad)
	start := headerLength(t, first)

	spliced := append(bytes.Clone(first[:start]), second[headerLength(t, second):]...)
	_, err := open(t, svc, spliced, aad)
	require.ErrorIs(t, err, ErrBlobTampered, "another blob's chunks under this blob's key")
}

func TestABlobOpensOnlyUnderItsOwnAssociatedData(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	sealed := seal(t, svc, []byte("rate confirmation"), blobAAD("documents/d.pdf"))

	_, err := open(t, svc, sealed, blobAAD("documents/elsewhere.pdf"))
	require.ErrorIs(t, err, ErrBlobAAD)

	forged := bytes.Clone(sealed)
	copy(forged[blobFixedSize-sha256.Size:], blobAADHash(blobAAD("documents/elsewhere.pdf")))
	_, err = open(t, svc, forged, blobAAD("documents/elsewhere.pdf"))
	require.Error(t, err, "rewriting the header's hash does not move the blob")
}

func TestADeclaredSizeTheSourceDoesNotMatchIsRefused(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	aad := blobAAD("documents/e.pdf")
	for name, size := range map[string]int64{"shorter": 10, "longer": 4} {
		sealed, err := svc.SealBlob(t.Context(), bytes.NewReader([]byte("12345678")), size, aad)
		require.NoError(t, err)
		_, err = io.ReadAll(sealed)
		require.ErrorIs(t, err, ErrBlobSize, name)
	}
}

func TestTheFinalChunkWaitsForTheSourcesEnd(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	refusal := io.ErrClosedPipe
	src := io.MultiReader(bytes.NewReader([]byte("whole file")), errReader{refusal})
	sealed, err := svc.SealBlob(t.Context(), src, 10, blobAAD("documents/f.pdf"))
	require.NoError(t, err)
	out, err := io.ReadAll(sealed)
	require.ErrorIs(t, err, refusal, "an error at the source's end stops the blob before its last chunk")
	assert.Less(t, int64(len(out)), sealed.Size())
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestABlobSealedUnderARetiredKeyStillOpensAndRewrapsWithoutReencrypting(t *testing.T) {
	t.Parallel()

	old := blobService(t, "blob-test-old-key-material")
	aad := blobAAD("documents/g.pdf")
	plaintext := randomBytes(t, 2*BlobChunkSize+3)
	sealed := seal(t, old, plaintext, aad)

	rotated := blobService(t, "blob-test-new-key-material", "blob-test-old-key-material")
	got, err := open(t, rotated, sealed, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)

	src := bytes.NewReader(sealed)
	rewrapped, changed, err := rotated.RewrapBlob(t.Context(), src, []AAD{blobAAD("other"), aad})
	require.NoError(t, err)
	require.True(t, changed)
	rest, err := io.ReadAll(src)
	require.NoError(t, err)
	assert.Equal(t, sealed[rewrapped.Consumed:], rest, "the chunks are copied as they are")

	moved := append(bytes.Clone(rewrapped.Header), rest...)
	newOnly := blobService(t, "blob-test-new-key-material")
	got, err = open(t, newOnly, moved, aad)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)

	_, changed, err = newOnly.RewrapBlob(t.Context(), bytes.NewReader(moved), []AAD{aad})
	require.NoError(t, err)
	assert.False(t, changed, "a blob under the active key is left alone")

	_, err = open(t, blobService(t, "blob-test-new-key-material"), sealed, aad)
	require.Error(t, err, "without the retired key the old wrap cannot be opened")
}

func TestOnlyTheBlobFormatIsRecognized(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	envelope, err := svc.EncryptBytesWithAAD([]byte("value"), blobAAD("x"))
	require.NoError(t, err)
	assert.False(t, IsBlob([]byte(envelope)))
	_, err = open(t, svc, []byte(envelope), blobAAD("x"))
	require.ErrorIs(t, err, ErrBlobFormat)
}

func TestOnlyAPermanentKeyFailureRejectsABlob(t *testing.T) {
	t.Parallel()

	aad := blobAAD("documents/kms.pdf")
	for code, rejected := range map[codes.Code]bool{
		codes.Unavailable:        false,
		codes.DeadlineExceeded:   false,
		codes.ResourceExhausted:  false,
		codes.Internal:           false,
		codes.InvalidArgument:    true,
		codes.NotFound:           true,
		codes.FailedPrecondition: true,
	} {
		client := &fakeKMSClient{}
		svc := NewWithKeyManager(NewGCPAutokeyManager(
			client,
			"projects/test/locations/us/keyRings/autokey/cryptoKeys/trenova",
			time.Second,
		))
		sealed := seal(t, svc, []byte("proof of delivery"), aad)
		client.decryptErr = status.Error(code, "kms said no")

		_, err := open(t, svc, sealed, aad)

		require.Error(t, err, code.String())
		assert.Equal(t, rejected, errors.Is(err, ErrBlobRejected), code.String())
		assert.Equal(t, code, status.Code(err), code.String())
	}
}

func TestABlobUnderAnUnknownKeyIsRejected(t *testing.T) {
	t.Parallel()

	aad := blobAAD("documents/unknown-key.pdf")
	sealed := seal(t, blobService(t, "blob-test-key-nobody-has-any-more"), []byte("bol"), aad)

	_, err := open(t, blobService(t, "blob-test-active-key-material"), sealed, aad)

	require.ErrorIs(t, err, ErrBlobRejected)
	require.ErrorIs(t, err, ErrUnknownKeyID)
}

func TestASourceEndingBeforeItsStoredSizeIsAReadError(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	aad := blobAAD("documents/truncated.pdf")
	sealed := seal(t, svc, randomBytes(t, 2*BlobChunkSize+10), aad)
	start := headerLength(t, sealed)

	for name, cut := range map[string]int{
		"inside the header":       start / 2,
		"at a chunk boundary":     start + BlobChunkSize + blobTagSize,
		"inside the middle chunk": start + BlobChunkSize + 100,
		"inside the last chunk":   len(sealed) - 3,
	} {
		_, err := func() ([]byte, error) {
			opened, err := svc.OpenBlob(
				t.Context(),
				bytes.NewReader(sealed[:cut]),
				int64(len(sealed)),
				aad,
			)
			if err != nil {
				return nil, err
			}
			return io.ReadAll(opened)
		}()

		require.ErrorIs(t, err, io.ErrUnexpectedEOF, name)
		require.ErrorIs(t, err, ErrBlobTruncated, name)
		assert.NotErrorIs(t, err, ErrBlobRejected, name)
	}
}

func TestAFlippedCiphertextByteIsTampered(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	aad := blobAAD("documents/flipped.pdf")
	sealed := seal(t, svc, randomBytes(t, BlobChunkSize+10), aad)
	sealed[headerLength(t, sealed)+3] ^= 0x80

	_, err := open(t, svc, sealed, aad)

	require.ErrorIs(t, err, ErrBlobTampered)
	assert.NotErrorIs(t, err, io.ErrUnexpectedEOF)
}

type cutOff struct {
	data []byte
	err  error
}

func (r *cutOff) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestADownloadCutShortWithUnexpectedEOFIsTruncated(t *testing.T) {
	t.Parallel()

	svc := blobService(t, "blob-test-active-key-material")
	aad := blobAAD("documents/cut-off.pdf")
	sealed := seal(t, svc, randomBytes(t, 2*BlobChunkSize+10), aad)
	start := headerLength(t, sealed)

	for name, cut := range map[string]int{
		"inside the header":      start / 2,
		"inside the first chunk": start + 100,
		"inside the last chunk":  len(sealed) - 3,
	} {
		_, err := func() ([]byte, error) {
			opened, err := svc.OpenBlob(
				t.Context(),
				&cutOff{data: sealed[:cut], err: io.ErrUnexpectedEOF},
				int64(len(sealed)),
				aad,
			)
			if err != nil {
				return nil, err
			}
			return io.ReadAll(opened)
		}()

		require.ErrorIs(t, err, ErrBlobTruncated, name)
		assert.NotErrorIs(t, err, ErrBlobRejected, name)
	}
}
