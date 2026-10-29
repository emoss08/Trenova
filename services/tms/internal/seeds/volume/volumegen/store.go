package volumegen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glbalance/glbalancedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol/accountingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/shared/pulid"
)

const BackdateChunk = 5_000

var errUnknownTable = errors.New("no table handle to backdate")

type Stamp struct {
	Id      pulid.ID
	Created int64
	Updated int64
}

func IDAt(prefix string, unix int64) pulid.ID {
	return pulid.MustNewAt(prefix, time.Unix(unix, 0))
}

func InsertRows[T any](ctx context.Context, db *keel.DB, h keel.Handle[*T], rows []*T) error {
	if len(rows) == 0 {
		return nil
	}
	if _, err := h.Copy(ctx, db, rows); err != nil {
		return fmt.Errorf("copy into %s: %w", h.Ref.Table.Name, err)
	}

	return nil
}

func Backdate[T any](ctx context.Context, db *keel.DB, h keel.Handle[*T], stamps []Stamp) error {
	return BackdateColumns(ctx, db, h, stamps, false)
}

func backdateCreated[T any](
	ctx context.Context,
	db *keel.DB,
	h keel.Handle[*T],
	stamps []Stamp,
) error {
	return BackdateColumns(ctx, db, h, stamps, true)
}

func BackdateColumns[T any](
	ctx context.Context,
	db *keel.DB,
	h keel.Handle[*T],
	stamps []Stamp,
	createdOnly bool,
) error {
	if len(stamps) == 0 {
		return nil
	}
	t := h.Ref.Table
	if err := analyze(ctx, db, h); err != nil {
		return err
	}

	id := keel.Col{Ref: h.Ref, Index: t.PK[0]}
	for begin := 0; begin < len(stamps); begin += BackdateChunk {
		chunk := stamps[begin:min(begin+BackdateChunk, len(stamps))]
		ids := make([]string, len(chunk))
		created := make([]int64, len(chunk))
		updated := make([]int64, len(chunk))
		for i, s := range chunk {
			ids[i] = s.Id.String()
			created[i] = s.Created
			updated[i] = max(s.Updated, s.Created)
		}

		d := keel.Unnest("d", []string{"id", "created_at", "updated_at"}, ids, created, updated)
		sets := make([]keel.Assignment, 0, 2)
		if t.AutoCreate >= 0 {
			sets = append(
				sets,
				keel.Col{Ref: h.Ref, Index: t.AutoCreate}.ToExpr(d.Col("created_at")),
			)
		}
		if !createdOnly && t.AutoUpdate >= 0 {
			sets = append(
				sets,
				keel.Col{Ref: h.Ref, Index: t.AutoUpdate}.ToExpr(d.Col("updated_at")),
			)
		}
		if len(sets) == 0 {
			return nil
		}
		if t.Version >= 0 {
			version := keel.Col{Ref: h.Ref, Index: t.Version}
			sets = append(sets, version.ToExpr(version))
		}
		if _, err := h.Query().With(d).
			Join(d, d.Col("id").EqCol(id)).
			Where(keel.True()).
			Update(ctx, db, sets...); err != nil {
			return fmt.Errorf("backdate %s: %w", t.Name, err)
		}
	}

	return nil
}

func analyze[T any](ctx context.Context, db *keel.DB, h keel.Handle[*T]) error {
	if _, err := keel.Exec(ctx, db, "ANALYZE "+h.Ref.Table.Quoted); err != nil {
		return fmt.Errorf("analyze %s: %w", h.Ref.Table.Name, err)
	}

	return nil
}

func (g *Generator) InTx(ctx context.Context, fn func(context.Context, *keel.DB) error) error {
	return keel.Write(ctx, g.DB, func(ctx context.Context) error {
		return fn(ctx, g.DB)
	})
}

func (g *Generator) Progress(entity string, done, total int) {
	g.Opts.Logf("  • %s: %d/%d", entity, done, total)
}

func Unix(t time.Time) int64 {
	return t.Unix()
}

func Ptr[T any](v T) *T {
	return &v
}

