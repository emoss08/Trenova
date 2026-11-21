package archivejobs

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/cloud/app/archiveengine"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

const maxNightlyScans = 20

type NightlyInput struct {
	InstanceID     string   `json:"instanceId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Day            string   `json:"day"`
}

func (a *Activities) ListNightlyTenantsActivity(
	ctx context.Context,
	after *temporaljobs.TenantWorkItem,
) (*temporaljobs.TenantPage, error) {
	cursor := ""
	if after != nil {
		cursor = after.Cursor
	}
	out := &temporaljobs.TenantPage{Tenants: make([]temporaljobs.TenantWorkItem, 0)}
	for range maxNightlyScans {
		page, err := a.lifecycle.NightlyTargets(ctx, cursor, nightlyPageSize)
		if err != nil {
			return nil, err
		}
		for _, target := range page.Targets {
			out.Tenants = append(out.Tenants, temporaljobs.TenantWorkItem{
				OrganizationID: target.OrganizationID,
				BusinessUnitID: target.BusinessUnitID,
				Cursor:         target.InstanceID,
			})
		}
		out.HasMore = page.Next != ""
		if len(out.Tenants) > 0 || !out.HasMore {
			return out, nil
		}
		cursor = page.Next
	}
	return out, nil
}

func (a *Activities) NightlyArchiveActivity(ctx context.Context, input *NightlyInput) (int, error) {
	day, err := time.Parse(time.DateOnly, input.Day)
	if err != nil {
		return 0, temporal.NewNonRetryableApplicationError(
			"the nightly run names no day",
			"BadDay",
			err,
		)
	}
	target, err := a.lifecycle.NightlyTarget(ctx, input.InstanceID)
	if errors.Is(err, tenantlifecycle.ErrTenantState) ||
		errors.Is(err, cloudtenantports.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if target.OrganizationID != input.OrganizationID {
		return 0, nil
	}
	cmd, access, err := a.lifecycle.BeginNightly(ctx, target, day)
	if errors.Is(err, cloudtenantports.ErrGrantMissing) {
		return 0, temporal.NewNonRetryableApplicationError(err.Error(), errorNoNightlyGrant, err)
	}
	if err != nil {
		return 0, err
	}
	if access != nil {
		if err = a.nightly(ctx, cmd.CommandID, target, access); err != nil {
			return 0, err
		}
	}
	grant, err := a.lifecycle.NightlyAccess(ctx)
	if err != nil {
		return 0, err
	}
	if err = a.prune(ctx, target, grant); err != nil {
		return 0, err
	}
	return 1, nil
}

func (a *Activities) nightly(
	ctx context.Context,
	commandID string,
	target *tenantlifecycle.NightlyTarget,
	access *tenantarchive.Access,
) error {
	release, err := a.slots.Hold(ctx, kernelports.HeavyJobArchive)
	if err != nil {
		return fmt.Errorf("wait for an archive slot: %w", err)
	}
	defer release()
	bounded, cancel := a.bounded(ctx)
	defer cancel()
	result, err := a.engine.Export(bounded, &archiveengine.ExportRequest{
		Purpose:    tenantarchive.PurposeNightly,
		InstanceID: target.InstanceID,
		Tenant:     target.Tenant(),
		Access:     access,
		Progress:   a.reporter(ctx, commandID),
		Beat:       heartbeat(ctx),
	})
	if refusal, ok := tenantarchive.RefusalOf(err); ok {
		_, ferr := a.fail(ctx, commandID, refusal, a.lifecycle.FailCommand)
		return ferr
	}
	if err != nil {
		return err
	}
	return a.lifecycle.FinishArchive(ctx, &tenantlifecycle.ArchiveResult{
		CommandID: commandID,
		Location:  result.Location,
		SHA256:    result.SHA256,
		SizeBytes: result.SizeBytes,
		TakenAt:   result.Manifest.TakenAt,
	})
}

type nightlyEntry struct {
	prefix   string
	day      time.Time
	complete bool
}

func (a *Activities) prune(
	ctx context.Context,
	target *tenantlifecycle.NightlyTarget,
	grant *tenantarchive.Access,
) error {
	bucket, err := a.buckets.Open(&grant.Location, &grant.Credentials)
	if err != nil {
		return err
	}
	entries, err := nightlyEntries(ctx, bucket, target.InstanceID)
	if err != nil {
		return err
	}
	for _, prefix := range prunable(entries, target.RetentionDays) {
		if _, err = bucket.DeletePrefix(ctx, prefix); err != nil {
			return fmt.Errorf("prune a nightly archive: %w", err)
		}
		a.l.Info("pruned a nightly tenant archive past its retention",
			zap.String("instanceId", target.InstanceID))
	}
	return nil
}

func nightlyEntries(
	ctx context.Context,
	bucket archiveports.Bucket,
	instanceID string,
) ([]nightlyEntry, error) {
	children, err := bucket.Children(ctx, tenantarchive.NightlyRoot(instanceID))
	if err != nil {
		return nil, err
	}
	out := make([]nightlyEntry, 0, len(children))
	for _, prefix := range children {
		day, ok := tenantarchive.NightlyDay(prefix, instanceID)
		if !ok {
			continue
		}
		complete := false
		if err = bucket.List(
			ctx,
			prefix+tenantarchive.ManifestName,
			func(key string, _ int64) error {
				complete = complete || strings.HasSuffix(key, "/"+tenantarchive.ManifestName)
				return nil
			},
		); err != nil {
			return nil, err
		}
		out = append(out, nightlyEntry{prefix: prefix, day: day, complete: complete})
	}
	return out, nil
}

func prunable(entries []nightlyEntry, keep int64) []string {
	sorted := slices.Clone(entries)
	slices.SortFunc(sorted, func(x, y nightlyEntry) int { return y.day.Compare(x.day) })
	newest := -1
	for i := range sorted {
		if sorted[i].complete {
			newest = i
			break
		}
	}
	if newest < 0 {
		return nil
	}
	out := make([]string, 0)
	kept := int64(0)
	for i := newest; i < len(sorted); i++ {
		e := sorted[i]
		if e.complete && kept < max(keep, 1) {
			kept++
			continue
		}
		out = append(out, e.prefix)
	}
	return out
}
