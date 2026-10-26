package shapetest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/keel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserversRecordNothingWithoutTheVariable(t *testing.T) {
	t.Setenv(Env, "")

	assert.Empty(t, Path())
	assert.Nil(t, Observers(t))
}

func TestPathTrimsTheVariable(t *testing.T) {
	t.Setenv(Env, "  /tmp/keel-shapes.jsonl \n")

	assert.Equal(t, "/tmp/keel-shapes.jsonl", Path())
}

func TestRecorderAppendsEachPlannableTextOnce(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "shapes.jsonl")
	require.NoError(
		t,
		os.WriteFile(
			path,
			[]byte(`{"sql":"SELECT 0","op":"select","table":"","args":0}`+"\n"),
			0o600,
		),
	)

	recorder, out, err := openRecorder(path)
	require.NoError(t, err)

	events := []*keel.QueryEvent{
		{
			Op:    "select",
			Table: "fleet_codes",
			SQL:   "SELECT id FROM fleet_codes WHERE code = $1",
			Args:  1,
		},
		{
			Op:    "select",
			Table: "fleet_codes",
			SQL:   "SELECT id FROM fleet_codes WHERE code = $1",
			Args:  1,
		},
		{Op: "exec", SQL: "SAVEPOINT keel_1"},
	}
	for _, e := range events {
		recorder.AfterQuery(recorder.BeforeQuery(t.Context(), e), e)
	}
	require.NoError(t, out.Err())

	file, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })

	shapes, err := keel.ReadShapes(file)
	require.NoError(t, err)
	require.Len(t, shapes, 2, "the earlier binary's line is kept and the new text appended once")
	assert.Equal(t, "SELECT 0", shapes[0].SQL)
	assert.Equal(t, keel.Shape{
		SQL:   "SELECT id FROM fleet_codes WHERE code = $1",
		Op:    "select",
		Table: "fleet_codes",
		Args:  1,
	}, shapes[1])
}

func TestOpenRecorderReportsAnUnwritablePath(t *testing.T) {
	t.Parallel()

	_, _, err := openRecorder(filepath.Join(t.TempDir(), "missing", "shapes.jsonl"))
	require.Error(t, err)
}
