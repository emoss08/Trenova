//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MailboxParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type mailboxRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewMailboxRepository(p MailboxParams) repositories.InboundMailboxRepository {
	return &mailboxRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.inbound-mailbox-repository"),
	}
}

func (r *mailboxRepository) List(
	ctx context.Context,
	req *repositories.ListMailboxesRequest,
) (*pagination.ListResult[*inboundmessage.Mailbox], error) {
	entities := make([]*inboundmessage.Mailbox, 0, req.Filter.Pagination.SafeLimit())
	cols := buncolgen.MailboxColumns

	query := r.db.DBForContext(ctx).NewSelect().Model(&entities)
	if req.Status != "" {
		query = query.Where(cols.Status.Eq(), req.Status)
	}

	total, err := querybuilder.ApplyFilters(query, "imbx", req.Filter, (*inboundmessage.Mailbox)(nil)).
		Order(cols.CreatedAt.OrderDesc()).
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}

	return &pagination.ListResult[*inboundmessage.Mailbox]{Items: entities, Total: total}, nil
}

func (r *mailboxRepository) GetByID(
	ctx context.Context,
	req repositories.GetMailboxByIDRequest,
) (*inboundmessage.Mailbox, error) {
	entity := new(inboundmessage.Mailbox)
	cols := buncolgen.MailboxColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.MailboxApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Mailbox")
	}

	return entity, nil
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
	req repositories.GetMailboxByTokenHashRequest,
) (*inboundmessage.Mailbox, error) {
	entity := new(inboundmessage.Mailbox)
	cols := buncolgen.MailboxColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.TokenHash.Eq(), req.TokenHash).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Mailbox")
	}

	return entity, nil
}

func (r *mailboxRepository) Create(
	ctx context.Context,
	entity *inboundmessage.Mailbox,
) (*inboundmessage.Mailbox, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *mailboxRepository) Update(
	ctx context.Context,
	entity *inboundmessage.Mailbox,
) (*inboundmessage.Mailbox, error) {
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		entity.Version = ov

		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "Mailbox", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}
