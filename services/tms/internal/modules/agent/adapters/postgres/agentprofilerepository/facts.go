package agentprofilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/lanequery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// The aggregates a profile is made from.
//
// Like the insight metrics these are read-only reporting queries over a
// window, written as explicit expressions because each counts several things
// in one pass. Every one is tenant-scoped on its driving table and again on
// each join: a join that forgets it is how one organization's customers would
// end up in another's prompt.

const (
	tenantJoin       = " AND %[1]s.organization_id = %[2]s.organization_id AND %[1]s.business_unit_id = %[2]s.business_unit_id"
	inactiveFleet    = "('Sold')"
	outOfServiceList = "('OutOfService', 'AtMaintenance')"
)

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

func joinOn(alias, table, on, parent string) string {
	return "JOIN " + table + " AS " + alias + " ON " + on + fmt.Sprintf(tenantJoin, alias, parent)
}

func (r *factsRepository) OrganizationActivity(
	ctx context.Context,
	window agentprofileports.FactsWindow,
) (*agentprofileports.OrganizationActivity, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*agentprofileports.OrganizationActivity, error) {
			out := new(agentprofileports.OrganizationActivity)
			steps := []struct {
				name string
				run  func(context.Context, bun.IDB, agentprofileports.FactsWindow, *agentprofileports.OrganizationActivity) error
			}{
				{"volume", readVolume},
				{"moves", readMoves},
				{"customers", readTopCustomers},
				{"lanes", readTopLanes},
				{"pickups", readTopPickups},
				{"tractors", readFleet("tractors", &out.Tractors)},
				{"trailers", readFleet("trailers", &out.Trailers)},
				{"workers", readWorkers},
				{"carriers", readCarriersUsed},
				{"integrations", readIntegrations},
				{"accounting", readAccounting},
				{"edi", readEDIPartners},
			}

			db := r.db.DBForContext(ctx)
			for _, step := range steps {
				if err := step.run(ctx, db, window, out); err != nil {
					r.l.Error("failed to read organization profile facts",
						zap.String("step", step.name),
						zap.String("organization", window.TenantInfo.OrgID.String()),
						zap.Error(err))

					return nil, fmt.Errorf("read organization %s: %w", step.name, err)
				}
			}

			return out, nil
		},
	)
}

func readVolume(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	return db.NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("COUNT(*)::bigint AS total").
		ColumnExpr("COUNT(*) FILTER (WHERE sp.status = ?)::bigint AS canceled", shipment.StatusCanceled).
		ColumnExpr("COUNT(DISTINCT sp.customer_id) FILTER (WHERE sp.status <> ?)::bigint AS customers",
			shipment.StatusCanceled).
		Where("sp.organization_id = ?", w.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", w.TenantInfo.BuID).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		Scan(ctx, &out.Volume.Total, &out.Volume.Canceled, &out.Volume.Customers)
}

func readMoves(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	return db.NewSelect().
		TableExpr("shipment_moves AS sm").
		ColumnExpr("COUNT(*)::bigint AS moves").
		ColumnExpr("COUNT(*) FILTER (WHERE sm.coverage_type = ?)::bigint AS brokered",
			shipment.MoveCoverageTypeCarrier).
		Join(joinOn("sp", "shipments", "sp.id = sm.shipment_id", "sm")).
		Where("sm.organization_id = ?", w.TenantInfo.OrgID).
		Where("sm.business_unit_id = ?", w.TenantInfo.BuID).
		Where("sp.status <> ?", shipment.StatusCanceled).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		Scan(ctx, &out.Volume.Moves, &out.Volume.Brokered)
}

func readTopCustomers(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	out.TopCustomers = make([]agentprofileports.NamedCount, 0, agentprofileports.TopRows)

	return db.NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("cus.name AS name").
		ColumnExpr("COUNT(*)::bigint AS count").
		Join(joinOn("cus", "customers", "cus.id = sp.customer_id", "sp")).
		Where("sp.organization_id = ?", w.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", w.TenantInfo.BuID).
		Where("sp.status <> ?", shipment.StatusCanceled).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		GroupExpr("cus.id, cus.name").
		OrderExpr("count DESC, cus.name ASC").
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.TopCustomers)
}

func readTopLanes(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	cols := shipmentcol.ShipmentColumns
	out.TopLanes = make([]agentprofileports.LaneCount, 0, agentprofileports.TopRows)
	tenant := pagination.TenantInfo{OrgID: w.TenantInfo.OrgID, BuID: w.TenantInfo.BuID}

	return lanequery.New(db, lanequery.Options{
		Tenant: &tenant,
		ShipmentFilter: func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(cols.CreatedAt.Gte(), w.Start).
				Where(cols.CreatedAt.Lt(), w.End).
				Where(cols.Status.Ne(), shipment.StatusCanceled)
		},
	}).
		OrderExpr("? DESC", bun.Ident(lanequery.CountColumn)).
		OrderExpr("? ASC", bun.Ident(lanequery.OriginStateColumn)).
		OrderExpr("? ASC", bun.Ident(lanequery.DestinationStateColumn)).
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.TopLanes)
}

