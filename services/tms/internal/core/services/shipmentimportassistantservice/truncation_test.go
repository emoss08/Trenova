package shipmentimportassistantservice

import (
	"context"
	"errors"
	"testing"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAddLocation_CutsTheCodeOnACharacterBoundary(t *testing.T) {
	t.Parallel()

	tenant := importTenant()
	engine := mocks.NewMockPermissionEngine(t)
	expectCheck(engine, tenant, permission.ResourceLocation, permission.OpCreate, true)

	states := mocks.NewMockUsStateRepository(t)
	states.EXPECT().
		GetByAbbreviation(mock.Anything, "QC").
		Return(&usstate.UsState{ID: pulid.MustNew("us_")}, nil).
		Once()

	categories := mocks.NewMockLocationCategoryRepository(t)
	categories.EXPECT().
		SelectOptions(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*locationcategory.LocationCategory]{
			Items: []*locationcategory.LocationCategory{{ID: pulid.MustNew("lc_")}},
			Total: 1,
		}, nil).
		Once()

	var created *location.Location
	transformer := mocks.NewMockDataTransformer(t)
	transformer.EXPECT().
		TransformLocation(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, entity *location.Location) error {
			created = entity
			return errors.New("stop before saving")
		}).
		Once()

	service := &Service{
		logger:               zap.NewNop(),
		permissions:          engine,
		usStateRepo:          states,
		locationCategoryRepo: categories,
		locationService: locationservice.New(locationservice.Params{
			Logger:      zap.NewNop(),
			Transformer: transformer,
		}),
	}

	service.RunTool(t.Context(), tenant, &serviceports.ToolCall{
		ID:   "call_1",
		Name: "add_location",
		Arguments: map[string]any{
			"name":          "Entrepôt Montréal Nord",
			"address_line1": "1 rue Principale",
			"city":          "Montréal",
			"state_abbrev":  "QC",
			"postal_code":   "H1A 0A1",
		},
	})

	require.NotNil(t, created)
	assert.True(t, utf8.ValidString(created.Code))
	assert.Equal(t, "Entrepôt M", created.Code)
}
