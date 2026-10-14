package formula_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type approvedVersionRepo struct {
	stubVersionRepo
	approved []*formulatemplate.FormulaTemplateVersion
}

func (r *approvedVersionRepo) ListApproved(
	_ context.Context,
	_ *formulatemplateports.ListScheduledVersionsRequest,
) ([]*formulatemplate.FormulaTemplateVersion, error) {
	return r.approved, nil
}

func setupLifecycleService(
	t *testing.T,
	template *formulatemplate.FormulaTemplate,
	approved ...*formulatemplate.FormulaTemplateVersion,
) *formula.Service {
	t.Helper()

	registry := newTestRegistry(t)
	res := resolver.NewResolver()
	eng, err := engine.NewEngine(engine.Params{
		Registry: registry,
		Resolver: res,
		EnvBuilder: engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
			Registry: registry,
			Resolver: res,
		}),
	})
	require.NoError(t, err)

	return formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Engine:         eng,
		Resolver:       res,
		Repo:           &singleTemplateRepo{template: template},
		VersionRepo:    &approvedVersionRepo{approved: approved},
		RateMatrixRepo: &stubMatrixRepo{},
	})
}

func unixDate(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC).Unix()
}

func approvedSnapshot(
	template *formulatemplate.FormulaTemplate,
	number int64,
	expression string,
	approvedAt int64,
	effectiveFrom *int64,
) *formulatemplate.FormulaTemplateVersion {
	return &formulatemplate.FormulaTemplateVersion{
		ID:            pulid.MustNew("ftv_"),
		TemplateID:    template.ID,
		VersionNumber: number,
		Expression:    expression,
		SchemaID:      template.SchemaID,
		Type:          template.Type,
		Status:        formulatemplate.StatusActive,
		CreatedAt:     approvedAt,
		EffectiveFrom: effectiveFrom,
	}
}

func calculateOn(
	t *testing.T,
	svc *formula.Service,
	template *formulatemplate.FormulaTemplate,
	ratingDate int64,
) (*formulatemplatetypes.CalculateResponse, error) {
	t.Helper()

	return svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
		TemplateID: template.ID,
		Entity:     shippedOnMarchFirst(),
		RatingDate: ratingDate,
	})
}

func TestCalculate_APendingRevisionKeepsTheApprovedContentBilling(t *testing.T) {
	t.Parallel()

	for _, status := range []formulatemplate.Status{
		formulatemplate.StatusDraft,
		formulatemplate.StatusInReview,
	} {
		t.Run(status.String(), func(t *testing.T) {
			t.Parallel()

			template := activeTemplate(`200`)
			template.Status = status
			template.CurrentVersionNumber = 2
			svc := setupLifecycleService(t, template,
				approvedSnapshot(template, 1, `100`, unixDate(2025, time.January, 1), nil),
			)

			resp, err := calculateOn(t, svc, template, unixDate(2025, time.June, 1))
			require.NoError(t, err)
			assert.Equal(t, "100", resp.Amount.String(), "the pending edit does not bill")
			assert.EqualValues(t, 1, resp.VersionNumber)
			assert.EqualValues(t, 1, resp.Receipt.VersionNumber)
		})
	}
}

func TestCalculate_RefusesWhatNeverPassedReviewOrWasArchived(t *testing.T) {
	t.Parallel()

	draft := activeTemplate(`200`)
	draft.Status = formulatemplate.StatusDraft
	_, err := calculateOn(t, setupLifecycleService(t, draft), draft, unixDate(2025, time.June, 1))
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "never been approved")

	archived := activeTemplate(`200`)
	archived.Status = formulatemplate.StatusInactive
	svc := setupLifecycleService(t, archived,
		approvedSnapshot(archived, 1, `100`, unixDate(2025, time.January, 1), nil),
	)
	_, err = calculateOn(t, svc, archived, unixDate(2025, time.June, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archived")
}

func TestCalculate_RatesWithTheSnapshotInForceOnTheRatingDate(t *testing.T) {
	t.Parallel()

	template := activeTemplate(`300`)
	template.CurrentVersionNumber = 3
	scheduledFrom := unixDate(2025, time.September, 1)
	svc := setupLifecycleService(t, template,
		approvedSnapshot(template, 1, `100`, unixDate(2025, time.January, 1), nil),
		approvedSnapshot(template, 2, `200`, unixDate(2025, time.May, 1), nil),
		approvedSnapshot(template, 3, `300`, unixDate(2025, time.July, 1), &scheduledFrom),
	)

	cases := map[string]struct {
		ratingDate int64
		want       string
		version    int64
	}{
		"before the first approval": {unixDate(2024, time.December, 1), "100", 1},
		"in the first window":       {unixDate(2025, time.March, 1), "100", 1},
		"in the second window":      {unixDate(2025, time.August, 1), "200", 2},
		"after the scheduled start": {unixDate(2025, time.October, 1), "300", 3},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp, err := calculateOn(t, svc, template, tc.ratingDate)
			require.NoError(t, err)
			assert.Equal(t, tc.want, resp.Amount.String())
			assert.Equal(t, tc.version, resp.VersionNumber)
		})
	}
}

func TestCalculate_RefusesToRateBeforeADeferredFirstApprovalStarts(t *testing.T) {
	t.Parallel()

	template := activeTemplate(`100`)
	template.CurrentVersionNumber = 1
	deferredTo := unixDate(2099, time.January, 1)
	svc := setupLifecycleService(t, template,
		approvedSnapshot(template, 1, `100`, unixDate(2025, time.January, 1), &deferredTo),
	)

	_, err := calculateOn(t, svc, template, unixDate(2026, time.March, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "approved to bill from 2099-01-01")
}

func TestCalculate_AnActiveTemplateWithNoApprovedSnapshotRatesItsRow(t *testing.T) {
	t.Parallel()

	template := activeTemplate(`125`)
	resp, err := calculateOn(
		t,
		setupLifecycleService(t, template),
		template,
		unixDate(2025, time.June, 1),
	)
	require.NoError(t, err)
	assert.Equal(t, "125", resp.Amount.String())
}
