package api_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/aiusagemeter"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentmeter"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/organizationmeter"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentmeter"
	"github.com/stretchr/testify/assert"
)

func TestTheInstanceReportsExactlyFourAppMeters(t *testing.T) {
	t.Parallel()

	counted := []platformcatalog.MeterKey{
		shipmentmeter.New(shipmentmeter.Params{}).Counter.Meter(),
		aiusagemeter.New(aiusagemeter.Params{}).Pages.Meter(),
	}
	levels := []platformcatalog.MeterKey{
		documentmeter.New(documentmeter.Params{}).Reader.Meter(),
		organizationmeter.New(organizationmeter.Params{}).Reader.Meter(),
	}

	assert.Equal(t, []platformcatalog.MeterKey{
		platformcatalog.MeterShipmentsTotal,
		platformcatalog.MeterDocumentIntelligencePages,
	}, counted)
	assert.Equal(t, []platformcatalog.MeterKey{
		platformcatalog.MeterDocumentStorageBytes,
		platformcatalog.MeterOrganizationsTotal,
	}, levels)
}
