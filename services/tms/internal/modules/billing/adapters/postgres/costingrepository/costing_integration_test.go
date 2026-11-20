//go:build integration

package costingrepository

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/billing/billingtest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol/costingcontroldb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type statementLog struct {
	mu         sync.Mutex
	recording  bool
	statements []string
	inTx       []bool
}

func (l *statementLog) BeforeQuery(ctx context.Context, _ *keel.QueryEvent) context.Context {
	return ctx
}

func (l *statementLog) AfterQuery(_ context.Context, event *keel.QueryEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.recording {
		return
	}
	l.statements = append(l.statements, event.Op+" "+event.Table)
	l.inTx = append(l.inTx, event.Tx)
}

func (l *statementLog) record(fn func()) ([]string, []bool) {
	l.mu.Lock()
	l.recording = true
	l.statements = nil
	l.inTx = nil
	l.mu.Unlock()

	fn()

	l.mu.Lock()
	defer l.mu.Unlock()
	l.recording = false
	return l.statements, l.inTx
}

type categoryLink struct {
	CostCategoryID pulid.ID
	GLAccountID    pulid.ID
}

type glAccountsEnv struct {
	ctx        context.Context
	db         *keel.DB
	conn       *postgres.Connection
	repo       costingcontrolports.CostingControlRepository
	log        *statementLog
	tenantInfo pagination.TenantInfo
	categories []pulid.ID
	accounts   []pulid.ID
}

func setupGLAccountsEnv(t *testing.T) *glAccountsEnv {
	t.Helper()

	ctx, db, dsn, cleanup := seededtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	seeded := billingtest.SeededTenant(t, ctx, db)
	ctx = seeded.Ctx

	log := &statementLog{}
	cfg := dbpool.KeelConfig(nil, []keel.Observer{log})
	cfg.DSN = dsn
	observed, err := keel.Open(ctx, cfg)
	require.NoError(t, err)
	t.Cleanup(observed.Close)

	conn := postgres.WrapDB(observed)
	repo := New(Params{DB: conn, Logger: zap.NewNop()})

	control, err := repo.GetByOrgID(
		ctx,
		&costingcontrolports.GetCostingControlRequest{TenantInfo: seeded.Info},
	)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(control.Categories), 3)

	gl := glaccountdb.GLAccounts
	accounts, err := keel.Into[pulid.ID](ctx, observed, gl.
		Pick(gl.ID).
		Order(gl.AccountCode.Asc(), gl.ID.Asc()).
		Limit(3))
	require.NoError(t, err)
	require.Len(t, accounts, 3)

	return &glAccountsEnv{
		ctx:        ctx,
		db:         observed,
		conn:       conn,
		repo:       repo,
		log:        log,
		tenantInfo: seeded.Info,
		categories: []pulid.ID{
			control.Categories[0].ID,
			control.Categories[1].ID,
			control.Categories[2].ID,
		},
		accounts: accounts,
	}
}

func (e *glAccountsEnv) replace(
	ctx context.Context,
	categories ...costingcontrolports.CategoryGLAccounts,
) error {
	return e.repo.ReplaceCategoryGLAccounts(
		ctx,
		&costingcontrolports.ReplaceCategoryGLAccountsRequest{
			TenantInfo: e.tenantInfo,
			Categories: categories,
		},
	)
}

func (e *glAccountsEnv) links(t *testing.T) map[pulid.ID][]pulid.ID {
	t.Helper()

	ccga := costingcontroldb.CostCategoryGLAccounts
	rows, err := keel.Into[categoryLink](e.ctx, e.db, ccga.
		Pick(ccga.CostCategoryID, ccga.GLAccountID).
		Where(ccga.CostCategoryID.In(e.categories...)))
	require.NoError(t, err)

	byCategory := make(map[pulid.ID][]pulid.ID, len(e.categories))
	for _, row := range rows {
		byCategory[row.CostCategoryID] = append(byCategory[row.CostCategoryID], row.GLAccountID)
	}
	for id, accounts := range byCategory {
		slices.SortFunc(accounts, func(a, b pulid.ID) int {
			return strings.Compare(a.String(), b.String())
		})
		byCategory[id] = accounts
	}

	return byCategory
}

