package formula_test

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// countingMatrixRepo records how often the expensive read happened.
type countingMatrixRepo struct {
	repositories.RateMatrixRepository

	mu    sync.Mutex
	calls int
}

func (c *countingMatrixRepo) GetLookupData(
	_ context.Context,
	_ *repositories.GetRateMatrixLookupDataRequest,
) ([]*repositories.RateMatrixLookupData, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls++

	return nil, nil
}

func (c *countingMatrixRepo) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls
}

func setupCountingService(t *testing.T, repo repositories.RateMatrixRepository) *formula.Service {
	t.Helper()

	registry := newTestRegistry(t)
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)

	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})

	eng, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	require.NoError(t, err)

	return formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Engine:         eng,
		Resolver:       res,
		VersionRepo:    &stubVersionRepo{},
		RateMatrixRepo: repo,
	})
}

func memoTemplate(orgID, buID pulid.ID) *formulatemplate.FormulaTemplate {
	return &formulatemplate.FormulaTemplate{
		ID:             pulid.MustNew("ft_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "Per Mile",
		Expression:     "totalDistance * 2",
		Type:           formulatemplate.TemplateTypeFreightCharge,
		SchemaID:       "shipment",
	}
}

// A fuel price refresh re-rates every affected shipment in one activity, and an
// invoice adjustment re-rates every leg of a multi-leg order. Each of those used
// to re-read every lookup matrix with every one of its cells, for the same
// tenant, from scratch — the batch paid the cost once per shipment.
//
// The memo makes the batch pay it once. This is the test that says so.
func TestRate_BuildsTheTenantsLookupOncePerBatch(t *testing.T) {
	t.Parallel()

	repo := &countingMatrixRepo{}
	svc := setupCountingService(t, repo)

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	template := memoTemplate(orgID, buID)

	ctx := ratetablecache.With(t.Context())

	for range 5 {
		_, err := svc.Rate(ctx, &formula.RateRequest{
			Template: template,
			Entity:   memoShipment(),
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, repo.count(),
		"a batch of five shipments should read the tenant's rate tables once, not once each")
}

// Two tenants in one batch are two different sets of tables, so the memo must
// key on the organization. Sharing one entry would rate a shipment against
// another organization's tariff, which is far worse than a slow batch.
func TestRate_MemoDoesNotShareLookupsAcrossTenants(t *testing.T) {
	t.Parallel()

	repo := &countingMatrixRepo{}
	svc := setupCountingService(t, repo)

	buID := pulid.MustNew("bu_")
	first := memoTemplate(pulid.MustNew("org_"), buID)
	second := memoTemplate(pulid.MustNew("org_"), buID)

	ctx := ratetablecache.With(t.Context())

	for _, template := range []*formulatemplate.FormulaTemplate{first, second, first, second} {
		_, err := svc.Rate(ctx, &formula.RateRequest{
			Template: template,
			Entity:   memoShipment(),
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 2, repo.count(), "each organization reads its own tables exactly once")
}

// The memo keys on the business unit as well as the organization. A valid
// tenant pairing never varies the business unit under one organization, but a
// malformed caller could — and it must get its own entry rather than silently
// inheriting whatever the first pairing loaded.
func TestRate_MemoDoesNotShareLookupsAcrossBusinessUnits(t *testing.T) {
	t.Parallel()

	repo := &countingMatrixRepo{}
	svc := setupCountingService(t, repo)

	orgID := pulid.MustNew("org_")
	first := memoTemplate(orgID, pulid.MustNew("bu_"))
	second := memoTemplate(orgID, pulid.MustNew("bu_"))

	ctx := ratetablecache.With(t.Context())

	for _, template := range []*formulatemplate.FormulaTemplate{first, second, first, second} {
		_, err := svc.Rate(ctx, &formula.RateRequest{
			Template: template,
			Entity:   memoShipment(),
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 2, repo.count(),
		"the same organization under two business units must build twice")
}

// A formula evaluated outside any unit of work — a CLI run, a test, an
// evaluation reached by a path that predates the memo — has to keep working.
// Without a memo on the context the service simply builds, exactly as it did
// before the memo existed.
func TestRate_WithoutAMemoEachEvaluationStillBuildsItsOwnLookup(t *testing.T) {
	t.Parallel()

	repo := &countingMatrixRepo{}
	svc := setupCountingService(t, repo)

	template := memoTemplate(pulid.MustNew("org_"), pulid.MustNew("bu_"))

	for range 3 {
		_, err := svc.Rate(t.Context(), &formula.RateRequest{
			Template: template,
			Entity:   memoShipment(),
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 3, repo.count())
}

// The memo spans the batch, so the inner install a service performs must leave
// the outer one in place. If it replaced it, every shipment in the batch would
// get a fresh memo and the batch would be back to reading per shipment.
func TestRateTableCache_NestedInstallKeepsTheOuterMemo(t *testing.T) {
	t.Parallel()

	repo := &countingMatrixRepo{}
	svc := setupCountingService(t, repo)

	template := memoTemplate(pulid.MustNew("org_"), pulid.MustNew("bu_"))

	batchCtx := ratetablecache.With(t.Context())

	for range 4 {
		perShipmentCtx := ratetablecache.With(batchCtx)

		_, err := svc.Rate(perShipmentCtx, &formula.RateRequest{
			Template: template,
			Entity:   memoShipment(),
		})
		require.NoError(t, err)
	}

	assert.Equal(t, 1, repo.count(),
		"a service installing its own memo must not undo the batch's")
}

func memoShipment() *shipment.Shipment {
	return &shipment.Shipment{ID: pulid.MustNew("shp_")}
}
