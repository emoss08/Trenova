package agentprofilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype/equipmenttypedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate/usstatedb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/places/domain/location/locationdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration/integrationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagefavorite/pagefavoritedb"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/lanequery"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// The aggregates a profile is made from.
//
// Like the insight metrics these are read-only reporting queries over a
// window, each counting several things in one pass. All of a profile's reads
// go out in one round trip; the scope keeps every table, joins included, to
// the tenant the profile is for.

type factsRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFacts(p Params) agentprofileports.FactsReader {
	return &factsRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-profile-facts-repository"),
	}
}

type volumeRow struct {
	Total     int64
	Canceled  int64
	Customers int64
}

type movesRow struct {
	Moves    int64
	Brokered int64
}

type fleetRow struct {
	Active       int64
	OutOfService int64
}

type workersRow struct {
	Active      int64
	Contractors int64
}

type organizationReads struct {
	volume   volumeRow
	moves    movesRow
	tractors fleetRow
	trailers fleetRow
	workers  workersRow
	activity *agentprofileports.OrganizationActivity
}

func inactiveFleet() []domaintypes.EquipmentStatus {
	return []domaintypes.EquipmentStatus{domaintypes.EquipmentStatusSold}
}

func outOfService() []domaintypes.EquipmentStatus {
	return []domaintypes.EquipmentStatus{
		domaintypes.EquipmentStatusOOS,
		domaintypes.EquipmentStatusAtMaintenance,
	}
}

func inWindow(w agentprofileports.FactsWindow) keel.Pred {
	sp := shipmentdb.Shipments

	return keel.And(sp.CreatedAt.Gte(w.Start), sp.CreatedAt.Lt(w.End))
}

func countDesc() keel.Order {
	return keel.Count().Desc()
}

func organizationItems(w agentprofileports.FactsWindow, reads *organizationReads) []keel.BatchItem {
	out := reads.activity
	sp := shipmentdb.Shipments
	sm := shipmentdb.ShipmentMoves
	stp := shipmentdb.Stops
	ca := shipmentdb.CarrierAssignments
	cus := customerdb.Customers
	loc := locationdb.Locations
	ust := usstatedb.UsStates
	wrk := workerdb.Workers
	integ := integrationdb.Integrations
	ac := accountingsyncdb.AccountingConnections
	ep := edidb.EDIPartners
	live := sp.Status.Ne(shipment.StatusCanceled)

	pickupName := keel.Raw(
		"concat_ws(', ', ?, NULLIF(concat_ws(' ', ?, ?), ''))",
		loc.Name,
		loc.City,
		ust.Abbreviation,
	)
	activeWorkers := wrk.Status.Eq(domaintypes.StatusActive)

	items := []keel.BatchItem{
		keel.IntoValue(sp.
			Pick(
				keel.Count(),
				keel.Count().Filter(sp.Status.Eq(shipment.StatusCanceled)),
				sp.CustomerID.CountDistinct().Filter(live),
			).
			Where(inWindow(w)), &reads.volume),
		keel.IntoValue(sm.
			Pick(
				keel.Count(),
				keel.Count().Filter(sm.CoverageType.Eq(shipment.MoveCoverageTypeCarrier)),
			).
			Join(sp, sp.ID.EqCol(sm.ShipmentID)).
			Where(live, inWindow(w)), &reads.moves),
		keel.IntoSlice(sp.
			Join(cus, cus.ID.EqCol(sp.CustomerID)).
			Group(cus.ID, cus.Name).
			Pick(cus.Name, keel.Count()).
			Where(live, inWindow(w)).
			Order(countDesc(), cus.Name.Asc()).
			Limit(agentprofileports.TopRows), &out.TopCustomers),
		keel.IntoSlice(lanequery.New(lanequery.Options{
			Shipments: []keel.Pred{
				sp.CreatedAt.Gte(w.Start),
				sp.CreatedAt.Lt(w.End),
				sp.Status.Ne(shipment.StatusCanceled),
			},
			Limit: agentprofileports.TopRows,
		}), &out.TopLanes),
		keel.IntoSlice(stp.
			Join(sm, sm.ID.EqCol(stp.ShipmentMoveID)).
			Join(sp, sp.ID.EqCol(sm.ShipmentID)).
			Join(loc, loc.ID.EqCol(stp.LocationID)).
			LeftJoin(ust, ust.ID.EqCol(loc.StateID)).
			Group(loc.ID, loc.Name, loc.City, ust.Abbreviation).
			Pick(pickupName, sp.ID.CountDistinct()).
			Where(stp.Type.In(shipment.PickupStopTypes()...), live, inWindow(w)).
			Order(keel.Desc(sp.ID.CountDistinct()), keel.Asc(pickupName)).
			Limit(agentprofileports.TopRows), &out.TopPickupLocations),
		keel.IntoValue(wrk.
			Pick(keel.Count(), keel.Count().Filter(wrk.Type.Eq(worker.WorkerTypeContractor))).
			Where(activeWorkers), &reads.workers),
		keel.IntoSlice(wrk.
			Group(wrk.DriverType).
			Pick(keel.Raw("?::text", wrk.DriverType), keel.Count()).
			Where(activeWorkers, wrk.DriverType.IsNotNull()).
			Order(countDesc(), keel.Asc(keel.Raw("?::text", wrk.DriverType))),
			&out.Workers.ByDriverType),
		keel.IntoValue(ca.
			Pick(ca.CarrierID.CountDistinct()).
			Join(sm, sm.ID.EqCol(ca.ShipmentMoveID)).
			Join(sp, sp.ID.EqCol(sm.ShipmentID)).
			Where(live, inWindow(w)), &out.CarriersUsed),
		keel.IntoSlice(integ.
			Pick(keel.Raw("?::text", integ.Type)).
			Where(integ.Enabled.IsTrue()).
			Order(integ.Type.Asc()), &out.Integrations),
		keel.IntoSlice(ac.
			Pick(keel.Raw("?::text", ac.IntegrationType), keel.Raw("?::text", ac.Status)).
			Where(ac.Status.Ne(accountingsync.ConnectionStatusDisconnected)).
			Order(ac.IntegrationType.Asc()), &out.AccountingConnections),
		keel.CountInto(ep.Where(ep.Status.Eq(domaintypes.StatusActive)), &out.EDIPartners),
	}

	items = append(items, fleetItems(tractorFleet(), &reads.tractors, &out.Tractors.ByType)...)

	return append(items, fleetItems(trailerFleet(), &reads.trailers, &out.Trailers.ByType)...)
}

