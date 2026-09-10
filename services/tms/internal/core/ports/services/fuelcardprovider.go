package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/pagination"
)

// FuelFeedCapability says how much of a fuel transaction a connector can be
// trusted to carry. A fleet network reports Level III detail — gallons, grade,
// unit price, odometer, jurisdiction — which is enough to file IFTA on. A
// corporate card reports only what a Visa authorization carries, so its rows
// need a person to complete them before they can become tax records.
type FuelFeedCapability string

const (
	FuelFeedCapabilityIFTAGrade FuelFeedCapability = "IFTAGrade"
	FuelFeedCapabilitySpendOnly FuelFeedCapability = "SpendOnly"
)

func (c FuelFeedCapability) IsIFTAGrade() bool { return c == FuelFeedCapabilityIFTAGrade }

// ProviderCard is a card as the provider describes it. Only the last four are
// ever carried; a full PAN must never enter Trenova.
type ProviderCard struct {
	LastFour       string
	ExternalCardID string
	Label          string
	Provider       fuelpurchase.CardProvider
}

type FetchFuelTransactionsRequest struct {
	TenantInfo pagination.TenantInfo
	Provider   fuelpurchase.CardProvider
	Config     map[string]string
	Since      int64
	Until      int64
	Cursor     string
}

// FetchFuelTransactionsResult carries the provider's rows in the same shape a
// spreadsheet upload produces, so the resolution and dedupe pipeline that backs
// manual imports serves feeds unchanged.
type FetchFuelTransactionsResult struct {
	Staged    *fuelimport.StageResult
	Cursor    string
	Reference string
	Format    fuelpurchase.SourceFormat
	Cards     []ProviderCard
}

// FuelCardProvider is one card network Trenova can read transactions from. A
// connector's only job is to turn what the vendor publishes into staged rows; it
// resolves nothing and writes nothing.
type FuelCardProvider interface {
	IntegrationType() integration.Type
	Capability() FuelFeedCapability
	FetchTransactions(
		ctx context.Context,
		req *FetchFuelTransactionsRequest,
	) (*FetchFuelTransactionsResult, error)
	TestConnection(ctx context.Context, config map[string]string) error
}
