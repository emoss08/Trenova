package document

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAShipmentDocumentUploadedWithNoProfileIsReadAsPaperwork(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		ProcessingProfileShipmentPaperwork,
		ProcessingProfileFor(ProcessingProfileNone, "shipment"),
	)
	assert.Equal(t, ProcessingProfileNone, ProcessingProfileFor(ProcessingProfileNone, "customer"))
	assert.Equal(
		t,
		ProcessingProfileRateConfirmationImport,
		ProcessingProfileFor(ProcessingProfileRateConfirmationImport, "shipment"),
	)
	assert.True(t, ProcessingProfileShipmentPaperwork.SupportsIntelligence())
	assert.True(t, ProcessingProfileShipmentPaperwork.IsValid())
}
