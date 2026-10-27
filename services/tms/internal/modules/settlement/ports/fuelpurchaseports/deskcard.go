package fuelpurchaseports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type SumFuelPurchasesRequest struct {
	TenantInfo     pagination.TenantInfo      `json:"tenantInfo"`
	TractorID      pulid.ID                   `json:"tractorId"`
	FuelCardID     pulid.ID                   `json:"fuelCardId"`
	JurisdictionID pulid.ID                   `json:"jurisdictionId"`
	FuelTypes      []domaintypes.IFTAFuelType `json:"fuelTypes"`
	From           int64                      `json:"from"`
	To             int64                      `json:"to"`
}

type FuelSpendRow struct {
	CurrencyCode     string          `json:"currencyCode"`
	PurchaseCount    int             `json:"purchaseCount"`
	Gallons          decimal.Decimal `json:"gallons"`
	UntaxedGallons   decimal.Decimal `json:"untaxedGallons"`
	TotalAmountMinor int64           `json:"totalAmountMinor"`
	LastPurchasedAt  int64           `json:"lastPurchasedAt"`
}

type FuelJurisdictionRow struct {
	JurisdictionID pulid.ID        `json:"jurisdictionId"`
	Gallons        decimal.Decimal `json:"gallons"`
	PurchaseCount  int             `json:"purchaseCount"`
}

type GetPreviousFillRequest struct {
	TenantInfo pagination.TenantInfo      `json:"tenantInfo"`
	Purchase   *fuelpurchase.FuelPurchase `json:"purchase"`
}

type GetPurchaseImportRowRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	PurchaseID pulid.ID              `json:"purchaseId"`
}

type ListHeldCardRowsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CardID     pulid.ID              `json:"cardId"`
	Limit      int                   `json:"limit"`
}

type ListRecentCardPurchasesRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	CardID     pulid.ID              `json:"cardId"`
	Limit      int                   `json:"limit"`
}

type FuelDeskCardReader interface {
	SumPurchases(ctx context.Context, req *SumFuelPurchasesRequest) ([]*FuelSpendRow, error)
	SumGallonsByJurisdiction(
		ctx context.Context,
		req *SumFuelPurchasesRequest,
	) ([]*FuelJurisdictionRow, error)
	GetPreviousFill(
		ctx context.Context,
		req *GetPreviousFillRequest,
	) (*fuelpurchase.FuelPurchase, error)
	GetPurchaseImportRow(
		ctx context.Context,
		req *GetPurchaseImportRowRequest,
	) (*fuelpurchase.ImportRow, error)
	ListHeldCardRows(
		ctx context.Context,
		req *ListHeldCardRowsRequest,
	) ([]*fuelpurchase.ImportRow, error)
	ListRecentCardPurchases(
		ctx context.Context,
		req *ListRecentCardPurchasesRequest,
	) ([]*fuelpurchase.FuelPurchase, error)
}
