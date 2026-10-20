package fueldeskcardservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	recentPurchaseLimit = 5
	heldRowsShown       = 10
	successorCardLimit  = 5
	moneyScale          = 2
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Purchases     fuelpurchaseports.FuelPurchaseRepository
	Desk          fuelpurchaseports.FuelDeskCardReader
	IFTA          iftaports.IFTARepository
	Organizations tenantports.OrganizationRepository
	Users         tenantports.UserRepository
}

type purchaseReader interface {
	GetPurchaseByID(
		ctx context.Context,
		req *fuelpurchaseports.GetFuelPurchaseByIDRequest,
	) (*fuelpurchase.FuelPurchase, error)
	GetCardByID(
		ctx context.Context,
		req *fuelpurchaseports.GetFuelCardByIDRequest,
	) (*fuelpurchase.FuelCard, error)
	ListCards(
		ctx context.Context,
		req *fuelpurchaseports.ListFuelCardsRequest,
	) (*pagination.CursorListResult[*fuelpurchase.FuelCard], error)
	GetFeedState(
		ctx context.Context,
		req *fuelpurchaseports.GetFuelFeedStateRequest,
	) (*fuelpurchase.CardFeedState, error)
}

type rateResolver interface {
	ResolveRates(
		ctx context.Context,
		req *iftaports.ResolveRatesRequest,
	) (map[ifta.RateKey]*ifta.TaxRate, error)
}

type userReader interface {
	GetByIDs(ctx context.Context, req tenantports.GetUsersByIDsRequest) ([]*tenant.User, error)
}

type Service struct {
	l             *zap.Logger
	purchases     purchaseReader
	desk          fuelpurchaseports.FuelDeskCardReader
	ifta          rateResolver
	organizations orgzone.OrganizationReader
	users         userReader
	now           func() time.Time
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:             p.Logger.Named("service.fuel-desk-card"),
		purchases:     p.Purchases,
		desk:          p.Desk,
		ifta:          p.IFTA,
		organizations: p.Organizations,
		users:         p.Users,
		now:           time.Now,
	}
}

type PurchaseIFTA struct {
	Year                 int
	Quarter              int
	CountsForIFTA        bool
	TaxPaidGallons       decimal.Decimal
	RatePerGallon        decimal.NullDecimal
	TaxPaidCredit        decimal.NullDecimal
	TractorGallons       decimal.Decimal
	TractorPurchaseCount int
}

type Provenance struct {
	Batch           *fuelpurchase.ImportBatch
	RowNumber       int
	ResolutionNotes []string
}

type PurchaseCard struct {
	Purchase     *fuelpurchase.FuelPurchase
	IFTA         *PurchaseIFTA
	PreviousFill *fuelpurchase.FuelPurchase
	Provenance   *Provenance
	CreatedBy    *tenant.User
}

type GetPurchaseCardRequest struct {
	TenantInfo        pagination.TenantInfo
	PurchaseID        pulid.ID
	IncludeProvenance bool
	IncludeCreatedBy  bool
}

