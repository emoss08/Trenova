package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/typeutils"
)

const tractorIDSource = "from list_tractors"

const (
	paramFuelIndexID       = "fuelIndexId"
	paramReportScheduleDef = "definitionId"
	fieldQuarter           = "quarter"
	fieldYear              = "year"
	fieldSource            = "source"
	fieldTractorID         = "tractorId"
	fieldJurisdictionID    = "jurisdictionId"
	maxFuelIndexPrices     = 104
	defaultFuelIndexPrices = 26
	maxReportSchedules     = 50
	priceDayLayout         = "2006-01-02"
)

func fuelRateReadProviders() []any {
	return []any{
		newListFuelPurchasesTool,
		newListFuelCardsTool,
		newListFuelPurchaseImportsTool,
		newListFuelIndexPricesTool,
		newListIFTAJurisdictionsTool,
		newListIFTAReturnsTool,
		newListIFTAMileageEntriesTool,
		newListRateImportsTool,
		provideListReportSchedulesTool,
	}
}

type fuelPurchaseRow struct {
	ID                   string       `json:"id"`
	TractorID            string       `json:"tractorId"`
	TractorCode          string       `json:"tractorCode,omitempty"`
	WorkerID             string       `json:"workerId,omitempty"`
	JurisdictionID       string       `json:"jurisdictionId"`
	Jurisdiction         string       `json:"jurisdiction,omitempty"`
	PurchasedAt          optionalDate `json:"purchasedAt"`
	Vendor               string       `json:"vendor,omitempty"`
	VendorCity           string       `json:"vendorCity,omitempty"`
	FuelType             string       `json:"fuelType"`
	Quantity             string       `json:"quantity"`
	QuantityUnit         string       `json:"quantityUnit"`
	Gallons              string       `json:"gallons"`
	TotalAmount          string       `json:"totalAmount"`
	CurrencyCode         string       `json:"currencyCode"`
	TaxPaid              bool         `json:"taxPaid"`
	FuelCardID           string       `json:"fuelCardId,omitempty"`
	CardLastFour         string       `json:"cardLastFour,omitempty"`
	TransactionReference string       `json:"transactionReference,omitempty"`
	Source               string       `json:"source"`
	ImportBatchID        string       `json:"importBatchId,omitempty"`
}

func fuelPurchaseRowFrom(item *fuelpurchase.FuelPurchase) fuelPurchaseRow {
	row := fuelPurchaseRow{
		ID:                   item.ID.String(),
		TractorID:            item.TractorID.String(),
		JurisdictionID:       item.JurisdictionID.String(),
		PurchasedAt:          recordedDate(item.PurchasedAt),
		Vendor:               item.Vendor,
		VendorCity:           item.VendorCity,
		FuelType:             string(item.FuelType),
		Quantity:             item.Quantity.String(),
		QuantityUnit:         string(item.QuantityUnit),
		Gallons:              item.Gallons.StringFixed(3),
		TotalAmount:          minorText(item.TotalAmountMinor),
		CurrencyCode:         item.CurrencyCode,
		TaxPaid:              item.TaxPaid,
		CardLastFour:         item.CardLastFour,
		TransactionReference: item.TransactionReference,
		Source:               string(item.Source),
	}
	if item.WorkerID != nil {
		row.WorkerID = pulidString(*item.WorkerID)
	}
	if item.FuelCardID != nil {
		row.FuelCardID = pulidString(*item.FuelCardID)
	}
	if item.ImportBatchID != nil {
		row.ImportBatchID = pulidString(*item.ImportBatchID)
	}
	if item.Tractor != nil {
		row.TractorCode = item.Tractor.Code
	}
	if item.Jurisdiction != nil {
		row.Jurisdiction = item.Jurisdiction.Code
	}

	return row
}

func cursorOf(opts *pagination.QueryOptions) pagination.CursorInfo {
	return pagination.CursorInfo{Limit: opts.Pagination.Limit}
}

