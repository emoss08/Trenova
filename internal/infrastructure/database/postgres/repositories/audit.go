package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type AuditRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type auditRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewAuditRepository(p AuditRepositoryParams) repositories.AuditRepository {
	log := p.Logger.With().
		Str("repository", "audit").
		Str("component", "database").
		Logger()

	return &auditRepository{
		db: p.DB,
		l:  &log,
	}
}

func (ar *auditRepository) InsertAuditEntries(ctx context.Context, entries []*audit.Entry) error {
	dba, err := ar.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}
	log := ar.l.With().
		Str("operation", "InsertAuditEntries").
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		_, err = tx.NewInsert().Model(&entries).Exec(c)
		if err != nil {
			ar.l.Error().Err(err).Msg("failed to insert audit entries")
			return eris.Wrap(err, "failed to insert audit entries")
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to insert audit entries")
		return err
	}

	return nil
}
