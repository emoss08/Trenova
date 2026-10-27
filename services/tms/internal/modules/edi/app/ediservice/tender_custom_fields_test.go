package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/mocks/customfieldportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSaveTenderCustomFields_SavesTheTendersValuesOnTheNewShipmentByName(t *testing.T) {
	t.Parallel()

	values := customfieldportsmock.NewMockNamedValues(t)
	svc := &Service{customFields: values}
	created := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	tendered := map[string]any{"po_number": "PO-559031"}
	values.EXPECT().
		SaveNamed(
			mock.Anything,
			pagination.TenantInfo{OrgID: created.OrganizationID, BuID: created.BusinessUnitID},
			"shipment",
			created.ID.String(),
			tendered,
		).
		Return(nil)

	require.NoError(t, svc.saveTenderCustomFields(t.Context(), created, tendered))
}

func TestSaveTenderCustomFields_RefusesTheShipmentWhenAValueIsRefused(t *testing.T) {
	t.Parallel()

	values := customfieldportsmock.NewMockNamedValues(t)
	svc := &Service{customFields: values}
	refused := errortypes.NewMultiError()
	refused.Add(
		"customFields.seal",
		errortypes.ErrInvalid,
		"There is no active custom field named {0}",
		"seal",
	)
	values.EXPECT().
		SaveNamed(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(refused)

	err := svc.saveTenderCustomFields(
		t.Context(),
		&shipment.Shipment{ID: pulid.MustNew("shp_")},
		map[string]any{"seal": "S-1"},
	)

	assert.ErrorIs(t, err, refused)
}

func TestSaveTenderCustomFields_WritesNothingForATenderWithoutCustomFields(t *testing.T) {
	t.Parallel()

	values := customfieldportsmock.NewMockNamedValues(t)
	svc := &Service{customFields: values}

	require.NoError(t, svc.saveTenderCustomFields(t.Context(), &shipment.Shipment{}, nil))
	require.NoError(t, (&Service{}).saveTenderCustomFields(
		t.Context(),
		&shipment.Shipment{},
		map[string]any{"po_number": "PO-1"},
	))
}