func FirstOrNil[T any](ctx context.Context, db *keel.DB, q *keel.Query[*T]) (*T, error) {
	row, err := q.First(ctx, db)
	if errors.Is(err, keel.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return row, nil
}

func (g *Generator) AccountingControl(
	ctx context.Context,
) (*accountingcontrol.AccountingControl, error) {
	control, err := FirstOrNil(ctx, g.DB, accountingcontroldb.AccountingControls.Query())
	if err != nil {
		return nil, fmt.Errorf("load accounting control: %w", err)
	}

	return control, nil
}

func (g *Generator) GLAccountsByCode(
	ctx context.Context,
	codes ...string,
) (map[string]pulid.ID, error) {
	ga := glaccountdb.GLAccounts
	accounts, err := ga.Pick(ga.ID, ga.AccountCode).
		Where(ga.AccountCode.In(codes...)).
		All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load gl accounts: %w", err)
	}
	byCode := make(map[string]pulid.ID, len(accounts))
	for _, a := range accounts {
		byCode[a.AccountCode] = a.ID
	}

	return byCode, nil
}

func (g *Generator) UsersByID(ctx context.Context) (map[pulid.ID]*tenant.User, error) {
	u := tenantdb.Users
	users, err := u.Where(u.ID.In(g.Refs.UserIDs...)).All(ctx, g.DB)
	if err != nil {
		return nil, fmt.Errorf("load users: %w", err)
	}
	byID := make(map[pulid.ID]*tenant.User, len(users))
	for _, user := range users {
		byID[user.ID] = user
	}

	return byID, nil
}

func NullIfEmpty(e keel.Expr) keel.StringExpr {
	return keel.StringOf(e).NullIf("")
}

func PostBalances(
	ctx context.Context,
	db *keel.DB,
	rows []*glbalance.GLAccountPeriodBalance,
) error {
	if len(rows) == 0 {
		return nil
	}

	b := glbalancedb.GLAccountPeriodBalances
	if _, err := b.Upsert(ctx, db, rows, keel.OnConflict(
		b.OrganizationID,
		b.BusinessUnitID,
		b.GLAccountID,
		b.FiscalYearID,
		b.FiscalPeriodID,
	).Set(
		b.PeriodDebitMinor.ToExpr(b.PeriodDebitMinor.Add(keel.Int64Of(b.PeriodDebitMinor.Excluded()))),
		b.PeriodCreditMinor.ToExpr(
			b.PeriodCreditMinor.Add(keel.Int64Of(b.PeriodCreditMinor.Excluded())),
		),
		b.NetChangeMinor.ToExpr(b.NetChangeMinor.Add(keel.Int64Of(b.NetChangeMinor.Excluded()))),
		b.LastJournalEntryID.ToExpr(b.LastJournalEntryID.Excluded()),
	),
	); err != nil {
		return fmt.Errorf("upsert gl balances by period: %w", err)
	}

	index := make(map[pulid.ID]int, len(rows))
	ids := make([]string, 0, len(rows))
	debits := make([]int64, 0, len(rows))
	credits := make([]int64, 0, len(rows))
	nets := make([]int64, 0, len(rows))
	for _, row := range rows {
		i, ok := index[row.GLAccountID]
		if !ok {
			i = len(ids)
			index[row.GLAccountID] = i
			ids = append(ids, row.GLAccountID.String())
			debits = append(debits, 0)
			credits = append(credits, 0)
			nets = append(nets, 0)
		}
		debits[i] += row.PeriodDebitMinor
		credits[i] += row.PeriodCreditMinor
		nets[i] += row.NetChangeMinor
	}

	return moveAccountBalances(
		ctx,
		db,
		accountMoves{ids: ids, debits: debits, credits: credits, nets: nets},
	)
}

type accountMoves struct {
	ids     []string
	debits  []int64
	credits []int64
	nets    []int64
}

