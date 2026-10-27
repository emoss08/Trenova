package customfieldservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/customfieldports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customfieldportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func namedDefinitions() (po, declared, appointment *customfield.CustomFieldDefinition) {
	po = &customfield.CustomFieldDefinition{
		ID:        pulid.MustNew("cfd_"),
		Name:      "po_number",
		Label:     "PO number",
		FieldType: customfield.FieldTypeText,
	}
	declared = &customfield.CustomFieldDefinition{
		ID:        pulid.MustNew("cfd_"),
		Name:      "declared_value",
		Label:     "Declared value",
		FieldType: customfield.FieldTypeNumber,
	}
	appointment = &customfield.CustomFieldDefinition{
		ID:        pulid.MustNew("cfd_"),
		Name:      "appointment_date",
		Label:     "Appointment date",
		FieldType: customfield.FieldTypeDate,
	}
	return po, declared, appointment
}

func TestValuesService_SaveNamed_MergesByNameAndStoresANumberGivenAsText(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := customfieldportsmock.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)
	tenantInfo := newTenantInfo()
	po, declared, appointment := namedDefinitions()

	definitions := customfieldports.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "shipment",
	}
	defRepo.On("GetActiveByResourceType", mock.Anything, definitions).
		Return([]*customfield.CustomFieldDefinition{po, declared, appointment}, nil)
	stored := &customfieldports.GetCustomFieldValuesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "shipment",
		ResourceID:   "shp_1",
	}
	valueRepo.On("GetByResource", mock.Anything, stored).Return([]*customfield.CustomFieldValue{
		{DefinitionID: appointment.ID, Value: "2026-03-10"},
		{DefinitionID: po.ID, Value: "PO-OLD"},
	}, nil)
	valueRepo.On("Upsert", mock.Anything, &customfieldports.UpsertCustomFieldValuesRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "shipment",
		ResourceID:   "shp_1",
		Values: map[string]any{
			po.ID.String():          "PO-559031",
			declared.ID.String():    float64(1250.5),
			appointment.ID.String(): "2026-03-10",
		},
	}).Return(nil)

	multiErr := svc.SaveNamed(t.Context(), tenantInfo, "shipment", "shp_1", map[string]any{
		"po_number":      "PO-559031",
		"declared_value": "1250.50",
	})

	assert.Nil(t, multiErr)
	valueRepo.AssertExpectations(t)
}

func TestValuesService_SaveNamed_RefusesANameWithNoActiveDefinitionAndSavesNothing(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := customfieldportsmock.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)
	tenantInfo := newTenantInfo()
	po, _, _ := namedDefinitions()

	defRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldDefinition{po}, nil)
	valueRepo.On("GetByResource", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldValue{}, nil)

	multiErr := svc.SaveNamed(t.Context(), tenantInfo, "shipment", "shp_1", map[string]any{
		"seal_number": "S-1",
	})

	require.NotNil(t, multiErr)
	assert.True(t, multiErr.HasErrors())
	valueRepo.AssertNotCalled(t, "Upsert", mock.Anything, mock.Anything)
}

func TestValuesService_GetNamed_KeysValuesByNameAndReadsADateAsADay(t *testing.T) {
	t.Parallel()

	defRepo := new(mockCustomFieldRepository)
	valueRepo := customfieldportsmock.NewMockCustomFieldValueRepository(t)
	svc := newTestValuesService(defRepo, valueRepo)
	tenantInfo := newTenantInfo()
	po, declared, appointment := namedDefinitions()

	defRepo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldDefinition{po, declared, appointment}, nil)
	valueRepo.On("GetByResource", mock.Anything, mock.Anything).
		Return([]*customfield.CustomFieldValue{
			{DefinitionID: po.ID, Value: "PO-559031"},
			{DefinitionID: declared.ID, Value: float64(1250.5)},
			{DefinitionID: appointment.ID, Value: "2026-03-10T15:30:00Z"},
			{DefinitionID: pulid.MustNew("cfd_"), Value: "from a retired field"},
		}, nil)

	named, err := svc.GetNamed(t.Context(), tenantInfo, "shipment", "shp_1")

	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"po_number":        "PO-559031",
		"declared_value":   float64(1250.5),
		"appointment_date": "2026-03-10",
	}, named)
}
