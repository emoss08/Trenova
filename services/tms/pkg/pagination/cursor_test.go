package pagination

import (
	"encoding/base64"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	expected := Cursor{
		CreatedAt: 1710000000,
		ID:        pulid.MustNew("tr_"),
	}

	encoded, err := EncodeCursor(expected)
	require.NoError(t, err)

	actual, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestDecodeCursor_MalformedBase64(t *testing.T) {
	t.Parallel()

	_, err := DecodeCursor("not base64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode cursor")
}

func TestDecodeCursor_MalformedJSON(t *testing.T) {
	t.Parallel()

	encoded := base64.RawURLEncoding.EncodeToString([]byte("not-json"))

	_, err := DecodeCursor(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal cursor")
}

func TestDecodeCursor_MissingID(t *testing.T) {
	t.Parallel()

	bytes, err := sonic.Marshal(map[string]any{"createdAt": int64(1710000000)})
	require.NoError(t, err)
	encoded := base64.RawURLEncoding.EncodeToString(bytes)

	_, err = DecodeCursor(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor id is required")
}

func TestDecodeCursor_EmptyID(t *testing.T) {
	t.Parallel()

	bytes, err := sonic.Marshal(map[string]any{
		"createdAt": int64(1710000000),
		"id":        "",
	})
	require.NoError(t, err)
	encoded := base64.RawURLEncoding.EncodeToString(bytes)

	_, err = DecodeCursor(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor id is required")
}

func TestCursorCodec_RoundTripCatalogID(t *testing.T) {
	t.Parallel()

	expected := Cursor{ID: pulid.ID("edidt_x12_204_outbound")}

	encoded, err := EncodeCursor(expected)
	require.NoError(t, err)

	actual, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestEncodeCursor_EmptyID(t *testing.T) {
	t.Parallel()

	_, err := EncodeCursor(Cursor{CreatedAt: 1710000000})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor id is required")
}

func TestEncodeCursor_OutputShape(t *testing.T) {
	t.Parallel()

	cursor := Cursor{
		CreatedAt: 1710000000,
		ID:        pulid.ID("tr_01ARZ3NDEKTSV4RRFFQ69G5FAV"),
	}

	encoded, err := EncodeCursor(cursor)
	require.NoError(t, err)

	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.JSONEq(
		t,
		`{"createdAt":1710000000,"id":"tr_01ARZ3NDEKTSV4RRFFQ69G5FAV"}`,
		string(bytes),
	)
}

func TestEncodeCursorFromEntityWithValues_UsesExplicitValues(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("item_")
	item := cursorTestItem{
		ID:        id,
		CreatedAt: 1710000000,
		Name:      "hydrated-name",
	}

	encoded, err := EncodeCursorFromEntityWithValues(
		item,
		[]CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		[]any{"sql-name", id.String()},
	)
	require.NoError(t, err)

	cursor, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, id, cursor.ID)
	assert.Equal(t, []any{"sql-name", id.String()}, cursor.Values)
}

func TestEncodeCursorFromEntityWithValues_AcceptsCatalogIDValue(t *testing.T) {
	t.Parallel()

	id := pulid.ID("edidt_x12_204_outbound")

	encoded, err := EncodeCursorFromEntityWithValues(
		cursorTestItem{ID: id, CreatedAt: 1710000000},
		[]CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		[]any{"sql-name", id.String()},
	)
	require.NoError(t, err)

	cursor, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, id, cursor.ID)
}

func TestEncodeCursorFromEntityWithValues_RejectsEmptyIDValue(t *testing.T) {
	t.Parallel()

	_, err := EncodeCursorFromEntityWithValues(
		cursorTestItem{ID: pulid.MustNew("item_"), CreatedAt: 1710000000},
		[]CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		[]any{"sql-name", ""},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor id is required")
}

func TestEncodeCursorFromEntityWithValues_RejectsValueCountMismatch(t *testing.T) {
	t.Parallel()

	_, err := EncodeCursorFromEntityWithValues(
		cursorTestItem{ID: pulid.MustNew("item_"), CreatedAt: 1710000000},
		[]CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		},
		[]any{"sql-name"},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor sort values do not match cursor sort shape")
}

func TestCursorInt64Value(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  int64
		ok    bool
	}{
		{
			name:  "int64 from a database scan",
			value: int64(1_700_000_000),
			want:  1_700_000_000,
			ok:    true,
		},
		{name: "int", value: 42, want: 42, ok: true},
		{name: "int32", value: int32(42), want: 42, ok: true},
		{
			name:  "float64 from a decoded cursor",
			value: float64(1_700_000_000),
			want:  1_700_000_000,
			ok:    true,
		},
		{name: "numeric string", value: "1700000000", want: 1_700_000_000, ok: true},
		{name: "non-numeric string", value: "yesterday", ok: false},
		{name: "nil", value: nil, ok: false},
		{name: "bool", value: true, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := CursorInt64Value(tt.value)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCursorInt64Value_SurvivesTheCursorCodec(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("tr_")
	sort := []CursorSortField{
		{Field: "sortAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}
	encoded, err := EncodeCursor(Cursor{
		ID:     id,
		Sort:   sort,
		Values: []any{int64(1_700_000_123), id.String()},
	})
	require.NoError(t, err)

	decoded, err := DecodeCursor(encoded)
	require.NoError(t, err)

	got, ok := CursorInt64Value(decoded.Values[0])
	require.True(t, ok)
	assert.Equal(t, int64(1_700_000_123), got)
}

func TestCursorListResult_NextCursor(t *testing.T) {
	t.Parallel()

	first := cursorTestItem{ID: pulid.MustNew("item_"), CreatedAt: 1710000000, Name: "a"}
	last := cursorTestItem{ID: pulid.MustNew("item_"), CreatedAt: 1710000100, Name: "b"}

	t.Run("no next page", func(t *testing.T) {
		t.Parallel()

		next, err := (&CursorListResult[cursorTestItem]{
			Items: []cursorTestItem{first, last},
		}).NextCursor()
		require.NoError(t, err)
		assert.Empty(t, next)
	})

	t.Run("created order", func(t *testing.T) {
		t.Parallel()

		next, err := (&CursorListResult[cursorTestItem]{
			Items:       []cursorTestItem{first, last},
			HasNextPage: true,
		}).NextCursor()
		require.NoError(t, err)

		cursor, err := DecodeCursor(next)
		require.NoError(t, err)
		assert.Equal(t, last.ID, cursor.ID)
		assert.Equal(t, last.CreatedAt, cursor.CreatedAt)
	})

	t.Run("sorted with scanned values", func(t *testing.T) {
		t.Parallel()

		sort := []CursorSortField{
			{Field: "name", Direction: "asc"},
			{Field: "id", Direction: "asc"},
		}
		result := (&CursorListResult[cursorTestItem]{
			Items:       []cursorTestItem{first, last},
			HasNextPage: true,
		}).WithCursorSort(sort)
		require.NoError(t, result.WithCursorValues([][]any{
			{"a", first.ID.String()},
			{"b", last.ID.String()},
		}))

		next, err := result.NextCursor()
		require.NoError(t, err)

		cursor, err := DecodeCursor(next)
		require.NoError(t, err)
		assert.Equal(t, last.ID, cursor.ID)
		assert.Equal(t, []any{"b", last.ID.String()}, cursor.Values)
	})
}
