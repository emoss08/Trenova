package assistantturnrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AssistantTurnRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assistantturn-repository"),
	}
}

// Start records a turn about to run.
//
// The partial unique index is what refuses a second live turn on one thread,
// so the collision is the answer rather than an error to log: two turns
// appending to the same conversation interleave their message sequence
// numbers, and the transcript stops describing anything that happened.
func (r *repository) Start(
	ctx context.Context,
	turn *conversation.AssistantTurn,
) (*conversation.AssistantTurn, error) {
	_, err := r.db.DBForContext(ctx).NewInsert().Model(turn).Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, repositories.ErrTurnAlreadyRunning
		}

		r.l.Error("failed to start an assistant turn",
			zap.String("thread", turn.ThreadID.String()),
			zap.Error(err),
		)

		return nil, fmt.Errorf("start assistant turn: %w", err)
	}

	return turn, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	cols := buncolgen.AssistantTurnColumns
	turn := new(conversation.AssistantTurn)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(turn).
		Apply(buncolgen.AssistantTurnApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.ID)
	// The reader supplied the id, so the owner is checked here rather than
	// trusted. A turn belongs to one person and this hands out their reply.
	if !req.UserID.IsNil() {
		q = q.Where(cols.UserID.Eq(), req.UserID)
	}

	if err := q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "AssistantTurn")
	}

	return turn, nil
}

// liveStatuses are a turn still producing its reply. A write that marks a
// turn live must be limited to them: a fast turn can finish before the
// request that started it records its workflow, and marking it Running after
// that would hold the conversation's one live slot for ever.
var liveStatuses = []conversation.AssistantTurnStatus{
	conversation.AssistantTurnStatusPending,
	conversation.AssistantTurnStatusRunning,
}

func (r *repository) Active(
	ctx context.Context,
	req repositories.ActiveAssistantTurnRequest,
) (*conversation.AssistantTurn, error) {
	cols := buncolgen.AssistantTurnColumns
	turn := new(conversation.AssistantTurn)

	err := r.db.DBForContext(ctx).NewSelect().
		Model(turn).
		Apply(buncolgen.AssistantTurnApplyTenant(req.TenantInfo)).
		Where(cols.ThreadID.Eq(), req.ThreadID).
		Where(cols.UserID.Eq(), req.UserID).
		Where(cols.Status.In(), bun.List(liveStatuses)).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			// Not producing a reply is the ordinary state of a conversation,
			// not an absence worth reporting as one.
			return nil, nil
		}

		return nil, fmt.Errorf("read the turn this conversation is producing: %w", err)
	}

	return turn, nil
}

// liveThreadTitle is the label the thread's title is read under beside a live
// turn.
const liveThreadTitle = "thread_title"

// threadJoin reads a turn's conversation with it. The thread is joined on its
// whole key, tenant included, so a turn can only ever be listed beside a
// conversation of its own tenant.
func threadJoin() string {
	turnCols := buncolgen.AssistantTurnColumns
	threadCols := buncolgen.ThreadColumns

	return "JOIN " + buncolgen.ThreadTable.As(buncolgen.ThreadTable.Alias) +
		" ON " + threadCols.ID.EqColumn(turnCols.ThreadID) +
		" AND " + threadCols.OrganizationID.EqColumn(turnCols.OrganizationID) +
		" AND " + threadCols.BusinessUnitID.EqColumn(turnCols.BusinessUnitID)
}

func (r *repository) ListLive(
	ctx context.Context,
	req repositories.ListLiveAssistantTurnsRequest,
) ([]*repositories.LiveAssistantTurn, error) {
	cols := buncolgen.AssistantTurnColumns
	turns := make([]*repositories.LiveAssistantTurn, 0)

	q := r.db.DBForContext(ctx).NewSelect().
		Model(&turns).
		ColumnExpr(buncolgen.AssistantTurnTable.All()).
		ColumnExpr(buncolgen.ThreadColumns.Title.As(liveThreadTitle)).
		Join(threadJoin()).
		Apply(buncolgen.AssistantTurnApplyTenant(req.TenantInfo)).
		Where(cols.UserID.Eq(), req.UserID).
		Where(buncolgen.ThreadColumns.UserID.Eq(), req.UserID).
		Where(cols.Status.In(), bun.List(liveStatuses))
	if len(req.ExcludeOrigins) > 0 {
		q = q.Where(buncolgen.ThreadColumns.Origin.NotIn(), bun.List(req.ExcludeOrigins))
	}
	err := q.Order(cols.StartedAt.OrderAsc()).Scan(ctx)
	if err != nil {
		r.l.Error("failed to list live assistant turns",
			zap.String("user", req.UserID.String()),
			zap.Error(err),
		)

		return nil, fmt.Errorf("list the replies still in progress: %w", err)
	}

	return turns, nil
}

func (r *repository) RecordFingerprint(
	ctx context.Context,
	req repositories.RecordAssistantTurnFingerprintRequest,
) error {
	if req.Fingerprint == nil {
		return nil
	}

	cols := buncolgen.AssistantTurnColumns
	q := r.db.DBForContext(ctx).NewUpdate().
		Model((*conversation.AssistantTurn)(nil)).
		Set(cols.Fingerprint.Set(), req.Fingerprint).
		Where(cols.ID.Eq(), req.ID)
	if _, err := buncolgen.AssistantTurnScopeTenantUpdate(q, req.TenantInfo).Exec(ctx); err != nil {
		return fmt.Errorf("record the agent a turn ran as: %w", err)
	}

	return nil
}

func (r *repository) Complete(
	ctx context.Context,
	req repositories.CompleteAssistantTurnRequest,
) error {
	cols := buncolgen.AssistantTurnColumns

	q := r.db.DBForContext(ctx).NewUpdate().
		Model((*conversation.AssistantTurn)(nil)).
		Set(cols.Status.Set(), req.Status).
		Set(cols.CompletedAt.Set(), timeutils.NowUnix()).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Where(cols.ID.Eq(), req.ID)
	q = buncolgen.AssistantTurnScopeTenantUpdate(q, req.TenantInfo)
	if req.Error != "" {
		q = q.Set(cols.ErrorMessage.Set(), req.Error)
	}
	if !req.RunID.IsNil() {
		q = q.Set(cols.RunID.Set(), req.RunID)
	}

	if _, err := q.Exec(ctx); err != nil {
		r.l.Error("failed to complete an assistant turn",
			zap.String("turn", req.ID.String()),
			zap.String("status", string(req.Status)),
			zap.Error(err),
		)

		return fmt.Errorf("complete assistant turn: %w", err)
	}

	return nil
}

func (r *repository) MarkWorkflow(
	ctx context.Context,
	id pulid.ID,
	tenant pagination.TenantInfo,
	workflowID string,
) error {
	cols := buncolgen.AssistantTurnColumns

	q := r.db.DBForContext(ctx).NewUpdate().
		Model((*conversation.AssistantTurn)(nil)).
		Set(cols.WorkflowID.Set(), workflowID).
		Set(cols.Status.Set(), conversation.AssistantTurnStatusRunning).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Where(cols.ID.Eq(), id).
		Where(cols.Status.In(), bun.List(liveStatuses))

	_, err := buncolgen.AssistantTurnScopeTenantUpdate(q, tenant).Exec(ctx)
	if err != nil {
		return fmt.Errorf("record the workflow carrying a turn: %w", err)
	}

	return nil
}
