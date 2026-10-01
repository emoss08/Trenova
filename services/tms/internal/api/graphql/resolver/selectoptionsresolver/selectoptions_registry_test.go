package selectoptionsresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectOptionConnection_UsesOpaqueEntityCursors(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wrk_")
	createdAt := int64(1780415883)

	result, err := selectOptionConnection(
		[]selectOptionConnectionItem{
			{
				option: &gqlmodel.SelectOption{
					ID:    id.String(),
					Label: "John Smith",
				},
				cursor: pagination.Cursor{
					CreatedAt: createdAt,
					ID:        id,
				},
			},
		},
		1,
		0,
	)
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.NotEqual(t, "1", result.Edges[0].Cursor)
	require.NotNil(t, result.PageInfo.EndCursor)
	assert.Equal(t, result.Edges[0].Cursor, *result.PageInfo.EndCursor)

	decoded, err := pagination.DecodeCursor(result.Edges[0].Cursor)
	require.NoError(t, err)
	assert.Equal(t, createdAt, decoded.CreatedAt)
	assert.Equal(t, id, decoded.ID)
}

func TestSelectOptionMappers(t *testing.T) {
	t.Parallel()

	primaryWorkerID := pulid.MustNew("wrk_")
	secondaryWorkerID := pulid.MustNew("wrk_")

	equipmentOption := equipmentTypeSelectOption(&equipmenttype.EquipmentType{
		ID:          pulid.MustNew("et_"),
		Code:        "REEFER",
		Description: "Refrigerated trailer",
		Class:       equipmenttype.ClassTrailer,
		Color:       "#00aaff",
	})
	assert.Equal(t, "REEFER", equipmentOption.Label)
	assert.Equal(t, "Refrigerated trailer", *equipmentOption.Description)
	assert.Equal(t, "#00aaff", equipmentOption.Meta["color"])
	assert.Equal(t, equipmenttype.ClassTrailer, equipmentOption.Meta["class"])

	manufacturer := &equipmentmanufacturer.EquipmentManufacturer{
		ID:          pulid.MustNew("em_"),
		Name:        "Great Dane",
		Description: "Trailer manufacturer",
		CreatedAt:   1780415999,
	}
	manufacturerOption := equipmentManufacturerSelectOption(manufacturer)
	assert.Equal(t, "Great Dane", manufacturerOption.Label)
	assert.Equal(t, "Trailer manufacturer", *manufacturerOption.Description)
	assert.Equal(
		t,
		manufacturer.CreatedAt,
		equipmentManufacturerSelectOptionItem(manufacturer).cursor.CreatedAt,
	)

	assert.Equal(t, "TRL-1", trailerSelectOption(&trailer.Trailer{
		ID:   pulid.MustNew("tr_"),
		Code: "TRL-1",
	}).Label)

	tractorOption := tractorSelectOption(&tractor.Tractor{
		ID:                pulid.MustNew("trac_"),
		Code:              "TRC-1",
		PrimaryWorkerID:   primaryWorkerID,
		SecondaryWorkerID: secondaryWorkerID,
	})
	assert.Equal(t, "TRC-1", tractorOption.Label)
	assert.Equal(t, primaryWorkerID.String(), tractorOption.Meta["primaryWorkerId"])
	assert.Equal(t, secondaryWorkerID.String(), tractorOption.Meta["secondaryWorkerId"])

	workerOption := workerSelectOption(&worker.Worker{
		ID:        pulid.MustNew("wrk_"),
		FirstName: "Ada",
		LastName:  "Lovelace",
		WholeName: "Ada Lovelace",
		FleetCode: &fleetcode.FleetCode{Code: "OTR"},
	})
	assert.Equal(t, "Ada Lovelace", workerOption.Label)
	assert.Equal(t, "Ada", workerOption.Meta["firstName"])
	assert.Equal(t, "OTR", workerOption.Meta["fleetCode"])

	locationOption := locationSelectOption(&location.Location{
		ID:   pulid.MustNew("loc_"),
		Code: "DAL01",
		Name: "Dallas DC",
	})
	assert.Equal(t, "Dallas DC", locationOption.Label)
	assert.Equal(t, "DAL01", *locationOption.Description)
	assert.Equal(t, "DAL01", locationOption.Meta["code"])

	zoneOption := rateZoneSelectOption(&ratezone.RateZone{
		ID:     pulid.MustNew("rzn_"),
		Code:   "SW",
		Name:   "Southwest",
		Status: domaintypes.StatusActive,
	})
	assert.Equal(t, "Southwest", zoneOption.Label)
	assert.Equal(t, "SW", *zoneOption.Description)
	assert.Equal(t, "Active", zoneOption.Meta["status"])

	stateOption := usStateSelectOption(&usstate.UsState{
		ID:           pulid.MustNew("us_"),
		Name:         "Illinois",
		Abbreviation: "IL",
		CountryIso3:  "USA",
	})
	assert.Equal(t, "Illinois", stateOption.Label)
	assert.Equal(t, "IL", stateOption.Meta["abbreviation"])
	assert.Equal(t, "USA", stateOption.Meta["countryIso3"])

	shipmentEntity := &shipment.Shipment{
		ID:        pulid.MustNew("sp_"),
		ProNumber: "PRO-1001",
		BOL:       "BOL-2002",
		Status:    shipment.StatusInTransit,
		CreatedAt: 1780415000,
	}
	shipmentOption := shipmentSelectOption(shipmentEntity)
	assert.Equal(t, "PRO-1001", shipmentOption.Label)
	assert.Equal(t, "BOL-2002", *shipmentOption.Description)
	assert.Equal(t, string(shipment.StatusInTransit), shipmentOption.Meta["status"])
	assert.Equal(t, "BOL-2002", shipmentOption.Meta["bol"])
	assert.Equal(
		t,
		shipmentEntity.CreatedAt,
		shipmentSelectOptionItem(shipmentEntity).cursor.CreatedAt,
	)

	transferEntity := &edi.EDITransfer{
		ID:        pulid.MustNew("edilt_"),
		Status:    edi.TransferStatusSubmitted,
		CreatedAt: 1780415500,
		TenderPayload: edi.LoadTenderPayload{
			BOL:           "BOL-3003",
			CustomerLabel: "ACME Freight",
		},
		SourcePartner: &edi.EDIPartner{Name: "Partner A"},
		TargetPartner: &edi.EDIPartner{Name: "Partner B"},
	}
	transferOption := ediTransferSelectOption(transferEntity)
	assert.Equal(t, "BOL-3003", transferOption.Label)
	assert.Equal(t, "ACME Freight", *transferOption.Description)
	assert.Equal(t, string(edi.TransferStatusSubmitted), transferOption.Meta["status"])
	assert.Equal(t, "Partner A", transferOption.Meta["sourcePartner"])
	assert.Equal(t, "Partner B", transferOption.Meta["targetPartner"])
	assert.Equal(
		t,
		transferEntity.CreatedAt,
		ediTransferSelectOptionItem(transferEntity).cursor.CreatedAt,
	)

	fallbackTransfer := &edi.EDITransfer{
		ID:        pulid.MustNew("edilt_"),
		Status:    edi.TransferStatusSubmitted,
		CreatedAt: 1780415600,
	}
	assert.Equal(
		t,
		"Load tender "+fallbackTransfer.ID.String(),
		ediTransferSelectOption(fallbackTransfer).Label,
	)

	connectionEntity := &edi.EDIConnection{
		ID:                   pulid.MustNew("edic_"),
		SourceOrganizationID: pulid.MustNew("org_"),
		TargetOrganizationID: pulid.MustNew("org_"),
		Method:               edi.ConnectionMethodInternal,
		Status:               edi.ConnectionStatusActive,
		CreatedAt:            1780415700,
		SourceOrganization:   &tenant.Organization{Name: "Acme Corp"},
		TargetOrganization:   &tenant.Organization{Name: "Beta LLC"},
	}
	connectionOption := ediConnectionSelectOption(connectionEntity)
	assert.Equal(t, "Acme Corp → Beta LLC", connectionOption.Label)
	assert.Equal(t, "Internal \u00b7 Active", *connectionOption.Description)
	assert.Equal(t, string(edi.ConnectionMethodInternal), connectionOption.Meta["method"])
	assert.Equal(t, string(edi.ConnectionStatusActive), connectionOption.Meta["status"])
	assert.Equal(t, "Acme Corp", connectionOption.Meta["sourceOrganizationName"])
	assert.Equal(t, "Beta LLC", connectionOption.Meta["targetOrganizationName"])
	assert.Equal(
		t,
		connectionEntity.CreatedAt,
		ediConnectionSelectOptionItem(connectionEntity).cursor.CreatedAt,
	)

	fallbackConnection := &edi.EDIConnection{
		ID:                   pulid.MustNew("edic_"),
		SourceOrganizationID: pulid.MustNew("org_"),
		TargetOrganizationID: pulid.MustNew("org_"),
		Method:               edi.ConnectionMethodInternal,
		Status:               edi.ConnectionStatusActive,
		CreatedAt:            1780415800,
	}
	fallbackOption := ediConnectionSelectOption(fallbackConnection)
	assert.Equal(
		t,
		fallbackConnection.SourceOrganizationID.String()+" → "+fallbackConnection.TargetOrganizationID.String(),
		fallbackOption.Label,
	)
}
