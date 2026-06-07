package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testGraphQLAuthContext(orgID, buID, userID pulid.ID) *authctx.AuthContext {
	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	}
}

func TestSelectOptionsRequestFromInput_MapsPaginationFiltersAndIDs(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	id := pulid.MustNew("tr_")
	query := "van"
	first := pagination.MaxLimit + 10
	offset := -4
	filters := map[string]any{"classes": []any{"Trailer"}}

	req, err := selectOptionsRequestFromInput(
		gqlmodel.SelectOptionsInput{
			Resource: gqlmodel.SelectOptionResourceTrailer,
			Query:    &query,
			First:    &first,
			Offset:   &offset,
			Ids:      []string{id.String()},
			Filters:  filters,
		},
		testGraphQLAuthContext(orgID, buID, userID),
	)
	require.NoError(t, err)

	assert.Equal(t, orgID, req.tenantInfo.OrgID)
	assert.Equal(t, buID, req.tenantInfo.BuID)
	assert.Equal(t, userID, req.tenantInfo.UserID)
	assert.Equal(t, pagination.MaxLimit, req.selectQuery.Pagination.Limit)
	assert.Equal(t, pagination.DefaultOffset, req.selectQuery.Pagination.Offset)
	assert.Equal(t, "van", req.selectQuery.Query)
	assert.Equal(t, []pulid.ID{id}, req.ids)
	assert.Equal(t, filters, req.filters)
}

func TestSelectOptions_InitialResourcesDoNotRequireResourcePermission(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	equipmentTypeID := pulid.MustNew("et_")
	repo := mocks.NewMockEquipmentTypeRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *repositories.EquipmentTypeSelectOptionsRequest) bool {
			return req.SelectQueryRequest.TenantInfo.OrgID == orgID &&
				req.SelectQueryRequest.TenantInfo.BuID == buID &&
				req.SelectQueryRequest.TenantInfo.UserID == userID &&
				req.SelectQueryRequest.Pagination.Limit == 20
		})).
		Return(&pagination.ListResult[*equipmenttype.EquipmentType]{
			Items: []*equipmenttype.EquipmentType{
				{
					ID:          equipmentTypeID,
					CreatedAt:   1780415883,
					Code:        "VAN",
					Description: "Dry van",
					Class:       equipmenttype.ClassTrailer,
					Color:       "#ffffff",
				},
			},
			Total: 1,
		}, nil).
		Once()

	permissionEngine := &recordingPermissionEngine{}
	resolver := &queryResolver{&Resolver{
		equipmentTypeService: equipmenttypeservice.New(equipmenttypeservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
		permissionEngine: permissionEngine,
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEquipmentType,
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "VAN", result.Edges[0].Node.Label)
	assert.Nil(t, permissionEngine.request)
}

func TestSelectOptions_EquipmentManufacturerUsesAuthContextOnly(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	manufacturerID := pulid.MustNew("em_")
	repo := mocks.NewMockEquipmentManufacturerRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *pagination.SelectQueryRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.TenantInfo.UserID == userID
		})).
		Return(&pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer]{
			Items: []*equipmentmanufacturer.EquipmentManufacturer{
				{
					ID:        manufacturerID,
					CreatedAt: 1780415883,
					Name:      "Great Dane",
				},
			},
			Total: 1,
		}, nil).
		Once()

	permissionEngine := &recordingPermissionEngine{}
	resolver := &queryResolver{&Resolver{
		equipmentManufacturerService: equipmentmanufacturerservice.New(equipmentmanufacturerservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
		permissionEngine: permissionEngine,
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEquipmentManufacturer,
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "Great Dane", result.Edges[0].Node.Label)
	assert.Nil(t, permissionEngine.request)
}

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
	assert.Equal(t, manufacturer.CreatedAt, equipmentManufacturerSelectOptionItem(manufacturer).cursor.CreatedAt)

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

	stateOption := usStateSelectOption(&usstate.UsState{
		ID:           pulid.MustNew("us_"),
		Name:         "Illinois",
		Abbreviation: "IL",
		CountryIso3:  "USA",
	})
	assert.Equal(t, "Illinois", stateOption.Label)
	assert.Equal(t, "IL", stateOption.Meta["abbreviation"])
	assert.Equal(t, "USA", stateOption.Meta["countryIso3"])
}
