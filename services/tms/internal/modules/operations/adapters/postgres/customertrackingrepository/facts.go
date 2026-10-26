package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/checkcall/checkcalldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking/customertrackingdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason/holdreasondb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
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
		p := customertrackingdb.Profiles
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), p.AllTenants().
			Group(p.OrganizationID, p.BusinessUnitID).
			Pick(p.OrganizationID, p.BusinessUnitID).
			Where(p.NotificationsEnabledAt.IsNotNull()))
		if err != nil {
			return nil, fmt.Errorf("list organizations with customer status emails: %w", err)
		}
		return tenantsOf(rows), nil
	})
}

func (r *factRepository) HasNotifyingCustomers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		p := customertrackingdb.Profiles
		notifying, err := p.Where(p.NotificationsEnabledAt.IsNotNull()).
			Cache(resultcache.Settings()...).
			Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check for customers with status emails: %w", err)
		}
		return notifying, nil
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
		sp := shipmentdb.Shipments
		pc := customertrackingdb.Profiles
		d := documentdb.Documents

		recentPod := d.Where(podPredicates(d)...).
			Where(d.ResourceID.EqCol(sp.ID), d.CreatedAt.Gte(req.Since))

		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), sp.
			Join(pc, pc.CustomerID.EqCol(sp.CustomerID)).
			Pick(sp.ID).
			Where(
				pc.NotificationsEnabledAt.IsNotNull(),
				sp.Status.Ne(shipment.StatusCanceled),
				keel.Or(
					sp.Status.In(movingStatuses()...),
					sp.UpdatedAt.Gte(req.Since),
					sp.ActualDeliveryDate.Gte(req.Since),
					keel.Exists(recentPod),
				),
			).
			Order(sp.UpdatedAt.Desc()).
			Limit(limit))
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

func podPredicates(d *documentdb.DocumentsTable) []keel.Pred {
	return []keel.Pred{
		d.ResourceType.Eq(shipmentResourceType),
		d.IsCurrentVersion.IsTrue(),
		d.Status.NotIn(document.StatusRejected, document.StatusArchived),
		d.DocumentType().Code.Eq(documenttype.CodePOD),
	}
}

type podRow struct {
	ShipmentID pulid.ID
	DocumentID pulid.ID
	FileName   string
	FiledAt    int64
}

// LatestPods returns the earliest current proof of delivery filed on each
// shipment. The earliest, so that filing a second copy neither moves the time
// a customer was told nor owes a second email.
func (r *factRepository) LatestPods(
	ctx context.Context,
	_ pagination.TenantInfo,
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
			resources := make([]string, len(shipmentIDs))
			for i, id := range shipmentIDs {
				resources[i] = id.String()
			}

			d := documentdb.Documents
			rows, err := keel.Into[podRow](ctx, r.db.Keel(ctx), d.
				DistinctOn(d.ResourceID).
				Pick(d.ResourceID, d.ID, d.OriginalName, d.CreatedAt).
				Where(podPredicates(d)...).
				Where(d.ResourceID.In(resources...)).
				Order(d.ResourceID.Asc(), d.CreatedAt.Asc(), d.ID.Asc()))
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
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]customertrackingports.PublicHold, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]customertrackingports.PublicHold, error) {
			hc := shipmentdb.ShipmentHolds
			hr := holdreasondb.HoldReasons
			rows, err := keel.Into[customertrackingports.PublicHold](ctx, r.db.Keel(ctx), hc.
				LeftJoin(hr, hr.ID.EqCol(hc.HoldReasonID)).
				Pick(
					hc.Type,
					hr.Label.Coalesce(hc.ReasonCode, ""),
					hc.StartedAt,
				).
				Where(
					hc.ShipmentID.Eq(shipmentID),
					hc.VisibleToCustomer.IsTrue(),
					hc.ReleasedAt.IsNull(),
				).
				Order(hc.StartedAt.Asc()).
				Limit(maxVisibleHolds))
			if err != nil {
				return nil, fmt.Errorf("list customer-visible holds: %w", err)
			}
			return rows, nil
		},
	)
}

func (r *factRepository) CustomerVisibleNotes(
	ctx context.Context,
	_ pagination.TenantInfo,
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
			cc := shipmentdb.ShipmentComments
			rows, err := keel.Into[customertrackingports.PublicNote](ctx, r.db.Keel(ctx), cc.
				Pick(cc.Comment, cc.CreatedAt).
				Where(
					cc.ShipmentID.Eq(shipmentID),
					cc.Visibility.Eq(shipment.CommentVisibilityCustomer),
					cc.DeletedAt.IsNull(),
				).
				Order(cc.CreatedAt.Desc()).
				Limit(limit))
			if err != nil {
				return nil, fmt.Errorf("list customer-visible notes: %w", err)
			}
			return rows, nil
		},
	)
}

func (r *factRepository) LastReportedLocation(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) (*customertrackingports.PublicLocation, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customertrackingports.PublicLocation, error) {
			cc := checkcalldb.CheckCalls
			rows, err := keel.Into[customertrackingports.PublicLocation](ctx, r.db.Keel(ctx), cc.
				Pick(cc.City, cc.StateCode, cc.ReportedAt).
				Where(
					cc.ShipmentID.Eq(shipmentID),
					cc.City.IsNotNull(),
					cc.City.Ne(""),
				).
				Order(cc.ReportedAt.Desc()).
				Limit(1))
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
	CarrierName string
	Equipment   string
}

// Descriptors names the carrier hauling the shipment and the equipment it was
// ordered on. The carrier is the one on the latest move with a live carrier
// assignment, so a brokered shipment shows who has it now.
func (r *factRepository) Descriptors(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) (*customertrackingports.ShipmentDescriptors, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customertrackingports.ShipmentDescriptors, error) {
			sp := shipmentdb.Shipments
			sm := shipmentdb.ShipmentMoves
			ca := shipmentdb.CarrierAssignments
			cr := carrierdb.Carriers
			et := equipmenttypedb.EquipmentTypes

			carrierName := ca.
				Join(sm, sm.ID.EqCol(ca.ShipmentMoveID)).
				Join(cr, cr.ID.EqCol(ca.CarrierID)).
				Pick(cr.Name).
				Where(
					sm.ShipmentID.EqCol(sp.ID),
					sm.Status.Ne(shipment.MoveStatusCanceled),
					ca.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
				).
				Order(sm.Sequence.Desc()).
				Limit(1)

			equipment := et.
				Pick(et.Description.NullIf("").Coalesce(et.Code)).
				Where(et.ID.EqCol(sp.TrailerTypeID)).
				Limit(1)

			row, err := keel.IntoOne[descriptorRow](ctx, r.db.Keel(ctx), sp.
				Pick(
					keel.StringOf(keel.Scalar(carrierName)).Coalesce(""),
					keel.StringOf(keel.Scalar(equipment)).Coalesce(""),
				).
				Where(sp.ID.Eq(shipmentID)))
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
