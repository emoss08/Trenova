package shipmentresolver

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	invoicedomain "github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	shipmentdomain "github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	carriersettlementdomain "github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	driversettlementdomain "github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listCapturingShipmentService struct {
	shipmentserviceports.ShipmentService

	requests []*shipmentports.ListShipmentsRequest
}

func (s *listCapturingShipmentService) List(
	_ context.Context,
	req *shipmentports.ListShipmentsRequest,
) (*pagination.CursorListResult[*shipmentdomain.Shipment], error) {
	s.requests = append(s.requests, req)
	return &pagination.CursorListResult[*shipmentdomain.Shipment]{}, nil
}

func shipmentListContext(t *testing.T) context.Context {
	t.Helper()

	ctx := graphql.WithOperationContext(t.Context(), &graphql.OperationContext{})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{})
	userID := pulid.MustNew("usr_")

	return gqlctx.WithAuthContext(ctx, &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func newShipmentListResolver(service shipmentserviceports.ShipmentService) *QueryResolver {
	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceShipment.String() + "|" + string(permission.OpRead): true,
	}}
	return &QueryResolver{&Deps{
		Core:            &base.Core{PermissionEngine: engine},
		ShipmentService: service,
	}}
}

func TestShipments_TractorIDsReachTheListOptions(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}
	tractorA := pulid.MustNew("tr_")
	tractorB := pulid.MustNew("tr_")
	first := 5

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{
			First:      &first,
			TractorIds: []string{tractorA.String(), tractorB.String()},
		},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	assert.Equal(
		t,
		[]pulid.ID{tractorA, tractorB},
		service.requests[0].ShipmentOptions.TractorIDs,
	)
}

func TestShipments_TrailerAndWorkerIDsReachTheListOptions(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}
	trailerID := pulid.MustNew("trl_")
	workerID := pulid.MustNew("wrk_")

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{
			TrailerIds: []string{trailerID.String()},
			WorkerIds:  []string{workerID.String()},
		},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	options := service.requests[0].ShipmentOptions
	assert.Equal(t, []pulid.ID{trailerID}, options.TrailerIDs)
	assert.Equal(t, []pulid.ID{workerID}, options.WorkerIDs)
	assert.Empty(t, options.TractorIDs)
}

func TestShipments_NoTractorIDsLeavesTheFilterOff(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	assert.Empty(t, service.requests[0].ShipmentOptions.TractorIDs)
}

func TestShipments_RejectsAMalformedTractorID(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{TractorIds: []string{"tractor-one"}},
	)
	require.Error(t, err)
	assert.Empty(t, service.requests)
}

func TestShipments_StopLocationIDsReachTheListOptions(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}
	locationA := pulid.MustNew("loc_")
	locationB := pulid.MustNew("loc_")

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{
			StopLocationIds: []string{locationA.String(), locationB.String()},
		},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	options := service.requests[0].ShipmentOptions
	assert.Equal(t, []pulid.ID{locationA, locationB}, options.StopLocationIDs)
	assert.Empty(t, options.TractorIDs)
}

func TestShipments_RejectsAMalformedStopLocationID(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{StopLocationIds: []string{"dallas-yard"}},
	)
	require.Error(t, err)
	assert.Empty(t, service.requests)
}

func TestShipments_CarrierIDsReachTheListOptions(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}
	carrierID := pulid.MustNew("car_")

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{CarrierIds: []string{carrierID.String()}},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	options := service.requests[0].ShipmentOptions
	assert.Equal(t, []pulid.ID{carrierID}, options.CarrierIDs)
	assert.Empty(t, options.TractorIDs)
}

func TestShipments_RejectsAMalformedCarrierID(t *testing.T) {
	t.Parallel()

	service := &listCapturingShipmentService{}

	_, err := newShipmentListResolver(service).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{CarrierIds: []string{"acme-freight"}},
	)
	require.Error(t, err)
	assert.Empty(t, service.requests)
}

type invoicesByIDService struct {
	invoiceports.InvoiceService

	invoices []*invoicedomain.Invoice
	requests []invoiceports.GetInvoicesByIDsRequest
}

func (s *invoicesByIDService) GetByIDs(
	_ context.Context,
	req invoiceports.GetInvoicesByIDsRequest,
) ([]*invoicedomain.Invoice, error) {
	s.requests = append(s.requests, req)
	return s.invoices, nil
}

func newInvoiceShipmentsResolver(
	service shipmentserviceports.ShipmentService,
	invoices invoiceports.InvoiceService,
	mayReadInvoices bool,
) *QueryResolver {
	engine := &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
		permission.ResourceShipment.String() + "|" + string(permission.OpRead): true,
		permission.ResourceInvoice.String() + "|" + string(permission.OpRead):  mayReadInvoices,
	}}
	return &QueryResolver{&Deps{
		Core:            &base.Core{PermissionEngine: engine},
		ShipmentService: service,
		InvoiceService:  invoices,
	}}
}

