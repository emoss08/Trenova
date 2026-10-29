package ediservicefailure

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

func TestBuildServiceFailure214LifecyclePayloadStatusPrecedence(t *testing.T) {
	t.Parallel()

	reasonCode := &servicefailure.ReasonCode{
		DefaultStatusCode: "a3",
		DefaultReasonCode: "ns",
	}
	base := &servicefailure.ServiceFailure{
		ID:                    pulid.MustNew("sf_"),
		ShipmentID:            pulid.MustNew("sp_"),
		Number:                "SF-1004",
		Type:                  servicefailure.TypeLateDelivery,
		Status:                servicefailure.StatusReviewed,
		ReasonCode:            reasonCode,
		DetectedAt:            1,
		LateMinutes:           1,
		X12ReasonCodeOverride: " ca ",
	}

	payload := buildServiceFailure214LifecyclePayload(
		base,
		nil,
		servicefailureports.ServiceFailureEDITriggerReviewed,
	)
	require.Equal(t, "A3", payload.ShipmentStatus.StatusCode)
	require.Equal(t, "CA", payload.ShipmentStatus.StatusReasonCode)
	require.Equal(
		t,
		"Reviewed",
		payload.ShipmentStatus.References[serviceFailure214TriggerReference],
	)

	base.X12StatusCodeOverride = " sd "
	payload = buildServiceFailure214LifecyclePayload(
		base,
		nil,
		servicefailureports.ServiceFailureEDITriggerReviewed,
	)
	require.Equal(t, "SD", payload.ShipmentStatus.StatusCode)

	base.X12StatusCodeOverride = ""
	payload = buildServiceFailure214LifecyclePayload(
		base,
		nil,
		servicefailureports.ServiceFailureEDITriggerReviewed,
	)
	require.Equal(t, "A3", payload.ShipmentStatus.StatusCode)

	base.ReasonCode.DefaultStatusCode = ""
	payload = buildServiceFailure214LifecyclePayload(
		base,
		nil,
		servicefailureports.ServiceFailureEDITriggerReviewed,
	)
	require.Equal(t, "SD", payload.ShipmentStatus.StatusCode)
}