func newListFuelPurchasesTool(
	repo repositories.FuelPurchaseRepository,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_fuel_purchases",
		entityPlural: "fuel purchases",
		summary: "List fuel bought for tractors, newest first, with where it was bought for " +
			"IFTA, the fuel type, the gallons and the amount. Each also says whether tax was " +
			"paid at the pump. Use it to find a purchase to correct with " +
			"correct_fuel_purchase or delete_fuel_purchase, or to check a receipt was " +
			"already recorded before record_fuel_purchase.",
		resource: permission.ResourceFuelPurchase,
		config:   querybuilder.GetFieldConfiguration((*fuelpurchase.FuelPurchase)(nil)),
		fields: []listField{
			{
				Name:   "fuelType",
				Kind:   filterEnum,
				Values: sliceutils.Strings(domaintypes.IFTAFuelTypeValues()),
			},
			{
				Name:   fieldSource,
				Kind:   filterEnum,
				Values: sliceutils.Strings(fuelpurchase.PurchaseSourceValues()),
			},
			{Name: "taxPaid", Kind: filterBool},
			{Name: "purchasedAt", Kind: filterDate, Sortable: true},
			{Name: "vendor", Kind: filterText},
			{Name: "transactionReference", Kind: filterText},
			{Name: "cardLastFour", Kind: filterText},
			{Name: fieldTractorID, Kind: filterText, Note: tractorIDSource},
			{Name: "importBatchId", Kind: filterText, Note: "from list_fuel_purchase_imports"},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.ListPurchases(ctx, &repositories.ListFuelPurchasesRequest{
				Filter:              opts,
				Cursor:              cursorOf(opts),
				IncludeTractor:      true,
				IncludeJurisdiction: true,
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *fuelpurchase.FuelPurchase) any {
				return fuelPurchaseRowFrom(item)
			}), nil
		},
	})
}

type fuelCardRow struct {
	ID                string       `json:"id"`
	Provider          string       `json:"provider"`
	LastFour          string       `json:"lastFour"`
	Label             string       `json:"label"`
	Status            string       `json:"status"`
	AssignedTractorID string       `json:"assignedTractorId,omitempty"`
	AssignedWorkerID  string       `json:"assignedWorkerId,omitempty"`
	Unassigned        bool         `json:"unassigned"`
	DiscoveredByFeed  bool         `json:"discoveredByFeed"`
	ExpiresAt         optionalDate `json:"expiresAt"`
}

func newListFuelCardsTool(repo repositories.FuelPurchaseRepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_fuel_cards",
		entityPlural: "fuel cards",
		summary: "List the organization's fuel cards with the provider, the last four " +
			"digits, the status and the tractor or driver each is assigned to. A card a feed " +
			"discovered is Suspended and unassigned until someone assigns it with " +
			"assign_fuel_card, and until then its purchases wait in their import.",
		resource: permission.ResourceFuelCard,
		config:   querybuilder.GetFieldConfiguration((*fuelpurchase.FuelCard)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: sliceutils.Strings(fuelpurchase.CardStatusValues()),
			},
			{
				Name:   "provider",
				Kind:   filterEnum,
				Values: sliceutils.Strings(fuelpurchase.CardProviderValues()),
			},
			{Name: "lastFour", Kind: filterText},
			{Name: "label", Kind: filterText, Sortable: true},
			{Name: "discoveredAt", Kind: filterDate, Sortable: true},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.ListCards(ctx, &repositories.ListFuelCardsRequest{
				Filter: opts,
				Cursor: cursorOf(opts),
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *fuelpurchase.FuelCard) any {
				row := fuelCardRow{
					ID:               item.ID.String(),
					Provider:         string(item.Provider),
					LastFour:         item.LastFour,
					Label:            item.Label,
					Status:           string(item.Status),
					Unassigned:       item.IsUnassigned(),
					DiscoveredByFeed: item.WasDiscovered(),
					ExpiresAt:        expectedDate(typeutils.ValueOrZero(item.ExpiresAt), "no expiry on file"),
				}
				if item.AssignedTractorID != nil {
					row.AssignedTractorID = pulidString(*item.AssignedTractorID)
				}
				if item.AssignedWorkerID != nil {
					row.AssignedWorkerID = pulidString(*item.AssignedWorkerID)
				}

				return row
			}), nil
		},
	})
}

