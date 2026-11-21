package cloudcommandrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand/tenantcommanddb"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

const (
	readReason        = "read a shared server's record of an archive, restore or import command: control plane metadata in a global table, never tenant data"
	writeReason       = "record a shared server's archive, restore or import command and its sealed access: control plane metadata in a global table, never tenant data"
	grantReason       = "keep the nightly archive location the control plane granted a shared server, sealed: control plane metadata in a global table, never tenant data"
	openLimit         = 20
	failuresPerTenant = 20
	failureWindow     = 30 * 24 * 60 * 60
)

type Params struct {
	fx.In

	DB    *postgres.Connection
	Clock timeutils.Clock
}

type repository struct {
	db    *postgres.Connection
	clock timeutils.Clock
}

func New(p Params) cloudtenantports.Commands {
	return &repository{db: p.DB, clock: p.Clock}
}

func (r *repository) Get(ctx context.Context, commandID string) (*tenantcommand.Command, error) {
	ctx = dbscope.WithSystem(ctx, readReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenantcommand.Command, error) {
		cmd, err := tenantcommanddb.Commands.Get(ctx, r.db.Keel(ctx), commandID)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, cloudtenantports.ErrCommandNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("read the command %s: %w", commandID, err)
		}
		return cmd, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	cmd *tenantcommand.Command,
) (*tenantcommand.Command, bool, error) {
	ctx = dbscope.WithSystem(ctx, writeReason)
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*tenantcommand.Command, bool, error) {
		c := tenantcommanddb.Commands
		db := r.db.Keel(ctx)
		inserted, err := c.CreateOrSkip(ctx, db, []*tenantcommand.Command{cmd}, c.CommandID)
		if err != nil {
			return nil, false, fmt.Errorf("record the command %s: %w", cmd.CommandID, err)
		}
		if inserted == 1 {
			return cmd, true, nil
		}
		held, err := c.Get(ctx, db, cmd.CommandID)
		if err != nil {
			return nil, false, fmt.Errorf("read the command %s: %w", cmd.CommandID, err)
		}
		return held, false, nil
	})
}

func (r *repository) Save(ctx context.Context, cmd *tenantcommand.Command) error {
	ctx = dbscope.WithSystem(ctx, writeReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := tenantcommanddb.Commands.Save(ctx, r.db.Keel(ctx), cmd); err != nil {
			return fmt.Errorf("record the command %s: %w", cmd.CommandID, err)
		}
		return nil
	})
}

func (r *repository) Open(
	ctx context.Context,
	instanceID string,
) ([]*tenantcommand.Command, error) {
	ctx = dbscope.WithSystem(ctx, readReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenantcommand.Command, error) {
		c := tenantcommanddb.Commands
		rows, err := c.Where(
			c.InstanceID.Eq(instanceID),
			c.State.In(tenantcommand.StateQueued, tenantcommand.StateRunning),
		).Limit(openLimit).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the tenant's open commands: %w", err)
		}
		return rows, nil
	})
}

func (r *repository) Failures(
	ctx context.Context,
	instanceIDs []string,
) ([]*tenantcommand.Command, error) {
	if len(instanceIDs) == 0 {
		return []*tenantcommand.Command{}, nil
	}
	ctx = dbscope.WithSystem(ctx, readReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*tenantcommand.Command, error) {
		c := tenantcommanddb.Commands
		rows, err := c.Where(
			c.InstanceID.In(instanceIDs...),
			c.State.Eq(tenantcommand.StateFailed),
			c.Kind.In(tenantcommand.KindArchive, tenantcommand.KindImport),
			c.FinishedAt.Gt(failureWindowStart(r.clock())),
		).Order(c.FinishedAt.Desc()).Limit(len(instanceIDs)*failuresPerTenant).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the tenants' failed commands: %w", err)
		}
		return rows, nil
	})
}

func failureWindowStart(now int64) int64 {
	return now - failureWindow
}

func (r *repository) Progress(
	ctx context.Context,
	commandID string,
	progress *tenantcommand.Progress,
) error {
	ctx = dbscope.WithSystem(ctx, writeReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		c := tenantcommanddb.Commands
		if _, err := c.Where(
			c.CommandID.Eq(commandID),
			c.State.Eq(tenantcommand.StateRunning),
		).Update(ctx, r.db.Keel(ctx), c.Progress.To(*progress)); err != nil {
			return fmt.Errorf("record the command's progress: %w", err)
		}
		return nil
	})
}

func (r *repository) SetGrant(ctx context.Context, grant *tenantcommand.Grant) error {
	ctx = dbscope.WithSystem(ctx, grantReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		g := tenantcommanddb.Grants
		if _, err := g.Upsert(ctx, r.db.Keel(ctx), []*tenantcommand.Grant{grant},
			keel.OnConflict(g.Name).Update(g.Location, g.SealedAccess, g.ExpiresAt, g.UpdatedAt),
		); err != nil {
			return fmt.Errorf("record the nightly archive location: %w", err)
		}
		return nil
	})
}

func (r *repository) Grant(ctx context.Context, name string) (*tenantcommand.Grant, error) {
	ctx = dbscope.WithSystem(ctx, grantReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*tenantcommand.Grant, error) {
		grant, err := tenantcommanddb.Grants.Get(ctx, r.db.Keel(ctx), name)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, cloudtenantports.ErrGrantMissing
		}
		if err != nil {
			return nil, fmt.Errorf("read the nightly archive location: %w", err)
		}
		return grant, nil
	})
}
