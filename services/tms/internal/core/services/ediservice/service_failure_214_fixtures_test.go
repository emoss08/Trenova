package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type serviceFailure214CertificationFixture struct {
	TestCase            *edi.EDITestCase
	Settings            serviceFailure214Settings
	WantDiagnosticPaths []string
}

func TestServiceFailure214CertificationFixtures(t *testing.T) {
	t.Parallel()

	fixtures := serviceFailure214CertificationFixtures()
	require.Len(t, fixtures, 8)

	for _, fixture := range fixtures {
		t.Run(fixture.TestCase.Name, func(t *testing.T) {
			t.Parallel()

			status := fixture.TestCase.Payload.ShipmentStatus
			require.NotNil(t, status)
			require.Equal(t, edi.TransactionSet214, fixture.TestCase.Payload.TransactionSet)
			require.Equal(t, len(fixture.WantDiagnosticPaths), fixture.TestCase.ExpectedErrors)

			diagnostics := serviceFailurePayloadDiagnostics(status, fixture.Settings)
			require.ElementsMatch(t, fixture.WantDiagnosticPaths, diagnosticPaths(diagnostics))
		})
	}
}

func serviceFailure214CertificationFixtures() []serviceFailure214CertificationFixture {
	fixtures := make([]serviceFailure214CertificationFixture, 0, 8)
	fixtures = append(fixtures,
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - Late Pickup",
			Description: "Late pickup service failure emits a partner-ready A3/NS shipment status.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:       "A3",
				StatusReasonCode: "NS",
				StopType:         "Pickup",
				LocationName:     "Dallas Terminal",
				City:             "Dallas",
				StateCode:        "TX",
				PostalCode:       "75001",
			},
		}),
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - Late Delivery",
			Description: "Late delivery service failure emits a partner-ready A3/NS shipment status.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:       "A3",
				StatusReasonCode: "NS",
				StopType:         "Delivery",
				LocationName:     "Chicago Terminal",
				City:             "Chicago",
				StateCode:        "IL",
				PostalCode:       "60601",
			},
		}),
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - SD With Reason",
			Description: "SD status remains valid when the partner reason code is present.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:       "SD",
				StatusReasonCode: "NS",
				LocationName:     "Atlanta Terminal",
			},
		}),
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - Missing SD Reason Diagnostic",
			Description: "SD status without AT7-02 produces the required diagnostic.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:   "SD",
				LocationName: "Atlanta Terminal",
			},
			WantDiagnosticPaths: []string{"shipmentStatus.statusReasonCode"},
		}),
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - Rejected Partner Reason Code",
			Description: "Partner accepted-code validation rejects an unmapped internal reason code.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:       "A3",
				StatusReasonCode: "NS",
				LocationName:     "Dallas Terminal",
			},
			Settings: serviceFailure214Settings{
				AcceptedReasonCodes: map[string]struct{}{"CA": {}},
			},
			WantDiagnosticPaths: []string{"shipmentStatus.statusReasonCode"},
		}),
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - Required Time Code",
			Description: "Partner-required AT7-07 time code is present.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:       "A3",
				StatusReasonCode: "NS",
				EventTimeCode:    "LT",
				LocationName:     "Dallas Terminal",
			},
			Settings: serviceFailure214Settings{RequireTimeCode: true},
		}),
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - Required City State",
			Description: "Partner-required city/state location context is present.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:       "A3",
				StatusReasonCode: "NS",
				LocationName:     "Dallas Terminal",
				City:             "Dallas",
				StateCode:        "TX",
			},
			Settings: serviceFailure214Settings{RequireCityState: true},
		}),
		serviceFailure214Fixture(serviceFailure214FixtureParams{
			Name:        "Service Failure 214 - Partner Reason Mapping",
			Description: "Mapped partner reason code is accepted after service-failure reason mapping.",
			Status: edi.ShipmentStatusPayload{
				StatusCode:                 "A3",
				StatusReasonCode:           "CA",
				ReasonCode:                 "CA",
				ReasonDescription:          "Carrier delay",
				ServiceFailureReasonCodeID: pulidPtr(pulid.MustNew("sfrc_")),
				ServiceFailureReasonCode:   "LATE_PICKUP",
				LocationName:               "Dallas Terminal",
			},
			Settings: serviceFailure214Settings{
				AcceptedReasonCodes: map[string]struct{}{"CA": {}},
			},
		}),
	)
	return fixtures
}

type serviceFailure214FixtureParams struct {
	Name                string
	Description         string
	Status              edi.ShipmentStatusPayload
	Settings            serviceFailure214Settings
	WantDiagnosticPaths []string
}

func serviceFailure214Fixture(params serviceFailure214FixtureParams) serviceFailure214CertificationFixture {
	status := params.Status
	applyServiceFailure214FixtureDefaults(&status)
	return serviceFailure214CertificationFixture{
		TestCase: &edi.EDITestCase{
			ID:                       pulid.MustNew("editc_"),
			BusinessUnitID:           pulid.MustNew("bu_"),
			OrganizationID:           pulid.MustNew("org_"),
			PartnerDocumentProfileID: pulid.MustNew("edidp_"),
			Name:                     params.Name,
			Description:              params.Description,
			Payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet214,
				ShipmentStatus: &status,
			},
			ExpectedErrors: len(params.WantDiagnosticPaths),
		},
		Settings:            params.Settings,
		WantDiagnosticPaths: params.WantDiagnosticPaths,
	}
}

func applyServiceFailure214FixtureDefaults(status *edi.ShipmentStatusPayload) {
	status.ShipmentID = pulid.MustNew("sp_")
	status.ServiceFailureID = pulidPtr(pulid.MustNew("sf_"))
	status.BOL = "BOL-SF-214"
	status.ProNumber = "PRO-SF-214"
	status.EventDate = 1779192000
	status.EventTime = 1779192000
	if status.StatusCode == "" {
		status.StatusCode = "A3"
	}
	if status.LocationID.IsNil() {
		status.LocationID = pulid.MustNew("loc_")
	}
	if status.References == nil {
		status.References = map[string]string{
			"serviceFailure214Trigger": "Reviewed",
			"serviceFailureId":         status.ServiceFailureID.String(),
		}
	}
}

func pulidPtr(id pulid.ID) *pulid.ID {
	return &id
}