func readTopPickups(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	out.TopPickupLocations = make([]agentprofileports.NamedCount, 0, agentprofileports.TopRows)

	return db.NewSelect().
		TableExpr("stops AS stp").
		ColumnExpr(
			"concat_ws(', ', loc.name, NULLIF(concat_ws(' ', loc.city, ust.abbreviation), '')) AS name",
		).
		ColumnExpr("COUNT(DISTINCT sp.id)::bigint AS count").
		Join(joinOn("sm", "shipment_moves", "sm.id = stp.shipment_move_id", "stp")).
		Join(joinOn("sp", "shipments", "sp.id = sm.shipment_id", "sm")).
		Join(joinOn("loc", "locations", "loc.id = stp.location_id", "stp")).
		Join("LEFT JOIN us_states AS ust ON ust.id = loc.state_id").
		Where("stp.organization_id = ?", w.TenantInfo.OrgID).
		Where("stp.business_unit_id = ?", w.TenantInfo.BuID).
		Where("stp.type IN (?)", bun.List(shipment.PickupStopTypes())).
		Where("sp.status <> ?", shipment.StatusCanceled).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		GroupExpr("loc.id, loc.name, loc.city, ust.abbreviation").
		OrderExpr("count DESC, name ASC").
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.TopPickupLocations)
}

func readFleet(
	table string,
	into *agentprofileports.FleetCounts,
) func(context.Context, bun.IDB, agentprofileports.FactsWindow, *agentprofileports.OrganizationActivity) error {
	return func(
		ctx context.Context,
		db bun.IDB,
		w agentprofileports.FactsWindow,
		_ *agentprofileports.OrganizationActivity,
	) error {
		if err := db.NewSelect().
			TableExpr(table+" AS eq").
			ColumnExpr("COUNT(*) FILTER (WHERE eq.status NOT IN "+inactiveFleet+")::bigint AS active").
			ColumnExpr("COUNT(*) FILTER (WHERE eq.status IN "+outOfServiceList+")::bigint AS out_of_service").
			Where("eq.organization_id = ?", w.TenantInfo.OrgID).
			Where("eq.business_unit_id = ?", w.TenantInfo.BuID).
			Scan(ctx, &into.Active, &into.OutOfService); err != nil {
			return err
		}

		into.ByType = make([]agentprofileports.NamedCount, 0, agentprofileports.TopRows)

		return db.NewSelect().
			TableExpr(table+" AS eq").
			ColumnExpr("COALESCE(NULLIF(et.description, ''), et.code) AS name").
			ColumnExpr("COUNT(*)::bigint AS count").
			Join(joinOn("et", "equipment_types", "et.id = eq.equipment_type_id", "eq")).
			Where("eq.organization_id = ?", w.TenantInfo.OrgID).
			Where("eq.business_unit_id = ?", w.TenantInfo.BuID).
			Where("eq.status NOT IN "+inactiveFleet).
			GroupExpr("et.id, et.description, et.code").
			OrderExpr("count DESC, name ASC").
			Limit(agentprofileports.TopRows).
			Scan(ctx, &into.ByType)
	}
}

