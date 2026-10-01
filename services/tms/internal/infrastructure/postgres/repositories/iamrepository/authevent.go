package iamrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type authEventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAuthEventRepository(p Params) repositories.AuthEventRepository {
	return &authEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.auth-event-repository"),
	}
}

func (r *authEventRepository) Create(ctx context.Context, event *iam.AuthEvent) error {
	if event.OrganizationID.IsNil() {
		ctx = dbscope.WithSystem(ctx, "record an authentication attempt that matched no organization")
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(event).Exec(ctx); err != nil {
			r.l.Error("failed to insert auth event", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *authEventRepository) DeleteBefore(ctx context.Context, before int64) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "delete authentication events past the retention period for every organization")

	var deleted int64
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if err := postgres.EnableAuditRetention(ctx, tx); err != nil {
			return err
		}

		result, err := tx.NewDelete().
			Model((*iam.AuthEvent)(nil)).
			Where(buncolgen.AuthEventColumns.OccurredAt.Lt(), before).
			Exec(ctx)
		if err != nil {
			return err
		}

		deleted, err = result.RowsAffected()
		return err
	})
	if err != nil {
		r.l.Error("failed to delete expired auth events", zap.Error(err))
		return 0, err
	}

	return deleted, nil
}
