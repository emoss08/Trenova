package assistantturnrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var turnErrors = dberror.Mapping{Entity: "AssistantTurn"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) assistantports.AssistantTurnRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assistantturn-repository"),
	}
}

var liveStatuses = []conversation.AssistantTurnStatus{
	conversation.AssistantTurnStatusPending,
	conversation.AssistantTurnStatusRunning,
}

type threadTitle struct {
	ID    pulid.ID
	Title string
}

func prepareTurn(turn *conversationturn.AssistantTurn) {
	if turn.Status == "" {
		turn.Status = conversation.AssistantTurnStatusPending
	}
	if turn.Origin == "" {
		turn.Origin = conversationturn.AssistantTurnOriginPerson
	}
	if turn.StartedAt == 0 {
		turn.StartedAt = timeutils.NowUnix()
	}
}

func getByIDQuery(
	req *assistantports.GetAssistantTurnRequest,
) *keel.Query[*conversationturn.AssistantTurn] {
	at := conversationdb.AssistantTurns

	return at.Where(
		at.ID.Eq(req.ID),
		keel.When(req.UserID.IsNotNil(), at.UserID.Eq(req.UserID)),
	)
}

func activeQuery(
	req *assistantports.ActiveAssistantTurnRequest,
) *keel.Query[*conversationturn.AssistantTurn] {
	at := conversationdb.AssistantTurns

	return at.Where(
		at.ThreadID.Eq(req.ThreadID),
		at.UserID.Eq(req.UserID),
		at.Status.In(liveStatuses...),
	)
}

func liveTurnsQuery(
	req *assistantports.ListLiveAssistantTurnsRequest,
) *keel.Query[*conversationturn.AssistantTurn] {
	at := conversationdb.AssistantTurns
	th := conversationdb.Threads.As("live_thread")

	return at.Join(th, th.ID.EqCol(at.ThreadID)).
		Where(
			at.UserID.Eq(req.UserID),
			th.UserID.Eq(req.UserID),
			at.Status.In(liveStatuses...),
			keel.When(len(req.ExcludeOrigins) > 0, th.Origin.NotIn(req.ExcludeOrigins...)),
		)
}

func liveThreadTitlesQuery(
	req *assistantports.ListLiveAssistantTurnsRequest,
) *keel.Query[*conversation.Thread] {
	th := conversationdb.Threads

	return th.Pick(th.ID, th.Title.Coalesce("")).Where(
		th.ID.InQuery(liveTurnsQuery(req).Pick(conversationdb.AssistantTurns.ThreadID)),
	)
}

func (r *repository) Start(
	ctx context.Context,
	turn *conversationturn.AssistantTurn,
) (*conversationturn.AssistantTurn, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*conversationturn.AssistantTurn, error) {
			prepareTurn(turn)
			if err := conversationdb.AssistantTurns.Create(ctx, r.db.Keel(ctx), turn); err != nil {
				if dberror.IsUniqueConstraintViolation(err) {
					return nil, assistantports.ErrTurnAlreadyRunning
				}

				r.l.Error("failed to start an assistant turn",
					zap.String("thread", turn.ThreadID.String()),
					zap.Error(err),
				)

				return nil, fmt.Errorf("start assistant turn: %w", err)
			}

			return turn, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req assistantports.GetAssistantTurnRequest,
) (*conversationturn.AssistantTurn, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversationturn.AssistantTurn, error) {
		turn, err := getByIDQuery(&req).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, turnErrors.Map(err, req.ID.String())
		}

		return turn, nil
	})
}

func (r *repository) Active(
	ctx context.Context,
	req assistantports.ActiveAssistantTurnRequest,
) (*conversationturn.AssistantTurn, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversationturn.AssistantTurn, error) {
		turn, err := activeQuery(&req).First(ctx, r.db.Keel(ctx))
		if err != nil {
			if errors.Is(err, keel.ErrNotFound) {
				return nil, nil
			}

			return nil, fmt.Errorf("read the turn this conversation is producing: %w", err)
		}

		return turn, nil
	})
}

func (r *repository) ListLive(
	ctx context.Context,
	req assistantports.ListLiveAssistantTurnsRequest,
) ([]*assistantports.LiveAssistantTurn, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*assistantports.LiveAssistantTurn, error) {
			at := conversationdb.AssistantTurns
			var (
				turns  []*conversationturn.AssistantTurn
				titles []threadTitle
			)
			err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				keel.AllInto(liveTurnsQuery(&req).Order(at.StartedAt.Asc()), &turns),
				keel.IntoSlice(liveThreadTitlesQuery(&req), &titles),
			)
			if err != nil {
				r.l.Error("failed to list live assistant turns",
					zap.String("user", req.UserID.String()),
					zap.Error(err),
				)

				return nil, fmt.Errorf("list the replies still in progress: %w", err)
			}

			byThread := make(map[pulid.ID]string, len(titles))
			for _, title := range titles {
				byThread[title.ID] = title.Title
			}
			out := make([]*assistantports.LiveAssistantTurn, 0, len(turns))
			for _, turn := range turns {
				out = append(out, &assistantports.LiveAssistantTurn{
					AssistantTurn: *turn,
					ThreadTitle:   byThread[turn.ThreadID],
				})
			}

			return out, nil
		},
	)
}

func (r *repository) RecordFingerprint(
	ctx context.Context,
	req assistantports.RecordAssistantTurnFingerprintRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req.Fingerprint == nil {
			return nil
		}

		at := conversationdb.AssistantTurns
		if _, err := at.Where(at.ID.Eq(req.ID)).
			Update(ctx, r.db.Keel(ctx), at.Fingerprint.To(req.Fingerprint)); err != nil {
			return fmt.Errorf("record the agent a turn ran as: %w", err)
		}

		return nil
	})
}

func (r *repository) Complete(
	ctx context.Context,
	req assistantports.CompleteAssistantTurnRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		at := conversationdb.AssistantTurns
		sets := []keel.Assignment{
			at.Status.To(req.Status),
			at.CompletedAt.To(timeutils.NowUnix()),
		}
		if req.Error != "" {
			sets = append(sets, at.ErrorMessage.To(req.Error))
		}
		if req.RunID.IsNotNil() {
			sets = append(sets, at.RunID.To(req.RunID))
		}

		if _, err := at.Where(at.ID.Eq(req.ID)).Update(ctx, r.db.Keel(ctx), sets...); err != nil {
			r.l.Error("failed to complete an assistant turn",
				zap.String("turn", req.ID.String()),
				zap.String("status", string(req.Status)),
				zap.Error(err),
			)

			return fmt.Errorf("complete assistant turn: %w", err)
		}

		return nil
	})
}

func (r *repository) MarkWorkflow(
	ctx context.Context,
	id pulid.ID,
	_ pagination.TenantInfo,
	workflowID string,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		at := conversationdb.AssistantTurns
		_, err := at.Where(at.ID.Eq(id), at.Status.In(liveStatuses...)).Update(
			ctx,
			r.db.Keel(ctx),
			at.WorkflowID.To(workflowID),
			at.Status.To(conversation.AssistantTurnStatusRunning),
		)
		if err != nil {
			return fmt.Errorf("record the workflow carrying a turn: %w", err)
		}

		return nil
	})
}