func readWorkers(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	if err := db.NewSelect().
		TableExpr("workers AS wrk").
		ColumnExpr("COUNT(*)::bigint AS active").
		ColumnExpr("COUNT(*) FILTER (WHERE wrk.type = 'Contractor')::bigint AS contractors").
		Where("wrk.organization_id = ?", w.TenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", w.TenantInfo.BuID).
		Where("wrk.status = ?", domaintypes.StatusActive).
		Scan(ctx, &out.Workers.Active, &out.Workers.Contractors); err != nil {
		return err
	}

	out.Workers.ByDriverType = make([]agentprofileports.NamedCount, 0, 4)

	return db.NewSelect().
		TableExpr("workers AS wrk").
		ColumnExpr("wrk.driver_type::text AS name").
		ColumnExpr("COUNT(*)::bigint AS count").
		Where("wrk.organization_id = ?", w.TenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", w.TenantInfo.BuID).
		Where("wrk.status = ?", domaintypes.StatusActive).
		Where("wrk.driver_type IS NOT NULL").
		GroupExpr("wrk.driver_type").
		OrderExpr("count DESC, name ASC").
		Scan(ctx, &out.Workers.ByDriverType)
}

func readCarriersUsed(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	return db.NewSelect().
		TableExpr("carrier_assignments AS ca").
		ColumnExpr("COUNT(DISTINCT ca.carrier_id)::bigint").
		Join(joinOn("sm", "shipment_moves", "sm.id = ca.shipment_move_id", "ca")).
		Join(joinOn("sp", "shipments", "sp.id = sm.shipment_id", "sm")).
		Where("ca.organization_id = ?", w.TenantInfo.OrgID).
		Where("ca.business_unit_id = ?", w.TenantInfo.BuID).
		Where("sp.status <> ?", shipment.StatusCanceled).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		Scan(ctx, &out.CarriersUsed)
}

func readIntegrations(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	out.Integrations = make([]string, 0, 8)

	return db.NewSelect().
		TableExpr("integrations AS integ").
		ColumnExpr("integ.type::text").
		Where("integ.organization_id = ?", w.TenantInfo.OrgID).
		Where("integ.business_unit_id = ?", w.TenantInfo.BuID).
		Where("integ.enabled IS TRUE").
		OrderExpr("integ.type ASC").
		Scan(ctx, &out.Integrations)
}

func readAccounting(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	out.AccountingConnections = make([]agentprofileports.NamedStatus, 0, 2)

	return db.NewSelect().
		TableExpr("accounting_connections AS ac").
		ColumnExpr("ac.integration_type::text AS name").
		ColumnExpr("ac.status::text AS status").
		Where("ac.organization_id = ?", w.TenantInfo.OrgID).
		Where("ac.business_unit_id = ?", w.TenantInfo.BuID).
		Where("ac.status <> 'Disconnected'").
		OrderExpr("ac.integration_type ASC").
		Scan(ctx, &out.AccountingConnections)
}

func readEDIPartners(
	ctx context.Context,
	db bun.IDB,
	w agentprofileports.FactsWindow,
	out *agentprofileports.OrganizationActivity,
) error {
	return db.NewSelect().
		TableExpr("edi_partners AS ep").
		ColumnExpr("COUNT(*)::bigint").
		Where("ep.organization_id = ?", w.TenantInfo.OrgID).
		Where("ep.business_unit_id = ?", w.TenantInfo.BuID).
		Where("ep.status = ?", domaintypes.StatusActive).
		Scan(ctx, &out.EDIPartners)
}

func (r *factsRepository) UserActivity(
	ctx context.Context,
	req *agentprofileports.UserFactsRequest,
) (*agentprofileports.UserActivity, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentprofileports.UserActivity, error) {
		out := new(agentprofileports.UserActivity)
		steps := []struct {
			name string
			run  func(context.Context, bun.IDB, *agentprofileports.UserFactsRequest, *agentprofileports.UserActivity) error
		}{
			{"person", readPerson},
			{"shipments", readUserShipments},
			{"customers", readUserCustomers},
			{"changes", readUserChanges},
			{"conversations", readUserConversations},
			{"tools", readUserTools},
			{"favorites", readUserFavorites},
		}

		db := r.db.DBForContext(ctx)
		for _, step := range steps {
			if err := step.run(ctx, db, req, out); err != nil {
				r.l.Error("failed to read person profile facts",
					zap.String("step", step.name),
					zap.String("user", req.UserID.String()),
					zap.Error(err))

				return nil, fmt.Errorf("read person %s: %w", step.name, err)
			}
		}

		return out, nil
	})
}

