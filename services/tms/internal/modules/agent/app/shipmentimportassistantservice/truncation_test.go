package shipmentimportassistantservice

import (
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/locationcategoryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/usstateportsmock"
	"context"
	"errors"
	"testing"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/domain/locationcategory"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/places/app/locationservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	engine := permissionportsmock.NewMockPermissionEngine(t)
	expectCheck(engine, tenant, permission.ResourceLocation, permission.OpCreate, true)

	states := usstateportsmock.NewMockUsStateRepository(t)
	states.EXPECT().
		GetByAbbreviation(mock.Anything, "QC").
		Return(&usstate.UsState{ID: pulid.MustNew("us_")}, nil).
		Once()

	categories := locationcategoryportsmock.NewMockLocationCategoryRepository(t)
	categories.EXPECT().
		SelectOptions(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*locationcategory.LocationCategory]{
			Items: []*locationcategory.LocationCategory{{ID: pulid.MustNew("lc_")}},
			Total: 1,
		}, nil).
		Once()

	var created *location.Location
	transformer := accessorialchargeportsmock.NewMockDataTransformer(t)
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

	service.RunTool(t.Context(), tenant, &agentports.ToolCall{
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
