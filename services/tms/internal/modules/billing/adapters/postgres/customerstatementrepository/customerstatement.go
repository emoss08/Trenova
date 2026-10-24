package customerstatementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerstatement/customerstatementdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) customerstatementports.Repository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.customer-statement-repository")}
}

func (r *repository) ListStatementCustomers(
	ctx context.Context,
	req customerstatementports.ListStatementCustomersRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		cle := customerledgerdb.CustomerLedgerEntries
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), cle.
			Pick(cle.CustomerID).
			Where(cle.TransactionDate.Lte(req.Period.End)).
			Group(cle.CustomerID).
			Having(keel.Or(
				keel.Sum(cle.AmountMinor).Ne(0),
				cle.TransactionDate.Max().Gte(req.Period.Start),
			)).
			Order(cle.CustomerID.Asc()))
		if err != nil {
			return nil, fmt.Errorf("list statement customers: %w", err)
		}
		return ids, nil
	})
}

//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
func (r *repository) ListMonthlySent(
	ctx context.Context,
	req customerstatementports.ListMonthlySentRequest,
) (map[pulid.ID]bool, error) {
	sent := make(map[pulid.ID]bool, len(req.CustomerIDs))
	if len(req.CustomerIDs) == 0 {
		return sent, nil
	}
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]bool, error) {
		cst := customerstatementdb.Statements
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), cst.
			Pick(cst.CustomerID).
			Where(
				cst.Monthly.IsTrue(),
				cst.PeriodEnd.Eq(req.PeriodEnd),
				cst.CustomerID.In(req.CustomerIDs...),
			))
		if err != nil {
			return nil, fmt.Errorf("list monthly statements sent: %w", err)
		}
		for _, id := range ids {
			sent[id] = true
		}
		return sent, nil
	})
}

func (r *repository) Insert(
	ctx context.Context,
	statement *customerstatement.Statement,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		inserted, err := customerstatementdb.Statements.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*customerstatement.Statement{statement},
		)
		if err != nil {
			return false, fmt.Errorf("insert customer statement: %w", err)
		}
		return inserted > 0, nil
	})
}

func (r *repository) ListSent(
	ctx context.Context,
	req customerstatementports.ListSentRequest,
) ([]*customerstatementports.SentStatement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerstatementports.SentStatement, error) {
			cst := customerstatementdb.Statements
			usr := tenantdb.Users.As("usr")
			rows, err := keel.Into[customerstatementports.SentStatement](
				ctx,
				r.db.Keel(ctx),
				cst.
					LeftJoin(usr, usr.ID.EqCol(cst.SentByID)).
					Pick(
						cst.ID,
						cst.PeriodStart,
						cst.PeriodEnd,
						cst.EndingBalanceMinor,
						cst.PastDueMinor,
						cst.CurrencyCode,
						cst.SentTo,
						cst.Monthly,
						usr.Name.Coalesce(""),
						cst.SentAt,
					).
					Where(cst.CustomerID.Eq(req.CustomerID)).
					Order(cst.SentAt.Desc(), cst.ID.Desc()).
					Limit(customerstatementports.MaxHistory),
			)
			if err != nil {
				return nil, fmt.Errorf("list sent customer statements: %w", err)
			}

			sent := make([]*customerstatementports.SentStatement, 0, len(rows))
			for i := range rows {
				sent = append(sent, &rows[i])
			}
			return sent, nil
		},
	)
}
