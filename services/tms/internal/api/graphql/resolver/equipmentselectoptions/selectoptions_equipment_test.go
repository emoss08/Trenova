package equipmentselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestEquipmentSelectOptionMappers(t *testing.T) {
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
		equipmentManufacturerSelectOptionItem(manufacturer).Cursor.At,
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
}