type fleet struct {
	counts keel.Projection
	byType keel.Projection
}

func tractorFleet() fleet {
	t := tractordb.Tractors
	et := equipmenttypedb.EquipmentTypes
	name := keel.Raw("COALESCE(NULLIF(?, ''), ?)", et.Description, et.Code)

	return fleet{
		counts: t.Pick(
			keel.Count().Filter(t.Status.NotIn(inactiveFleet()...)),
			keel.Count().Filter(t.Status.In(outOfService()...)),
		),
		byType: t.Join(et, et.ID.EqCol(t.EquipmentTypeID)).
			Group(et.ID, et.Description, et.Code).
			Pick(name, keel.Count()).
			Where(t.Status.NotIn(inactiveFleet()...)).
			Order(countDesc(), keel.Asc(name)).
			Limit(agentprofileports.TopRows),
	}
}

func trailerFleet() fleet {
	t := trailerdb.Trailers
	et := equipmenttypedb.EquipmentTypes
	name := keel.Raw("COALESCE(NULLIF(?, ''), ?)", et.Description, et.Code)

	return fleet{
		counts: t.Pick(
			keel.Count().Filter(t.Status.NotIn(inactiveFleet()...)),
			keel.Count().Filter(t.Status.In(outOfService()...)),
		),
		byType: t.Join(et, et.ID.EqCol(t.EquipmentTypeID)).
			Group(et.ID, et.Description, et.Code).
			Pick(name, keel.Count()).
			Where(t.Status.NotIn(inactiveFleet()...)).
			Order(countDesc(), keel.Asc(name)).
			Limit(agentprofileports.TopRows),
	}
}

func fleetItems(
	f fleet,
	counts *fleetRow,
	byType *[]agentprofileports.NamedCount,
) []keel.BatchItem {
	return []keel.BatchItem{
		keel.IntoValue(f.counts, counts),
		keel.IntoSlice(f.byType, byType),
	}
}

