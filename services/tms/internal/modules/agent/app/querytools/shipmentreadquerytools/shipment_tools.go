package shipmentreadquerytools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxShipmentComments = 20
	maxShipmentHolds    = 25
	paramDetail         = "detail"
)

type getShipmentTool struct {
	repo          shipmentports.ShipmentRepository
	comments      shipmentcommentports.ShipmentCommentRepository
	holds         shipmentholdports.ShipmentHoldRepository
	batchHolds    shipmentports.ActiveHoldReader
	batchComments shipmentports.NewestCommentReader
	access        fieldgate.Access
}

func ProvideGetShipmentTool(
	repo shipmentports.ShipmentRepository,
	comments shipmentcommentports.ShipmentCommentRepository,
	holds shipmentholdports.ShipmentHoldRepository,
	batchHolds shipmentports.ActiveHoldReader,
	batchComments shipmentports.NewestCommentReader,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return newGetShipmentTool(repo, comments, holds, permissions).
		withBatchReads(batchHolds, batchComments)
}

func newGetShipmentTool(
	repo shipmentports.ShipmentRepository,
	comments shipmentcommentports.ShipmentCommentRepository,
	holds shipmentholdports.ShipmentHoldRepository,
	permissions permissionports.PermissionEngine,
) *getShipmentTool {
	return &getShipmentTool{
		repo:     repo,
		comments: comments,
		holds:    holds,
		access:   fieldgate.NewAccess(permissions),
	}
}

func (t *getShipmentTool) Name() string { return "shipment_get" }

func (t *getShipmentTool) Description() string {
	return "Read shipments (loads) by id, passing every load you need in one call, up to 25, " +
		"or one by its PRO number. One load shows the customer and rating, each move's " +
		"stops with their windows in the stop's local time, who drives it with what " +
		"equipment, the holds on it now and its newest comments; pass detail full only for " +
		"a field that summary leaves out. Several come as a table of each load's route, " +
		"driver, holds in force with their notes and newest comment. Each hold carries the " +
		"holdId shipment_update_hold and shipment_release_hold take. leftOutForAccess says " +
		"when the holds or comments could not be read. Find loads with shipment_list when " +
		"you have only a customer, a place, a PO or a BOL."
}

func (t *getShipmentTool) SearchTerms() []string {
	return []string{"assigned driver", "who is driving", "shipment details"}
}

func (t *getShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramShipmentIDs:        shipmentRefsProperty(),
			listtool.ParamProNumber: proNumberProperty(),
			paramDetail: agenttoolschema.Enum(
				"How much to return. Omit it for the summary: the customer, the rating, "+
					"the moves with their stops, assignment and carrier, the commodities, "+
					"charges, holds and comments. full returns every stored field.",
				shipmentquerytools.DetailLevels,
			),
		},
		"additionalProperties": false,
	}
}

func (t *getShipmentTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceShipment,
		Reads:    agenttooltrust.ExternalReadMarked,
		Source:   agent.TaintSourceRecordNote,
		Rationale: "Reads a shipment with its newest comments, some of which a driver, a " +
			"trading partner or another system outside the organization wrote; nothing " +
			"changes and nothing is sent.",
	})
}

func (t *getShipmentTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	detail, err := shipmentDetailOf(params.Params)
	if err != nil {
		return nil, err
	}

	ids, pro, err := shipmentRefs(params.Params)
	if err != nil {
		return nil, err
	}
	if len(ids) > 1 {
		return t.several(ctx, params, ids, detail)
	}
	var id pulid.ID
	if len(ids) == 1 {
		id = ids[0]
	}

	tenant := querytool.TenantOf(params)
	entity, err := loadShipment(ctx, t.repo, tenant, id, pro)
	if err != nil {
		return nil, err
	}

	readable := t.readableActivity(ctx, params)
	comments, err := t.recentComments(ctx, readable, tenant, entity.ID)
	if err != nil {
		return nil, err
	}

	holds, err := t.activeHolds(ctx, readable, tenant, entity.ID)
	if err != nil {
		return nil, err
	}

	activity := newShipmentActivity(comments, holds)
	activity.LeftOutForAccess = readable.leftOutNote(params)
	people := t.access.Redactor(ctx, params)

	if detail == shipmentquerytools.DetailFull {
		return people.Withhold(newShipmentView(entity, activity))
	}

	return people.Annotate(summarizeShipment(&shipmentSummaryInput{
		entity:   entity,
		activity: activity,
		timezone: params.Timezone,
		people:   people,
	}))
}