func TestShipments_InvoiceIDsListTheShipmentsTheInvoicesBill(t *testing.T) {
	t.Parallel()

	header := pulid.MustNew("shp_")
	onALine := pulid.MustNew("shp_")
	consolidated := pulid.MustNew("inv_")
	single := pulid.MustNew("inv_")
	invoices := &invoicesByIDService{invoices: []*invoicedomain.Invoice{
		{
			ID: consolidated,
			Lines: []*invoicedomain.InvoiceLine{
				{ShipmentID: onALine},
				{ShipmentID: header},
				{ShipmentID: onALine},
			},
		},
		{ID: single, ShipmentID: header, Lines: []*invoicedomain.InvoiceLine{{ShipmentID: header}}},
	}}
	service := &listCapturingShipmentService{}

	_, err := newInvoiceShipmentsResolver(service, invoices, true).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{InvoiceIds: []string{consolidated.String(), single.String()}},
	)
	require.NoError(t, err)
	require.Len(t, invoices.requests, 1)
	assert.Equal(t, []pulid.ID{consolidated, single}, invoices.requests[0].InvoiceIDs)
	require.Len(t, service.requests, 1)
	assert.Equal(t, []pulid.ID{onALine, header}, service.requests[0].ShipmentOptions.IDs)
}

func TestShipments_InvoiceIDsThatBillNoShipmentListNone(t *testing.T) {
	t.Parallel()

	memo := pulid.MustNew("inv_")
	invoices := &invoicesByIDService{invoices: []*invoicedomain.Invoice{
		{ID: memo, Lines: []*invoicedomain.InvoiceLine{{Description: "Goodwill credit"}}},
	}}
	service := &listCapturingShipmentService{}

	result, err := newInvoiceShipmentsResolver(service, invoices, true).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{InvoiceIds: []string{memo.String()}},
	)
	require.NoError(t, err)
	assert.Empty(t, service.requests, "an empty list of shipments is not a missing filter")
	require.NotNil(t, result)
	assert.Empty(t, result.Edges)
	require.NotNil(t, result.TotalCount)
	assert.Zero(t, *result.TotalCount)
}

func TestShipments_InvoiceIDsNeedPermissionToReadInvoices(t *testing.T) {
	t.Parallel()

	invoices := &invoicesByIDService{}
	service := &listCapturingShipmentService{}

	_, err := newInvoiceShipmentsResolver(service, invoices, false).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{InvoiceIds: []string{pulid.MustNew("inv_").String()}},
	)
	require.Error(t, err)
	assert.Empty(t, invoices.requests)
	assert.Empty(t, service.requests)
}

type settlementLinesRepo struct {
	driversettlementports.DriverSettlementRepository

	settlements map[pulid.ID]*driversettlementdomain.Settlement
	requests    []driversettlementports.GetDriverSettlementByIDRequest
}

func (s *settlementLinesRepo) GetByID(
	_ context.Context,
	req driversettlementports.GetDriverSettlementByIDRequest,
) (*driversettlementdomain.Settlement, error) {
	s.requests = append(s.requests, req)
	return s.settlements[req.ID], nil
}

type carrierSettlementLinesRepo struct {
	carriersettlementports.CarrierSettlementRepository

	settlements map[pulid.ID]*carriersettlementdomain.CarrierSettlement
}

func (s *carrierSettlementLinesRepo) GetByID(
	_ context.Context,
	req carriersettlementports.GetCarrierSettlementByIDRequest,
) (*carriersettlementdomain.CarrierSettlement, error) {
	return s.settlements[req.ID], nil
}

func newSettlementShipmentsResolver(
	service shipmentserviceports.ShipmentService,
	grants map[permission.Resource]bool,
	deps Deps,
) *QueryResolver {
	granted := map[string]bool{
		permission.ResourceShipment.String() + "|" + string(permission.OpRead): true,
	}
	for resource, allowed := range grants {
		granted[resource.String()+"|"+string(permission.OpRead)] = allowed
	}
	deps.Core = &base.Core{
		PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: granted},
	}
	deps.ShipmentService = service
	return &QueryResolver{&deps}
}

func idRef(id pulid.ID) *pulid.ID {
	return &id
}

func TestShipments_DriverSettlementIDsListTheLoadsTheSettlementPays(t *testing.T) {
	t.Parallel()

	first := pulid.MustNew("shp_")
	second := pulid.MustNew("shp_")
	settlementID := pulid.MustNew("dstl_")
	repo := &settlementLinesRepo{settlements: map[pulid.ID]*driversettlementdomain.Settlement{
		settlementID: {
			ID: settlementID,
			Lines: []*driversettlementdomain.SettlementLine{
				{ShipmentID: idRef(second), AmountMinor: 100},
				{AmountMinor: -9600},
				{ShipmentID: idRef(first), AmountMinor: 100},
				{ShipmentID: idRef(second), AmountMinor: 100},
			},
		},
	}}
	service := &listCapturingShipmentService{}

	_, err := newSettlementShipmentsResolver(
		service,
		map[permission.Resource]bool{permission.ResourceDriverSettlement: true},
		Deps{DriverSettlementRepo: repo},
	).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{
			DriverSettlementIds: []string{settlementID.String(), settlementID.String()},
		},
	)
	require.NoError(t, err)
	require.Len(t, repo.requests, 1)
	assert.True(t, repo.requests[0].IncludeLines)
	require.Len(t, service.requests, 1)
	assert.Equal(t, []pulid.ID{second, first}, service.requests[0].ShipmentOptions.IDs)
}

