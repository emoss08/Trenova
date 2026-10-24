//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editransferrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	directionInbound  = "inbound"
	directionOutbound = "outbound"
)

var transferErrors = dberror.Mapping{Entity: "EDITenderTransfer"}

var actionableInboundStatuses = []edi.TransferStatus{
	edi.TransferStatusSubmitted,
	edi.TransferStatusMappingRequired,
	edi.TransferStatusPendingApproval,
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

func New(p Params) ediports.EDILoadTenderTransferRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transfer-repository"),
	}
}

type statusCount struct {
	Status edi.TransferStatus
	Count  int64
}

func (r *repository) GetInboundStatusCounts(
	ctx context.Context,
	req ediports.GetEDITransferStatusCountsRequest,
) (map[edi.TransferStatus]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[edi.TransferStatus]int, error) {
		t := edidb.EDITransfers
		rows, err := keel.Into[statusCount](
			ctx,
			r.db.Keel(ctx),
			t.Where(
				tenantFilter(req.TenantInfo, directionInbound),
				keel.When(req.Since > 0, t.SubmittedAt.Gte(req.Since)),
			).
				Group(t.Status).
				Pick(t.Status, keel.Count()),
		)
		if err != nil {
			return nil, err
		}

		counts := make(map[edi.TransferStatus]int, len(rows))
		for _, row := range rows {
			counts[row.Status] = int(row.Count)
		}

		return counts, nil
	})
}

func (r *repository) ListInbound(
	ctx context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDITransfer], error) {
			q := listedTransfers(req, directionInbound).Order(newestFirst()...)
			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func (r *repository) ListOutbound(
	ctx context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDITransfer], error) {
			t := edidb.EDITransfers
			q := t.Where(
				tenantFilter(req.Filter.TenantInfo, directionOutbound),
				t.InboundMessageID.IsNull(),
			).Order(newestFirst()...)

			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func (r *repository) ListInboundCursor(
	ctx context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return r.listCursor(ctx, req, directionInbound)
}

func (r *repository) ListOutboundCursor(
	ctx context.Context,
	req *ediports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return r.listCursor(ctx, req, directionOutbound)
}

func (r *repository) listCursor(
	ctx context.Context,
	req *ediports.ListEDITransfersRequest,
	direction string,
) (*pagination.CursorListResult[*edi.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*edi.EDITransfer], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				listedTransfers(req, direction),
				&listquery.Request[edi.EDITransfer]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withPartners,
				},
			)
		},
	)
}

func (r *repository) GetTransferByID(
	ctx context.Context,
	req ediports.GetEDITransferByIDRequest,
) (*edi.EDITransfer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDITransfer, error) {
		t := edidb.EDITransfers
		entity, err := withPartners(t.Where(t.ID.Eq(req.ID), tenantFilter(req.TenantInfo, req.Direction))).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, transferErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetTransfersByIDs(
	ctx context.Context,
	req ediports.GetEDITransfersByIDsRequest,
) ([]*edi.EDITransfer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*edi.EDITransfer, error) {
		if len(req.TransferIDs) == 0 {
			return []*edi.EDITransfer{}, nil
		}

		t := edidb.EDITransfers
		return withPartners(t.Where(t.ID.In(req.TransferIDs...), tenantFilter(req.TenantInfo, ""))).
			Limit(len(req.TransferIDs)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *ediports.EDITransferSelectOptionsRequest,
) (*pagination.ListResult[*edi.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*edi.EDITransfer], error) {
			t := edidb.EDITransfers
			q := withPartners(t.Where(tenantFilter(req.SelectQueryRequest.TenantInfo, ""))).
				Order(newestFirst()...)

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return listquery.TextContainsFold(t.TenderPayload.Text("bol"), s)
				},
			)
		},
	)
}