func shipmentDetailOf(params map[string]any) (shipmentquerytools.Detail, error) {
	value, err := querytool.ValidEnum(
		params,
		paramDetail,
		shipmentquerytools.DetailLevels.AsStrings(),
	)
	if err != nil {
		return "", err
	}
	if value == "" {
		return shipmentquerytools.DetailSummary, nil
	}

	return shipmentquerytools.Detail(value), nil
}

type shipmentActivityAccess struct {
	comments         bool
	holds            bool
	commentsWithheld bool
	holdsWithheld    bool
}

func (t *getShipmentTool) readableActivity(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) shipmentActivityAccess {
	var access shipmentActivityAccess
	if t.comments != nil {
		access.comments = t.access.MayRead(ctx, params, permission.ResourceShipmentComment)
		access.commentsWithheld = !access.comments
	}
	if t.holds != nil {
		access.holds = t.access.MayRead(ctx, params, permission.ResourceShipmentHold)
		access.holdsWithheld = !access.holds
	}

	return access
}

func (a shipmentActivityAccess) leftOutNote(params *agenttoolports.QueryToolParams) string {
	switch {
	case a.commentsWithheld && a.holdsWithheld:
		return fmt.Sprintf("Comments and holds are left out: %s does not include reading "+
			"shipment comments or shipment holds. A note a dispatcher left or a hold on this "+
			"load would not appear here, so the empty lists do not mean there are none; say "+
			"so rather than concluding the record is silent.", shipmentquerytools.AccessHolder(params))
	case a.commentsWithheld:
		return fmt.Sprintf("Comments are left out: %s does not include reading shipment "+
			"comments. A note a dispatcher left on this load would not appear here, so do "+
			"not conclude the record says nothing about why; say the comments could not be "+
			"read.", shipmentquerytools.AccessHolder(params))
	case a.holdsWithheld:
		return fmt.Sprintf("Holds are left out: %s does not include reading shipment holds, "+
			"so the empty activeHolds does not mean the load is not held.", shipmentquerytools.AccessHolder(params))
	default:
		return ""
	}
}

func (t *getShipmentTool) activeHolds(
	ctx context.Context,
	readable shipmentActivityAccess,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipmenthold.ShipmentHold, error) {
	if !readable.holds {
		return nil, nil
	}

	page, err := t.holds.ListByShipmentID(ctx, &shipmentholdports.ListShipmentHoldsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: maxShipmentHolds},
		},
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("read the shipment's holds: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	active := make([]*shipmenthold.ShipmentHold, 0, len(page.Items))
	for _, hold := range page.Items {
		if hold != nil && hold.ReleasedAt == nil {
			active = append(active, hold)
		}
	}

	return active, nil
}

func (t *getShipmentTool) recentComments(
	ctx context.Context,
	readable shipmentActivityAccess,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*shipment.ShipmentComment, error) {
	if !readable.comments {
		return nil, nil
	}

	page, err := t.comments.ListByShipmentID(ctx, &shipmentcommentports.ListShipmentCommentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: maxShipmentComments},
		},
		Cursor:     pagination.CursorInfo{Limit: maxShipmentComments},
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("read the shipment's comments: %w", err)
	}
	if page == nil {
		return nil, nil
	}

	return page.Items, nil
}
