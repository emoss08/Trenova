package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/checkcallcol"
	"github.com/emoss08/trenova/pkg/buncolgen/customertrackingcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/documenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/equipmenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/holdreasoncol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	shipmentResourceType = "shipment"
	maxVisibleHolds      = 20
	maxVisibleNotes      = 50
)

type factRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFacts(p Params) customertrackingports.FactRepository {
	return &factRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.customer-tracking-fact-repository"),
	}
}

func (r *factRepository) ListNotifyingTenants(
	ctx context.Context,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "find organizations whose customers receive status emails")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		cols := customertrackingcol.ProfileColumns
		rows := make([]tenantRow, 0, 8)
		err := r.db.DBForContext(ctx).NewSelect().
			Model((*customertracking.Profile)(nil)).
			ColumnExpr(cols.OrganizationID.As(orgColumn)).
			ColumnExpr(cols.BusinessUnitID.As(buColumn)).
			Where(cols.NotificationsEnabledAt.IsNotNull()).
			GroupExpr(cols.OrganizationID.String()).
			GroupExpr(cols.BusinessUnitID.String()).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list organizations with customer status emails: %w", err)
		}
		return tenantsOf(rows), nil
	})
}

// ListCandidateShipments finds the shipments that could owe their customer a
// status email: not canceled, for a customer with a rule on, and either still
// moving or touched since the given instant, including by a POD being filed.
func (r *factRepository) ListCandidateShipments(
	ctx context.Context,
	req *customertrackingports.CandidateShipmentsRequest,
) ([]pulid.ID, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = defaultCandidateLimit
	}
	limit = min(limit, maxCandidateLimit)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		db := r.db.DBForContext(ctx)
		sp := shipmentcol.ShipmentColumns
		pc := customertrackingcol.ProfileColumns
		pcTable := customertrackingcol.ProfileTable
		doc := documentcol.DocumentColumns

		recentPod := podQuery(db).
			ColumnExpr("1").
			Where(doc.ResourceID.EqColumn(sp.ID)).
			Where(doc.OrganizationID.EqColumn(sp.OrganizationID)).
			Where(doc.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
			Where(doc.CreatedAt.Gte(), req.Since)

		ids := make([]pulid.ID, 0, 64)
		err := db.NewSelect().
			Model((*shipment.Shipment)(nil)).
			Column(sp.ID.Bare()).
			Join("JOIN "+pcTable.As(pcTable.Alias)).
			JoinOn(pc.CustomerID.EqColumn(sp.CustomerID)).
			JoinOn(pc.OrganizationID.EqColumn(sp.OrganizationID)).
			JoinOn(pc.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
					Where(pc.NotificationsEnabledAt.IsNotNull()).
					Where(sp.Status.Ne(), shipment.StatusCanceled).
					WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
						return inner.
							Where(sp.Status.In(), bun.List(movingStatuses())).
							WhereOr(sp.UpdatedAt.Gte(), req.Since).
							WhereOr(sp.ActualDeliveryDate.Gte(), req.Since).
							WhereOr("EXISTS (?)", recentPod)
					})
			}).
			Order(sp.UpdatedAt.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx, &ids)
		if err != nil {
			return nil, fmt.Errorf("list shipments that may owe status emails: %w", err)
		}
		return ids, nil
	})
}

func movingStatuses() []shipment.Status {
	return []shipment.Status{
		shipment.StatusAssigned,
		shipment.StatusPartiallyAssigned,
		shipment.StatusInTransit,
		shipment.StatusDelayed,
		shipment.StatusPartiallyCompleted,
	}
}

func podQuery(db bun.IDB) *bun.SelectQuery {
	doc := documentcol.DocumentColumns
	dt := documenttypecol.DocumentTypeColumns
	dtTable := documenttypecol.DocumentTypeTable
	docTable := documentcol.DocumentTable

	return db.NewSelect().
		TableExpr(docTable.As(docTable.Alias)).
		Join("JOIN "+dtTable.As(dtTable.Alias)).
		JoinOn(dt.ID.EqColumn(doc.DocumentTypeID)).
		JoinOn(dt.OrganizationID.EqColumn(doc.OrganizationID)).
		JoinOn(dt.BusinessUnitID.EqColumn(doc.BusinessUnitID)).
		Where(doc.ResourceType.Eq(), shipmentResourceType).
		Where(doc.IsCurrentVersion.IsTrue()).
		Where(doc.Status.NotIn(), bun.List([]document.Status{
			document.StatusRejected,
			document.StatusArchived,
		})).
		Where(dt.Code.Eq(), documenttype.CodePOD)
}