func (r *repository) GetTransferForUpdate(
	ctx context.Context,
	req ediports.GetEDITransferForUpdateRequest,
) (*edi.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITransfer, error) {
		t := edidb.EDITransfers
		entity, err := t.Where(t.ID.Eq(req.ID), tenantFilter(req.TenantInfo, req.Direction)).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, transferErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetActionableInboundTransferByExternalReference(
	ctx context.Context,
	req ediports.GetActionableInboundEDITransferByExternalReferenceRequest,
) (*edi.EDITransfer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*edi.EDITransfer, error) {
		t := edidb.EDITransfers
		entity, err := t.Where(
			tenantFilter(req.TenantInfo, directionInbound),
			t.TargetPartnerID.Eq(req.PartnerID),
			t.InboundMessageID.IsNotNull(),
			t.TenderPayload.Text("ratingDetail", "externalShipmentId").Eq(req.ExternalReference),
			t.Status.In(actionableInboundStatuses...),
		).
			Order(newestFirst()...).
			First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, transferErrors.Map(err, req.ExternalReference)
		}

		return entity, nil
	})
}

func (r *repository) CreateTransfer(
	ctx context.Context,
	entity *edi.EDITransfer,
) (*edi.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITransfer, error) {
		if err := edidb.EDITransfers.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, transferErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateTransfer(
	ctx context.Context,
	entity *edi.EDITransfer,
) (*edi.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITransfer, error) {
		if err := edidb.EDITransfers.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, transferErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) SetApprovalWorkflowRunID(
	ctx context.Context,
	req ediports.SetEDITransferApprovalWorkflowRunIDRequest,
) (*edi.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*edi.EDITransfer, error) {
		t := edidb.EDITransfers
		rows, err := t.Where(t.ID.Eq(req.ID), tenantFilter(req.TenantInfo, directionInbound)).UpdateReturning(ctx, r.db.Keel(ctx), t.ApprovalWorkflowRunID.To(req.RunID), t.Version.ToExpr(t.Version))
		if err != nil {
			return nil, transferErrors.Map(err, req.ID.String())
		}
		if err = dberror.CheckAffected(
			int64(len(rows)),
			"EDITenderTransfer",
			req.ID.String(),
		); err != nil {
			return nil, err
		}

		return rows[0], nil
	})
}

func listedTransfers(
	req *ediports.ListEDITransfersRequest,
	direction string,
) *keel.Query[edi.EDITransfer] {
	t := edidb.EDITransfers
	return t.Where(
		tenantFilter(req.Filter.TenantInfo, direction),
		keel.When(direction == directionOutbound, t.InboundMessageID.IsNull()),
		keel.When(
			req.InboundFileID.IsNotNil(),
			t.InboundMessageID.InQuery(inboundFileMessages(req.InboundFileID)),
		),
	)
}

func inboundFileMessages(fileID pulid.ID) *keel.Query[edi.EDIMessage] {
	m := edidb.EDIMessages
	return m.Pick(m.ID).Where(m.InboundFileID.Eq(fileID))
}

func newestFirst() []keel.Order {
	t := edidb.EDITransfers
	return []keel.Order{t.CreatedAt.Desc(), t.ID.Desc()}
}

func withPartners(q *keel.Query[edi.EDITransfer]) *keel.Query[edi.EDITransfer] {
	t := edidb.EDITransfers
	return q.Load(t.SourcePartner(), t.TargetPartner())
}

func tenantFilter(tenant pagination.TenantInfo, direction string) keel.Pred {
	t := edidb.EDITransfers
	inbound := keel.And(
		t.TargetOrganizationID.Eq(tenant.OrgID),
		t.TargetBusinessUnitID.Eq(tenant.BuID),
	)
	outbound := keel.And(
		t.SourceOrganizationID.Eq(tenant.OrgID),
		t.SourceBusinessUnitID.Eq(tenant.BuID),
	)

	switch direction {
	case directionInbound:
		return inbound
	case directionOutbound:
		return outbound
	default:
		return keel.Or(inbound, outbound)
	}
}