type fuelImportRow struct {
	ID              string       `json:"id"`
	Provider        string       `json:"provider"`
	Origin          string       `json:"origin"`
	Status          string       `json:"status"`
	FileName        string       `json:"fileName,omitempty"`
	RowCount        int          `json:"rowCount"`
	ReadyToCommit   int          `json:"readyToCommit"`
	AlreadyImported int          `json:"alreadyImported"`
	DuplicateInFile int          `json:"duplicateInFile"`
	ErrorCount      int          `json:"errorCount"`
	CommittedCount  int          `json:"committedCount"`
	TotalGallons    string       `json:"totalGallons,omitempty"`
	TotalAmount     string       `json:"totalAmount,omitempty"`
	StagedAt        optionalDate `json:"stagedAt"`
	CommittedAt     optionalDate `json:"committedAt"`
	Error           string       `json:"error,omitempty"`
}

func fuelImportRowFrom(item *fuelpurchase.ImportBatch) fuelImportRow {
	row := fuelImportRow{
		ID:             item.ID.String(),
		Provider:       string(item.Provider),
		Origin:         string(item.Origin),
		Status:         string(item.Status),
		FileName:       item.FileName,
		RowCount:       item.RowCount,
		ErrorCount:     item.ErrorCount,
		CommittedCount: item.CommittedCount,
		StagedAt:       expectedDate(typeutils.ValueOrZero(item.StagedAt), "not staged"),
		CommittedAt:    expectedDate(typeutils.ValueOrZero(item.CommittedAt), "not committed"),
		Error:          item.Error,
	}
	if item.Summary != nil {
		row.ReadyToCommit = item.Summary.NewCount
		row.AlreadyImported = item.Summary.AlreadyImportedCount
		row.DuplicateInFile = item.Summary.DuplicateInFileCount
		row.TotalGallons = item.Summary.TotalGallons
		row.TotalAmount = minorText(item.Summary.TotalAmountMinor)
	}

	return row
}

func newListFuelPurchaseImportsTool(
	repo repositories.FuelPurchaseRepository,
) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_fuel_purchase_imports",
		entityPlural: "fuel purchase imports",
		summary: "List fuel card statements and feed runs staged for import, newest first, " +
			"with how many rows are ready to commit, already on file, duplicated or in error. " +
			"A Parsed import waits for commit_fuel_purchase_import or " +
			"discard_fuel_purchase_import; rows held because a card or tractor was unknown " +
			"are worked out again with resolve_fuel_purchase_import_rows.",
		resource: permission.ResourceFuelPurchaseImport,
		config:   querybuilder.GetFieldConfiguration((*fuelpurchase.ImportBatch)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: sliceutils.Strings(fuelpurchase.ImportStatusValues()),
				Note:   "Parsed imports are waiting to be committed",
			},
			{
				Name:   "provider",
				Kind:   filterEnum,
				Values: sliceutils.Strings(fuelpurchase.CardProviderValues()),
			},
			{Name: "fileName", Kind: filterText},
			{Name: "stagedAt", Kind: filterDate, Sortable: true},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.ListImportBatches(ctx, &repositories.ListImportBatchesRequest{
				Filter: opts,
				Cursor: cursorOf(opts),
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *fuelpurchase.ImportBatch) any {
				return fuelImportRowFrom(item)
			}), nil
		},
	})
}

type fuelIndexPriceRow struct {
	ID        string `json:"id"`
	PriceDate string `json:"priceDate"`
	Price     string `json:"price"`
	Currency  string `json:"currency"`
	IsManual  bool   `json:"isManual"`
}

type listFuelIndexPricesTool struct {
	prices repositories.FuelIndexPriceRepository
}

func newListFuelIndexPricesTool(
	prices repositories.FuelIndexPriceRepository,
) serviceports.AgentQueryTool {
	return &listFuelIndexPricesTool{prices: prices}
}

func (t *listFuelIndexPricesTool) Name() string { return "list_fuel_index_prices" }

func (t *listFuelIndexPricesTool) Description() string {
	return "List the prices on file for one fuel index, newest first, with the day each " +
		"applies to and whether a person entered it or it was fetched. Only a manual price " +
		"on a custom index can be corrected, with correct_fuel_index_price; " +
		"record_fuel_index_price adds the next one."
}

