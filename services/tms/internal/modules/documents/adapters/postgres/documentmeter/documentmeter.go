package documentmeter

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Result struct {
	fx.Out

	Reader managedusageports.LevelReader `group:"managed_level_readers"`
}

type storage struct {
	db *postgres.Connection
}

func New(p Params) Result {
	return Result{Reader: &storage{db: p.DB}}
}

func (s *storage) Meter() platformcatalog.MeterKey {
	return platformcatalog.MeterDocumentStorageBytes
}

func (s *storage) Level(ctx context.Context) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"add up the bytes of every stored document on this managed instance",
	)
	return dbtx.Read(ctx, s.db, func(ctx context.Context) (int64, error) {
		d := documentdb.Documents
		bytes, err := keel.Into[int64](
			ctx,
			s.db.Keel(ctx),
			d.AllTenants().Pick(keel.CastInt64(keel.Int64Of(d.FileSize.Sum()).Coalesce(0))),
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
