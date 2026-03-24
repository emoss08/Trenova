package pulid_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMustNew(t *testing.T) {
	t.Parallel()

	t.Run("generates ID with prefix", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")

		assert.NotEmpty(t, string(id))
		assert.Contains(t, string(id), "usr_")
		assert.True(t, len(string(id)) > 26)
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		t.Parallel()
		id1 := pulid.MustNew("usr_")
		id2 := pulid.MustNew("usr_")

		assert.NotEqual(t, id1, id2)
	})

	t.Run("works with empty prefix", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("")

		assert.NotEmpty(t, string(id))
		assert.Equal(t, 26, len(string(id)))
	})

	t.Run("preserves prefix", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("org_")
		assert.Equal(t, "org_", id.Prefix())
	})
}

func TestMust(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil pointer with valid ID", func(t *testing.T) {
		t.Parallel()
		ptr := pulid.Must("bu_")

		require.NotNil(t, ptr)
		assert.Contains(t, string(*ptr), "bu_")
	})
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("parses valid ID", func(t *testing.T) {
		t.Parallel()
		original := pulid.MustNew("usr_")
		parsed, err := pulid.Parse(string(original))

		require.NoError(t, err)
		assert.Equal(t, original, parsed)
	})

	t.Run("returns error for short string", func(t *testing.T) {
		t.Parallel()
		_, err := pulid.Parse("short")

		assert.ErrorIs(t, err, pulid.ErrInvalidLength)
	})

	t.Run("returns error for empty string", func(t *testing.T) {
		t.Parallel()
		_, err := pulid.Parse("")

		assert.ErrorIs(t, err, pulid.ErrInvalidLength)
	})
}

func TestMustParse(t *testing.T) {
	t.Parallel()

	t.Run("parses valid ID", func(t *testing.T) {
		t.Parallel()
		original := pulid.MustNew("org_")
		parsed, err := pulid.MustParse(string(original))

		require.NoError(t, err)
		assert.Equal(t, original, parsed)
	})

	t.Run("returns error for invalid ID", func(t *testing.T) {
		t.Parallel()
		_, err := pulid.MustParse("bad")

		assert.Error(t, err)
		assert.ErrorIs(t, err, pulid.ErrInvalidLength)
	})
}

func TestIsNil(t *testing.T) {
	t.Parallel()

	t.Run("nil for empty ID", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		assert.True(t, id.IsNil())
	})

	t.Run("nil for Nil constant", func(t *testing.T) {
		t.Parallel()
		assert.True(t, pulid.Nil.IsNil())
	})

	t.Run("not nil for valid ID", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		assert.False(t, id.IsNil())
	})

	t.Run("IsNotNil for valid ID", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		assert.True(t, id.IsNotNil())
	})

	t.Run("IsNotNil false for nil ID", func(t *testing.T) {
		t.Parallel()
		assert.False(t, pulid.Nil.IsNotNil())
	})
}

func TestConvertFromPtr(t *testing.T) {
	t.Parallel()

	t.Run("converts non-nil pointer", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		result := pulid.ConvertFromPtr(&id)

		assert.Equal(t, id, result)
	})

	t.Run("returns Nil for nil pointer", func(t *testing.T) {
		t.Parallel()
		result := pulid.ConvertFromPtr(nil)

		assert.Equal(t, pulid.Nil, result)
	})
}

func TestEquals(t *testing.T) {
	t.Parallel()

	t.Run("equal IDs", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		assert.True(t, pulid.Equals(id, id))
	})

	t.Run("different IDs", func(t *testing.T) {
		t.Parallel()
		id1 := pulid.MustNew("usr_")
		id2 := pulid.MustNew("usr_")
		assert.False(t, pulid.Equals(id1, id2))
	})
}

func TestID_String(t *testing.T) {
	t.Parallel()

	t.Run("returns string representation", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		assert.Equal(t, string(id), id.String())
	})
}

func TestID_Prefix(t *testing.T) {
	t.Parallel()

	t.Run("returns prefix", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		assert.Equal(t, "usr_", id.Prefix())
	})

	t.Run("returns empty for nil ID", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", pulid.Nil.Prefix())
	})

	t.Run("returns empty for ID with no prefix", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("")
		assert.Equal(t, "", id.Prefix())
	})
}

