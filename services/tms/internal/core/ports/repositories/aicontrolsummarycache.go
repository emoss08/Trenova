package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/pkg/pagination"
)

// AIControlSummaryKey names one sentence: a tab of one organization, for one
// set of facts. A sentence written for a key is true for every read of it.
type AIControlSummaryKey struct {
	TenantInfo pagination.TenantInfo
	Tab        aicontrolsummary.Tab
	FactsHash  string
}

// AIControlSummaryCache keeps the sentences a model wrote, so reading a tab
// never asks a model again until the facts move, and bounds how often a day
// it may be asked.
type AIControlSummaryCache interface {
	Get(ctx context.Context, key *AIControlSummaryKey, dest *aicontrolsummary.Summary) (bool, error)
	Set(ctx context.Context, key *AIControlSummaryKey, value *aicontrolsummary.Summary, ttl time.Duration) error
	// ClaimNarration counts one model rewording of a tab today, and reports
	// whether it is within the day's allowance.
	ClaimNarration(ctx context.Context, key *AIControlSummaryKey, perDay int) (bool, error)
}
