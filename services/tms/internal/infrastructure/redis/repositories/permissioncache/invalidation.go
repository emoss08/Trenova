package permissioncache

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/sourcegraph/conc"
	"go.uber.org/zap"
)

// InvalidationEvent represents a cache invalidation event
type InvalidationEvent struct {
	Type           InvalidationType
	UserID         *pulid.ID
	OrganizationID *pulid.ID
	BusinessUnitID *pulid.ID
	PolicyID       *pulid.ID
	RoleID         *pulid.ID
	Timestamp      time.Time
	Reason         string
}

// InvalidationType defines types of cache invalidation
type InvalidationType string

const (
	InvalidationTypeUser         InvalidationType = "user"          // Specific user
	InvalidationTypeOrganization InvalidationType = "organization"  // All users in org
	InvalidationTypeBusinessUnit InvalidationType = "business_unit" // All users in BU
	InvalidationTypePolicy       InvalidationType = "policy"        // Users affected by policy
	InvalidationTypeRole         InvalidationType = "role"          // Users with role
	InvalidationTypeGlobal       InvalidationType = "global"        // All users
)

type InvalidationStrategy struct {
	Cascade           bool
	MaxConcurrent     int
	WaitForCompletion bool
	PublishEvent      bool
}

func DefaultInvalidationStrategy() *InvalidationStrategy {
	return &InvalidationStrategy{
		Cascade:           true,
		MaxConcurrent:     10,
		WaitForCompletion: true,
		PublishEvent:      true,
	}
}

