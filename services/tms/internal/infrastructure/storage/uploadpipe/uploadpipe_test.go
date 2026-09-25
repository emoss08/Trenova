package uploadpipe

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStorage struct {
	storage.Client

	key      string
	body     bytes.Buffer
	failWith error
}

func (f *fakeStorage) Upload(
	_ context.Context,
	params *storage.UploadParams,
) (*storage.FileInfo, error) {
	f.key = params.Key
	if f.failWith != nil {
		_, _ = io.Copy(io.Discard, params.Body)

		return nil, f.failWith
	}
	n, err := io.Copy(&f.body, params.Body)
	if err != nil {
		return nil, err
	}

	return &storage.FileInfo{Key: params.Key, Size: n}, nil
}

func TestPipe_StreamsWhatIsWrittenIntoStorage(t *testing.T) {
	t.Parallel()

	store := &fakeStorage{}
	pipe := Open(t.Context(), store, Params{Key: "exports/a.csv", ContentType: "text/csv"})

	for range 3 {
		_, err := pipe.Writer().Write([]byte("row\n"))
		require.NoError(t, err)
	}

	size, err := pipe.Close()
	require.NoError(t, err)
	assert.Equal(t, int64(12), size)
	assert.Equal(t, "exports/a.csv", store.key)
	assert.Equal(t, "row\nrow\nrow\n", store.body.String())
}

func TestPipe_RefusesAWriteThatPassesTheLimit(t *testing.T) {
	t.Parallel()

	store := &fakeStorage{}
	pipe := Open(t.Context(), store, Params{Key: "k", MaxBytes: 5})

	_, err := pipe.Writer().Write([]byte("1234"))
	require.NoError(t, err)
	_, err = pipe.Writer().Write([]byte("56"))
	require.ErrorIs(t, err, ErrTooLarge)

	pipe.Abort(err)
}

func TestPipe_ReportsTheUploadFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("bucket gone")
	pipe := Open(t.Context(), &fakeStorage{failWith: failure}, Params{Key: "k"})
	_, _ = pipe.Writer().Write([]byte("x"))

	_, err := pipe.Close()
	require.ErrorIs(t, err, failure)
}

func TestRowHeartbeat_BeatsEverySoManyRows(t *testing.T) {
	t.Parallel()

	var beats []int64
	heartbeat := NewRowHeartbeat(t.Context(), 3, func(_ context.Context, details ...any) {
		beats = append(beats, details[0].(int64))
	})
	for range 7 {
		heartbeat.Tick()
	}

	assert.Equal(t, []int64{3, 6}, beats)
	assert.Equal(t, int64(7), heartbeat.Rows())
}

func TestRowHeartbeat_WithoutAHeartbeatOnlyCounts(t *testing.T) {
	t.Parallel()

	heartbeat := NewRowHeartbeat(t.Context(), 0, nil)
	heartbeat.Tick()
	heartbeat.Tick()

	assert.Equal(t, int64(2), heartbeat.Rows())
}
