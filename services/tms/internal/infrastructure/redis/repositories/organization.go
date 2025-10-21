package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultUserOrgTTL      = 1 * time.Hour  // * 1 hour
	defaultOrganizationTTL = 24 * time.Hour // * 24 hours
	orgKeyPrefix           = "org:"
	userOrgKeyPrefix       = "user_orgs:"
	orgMembersKeyPrefix    = "org_members:" // Track users in an organization
)

type OrganizationRepositoryParams struct {
	fx.In

	Cache  *redis.Connection
	Logger *zap.Logger
}

type organizationRepository struct {
	cache *redis.Connection
	l     *zap.Logger
}

func NewOrganizationRepository(
	p OrganizationRepositoryParams,
) repositories.OrganizationCacheRepository {
	return &organizationRepository{
		cache: p.Cache,
		l:     p.Logger.Named("redis.organization-repository"),
	}
}

func (or *organizationRepository) GetByID(
	ctx context.Context,
	orgID pulid.ID,
) (*tenant.Organization, error) {
	log := or.l.With(
		zap.String("operation", "GetByID"),
		zap.String("orgID", orgID.String()),
	)

	org := new(tenant.Organization)
	key := or.formatKey(orgID)
	if err := or.cache.GetJSON(ctx, key, org); err != nil {
		log.Error("failed to get organization from cache", zap.Error(err))
		return nil, err
	}

	log.Debug("retrieved organization from cache", zap.String("key", key))
	return org, nil
}

func (or *organizationRepository) GetUserOrganizations(
	ctx context.Context,
	userID pulid.ID,
) ([]*tenant.Organization, error) {
	log := or.l.With(
		zap.String("operation", "GetUserOrganizations"),
		zap.String("userID", userID.String()),
	)

	orgs := make([]*tenant.Organization, 0)
	key := or.formatUserOrgKey(userID)

	if err := or.cache.GetJSON(ctx, key, &orgs); err != nil {
		log.Warn("failed to get user organizations from cache", zap.Error(err))
		// ! Do not return an error because it will not affect the user experience
	}

	log.Debug("retrieved organizations from cache", zap.String("key", key))
	return orgs, nil
}

func (or *organizationRepository) SetUserOrganizations(
	ctx context.Context,
	userID pulid.ID,
	orgs []*tenant.Organization,
) error {
	log := or.l.With(
		zap.String("operation", "SetUserOrganizations"),
		zap.String("userID", userID.String()),
	)

	key := or.formatUserOrgKey(userID)
	if err := or.cache.SetJSON(ctx, key, orgs, defaultUserOrgTTL); err != nil {
		log.Error("failed to set user organizations in cache", zap.Error(err))
		return err
	}

	for _, org := range orgs {
		if err := or.addOrganizationMember(ctx, org.ID, userID); err != nil {
			log.Warn("failed to track organization member",
				zap.String("orgID", org.ID.String()),
				zap.String("userID", userID.String()),
				zap.Error(err))
		}
	}

	log.Debug("stored user organizations in cache", zap.String("key", key))
	return nil
}

func (or *organizationRepository) Set(ctx context.Context, org *tenant.Organization) error {
	log := or.l.With(
		zap.String("operation", "Set"),
		zap.String("orgID", org.ID.String()),
	)

	key := or.formatKey(org.ID)
	if err := or.cache.SetJSON(ctx, key, org, defaultOrganizationTTL); err != nil {
		log.Error("failed to set organization in cache", zap.Error(err))
		return err
	}

	log.Debug("stored organization in cache", zap.String("key", key))
	return nil
}

func (or *organizationRepository) Invalidate(ctx context.Context, orgID pulid.ID) error {
	log := or.l.With(
		zap.String("operation", "Invalidate"),
		zap.String("orgID", orgID.String()),
	)

	key := or.formatKey(orgID)
	if err := or.cache.Delete(ctx, key); err != nil {
		log.Error("failed to invalidate organization in cache", zap.Error(err))
		return err
	}

	log.Debug("invalidated organization in cache", zap.String("key", key))
	return nil
}

func (or *organizationRepository) InvalidateUserOrganizations(
	ctx context.Context,
	userID pulid.ID,
) error {
	log := or.l.With(
		zap.String("operation", "InvalidateUserOrganizations"),
		zap.String("userID", userID.String()),
	)

	key := or.formatUserOrgKey(userID)
	if err := or.cache.Delete(ctx, key); err != nil {
		return err
	}

	log.Debug("invalidated user organizations in cache", zap.String("key", key))
	return nil
}

func (or *organizationRepository) InvalidateOrganizationForAllUsers(
	ctx context.Context,
	orgID pulid.ID,
) error {
	log := or.l.With(
		zap.String("operation", "InvalidateOrganizationForAllUsers"),
		zap.String("orgID", orgID.String()),
	)

	if err := or.Invalidate(ctx, orgID); err != nil {
		log.Error("failed to invalidate organization cache", zap.Error(err))
		return err
	}

	membersKey := or.formatOrgMembersKey(orgID)
	members, err := or.cache.SMembers(ctx, membersKey)
	if err != nil {
		log.Warn(
			"failed to get organization members, skipping user cache invalidation",
			zap.Error(err),
		)
		// ! Do not return an error because it will not affect the user experience
		return nil
	}

	invalidatedCount := 0
	for _, memberIDStr := range members {
		memberID, mErr := pulid.Parse(memberIDStr)
		if mErr != nil {
			log.Warn("invalid member ID in set",
				zap.String("memberID", memberIDStr),
				zap.Error(mErr))
			continue
		}

		userOrgKey := or.formatUserOrgKey(memberID)
		if err = or.cache.Delete(ctx, userOrgKey); err != nil {
			log.Warn("failed to invalidate user organization cache",
				zap.String("userID", memberIDStr),
				zap.Error(err))
			continue
		}
		invalidatedCount++
	}

	log.Info("invalidated organization cache for all users",
		zap.Int("totalMembers", len(members)),
		zap.Int("invalidatedCount", invalidatedCount))

	return nil
}

func (or *organizationRepository) addOrganizationMember(
	ctx context.Context,
	orgID, userID pulid.ID,
) error {
	key := or.formatOrgMembersKey(orgID)
	return or.cache.SAdd(ctx, key, userID.String())
}

// func (or *organizationRepository) removeOrganizationMember(
// 	ctx context.Context,
// 	orgID, userID pulid.ID,
// ) error {
// 	key := or.formatOrgMembersKey(orgID)
// 	return or.cache.SRem(ctx, key, userID.String())
// }

func (or *organizationRepository) formatKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", orgKeyPrefix, orgID)
}

func (or *organizationRepository) formatUserOrgKey(userID pulid.ID) string {
	return fmt.Sprintf("%s%s", userOrgKeyPrefix, userID)
}

func (or *organizationRepository) formatOrgMembersKey(orgID pulid.ID) string {
	return fmt.Sprintf("%s%s", orgMembersKeyPrefix, orgID)
}
