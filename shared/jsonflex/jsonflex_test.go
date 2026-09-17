package jsonflex_test

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKindOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want jsonflex.Kind
	}{
		{raw: `  "x"`, want: jsonflex.KindString},
		{raw: `12`, want: jsonflex.KindNumber},
		{raw: `-1.5`, want: jsonflex.KindNumber},
		{raw: `true`, want: jsonflex.KindBool},
		{raw: `null`, want: jsonflex.KindNull},
		{raw: `[1]`, want: jsonflex.KindArray},
		{raw: `{"a":1}`, want: jsonflex.KindObject},
		{raw: ``, want: jsonflex.KindInvalid},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, jsonflex.KindOf([]byte(tt.raw)), tt.raw)
	}
}

func TestParseString(t *testing.T) {
	t.Parallel()

	value, ok := jsonflex.ParseString([]byte(`" ACME "`))
	require.True(t, ok)
	assert.Equal(t, "ACME", value.Value())

	value, ok = jsonflex.ParseString([]byte(`818175`))
	require.True(t, ok)
	assert.Equal(t, "818175", value.Value())

	_, ok = jsonflex.ParseString([]byte(`""`))
	assert.False(t, ok)

	_, ok = jsonflex.ParseString([]byte(`null`))
	assert.False(t, ok)

	_, ok = jsonflex.ParseString([]byte(`{"a":1}`))
	assert.False(t, ok)

	var nilValue *jsonflex.String
	assert.Empty(t, nilValue.Value())
	assert.Nil(t, nilValue.Ptr())
}

func TestParseInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want int64
		ok   bool
	}{
		{raw: `750000`, want: 750000, ok: true},
		{raw: `"750000"`, want: 750000, ok: true},
		{raw: `"750,000"`, want: 750000, ok: true},
		{raw: `"$1,000"`, want: 1000, ok: true},
		{raw: `12.9`, want: 12, ok: true},
		{raw: `""`, ok: false},
		{raw: `"abc"`, ok: false},
		{raw: `true`, ok: false},
		{raw: `null`, ok: false},
	}

	for _, tt := range tests {
		value, ok := jsonflex.ParseInt([]byte(tt.raw))
		assert.Equal(t, tt.ok, ok, tt.raw)
		if tt.ok {
			assert.Equal(t, tt.want, value.Value(), tt.raw)
		}
	}
}

func TestParseFloat(t *testing.T) {
	t.Parallel()

	value, ok := jsonflex.ParseFloat([]byte(`"0.8712"`))
	require.True(t, ok)
	assert.InDelta(t, 0.8712, value.Value(), 1e-9)

	value, ok = jsonflex.ParseFloat([]byte(`"12.5%"`))
	require.True(t, ok)
	assert.InDelta(t, 12.5, value.Value(), 1e-9)

	_, ok = jsonflex.ParseFloat([]byte(`"Not Public"`))
	assert.False(t, ok)
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want bool
		ok   bool
	}{
		{raw: `"Y"`, want: true, ok: true},
		{raw: `"n"`, want: false, ok: true},
		{raw: `"true"`, want: true, ok: true},
		{raw: `false`, want: false, ok: true},
		{raw: `1`, want: true, ok: true},
		{raw: `0`, want: false, ok: true},
		{raw: `"-1"`, ok: false},
		{raw: `2`, ok: false},
		{raw: `"Active"`, ok: false},
	}

	for _, tt := range tests {
		value, ok := jsonflex.ParseBool([]byte(tt.raw))
		assert.Equal(t, tt.ok, ok, tt.raw)
		if tt.ok {
			assert.Equal(t, tt.want, value.Value(), tt.raw)
		}
	}
}

