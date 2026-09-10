package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/pkg/pagination"
)

// ResolvedFuelFeed is one organization's live connection to a card network: the
// connector that can read it and the decrypted settings to read it with.
type ResolvedFuelFeed struct {
	Connector       FuelCardProvider
	IntegrationType integration.Type
	Provider        fuelpurchase.CardProvider
	Config          map[string]string
	DiscoverCards   bool
}

// FuelCardFeedResolver turns an organization's integration settings into a
// connector ready to read. It lives behind a port so the fuel service never has
// to know how integrations are stored or how their secrets are unwrapped.
type FuelCardFeedResolver interface {
	// ResolveFuelFeed returns the feed for one provider. It reports a nil feed
	// with no error when the organization has no enabled, fully configured
	// connection for it, because a tenant without a feed is ordinary rather than
	// a failure.
	ResolveFuelFeed(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		provider fuelpurchase.CardProvider,
	) (*ResolvedFuelFeed, error)

	// ListFuelFeeds returns every enabled, fully configured feed an organization
	// has, which is what a scheduled run iterates.
	ListFuelFeeds(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*ResolvedFuelFeed, error)
}
