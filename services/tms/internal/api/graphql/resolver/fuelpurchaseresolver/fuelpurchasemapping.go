package fuelpurchaseresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

func fuelCardCursorConnectionToModel(
	result *pagination.CursorListResult[*fuelpurchase.FuelCard],
) (*gqlmodel.FuelCardConnection, error) {
	edges, err := base.EntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *fuelpurchase.FuelCard, cursor string) *gqlmodel.FuelCardEdge {
			return &gqlmodel.FuelCardEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FuelCardConnection{
		Edges: edges,
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.FuelCardEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func fuelPurchaseCursorConnectionToModel(
	result *pagination.CursorListResult[*fuelpurchase.FuelPurchase],
) (*gqlmodel.FuelPurchaseConnection, error) {
	edges, err := base.EntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *fuelpurchase.FuelPurchase, cursor string) *gqlmodel.FuelPurchaseEdge {
			return &gqlmodel.FuelPurchaseEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FuelPurchaseConnection{
		Edges: edges,
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.FuelPurchaseEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func fuelImportRowCursorConnectionToModel(
	result *pagination.CursorListResult[*fuelpurchase.ImportRow],
) (*gqlmodel.FuelPurchaseImportRowConnection, error) {
	edges, err := base.EntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(node *fuelpurchase.ImportRow, cursor string) *gqlmodel.FuelPurchaseImportRowEdge {
			return &gqlmodel.FuelPurchaseImportRowEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FuelPurchaseImportRowConnection{
		Edges: edges,
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.FuelPurchaseImportRowEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}

func fuelCardsRequestFromGraphQL(
	connection *base.GqlDataTableConnection,
	input gqlmodel.FuelCardsInput,
) (*repositories.ListFuelCardsRequest, error) {
	tractorID, err := base.OptionalScopedID("assignedTractorId", input.AssignedTractorID)
	if err != nil {
		return nil, err
	}

	workerID, err := base.OptionalScopedID("assignedWorkerId", input.AssignedWorkerID)
	if err != nil {
		return nil, err
	}

	req := &repositories.ListFuelCardsRequest{
		Filter:             connection.Filter,
		Cursor:             connection.Cursor,
		AssignedTractorID:  tractorID,
		AssignedWorkerID:   workerID,
		IncludeAssignments: true,
	}
	if input.Provider != nil {
		req.Provider = *input.Provider
	}
	if input.Status != nil {
		req.Statuses = []fuelpurchase.CardStatus{*input.Status}
	}
	if input.UnassignedOnly != nil {
		req.UnassignedOnly = *input.UnassignedOnly
	}
	if input.DiscoveredOnly != nil {
		req.DiscoveredOnly = *input.DiscoveredOnly
	}

	return req, nil
}

func fuelPurchasesRequestFromGraphQL(
	connection *base.GqlDataTableConnection,
	input gqlmodel.FuelPurchasesInput,
) (*repositories.ListFuelPurchasesRequest, error) {
	tractorID, err := base.OptionalScopedID("tractorId", input.TractorID)
	if err != nil {
		return nil, err
	}

	fuelCardID, err := base.OptionalScopedID("fuelCardId", input.FuelCardID)
	if err != nil {
		return nil, err
	}

	jurisdictionID, err := base.OptionalScopedID("jurisdictionId", input.JurisdictionID)
	if err != nil {
		return nil, err
	}

	return &repositories.ListFuelPurchasesRequest{
		Filter:              connection.Filter,
		Cursor:              connection.Cursor,
		TractorID:           tractorID,
		FuelCardID:          fuelCardID,
		JurisdictionID:      jurisdictionID,
		FuelTypes:           input.FuelTypes,
		Sources:             input.Sources,
		TaxPaid:             input.TaxPaid,
		From:                base.Int64Value(input.From),
		To:                  base.Int64Value(input.To),
		IncludeTractor:      true,
		IncludeWorker:       true,
		IncludeJurisdiction: true,
		IncludeFuelCard:     true,
	}, nil
}

func fuelCardFromInput(
	input gqlmodel.FuelCardInput,
	tenant pagination.TenantInfo,
) (*fuelpurchase.FuelCard, error) {
	card := &fuelpurchase.FuelCard{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
	if err := applyFuelCardInput(card, input); err != nil {
		return nil, err
	}

	return card, nil
}

func applyFuelCardInput(card *fuelpurchase.FuelCard, input gqlmodel.FuelCardInput) error {
	workerID, err := base.OptionalScopedID("assignedWorkerId", input.AssignedWorkerID)
	if err != nil {
		return err
	}

	tractorID, err := base.OptionalScopedID("assignedTractorId", input.AssignedTractorID)
	if err != nil {
		return err
	}

	card.Provider = input.Provider
	card.LastFour = input.LastFour
	card.Label = input.Label
	card.ExternalCardID = base.StringValue(input.ExternalCardID)
	card.AssignedWorkerID = base.IDValuePtr(workerID)
	card.AssignedTractorID = base.IDValuePtr(tractorID)
	card.ExpiresAt = base.Int64Ptr(input.ExpiresAt)
	card.Notes = base.StringValue(input.Notes)
	if input.Status != nil {
		card.Status = *input.Status
	}

	return nil
}

func fuelPurchaseFromInput(
	input gqlmodel.FuelPurchaseInput,
	tenant pagination.TenantInfo,
) (*fuelpurchase.FuelPurchase, error) {
	purchase := &fuelpurchase.FuelPurchase{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
	if err := applyFuelPurchaseInput(purchase, input); err != nil {
		return nil, err
	}

	return purchase, nil
}

func applyFuelPurchaseInput(
	purchase *fuelpurchase.FuelPurchase,
	input gqlmodel.FuelPurchaseInput,
) error {
	tractorID, err := base.RequiredID("tractorId", input.TractorID)
	if err != nil {
		return err
	}

	jurisdictionID, err := base.RequiredID("jurisdictionId", input.JurisdictionID)
	if err != nil {
		return err
	}

	workerID, err := base.OptionalScopedID("workerId", input.WorkerID)
	if err != nil {
		return err
	}

	fuelCardID, err := base.OptionalScopedID("fuelCardId", input.FuelCardID)
	if err != nil {
		return err
	}

	quantity, err := base.DecimalFromString(input.Quantity, "quantity")
	if err != nil {
		return err
	}

	unitPrice, err := base.NullDecimalFromStringPtr(input.UnitPrice, "unitPrice")
	if err != nil {
		return err
	}

	totalAmount, err := base.DecimalFromString(input.TotalAmount, "totalAmount")
	if err != nil {
		return err
	}

	purchase.TractorID = tractorID
	purchase.JurisdictionID = jurisdictionID
	purchase.WorkerID = base.IDValuePtr(workerID)
	purchase.FuelCardID = base.IDValuePtr(fuelCardID)
	purchase.CardLastFour = base.StringValue(input.CardLastFour)
	purchase.PurchasedAt = base.Int64Value(&input.PurchasedAt)
	purchase.Vendor = base.StringValue(input.Vendor)
	purchase.VendorCity = base.StringValue(input.VendorCity)
	purchase.FuelType = input.FuelType
	purchase.Quantity = quantity
	purchase.UnitPrice = unitPrice
	purchase.TotalAmountMinor = money.MinorUnits(totalAmount)
	purchase.Odometer = base.Int64Ptr(input.Odometer)
	purchase.TransactionReference = base.StringValue(input.TransactionReference)
	purchase.Notes = base.StringValue(input.Notes)
	purchase.TaxPaid = true
	if input.TaxPaid != nil {
		purchase.TaxPaid = *input.TaxPaid
	}
	if input.QuantityUnit != nil {
		purchase.QuantityUnit = *input.QuantityUnit
	}
	if input.CurrencyCode != nil {
		purchase.CurrencyCode = *input.CurrencyCode
	}

	return nil
}

func fuelImportMappingFromGraphQL(raw map[string]any) (map[string]int, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	mapping := make(map[string]int, len(raw))
	for name, value := range raw {
		index, ok := columnIndexFromAny(value)
		if !ok {
			return nil, errortypes.NewValidationError(
				"mapping",
				errortypes.ErrInvalid,
				"Column index for {0} must be a whole number", name,
			)
		}
		mapping[name] = index
	}

	return mapping, nil
}

func columnIndexFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	default:
		return 0, false
	}
}

func fuelImportParsedToModel(parsed *fuelpurchase.FuelPurchase) *gqlmodel.FuelPurchaseImportParsed {
	if parsed == nil {
		return nil
	}

	model := &gqlmodel.FuelPurchaseImportParsed{
		Vendor:               base.NonEmptyPtr(parsed.Vendor),
		VendorCity:           base.NonEmptyPtr(parsed.VendorCity),
		Quantity:             base.NonEmptyPtr(parsed.Quantity.String()),
		Gallons:              base.NonEmptyPtr(parsed.Gallons.String()),
		UnitPrice:            base.NullDecimalToStringPtr(parsed.UnitPrice),
		CurrencyCode:         base.NonEmptyPtr(parsed.CurrencyCode),
		TransactionReference: base.NonEmptyPtr(parsed.TransactionReference),
		CardLastFour:         base.NonEmptyPtr(parsed.CardLastFour),
		Odometer:             base.IntPtr(parsed.Odometer),
	}
	if parsed.PurchasedAt > 0 {
		purchasedAt := int(parsed.PurchasedAt)
		model.PurchasedAt = &purchasedAt
	}
	if parsed.FuelType != "" {
		fuelType := parsed.FuelType
		model.FuelType = &fuelType
	}
	if parsed.QuantityUnit != "" {
		unit := parsed.QuantityUnit
		model.QuantityUnit = &unit
	}
	if parsed.TotalAmountMinor != 0 {
		model.TotalAmount = base.NonEmptyPtr(
			money.DecimalFromMinor(parsed.TotalAmountMinor).StringFixed(2),
		)
	}

	return model
}

func fuelImportRowsRequest(
	ctx context.Context,
	batch *fuelpurchase.ImportBatch,
	input *gqlmodel.FuelPurchaseImportRowsInput,
	tenant pagination.TenantInfo,
) (*repositories.ListImportRowsRequest, error) {
	pageInput := base.GqlCursorPageInput{}
	var statuses []fuelpurchase.ImportRowStatus
	if input != nil {
		pageInput.First = input.First
		pageInput.After = input.After
		statuses = input.Statuses
	}

	page, err := base.EntityCursorPageFromGraphQL(ctx, pageInput)
	if err != nil {
		return nil, err
	}

	return &repositories.ListImportRowsRequest{
		BatchID:    batch.ID,
		TenantInfo: tenant,
		Filter: base.QueryOptionsFromGraphQL(&base.GqlListOptions{
			TenantInfo: tenant,
			Limit:      page.Cursor.Limit,
		}),
		Cursor:   page.Cursor,
		Statuses: statuses,
	}, nil
}

func stringMapToAnyMap(values map[string]string) map[string]any {
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}

	return out
}

func intMapToAnyMap(values map[string]int) map[string]any {
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}

	return out
}

func cancelCardRequestFromInput(
	input gqlmodel.CancelFuelCardInput,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) (*fuelpurchaseservice.CancelCardRequest, error) {
	id, err := base.RequiredID("id", input.ID)
	if err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.CancelCardRequest{
		TenantInfo: tenant,
		ID:         id,
		Version:    int64(input.Version),
		Reason:     input.Reason,
		UserID:     userID,
	}, nil
}

func assignCardRequestFromInput(
	input gqlmodel.AssignFuelCardInput,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) (*fuelpurchaseservice.AssignCardRequest, error) {
	id, err := base.RequiredID("id", input.ID)
	if err != nil {
		return nil, err
	}

	tractorID, err := base.OptionalScopedID("assignedTractorId", input.AssignedTractorID)
	if err != nil {
		return nil, err
	}

	workerID, err := base.OptionalScopedID("assignedWorkerId", input.AssignedWorkerID)
	if err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.AssignCardRequest{
		TenantInfo:        tenant,
		ID:                id,
		Version:           int64(input.Version),
		AssignedTractorID: base.IDValuePtr(tractorID),
		AssignedWorkerID:  base.IDValuePtr(workerID),
		UserID:            userID,
	}, nil
}

func fuelCardSyncResultToModel(
	result *fuelpurchaseservice.SyncFeedResult,
) *gqlmodel.FuelCardSyncResult {
	if result == nil {
		return nil
	}

	model := &gqlmodel.FuelCardSyncResult{
		Provider:        result.Provider,
		Fetched:         result.Fetched,
		Committed:       result.Committed,
		Queued:          result.Queued,
		AlreadyImported: result.AlreadyImported,
		CardsDiscovered: result.CardsDiscovered,
	}
	if !result.BatchID.IsNil() {
		batchID := result.BatchID.String()
		model.BatchID = &batchID
	}

	return model
}

func loadFuelCard(ctx context.Context, id *pulid.ID) (*fuelpurchase.FuelCard, error) {
	if id == nil || id.IsNil() {
		return nil, nil
	}

	l, ok := loaders.FromContext(ctx)
	if !ok || l == nil {
		return nil, nil
	}

	return l.FuelCardByID.Load(ctx, id.String())
}

func epochPtr(value int64) *int {
	if value == 0 {
		return nil
	}
	converted := int(value)
	return &converted
}

func fuelImportsRequestFromGraphQL(
	connection *base.GqlDataTableConnection,
	input gqlmodel.FuelPurchaseImportsInput,
) *repositories.ListImportBatchesRequest {
	req := &repositories.ListImportBatchesRequest{
		Filter:             connection.Filter,
		Cursor:             connection.Cursor,
		Statuses:           input.Statuses,
		IncludeDefaultCard: true,
	}
	if input.Origin != nil {
		req.Origin = *input.Origin
	}
	if input.Provider != nil {
		req.Provider = *input.Provider
	}
	if input.HeldRowsOnly != nil {
		req.HeldRowsOnly = *input.HeldRowsOnly
	}

	return req
}

func fuelImportCursorConnectionToModel(
	result *pagination.CursorListResult[*fuelpurchase.ImportBatch],
) (*gqlmodel.FuelPurchaseImportBatchConnection, error) {
	edges, err := base.EntityCursorEdges(
		result.Items,
		result.CursorSort,
		result,
		func(
			node *fuelpurchase.ImportBatch,
			cursor string,
		) *gqlmodel.FuelPurchaseImportBatchEdge {
			return &gqlmodel.FuelPurchaseImportBatchEdge{Node: node, Cursor: cursor}
		},
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.FuelPurchaseImportBatchConnection{
		Edges: edges,
		PageInfo: base.PageInfo(
			result.HasNextPage,
			base.LastEdgeCursor(
				edges,
				func(edge *gqlmodel.FuelPurchaseImportBatchEdge) string { return edge.Cursor },
			),
		),
		TotalCount: result.TotalCount,
	}, nil
}