func (t *listFuelIndexPricesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramFuelIndexID: stringParam("The fuel index, from get_fuel_surcharge_rates. Never " +
			"guess one."),
		"from": stringParam("The earliest price day to include, YYYY-MM-DD."),
		"to":   stringParam("The latest price day to include, YYYY-MM-DD."),
		paramLimit: intParam(
			fmt.Sprintf("How many prices to return, at most %d.", maxFuelIndexPrices),
		),
	}, paramFuelIndexID)
}

func (t *listFuelIndexPricesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFuelSurchargeProgram})
}

func optionalPriceDay(params map[string]any, key string) (string, error) {
	day := strings.TrimSpace(optionalString(params, key))
	if day == "" {
		return "", nil
	}
	if _, err := time.Parse(priceDayLayout, day); err != nil {
		return "", fmt.Errorf("%s must be YYYY-MM-DD, got %q", key, day)
	}

	return day, nil
}

func (t *listFuelIndexPricesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	indexID, err := requirePulid(params.Params, paramFuelIndexID)
	if err != nil {
		return nil, err
	}
	from, err := optionalPriceDay(params.Params, "from")
	if err != nil {
		return nil, err
	}
	to, err := optionalPriceDay(params.Params, "to")
	if err != nil {
		return nil, err
	}

	prices, err := t.prices.ListByIndex(ctx, &repositories.ListFuelIndexPricesRequest{
		FuelIndexID: indexID,
		TenantInfo:  tenantOf(params),
		From:        from,
		To:          to,
		Limit: min(
			max(optionalInt(params.Params, paramLimit, defaultFuelIndexPrices), 1),
			maxFuelIndexPrices,
		),
	})
	if err != nil {
		return nil, err
	}

	rows := make([]fuelIndexPriceRow, 0, len(prices))
	for _, price := range prices {
		rows = append(rows, fuelIndexPriceRow{
			ID:        price.ID.String(),
			PriceDate: price.PriceDate,
			Price:     price.Price.String(),
			Currency:  price.Currency,
			IsManual:  price.IsManual,
		})
	}

	return map[string]any{"fuelIndexId": indexID.String(), "prices": rows}, nil
}

type iftaJurisdictionRow struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	IFTAMember  bool   `json:"iftaMember"`
}

type listIFTAJurisdictionsTool struct {
	repo repositories.IFTARepository
}

func newListIFTAJurisdictionsTool(repo repositories.IFTARepository) serviceports.AgentQueryTool {
	return &listIFTAJurisdictionsTool{repo: repo}
}

func (t *listIFTAJurisdictionsTool) Name() string { return "list_ifta_jurisdictions" }

func (t *listIFTAJurisdictionsTool) Description() string {
	return "List the active states and provinces fuel tax is reported by, with each " +
		"one's two-letter code and whether it is an IFTA member. Its id is what " +
		"record_fuel_purchase and record_ifta_mileage_entry take. Narrow by country code, " +
		"US, CA or MX."
}

func (t *listIFTAJurisdictionsTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		"countryCode": stringParam("Only this country's jurisdictions: US, CA or MX."),
		"membersOnly": boolParam("Only IFTA member jurisdictions."),
	})
}

func (t *listIFTAJurisdictionsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFuelPurchase})
}

func (t *listIFTAJurisdictionsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	jurisdictions, err := t.repo.ListJurisdictions(ctx, &repositories.ListJurisdictionsRequest{
		MembersOnly: optionalBool(params.Params, "membersOnly"),
		CountryCode: strings.ToUpper(
			strings.TrimSpace(optionalString(params.Params, "countryCode")),
		),
		Statuses: []ifta.JurisdictionStatus{ifta.JurisdictionStatusActive},
	})
	if err != nil {
		return nil, err
	}

	rows := make([]iftaJurisdictionRow, 0, len(jurisdictions))
	for _, jurisdiction := range jurisdictions {
		rows = append(rows, iftaJurisdictionRow{
			ID:          jurisdiction.ID.String(),
			Code:        jurisdiction.Code,
			Name:        jurisdiction.Name,
			CountryCode: jurisdiction.CountryCode,
			IFTAMember:  jurisdiction.IsIftaMember,
		})
	}

	return map[string]any{"jurisdictions": rows}, nil
}