func moveAccountBalances(ctx context.Context, db *keel.DB, m accountMoves) error {
	if len(m.ids) == 0 {
		return nil
	}

	ga := glaccountdb.GLAccounts
	d := keel.Unnest(
		"d",
		[]string{"id", "debit", "credit", "net"},
		m.ids,
		m.debits,
		m.credits,
		m.nets,
	)
	if _, err := ga.With(d).Join(d, d.Col("id").EqCol(ga.ID)).Where(keel.True()).Update(
		ctx,
		db,
		ga.CurrentBalance.ToExpr(ga.CurrentBalance.Add(keel.Int64Of(d.Col("net")))),
		ga.DebitBalance.ToExpr(ga.DebitBalance.Add(keel.Int64Of(d.Col("debit")))),
		ga.CreditBalance.ToExpr(ga.CreditBalance.Add(keel.Int64Of(d.Col("credit")))),
	); err != nil {
		return fmt.Errorf("update gl account running balances: %w", err)
	}

	return nil
}

type balanceKey struct {
	account pulid.ID
	year    pulid.ID
	period  pulid.ID
}

func RollBackBalances(
	ctx context.Context,
	db *keel.DB,
	lines []*journalentry.JournalEntryLine,
	entries map[pulid.ID]*journalentry.JournalEntry,
) error {
	if len(lines) == 0 {
		return nil
	}

	periodIndex := make(map[balanceKey]int)
	var accounts, years, periods []string
	var pDebit, pCredit, pNet []int64
	accountIndex := make(map[pulid.ID]int)
	moves := accountMoves{}
	for _, line := range lines {
		entry := entries[line.JournalEntryID]
		if entry == nil {
			continue
		}
		key := balanceKey{
			account: line.GLAccountID,
			year:    entry.FiscalYearID,
			period:  entry.FiscalPeriodID,
		}
		i, ok := periodIndex[key]
		if !ok {
			i = len(accounts)
			periodIndex[key] = i
			accounts = append(accounts, key.account.String())
			years = append(years, key.year.String())
			periods = append(periods, key.period.String())
			pDebit, pCredit, pNet = append(pDebit, 0), append(pCredit, 0), append(pNet, 0)
		}
		pDebit[i] -= line.DebitAmount
		pCredit[i] -= line.CreditAmount
		pNet[i] -= line.NetAmount

		j, ok := accountIndex[line.GLAccountID]
		if !ok {
			j = len(moves.ids)
			accountIndex[line.GLAccountID] = j
			moves.ids = append(moves.ids, line.GLAccountID.String())
			moves.debits = append(moves.debits, 0)
			moves.credits = append(moves.credits, 0)
			moves.nets = append(moves.nets, 0)
		}
		moves.debits[j] -= line.DebitAmount
		moves.credits[j] -= line.CreditAmount
		moves.nets[j] -= line.NetAmount
	}

	b := glbalancedb.GLAccountPeriodBalances
	d := keel.Unnest("d", []string{"account", "year", "period", "debit", "credit", "net"},
		accounts, years, periods, pDebit, pCredit, pNet)
	if _, err := b.With(d).Join(d,
		d.Col("account").EqCol(b.GLAccountID),
		d.Col("year").EqCol(b.FiscalYearID),
		d.Col("period").EqCol(b.FiscalPeriodID),
	).Where(keel.True()).Update(
		ctx,
		db,
		b.PeriodDebitMinor.ToExpr(b.PeriodDebitMinor.Add(keel.Int64Of(d.Col("debit")))),
		b.PeriodCreditMinor.ToExpr(b.PeriodCreditMinor.Add(keel.Int64Of(d.Col("credit")))),
		b.NetChangeMinor.ToExpr(b.NetChangeMinor.Add(keel.Int64Of(d.Col("net")))),
	); err != nil {
		return fmt.Errorf("roll back period balances: %w", err)
	}

	return moveAccountBalances(ctx, db, moves)
}

func KeepUpdated[T any](h keel.Handle[*T]) keel.Assignment {
	index := h.Ref.Table.AutoUpdate
	if index < 0 {
		index = h.Ref.Table.PK[0]
	}
	col := keel.Col{Ref: h.Ref, Index: index}
	return col.ToExpr(col)
}

func (g *Generator) PickUser(users []pulid.ID) pulid.ID {
	if len(users) == 0 {
		return g.Refs.AdminID
	}
	return Pick(g.Fake, users)
}
