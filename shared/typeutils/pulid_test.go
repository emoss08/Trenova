package typeutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDerefID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("worker_")

	assert.Equal(t, id, typeutils.DerefID(&id))
	assert.True(t, typeutils.DerefID(nil).IsNil(), "an absent pointer is no id")
	assert.True(t, typeutils.DerefID(&pulid.Nil).IsNil(), "a pointer to the nil id is no id")
}

func TestIDPtr(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shipment_")

	ptr := typeutils.IDPtr(id)
	require.NotNil(t, ptr)
	assert.Equal(t, id, *ptr)

	assert.Nil(t, typeutils.IDPtr(pulid.Nil), "the nil id stays absent")
}

func TestIDString(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("invoice_")

	assert.Equal(t, id.String(), *typeutils.IDString(id))
	assert.Nil(t, typeutils.IDString(pulid.Nil), "a nil id is written as NULL, not an empty string")
}

func TestIDStringOrEmpty(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("invoice_")

	assert.Equal(t, id.String(), typeutils.IDStringOrEmpty(&id))
	assert.Empty(t, typeutils.IDStringOrEmpty(nil))
	assert.Empty(t, typeutils.IDStringOrEmpty(&pulid.Nil))
}
