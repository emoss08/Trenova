package briefingjobs

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type pagedDeletes struct {
	pages   []int
	cutoffs []string
}

func (p *pagedDeletes) next(cutoff string) int {
	p.cutoffs = append(p.cutoffs, cutoff)
	if len(p.pages) == 0 {
		return 0
	}
	deleted := p.pages[0]
	p.pages = p.pages[1:]
	return deleted
}

type morningBriefs struct {
	repositories.BriefingRepository
	deletes *pagedDeletes
}

func (m morningBriefs) DeleteBefore(
	_ context.Context,
	req repositories.DeleteBriefingsBeforeRequest,
) (int, error) {
	return m.deletes.next(req.BeforeDate), nil
}

type boardBriefs struct {
	repositories.ShipmentBriefRepository
	deletes *pagedDeletes
}

func (b boardBriefs) DeleteBefore(
	_ context.Context,
	req *repositories.DeleteShipmentBriefsBeforeRequest,
) (int, error) {
	return b.deletes.next(req.BeforeDate), nil
}

func TestBriefingRetentionAlsoClearsOldBoardBriefs(t *testing.T) {
	t.Parallel()

	morning := &pagedDeletes{pages: []int{retentionBatch, 4}}
	board := &pagedDeletes{pages: []int{retentionBatch, retentionBatch, 7}}
	a := &Activities{
		l:         zap.NewNop(),
		repo:      morningBriefs{deletes: morning},
		boardRepo: boardBriefs{deletes: board},
	}

	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	value, err := env.ExecuteActivity(a.BriefingRetentionActivity)
	require.NoError(t, err)
	var result BriefingRetentionResult
	require.NoError(t, value.Get(&result))

	assert.Equal(t, retentionBatch+4, result.Deleted)
	assert.Equal(t, 2*retentionBatch+7, result.DeletedBoardBriefs)
	require.Len(t, board.cutoffs, 3)
	assert.Equal(t, morning.cutoffs[0], board.cutoffs[0], "both kinds of brief keep the same window")
}
