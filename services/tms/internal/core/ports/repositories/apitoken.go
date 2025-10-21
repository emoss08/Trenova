package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
)

// CreateAPITokenRequest contains the data needed to create a new API token
type CreateAPITokenRequest struct {
	Token *tenant.APIToken
}

// FindAPITokenByTokenRequest contains parameters for finding a token by its value
type FindAPITokenByTokenRequest struct {
	TokenPrefix string // Used for initial lookup
	PlainToken  string // Full token for verification
}

// UpdateAPITokenLastUsedRequest contains parameters for updating token last used info
type UpdateAPITokenLastUsedRequest struct {
	TokenID pulid.ID
	IP      string
}

// ListAPITokensRequest contains parameters for listing API tokens
type ListAPITokensRequest struct {
	UserID         *pulid.ID
	OrganizationID *pulid.ID
	BusinessUnitID *pulid.ID
	IncludeExpired bool
	IncludeRevoked bool
	Limit          int
	Offset         int
}

// APITokenRepository defines the interface for API token persistence
type APITokenRepository interface {
	// Create creates a new API token
	Create(ctx context.Context, req CreateAPITokenRequest) error

	// FindByID finds a token by its ID
	FindByID(ctx context.Context, tokenID pulid.ID) (*tenant.APIToken, error)

	// FindByToken finds a token by its prefix and verifies the full token
	// This method should be optimized with caching
	FindByToken(ctx context.Context, req FindAPITokenByTokenRequest) (*tenant.APIToken, error)

	// FindByUserID returns all tokens for a specific user
	FindByUserID(ctx context.Context, userID pulid.ID) ([]*tenant.APIToken, error)

	// List returns a paginated list of tokens based on filters
	List(ctx context.Context, req ListAPITokensRequest) ([]*tenant.APIToken, error)

	// UpdateLastUsed updates the last used timestamp and IP for a token
	UpdateLastUsed(ctx context.Context, req UpdateAPITokenLastUsedRequest) error

	// Revoke revokes a token by ID
	Revoke(ctx context.Context, tokenID pulid.ID) error

	// Delete permanently deletes a token
	Delete(ctx context.Context, tokenID pulid.ID) error

	// Count returns the total number of active tokens for a user
	CountActiveByUserID(ctx context.Context, userID pulid.ID) (int64, error)
}
