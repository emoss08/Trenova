package bankreceiptworkitemrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptworkitem/bankreceiptworkitemdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptworkitemports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var workItemErrors = dberror.Mapping{
	Entity: "BankReceiptWorkItem",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_bank_receipt_work_items_active",
			Field:   "bankReceiptId",
			Message: "An active work item already exists for this bank receipt",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_bank_receipt_work_items_receipt",
			Field:   "bankReceiptId",
			Message: "Bank receipt does not exist in your organization",
		},
		{
			Name:    "fk_bank_receipt_work_items_assigned_to",
			Field:   "assignedToUserId",
			Message: "Assigned user does not exist",
		},
		{
			Name:    "fk_bank_receipt_work_items_resolved_by",
			Field:   "resolvedByUserId",
			Message: "Resolving user does not exist",
		},
		{
			Name:    "fk_bank_receipt_work_items_created_by",
			Field:   "createdById",
			Message: "Created by user does not exist",
		},
		{
			Name:    "fk_bank_receipt_work_items_updated_by",
			Field:   "updatedById",
			Message: "Updated by user does not exist",
		},
	},
}

var activeStatuses = []bankreceiptworkitem.Status{
	bankreceiptworkitem.StatusOpen,
	bankreceiptworkitem.StatusAssigned,
	bankreceiptworkitem.StatusInReview,
}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) bankreceiptworkitemports.BankReceiptWorkItemRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.bank-receipt-work-item-repository")}
}

func (r *repository) GetByID(
	ctx context.Context,
	req bankreceiptworkitemports.GetBankReceiptWorkItemByIDRequest,
) (*bankreceiptworkitem.WorkItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*bankreceiptworkitem.WorkItem, error) {
		entity, err := bankreceiptworkitemdb.WorkItems.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, workItemErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) GetActiveByReceiptID(
	ctx context.Context,
	_ pagination.TenantInfo,
	bankReceiptID pulid.ID,
) (*bankreceiptworkitem.WorkItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*bankreceiptworkitem.WorkItem, error) {
		wi := bankreceiptworkitemdb.WorkItems
		entity, err := wi.Where(
			wi.BankReceiptID.Eq(bankReceiptID),
			wi.Status.In(activeStatuses...),
		).
			Limit(1).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewNotFoundError("bank receipt work item not found")
			}
			return nil, err
		}
		return entity, nil
	})
}

func (r *repository) ListActive(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*bankreceiptworkitem.WorkItem, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*bankreceiptworkitem.WorkItem, error) {
		wi := bankreceiptworkitemdb.WorkItems
		items, err := wi.Where(wi.Status.In(activeStatuses...)).
			Order(wi.CreatedAt.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list active bank receipt work items: %w", err)
		}
		return items, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bankreceiptworkitem.WorkItem, error) {
		if err := bankreceiptworkitemdb.WorkItems.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, workItemErrors.Map(err, entity.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *bankreceiptworkitem.WorkItem,
) (*bankreceiptworkitem.WorkItem, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*bankreceiptworkitem.WorkItem, error) {
		wi := bankreceiptworkitemdb.WorkItems
		db := r.db.Keel(ctx)
		if err := wi.Save(
			ctx,
			db,
			entity,
			wi.Status,
			wi.AssignedToUserID,
			wi.AssignedAt,
			wi.ResolutionType,
			wi.ResolutionNote,
			wi.ResolvedByUserID,
			wi.ResolvedAt,
			wi.UpdatedByID,
		); err != nil {
			return nil, workItemErrors.Map(err, entity.ID.String())
		}

		updated, err := wi.Get(ctx, db, entity.ID)
		if err != nil {
			return nil, workItemErrors.Map(err, entity.ID.String())
		}
		return updated, nil
	})
}