func (s *Service) GetPurchaseCard(
	ctx context.Context,
	req *GetPurchaseCardRequest,
) (*PurchaseCard, error) {
	purchase, err := s.purchases.GetPurchaseByID(ctx, &fuelpurchaseports.GetFuelPurchaseByIDRequest{
		ID:                  req.PurchaseID,
		TenantInfo:          req.TenantInfo,
		IncludeTractor:      true,
		IncludeWorker:       true,
		IncludeJurisdiction: true,
		IncludeFuelCard:     true,
	})
	if err != nil {
		return nil, err
	}

	loc, err := orgzone.Location(ctx, s.organizations, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	card := &PurchaseCard{Purchase: purchase}
	if card.IFTA, err = s.purchaseIFTA(ctx, req.TenantInfo, purchase, loc); err != nil {
		return nil, err
	}
	if card.PreviousFill, err = s.previousFill(ctx, req.TenantInfo, purchase); err != nil {
		return nil, err
	}
	if req.IncludeProvenance && purchase.Source == fuelpurchase.PurchaseSourceCardImport {
		if card.Provenance, err = s.provenance(ctx, req.TenantInfo, purchase.ID); err != nil {
			return nil, err
		}
	}
	if req.IncludeCreatedBy {
		if card.CreatedBy, err = s.user(ctx, req.TenantInfo, purchase.CreatedByID); err != nil {
			return nil, err
		}
	}

	return card, nil
}

func (s *Service) purchaseIFTA(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purchase *fuelpurchase.FuelPurchase,
	loc *time.Location,
) (*PurchaseIFTA, error) {
	period := ifta.PeriodOf(purchase.PurchasedAt, loc)
	start, end := period.Bounds(loc)

	out := &PurchaseIFTA{
		Year:           period.Year,
		Quarter:        period.Quarter,
		CountsForIFTA:  purchase.FuelType.CountsForIFTA(),
		TaxPaidGallons: purchase.TaxPaidGallons(),
	}

	totals, err := s.desk.SumPurchases(ctx, &fuelpurchaseports.SumFuelPurchasesRequest{
		TenantInfo:     tenantInfo,
		TractorID:      purchase.TractorID,
		JurisdictionID: purchase.JurisdictionID,
		FuelTypes:      domaintypes.IFTAFuelTypes(),
		From:           start,
		To:             end,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range totals {
		out.TractorGallons = out.TractorGallons.Add(row.Gallons)
		out.TractorPurchaseCount += row.PurchaseCount
	}

	if !out.CountsForIFTA {
		return out, nil
	}

	rates, err := s.ifta.ResolveRates(ctx, &iftaports.ResolveRatesRequest{
		Year:    period.Year,
		Quarter: period.Quarter,
	})
	if err != nil {
		return nil, err
	}
	rate := rates[ifta.RateKey{JurisdictionID: purchase.JurisdictionID, FuelType: purchase.FuelType}]
	if rate == nil {
		return out, nil
	}

	out.RatePerGallon = decimal.NewNullDecimal(rate.RatePerGallon)
	out.TaxPaidCredit = decimal.NewNullDecimal(
		out.TaxPaidGallons.Mul(rate.RatePerGallon).Round(moneyScale),
	)

	return out, nil
}

func (s *Service) previousFill(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purchase *fuelpurchase.FuelPurchase,
) (*fuelpurchase.FuelPurchase, error) {
	if purchase.Odometer == nil || !purchase.FuelType.CountsForIFTA() {
		return nil, nil //nolint:nilnil // miles since a fill are read only from a road-fuel fill with an odometer
	}

	return s.desk.GetPreviousFill(ctx, &fuelpurchaseports.GetPreviousFillRequest{
		TenantInfo: tenantInfo,
		Purchase:   purchase,
	})
}

func (s *Service) provenance(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	purchaseID pulid.ID,
) (*Provenance, error) {
	row, err := s.desk.GetPurchaseImportRow(ctx, &fuelpurchaseports.GetPurchaseImportRowRequest{
		TenantInfo: tenantInfo,
		PurchaseID: purchaseID,
	})
	if err != nil || row == nil || row.Batch == nil {
		return nil, err
	}

	notes := row.ResolutionNotes
	if notes == nil {
		notes = []string{}
	}

	return &Provenance{Batch: row.Batch, RowNumber: row.RowNumber, ResolutionNotes: notes}, nil
}

func (s *Service) user(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*tenant.User, error) {
	if id.IsNil() {
		return nil, nil //nolint:nilnil // a feed records no person
	}

	users, err := s.users.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    []pulid.ID{id},
	})
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		if user != nil && user.ID == id {
			return user, nil
		}
	}

	return nil, nil //nolint:nilnil // a user no longer on file is shown as nobody
}

type Spend struct {
	CurrencyCode     string
	PurchaseCount    int
	Gallons          decimal.Decimal
	UntaxedGallons   decimal.Decimal
	TotalAmountMinor int64
}

type JurisdictionGallons struct {
	JurisdictionID pulid.ID
	Gallons        decimal.Decimal
	PurchaseCount  int
}

type HeldSpend struct {
	CurrencyCode     string
	Gallons          decimal.Decimal
	TotalAmountMinor int64
}