func sortedIDs(ids ...pulid.ID) []pulid.ID {
	out := slices.Clone(ids)
	slices.SortFunc(out, func(a, b pulid.ID) int {
		return strings.Compare(a.String(), b.String())
	})
	return out
}

func link(category pulid.ID, accounts ...pulid.ID) costingcontrolports.CategoryGLAccounts {
	return costingcontrolports.CategoryGLAccounts{CostCategoryID: category, GLAccountIDs: accounts}
}

func TestReplaceCategoryGLAccountsInOneDeleteAndOneInsert(t *testing.T) {
	env := setupGLAccountsEnv(t)
	first, second, untouched := env.categories[0], env.categories[1], env.categories[2]
	a, b, c := env.accounts[0], env.accounts[1], env.accounts[2]

	require.NoError(t, env.replace(env.ctx,
		link(first, a, b),
		link(second, c),
		link(untouched, a),
	))
	assert.Equal(t, map[pulid.ID][]pulid.ID{
		first:     sortedIDs(a, b),
		second:    {c},
		untouched: {a},
	}, env.links(t))

	var err error
	statements, _ := env.log.record(func() {
		err = env.replace(env.ctx,
			link(first, c),
			link(second, a, b),
		)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{
		"delete cost_category_gl_accounts",
		"insert cost_category_gl_accounts",
	}, statements)
	assert.Equal(t, map[pulid.ID][]pulid.ID{
		first:     {c},
		second:    sortedIDs(a, b),
		untouched: {a},
	}, env.links(t), "named categories hold exactly the new accounts; others keep theirs")

	statements, _ = env.log.record(func() {
		err = env.replace(env.ctx, link(first), link(second))
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"delete cost_category_gl_accounts"}, statements,
		"clearing every named category deletes and inserts nothing")
	assert.Equal(t, map[pulid.ID][]pulid.ID{untouched: {a}}, env.links(t))

	statements, _ = env.log.record(func() {
		err = env.replace(env.ctx)
	})
	require.NoError(t, err)
	assert.Empty(t, statements, "no categories sends no statement")
}

func TestReplaceCategoryGLAccountsJoinsTheSaveTransaction(t *testing.T) {
	env := setupGLAccountsEnv(t)
	first, second := env.categories[0], env.categories[1]
	a, b, c := env.accounts[0], env.accounts[1], env.accounts[2]

	require.NoError(t, env.replace(env.ctx, link(first, a), link(second, b)))
	before := env.links(t)

	errSaveFailed := errors.New("a later step of the save failed")
	var err error
	statements, inTx := env.log.record(func() {
		err = env.conn.Transact(
			env.ctx,
			kernelports.TxOptions{},
			func(txCtx context.Context) error {
				replaceErr := env.replace(txCtx, link(first, c), link(second, a, c))
				if replaceErr != nil {
					return replaceErr
				}
				return errSaveFailed
			},
		)
	})
	require.ErrorIs(t, err, errSaveFailed)
	assert.Equal(t, []string{
		"delete cost_category_gl_accounts",
		"insert cost_category_gl_accounts",
	}, statements)
	assert.Equal(t, []bool{true, true}, inTx)
	assert.Equal(
		t,
		before,
		env.links(t),
		"a failed save leaves every category's accounts as they were",
	)

	require.NoError(t, env.conn.Transact(
		env.ctx,
		kernelports.TxOptions{},
		func(txCtx context.Context) error {
			return env.replace(txCtx, link(first, c), link(second, a, c))
		},
	))
	assert.Equal(t, map[pulid.ID][]pulid.ID{
		first:  {c},
		second: sortedIDs(a, c),
	}, env.links(t))
}