type iftaReturnRow struct {
	ID               string       `json:"id"`
	Year             int          `json:"year"`
	Quarter          int          `json:"quarter"`
	Status           string       `json:"status"`
	AmendmentNumber  int          `json:"amendmentNumber"`
	TotalMiles       string       `json:"totalMiles"`
	TotalGallons     string       `json:"totalGallons"`
	NetDue           string       `json:"netDue"`
	CurrencyCode     string       `json:"currencyCode"`
	Problems         int          `json:"problems"`
	BlockingProblems int          `json:"blockingProblems"`
	FinalizedAt      optionalDate `json:"finalizedAt"`
	FiledAt          optionalDate `json:"filedAt"`
}

func newListIFTAReturnsTool(repo repositories.IFTARepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_ifta_returns",
		entityPlural: "IFTA returns",
		summary: "List the organization's quarterly IFTA fuel tax returns with their status, " +
			"miles, gallons, the net tax due and how many problems each carries. A Draft is " +
			"recomputed with recompute_ifta_return or deleted with delete_ifta_return; a " +
			"Filed return is corrected with amend_ifta_return.",
		resource: permission.ResourceIFTAReturn,
		config:   querybuilder.GetFieldConfiguration((*ifta.Return)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: sliceutils.Strings(ifta.ReturnStatusValues()),
			},
			{Name: fieldYear, Kind: filterNumber, Sortable: true},
			{Name: fieldQuarter, Kind: filterNumber, Sortable: true},
			{Name: fieldCreatedAt, Kind: filterDate, Sortable: true},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.ListReturns(ctx, &repositories.ListReturnsRequest{
				Filter: opts,
				Cursor: cursorOf(opts),
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *ifta.Return) any {
				return iftaReturnRow{
					ID:               item.ID.String(),
					Year:             item.Year,
					Quarter:          item.Quarter,
					Status:           string(item.Status),
					AmendmentNumber:  item.AmendmentNumber,
					TotalMiles:       item.TotalMiles.StringFixed(2),
					TotalGallons:     item.TotalGallons.StringFixed(3),
					NetDue:           item.NetDue().StringFixed(2),
					CurrencyCode:     item.CurrencyCode,
					Problems:         len(item.Problems),
					BlockingProblems: len(item.BlockingProblems()),
					FinalizedAt:      expectedDate(typeutils.ValueOrZero(item.FinalizedAt), "not finalized"),
					FiledAt:          expectedDate(typeutils.ValueOrZero(item.FiledAt), "not filed"),
				}
			}), nil
		},
	})
}

type iftaMileageRow struct {
	ID             string       `json:"id"`
	TractorID      string       `json:"tractorId"`
	TractorCode    string       `json:"tractorCode,omitempty"`
	JurisdictionID string       `json:"jurisdictionId"`
	Jurisdiction   string       `json:"jurisdiction,omitempty"`
	TraveledAt     optionalDate `json:"traveledAt"`
	Year           int          `json:"year"`
	Quarter        int          `json:"quarter"`
	Miles          string       `json:"miles"`
	Loaded         bool         `json:"loaded"`
	Source         string       `json:"source"`
	ShipmentMoveID string       `json:"shipmentMoveId,omitempty"`
	Notes          string       `json:"notes,omitempty"`
}