func (r *factsRepository) OrganizationActivity(
	ctx context.Context,
	window agentprofileports.FactsWindow,
) (*agentprofileports.OrganizationActivity, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentprofileports.OrganizationActivity, error) {
			reads := &organizationReads{activity: new(agentprofileports.OrganizationActivity)}
			if err := keel.Batch(
				ctx,
				r.db.Keel(ctx),
				organizationItems(window, reads)...); err != nil {
				r.l.Error("failed to read organization profile facts",
					zap.String("organization", window.TenantInfo.OrgID.String()),
					zap.Error(err))

				return nil, fmt.Errorf("read organization activity: %w", err)
			}

			out := reads.activity
			out.Volume = agentprofileports.ShipmentVolume{
				Total:     reads.volume.Total,
				Canceled:  reads.volume.Canceled,
				Customers: reads.volume.Customers,
				Moves:     reads.moves.Moves,
				Brokered:  reads.moves.Brokered,
			}
			out.Tractors.Active, out.Tractors.OutOfService = reads.tractors.Active,
				reads.tractors.OutOfService
			out.Trailers.Active, out.Trailers.OutOfService = reads.trailers.Active,
				reads.trailers.OutOfService
			out.Workers.Active, out.Workers.Contractors = reads.workers.Active,
				reads.workers.Contractors
			emptyOrganizationLists(out)

			return out, nil
		},
	)
}

func emptyOrganizationLists(out *agentprofileports.OrganizationActivity) {
	if out.TopCustomers == nil {
		out.TopCustomers = []agentprofileports.NamedCount{}
	}
	if out.TopLanes == nil {
		out.TopLanes = []agentprofileports.LaneCount{}
	}
	if out.TopPickupLocations == nil {
		out.TopPickupLocations = []agentprofileports.NamedCount{}
	}
	if out.Tractors.ByType == nil {
		out.Tractors.ByType = []agentprofileports.NamedCount{}
	}
	if out.Trailers.ByType == nil {
		out.Trailers.ByType = []agentprofileports.NamedCount{}
	}
	if out.Workers.ByDriverType == nil {
		out.Workers.ByDriverType = []agentprofileports.NamedCount{}
	}
	if out.Integrations == nil {
		out.Integrations = []string{}
	}
	if out.AccountingConnections == nil {
		out.AccountingConnections = []agentprofileports.NamedStatus{}
	}
}

type userReads struct {
	titles   []string
	activity *agentprofileports.UserActivity
}

func userItems(req *agentprofileports.UserFactsRequest, reads *userReads) []keel.BatchItem {
	out := reads.activity
	w := req.Window
	usr := tenantdb.Users
	wrk := workerdb.Workers
	sp := shipmentdb.Shipments
	cus := customerdb.Customers
	ae := auditdb.Entries
	th := conversationdb.Threads
	msg := conversationdb.Messages
	msgThread := conversationdb.Threads.As("message_thread")
	pf := pagefavoritedb.PageFavorites
	statusName := keel.Raw("?::text", sp.Status)
	aboutName := keel.Raw("?::text", th.SubjectType)

	return []keel.BatchItem{
		keel.IntoSlice(usr.
			Pick(keel.Raw("COALESCE(?, '')", usr.JobTitle)).
			Where(usr.ID.Eq(req.UserID)).
			Limit(1), &reads.titles),
		keel.ExistsInto(wrk.Where(wrk.UserID.Eq(req.UserID)), &out.Driver),
		keel.IntoSlice(sp.
			Group(sp.Status).
			Pick(statusName, keel.Count()).
			Where(sp.OwnerID.Eq(req.UserID), inWindow(w)).
			Order(countDesc(), keel.Asc(statusName)), &out.OwnedShipments),
		keel.CountInto(
			sp.Where(sp.EnteredByID.Eq(req.UserID), inWindow(w)),
			&out.EnteredShipments,
		),
		keel.IntoSlice(sp.
			Join(cus, cus.ID.EqCol(sp.CustomerID)).
			Group(cus.ID, cus.Name).
			Pick(cus.Name, keel.Count()).
			Where(
				keel.Or(sp.OwnerID.Eq(req.UserID), sp.EnteredByID.Eq(req.UserID)),
				sp.Status.Ne(shipment.StatusCanceled),
				inWindow(w),
			).
			Order(countDesc(), cus.Name.Asc()).
			Limit(agentprofileports.TopRows), &out.Customers),
		keel.IntoSlice(ae.
			Group(ae.Resource).
			Pick(keel.Raw("?::text", ae.Resource), keel.Count()).
			Where(
				ae.UserID.Eq(req.UserID),
				ae.PrincipalType.Eq(authctx.PrincipalTypeUser),
				ae.Timestamp.Gte(w.Start),
				ae.Timestamp.Lt(w.End),
			).
			Order(countDesc(), ae.Resource.Asc()).
			Limit(agentprofileports.TopRows), &out.ChangedRecords),
		keel.CountInto(
			th.Where(th.UserID.Eq(req.UserID), th.LastMessageAt.Gte(w.Start)),
			&out.ConversationCount,
		),
		keel.IntoSlice(th.
			Group(th.SubjectType).
			Pick(aboutName, keel.Count()).
			Where(
				th.UserID.Eq(req.UserID),
				th.LastMessageAt.Gte(w.Start),
				th.SubjectType.IsNotNull(),
				keel.Raw("? <> ''", th.SubjectType),
			).
			Order(countDesc(), keel.Asc(aboutName)).
			Limit(agentprofileports.TopRows), &out.ConversationAbout),
		keel.IntoSlice(msg.
			Join(msgThread, msgThread.ID.EqCol(msg.ThreadID)).
			Group(msg.ToolName).
			Pick(msg.ToolName, keel.Count()).
			Where(
				msgThread.UserID.Eq(req.UserID),
				msg.Role.Eq(conversation.RoleTool),
				msg.ToolName.IsNotNull(),
				msg.CreatedAt.Gte(w.Start),
				msg.CreatedAt.Lt(w.End),
			).
			Order(countDesc(), msg.ToolName.Asc()).
			Limit(agentprofileports.TopRows), &out.ToolsUsed),
		keel.IntoSlice(pf.
			Pick(pf.PageTitle).
			Where(pf.UserID.Eq(req.UserID)).
			Order(pf.CreatedAt.Asc(), pf.ID.Asc()).
			Limit(agentprofileports.TopRows), &out.FavoritePages),
	}
}