func TestShipments_CarrierSettlementIDsListTheLoadsTheSettlementPays(t *testing.T) {
	t.Parallel()

	load := pulid.MustNew("shp_")
	settlementID := pulid.MustNew("carstl_")
	repo := &carrierSettlementLinesRepo{
		settlements: map[pulid.ID]*carriersettlementdomain.CarrierSettlement{
			settlementID: {
				ID: settlementID,
				Lines: []*carriersettlementdomain.CarrierSettlementLine{
					{ShipmentID: idRef(load), AmountMinor: 230000},
					{AmountMinor: -35000},
				},
			},
		},
	}
	service := &listCapturingShipmentService{}

	_, err := newSettlementShipmentsResolver(
		service,
		map[permission.Resource]bool{permission.ResourceCarrierSettlement: true},
		Deps{CarrierSettlementRepo: repo},
	).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{CarrierSettlementIds: []string{settlementID.String()}},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	assert.Equal(t, []pulid.ID{load}, service.requests[0].ShipmentOptions.IDs)
}

func TestShipments_ASettlementThatPaysNoLoadListsNone(t *testing.T) {
	t.Parallel()

	settlementID := pulid.MustNew("dstl_")
	repo := &settlementLinesRepo{settlements: map[pulid.ID]*driversettlementdomain.Settlement{
		settlementID: {ID: settlementID},
	}}
	service := &listCapturingShipmentService{}

	result, err := newSettlementShipmentsResolver(
		service,
		map[permission.Resource]bool{permission.ResourceDriverSettlement: true},
		Deps{DriverSettlementRepo: repo},
	).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{DriverSettlementIds: []string{settlementID.String()}},
	)
	require.NoError(t, err)
	assert.Empty(t, service.requests)
	require.NotNil(t, result.TotalCount)
	assert.Zero(t, *result.TotalCount)
}

func TestShipments_SettlementIDsNeedPermissionToReadThatKindOfSettlement(t *testing.T) {
	t.Parallel()

	driverRepo := &settlementLinesRepo{}
	service := &listCapturingShipmentService{}

	_, err := newSettlementShipmentsResolver(
		service,
		map[permission.Resource]bool{
			permission.ResourceDriverSettlement:  false,
			permission.ResourceCarrierSettlement: true,
		},
		Deps{DriverSettlementRepo: driverRepo},
	).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{DriverSettlementIds: []string{pulid.MustNew("dstl_").String()}},
	)
	require.Error(t, err)
	assert.Empty(t, driverRepo.requests)
	assert.Empty(t, service.requests)
}

func TestShipments_SeveralScopesListOnlyTheLoadsTheyShare(t *testing.T) {
	t.Parallel()

	shared := pulid.MustNew("shp_")
	invoicedOnly := pulid.MustNew("shp_")
	paidOnly := pulid.MustNew("shp_")
	invoiceID := pulid.MustNew("inv_")
	settlementID := pulid.MustNew("dstl_")
	invoices := &invoicesByIDService{invoices: []*invoicedomain.Invoice{{
		ID: invoiceID,
		Lines: []*invoicedomain.InvoiceLine{
			{ShipmentID: invoicedOnly},
			{ShipmentID: shared},
		},
	}}}
	repo := &settlementLinesRepo{settlements: map[pulid.ID]*driversettlementdomain.Settlement{
		settlementID: {
			ID: settlementID,
			Lines: []*driversettlementdomain.SettlementLine{
				{ShipmentID: idRef(paidOnly)},
				{ShipmentID: idRef(shared)},
			},
		},
	}}
	service := &listCapturingShipmentService{}

	_, err := newSettlementShipmentsResolver(
		service,
		map[permission.Resource]bool{
			permission.ResourceInvoice:          true,
			permission.ResourceDriverSettlement: true,
		},
		Deps{InvoiceService: invoices, DriverSettlementRepo: repo},
	).Shipments(
		shipmentListContext(t),
		gqlmodel.ShipmentsInput{
			InvoiceIds:          []string{invoiceID.String()},
			DriverSettlementIds: []string{settlementID.String()},
		},
	)
	require.NoError(t, err)
	require.Len(t, service.requests, 1)
	assert.Equal(t, []pulid.ID{shared}, service.requests[0].ShipmentOptions.IDs)
}