func newListIFTAMileageEntriesTool(repo repositories.IFTARepository) serviceports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "list_ifta_mileage_entries",
		entityPlural: "IFTA mileage entries",
		summary: "List the jurisdiction miles entered by hand or from telematics rather than " +
			"routed from moves. Each has the tractor, the state or province, the day and the " +
			"quarter it counts in. correct_ifta_mileage_entry and delete_ifta_mileage_entry " +
			"take their ids.",
		resource: permission.ResourceIFTAJurisdictionMileage,
		config:   querybuilder.GetFieldConfiguration((*ifta.JurisdictionMileageEntry)(nil)),
		fields: []listField{
			{
				Name:   fieldSource,
				Kind:   filterEnum,
				Values: sliceutils.Strings(ifta.MileageSourceValues()),
			},
			{Name: fieldYear, Kind: filterNumber, Sortable: true},
			{Name: fieldQuarter, Kind: filterNumber, Sortable: true},
			{Name: "traveledAt", Kind: filterDate, Sortable: true},
			{Name: fieldTractorID, Kind: filterText, Note: tractorIDSource},
			{Name: fieldJurisdictionID, Kind: filterText, Note: "from list_ifta_jurisdictions"},
			{Name: "loaded", Kind: filterBool},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.ListMileageEntries(ctx, &repositories.ListMileageEntriesRequest{
				Filter:              opts,
				Cursor:              cursorOf(opts),
				IncludeTractor:      true,
				IncludeJurisdiction: true,
			})
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *ifta.JurisdictionMileageEntry) any {
				row := iftaMileageRow{
					ID:             item.ID.String(),
					TractorID:      item.TractorID.String(),
					JurisdictionID: item.JurisdictionID.String(),
					TraveledAt:     recordedDate(item.TraveledAt),
					Year:           item.Year,
					Quarter:        item.Quarter,
					Miles:          item.Miles.StringFixed(2),
					Loaded:         item.Loaded,
					Source:         string(item.Source),
					Notes:          item.Notes,
				}
				if item.ShipmentMoveID != nil {
					row.ShipmentMoveID = pulidString(*item.ShipmentMoveID)
				}
				if item.Tractor != nil {
					row.TractorCode = item.Tractor.Code
				}
				if item.Jurisdiction != nil {
					row.Jurisdiction = item.Jurisdiction.Code
				}

				return row
			}), nil
		},
	})
}

type rateImportRow struct {
	ID              string       `json:"id"`
	RateAgreementID string       `json:"rateAgreementId"`
	FileName        string       `json:"fileName"`
	Status          string       `json:"status"`
	EffectiveFrom   optionalDate `json:"effectiveFrom"`
	RowCount        int          `json:"rowCount"`
	ErrorCount      int          `json:"errorCount"`
	Added           int          `json:"lanesAdded"`
	Changed         int          `json:"lanesChanged"`
	Removed         int          `json:"lanesRemoved"`
	Unchanged       int          `json:"lanesUnchanged"`
	CommittedAt     optionalDate `json:"committedAt"`
	Error           string       `json:"error,omitempty"`
}

type listRateImportsTool struct {
	imports repositories.RateImportRepository
}

func newListRateImportsTool(imports repositories.RateImportRepository) serviceports.AgentQueryTool {
	return &listRateImportsTool{imports: imports}
}

func (t *listRateImportsTool) Name() string { return "list_rate_imports" }

func (t *listRateImportsTool) SearchTerms() []string {
	return []string{"rate sheet imports", "imports waiting review", "pending rate imports"}
}

func (t *listRateImportsTool) Description() string {
	return "List rate sheets uploaded against rate agreements, newest first, with the " +
		"agreement and the day the new rates take effect. Each says how many lanes the " +
		"sheet adds, changes and removes. A Parsed import waits for commit_rate_import or " +
		"discard_rate_import."
}

func (t *listRateImportsTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		"rateAgreementId": stringParam("Only the imports against this agreement, from " +
			"list_rate_agreements."),
		paramLimit: intParam(fmt.Sprintf("How many imports to return, at most %d.", maxListLimit)),
	})
}

func (t *listRateImportsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRateAgreement})
}

func (t *listRateImportsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	req := &repositories.ListRateImportBatchesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: min(
				max(optionalInt(params.Params, paramLimit, defaultListLimit), 1),
				maxListLimit,
			)},
		},
	}
	if raw := strings.TrimSpace(optionalString(params.Params, "rateAgreementId")); raw != "" {
		agreementID, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%q is not a rate agreement id; find it with "+
				"list_rate_agreements", raw)
		}
		req.RateAgreementID = &agreementID
	}

	result, err := t.imports.List(ctx, req)
	if err != nil {
		return nil, err
	}

	rows := make([]rateImportRow, 0, len(result.Items))
	for _, item := range result.Items {
		row := rateImportRow{
			ID:              item.ID.String(),
			RateAgreementID: item.RateAgreementID.String(),
			FileName:        item.FileName,
			Status:          string(item.Status),
			EffectiveFrom:   recordedDate(item.EffectiveFrom),
			RowCount:        item.RowCount,
			ErrorCount:      item.ErrorCount,
			CommittedAt:     expectedDate(typeutils.ValueOrZero(item.CommittedAt), "not committed"),
			Error:           item.Error,
		}
		if item.Summary != nil {
			row.Added = item.Summary.Added
			row.Changed = item.Summary.Changed
			row.Removed = item.Summary.Removed
			row.Unchanged = item.Summary.Unchanged
		}
		rows = append(rows, row)
	}

	return map[string]any{"imports": rows, "total": result.Total}, nil
}

