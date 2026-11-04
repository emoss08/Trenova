package paletteports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// QuestionRepository keeps the record of each palette question: what the
// allowance counts, how many a person has being answered, and the Desk
// conversation one was carried into.
type QuestionRepository interface {
	Create(ctx context.Context, question *palettequestion.Question) error
	Get(ctx context.Context, ref AskRef) (*palettequestion.Question, error)
	// CountAsking is how many of the person's questions are still being
	// answered and were asked since then.
	CountAsking(ctx context.Context, tenant pagination.TenantInfo, since int64) (int, error)
	// CountUnkeptSince is how many questions the person asked since then that
	// were not carried into the Desk, where they count as a message instead.
	CountUnkeptSince(ctx context.Context, tenant pagination.TenantInfo, since int64) (int, error)
	Finish(ctx context.Context, ref AskRef, status palettequestion.Status, at int64) error
	// ClaimKeep starts carrying the question into the Desk. It is false when
	// it was carried already or another request is carrying it now.
	ClaimKeep(ctx context.Context, ref AskRef, now int64) (bool, error)
	MarkKept(ctx context.Context, ref AskRef, threadID pulid.ID) error
	ReleaseKeep(ctx context.Context, ref AskRef) error
	// DeleteBefore removes questions asked before then, across every
	// organization, at most limit at a time. Their reads go with them.
	DeleteBefore(ctx context.Context, before int64, limit int) (int, error)
	// RecordToolCall keeps one read Nova made answering a question.
	RecordToolCall(ctx context.Context, call *palettequestion.ToolCall) error
}
