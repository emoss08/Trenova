package carriersettlementresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	tenantdomain "github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/settlement/app/carriersettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

func (r *MutationResolver) carrierSettlementAction(
	ctx context.Context,
	settlementID string,
	op permission.Operation,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceCarrierSettlement, op)
	if err != nil {
		return nil, pulid.Nil, err
	}
	id, err := pulid.MustParse(settlementID)
	if err != nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"settlementId",
			errortypes.ErrInvalid,
			"Invalid settlement",
		)
	}
	return authCtx, id, nil
}

func (r *MutationResolver) carrierInvoiceMatchAction(
	ctx context.Context,
	matchID string,
	op permission.Operation,
) (*authctx.AuthContext, pulid.ID, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceCarrierInvoiceMatch, op)
	if err != nil {
		return nil, pulid.Nil, err
	}
	id, err := pulid.MustParse(matchID)
	if err != nil {
		return nil, pulid.Nil, errortypes.NewValidationError(
			"matchId",
			errortypes.ErrInvalid,
			"Invalid match",
		)
	}
	return authCtx, id, nil
}

func listFilterOptions(
	tenantInfo pagination.TenantInfo,
	limit, offset *int,
) *pagination.QueryOptions {
	filter := &pagination.QueryOptions{TenantInfo: tenantInfo}
	if limit != nil {
		filter.Pagination.Limit = *limit
	}
	if offset != nil {
		filter.Pagination.Offset = *offset
	}
	return filter
}

func carrierSettlementConnectionToModel(
	result *pagination.CursorListResult[*carriersettlement.CarrierSettlement],
) (*gqlmodel.CarrierSettlementConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *carriersettlement.CarrierSettlement, cursor string) *gqlmodel.CarrierSettlementEdge {
			return &gqlmodel.CarrierSettlementEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.CarrierSettlementEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.CarrierSettlementConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func carrierSettlementBatchConnectionToModel(
	result *pagination.CursorListResult[*carriersettlement.CarrierSettlementBatch],
) (*gqlmodel.CarrierSettlementBatchConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *carriersettlement.CarrierSettlementBatch, cursor string) *gqlmodel.CarrierSettlementBatchEdge {
			return &gqlmodel.CarrierSettlementBatchEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.CarrierSettlementBatchEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.CarrierSettlementBatchConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func carrierCostEventConnectionToModel(
	result *pagination.CursorListResult[*carriersettlement.CostEvent],
) (*gqlmodel.CarrierCostEventConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *carriersettlement.CostEvent, cursor string) *gqlmodel.CarrierCostEventEdge {
			return &gqlmodel.CarrierCostEventEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.CarrierCostEventEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.CarrierCostEventConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func carrierSettlementControlFromInput(
	input *gqlmodel.UpdateCarrierSettlementControlInput,
	tenantInfo pagination.TenantInfo,
) (*tenantdomain.CarrierSettlementControl, error) {
	apAccountID, err := gqlid.OptionalPulid(input.DefaultApAccountID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"defaultApAccountId",
			errortypes.ErrInvalid,
			"Invalid AP account",
		)
	}
	expenseAccountID, err := gqlid.OptionalPulid(input.DefaultPurchasedTransportationAccountID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"defaultPurchasedTransportationAccountId",
			errortypes.ErrInvalid,
			"Invalid purchased transportation account",
		)
	}
	return &tenantdomain.CarrierSettlementControl{
		OrganizationID:            tenantInfo.OrgID,
		BusinessUnitID:            tenantInfo.BuID,
		Version:                   int64(input.Version),
		PayTrigger:                input.PayTrigger,
		PayPeriodFrequency:        input.PayPeriodFrequency,
		PeriodEndDayOfWeek:        input.PeriodEndDayOfWeek,
		PayDelayDays:              input.PayDelayDays,
		AutoGenerateBatches:       input.AutoGenerateBatches,
		AutoPostOnApprove:         input.AutoPostOnApprove,
		VarianceToleranceMinor:    int64(input.VarianceToleranceMinor),
		AutoMatchInboundInvoices:  input.AutoMatchInboundInvoices,
		AutoAcceptWithinTolerance: input.AutoAcceptWithinTolerance,
		HoldUntilInvoiceMatched: typeutils.ValueOrZero(
			input.HoldUntilInvoiceMatched,
		),
		DefaultAPAccountID:                      apAccountID,
		DefaultPurchasedTransportationAccountID: expenseAccountID,
	}, nil
}

func createCarrierInvoiceMatchRequest(
	input *gqlmodel.CreateCarrierInvoiceMatchInput,
	tenantInfo pagination.TenantInfo,
) (*carriersettlementservice.CreateMatchRequest, error) {
	ediInvoiceID, err := gqlid.OptionalPulid(input.EDICarrierInvoiceID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"ediCarrierInvoiceId",
			errortypes.ErrInvalid,
			"Invalid EDI carrier invoice",
		)
	}
	extractionID, err := gqlid.OptionalPulid(input.DocumentAiExtractionID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentAiExtractionId",
			errortypes.ErrInvalid,
			"Invalid document AI extraction",
		)
	}
	carrierID, err := gqlid.OptionalPulid(input.CarrierID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"carrierId",
			errortypes.ErrInvalid,
			"Invalid carrier",
		)
	}
	assignmentID, err := gqlid.OptionalPulid(input.CarrierAssignmentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"carrierAssignmentId",
			errortypes.ErrInvalid,
			"Invalid carrier assignment",
		)
	}
	shipmentID, err := gqlid.OptionalPulid(input.ShipmentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"Invalid shipment",
		)
	}

	req := &carriersettlementservice.CreateMatchRequest{
		TenantInfo:             tenantInfo,
		EDICarrierInvoiceID:    ediInvoiceID,
		DocumentAIExtractionID: extractionID,
		CarrierAssignmentID:    assignmentID,
		InvoiceNumber:          gqlscalar.StringValue(input.InvoiceNumber),
		ProNumber:              gqlscalar.StringValue(input.ProNumber),
	}
	if carrierID != nil {
		req.CarrierID = *carrierID
	}
	if shipmentID != nil {
		req.ShipmentID = *shipmentID
	}
	if input.InvoiceTotalMinor != nil {
		req.InvoiceTotalMinor = int64(*input.InvoiceTotalMinor)
	}
	return req, nil
}