type podRow struct {
	ShipmentID pulid.ID `bun:"shipment_id"`
	DocumentID pulid.ID `bun:"document_id"`
	FileName   string   `bun:"file_name"`
	FiledAt    int64    `bun:"filed_at"`
}

// LatestPods returns the earliest current proof of delivery filed on each
// shipment. The earliest, so that filing a second copy neither moves the time
// a customer was told nor owes a second email.
func (r *factRepository) LatestPods(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (map[pulid.ID]*customertrackingports.PodDocument, error) {
	result := make(map[pulid.ID]*customertrackingports.PodDocument, len(shipmentIDs))
	if len(shipmentIDs) == 0 {
		return result, nil
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*customertrackingports.PodDocument, error) {
			doc := documentcol.DocumentColumns
			rows := make([]podRow, 0, len(shipmentIDs))
			err := podQuery(r.db.DBForContext(ctx)).
				DistinctOn(doc.ResourceID.Qualified()).
				ColumnExpr(doc.ResourceID.As("shipment_id")).
				ColumnExpr(doc.ID.As("document_id")).
				ColumnExpr(doc.OriginalName.As("file_name")).
				ColumnExpr(doc.CreatedAt.As("filed_at")).
				Where(doc.OrganizationID.Eq(), tenantInfo.OrgID).
				Where(doc.BusinessUnitID.Eq(), tenantInfo.BuID).
				Where(doc.ResourceID.In(), bun.List(shipmentIDs)).
				OrderExpr(doc.ResourceID.Qualified()).
				Order(doc.CreatedAt.OrderAsc(), doc.ID.OrderAsc()).
				Scan(ctx, &rows)
			if err != nil {
				return nil, fmt.Errorf("list proof of delivery documents: %w", err)
			}
			for _, row := range rows {
				result[row.ShipmentID] = &customertrackingports.PodDocument{
					ShipmentID: row.ShipmentID,
					DocumentID: row.DocumentID,
					FileName:   row.FileName,
					FiledAt:    row.FiledAt,
				}
			}
			return result, nil
		},
	)
}

func (r *factRepository) CustomerVisibleHolds(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]customertrackingports.PublicHold, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]customertrackingports.PublicHold, error) {
			hc := shipmentcol.ShipmentHoldColumns
			hr := holdreasoncol.HoldReasonColumns
			hrTable := holdreasoncol.HoldReasonTable
			rows := make([]customertrackingports.PublicHold, 0, 2)
			err := r.db.DBForContext(ctx).NewSelect().
				Model((*shipment.ShipmentHold)(nil)).
				ColumnExpr(hc.Type.As("type")).
				ColumnExpr(buncolgen.Expr("COALESCE({0}, {1}, '')", hr.Label, hc.ReasonCode)+" AS label").
				ColumnExpr(hc.StartedAt.As("started_at")).
				Join("LEFT JOIN "+hrTable.As(hrTable.Alias)).
				JoinOn(hr.ID.EqColumn(hc.HoldReasonID)).
				JoinOn(hr.OrganizationID.EqColumn(hc.OrganizationID)).
				JoinOn(hr.BusinessUnitID.EqColumn(hc.BusinessUnitID)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return shipmentcol.ShipmentHoldScopeTenant(sq, tenantInfo).
						Where(hc.ShipmentID.Eq(), shipmentID).
						Where(hc.VisibleToCustomer.IsTrue()).
						Where(hc.ReleasedAt.IsNull())
				}).
				Order(hc.StartedAt.OrderAsc()).
				Limit(maxVisibleHolds).
				Scan(ctx, &rows)
			if err != nil {
				return nil, fmt.Errorf("list customer-visible holds: %w", err)
			}
			return rows, nil
		},
	)
}

func (r *factRepository) CustomerVisibleNotes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
	limit int,
) ([]customertrackingports.PublicNote, error) {
	if limit <= 0 || limit > maxVisibleNotes {
		limit = maxVisibleNotes
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]customertrackingports.PublicNote, error) {
			cc := shipmentcol.ShipmentCommentColumns
			rows := make([]customertrackingports.PublicNote, 0, 4)
			err := r.db.DBForContext(ctx).NewSelect().
				Model((*shipment.ShipmentComment)(nil)).
				ColumnExpr(cc.Comment.As("text")).
				ColumnExpr(cc.CreatedAt.As("posted_at")).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return shipmentcol.ShipmentCommentScopeTenant(sq, tenantInfo).
						Where(cc.ShipmentID.Eq(), shipmentID).
						Where(cc.Visibility.Eq(), shipment.CommentVisibilityCustomer).
						Where(cc.DeletedAt.IsNull())
				}).
				Order(cc.CreatedAt.OrderDesc()).
				Limit(int64(limit)).
				Scan(ctx, &rows)
			if err != nil {
				return nil, fmt.Errorf("list customer-visible notes: %w", err)
			}
			return rows, nil
		},
	)
}

