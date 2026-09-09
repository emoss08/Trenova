package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
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
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/core/services/ratezoneservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
		selectOptionRegistryEntry{},
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
		equipmentManufacturerService: equipmentmanufacturerservice.New(
			equipmentmanufacturerservice.Params{
				Logger: zap.NewNop(),
				Repo:   repo,
			},
		),
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

func TestSelectOptions_LocationByIDsResolvesNames(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	firstID := pulid.MustNew("loc_")
	secondID := pulid.MustNew("loc_")
	repo := mocks.NewMockLocationRepository(t)
	repo.EXPECT().
		GetByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetLocationsByIDsRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				len(req.LocationIDs) == 2
		})).
		Return([]*location.Location{
			// Returned out of order on purpose: the connection must follow the
			// order the ids were asked in, not the order the database found them.
			{ID: secondID, Code: "CHI01", Name: "Chicago DC", CreatedAt: 1780415884},
			{ID: firstID, Code: "DAL01", Name: "Dallas DC", CreatedAt: 1780415883},
		}, nil).
		Once()

	resolver := &queryResolver{&Resolver{
		locationService: locationservice.New(locationservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceLocation,
		Ids:      []string{firstID.String(), secondID.String()},
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 2)
	assert.Equal(t, "Dallas DC", result.Edges[0].Node.Label)
	assert.Equal(t, "DAL01", result.Edges[0].Node.Meta["code"])
	assert.Equal(t, "Chicago DC", result.Edges[1].Node.Label)
}

func TestSelectOptions_RateZoneByIDsResolvesNames(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	zoneID := pulid.MustNew("rzn_")
	repo := mocks.NewMockRateZoneRepository(t)
	repo.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req *repositories.GetRateZoneByIDRequest) bool {
			return req.RateZoneID == zoneID &&
				req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID
		})).
		Return(&ratezone.RateZone{
			ID:        zoneID,
			Code:      "SW",
			Name:      "Southwest",
			CreatedAt: 1780415883,
		}, nil).
		Once()

	resolver := &queryResolver{&Resolver{
		rateZoneService: ratezoneservice.New(ratezoneservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceRateZone,
		Ids:      []string{zoneID.String()},
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "Southwest", result.Edges[0].Node.Label)
	assert.Equal(t, "SW", result.Edges[0].Node.Meta["code"])
}

func TestSelectOptions_EDIConnectionByIDsResolvesLabels(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	firstID := pulid.MustNew("edic_")
	secondID := pulid.MustNew("edic_")
	repo := mocks.NewMockEDIConnectionRepository(t)
	repo.EXPECT().
		GetConnectionsByIDs(mock.Anything, mock.MatchedBy(func(req repositories.GetEDIConnectionsByIDsRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				len(req.ConnectionIDs) == 2
		})).
		Return([]*edi.EDIConnection{
			{
				ID:                   secondID,
				SourceOrganization:   &tenant.Organization{Name: "Beta LLC"},
				TargetOrganization:   &tenant.Organization{Name: "Gamma Inc"},
				SourceOrganizationID: pulid.MustNew("org_"),
				TargetOrganizationID: pulid.MustNew("org_"),
				Method:               edi.ConnectionMethodInternal,
				Status:               edi.ConnectionStatusActive,
				CreatedAt:            1780415884,
			},
			{
				ID:                   firstID,
				SourceOrganization:   &tenant.Organization{Name: "Acme Corp"},
				TargetOrganization:   &tenant.Organization{Name: "Beta LLC"},
				SourceOrganizationID: pulid.MustNew("org_"),
				TargetOrganizationID: pulid.MustNew("org_"),
				Method:               edi.ConnectionMethodInternal,
				Status:               edi.ConnectionStatusActive,
				CreatedAt:            1780415883,
			},
		}, nil).
		Once()

	resolver := &queryResolver{&Resolver{
		ediService: ediservice.New(ediservice.Params{
			Logger:         zap.NewNop(),
			ConnectionRepo: repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEDIConnection,
		Ids:      []string{firstID.String(), secondID.String()},
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 2)
	assert.Equal(t, "Acme Corp → Beta LLC", result.Edges[0].Node.Label)
	assert.Equal(t, "Beta LLC → Gamma Inc", result.Edges[1].Node.Label)
}

func TestSelectOptions_EDIConnectionSelectOptionsFiltersActive(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	repo := mocks.NewMockEDIConnectionRepository(t)
	repo.EXPECT().
		SelectOptions(mock.Anything, mock.MatchedBy(func(req *repositories.EDIConnectionSelectOptionsRequest) bool {
			return req.SelectQueryRequest.TenantInfo.OrgID == orgID &&
				req.SelectQueryRequest.TenantInfo.BuID == buID
		})).
		Return(&pagination.ListResult[*edi.EDIConnection]{
			Items: []*edi.EDIConnection{
				{
					ID:                   pulid.MustNew("edic_"),
					SourceOrganization:   &tenant.Organization{Name: "Acme Corp"},
					TargetOrganization:   &tenant.Organization{Name: "Beta LLC"},
					SourceOrganizationID: orgID,
					TargetOrganizationID: pulid.MustNew("org_"),
					Method:               edi.ConnectionMethodInternal,
					Status:               edi.ConnectionStatusActive,
					CreatedAt:            1780415883,
				},
			},
			Total: 1,
		}, nil).
		Once()

	resolver := &queryResolver{&Resolver{
		ediService: ediservice.New(ediservice.Params{
			Logger:         zap.NewNop(),
			ConnectionRepo: repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEDIConnection,
		Query:    stringPtr("Acme"),
	})
	require.NoError(t, err)

	require.Len(t, result.Edges, 1)
	assert.Equal(t, "Acme Corp → Beta LLC", result.Edges[0].Node.Label)
	assert.Equal(t, "Internal \u00b7 Active", *result.Edges[0].Node.Description)
}

func TestSelectOptions_EDIDocumentTypeCatalogIDs(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	outbound := &edi.EDIDocumentType{
		ID:             pulid.ID("edidt_x12_204_outbound"),
		Code:           "X12-204-OUT",
		Name:           "X12 204 Motor Carrier Load Tender",
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		DefaultVersion: "004010",
		CreatedAt:      1780415883,
	}
	inbound := &edi.EDIDocumentType{
		ID:             pulid.ID("edidt_x12_990_inbound"),
		Code:           "X12-990-IN",
		Name:           "X12 990 Response to Load Tender",
		TransactionSet: edi.TransactionSet990,
		Direction:      edi.DocumentDirectionInbound,
		DefaultVersion: "004010",
		CreatedAt:      1780415884,
	}
	repo := mocks.NewMockEDIDocumentTypeRepository(t)
	repo.EXPECT().
		SelectDocumentTypeOptions(mock.Anything, mock.MatchedBy(func(req *repositories.EDIDocumentTypeSelectOptionsRequest) bool {
			return req.SelectQueryRequest.TenantInfo.OrgID == orgID &&
				req.SelectQueryRequest.TenantInfo.BuID == buID
		})).
		Return(&pagination.ListResult[*edi.EDIDocumentType]{
			Items: []*edi.EDIDocumentType{outbound, inbound},
			Total: 2,
		}, nil).
		Twice()

	resolver := &queryResolver{&Resolver{
		ediService: ediservice.New(ediservice.Params{
			Logger:           zap.NewNop(),
			DocumentTypeRepo: repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	listed, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEDIDocumentType,
	})
	require.NoError(t, err)
	require.Len(t, listed.Edges, 2)
	assert.Equal(t, "edidt_x12_204_outbound", listed.Edges[0].Node.ID)
	assert.Equal(t, "X12-204-OUT - X12 204 Motor Carrier Load Tender", listed.Edges[0].Node.Label)
	assert.Equal(t, "204", listed.Edges[0].Node.Meta["transactionSet"])
	assert.Equal(t, "Outbound", listed.Edges[0].Node.Meta["direction"])
	assert.Equal(t, "004010", listed.Edges[0].Node.Meta["defaultVersion"])

	cursor, err := pagination.DecodeCursor(listed.Edges[1].Cursor)
	require.NoError(t, err)
	assert.Equal(t, inbound.ID, cursor.ID)
	assert.Equal(t, inbound.CreatedAt, cursor.CreatedAt)

	hydrated, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEDIDocumentType,
		Ids:      []string{"edidt_x12_990_inbound"},
	})
	require.NoError(t, err)
	require.Len(t, hydrated.Edges, 1)
	assert.Equal(t, "edidt_x12_990_inbound", hydrated.Edges[0].Node.ID)
	assert.Equal(t, "X12-990-IN - X12 990 Response to Load Tender", hydrated.Edges[0].Node.Label)
}

func TestSelectOptions_EDIDocumentTypeRejectsBlankID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	resolver := &queryResolver{&Resolver{
		ediService: ediservice.New(ediservice.Params{
			Logger:           zap.NewNop(),
			DocumentTypeRepo: mocks.NewMockEDIDocumentTypeRepository(t),
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	_, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEDIDocumentType,
		Ids:      []string{" "},
	})
	require.Error(t, err)
}

func TestSelectOptions_EDITransactionSetCatalogIDs(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	tender := &edi.EDITransactionSet{
		ID:             pulid.ID("edits_x12_204"),
		Standard:       edi.EDIStandardX12,
		Code:           edi.TransactionSet204,
		Name:           "Motor Carrier Load Tender",
		Description:    "Outbound load tender document.",
		DefaultVersion: "004010",
		Status:         edi.DocumentStatusActive,
		CreatedAt:      1780415883,
	}
	response := &edi.EDITransactionSet{
		ID:             pulid.ID("edits_x12_990"),
		Standard:       edi.EDIStandardX12,
		Code:           edi.TransactionSet990,
		Name:           "Response to Load Tender",
		DefaultVersion: "004010",
		Status:         edi.DocumentStatusActive,
		CreatedAt:      1780415884,
	}
	repo := mocks.NewMockEDITransactionSetRepository(t)
	repo.EXPECT().
		SelectTransactionSetOptions(mock.Anything, mock.MatchedBy(func(req *repositories.EDITransactionSetSelectOptionsRequest) bool {
			return req.SelectQueryRequest.TenantInfo.OrgID == orgID &&
				req.SelectQueryRequest.TenantInfo.BuID == buID &&
				len(req.IDs) == 0 &&
				req.Status == edi.DocumentStatusActive &&
				req.Standard == ""
		})).
		Return(&pagination.ListResult[*edi.EDITransactionSet]{
			Items: []*edi.EDITransactionSet{tender, response},
			Total: 2,
		}, nil).
		Once()
	repo.EXPECT().
		SelectTransactionSetOptions(mock.Anything, mock.MatchedBy(func(req *repositories.EDITransactionSetSelectOptionsRequest) bool {
			return len(req.IDs) == 2 &&
				req.IDs[0] == response.ID &&
				req.IDs[1] == tender.ID &&
				req.SelectQueryRequest.Pagination.Limit >= 2
		})).
		Return(&pagination.ListResult[*edi.EDITransactionSet]{
			Items: []*edi.EDITransactionSet{tender, response},
			Total: 2,
		}, nil).
		Once()

	resolver := &queryResolver{&Resolver{
		ediService: ediservice.New(ediservice.Params{
			Logger:             zap.NewNop(),
			TransactionSetRepo: repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(orgID, buID, userID),
	)

	listed, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEDITransactionSet,
		Filters:  map[string]any{"status": "Active"},
	})
	require.NoError(t, err)
	require.Len(t, listed.Edges, 2)
	assert.Equal(t, "edits_x12_204", listed.Edges[0].Node.ID)
	assert.Equal(t, "204 - Motor Carrier Load Tender", listed.Edges[0].Node.Label)
	require.NotNil(t, listed.Edges[0].Node.Description)
	assert.Equal(t, "Outbound load tender document.", *listed.Edges[0].Node.Description)
	assert.Equal(t, "204", listed.Edges[0].Node.Meta["code"])
	assert.Equal(t, "X12", listed.Edges[0].Node.Meta["standard"])
	assert.Equal(t, "004010", listed.Edges[0].Node.Meta["defaultVersion"])
	assert.Equal(t, "Active", listed.Edges[0].Node.Meta["status"])
	assert.Nil(t, listed.Edges[1].Node.Description)

	cursor, err := pagination.DecodeCursor(listed.Edges[1].Cursor)
	require.NoError(t, err)
	assert.Equal(t, response.ID, cursor.ID)
	assert.Equal(t, response.CreatedAt, cursor.CreatedAt)

	first := 1
	hydrated, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceEDITransactionSet,
		First:    &first,
		Ids:      []string{"edits_x12_990", "edits_x12_204"},
	})
	require.NoError(t, err)
	require.Len(t, hydrated.Edges, 2)
	assert.Equal(t, "edits_x12_990", hydrated.Edges[0].Node.ID)
	assert.Equal(t, "edits_x12_204", hydrated.Edges[1].Node.ID)
	assert.False(t, hydrated.PageInfo.HasNextPage)
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

func iftaFuelTypeResolver() *queryResolver {
	return &queryResolver{&Resolver{}}
}

func iftaFuelTypeSelectOptionIDs(connection *gqlmodel.SelectOptionConnection) []string {
	ids := make([]string, 0, len(connection.Edges))
	for _, edge := range connection.Edges {
		ids = append(ids, edge.Node.ID)
	}
	return ids
}

func TestSelectOptions_IFTAFuelTypeListsEveryFuelWithReportingMeta(t *testing.T) {
	t.Parallel()

	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	result, err := iftaFuelTypeResolver().SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
	})
	require.NoError(t, err)

	assert.Equal(
		t,
		[]string{
			"Diesel", "Gasoline", "Gasohol", "Propane", "CNG", "LNG", "Ethanol", "Methanol",
			"E85", "M85", "A55", "Biodiesel", "Electricity", "Hydrogen", "DEF", "Reefer", "Other",
		},
		iftaFuelTypeSelectOptionIDs(result),
	)
	require.NotNil(t, result.TotalCount)
	assert.Equal(t, 17, *result.TotalCount)
	assert.False(t, result.PageInfo.HasNextPage)

	diesel := result.Edges[0].Node
	assert.Equal(t, domaintypes.IFTAFuelTypeDiesel.Label(), diesel.Label)
	assert.Equal(t, "Reported on the quarterly IFTA return", *diesel.Description)
	assert.Equal(t, true, diesel.Meta["countsForIfta"])
	assert.Equal(t, false, diesel.Meta["gaseous"])

	cng := result.Edges[4].Node
	assert.Equal(t, "CNG", cng.ID)
	assert.Equal(t, "Reported on the quarterly IFTA return in gallon equivalents", *cng.Description)
	assert.Equal(t, true, cng.Meta["gaseous"])

	reefer := result.Edges[15].Node
	assert.Equal(t, "Reefer", reefer.ID)
	assert.Equal(t, "Not reported on the quarterly IFTA return", *reefer.Description)
	assert.Equal(t, false, reefer.Meta["countsForIfta"])

	cursor, err := pagination.DecodeCursor(result.Edges[0].Cursor)
	require.NoError(t, err)
	assert.Equal(t, pulid.ID("Diesel"), cursor.ID)
}

func TestSelectOptions_IFTAFuelTypeSearchesValueAndLabel(t *testing.T) {
	t.Parallel()

	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)
	resolver := iftaFuelTypeResolver()

	tests := []struct {
		query string
		want  []string
	}{
		{query: "natural", want: []string{"CNG", "LNG"}},
		{query: "cng", want: []string{"CNG"}},
		{query: "e85", want: []string{"E85"}},
		{query: "  DIESEL  ", want: []string{"Diesel", "Biodiesel", "DEF"}},
		{query: "kerosene", want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			t.Parallel()

			query := tt.query
			result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
				Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
				Query:    &query,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.want, iftaFuelTypeSelectOptionIDs(result))
			require.NotNil(t, result.TotalCount)
			assert.Equal(t, len(tt.want), *result.TotalCount)
		})
	}
}

func TestSelectOptions_IFTAFuelTypePagesAgainstTheFullTotal(t *testing.T) {
	t.Parallel()

	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)
	resolver := iftaFuelTypeResolver()
	first := 5

	firstPageOffset := 0
	firstPage, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
		First:    &first,
		Offset:   &firstPageOffset,
	})
	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{"Diesel", "Gasoline", "Gasohol", "Propane", "CNG"},
		iftaFuelTypeSelectOptionIDs(firstPage),
	)
	require.NotNil(t, firstPage.TotalCount)
	assert.Equal(t, 17, *firstPage.TotalCount)
	assert.True(t, firstPage.PageInfo.HasNextPage)

	lastPageOffset := 15
	lastPage, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
		First:    &first,
		Offset:   &lastPageOffset,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Reefer", "Other"}, iftaFuelTypeSelectOptionIDs(lastPage))
	assert.False(t, lastPage.PageInfo.HasNextPage)

	pastEndOffset := 40
	pastEnd, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
		First:    &first,
		Offset:   &pastEndOffset,
	})
	require.NoError(t, err)
	assert.Empty(t, pastEnd.Edges)
	assert.False(t, pastEnd.PageInfo.HasNextPage)
}

