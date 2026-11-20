package documentmeter

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentstorage/documentstoragedb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Result struct {
	fx.Out

	Reader managedusageports.LevelReader `group:"managed_level_readers"`
	Source datausageports.Source         `group:"managed_data_sources"`
}

const (
	measureTimeout           = 10 * time.Second
	organizationsLevelReason = "add up the stored document bytes of each organization of a shared server, in one statement, for the control plane's usage report; each organization is measured only for itself"
)

type storage struct {
	db *postgres.Connection
}

func New(p Params) Result {
	s := &storage{db: p.DB}
	return Result{Reader: s, Source: s}
}

func (s *storage) Meter() platformcatalog.MeterKey {
	return platformcatalog.MeterDocumentStorageBytes
}

func (s *storage) Figure() datausageports.Figure {
	return datausageports.FigureDocuments
}

func (s *storage) Timeout() time.Duration {
	return measureTimeout
}

func (s *storage) Measure(ctx context.Context) (int64, error) {
	return s.Level(ctx, managedscope.Instance())
}

func (s *storage) Level(ctx context.Context, scope managedscope.Scope) (int64, error) {
	if tenant, ok := scope.Tenant(); ok {
		ctx = dbscope.WithTenant(ctx, tenant)
	} else {
		ctx = dbscope.WithSystem(
			ctx,
			"add up the bytes of every stored document on this managed instance",
		)
	}
	return dbtx.Read(ctx, s.db, func(ctx context.Context) (int64, error) {
		t := documentstoragedb.Totals
		bytes, err := keel.Into[int64](
			ctx,
			s.db.Keel(ctx),
			tenantscope.Scoped(ctx, t.Query()).
				Pick(keel.CastInt64(keel.NumericOf(t.Bytes.Sum()).Coalesce(0))),
		)
		if err != nil {
			return 0, fmt.Errorf("add up stored document bytes: %w", err)
		}
		if len(bytes) == 0 {
			return 0, nil
		}
		return bytes[0], nil
	})
}

func (s *storage) LevelsForOrganizations(
	ctx context.Context,
	organizationIDs []pulid.ID,
) ([]managedusageports.OrganizationLevel, error) {
	if len(organizationIDs) == 0 {
		return []managedusageports.OrganizationLevel{}, nil
	}
	ctx = dbscope.WithSystem(ctx, organizationsLevelReason)
	return dbtx.Read(
		ctx,
		s.db,
		func(ctx context.Context) ([]managedusageports.OrganizationLevel, error) {
			t := documentstoragedb.Totals
			levels, err := keel.Into[managedusageports.OrganizationLevel](
				ctx,
				s.db.Keel(ctx),
				t.Where(t.OrganizationID.In(organizationIDs...)).
					AllTenants().
					Group(t.OrganizationID).
					Pick(
						t.OrganizationID,
						keel.CastInt64(keel.NumericOf(t.Bytes.Sum()).Coalesce(0)),
					),
			)
			if err != nil {
				return nil, fmt.Errorf("add up stored document bytes by organization: %w", err)
			}
			return levels, nil
		},
	)
}
