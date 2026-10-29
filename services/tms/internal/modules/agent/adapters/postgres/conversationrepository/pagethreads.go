package conversationrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

var _ conversationports.PageThreadRepository = (*repository)(nil)

var errNotPageThread = errors.New("a page conversation needs a page origin, a subject and an owner")

func NewPageThreads(p Params) conversationports.PageThreadRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.page-thread-repository"),
	}
}

func pageKeyValid(key *conversationports.PageThreadKey) bool {
	return key.Origin.PageBound() && key.SubjectType != "" && key.SubjectID.IsNotNil() &&
		key.UserID.IsNotNil() && key.TenantInfo.OrgID.IsNotNil() && key.TenantInfo.BuID.IsNotNil()
}

func (r *repository) GetPageThread(
	ctx context.Context,
	key *conversationports.PageThreadKey,
) (*conversation.Thread, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.Thread, error) {
		if !pageKeyValid(key) {
			return nil, errNotPageThread
		}

		th := conversationdb.Threads
		entity, err := th.Where(
			th.UserID.Eq(key.UserID),
			th.Origin.Eq(key.Origin),
			th.SubjectType.Eq(key.SubjectType),
			th.SubjectID.Eq(key.SubjectID),
			th.Status.Eq(conversation.ThreadStatusActive),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Thread")
		}

		return entity, nil
	})
}

func (r *repository) ClaimPageThread(
	ctx context.Context,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.Thread, error) {
		key := &conversationports.PageThreadKey{
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

		if _, err := conversationdb.Threads.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*conversation.Thread{thread},
		); err != nil {
			r.l.Error("failed to claim a page conversation", zap.Error(err))

			return nil, fmt.Errorf("open the page's conversation: %w", err)
		}

		claimed, err := r.GetPageThread(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("read the page's conversation back: %w", err)
		}

		return claimed, nil
	})
}

func (r *repository) ArchiveSubjectThreads(
	ctx context.Context,
	req *conversationports.ArchiveSubjectThreadsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		if !req.Origin.PageBound() || req.SubjectType == "" || req.SubjectID.IsNil() {
			return 0, errNotPageThread
		}

		th := conversationdb.Threads
		n, err := th.Where(
			th.Origin.Eq(req.Origin),
			th.SubjectType.Eq(req.SubjectType),
			th.SubjectID.Eq(req.SubjectID),
			th.Status.Eq(conversation.ThreadStatusActive),
		).Update(ctx, r.db.Keel(ctx), th.Status.To(conversation.ThreadStatusArchived))
		if err != nil {
			r.l.Error("failed to archive a subject's page conversations", zap.Error(err))

			return 0, fmt.Errorf("close the page's conversations: %w", err)
		}

		return int(n), nil
	})
}
