package artifactpayload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordEntityOf_FindsTheRegistrysName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "shipment", RecordEntityOf("shipments"))
	assert.Equal(t, "worker", RecordEntityOf("worker"))
	assert.Equal(t, "rate_matrix", RecordEntityOf("rate_matrices"))
	assert.Empty(t, RecordEntityOf("insights"))
	assert.Empty(t, RecordEntityOf(""))
}
