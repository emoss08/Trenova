package cloudtenantrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant/cloudtenantdb"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	lockReason     = "hold one shared-server tenant's bookkeeping while its control plane entry is applied"
	readReason     = "read a shared server's bookkeeping about its tenants: control plane metadata in a global table, never tenant data"
	writeReason    = "record a shared server's bookkeeping about one tenant: control plane metadata in a global table, never tenant data"
	eventReason    = "record one change to a shared-server tenant's lifecycle in the ledger kept after removal: counts and reasons only"
	instanceReason = "find the control plane instance a shared-server organization is, to tag the system email it sends"
	lockPrefix     = "trenova:cloud-tenant:"
	defaultLimit   = 100
	maxLimit       = 500
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) cloudtenantports.Repository {
	return &repository{db: p.DB}
}

func (r *repository) Lock(ctx context.Context, instanceID string) error {
	ctx = dbscope.WithSystem(ctx, lockReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := keel.LockKey(ctx, r.db.Keel(ctx), lockPrefix+instanceID); err != nil {
			return fmt.Errorf("lock the tenant %s: %w", instanceID, err)
		}
		return nil
	})
}

func (r *repository) Get(ctx context.Context, instanceID string) (*cloudtenant.Tenant, error) {
	ctx = dbscope.WithSystem(ctx, readReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*cloudtenant.Tenant, error) {
		tenant, err := cloudtenantdb.Tenants.Get(ctx, r.db.Keel(ctx), instanceID)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, cloudtenantports.ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("read the tenant %s: %w", instanceID, err)
		}
		return tenant, nil
	})
}

func (r *repository) Save(ctx context.Context, tenant *cloudtenant.Tenant) error {
	ctx = dbscope.WithSystem(ctx, writeReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		t := cloudtenantdb.Tenants
		if _, err := t.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*cloudtenant.Tenant{tenant},
			keel.OnConflict(t.InstanceID).Update(
				t.OrganizationID,
				t.BusinessUnitID,
				t.AdminUserID,
				t.State,
				t.AppliedRevision,
				t.DesiredRevision,
				t.DesiredDigest,
				t.Desired,
				t.InviteSHA256,
				t.LastArchiveCommandID,
				t.LastArchivePurpose,
				t.LastArchiveLocation,
				t.LastArchiveSizeBytes,
				t.LastArchiveSHA256,
				t.LastArchiveTakenAt,
				t.LastRestoreCommandID,
				t.LastRestoreState,
				t.LastRestoreAt,
				t.LastRestoreReason,
				t.LastRestoreNote,
				t.ImportedAt,
				t.RefusedRevision,
				t.RefusedCode,
				t.RefusedReason,
				t.RemovalStartedAt,
				t.RemovedAt,
				t.UpdatedAt,
			),
		); err != nil {
			return fmt.Errorf("record the tenant %s: %w", tenant.InstanceID, err)
		}
		return nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req *cloudtenantports.ListRequest,
) (*cloudtenantports.Page, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	limit = min(limit, maxLimit)
	ctx = dbscope.WithSystem(ctx, readReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*cloudtenantports.Page, error) {
		t := cloudtenantdb.Tenants
		rows, err := t.Where(keel.When(req.After != "", t.InstanceID.Gt(req.After))).
			Order(t.InstanceID.Asc()).
			Limit(limit+1).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list the tenants: %w", err)
		}
		page := &cloudtenantports.Page{Tenants: rows}
		if len(rows) > limit {
			page.Tenants = rows[:limit]
			page.Next = rows[limit-1].InstanceID
		}
		return page, nil
	})
}

func (r *repository) ByInstances(
	ctx context.Context,
	instanceIDs []string,
) ([]*cloudtenant.Tenant, error) {
	if len(instanceIDs) == 0 {
		return []*cloudtenant.Tenant{}, nil
	}
	ctx = dbscope.WithSystem(ctx, readReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*cloudtenant.Tenant, error) {
		t := cloudtenantdb.Tenants
		rows, err := t.Where(t.InstanceID.In(instanceIDs...)).
			Limit(len(instanceIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the tenants by instance: %w", err)
		}
		return rows, nil
	})
}

func (r *repository) InstanceOf(ctx context.Context, organizationID pulid.ID) (string, error) {
	ctx = dbscope.WithSystem(ctx, instanceReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		t := cloudtenantdb.Tenants
		tenant, err := t.Where(t.OrganizationID.Eq(organizationID)).One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return "", cloudtenantports.ErrNotHeld
		}
		if err != nil {
			return "", fmt.Errorf("find the tenant of an organization: %w", err)
		}
		return tenant.InstanceID, nil
	})
}

func (r *repository) Record(ctx context.Context, event *cloudtenant.Event) error {
	ctx = dbscope.WithSystem(ctx, eventReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if event.Detail == nil {
			event.Detail = map[string]any{}
		}
		if err := cloudtenantdb.Events.Create(ctx, r.db.Keel(ctx), event); err != nil {
			return fmt.Errorf("record a tenant event: %w", err)
		}
		return nil
	})
}
