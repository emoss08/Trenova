package shipmentservicetest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
)

func ValidShipment() *shipment.Shipment {
	return &shipment.Shipment{
		BusinessUnitID:    pulid.MustNew("bu_"),
		OrganizationID:    pulid.MustNew("org_"),
		ServiceTypeID:     pulid.MustNew("svc_"),
		ShipmentTypeID:    pulid.MustNew("sht_"),
		CustomerID:        pulid.MustNew("cus_"),
		FormulaTemplateID: pulid.MustNew("fmt_"),
		Status:            shipment.StatusNew,
		BOL:               "BOL-100",
		Moves: []*shipment.ShipmentMove{
			ValidMove(),
		},
	}
}

func ValidMove() *shipment.ShipmentMove {
	return &shipment.ShipmentMove{
		Status:   shipment.MoveStatusNew,
		Loaded:   true,
		Sequence: 0,
		Stops: []*shipment.Stop{
			MakeStop(shipment.StopTypePickup, 0, 1, 2),
			MakeStop(shipment.StopTypeDelivery, 1, 3, 4),
		},
	}
}

func MakeStop(
	stopType shipment.StopType,
	sequence int64,
	scheduledWindowStart int64,
	scheduledWindowEnd int64,
) *shipment.Stop {
	stop := ValidStop()
	stop.Type = stopType
	stop.Sequence = sequence
	stop.ScheduledWindowStart = scheduledWindowStart
	stop.ScheduledWindowEnd = new(scheduledWindowEnd)
	return stop
}

func AssertErrorField(t *testing.T, multiErr *errortypes.MultiError, field string) {
	t.Helper()

	for _, validationErr := range multiErr.Errors {
		if validationErr.Field == field {
			return
		}
	}

	t.Fatalf("expected validation error on field %s", field)
}

func CloneShipment(source *shipment.Shipment) *shipment.Shipment {
	return source.CloneMoveGraph()
}

func BillingControlWith(
	t *testing.T,
	disposition billingcontrol.UnratedShipmentDisposition,
) *tenantportsmock.MockBillingControlRepository {
	t.Helper()

	repo := tenantportsmock.NewMockBillingControlRepository(t)
	repo.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(&billingcontrol.BillingControl{UnratedShipmentDisposition: disposition}, nil).
		Maybe()

	return repo
}

func ValidStop() *shipment.Stop {
	return &shipment.Stop{
		LocationID:           pulid.MustNew("loc_"),
		Status:               shipment.StopStatusNew,
		Type:                 shipment.StopTypePickup,
		ScheduleType:         shipment.StopScheduleTypeOpen,
		Sequence:             0,
		ScheduledWindowStart: 1,
		ScheduledWindowEnd:   new(int64(1)),
	}
}
