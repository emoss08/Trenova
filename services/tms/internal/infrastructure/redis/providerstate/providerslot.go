package providerstate

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/redis/slotstore"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/redis"
	"go.uber.org/fx"
)

const providerSlotPrefix = "ai:slots"

var errIncompleteSlotRequest = errors.New("provider slot request is incomplete")

type ProviderSlotParams struct {
	fx.In

	Client *redis.Client
}

type providerSlotRepository struct {
	slots *slotstore.Store
}

func NewProviderSlotRepository(p ProviderSlotParams) llmports.ProviderSlotRepository {
	return &providerSlotRepository{slots: slotstore.New(p.Client)}
}

func providerSlotKey(providerID pulid.ID) string {
	return providerSlotPrefix + ":" + providerID.String()
}

func organizationSlotKey(providerID, organizationID pulid.ID) string {
	return providerSlotKey(providerID) + ":org:" + organizationID.String()
}

func bounds(
	providerID pulid.ID,
	limit int,
	organizationID pulid.ID,
	organizationLimit int,
) []slotstore.Bound {
	out := []slotstore.Bound{{Key: providerSlotKey(providerID), Limit: max(limit, 1)}}
	if organizationID.IsNotNil() && organizationLimit > 0 {
		out = append(out, slotstore.Bound{
			Key:   organizationSlotKey(providerID, organizationID),
			Limit: organizationLimit,
		})
	}
	return out
}

func (r *providerSlotRepository) Acquire(
	ctx context.Context,
	req llmports.AcquireProviderSlotRequest,
) (bool, error) {
	if req.ProviderID.IsNil() || req.Token == "" || req.Limit <= 0 || req.TTL <= 0 {
		return false, fmt.Errorf("acquire provider slot: %w", errIncompleteSlotRequest)
	}
	return r.slots.Acquire(ctx, slotstore.Claim{
		Bounds: bounds(req.ProviderID, req.Limit, req.OrganizationID, req.OrganizationLimit),
		Token:  req.Token,
		TTL:    req.TTL,
	})
}

func (r *providerSlotRepository) Refresh(
	ctx context.Context,
	req llmports.ProviderSlotRequest,
) (bool, error) {
	if req.ProviderID.IsNil() || req.Token == "" || req.TTL <= 0 {
		return false, fmt.Errorf("refresh provider slot: %w", errIncompleteSlotRequest)
	}
	return r.slots.Refresh(ctx, slotstore.Claim{
		Bounds: bounds(req.ProviderID, 1, req.OrganizationID, 1),
		Token:  req.Token,
		TTL:    req.TTL,
	})
}

func (r *providerSlotRepository) Release(
	ctx context.Context,
	req llmports.ProviderSlotRequest,
) error {
	if req.ProviderID.IsNil() || req.Token == "" {
		return nil
	}
	return r.slots.Release(ctx, slotstore.Claim{
		Bounds: bounds(req.ProviderID, 1, req.OrganizationID, 1),
		Token:  req.Token,
	})
}