func (r *factRepository) LastReportedLocation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*customertrackingports.PublicLocation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customertrackingports.PublicLocation, error) {
			cc := checkcallcol.CheckCallColumns
			rows := make([]customertrackingports.PublicLocation, 0, 1)
			err := r.db.DBForContext(ctx).NewSelect().
				Model((*checkcall.CheckCall)(nil)).
				ColumnExpr(cc.City.As("city")).
				ColumnExpr(cc.StateCode.As("state_code")).
				ColumnExpr(cc.ReportedAt.As("reported_at")).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return checkcallcol.CheckCallScopeTenant(sq, tenantInfo).
						Where(cc.ShipmentID.Eq(), shipmentID).
						Where(cc.City.IsNotNull()).
						Where(cc.City.Ne(), "")
				}).
				Order(cc.ReportedAt.OrderDesc()).
				Limit(1).
				Scan(ctx, &rows)
			if err != nil {
				return nil, fmt.Errorf("get last reported location: %w", err)
			}
			if len(rows) == 0 {
				return nil, nil
			}
			return &rows[0], nil
		},
	)
}

type descriptorRow struct {
	CarrierName string `bun:"carrier_name"`
	Equipment   string `bun:"equipment"`
}

// Descriptors names the carrier hauling the shipment and the equipment it was
// ordered on. The carrier is the one on the latest move with a live carrier
// assignment, so a brokered shipment shows who has it now.
func (r *factRepository) Descriptors(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*customertrackingports.ShipmentDescriptors, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customertrackingports.ShipmentDescriptors, error) {
			db := r.db.DBForContext(ctx)
			sp := shipmentcol.ShipmentColumns
			sm := shipmentcol.ShipmentMoveColumns
			ca := shipmentcol.CarrierAssignmentColumns
			cr := carriercol.CarrierColumns
			et := equipmenttypecol.EquipmentTypeColumns
			smTable := shipmentcol.ShipmentMoveTable
			caTable := shipmentcol.CarrierAssignmentTable
			crTable := carriercol.CarrierTable
			etTable := equipmenttypecol.EquipmentTypeTable

			carrierName := db.NewSelect().
				TableExpr(caTable.As(caTable.Alias)).
				ColumnExpr(cr.Name.Qualified()).
				Join("JOIN "+smTable.As(smTable.Alias)).
				JoinOn(sm.ID.EqColumn(ca.ShipmentMoveID)).
				JoinOn(sm.OrganizationID.EqColumn(ca.OrganizationID)).
				JoinOn(sm.BusinessUnitID.EqColumn(ca.BusinessUnitID)).
				Join("JOIN "+crTable.As(crTable.Alias)).
				JoinOn(cr.ID.EqColumn(ca.CarrierID)).
				JoinOn(cr.OrganizationID.EqColumn(ca.OrganizationID)).
				JoinOn(cr.BusinessUnitID.EqColumn(ca.BusinessUnitID)).
				Where(sm.ShipmentID.EqColumn(sp.ID)).
				Where(sm.OrganizationID.EqColumn(sp.OrganizationID)).
				Where(sm.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
				Where(sm.Status.Ne(), shipment.MoveStatusCanceled).
				Where(ca.Status.Ne(), shipment.CarrierAssignmentStatusCanceled).
				Order(sm.Sequence.OrderDesc()).
				Limit(1)

			equipment := db.NewSelect().
				TableExpr(etTable.As(etTable.Alias)).
				ColumnExpr(buncolgen.Expr("COALESCE(NULLIF({0}, ''), {1})", et.Description, et.Code)).
				Where(et.ID.EqColumn(sp.TrailerTypeID)).
				Where(et.OrganizationID.EqColumn(sp.OrganizationID)).
				Where(et.BusinessUnitID.EqColumn(sp.BusinessUnitID)).
				Limit(1)

			row := descriptorRow{}
			err := db.NewSelect().
				Model((*shipment.Shipment)(nil)).
				ColumnExpr("COALESCE((?), '') AS carrier_name", carrierName).
				ColumnExpr("COALESCE((?), '') AS equipment", equipment).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return shipmentcol.ShipmentScopeTenant(sq, tenantInfo).
						Where(sp.ID.Eq(), shipmentID)
				}).
				Scan(ctx, &row)
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return &customertrackingports.ShipmentDescriptors{}, nil
				}
				return nil, fmt.Errorf("describe shipment for customer tracking: %w", err)
			}
			return &customertrackingports.ShipmentDescriptors{
				CarrierName: row.CarrierName,
				Equipment:   row.Equipment,
			}, nil
		},
	)
}