func TestID_Time(t *testing.T) {
	t.Parallel()

	t.Run("returns time for valid ID", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		ts, err := id.Time()

		require.NoError(t, err)
		assert.False(t, ts.IsZero())
	})

	t.Run("returns error for nil ID", func(t *testing.T) {
		t.Parallel()
		_, err := pulid.Nil.Time()

		assert.Error(t, err)
	})
}

func TestID_Scan(t *testing.T) {
	t.Parallel()

	t.Run("scans string value", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.Scan("usr_01ABCDEFGHJKMNPQRSTVWXYZ")

		require.NoError(t, err)
		assert.Equal(t, pulid.ID("usr_01ABCDEFGHJKMNPQRSTVWXYZ"), id)
	})

	t.Run("scans byte slice", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.Scan([]byte("usr_01ABCDEFGHJKMNPQRSTVWXYZ"))

		require.NoError(t, err)
		assert.Equal(t, pulid.ID("usr_01ABCDEFGHJKMNPQRSTVWXYZ"), id)
	})

	t.Run("scans nil to Nil", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.Scan(nil)

		require.NoError(t, err)
		assert.Equal(t, pulid.Nil, id)
	})

	t.Run("scans ID value", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		src := pulid.MustNew("usr_")
		err := id.Scan(src)

		require.NoError(t, err)
		assert.Equal(t, src, id)
	})

	t.Run("returns error for unexpected type", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.Scan(12345)

		assert.Error(t, err)
	})
}

func TestID_Value(t *testing.T) {
	t.Parallel()

	t.Run("returns string for valid ID", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		val, err := id.Value()

		require.NoError(t, err)
		assert.Equal(t, string(id), val)
	})

	t.Run("returns nil for Nil ID", func(t *testing.T) {
		t.Parallel()
		val, err := pulid.Nil.Value()

		require.NoError(t, err)
		assert.Nil(t, val)
	})
}

func TestID_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("marshals valid ID to JSON string", func(t *testing.T) {
		t.Parallel()
		id := pulid.MustNew("usr_")
		data, err := id.MarshalJSON()

		require.NoError(t, err)
		assert.Contains(t, string(data), "usr_")
	})

	t.Run("marshals nil ID to null", func(t *testing.T) {
		t.Parallel()
		data, err := pulid.Nil.MarshalJSON()

		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})
}

func TestID_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("unmarshals valid JSON string", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.UnmarshalJSON([]byte(`"usr_01ABCDEFGHJKMNPQRSTVWXYZ"`))

		require.NoError(t, err)
		assert.Equal(t, pulid.ID("usr_01ABCDEFGHJKMNPQRSTVWXYZ"), id)
	})

	t.Run("unmarshals null to Nil", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.UnmarshalJSON([]byte("null"))

		require.NoError(t, err)
		assert.Equal(t, pulid.Nil, id)
	})

	t.Run("unmarshals empty string to Nil", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.UnmarshalJSON([]byte(`""`))

		require.NoError(t, err)
		assert.Equal(t, pulid.Nil, id)
	})

	t.Run("unmarshals empty bytes to Nil", func(t *testing.T) {
		t.Parallel()
		var id pulid.ID
		err := id.UnmarshalJSON([]byte{})

		require.NoError(t, err)
		assert.Equal(t, pulid.Nil, id)
	})
}

func TestMap(t *testing.T) {
	t.Parallel()

	t.Run("maps IDs to strings", func(t *testing.T) {
		t.Parallel()
		ids := []pulid.ID{
			pulid.MustNew("usr_"),
			pulid.MustNew("usr_"),
		}

		result := pulid.Map(ids, func(id pulid.ID) string {
			return id.String()
		})

		assert.Len(t, result, 2)
		assert.Equal(t, ids[0].String(), result[0])
		assert.Equal(t, ids[1].String(), result[1])
	})

	t.Run("maps empty slice", func(t *testing.T) {
		t.Parallel()
		var ids []pulid.ID

		result := pulid.Map(ids, func(id pulid.ID) string {
			return id.String()
		})

		assert.Empty(t, result)
	})
}
