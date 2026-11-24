package accountingboardreader

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingboard"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalclose"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeriodCloseDrawsEveryCheckBlockedThenClear(t *testing.T) {
	t.Parallel()

	sources := 408
	oldest := int64(1_000)
	read := closeRead(t.Context(), 1_000+4*86_400, &periodFacts{
		name:    "Period 11 - November 2026",
		pending: pendingEntries{Count: 408, Debit: 1_234_567, OldestCreated: &oldest},
	}, &fiscalclose.Result{
		Blockers: []*fiscalclose.Blocker{
			{
				Check:   fiscalclose.CheckEarlierPeriods,
				Message: "Cannot close period 11: period 1 is still open. Close periods sequentially.",
			},
			{
				Check:   fiscalclose.CheckUnpostedSources,
				Message: "Cannot close fiscal period while 408 accounting sources remain unposted",
				Count:   &sources,
			},
		},
		Cleared: []fiscalclose.Check{
			fiscalclose.CheckPeriodStatus,
			fiscalclose.CheckJournalReversals,
		},
	})

	assert.Equal(t, "What holds Period 11 - November 2026", read.Title)
	require.Len(t, read.Checks, 4)

	assert.Equal(t, "An earlier period is not closed", read.Checks[0].Title)
	assert.Equal(t, boardcatalog.ToneBad, read.Checks[0].Tone)
	assert.Contains(t, read.Checks[0].Detail, "period 1 is still open")
	assert.Nil(t, read.Checks[0].Count, "a blocker that states no count carries none")

	assert.Equal(t, "Accounting sources unposted", read.Checks[1].Title)
	require.NotNil(t, read.Checks[1].Count)
	assert.Equal(t, "408", read.Checks[1].Count.Number.String())
	assert.Empty(t, read.Checks[1].Refs, "blockers carry no records")

	assert.Equal(t, boardcatalog.ToneOK, read.Checks[2].Tone)
	assert.Nil(t, read.Checks[2].Count)
	assert.Equal(t, "No journal reversals waiting", read.Checks[3].Title)
	require.NotNil(t, read.Checks[3].Count)
	assert.True(t, read.Checks[3].Count.Number.IsZero())

	assert.Equal(t, "2", read.Figures[figureBlocking].Number.String())
	assert.Equal(t, "408", read.Figures[figurePendingEntries].Number.String())
	assert.Equal(t, "12345.67", read.Figures[figurePendingAmount].Number.String())
	assert.Equal(t, boardcatalog.ValueDuration, read.Figures[figureOldestPending].Kind)
	assert.Equal(t, "345600", read.Figures[figureOldestPending].Number.String())
}

func TestPeriodCloseDeclaresEveryFigureTheCatalogLists(t *testing.T) {
	t.Parallel()

	metric, ok := boardcatalog.New(boardcatalog.Entries{Metrics: accountingboard.Metrics}).
		MetricNamed(accountingboard.MetricPeriodClose)
	require.True(t, ok)
	read := closeRead(t.Context(), 0, &periodFacts{name: "October"}, &fiscalclose.Result{})
	for _, figure := range metric.Figures {
		value, held := read.Figures[figure.Name]
		require.True(t, held, figure.Name)
		require.NotNil(t, value, figure.Name)
		assert.Equal(t, figure.Value, value.Kind, figure.Name)
	}
	assert.True(t, read.Figures[figureOldestPending].Number.IsZero(),
		"nothing waiting is no time waited")
}

func TestPeriodCloseWithoutAPeriodReadsNothing(t *testing.T) {
	t.Parallel()

	read, err := (&PeriodClose{}).Read(t.Context(), &boardreaderports.MetricRequest{})
	require.NoError(t, err)
	assert.Nil(t, read)
}

type scopedPeriods struct {
	asked []fiscalperiodports.GetFiscalPeriodByIDRequest
}

func (s *scopedPeriods) GetCloseBlockers(
	_ context.Context,
	req fiscalperiodports.GetFiscalPeriodByIDRequest,
) (*fiscalclose.Result, error) {
	s.asked = append(s.asked, req)

	return nil, errortypes.NewNotFoundError("Fiscal period not found")
}

func TestPeriodCloseOutsideTheReadersScopeReadsAsNothing(t *testing.T) {
	t.Parallel()

	periods := &scopedPeriods{}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	other := pulid.MustNew("fp_")

	read, err := (&PeriodClose{periods: periods}).Read(t.Context(),
		&boardreaderports.MetricRequest{Tenant: tenant, For: other})
	require.NoError(t, err)
	assert.Nil(t, read, "a period the reader's tenant does not hold is not drawn")
	require.Len(t, periods.asked, 1)
	assert.Equal(t, tenant, periods.asked[0].TenantInfo, "the period is read in the reader's tenant")
	assert.Equal(t, other, periods.asked[0].ID)
}