func readPerson(
	ctx context.Context,
	db bun.IDB,
	req *agentprofileports.UserFactsRequest,
	out *agentprofileports.UserActivity,
) error {
	title := db.NewSelect().
		TableExpr("users AS usr").
		ColumnExpr("usr.job_title").
		Where("usr.id = ?", req.UserID).
		Limit(1)
	driver := db.NewSelect().
		TableExpr("workers AS wrk").
		ColumnExpr("1").
		Where("wrk.organization_id = ?", req.Window.TenantInfo.OrgID).
		Where("wrk.business_unit_id = ?", req.Window.TenantInfo.BuID).
		Where("wrk.user_id = ?", req.UserID)

	return db.NewSelect().
		ColumnExpr("COALESCE((?), '')", title).
		ColumnExpr("EXISTS (?)", driver).
		Scan(ctx, &out.JobTitle, &out.Driver)
}

func readUserShipments(
	ctx context.Context,
	db bun.IDB,
	req *agentprofileports.UserFactsRequest,
	out *agentprofileports.UserActivity,
) error {
	w := req.Window
	out.OwnedShipments = make([]agentprofileports.NamedCount, 0, 10)
	if err := db.NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("sp.status::text AS name").
		ColumnExpr("COUNT(*)::bigint AS count").
		Where("sp.organization_id = ?", w.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", w.TenantInfo.BuID).
		Where("sp.owner_id = ?", req.UserID).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		GroupExpr("sp.status").
		OrderExpr("count DESC, name ASC").
		Scan(ctx, &out.OwnedShipments); err != nil {
		return err
	}

	return db.NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("COUNT(*)::bigint").
		Where("sp.organization_id = ?", w.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", w.TenantInfo.BuID).
		Where("sp.entered_by_id = ?", req.UserID).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		Scan(ctx, &out.EnteredShipments)
}

func readUserCustomers(
	ctx context.Context,
	db bun.IDB,
	req *agentprofileports.UserFactsRequest,
	out *agentprofileports.UserActivity,
) error {
	w := req.Window
	out.Customers = make([]agentprofileports.NamedCount, 0, agentprofileports.TopRows)

	return db.NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("cus.name AS name").
		ColumnExpr("COUNT(*)::bigint AS count").
		Join(joinOn("cus", "customers", "cus.id = sp.customer_id", "sp")).
		Where("sp.organization_id = ?", w.TenantInfo.OrgID).
		Where("sp.business_unit_id = ?", w.TenantInfo.BuID).
		Where("(sp.owner_id = ? OR sp.entered_by_id = ?)", req.UserID, req.UserID).
		Where("sp.status <> ?", shipment.StatusCanceled).
		Where("sp.created_at >= ?", w.Start).
		Where("sp.created_at < ?", w.End).
		GroupExpr("cus.id, cus.name").
		OrderExpr("count DESC, cus.name ASC").
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.Customers)
}

func readUserChanges(
	ctx context.Context,
	db bun.IDB,
	req *agentprofileports.UserFactsRequest,
	out *agentprofileports.UserActivity,
) error {
	w := req.Window
	out.ChangedRecords = make([]agentprofileports.NamedCount, 0, agentprofileports.TopRows)

	return db.NewSelect().
		TableExpr("audit_entries AS ae").
		ColumnExpr("ae.resource AS name").
		ColumnExpr("COUNT(*)::bigint AS count").
		Where("ae.organization_id = ?", w.TenantInfo.OrgID).
		Where("ae.business_unit_id = ?", w.TenantInfo.BuID).
		Where("ae.user_id = ?", req.UserID).
		Where("ae.principal_type = ?", authctx.PrincipalTypeUser).
		Where("ae.timestamp >= ?", w.Start).
		Where("ae.timestamp < ?", w.End).
		GroupExpr("ae.resource").
		OrderExpr("count DESC, name ASC").
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.ChangedRecords)
}