func TestParseTime(t *testing.T) {
	t.Parallel()

	expected := time.Date(2024, time.March, 15, 0, 0, 0, 0, time.UTC).Unix()

	tests := []string{
		`1710460800`,
		`"1710460800"`,
		`1710460800000`,
		`"2024-03-15"`,
		`"20240315"`,
		`"2024-03-15T00:00:00Z"`,
		`"03/15/2024"`,
		`"2024-03-15T00:00:00.000+0000"`,
		`"2024-03-14T19:00:00-0500"`,
	}

	for _, raw := range tests {
		value, ok := jsonflex.ParseTime([]byte(raw))
		require.True(t, ok, raw)
		require.NotNil(t, value.Unix(), raw)
		assert.Equal(t, expected, *value.Unix(), raw)
	}

	_, ok := jsonflex.ParseTime([]byte(`"not a date"`))
	assert.False(t, ok)

	_, ok = jsonflex.ParseTime([]byte(`0`))
	assert.False(t, ok)

	var nilValue *jsonflex.Time
	assert.Nil(t, nilValue.Unix())
}

func TestStructDecoding(t *testing.T) {
	t.Parallel()

	var target struct {
		Amount  *jsonflex.Int    `json:"amount"`
		Rate    *jsonflex.Float  `json:"rate"`
		Flag    *jsonflex.Bool   `json:"flag"`
		Name    *jsonflex.String `json:"name"`
		When    *jsonflex.Time   `json:"when"`
		Missing *jsonflex.Int    `json:"missing"`
	}

	err := sonic.Unmarshal(
		[]byte(`{"amount":"750000","rate":0.25,"flag":"Y","name":123,"when":"20240315"}`),
		&target,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(750000), target.Amount.Value())
	assert.InDelta(t, 0.25, target.Rate.Value(), 1e-9)
	assert.True(t, target.Flag.Value())
	assert.Equal(t, "123", target.Name.Value())
	assert.Equal(t, 2024, target.When.Time().Year())
	assert.Nil(t, target.Missing)

	encoded, err := sonic.Marshal(target)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"amount":750000`)
	assert.Contains(t, string(encoded), `"when":"2024-03-15T00:00:00Z"`)

	err = sonic.Unmarshal([]byte(`{"flag":"maybe"}`), &target)
	require.Error(t, err)
}

func TestObjectAccessors(t *testing.T) {
	t.Parallel()

	obj, err := jsonflex.DecodeObject([]byte(`{
		"primary": null,
		"secondary": "fallback",
		"count": "42",
		"cargo": ["General Freight", "", "Household Goods"],
		"cargo_text": "Metal, Lumber",
		"nested": {"inner": "value"},
		"list": [1, 2]
	}`))
	require.NoError(t, err)

	assert.True(t, obj.Has("primary"))
	assert.Equal(t, "fallback", obj.Text("primary", "secondary"))
	assert.Equal(t, int64(42), obj.Int("count").Value())
	assert.Nil(t, obj.Int("absent"))
	assert.Equal(t, []string{"General Freight", "Household Goods"}, obj.Strings("cargo"))
	assert.Equal(t, []string{"Metal", "Lumber"}, obj.Strings("cargo_text"))
	assert.Nil(t, obj.Raw("primary"))
	assert.JSONEq(t, `{"inner":"value"}`, string(obj.Raw("nested")))

	nested, ok := obj.Object("nested")
	require.True(t, ok)
	assert.Equal(t, "value", nested.Text("inner"))

	items, ok := obj.Array("list")
	require.True(t, ok)
	assert.Len(t, items, 2)

	assert.Equal(
		t,
		[]string{"cargo", "cargo_text", "count", "list", "nested", "primary", "secondary"},
		obj.Keys(),
	)

	_, err = jsonflex.DecodeObject([]byte(`[1]`))
	require.ErrorIs(t, err, jsonflex.ErrNotObject)

	_, err = jsonflex.DecodeArray([]byte(`{}`))
	require.ErrorIs(t, err, jsonflex.ErrNotArray)
}

func TestUnwrap(t *testing.T) {
	t.Parallel()

	assert.JSONEq(
		t,
		`{"dotNumber":1}`,
		string(jsonflex.Unwrap([]byte(`{"carrier":{"dotNumber":1}}`), "carrier")),
	)
	assert.JSONEq(
		t,
		`{"dotNumber":1}`,
		string(jsonflex.Unwrap([]byte(`{"dotNumber":1}`), "carrier")),
	)
	assert.JSONEq(
		t,
		`{"carrier":"text"}`,
		string(jsonflex.Unwrap([]byte(`{"carrier":"text"}`), "carrier")),
	)
}