func (r *factsRepository) UserActivity(
	ctx context.Context,
	req *agentprofileports.UserFactsRequest,
) (*agentprofileports.UserActivity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentprofileports.UserActivity, error) {
		reads := &userReads{activity: new(agentprofileports.UserActivity)}
		if err := keel.Batch(ctx, r.db.Keel(ctx), userItems(req, reads)...); err != nil {
			r.l.Error("failed to read person profile facts",
				zap.String("user", req.UserID.String()),
				zap.Error(err))

			return nil, fmt.Errorf("read person activity: %w", err)
		}

		out := reads.activity
		if len(reads.titles) > 0 {
			out.JobTitle = reads.titles[0]
		}
		emptyUserLists(out)

		return out, nil
	})
}

func emptyUserLists(out *agentprofileports.UserActivity) {
	if out.OwnedShipments == nil {
		out.OwnedShipments = []agentprofileports.NamedCount{}
	}
	if out.Customers == nil {
		out.Customers = []agentprofileports.NamedCount{}
	}
	if out.ChangedRecords == nil {
		out.ChangedRecords = []agentprofileports.NamedCount{}
	}
	if out.ConversationAbout == nil {
		out.ConversationAbout = []agentprofileports.NamedCount{}
	}
	if out.ToolsUsed == nil {
		out.ToolsUsed = []agentprofileports.NamedCount{}
	}
	if out.FavoritePages == nil {
		out.FavoritePages = []string{}
	}
}

// ActiveUsers are the people who talked to an agent since req.Since: a profile
// is read only by the turns a person starts, so nobody else needs one.
func (r *factsRepository) ActiveUsers(
	ctx context.Context,
	req agentprofileports.ActiveUsersRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		limit := req.Limit
		if limit <= 0 || limit > agentprofileports.MaxActiveUsers {
			limit = agentprofileports.MaxActiveUsers
		}

		turn := conversationdb.AssistantTurns
		uom := tenantdb.OrganizationMemberships
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), turn.
			Join(uom, uom.UserID.EqCol(turn.UserID)).
			Group(turn.UserID).
			Pick(turn.UserID).
			Where(
				turn.StartedAt.Gte(req.Since),
				keel.Or(uom.ExpiresAt.IsNull(), uom.ExpiresAt.Gt(req.Since)),
			).
			Order(turn.StartedAt.Max().Desc(), turn.UserID.Asc()).
			Limit(limit))
		if err != nil {
			return nil, fmt.Errorf("list people active with agents: %w", err)
		}
		if ids == nil {
			ids = []pulid.ID{}
		}

		return ids, nil
	})
}