func TestSelectOptions_IFTAFuelTypeCountsForIftaFilterDropsNonReportedFuels(t *testing.T) {
	t.Parallel()

	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)

	result, err := iftaFuelTypeResolver().SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
		Filters:  map[string]any{"countsForIfta": true},
	})
	require.NoError(t, err)

	ids := iftaFuelTypeSelectOptionIDs(result)
	assert.Len(t, ids, 14)
	assert.NotContains(t, ids, "DEF")
	assert.NotContains(t, ids, "Reefer")
	assert.NotContains(t, ids, "Other")
	require.NotNil(t, result.TotalCount)
	assert.Equal(t, 14, *result.TotalCount)
}

func TestSelectOptions_IFTAFuelTypeByIDsKeepsRequestOrderAndDropsUnknownFuels(t *testing.T) {
	t.Parallel()

	ctx := gqlctx.WithAuthContext(
		t.Context(),
		testGraphQLAuthContext(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_")),
	)
	resolver := iftaFuelTypeResolver()

	result, err := resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
		Ids:      []string{"Reefer", "Kerosene", "Diesel"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Reefer", "Diesel"}, iftaFuelTypeSelectOptionIDs(result))
	require.NotNil(t, result.TotalCount)
	assert.Equal(t, 2, *result.TotalCount)

	_, err = resolver.SelectOptions(ctx, gqlmodel.SelectOptionsInput{
		Resource: gqlmodel.SelectOptionResourceIFTAFuelType,
		Ids:      []string{"  "},
	})
	require.Error(t, err)
}
