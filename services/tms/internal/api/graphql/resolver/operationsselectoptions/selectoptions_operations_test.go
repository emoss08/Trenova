package operationsselectoptions

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/reference/domain/commodity"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestShipmentSelectOptionMapper(t *testing.T) {
	t.Parallel()

	shipmentEntity := &shipment.Shipment{
		ID:        pulid.MustNew("sp_"),
		ProNumber: "PRO-1001",
		BOL:       "BOL-2002",
		Status:    shipmentvocab.StatusInTransit,
		CreatedAt: 1780415000,
	}
	shipmentOption := shipmentSelectOption(shipmentEntity)
	assert.Equal(t, "PRO-1001", shipmentOption.Label)
	assert.Equal(t, "BOL-2002", *shipmentOption.Description)
	assert.Equal(t, string(shipmentvocab.StatusInTransit), shipmentOption.Meta["status"])
	assert.Equal(t, "BOL-2002", shipmentOption.Meta["bol"])
	assert.Equal(
		t,
		shipmentEntity.CreatedAt,
		shipmentSelectOptionItem(shipmentEntity).Cursor.At,
	)
}

func TestCommoditySelectOptionMapper(t *testing.T) {
	t.Parallel()

	hazmatID := pulid.MustNew("hm_")
	commodityEntity := &commodity.Commodity{
		ID:                  pulid.MustNew("com_"),
		Name:                "Lithium batteries",
		Description:         "Boxed cells",
		HazardousMaterialID: hazmatID,
		Stackable:           true,
		Fragile:             false,
	}
	option := commoditySelectOption(commodityEntity)
	assert.Equal(t, "Lithium batteries", option.Label)
	assert.Equal(t, "Boxed cells", *option.Description)
	assert.Equal(t, hazmatID.String(), option.Meta["hazardousMaterialId"])
	assert.Equal(t, true, option.Meta["stackable"])
	assert.Equal(t, false, option.Meta["fragile"])
}

func TestCommoditySelectOptionMapperWithoutHazmat(t *testing.T) {
	t.Parallel()

	option := commoditySelectOption(&commodity.Commodity{
		ID:   pulid.MustNew("com_"),
		Name: "Paper rolls",
	})
	assert.Equal(t, "", option.Meta["hazardousMaterialId"])
}
