package analysisports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ListRunsRequest pages the history newest first. Before and BeforeID are the
// last run of the page before, so runs started in the same second are neither
// skipped nor repeated.
type ListRunsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	BeforeID   pulid.ID
	Limit      int
}

type RunRepository interface {
	Create(ctx context.Context, run *analysis.Run) (*analysis.Run, error)
	Get(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) (*analysis.Run, error)
	// List is the history: each run's status, title, purpose and timing, without
	// its code, output or what it showed.
	List(ctx context.Context, req *ListRunsRequest) ([]*analysis.Run, error)
}

// RunRecorder keeps an analysis a script showed the person, so the Desk's card
// reads the tables it holds past the rows the agent was shown.
type RunRecorder interface {
	Record(ctx context.Context, run *analysis.Run) (*analysis.Run, error)
}

type RunListResponse struct {
	Items        []*analysis.Run `json:"items"`
	NextBefore   int64           `json:"nextBefore"`
	NextBeforeID pulid.ID        `json:"nextBeforeId"`
}

type RunRequest struct {
	TenantInfo pagination.TenantInfo
	RunID      pulid.ID

	// Oversees lets someone who oversees agents read every analysis; anyone
	// else reads only the ones run for them.
	Oversees bool
}

// Runs is the history of the organization's analyses and one analysis whole.
type Runs interface {
	ListRuns(ctx context.Context, req *ListRunsRequest) (*RunListResponse, error)
	Run(ctx context.Context, req *RunRequest) (*analysis.Run, error)
}
