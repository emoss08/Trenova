package tenantjobs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	purgeBatchSize  = 500
	purgeMaxBatches = 200

	errorNotRemoving    = "CloudTenantNotRemoving"
	errorRowsRemain     = "CloudTenantRowsRemain"
	errorNoPrefixDelete = "CloudTenantStorageCannotDeletePrefix"
)

type ActivitiesParams struct {
	fx.In

	Lifecycle *tenantlifecycle.Service
	Purge     cloudtenantports.Purge
	Keys      cloudtenantports.KeyPurge
	Search    cloudtenantports.SearchPurge
	Storage   kernelports.Client
	Reporting *reportingconfig.Config
	Slots     kernelports.HeavyJobSlots
	Logger    *zap.Logger
}

type Activities struct {
	lifecycle *tenantlifecycle.Service
	purge     cloudtenantports.Purge
	keys      cloudtenantports.KeyPurge
	search    cloudtenantports.SearchPurge
	storage   kernelports.Client
	reporting *reportingconfig.Config
	slots     kernelports.HeavyJobSlots
	l         *zap.Logger
}

//nolint:gocritic // this is dependency injection
func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		lifecycle: p.Lifecycle,
		purge:     p.Purge,
		keys:      p.Keys,
		search:    p.Search,
		storage:   p.Storage,
		reporting: p.Reporting,
		slots:     p.Slots,
		l:         p.Logger.Named("cloud.tenant-jobs"),
	}
}

func heartbeat(ctx context.Context) func() {
	return func() {
		if activity.IsActivity(ctx) {
			activity.RecordHeartbeat(ctx)
		}
	}
}

func (a *Activities) CheckRemovalActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) (bool, error) {
	t, err := a.lifecycle.Removing(ctx, payload.InstanceID)
	if errors.Is(err, tenantlifecycle.ErrNotRemoving) {
		return false, temporal.NewNonRetryableApplicationError(err.Error(), errorNotRemoving, err)
	}
	if err != nil {
		return false, err
	}
	return t.State == cloudtenant.StateRemoving, nil
}

func (a *Activities) RevokeSessionsActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) (*tenantlifecycle.RevocationResult, error) {
	return a.lifecycle.RevokeSessions(ctx, &tenantlifecycle.RevocationRequest{
		Tenant: payload.Tenant(),
	}, heartbeat(ctx))
}

func (a *Activities) ReleaseMembersActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) (*tenantlifecycle.RevocationResult, error) {
	return a.lifecycle.RevokeSessions(ctx, &tenantlifecycle.RevocationRequest{
		Tenant:   payload.Tenant(),
		Reassign: true,
	}, heartbeat(ctx))
}

func (a *Activities) PurgeRowsActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) (*cloudtenantports.PurgeRowsResult, error) {
	release, err := a.holdSlot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	beat := heartbeat(ctx)
	beat()
	return a.purge.PurgeRows(ctx, &cloudtenantports.PurgeRowsRequest{
		Tenant:     payload.Tenant(),
		BatchSize:  purgeBatchSize,
		MaxBatches: purgeMaxBatches,
		Beat:       beat,
	})
}

func (a *Activities) holdSlot(ctx context.Context) (func(), error) {
	release, err := a.slots.Hold(ctx, kernelports.HeavyJobRemoval)
	if err != nil {
		return nil, fmt.Errorf("wait for a tenant removal slot: %w", err)
	}
	return release, nil
}

func (a *Activities) DeleteOrganizationActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) (*cloudtenantports.DeleteResult, error) {
	release, err := a.holdSlot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	return a.purge.DeleteOrganization(ctx, payload.Tenant())
}

func (a *Activities) DeleteBusinessUnitActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) (*cloudtenantports.DeleteResult, error) {
	return a.purge.DeleteBusinessUnit(ctx, payload.Tenant())
}

type StoresResult struct {
	Objects       int64    `json:"objects"`
	Prefixes      []string `json:"prefixes"`
	SearchIndexes int      `json:"searchIndexes"`
	CacheKeys     int64    `json:"cacheKeys"`
}

func (a *Activities) PurgeStoresActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) (*StoresResult, error) {
	release, err := a.holdSlot(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	out := &StoresResult{Prefixes: a.prefixes(payload)}
	deleter, ok := a.storage.(kernelports.PrefixDeleter)
	if !ok {
		return nil, temporal.NewNonRetryableApplicationError(
			"the object store cannot delete by prefix, so the tenant's documents cannot be removed",
			errorNoPrefixDelete,
			nil,
		)
	}
	for _, prefix := range out.Prefixes {
		n, derr := deleter.DeletePrefix(ctx, prefix)
		if derr != nil {
			return nil, fmt.Errorf("delete the tenant's objects under a prefix: %w", derr)
		}
		out.Objects += n
		heartbeat(ctx)()
	}
	indexes, err := a.search.DeleteOrganizationDocuments(ctx, payload.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("delete the tenant's search documents: %w", err)
	}
	out.SearchIndexes = indexes
	for _, fragment := range []string{
		payload.OrganizationID.String(),
		payload.BusinessUnitID.String(),
	} {
		n, kerr := a.keys.DeleteMatching(ctx, fragment)
		if kerr != nil {
			return nil, fmt.Errorf("delete the tenant's cache keys: %w", kerr)
		}
		out.CacheKeys += n
	}
	return out, nil
}

func (a *Activities) prefixes(payload *tenantjobscontract.TenantPayload) []string {
	org := payload.OrganizationID.String() + "/"
	prefixes := []string{org}
	if artifacts := strings.Trim(a.reporting.GetArtifactPrefix(), "/"); artifacts != "" {
		prefixes = append(prefixes, artifacts+"/"+org)
	}
	return prefixes
}

func (a *Activities) VerifyRemovedActivity(
	ctx context.Context,
	payload *tenantjobscontract.TenantPayload,
) ([]cloudtenantports.TableCount, error) {
	counts, err := a.purge.Remaining(ctx, payload.Tenant())
	if err != nil {
		return nil, err
	}
	left := make([]cloudtenantports.TableCount, 0)
	for _, c := range counts {
		if c.Rows > 0 {
			left = append(left, c)
		}
	}
	if len(left) > 0 {
		names := make([]string, 0, len(left))
		for _, c := range left {
			names = append(names, c.Table)
		}
		return left, temporal.NewApplicationError(
			"rows of the removed tenant remain in "+strings.Join(names, ", "),
			errorRowsRemain,
		)
	}
	return left, nil
}

func (a *Activities) FinishRemovalActivity(
	ctx context.Context,
	input *FinishInput,
) error {
	return a.lifecycle.FinishRemoval(ctx, input.Payload.InstanceID, map[string]any{
		"rows":          input.Rows,
		"objects":       input.Stores.Objects,
		"searchIndexes": input.Stores.SearchIndexes,
		"cacheKeys":     input.Stores.CacheKeys,
		"sessions":      input.Sessions,
	})
}

type FinishInput struct {
	Payload  *tenantjobscontract.TenantPayload `json:"payload"`
	Rows     int64                             `json:"rows"`
	Stores   *StoresResult                     `json:"stores"`
	Sessions int                               `json:"sessions"`
}
