package jsonutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToJSONString(t *testing.T) {
	t.Parallel()

	t.Run("marshals struct to JSON string", func(t *testing.T) {
		t.Parallel()
		input := struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{Name: "Alice", Age: 30}

		result, err := jsonutils.ToJSONString(input)

		require.NoError(t, err)
		assert.Contains(t, result, `"name":"Alice"`)
		assert.Contains(t, result, `"age":30`)
	})

	t.Run("marshals map to JSON string", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"key": "value", "num": 42}

		result, err := jsonutils.ToJSONString(input)

		require.NoError(t, err)
		assert.Contains(t, result, `"key":"value"`)
		assert.Contains(t, result, `"num":42`)
	})

	t.Run("marshals nil to null", func(t *testing.T) {
		t.Parallel()
		result, err := jsonutils.ToJSONString(nil)

		require.NoError(t, err)
		assert.Equal(t, "null", result)
	})

	t.Run("marshals slice", func(t *testing.T) {
		t.Parallel()
		input := []int{1, 2, 3}

		result, err := jsonutils.ToJSONString(input)

		require.NoError(t, err)
		assert.Equal(t, "[1,2,3]", result)
	})

	t.Run("returns error for unmarshalable value", func(t *testing.T) {
		t.Parallel()
		input := make(chan int)

		_, err := jsonutils.ToJSONString(input)

		assert.Error(t, err)
	})
}

func TestToJSON(t *testing.T) {
	t.Parallel()

	t.Run("converts struct to map", func(t *testing.T) {
		t.Parallel()
		input := struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{Name: "Bob", Age: 25}

		result, err := jsonutils.ToJSON(input)

		require.NoError(t, err)
		assert.Equal(t, "Bob", result["name"])
		assert.Equal(t, float64(25), result["age"])
	})

	t.Run("converts map to map", func(t *testing.T) {
		t.Parallel()
		input := map[string]any{"foo": "bar"}

		result, err := jsonutils.ToJSON(input)

		require.NoError(t, err)
		assert.Equal(t, "bar", result["foo"])
	})

	t.Run("returns error for unmarshalable value", func(t *testing.T) {
		t.Parallel()
		input := make(chan int)

		_, err := jsonutils.ToJSON(input)

		assert.Error(t, err)
	})
}

func TestMustToJSONString(t *testing.T) {
	t.Parallel()

	t.Run("returns JSON string for valid input", func(t *testing.T) {
		t.Parallel()
		input := map[string]string{"hello": "world"}

		result := jsonutils.MustToJSONString(input)

		assert.Equal(t, `{"hello":"world"}`, result)
	})

	t.Run("panics for unmarshalable value", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			jsonutils.MustToJSONString(make(chan int))
		})
	})
}

func TestMustToJSON(t *testing.T) {
	t.Parallel()

	t.Run("returns map for valid input", func(t *testing.T) {
		t.Parallel()
		input := struct {
			Value string `json:"value"`
		}{Value: "test"}

		result := jsonutils.MustToJSON(input)

		assert.Equal(t, "test", result["value"])
	})

	t.Run("panics for unmarshalable value", func(t *testing.T) {
		t.Parallel()
		assert.Panics(t, func() {
			jsonutils.MustToJSON(make(chan int))
		})
	})
}
