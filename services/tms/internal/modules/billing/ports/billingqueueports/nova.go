package billingqueueports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// NovaPhase is the step a queue's Nova generation is on.
type NovaPhase string

const (
	NovaPhaseReviewing = NovaPhase("Reviewing")
	NovaPhaseBriefing  = NovaPhase("Briefing")
	NovaPhaseDone      = NovaPhase("Done")
	// NovaPhaseLeft is no generation running, with items the last one could
	// not review waiting until RetryAt for another try.
	NovaPhaseLeft = NovaPhase("Left")
)

// NovaGeneration is a queue's reviews and brief being written in bulk.
type NovaGeneration struct {
	Phase NovaPhase `json:"phase"`
	Total int       `json:"total"`
	Done  int       `json:"done"`
	// RetryAt is when opening the queue next tries what was left (NovaPhaseLeft).
	RetryAt int64 `json:"retryAt,omitempty"`
}

// NovaTrigger is what asked for a generation. Someone who opened the queue is
// waiting, so their generation reviews at once; the schedule's waits for the
// provider's batch API, which costs less and can take hours.
type NovaTrigger string

const (
	NovaTriggerQueueOpened = NovaTrigger("QueueOpened")
	NovaTriggerSchedule    = NovaTrigger("Schedule")
)

// NovaGenerationState is where an organization's generations stand.
type NovaGenerationState struct {
	// Running is the generation under way, nil when there is none.
	Running *NovaGeneration
	// EndedAt is when the last generation ended, zero while one runs or
	// when none has.
	EndedAt int64
	// LeftUnreviewed is whether the last generation ended with items it
	// could not review.
	LeftUnreviewed bool
}

// NovaGenerator writes, in bulk, the reviews a queue is missing and its
// brief: one generation per organization at a time, asked for when someone
// opens the queue and by a schedule.
type NovaGenerator interface {
	RequestGeneration(ctx context.Context, tenantInfo pagination.TenantInfo, trigger NovaTrigger)
	Generation(ctx context.Context, tenantInfo pagination.TenantInfo) (NovaGenerationState, error)
}

// ReviewCursor pages through the items Nova has not reviewed, oldest first.
type ReviewCursor struct {
	CreatedAt int64    `json:"createdAt"`
	ID        pulid.ID `json:"id"`
}

type ListUnreviewedItemsRequest struct {
	TenantInfo pagination.TenantInfo
	After      *ReviewCursor
	Limit      int
}

type UnreviewedItem struct {
	ID        pulid.ID `json:"id"`
	CreatedAt int64    `json:"createdAt"`
}

// ReviewTicket is one item in a submitted batch and the evidence it was sent
// with, so an answer to evidence that has since moved is not recorded.
type ReviewTicket struct {
	ItemID pulid.ID `json:"itemId"`
	Hash   string   `json:"hash"`
}

// ReviewSubmission is one page of unreviewed items sent for review: as a
// batch when the provider has a batch API, otherwise to be reviewed one at a
// time.
type ReviewSubmission struct {
	Handle  *llmports.BatchHandle `json:"handle,omitempty"`
	Tickets []ReviewTicket        `json:"tickets"`
	Inline  []pulid.ID            `json:"inline"`
	Next    *ReviewCursor         `json:"next,omitempty"`
}

// Count is how many items the submission reviews.
func (s *ReviewSubmission) Count() int {
	return len(s.Tickets) + len(s.Inline)
}

type SubmitReviewsRequest struct {
	TenantInfo pagination.TenantInfo
	After      *ReviewCursor
	// Inline leaves the page to be reviewed one item at a time, for someone
	// waiting on it, rather than sending it as a batch.
	Inline bool
}

type ApplyReviewsRequest struct {
	TenantInfo pagination.TenantInfo
	Submission *ReviewSubmission
}
