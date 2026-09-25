package jsonutils

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObjectKeyOrder_ReadsTheDeclaredOrder(t *testing.T) {
	t.Parallel()

	type record struct {
		Zeta  string `json:"zeta"`
		Alpha int    `json:"alpha"`
		Mid   bool   `json:"mid"`
	}
	encoded, err := sonic.Marshal(record{})
	require.NoError(t, err)

	assert.Equal(t, []string{"zeta", "alpha", "mid"}, ObjectKeyOrder(encoded))
	assert.Nil(t, ObjectKeyOrder([]byte("not json")))
}
