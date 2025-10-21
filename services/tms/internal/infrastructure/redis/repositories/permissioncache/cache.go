package permissioncache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	cacheKeyPattern = "perm:%s:%s"     // perm:userID:orgID
	defaultL1TTL    = 5 * time.Minute  // Memory cache
	defaultL2TTL    = 15 * time.Minute // Redis cache
	defaultL3TTL    = 30 * time.Minute // Database cache
	maxL1CacheSize  = 10000
)

type Params struct {
	fx.In

	Redis  *redis.Connection
	DB     *postgres.Connection
	Logger *zap.Logger
}

type cache struct {
	redis   *redis.Connection
	db      *postgres.Connection
	logger  *zap.Logger
	l1Cache map[string]*cacheEntry
	l1Mutex sync.RWMutex
	l1TTL   time.Duration
	l2TTL   time.Duration
	l3TTL   time.Duration
}

type cacheEntry struct {
	data      *ports.CachedPermissions
	expiresAt time.Time
}

type permissionCacheRow struct {
	bun.BaseModel  `bun:"table:permission_cache,alias:pc"`
	UserID         pulid.ID `bun:"user_id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	Version        string   `bun:"version"`
	ComputedAt     int64    `bun:"computed_at"`
	ExpiresAt      int64    `bun:"expires_at"`
	PermissionData []byte   `bun:"permission_data"`
	BloomFilter    []byte   `bun:"bloom_filter"`
	Checksum       string   `bun:"checksum"`
}

func NewPermissionRepository(p Params) ports.PermissionCacheRepository {
	c := &cache{
		redis:   p.Redis,
		db:      p.DB,
		logger:  p.Logger.Named("permission-cache"),
		l1Cache: make(map[string]*cacheEntry, maxL1CacheSize),
		l1TTL:   defaultL1TTL,
		l2TTL:   defaultL2TTL,
		l3TTL:   defaultL3TTL,
	}

	go c.l1CleanupLoop()

	return c
}

func (c *cache) Get(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (*ports.CachedPermissions, error) {
	log := c.logger.With(
		zap.String("operation", "Get"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	key := c.buildKey(userID, organizationID)

	if perms := c.getFromL1(key); perms != nil {
		log.Debug("cache hit: L1 (memory)")
		return perms, nil
	}

	perms, err := c.getFromL2(ctx, key)
	if err == nil && perms != nil {
		log.Debug("cache hit: L2 (redis)")
		c.setToL1(key, perms)
		return perms, nil
	}

	perms, err = c.getFromL3(ctx, userID, organizationID)
	if err == nil && perms != nil {
		log.Debug("cache hit: L3 (database)")

		if err = c.setToL2(ctx, key, perms, c.l2TTL); err != nil {
			log.Warn("failed to set L2 cache", zap.Error(err))
		}

		c.setToL1(key, perms)
		return perms, nil
	}

	log.Debug("cache miss: all levels")
	return nil, ErrCacheMiss
}

func (c *cache) Set(
	ctx context.Context,
	userID, organizationID pulid.ID,
	permissions *ports.CachedPermissions,
	ttl time.Duration,
) error {
	log := c.logger.With(
		zap.String("operation", "Set"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	key := c.buildKey(userID, organizationID)

	c.setToL1(key, permissions)

	if err := c.setToL2(ctx, key, permissions, ttl); err != nil {
		log.Warn("failed to set L2 cache", zap.Error(err))
	}

	if err := c.setToL3(ctx, userID, organizationID, permissions); err != nil {
		if ctx.Err() == nil {
			log.Warn("failed to set L3 cache", zap.Error(err))
		}
	}

	log.Debug("cached permissions at all levels")
	return nil
}

func (c *cache) Delete(
	ctx context.Context,
	userID, organizationID pulid.ID,
) error {
	log := c.logger.With(
		zap.String("operation", "Delete"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	key := c.buildKey(userID, organizationID)

	c.deleteFromL1(key)

	if err := c.deleteFromL2(ctx, key); err != nil {
		log.Warn("failed to delete from L2 cache", zap.Error(err))
	}

	if err := c.deleteFromL3(ctx, userID, organizationID); err != nil {
		log.Warn("failed to delete from L3 cache", zap.Error(err))
	}

	log.Debug("deleted permissions from all cache levels")
	return nil
}

func (c *cache) DeletePattern(pattern string) error {
	log := c.logger.With(
		zap.String("operation", "DeletePattern"),
		zap.String("pattern", pattern),
	)

	c.l1Mutex.Lock()
	for key := range c.l1Cache {
		delete(c.l1Cache, key)
	}
	c.l1Mutex.Unlock()

	// ! Pattern deletion from Redis (L2) would require scanning keys
	// ! which is expensive. For now, we clear L1 and rely on TTL for L2/L3
	log.Debug("cleared L1 cache (pattern deletion)")
	return nil
}

func (c *cache) Exists(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (bool, error) {
	key := c.buildKey(userID, organizationID)

	if c.existsInL1(key) {
		return true, nil
	}

	exists, err := c.existsInL2(ctx, key)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	return c.existsInL3(ctx, userID, organizationID)
}

func (c *cache) GetVersion(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (string, error) {
	perms, err := c.Get(ctx, userID, organizationID)
	if err != nil {
		return "", err
	}
	if perms == nil {
		return "", nil
	}
	return perms.Version, nil
}

func (c *cache) getFromL1(key string) *ports.CachedPermissions {
	c.l1Mutex.RLock()
	defer c.l1Mutex.RUnlock()

	entry, ok := c.l1Cache[key]
	if !ok {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.data
}

func (c *cache) setToL1(key string, permissions *ports.CachedPermissions) {
	c.l1Mutex.Lock()
	defer c.l1Mutex.Unlock()

	if len(c.l1Cache) >= maxL1CacheSize {
		c.evictOldestL1()
	}

	c.l1Cache[key] = &cacheEntry{
		data:      permissions,
		expiresAt: time.Now().Add(c.l1TTL),
	}
}

func (c *cache) deleteFromL1(key string) {
	c.l1Mutex.Lock()
	defer c.l1Mutex.Unlock()
	delete(c.l1Cache, key)
}

func (c *cache) existsInL1(key string) bool {
	c.l1Mutex.RLock()
	defer c.l1Mutex.RUnlock()

	entry, ok := c.l1Cache[key]
	if !ok {
		return false
	}

	return time.Now().Before(entry.expiresAt)
}

func (c *cache) evictOldestL1() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.l1Cache {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.l1Cache, oldestKey)
	}
}

func (c *cache) l1CleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanupExpiredL1()
	}
}

func (c *cache) cleanupExpiredL1() {
	c.l1Mutex.Lock()
	defer c.l1Mutex.Unlock()

	now := time.Now()
	for key, entry := range c.l1Cache {
		if now.After(entry.expiresAt) {
			delete(c.l1Cache, key)
		}
	}
}

func (c *cache) getFromL2(ctx context.Context, key string) (*ports.CachedPermissions, error) {
	data, err := c.redis.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var perms ports.CachedPermissions
	if err = sonic.Unmarshal([]byte(data), &perms); err != nil {
		return nil, err
	}

	return &perms, nil
}

func (c *cache) setToL2(
	ctx context.Context,
	key string,
	permissions *ports.CachedPermissions,
	ttl time.Duration,
) error {
	data, err := sonic.Marshal(permissions)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, string(data), ttl)
}

func (c *cache) deleteFromL2(ctx context.Context, key string) error {
	return c.redis.Delete(ctx, key)
}

func (c *cache) existsInL2(ctx context.Context, key string) (bool, error) {
	exists, err := c.redis.Exists(ctx, key)
	return exists > 0, err
}

func (c *cache) getFromL3(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (*ports.CachedPermissions, error) {
	db, err := c.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	row := new(permissionCacheRow)
	err = db.NewSelect().
		Model(&row).
		Where("user_id = ?", userID).
		Where("organization_id = ?", organizationID).
		Where("expires_at > ?", time.Now().Unix()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	var compiledPerms ports.CompiledPermissions
	if err = sonic.Unmarshal(row.PermissionData, &compiledPerms); err != nil {
		return nil, err
	}

	return &ports.CachedPermissions{
		Version:     row.Version,
		ComputedAt:  time.Unix(row.ComputedAt, 0),
		ExpiresAt:   time.Unix(row.ExpiresAt, 0),
		Permissions: &compiledPerms,
		BloomFilter: row.BloomFilter,
		Checksum:    row.Checksum,
	}, nil
}

func (c *cache) setToL3(
	ctx context.Context,
	userID, organizationID pulid.ID,
	permissions *ports.CachedPermissions,
) error {
	db, err := c.db.DB(ctx)
	if err != nil {
		return err
	}

	permData, err := sonic.Marshal(permissions.Permissions)
	if err != nil {
		return err
	}

	row := &permissionCacheRow{
		UserID:         userID,
		OrganizationID: organizationID,
		Version:        permissions.Version,
		ComputedAt:     permissions.ComputedAt.Unix(),
		ExpiresAt:      permissions.ExpiresAt.Unix(),
		PermissionData: permData,
		BloomFilter:    permissions.BloomFilter,
		Checksum:       permissions.Checksum,
	}

	_, err = db.NewInsert().
		Model(row).
		On("CONFLICT (user_id, organization_id) DO UPDATE").
		Set("version = EXCLUDED.version").
		Set("computed_at = EXCLUDED.computed_at").
		Set("expires_at = EXCLUDED.expires_at").
		Set("permission_data = EXCLUDED.permission_data").
		Set("bloom_filter = EXCLUDED.bloom_filter").
		Set("checksum = EXCLUDED.checksum").
		Exec(ctx)

	return err
}

func (c *cache) deleteFromL3(
	ctx context.Context,
	userID, organizationID pulid.ID,
) error {
	db, err := c.db.DB(ctx)
	if err != nil {
		return err
	}

	_, err = db.NewDelete().
		Model((*permissionCacheRow)(nil)).
		Where("user_id = ?", userID).
		Where("organization_id = ?", organizationID).
		Exec(ctx)

	return err
}

func (c *cache) existsInL3(
	ctx context.Context,
	userID, organizationID pulid.ID,
) (bool, error) {
	db, err := c.db.DB(ctx)
	if err != nil {
		return false, err
	}

	exists, err := db.NewSelect().
		Model((*permissionCacheRow)(nil)).
		Where("user_id = ?", userID).
		Where("organization_id = ?", organizationID).
		Where("expires_at > ?", time.Now().Unix()).
		Exists(ctx)

	return exists, err
}

func (c *cache) buildKey(userID, organizationID pulid.ID) string {
	return fmt.Sprintf(cacheKeyPattern, userID.String(), organizationID.String())
}
