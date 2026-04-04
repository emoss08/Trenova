package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	ports "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/redishelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const shipmentImportChatCacheTTL = 24 * time.Hour

type ShipmentImportChatCacheRepositoryParams struct {
	fx.In

	Client *redis.Client
	Logger *zap.Logger
}

type shipmentImportChatCacheRepository struct {
	client *redis.Client
	l      *zap.Logger
}

func NewShipmentImportChatCacheRepository(
	p ShipmentImportChatCacheRepositoryParams,
) ports.ShipmentImportChatCacheRepository {
	return &shipmentImportChatCacheRepository{
		client: p.Client,
		l:      p.Logger.Named("redis.shipment-import-chat-cache-repository"),
	}
}

func (r *shipmentImportChatCacheRepository) GetHistory(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentimportchat.HistorySnapshot, error) {
	entity := new(shipmentimportchat.HistorySnapshot)
	if err := redishelpers.GetJSON(ctx, r.client, r.key(documentID, tenantInfo), entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *shipmentImportChatCacheRepository) SetHistory(
	ctx context.Context,
	snapshot *shipmentimportchat.HistorySnapshot,
	tenantInfo pagination.TenantInfo,
) error {
	documentID, err := pulid.Parse(snapshot.DocumentID)
	if err != nil {
		return err
	}

	return redishelpers.SetJSON(ctx, r.client, r.key(documentID, tenantInfo), snapshot, shipmentImportChatCacheTTL)
}

func (r *shipmentImportChatCacheRepository) DeleteHistory(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	return r.client.Del(ctx, r.key(documentID, tenantInfo)).Err()
}

func (r *shipmentImportChatCacheRepository) key(
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) string {
	return fmt.Sprintf("cache:shipment-import-chat:%s:%s:%s", tenantInfo.OrgID, tenantInfo.BuID, documentID)
}

var _ ports.ShipmentImportChatCacheRepository = (*shipmentImportChatCacheRepository)(nil)
