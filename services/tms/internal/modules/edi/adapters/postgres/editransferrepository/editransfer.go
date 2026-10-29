//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package editransferrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
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

var actionableInboundStatuses = []editransfer.TransferStatus{
	editransfer.TransferStatusSubmitted,
	editransfer.TransferStatusMappingRequired,
	editransfer.TransferStatusPendingApproval,
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

func New(p Params) editenderports.EDILoadTenderTransferRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.edi-transfer-repository"),
	}
}

type statusCount struct {
	Status editransfer.TransferStatus
	Count  int64
}

func (r *repository) GetInboundStatusCounts(
	ctx context.Context,
	req editenderports.GetEDITransferStatusCountsRequest,
) (map[editransfer.TransferStatus]int, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[editransfer.TransferStatus]int, error) {
			t := editransferdb.EDITransfers
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

			counts := make(map[editransfer.TransferStatus]int, len(rows))
			for _, row := range rows {
				counts[row.Status] = int(row.Count)
			}

			return counts, nil
		},
	)
}

func (r *repository) ListInbound(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.ListResult[*editransfer.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*editransfer.EDITransfer], error) {
			q := listedTransfers(req, directionInbound).Order(newestFirst()...)
			return listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
		},
	)
}

func (r *repository) ListOutbound(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.ListResult[*editransfer.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*editransfer.EDITransfer], error) {
			t := editransferdb.EDITransfers
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
	req *editenderports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*editransfer.EDITransfer], error) {
	return r.listCursor(ctx, req, directionInbound)
}

func (r *repository) ListOutboundCursor(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
) (*pagination.CursorListResult[*editransfer.EDITransfer], error) {
	return r.listCursor(ctx, req, directionOutbound)
}

func (r *repository) listCursor(
	ctx context.Context,
	req *editenderports.ListEDITransfersRequest,
	direction string,
) (*pagination.CursorListResult[*editransfer.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*editransfer.EDITransfer], error) {
			return listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				listedTransfers(req, direction),
				&listquery.Request[editransfer.EDITransfer]{
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
	req editenderports.GetEDITransferByIDRequest,
) (*editransfer.EDITransfer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*editransfer.EDITransfer, error) {
		t := editransferdb.EDITransfers
		entity, err := withPartners(
			t.Where(t.ID.Eq(req.ID), tenantFilter(req.TenantInfo, req.Direction)),
		).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, transferErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetTransfersByIDs(
	ctx context.Context,
	req editenderports.GetEDITransfersByIDsRequest,
) ([]*editransfer.EDITransfer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*editransfer.EDITransfer, error) {
		if len(req.TransferIDs) == 0 {
			return []*editransfer.EDITransfer{}, nil
		}

		t := editransferdb.EDITransfers
		return withPartners(t.Where(t.ID.In(req.TransferIDs...), tenantFilter(req.TenantInfo, ""))).
			Limit(len(req.TransferIDs)).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *editenderports.EDITransferSelectOptionsRequest,
) (*pagination.ListResult[*editransfer.EDITransfer], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*editransfer.EDITransfer], error) {
			t := editransferdb.EDITransfers
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
	req editenderports.GetEDITransferForUpdateRequest,
) (*editransfer.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*editransfer.EDITransfer, error) {
		t := editransferdb.EDITransfers
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
	req editenderports.GetActionableInboundEDITransferByExternalReferenceRequest,
) (*editransfer.EDITransfer, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*editransfer.EDITransfer, error) {
		t := editransferdb.EDITransfers
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
	entity *editransfer.EDITransfer,
) (*editransfer.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*editransfer.EDITransfer, error) {
		if err := editransferdb.EDITransfers.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, transferErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) UpdateTransfer(
	ctx context.Context,
	entity *editransfer.EDITransfer,
) (*editransfer.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*editransfer.EDITransfer, error) {
		if err := editransferdb.EDITransfers.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, transferErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) SetApprovalWorkflowRunID(
	ctx context.Context,
	req editenderports.SetEDITransferApprovalWorkflowRunIDRequest,
) (*editransfer.EDITransfer, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*editransfer.EDITransfer, error) {
		t := editransferdb.EDITransfers
		rows, err := t.Where(t.ID.Eq(req.ID), tenantFilter(req.TenantInfo, directionInbound)).
			UpdateReturning(ctx, r.db.Keel(ctx), t.ApprovalWorkflowRunID.To(req.RunID), t.Version.ToExpr(t.Version))
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
	req *editenderports.ListEDITransfersRequest,
	direction string,
) *keel.Query[*editransfer.EDITransfer] {
	t := editransferdb.EDITransfers
	return t.Where(
		tenantFilter(req.Filter.TenantInfo, direction),
		keel.When(direction == directionOutbound, t.InboundMessageID.IsNull()),
		keel.When(
			req.InboundFileID.IsNotNil(),
			t.InboundMessageID.InQuery(inboundFileMessages(req.InboundFileID)),
		),
	)
}

func inboundFileMessages(fileID pulid.ID) *keel.Query[*edimessage.EDIMessage] {
	m := edimessagedb.EDIMessages
	return m.Pick(m.ID).Where(m.InboundFileID.Eq(fileID))
}

func newestFirst() []keel.Order {
	t := editransferdb.EDITransfers
	return []keel.Order{t.CreatedAt.Desc(), t.ID.Desc()}
}

func withPartners(q *keel.Query[*editransfer.EDITransfer]) *keel.Query[*editransfer.EDITransfer] {
	t := editransferdb.EDITransfers
	return q.Load(t.SourcePartner(), t.TargetPartner())
}

func tenantFilter(tenant pagination.TenantInfo, direction string) keel.Pred {
	t := editransferdb.EDITransfers
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