func (c *cache) InvalidateUser(
	ctx context.Context,
	userID, organizationID pulid.ID,
	strategy *InvalidationStrategy,
) error {
	log := c.logger.With(
		zap.String("operation", "InvalidateUser"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	if strategy == nil {
		strategy = DefaultInvalidationStrategy()
	}

	if err := c.Delete(ctx, userID, organizationID); err != nil {
		log.Error("failed to invalidate user cache", zap.Error(err))
		return err
	}

	if strategy.PublishEvent {
		event := &InvalidationEvent{
			Type:           InvalidationTypeUser,
			UserID:         &userID,
			OrganizationID: &organizationID,
			Timestamp:      time.Now(),
			Reason:         "user_invalidation",
		}
		c.publishInvalidationEvent(ctx, event)
	}

	log.Debug("invalidated user cache")
	return nil
}

func (c *cache) InvalidateOrganization(
	ctx context.Context,
	organizationID pulid.ID,
	strategy *InvalidationStrategy,
) error {
	log := c.logger.With(
		zap.String("operation", "InvalidateOrganization"),
		zap.String("orgID", organizationID.String()),
	)

	if strategy == nil {
		strategy = DefaultInvalidationStrategy()
	}

	userIDs, err := c.getOrganizationUsers(ctx, organizationID)
	if err != nil {
		log.Error("failed to get organization users", zap.Error(err))
		return err
	}

	log.Info("invalidating organization cache", zap.Int("userCount", len(userIDs)))

	c.invalidateUsers(ctx, userIDs, organizationID, strategy)

	if strategy.PublishEvent {
		event := &InvalidationEvent{
			Type:           InvalidationTypeOrganization,
			OrganizationID: &organizationID,
			Timestamp:      time.Now(),
			Reason:         "organization_invalidation",
		}
		c.publishInvalidationEvent(ctx, event)
	}

	log.Info("invalidated organization cache", zap.Int("userCount", len(userIDs)))
	return nil
}

func (c *cache) InvalidateBusinessUnit(
	ctx context.Context,
	businessUnitID pulid.ID,
	strategy *InvalidationStrategy,
) error {
	log := c.logger.With(
		zap.String("operation", "InvalidateBusinessUnit"),
		zap.String("buID", businessUnitID.String()),
	)

	if strategy == nil {
		strategy = DefaultInvalidationStrategy()
	}

	orgIDs, err := c.getBusinessUnitOrganizations(ctx, businessUnitID)
	if err != nil {
		log.Error("failed to get business unit organizations", zap.Error(err))
		return err
	}

	log.Info("invalidating business unit cache", zap.Int("orgCount", len(orgIDs)))

	var wg conc.WaitGroup
	for _, orgID := range orgIDs {
		wg.Go(func() {
			if err = c.InvalidateOrganization(ctx, orgID, strategy); err != nil {
				log.Warn("failed to invalidate organization",
					zap.String("orgID", orgID.String()),
					zap.Error(err),
				)
			}
		})
	}

	if strategy.WaitForCompletion {
		wg.Wait()
	}

	if strategy.PublishEvent {
		event := &InvalidationEvent{
			Type:           InvalidationTypeBusinessUnit,
			BusinessUnitID: &businessUnitID,
			Timestamp:      time.Now(),
			Reason:         "business_unit_invalidation",
		}
		c.publishInvalidationEvent(ctx, event)
	}

	log.Info("invalidated business unit cache")
	return nil
}

func (c *cache) InvalidateByPolicy(
	ctx context.Context,
	policyID pulid.ID,
	strategy *InvalidationStrategy,
) error {
	log := c.logger.With(
		zap.String("operation", "InvalidateByPolicy"),
		zap.String("policyID", policyID.String()),
	)

	if strategy == nil {
		strategy = DefaultInvalidationStrategy()
	}

	userOrgs, err := c.getUsersAffectedByPolicy(ctx, policyID)
	if err != nil {
		log.Error("failed to get users affected by policy", zap.Error(err))
		return err
	}

	log.Info("invalidating policy-affected users", zap.Int("count", len(userOrgs)))

	c.invalidateUserOrgs(ctx, userOrgs, strategy)

	if strategy.PublishEvent {
		event := &InvalidationEvent{
			Type:      InvalidationTypePolicy,
			PolicyID:  &policyID,
			Timestamp: time.Now(),
			Reason:    "policy_update",
		}
		c.publishInvalidationEvent(ctx, event)
	}

	log.Info("invalidated policy-affected users")
	return nil
}

func (c *cache) InvalidateByRole(
	ctx context.Context,
	roleID pulid.ID,
	strategy *InvalidationStrategy,
) error {
	log := c.logger.With(
		zap.String("operation", "InvalidateByRole"),
		zap.String("roleID", roleID.String()),
	)

	if strategy == nil {
		strategy = DefaultInvalidationStrategy()
	}

	userOrgs, err := c.getUsersWithRole(ctx, roleID)
	if err != nil {
		log.Error("failed to get users with role", zap.Error(err))
		return err
	}

	log.Info("invalidating role users", zap.Int("count", len(userOrgs)))

	c.invalidateUserOrgs(ctx, userOrgs, strategy)

	if strategy.PublishEvent {
		event := &InvalidationEvent{
			Type:      InvalidationTypeRole,
			RoleID:    &roleID,
			Timestamp: time.Now(),
			Reason:    "role_update",
		}
		c.publishInvalidationEvent(ctx, event)
	}

	log.Info("invalidated role users")
	return nil
}

func (c *cache) InvalidateAll(
	ctx context.Context,
	strategy *InvalidationStrategy,
) error {
	log := c.logger.With(zap.String("operation", "InvalidateAll"))

	if strategy == nil {
		strategy = DefaultInvalidationStrategy()
	}

	log.Warn("invalidating all permission caches")

	c.l1Mutex.Lock()
	c.l1Cache = make(map[string]*cacheEntry, maxL1CacheSize)
	c.l1Mutex.Unlock()
	log.Warn("L1 cache cleared, L2/L3 will expire naturally via TTL")

	if strategy.PublishEvent {
		event := &InvalidationEvent{
			Type:      InvalidationTypeGlobal,
			Timestamp: time.Now(),
			Reason:    "global_invalidation",
		}
		c.publishInvalidationEvent(ctx, event)
	}

	log.Info("completed global cache invalidation")
	return nil
}

type userOrg struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
}

func (c *cache) invalidateUsers(
	ctx context.Context,
	userIDs []pulid.ID,
	organizationID pulid.ID,
	strategy *InvalidationStrategy,
) {
	semaphore := make(chan struct{}, strategy.MaxConcurrent)
	var wg conc.WaitGroup

	for _, userID := range userIDs {
		semaphore <- struct{}{} // Acquire

		wg.Go(func() {
			defer func() { <-semaphore }() // Release

			if err := c.Delete(ctx, userID, organizationID); err != nil {
				c.logger.Warn("failed to invalidate user",
					zap.String("userID", userID.String()),
					zap.Error(err),
				)
			}
		})
	}

	if strategy.WaitForCompletion {
		wg.Wait()
	}
}

func (c *cache) invalidateUserOrgs(
	ctx context.Context,
	userOrgs []userOrg,
	strategy *InvalidationStrategy,
) {
	semaphore := make(chan struct{}, strategy.MaxConcurrent)
	var wg conc.WaitGroup

	for _, uo := range userOrgs {
		semaphore <- struct{}{} // Acquire

		wg.Go(func() {
			defer func() { <-semaphore }() // Release

			if err := c.Delete(ctx, uo.UserID, uo.OrganizationID); err != nil {
				c.logger.Warn("failed to invalidate user",
					zap.String("userID", uo.UserID.String()),
					zap.String("orgID", uo.OrganizationID.String()),
					zap.Error(err),
				)
			}
		})
	}

	if strategy.WaitForCompletion {
		wg.Wait()
	}
}

func (c *cache) getOrganizationUsers(
	ctx context.Context,
	organizationID pulid.ID,
) ([]pulid.ID, error) {
	db, err := c.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	var userIDs []pulid.ID
	err = db.NewSelect().
		Column("user_id").
		Table("user_organization_memberships").
		Where("organization_id = ?", organizationID).
		Scan(ctx, &userIDs)

	return userIDs, err
}

func (c *cache) getBusinessUnitOrganizations(
	ctx context.Context,
	businessUnitID pulid.ID,
) ([]pulid.ID, error) {
	db, err := c.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	var orgIDs []pulid.ID
	err = db.NewSelect().
		Column("id").
		Table("organizations").
		Where("business_unit_id = ?", businessUnitID).
		Scan(ctx, &orgIDs)

	return orgIDs, err
}

func (c *cache) getUsersAffectedByPolicy(
	ctx context.Context,
	policyID pulid.ID,
) ([]userOrg, error) {
	db, err := c.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	var userOrgs []userOrg
	err = db.NewSelect().
		ColumnExpr("user_id").
		ColumnExpr("organization_id").
		Table("user_effective_policies").
		Where("policy_id = ?", policyID).
		Scan(ctx, &userOrgs)

	return userOrgs, err
}

func (c *cache) getUsersWithRole(
	ctx context.Context,
	roleID pulid.ID,
) ([]userOrg, error) {
	db, err := c.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	var userOrgs []userOrg
	err = db.NewSelect().
		ColumnExpr("user_id").
		ColumnExpr("organization_id").
		Table("user_organization_roles").
		Where("role_id = ?", roleID).
		Scan(ctx, &userOrgs)

	return userOrgs, err
}

func (c *cache) publishInvalidationEvent(_ context.Context, event *InvalidationEvent) {
	channel := "permission:invalidation"

	c.logger.Debug("publishing invalidation event",
		zap.String("channel", channel),
		zap.String("type", string(event.Type)),
		zap.String("reason", event.Reason),
	)
}
