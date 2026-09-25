package conversationrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

var _ repositories.PageThreadRepository = (*repository)(nil)

var errNotPageThread = errors.New("a page conversation needs a page origin, a subject and an owner")

func NewPageThreads(p Params) repositories.PageThreadRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.page-thread-repository"),
	}
}

func pageKeyValid(key repositories.PageThreadKey) bool {
	return key.Origin.PageBound() && key.SubjectType != "" && key.SubjectID.IsNotNil() &&
		key.UserID.IsNotNil() && key.TenantInfo.OrgID.IsNotNil() && key.TenantInfo.BuID.IsNotNil()
}

func (r *repository) GetPageThread(
	ctx context.Context,
	key repositories.PageThreadKey,
) (*conversation.Thread, error) {
	if !pageKeyValid(key) {
		return nil, errNotPageThread
	}

	cols := buncolgen.ThreadColumns
	entity := new(conversation.Thread)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ThreadScopeTenant(sq, key.TenantInfo).
				Where(cols.UserID.Eq(), key.UserID).
				Where(cols.Origin.Eq(), key.Origin).
				Where(cols.SubjectType.Eq(), key.SubjectType).
				Where(cols.SubjectID.Eq(), key.SubjectID).
				Where(cols.Status.Eq(), conversation.ThreadStatusActive)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Thread")
	}

	return entity, nil
}

func (r *repository) ClaimPageThread(
	ctx context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	key := repositories.PageThreadKey{
		TenantInfo: pagination.TenantInfo{
			OrgID: thread.OrganizationID,
			BuID:  thread.BusinessUnitID,
		},
		UserID:      thread.UserID,
		Origin:      thread.Origin,
		SubjectType: thread.SubjectType,
		SubjectID:   thread.SubjectID,
	}
	if !pageKeyValid(key) {
		return nil, errNotPageThread
	}

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(thread).
		On("CONFLICT DO NOTHING").
		Exec(ctx); err != nil {
		r.l.Error("failed to claim a page conversation", zap.Error(err))

		return nil, fmt.Errorf("open the page's conversation: %w", err)
	}

	claimed, err := r.GetPageThread(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read the page's conversation back: %w", err)
	}

	return claimed, nil
}

func (r *repository) ArchiveSubjectThreads(
	ctx context.Context,
	req repositories.ArchiveSubjectThreadsRequest,
) (int, error) {
	if !req.Origin.PageBound() || req.SubjectType == "" || req.SubjectID.IsNil() {
		return 0, errNotPageThread
	}

	cols := buncolgen.ThreadColumns
	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*conversation.Thread)(nil)).
		Set(cols.Status.Set(), conversation.ThreadStatusArchived).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Inc(1)).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ThreadScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.Origin.Eq(), req.Origin).
				Where(cols.SubjectType.Eq(), req.SubjectType).
				Where(cols.SubjectID.Eq(), req.SubjectID).
				Where(cols.Status.Eq(), conversation.ThreadStatusActive)
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to archive a subject's page conversations", zap.Error(err))

		return 0, fmt.Errorf("close the page's conversations: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count the closed conversations: %w", err)
	}

	return int(affected), nil
}