func readUserConversations(
	ctx context.Context,
	db bun.IDB,
	req *agentprofileports.UserFactsRequest,
	out *agentprofileports.UserActivity,
) error {
	w := req.Window
	if err := db.NewSelect().
		TableExpr("assistant_threads AS th").
		ColumnExpr("COUNT(*)::bigint").
		Where("th.organization_id = ?", w.TenantInfo.OrgID).
		Where("th.business_unit_id = ?", w.TenantInfo.BuID).
		Where("th.user_id = ?", req.UserID).
		Where("th.last_message_at >= ?", w.Start).
		Scan(ctx, &out.ConversationCount); err != nil {
		return err
	}

	out.ConversationAbout = make([]agentprofileports.NamedCount, 0, agentprofileports.TopRows)

	return db.NewSelect().
		TableExpr("assistant_threads AS th").
		ColumnExpr("th.subject_type AS name").
		ColumnExpr("COUNT(*)::bigint AS count").
		Where("th.organization_id = ?", w.TenantInfo.OrgID).
		Where("th.business_unit_id = ?", w.TenantInfo.BuID).
		Where("th.user_id = ?", req.UserID).
		Where("th.last_message_at >= ?", w.Start).
		Where("th.subject_type IS NOT NULL").
		Where("th.subject_type <> ''").
		GroupExpr("th.subject_type").
		OrderExpr("count DESC, name ASC").
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.ConversationAbout)
}

func readUserTools(
	ctx context.Context,
	db bun.IDB,
	req *agentprofileports.UserFactsRequest,
	out *agentprofileports.UserActivity,
) error {
	w := req.Window
	out.ToolsUsed = make([]agentprofileports.NamedCount, 0, agentprofileports.TopRows)

	return db.NewSelect().
		TableExpr("assistant_messages AS msg").
		ColumnExpr("msg.tool_name AS name").
		ColumnExpr("COUNT(*)::bigint AS count").
		Join(joinOn("th", "assistant_threads", "th.id = msg.thread_id", "msg")).
		Where("msg.organization_id = ?", w.TenantInfo.OrgID).
		Where("msg.business_unit_id = ?", w.TenantInfo.BuID).
		Where("th.user_id = ?", req.UserID).
		Where("msg.role = ?", conversation.RoleTool).
		Where("msg.tool_name IS NOT NULL").
		Where("msg.created_at >= ?", w.Start).
		Where("msg.created_at < ?", w.End).
		GroupExpr("msg.tool_name").
		OrderExpr("count DESC, name ASC").
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.ToolsUsed)
}

func readUserFavorites(
	ctx context.Context,
	db bun.IDB,
	req *agentprofileports.UserFactsRequest,
	out *agentprofileports.UserActivity,
) error {
	w := req.Window
	out.FavoritePages = make([]string, 0, agentprofileports.TopRows)

	return db.NewSelect().
		TableExpr("page_favorites AS pf").
		ColumnExpr("pf.page_title").
		Where("pf.organization_id = ?", w.TenantInfo.OrgID).
		Where("pf.business_unit_id = ?", w.TenantInfo.BuID).
		Where("pf.user_id = ?", req.UserID).
		OrderExpr("pf.created_at ASC, pf.id ASC").
		Limit(agentprofileports.TopRows).
		Scan(ctx, &out.FavoritePages)
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

		ids := make([]pulid.ID, 0, min(limit, 64))
		err := r.db.DBForContext(ctx).
			NewSelect().
			TableExpr("assistant_turns AS atrn").
			ColumnExpr("atrn.user_id").
			Join(joinOn("uom", "user_organization_memberships",
				"uom.user_id = atrn.user_id", "atrn")).
			Where("atrn.organization_id = ?", req.TenantInfo.OrgID).
			Where("atrn.business_unit_id = ?", req.TenantInfo.BuID).
			Where("atrn.started_at >= ?", req.Since).
			Where("(uom.expires_at IS NULL OR uom.expires_at > ?)", req.Since).
			GroupExpr("atrn.user_id").
			OrderExpr("MAX(atrn.started_at) DESC, atrn.user_id ASC").
			Limit(int64(limit)).
			Scan(ctx, &ids)
		if err != nil {
			return nil, fmt.Errorf("list people active with agents: %w", err)
		}

		return ids, nil
	})
}
