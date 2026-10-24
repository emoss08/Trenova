//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var mailboxErrors = dberror.Mapping{Entity: "Mailbox"}

type MailboxParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type mailboxRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMailboxRepository(p MailboxParams) inboundmessageports.InboundMailboxRepository {
	return &mailboxRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.inbound-mailbox-repository"),
	}
}

func (r *mailboxRepository) List(
	ctx context.Context,
	req *inboundmessageports.ListMailboxesRequest,
) (*pagination.ListResult[*inboundmessage.Mailbox], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*inboundmessage.Mailbox], error) {
			mb := inboundmessagedb.Mailboxes
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				mb.Where(keel.When(req.Status != "", mb.Status.Eq(req.Status))),
				&listquery.Request[inboundmessage.Mailbox]{
					Filter: req.Filter,
					Then:   []keel.Order{mb.CreatedAt.Desc()},
				},
			)
		},
	)
}

func (r *mailboxRepository) GetByID(
	ctx context.Context,
	req inboundmessageports.GetMailboxByIDRequest,
) (*inboundmessage.Mailbox, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*inboundmessage.Mailbox, error) {
		entity, err := inboundmessagedb.Mailboxes.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, mailboxErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

// GetByTokenHash resolves a delivery to its tenant.
//
// It is the one read here with no tenant scope, because the token is what
// identifies the tenant — there is nothing to scope by until it resolves. The
// hash is unique across every organization, so this is one indexed read, and a
// miss is an ordinary not-found rather than anything the caller should describe
// to whoever posted.
func (r *mailboxRepository) GetByTokenHash(
	ctx context.Context,
	req inboundmessageports.GetMailboxByTokenHashRequest,
) (*inboundmessage.Mailbox, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"resolve the inbound mailbox a provider delivered to before its tenant is known",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*inboundmessage.Mailbox, error) {
		mb := inboundmessagedb.Mailboxes
		entity, err := mb.AllTenants().Where(mb.TokenHash.Eq(req.TokenHash)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, mailboxErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *mailboxRepository) Create(
	ctx context.Context,
	entity *inboundmessage.Mailbox,
) (*inboundmessage.Mailbox, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*inboundmessage.Mailbox, error) {
		if err := inboundmessagedb.Mailboxes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *mailboxRepository) Update(
	ctx context.Context,
	entity *inboundmessage.Mailbox,
) (*inboundmessage.Mailbox, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*inboundmessage.Mailbox, error) {
		mb := inboundmessagedb.Mailboxes
		if err := mb.SaveExcept(
			ctx,
			r.db.Keel(ctx),
			entity,
			mb.LastReceivedAt,
			mb.TestToken,
			mb.TestSentAt,
			mb.TestReceivedAt,
		); err != nil {
			return nil, mailboxErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func NewMailboxStateRepository(p MailboxParams) inboundmessageports.TriageMailboxStateRepository {
	return &mailboxRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.inbound-mailbox-state-repository"),
	}
}

func (r *mailboxRepository) MarkReceived(
	ctx context.Context,
	req inboundmessageports.MailboxReceivedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		_, err := markReceivedQuery(req).Update(ctx, r.db.Keel(ctx), markReceivedAssignments(req)...)

		return err
	})
}

func markReceivedQuery(
	req inboundmessageports.MailboxReceivedRequest,
) *keel.Query[inboundmessage.Mailbox] {
	mb := inboundmessagedb.Mailboxes

	return mb.Where(mb.ID.Eq(req.MailboxID))
}

func markReceivedAssignments(req inboundmessageports.MailboxReceivedRequest) []keel.Assignment {
	mb := inboundmessagedb.Mailboxes

	return []keel.Assignment{
		mb.LastReceivedAt.ToExpr(keel.Raw(
			"GREATEST(COALESCE(?, 0), ?::bigint)",
			mb.LastReceivedAt,
			req.ReceivedAt,
		)),
		mb.Version.ToExpr(mb.Version),
		mb.UpdatedAt.ToExpr(mb.UpdatedAt),
	}
}

func (r *mailboxRepository) MarkTestSent(
	ctx context.Context,
	req inboundmessageports.MailboxTestRequest,
) (*inboundmessage.Mailbox, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*inboundmessage.Mailbox, error) {
		mb := inboundmessagedb.Mailboxes
		updated, err := mb.Where(mb.ID.Eq(req.MailboxID)).UpdateReturning(
			ctx,
			r.db.Keel(ctx),
			mb.TestToken.To(req.Token),
			mb.TestSentAt.To(req.At),
			mb.Version.ToExpr(mb.Version),
			mb.UpdatedAt.ToExpr(mb.UpdatedAt),
		)
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckAffected(
			int64(len(updated)),
			"Mailbox",
			req.MailboxID.String(),
		); err != nil {
			return nil, err
		}

		return updated[0], nil
	})
}

func (r *mailboxRepository) MarkTestReceived(
	ctx context.Context,
	req inboundmessageports.MailboxTestRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		n, err := markTestReceivedQuery(req).
			Update(ctx, r.db.Keel(ctx), markTestReceivedAssignments(req)...)
		if err != nil {
			return false, err
		}

		return n > 0, nil
	})
}

func markTestReceivedQuery(
	req inboundmessageports.MailboxTestRequest,
) *keel.Query[inboundmessage.Mailbox] {
	mb := inboundmessagedb.Mailboxes

	return mb.Where(mb.ID.Eq(req.MailboxID), mb.TestToken.Eq(req.Token))
}

func markTestReceivedAssignments(req inboundmessageports.MailboxTestRequest) []keel.Assignment {
	mb := inboundmessagedb.Mailboxes

	return []keel.Assignment{
		mb.TestReceivedAt.To(req.At),
		mb.TestToken.ToNull(),
		mb.Version.ToExpr(mb.Version),
		mb.UpdatedAt.ToExpr(mb.UpdatedAt),
	}
}
