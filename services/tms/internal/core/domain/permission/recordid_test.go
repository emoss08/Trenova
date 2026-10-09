package permission_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/order"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/recurringshipment"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

type minter func(context.Context) (pulid.ID, error)

// mint inserts nothing: it runs the entity's own insert hook, which is where
// the domain gives a new record its id, and reads the id back.
func mint[T any, P interface {
	*T
	bun.BeforeAppendModelHook
}](id func(P) pulid.ID) minter {
	return func(ctx context.Context) (pulid.ID, error) {
		entity := P(new(T))
		err := entity.BeforeAppendModel(ctx, &bun.InsertQuery{})

		return id(entity), err
	}
}

var minters = map[permission.Resource]minter{
	permission.ResourceShipment: mint(func(e *shipment.Shipment) pulid.ID { return e.ID }),
	permission.ResourceShipmentMove: mint(
		func(e *shipment.ShipmentMove) pulid.ID { return e.ID },
	),
	permission.ResourceShipmentComment: mint(
		func(e *shipment.ShipmentComment) pulid.ID { return e.ID },
	),
	permission.ResourceOrder: mint(func(e *order.Order) pulid.ID { return e.ID }),
	permission.ResourceRecurringShipment: mint(
		func(e *recurringshipment.RecurringShipment) pulid.ID { return e.ID },
	),
	permission.ResourceServiceFailure: mint(
		func(e *servicefailure.ServiceFailure) pulid.ID { return e.ID },
	),
	permission.ResourceCustomer: mint(func(e *customer.Customer) pulid.ID { return e.ID }),
	permission.ResourceCarrier:  mint(func(e *carrier.Carrier) pulid.ID { return e.ID }),
	permission.ResourceLocation: mint(func(e *location.Location) pulid.ID { return e.ID }),
	permission.ResourceWorker:   mint(func(e *worker.Worker) pulid.ID { return e.ID }),
	permission.ResourceTractor:  mint(func(e *tractor.Tractor) pulid.ID { return e.ID }),
	permission.ResourceTrailer:  mint(func(e *trailer.Trailer) pulid.ID { return e.ID }),
	permission.ResourceCommodity: mint(
		func(e *commodity.Commodity) pulid.ID { return e.ID },
	),
	permission.ResourceHazardousMaterial: mint(
		func(e *hazardousmaterial.HazardousMaterial) pulid.ID { return e.ID },
	),
	permission.ResourceDocument: mint(func(e *document.Document) pulid.ID { return e.ID }),
	permission.ResourceInvoice:  mint(func(e *invoice.Invoice) pulid.ID { return e.ID }),
	permission.ResourceInvoiceDispute: mint(
		func(e *invoice.InvoiceDispute) pulid.ID { return e.ID },
	),
	permission.ResourceInvoiceRun: mint(
		func(e *invoicerun.InvoiceRun) pulid.ID { return e.ID },
	),
	permission.ResourceRateAgreement: mint(
		func(e *rateagreement.RateAgreement) pulid.ID { return e.ID },
	),
	permission.ResourceCustomerPayment: mint(
		func(e *customerpayment.Payment) pulid.ID { return e.ID },
	),
	permission.ResourceReport: mint(
		func(e *report.ReportDefinition) pulid.ID { return e.ID },
	),
	permission.ResourceDashboard: mint(func(e *report.Dashboard) pulid.ID { return e.ID }),
	permission.ResourceWorkerCredential: mint(
		func(e *worker.WorkerCredential) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerTraining: mint(
		func(e *worker.WorkerTrainingRecord) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerSafetyEvent: mint(
		func(e *worker.WorkerSafetyEvent) pulid.ID { return e.ID },
	),
	permission.ResourcePerformanceReview: mint(
		func(e *worker.PerformanceReview) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerInjury: mint(
		func(e *worker.WorkerInjury) pulid.ID { return e.ID },
	),
	permission.ResourceWorkerLeave: mint(
		func(e *worker.WorkerLeaveCase) pulid.ID { return e.ID },
	),
	permission.ResourceDriverSettlement: mint(
		func(e *driversettlement.Settlement) pulid.ID { return e.ID },
	),
	permission.ResourceSettlementDispute: mint(
		func(e *driversettlement.Dispute) pulid.ID { return e.ID },
	),
	permission.ResourceCaptureBatch: mint(
		func(e *capture.CaptureBatch) pulid.ID { return e.ID },
	),
}

// A prefix the table invents is worse than none: the runtime would refuse a
// real id as the wrong kind. Every entry is held to the id the domain itself
// mints for a new record of that resource.
func TestRecordIDPrefixes_MatchWhatTheDomainMints(t *testing.T) {
	t.Parallel()

	table := permission.RecordIDPrefixes()
	require.NotEmpty(t, table)

	for resource, prefix := range table {
		mintID, tested := minters[resource]
		if !assert.Truef(t, tested, "%s has a prefix and no entity minting it here", resource) {
			continue
		}
		id, err := mintID(t.Context())
		require.NoError(t, err, resource)
		assert.Equalf(t, prefix, id.Prefix(), "%s ids", resource)
	}
}

func TestResourceOfIDPrefix_ReadsTheTableBackwards(t *testing.T) {
	t.Parallel()

	for resource, prefix := range permission.RecordIDPrefixes() {
		got, ok := permission.ResourceOfIDPrefix(prefix)
		require.Truef(t, ok, "prefix %s", prefix)
		assert.Equal(t, resource, got, "two resources share prefix %s", prefix)

		declared, has := resource.IDPrefix()
		assert.True(t, has)
		assert.Equal(t, prefix, declared)
	}

	_, ok := permission.ResourceOfIDPrefix("nope_")
	assert.False(t, ok)
}

func TestResource_Noun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "worker credential", permission.ResourceWorkerCredential.Noun())
	assert.Equal(t, "shipment", permission.ResourceShipment.Noun())
}
