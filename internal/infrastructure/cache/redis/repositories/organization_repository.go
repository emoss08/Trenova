package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	defaultUserOrgTTL      = 1 * time.Hour  // * 1 hour
	defaultOrganizationTTL = 24 * time.Hour // * 24 hours
	orgKeyPrefix           = "org:"
	orgListKeyPrefix       = "orgs:"
	userOrgKeyPrefix       = "user_orgs:"
)

type OrganizationRepositoryParams struct {
	fx.In

	Cache  *redis.Client
	Logger *logger.Logger
}

type organizationRepository struct {
	cache *redis.Client
	l     *zerolog.Logger
}

func NewOrganizationRepository(p OrganizationRepositoryParams) repositories.OrganizationCacheRepository {
	log := p.Logger.With().
		Str("repository", "organization").
		Str("component", "redis").
		Logger()

	return &organizationRepository{
		cache: p.Cache,
		l:     &log,
	}
}

// GetByID retrieves an organization from the cache by its ID
//
// Parameters:
//   - ctx: The context of the request
//   - opts: The options for the request
//
// Returns:
//   - org: The organization
//   - error: An error if the organization is not found in the cache
func (or *organizationRepository) GetByID(ctx context.Context, orgID pulid.ID) (*organization.Organization, error) {
	log := or.l.With().
		Str("operation", "GetByID").
		Str("orgID", orgID.String()).
		Logger()

	// * initialize a new organization object
	org := new(organization.Organization)

	// * format the key
	key := or.formatKey(orgID)

	// * get the organization from the cache
	if err := or.cache.GetJSON(ctx, key, org); err != nil {
		// * If the organization is not found in the cache, we need to fetch it from the database
		if eris.Is(err, redis.ErrNil) {
			log.Debug().Str("key", key).Msg("no organization found in cache")
			return nil, eris.New("organization not found in cache")
		}

		return nil, eris.Wrapf(err, "failed to get organization %s from cache", orgID)
	}

	log.Debug().Str("key", key).Msg("retrieved organization from cache")
	return org, nil
}

// GetUserOrganizations retrieves the organizations for a user from the cache
//
// Parameters:
//   - ctx: The context of the request
//   - userID: The ID of the user
//
// Returns:
//   - orgs: The organizations
//   - error: An error if the organizations are not found in the cache
func (or *organizationRepository) GetUserOrganizations(ctx context.Context, userID pulid.ID) ([]*organization.Organization, error) {
	log := or.l.With().
		Str("operation", "GetUserOrganizations").
		Str("userID", userID.String()).
		Logger()

	orgs := make([]*organization.Organization, 0)
	key := or.formatUserOrgKey(userID)

	if err := or.cache.GetJSON(ctx, key, &orgs); err != nil {
		if eris.Is(err, redis.ErrNil) {
			log.Debug().Str("key", key).Msg("no organizations found in cache")
		}
	}

	log.Debug().Str("key", key).Msg("retrieved organizations from cache")
	return orgs, nil
}

// SetUserOrganizations sets the organizations for a user in the cache
//
// Parameters:
//   - ctx: The context of the request
//   - userID: The ID of the user
//   - orgs: The organizations
//
// Returns:
//   - error: An error if the organizations are not set in the cache
func (or *organizationRepository) SetUserOrganizations(ctx context.Context, userID pulid.ID, orgs []*organization.Organization) error {
	log := or.l.With().
		Str("operation", "SetUserOrganizations").
		Str("userID", userID.String()).
		Logger()

	key := or.formatUserOrgKey(userID)
	if err := or.cache.SetJSON(ctx, key, orgs, defaultUserOrgTTL); err != nil {
		return eris.Wrapf(err, "failed to set user organizations %s in cache", userID)
	}

	log.Debug().Str("key", key).Msg("stored user organizations in cache")
	return nil
}

// Set the organization in the cache
//
// Parameters:
//   - ctx: The context of the request
//   - org: The organization to set in the cache
//
// Returns:
//   - error: An error if the organization is not found in the cache
func (or *organizationRepository) Set(ctx context.Context, org *organization.Organization) error {
	log := or.l.With().
		Str("operation", "Set").
		Str("orgID", org.ID.String()).
		Logger()

	// * format the key
	key := or.formatKey(org.ID)

	// * Set the organization in the cache
	if err := or.cache.SetJSON(ctx, key, org, defaultOrganizationTTL); err != nil {
		return eris.Wrapf(err, "failed to set organization %s in cache", org.ID)
	}

	log.Debug().Str("key", key).Msg("stored organization in cache")
	return nil
}

// Invalidate invalidates an organization in the cache
//
// Parameters:
//   - ctx: The context of the request
//   - orgID: The ID of the organization
//
// Returns:
//   - error: An error if the organization is not invalidated in the cache
func (or *organizationRepository) Invalidate(ctx context.Context, orgID pulid.ID) error {
	log := or.l.With().
		Str("operation", "Invalidate").
		Str("orgID", orgID.String()).
		Logger()

	key := or.formatKey(orgID)
	if err := or.cache.Del(ctx, key); err != nil {
		return eris.Wrapf(err, "failed to invalidate organization %s in cache", orgID)
	}

	log.Debug().Str("key", key).Msg("invalidated organization in cache")
	return nil
}

// InvalidateUserOrganizations invalidates the organizations for a user in the cache
//
// Parameters:
//   - ctx: The context of the request
//   - userID: The ID of the user
//
// Returns:
//   - error: An error if the organizations are not invalidated in the cache
func (or *organizationRepository) InvalidateUserOrganizations(ctx context.Context, userID pulid.ID) error {
	log := or.l.With().
		Str("operation", "InvalidateUserOrganizations").
		Str("userID", userID.String()).
		Logger()

	key := or.formatUserOrgKey(userID)
	if err := or.cache.Del(ctx, key); err != nil {
		return eris.Wrapf(err, "failed to invalidate user organizations %s in cache", userID)
	}

	log.Debug().Str("key", key).Msg("invalidated user organizations in cache")
	return nil
}

// formatKey formats the key for an organization in the cache
//
// Parameters:
//   - orgID: The ID of the organization
//
// Returns:
//   - key: The key for the organization in the cache
func (or *organizationRepository) formatKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", orgKeyPrefix, orgID)
}

// formatUserOrgKey formats the key for the organizations for a user in the cache
//
// Parameters:
//   - userID: The ID of the user
//
// Returns:
//   - key: The key for the organizations for a user in the cache
func (or *organizationRepository) formatUserOrgKey(userID pulid.ID) string {
	return fmt.Sprintf("%s%s", userOrgKeyPrefix, userID)
}