type FuelCardCard struct {
	Card                  *fuelpurchase.FuelCard
	MonthStart            int64
	Year                  int
	Quarter               int
	Month                 []*Spend
	QuarterByJurisdiction []*JurisdictionGallons
	PurchaseCount         int
	LastPurchasedAt       *int64
	Recent                []*fuelpurchase.FuelPurchase
	Held                  []*fuelpurchase.ImportRow
	HeldCount             int
	HeldSpend             []*HeldSpend
	HeldBatches           []*fuelpurchase.ImportBatch
	Feed                  *fuelpurchase.CardFeedState
	Successors            []*fuelpurchase.FuelCard
}

type GetFuelCardCardRequest struct {
	TenantInfo       pagination.TenantInfo
	CardID           pulid.ID
	IncludePurchases bool
	IncludeImports   bool
}

func (s *Service) GetFuelCardCard(
	ctx context.Context,
	req *GetFuelCardCardRequest,
) (*FuelCardCard, error) {
	card, err := s.purchases.GetCardByID(ctx, &fuelpurchaseports.GetFuelCardByIDRequest{
		ID:                 req.CardID,
		TenantInfo:         req.TenantInfo,
		IncludeAssignments: true,
	})
	if err != nil {
		return nil, err
	}

	loc, err := orgzone.Location(ctx, s.organizations, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now().In(loc)
	period := ifta.PeriodOf(now.Unix(), loc)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).Unix()

	out := &FuelCardCard{
		Card:                  card,
		MonthStart:            monthStart,
		Year:                  period.Year,
		Quarter:               period.Quarter,
		Month:                 []*Spend{},
		QuarterByJurisdiction: []*JurisdictionGallons{},
		Recent:                []*fuelpurchase.FuelPurchase{},
		Held:                  []*fuelpurchase.ImportRow{},
		HeldSpend:             []*HeldSpend{},
		HeldBatches:           []*fuelpurchase.ImportBatch{},
		Successors:            []*fuelpurchase.FuelCard{},
	}

	if req.IncludePurchases {
		if err = s.cardPurchases(ctx, req.TenantInfo, out, period, loc); err != nil {
			return nil, err
		}
	}
	if req.IncludeImports {
		if err = s.cardHeldRows(ctx, req.TenantInfo, out); err != nil {
			return nil, err
		}
	}
	if out.Feed, err = s.feed(ctx, req.TenantInfo, card.Provider); err != nil {
		return nil, err
	}
	if card.IsCancelled() {
		if out.Successors, err = s.successors(ctx, req.TenantInfo, card); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func (s *Service) cardPurchases(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *FuelCardCard,
	period ifta.Period,
	loc *time.Location,
) error {
	cardID := out.Card.ID

	month, err := s.desk.SumPurchases(ctx, &fuelpurchaseports.SumFuelPurchasesRequest{
		TenantInfo: tenantInfo,
		FuelCardID: cardID,
		From:       out.MonthStart,
	})
	if err != nil {
		return err
	}
	for _, row := range month {
		out.Month = append(out.Month, spendOf(row))
	}

	lifetime, err := s.desk.SumPurchases(ctx, &fuelpurchaseports.SumFuelPurchasesRequest{
		TenantInfo: tenantInfo,
		FuelCardID: cardID,
	})
	if err != nil {
		return err
	}
	for _, row := range lifetime {
		out.PurchaseCount += row.PurchaseCount
		if row.LastPurchasedAt > 0 &&
			(out.LastPurchasedAt == nil || row.LastPurchasedAt > *out.LastPurchasedAt) {
			last := row.LastPurchasedAt
			out.LastPurchasedAt = &last
		}
	}

	start, end := period.Bounds(loc)
	byJurisdiction, err := s.desk.SumGallonsByJurisdiction(
		ctx,
		&fuelpurchaseports.SumFuelPurchasesRequest{
			TenantInfo: tenantInfo,
			FuelCardID: cardID,
			FuelTypes:  domaintypes.IFTAFuelTypes(),
			From:       start,
			To:         end,
		},
	)
	if err != nil {
		return err
	}
	for _, row := range byJurisdiction {
		out.QuarterByJurisdiction = append(out.QuarterByJurisdiction, &JurisdictionGallons{
			JurisdictionID: row.JurisdictionID,
			Gallons:        row.Gallons,
			PurchaseCount:  row.PurchaseCount,
		})
	}

	out.Recent, err = s.desk.ListRecentCardPurchases(
		ctx,
		&fuelpurchaseports.ListRecentCardPurchasesRequest{
			TenantInfo: tenantInfo,
			CardID:     cardID,
			Limit:      recentPurchaseLimit,
		},
	)

	return err
}

func spendOf(row *fuelpurchaseports.FuelSpendRow) *Spend {
	return &Spend{
		CurrencyCode:     row.CurrencyCode,
		PurchaseCount:    row.PurchaseCount,
		Gallons:          row.Gallons,
		UntaxedGallons:   row.UntaxedGallons,
		TotalAmountMinor: row.TotalAmountMinor,
	}
}

func (s *Service) cardHeldRows(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	out *FuelCardCard,
) error {
	rows, err := s.desk.ListHeldCardRows(ctx, &fuelpurchaseports.ListHeldCardRowsRequest{
		TenantInfo: tenantInfo,
		CardID:     out.Card.ID,
	})
	if err != nil {
		return err
	}

	out.HeldCount = len(rows)
	spend := make(map[string]*HeldSpend, 1)
	batches := make(map[pulid.ID]struct{}, 1)
	for index, row := range rows {
		if index < heldRowsShown {
			out.Held = append(out.Held, row)
		}
		if row.Batch != nil {
			if _, seen := batches[row.Batch.ID]; !seen {
				batches[row.Batch.ID] = struct{}{}
				out.HeldBatches = append(out.HeldBatches, row.Batch)
			}
		}
		if row.Parsed == nil {
			continue
		}
		code := row.Parsed.CurrencyCode
		if code == "" {
			code = money.DefaultCurrencyCode
		}
		total, ok := spend[code]
		if !ok {
			total = &HeldSpend{CurrencyCode: code}
			spend[code] = total
			out.HeldSpend = append(out.HeldSpend, total)
		}
		total.Gallons = total.Gallons.Add(row.Parsed.Gallons)
		total.TotalAmountMinor += row.Parsed.TotalAmountMinor
	}

	return nil
}

func (s *Service) feed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider fuelpurchase.CardProvider,
) (*fuelpurchase.CardFeedState, error) {
	state, err := s.purchases.GetFeedState(ctx, &fuelpurchaseports.GetFuelFeedStateRequest{
		TenantInfo: tenantInfo,
		Provider:   provider,
		FeedType:   fuelpurchase.FeedTypeTransactions,
	})
	if err != nil {
		return nil, err
	}
	if state == nil || state.LastPolledAt == 0 {
		return nil, nil //nolint:nilnil // a provider with no feed has no feed to report
	}

	return state, nil
}

func (s *Service) successors(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	card *fuelpurchase.FuelCard,
) ([]*fuelpurchase.FuelCard, error) {
	out := make([]*fuelpurchase.FuelCard, 0, successorCardLimit)
	seen := map[pulid.ID]struct{}{card.ID: {}}

	holders := []fuelpurchaseports.ListFuelCardsRequest{}
	if card.AssignedTractorID != nil && !card.AssignedTractorID.IsNil() {
		holders = append(holders, fuelpurchaseports.ListFuelCardsRequest{
			AssignedTractorID: *card.AssignedTractorID,
		})
	}
	if card.AssignedWorkerID != nil && !card.AssignedWorkerID.IsNil() {
		holders = append(holders, fuelpurchaseports.ListFuelCardsRequest{
			AssignedWorkerID: *card.AssignedWorkerID,
		})
	}

	for index := range holders {
		req := &holders[index]
		req.Filter = &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: successorCardLimit},
		}
		req.Cursor = pagination.CursorInfo{Limit: successorCardLimit}
		req.Statuses = []fuelpurchase.CardStatus{fuelpurchase.CardStatusActive}
		req.IncludeAssignments = true

		result, err := s.purchases.ListCards(ctx, req)
		if err != nil {
			return nil, err
		}
		for _, candidate := range result.Items {
			if _, dup := seen[candidate.ID]; dup || len(out) == successorCardLimit {
				continue
			}
			seen[candidate.ID] = struct{}{}
			out = append(out, candidate)
		}
	}

	return out, nil
}