type reportScheduleReader interface {
	ListSchedules(
		ctx context.Context,
		req *reporting.ListSchedulesRequest,
	) ([]*report.ReportSchedule, error)
}

type reportScheduleRow struct {
	ID             string       `json:"id"`
	DefinitionID   string       `json:"definitionId"`
	CronExpression string       `json:"cronExpression"`
	Timezone       string       `json:"timezone"`
	Formats        []string     `json:"formats"`
	Recipients     []string     `json:"emailRecipients"`
	Enabled        bool         `json:"enabled"`
	Yours          bool         `json:"yours"`
	NextRunAt      optionalDate `json:"nextRunAt"`
	Failures       int          `json:"consecutiveFailures,omitempty"`
}

type listReportSchedulesTool struct {
	schedules reportScheduleReader
}

func provideListReportSchedulesTool(schedules *reporting.Service) serviceports.AgentQueryTool {
	return &listReportSchedulesTool{schedules: schedules}
}

func (t *listReportSchedulesTool) Name() string { return "list_report_schedules" }

func (t *listReportSchedulesTool) SearchTerms() []string {
	return []string{"scheduled", "emailed", "recurring", "subscription"}
}

func (t *listReportSchedulesTool) Description() string {
	return "List reports on a schedule: when each runs, in what formats, who it is emailed " +
		"to, whether it is on and when it runs next. Only the person who set a schedule up " +
		"can change it with update_report_schedule or remove it with " +
		"delete_report_schedule; yours says which are theirs."
}

func (t *listReportSchedulesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramReportScheduleDef: stringParam("Only the schedules of this saved report, from " +
			"list_reports."),
		"enabledOnly": boolParam("Only schedules that are switched on."),
		paramLimit: intParam(fmt.Sprintf("How many schedules to return, at most %d.",
			maxReportSchedules)),
	})
}

func (t *listReportSchedulesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceReport})
}

func (t *listReportSchedulesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	definitionID := pulid.Nil
	if raw := strings.TrimSpace(optionalString(params.Params, paramReportScheduleDef)); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%q is not a report id; find it with list_reports", raw)
		}
		definitionID = parsed
	}

	tenant := tenantOf(params)
	schedules, err := t.schedules.ListSchedules(ctx, &reporting.ListSchedulesRequest{
		Request:      reporting.Request{TenantInfo: tenant},
		DefinitionID: definitionID,
		EnabledOnly:  optionalBool(params.Params, "enabledOnly"),
		Limit: min(
			max(optionalInt(params.Params, paramLimit, maxReportSchedules), 1),
			maxReportSchedules,
		),
	})
	if err != nil {
		return nil, err
	}

	rows := make([]reportScheduleRow, 0, len(schedules))
	for _, schedule := range schedules {
		row := reportScheduleRow{
			ID:             schedule.ID.String(),
			DefinitionID:   schedule.DefinitionID.String(),
			CronExpression: schedule.CronExpression,
			Timezone:       schedule.Timezone,
			Formats:        schedule.Formats,
			Enabled:        schedule.Enabled,
			Yours:          schedule.RunAsID == tenant.UserID,
			NextRunAt:      expectedDate(schedule.NextRunAt, "not scheduled"),
			Failures:       schedule.ConsecutiveFailures,
		}
		if schedule.Delivery != nil {
			row.Recipients = schedule.Delivery.EmailRecipients
		}
		rows = append(rows, row)
	}

	return map[string]any{"schedules": rows}, nil
}
