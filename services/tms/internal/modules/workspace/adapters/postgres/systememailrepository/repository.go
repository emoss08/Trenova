package systememailrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail/systememaildb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) emailports.SystemSenderRepository {
	return &repository{db: p.DB}
}

func (r *repository) Get(ctx context.Context) (*systememail.Sender, error) {
	ctx = dbscope.WithSystem(ctx, "read the system email sender Trenova set for this instance")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*systememail.Sender, error) {
		sender, err := systememaildb.Senders.Get(ctx, r.db.Keel(ctx), systememail.CurrentID)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, systememail.ErrNoSender
		}
		if err != nil {
			return nil, fmt.Errorf("read the system email sender: %w", err)
		}
		return sender, nil
	})
}

func (r *repository) Save(ctx context.Context, sender *systememail.Sender) error {
	ctx = dbscope.WithSystem(ctx, "record the system email sender Trenova set for this instance")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := systememaildb.Senders
		sender.ID = systememail.CurrentID
		if _, err := s.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*systememail.Sender{sender},
			keel.OnConflict(s.ID).Update(
				s.Provider,
				s.FromAddress,
				s.FromName,
				s.APIKeySealed,
				s.Fingerprint,
				s.ReceivedAt,
			),
		); err != nil {
			return fmt.Errorf("record the system email sender: %w", err)
		}
		return nil
	})
}

func (r *repository) Delete(ctx context.Context) (bool, error) {
	ctx = dbscope.WithSystem(ctx, "remove the system email sender Trenova set for this instance")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		s := systememaildb.Senders
		rows, err := s.Where(s.ID.Eq(systememail.CurrentID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("remove the system email sender: %w", err)
		}
		return rows > 0, nil
	})
}
