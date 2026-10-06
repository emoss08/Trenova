package repositories

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	shipmentBoardPrefix    = "shipment-board"
	shipmentBoardEpochPart = "epoch"
	ShipmentBoardCacheTTL  = 5 * time.Minute
	shipmentBoardEpochTTL  = 24 * time.Hour
)

type ShipmentBoardCacheParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type shipmentBoardCache struct {
	client *redis.Client
	l      *zap.Logger
}

func NewShipmentBoardCache(p ShipmentBoardCacheParams) repositories.ShipmentBoardCache {
	return newShipmentBoardCache(p)
}

func NewShipmentBoardEpochBumper(p ShipmentBoardCacheParams) repositories.ShipmentBoardEpochBumper {
	return newShipmentBoardCache(p)
}

func newShipmentBoardCache(p ShipmentBoardCacheParams) *shipmentBoardCache {
	return &shipmentBoardCache{
		client: p.Client,
		l:      p.Logger.Named("repository.shipment-board-cache"),
	}
}

func ShipmentBoardEpochKey(organizationID, businessUnitID pulid.ID) string {
	return strings.Join([]string{
		shipmentBoardPrefix,
		shipmentBoardEpochPart,
		organizationID.String(),
		businessUnitID.String(),
	}, ":")
}

func ShipmentBoardSectionKey(key *repositories.ShipmentBoardCacheKey) string {
	parts := []string{
		shipmentBoardPrefix,
		string(key.Section),
		key.TenantInfo.OrgID.String(),
		key.TenantInfo.BuID.String(),
		key.Timezone,
		strconv.FormatInt(key.Epoch, 10),
	}
	if key.Variant != "" {
		parts = append(parts, key.Variant)
	}
	return strings.Join(parts, ":")
}

func (c *shipmentBoardCache) BumpEpoch(
	ctx context.Context,
	organizationID, businessUnitID pulid.ID,
) error {
	key := ShipmentBoardEpochKey(organizationID, businessUnitID)
	pipe := c.client.TxPipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, shipmentBoardEpochTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("bump shipment board epoch: %w", err)
	}
	return nil
}

func (c *shipmentBoardCache) Epoch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int64, error) {
	value, err := c.client.Get(ctx, ShipmentBoardEpochKey(tenantInfo.OrgID, tenantInfo.BuID)).
		Int64()
	if err != nil {
		if redishelpers.IsRedisNil(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read shipment board epoch: %w", err)
	}
	return value, nil
}

func (c *shipmentBoardCache) Get(
	ctx context.Context,
	key *repositories.ShipmentBoardCacheKey,
	dest any,
) (bool, error) {
	err := redishelpers.GetStringJSON(ctx, c.client, ShipmentBoardSectionKey(key), dest)
	if err == nil {
		return true, nil
	}
	if redishelpers.IsRedisNil(err) || errors.Is(err, redis.Nil) {
		return false, nil
	}
	return false, fmt.Errorf("read shipment board %s: %w", key.Section, err)
}

func (c *shipmentBoardCache) Set(
	ctx context.Context,
	key *repositories.ShipmentBoardCacheKey,
	value any,
) error {
	if err := redishelpers.SetStringJSON(
		ctx,
		c.client,
		ShipmentBoardSectionKey(key),
		value,
		ShipmentBoardCacheTTL,
	); err != nil {
		return fmt.Errorf("write shipment board %s: %w", key.Section, err)
	}
	return nil
}
