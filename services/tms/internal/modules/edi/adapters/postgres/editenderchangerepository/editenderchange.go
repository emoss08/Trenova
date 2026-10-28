package editenderchangerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var tenderChangeErrors = dberror.Mapping{Entity: "EDITenderChange"}

var actionableStatuses = []edi.TenderChangeStatus{
	edi.TenderChangeStatusPendingReview,
	edi.TenderChangeStatusQueued,
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

func New(p Params) editenderports.EDITenderChangeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-tender-change-repository"),
	}
}

func (r *repository) ListTenderChanges(
	ctx context.Context,
	req *editenderports.ListEDITenderChangesRequest,
) (*pagination.ListResult[*edi.TenderChange], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.TenderChange], error) {
			tc := edidb.TenderChanges
			q := tc.Where(
				visibleTo(req.Filter.TenantInfo),
				keel.When(req.RecipientID.IsNotNil(), tc.RecipientID.Eq(req.RecipientID)),
				keel.When(
					req.SourceShipmentID.IsNotNil(),
					tc.SourceShipmentID.Eq(req.SourceShipmentID),
				),
				keel.When(req.Status != "", tc.Status.Eq(req.Status)),
			)

			return listquery.List(ctx, r.db.Keel(ctx), q, &listquery.Request[edi.TenderChange]{
				Filter: req.Filter,
				Shape:  withRecipient,
				Then:   []keel.Order{tc.CreatedAt.Desc()},
			})
		},
	)
}

func (r *repository) GetTenderChangeByID(
	ctx context.Context,
	req editenderports.GetEDITenderChangeByIDRequest,
) (*edi.TenderChange, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.TenderChange, error) {
		tc := edidb.TenderChanges
		entity, err := withRecipient(tc.Where(tc.ID.Eq(req.ID), visibleTo(req.TenantInfo))).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, tenderChangeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetTenderChangeByOutboundMessageID(
	ctx context.Context,
	req editenderports.GetEDITenderChangeByOutboundMessageIDRequest,
) (*edi.TenderChange, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.TenderChange, error) {
		tc := edidb.TenderChanges
		entity, err := withRecipient(tc.Where(
			tc.OutboundMessageID.Eq(req.OutboundMessageID),
			visibleTo(req.TenantInfo),
		)).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, tenderChangeErrors.Map(err, req.OutboundMessageID.String())
		}

		return entity, nil
	})
}

func (r *repository) CreateTenderChangeIdempotent(
	ctx context.Context,
	entity *edi.TenderChange,
) (*editenderports.CreateEDITenderChangeIdempotentResult, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*editenderports.CreateEDITenderChangeIdempotentResult, error) {
			db := r.db.Keel(ctx)
			tc := edidb.TenderChanges
			created, err := tc.CreateOrSkip(
				ctx,
				db,
				[]*edi.TenderChange{entity},
				tc.RecipientID,
				tc.SourceShipmentVersion,
				tc.NewPayloadHash,
				tc.ChangeType,
			)
			if err != nil {
				return nil, tenderChangeErrors.Map(err, entity.ID.String())
			}
			if created > 0 {
				return &editenderports.CreateEDITenderChangeIdempotentResult{
					TenderChange: entity,
					Created:      true,
				}, nil
			}

			existing, err := tc.Where(
				tc.RecipientID.Eq(entity.RecipientID),
				tc.SourceShipmentVersion.Eq(entity.SourceShipmentVersion),
				tc.NewPayloadHash.Eq(entity.NewPayloadHash),
				tc.ChangeType.Eq(entity.ChangeType),
			).One(ctx, db)
			if err != nil {
				return nil, tenderChangeErrors.Map(err, entity.ID.String())
			}

			return &editenderports.CreateEDITenderChangeIdempotentResult{
				TenderChange: existing,
				Created:      false,
			}, nil
		},
	)
}

func (r *repository) SupersedeActionableTenderChanges(
	ctx context.Context,
	req editenderports.SupersedeActionableEDITenderChangesRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		statuses := req.Statuses
		if len(statuses) == 0 {
			statuses = actionableStatuses
		}

		tc := edidb.TenderChanges
		_, err := tc.Where(
			tc.RecipientID.Eq(req.RecipientID),
			tc.Status.In(statuses...),
			keel.When(req.ExcludeChangeID.IsNotNil(), tc.ID.Ne(req.ExcludeChangeID)),
		).Update(ctx, r.db.Keel(ctx), tc.Status.To(edi.TenderChangeStatusSuperseded))

		return tenderChangeErrors.Map(err, req.RecipientID.String())
	})
}

func (r *repository) UpdateTenderChange(
	ctx context.Context,
	entity *edi.TenderChange,
) (*edi.TenderChange, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.TenderChange, error) {
		if err := edidb.TenderChanges.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, tenderChangeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func withRecipient(q *keel.Query[*edi.TenderChange]) *keel.Query[*edi.TenderChange] {
	return q.Load(edidb.TenderChanges.Recipient())
}

func visibleTo(tenant pagination.TenantInfo) keel.Pred {
	tc := edidb.TenderChanges
	scope := edidb.TenderRecipients.As("etr_scope")

	return keel.Or(
		tc.SourceOrganizationID.Eq(tenant.OrgID),
		keel.Exists(scope.Where(
			scope.ID.EqCol(tc.RecipientID),
			scope.RecipientOrganizationID.Eq(tenant.OrgID),
		)),
	)
}
