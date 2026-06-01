package servicefailure

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func TestServiceFailureValidateGeneratesIdentity(t *testing.T) {
	entity := &ServiceFailure{
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		ShipmentID:         pulid.MustNew("shp_"),
		ShipmentMoveID:     pulid.MustNew("shpm_"),
		StopID:             pulid.MustNew("stp_"),
		Type:               TypeLateDelivery,
		Source:             SourceDetected,
		Status:             StatusOpen,
		StopType:           shipment.StopTypeDelivery,
		ScheduledCutoff:    1_799_000_000,
		ActualArrival:      1_799_003_601,
		GracePeriodMinutes: 30,
		LateMinutes:        31,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		t.Fatalf("expected service failure to validate, got %v", multiErr)
	}

	if entity.ID.IsNil() {
		t.Fatal("expected validation to generate an ID")
	}
	if entity.Number == "" {
		t.Fatal("expected validation to generate a service failure number")
	}
}
