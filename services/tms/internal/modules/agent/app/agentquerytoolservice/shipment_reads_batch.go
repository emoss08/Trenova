package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramShipmentIDs = "shipmentIds"
	maxShipmentReads = 25
)

var errShipmentRefMissing = errors.New("give shipmentIds or proNumber")

func shipmentRefsProperty() map[string]any {
	return agenttoolschema.RecordIDs(permission.ResourceShipment,
		"The shipments' ids, from shipment_list, dispatch_get_board, the page you are on or "+
			"this run's subject: every load you need, in one call. Give these or proNumber.",
		maxShipmentReads)
}

func shipmentRefs(params map[string]any) ([]pulid.ID, string, error) {
	ids, err := optionalPulidSlice(params, paramShipmentIDs, maxShipmentReads)
	if err != nil {
		return nil, "", err
	}
	pro := strings.TrimSpace(optionalString(params, paramProNumber))
	switch {
	case len(ids) > 0 && pro != "":
		return nil, "", fmt.Errorf("give %s or %s, not both", paramShipmentIDs, paramProNumber)
	case len(ids) == 0 && pro == "":
		return nil, "", errShipmentRefMissing
	}

	return sliceutils.Dedupe(ids), pro, nil
}

func loadShipments(
	ctx context.Context,
	shipments shipmentports.ShipmentRepository,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
	options shipmentports.ShipmentOptions,
) ([]*shipment.Shipment, []string, error) {
	options.IDs = ids
	result, err := shipments.List(ctx, &shipmentports.ListShipmentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: len(ids)},
		},
		Cursor:          pagination.CursorInfo{Limit: len(ids)},
		ShipmentOptions: options,
	})
	if err != nil {
		return nil, nil, err
	}

	byID := make(map[pulid.ID]*shipment.Shipment, len(ids))
	if result != nil {
		for _, item := range result.Items {
			if item != nil {
				byID[item.ID] = item
			}
		}
	}
	found := make([]*shipment.Shipment, 0, len(ids))
	var missing []string
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			found = append(found, item)
			continue
		}
		missing = append(missing, id.String())
	}

	return found, missing, nil
}

func missingShipmentsNote(missing []string) string {
	if len(missing) == 0 {
		return ""
	}

	return "No shipment of this organization has these ids: " + strings.Join(missing, ", ") + "."
}

var shipmentReadColumns = []string{
	"proNumber", "status", "customer", "pickup", "delivery", "drivers", "carrying",
}

type shipmentReadRow struct {
	ID            string              `json:"id"`
	ProNumber     string              `json:"proNumber"`
	BOL           string              `json:"bol,omitempty"`
	Status        string              `json:"status"`
	Customer      string              `json:"customer,omitempty"`
	Pickup        string              `json:"pickup,omitempty"`
	Delivery      string              `json:"delivery,omitempty"`
	Drivers       string              `json:"drivers,omitempty"`
	Carrying      string              `json:"carrying,omitempty"`
	TotalCharge   string              `json:"totalCharge,omitempty"`
	BillingStatus string              `json:"billingStatus,omitempty"`
	ActiveHolds   []shipmentHoldRow   `json:"activeHolds,omitempty"`
	NewestComment *shipmentCommentRow `json:"newestComment,omitempty"`
}

type shipmentReadsOutcome struct {
	Count            int               `json:"count"`
	SearchedFor      []string          `json:"searchedFor"`
	Columns          []string          `json:"columns"`
	Items            []shipmentReadRow `json:"items"`
	Note             string            `json:"note,omitempty"`
	CommentsNote     string            `json:"commentsNote,omitempty"`
	LeftOutForAccess string            `json:"leftOutForAccess,omitempty"`

	outside []agent.RecordRef
}

func (o *shipmentReadsOutcome) TaintedRecords() []agent.RecordRef { return o.outside }

func (t *getShipmentTool) withBatchReads(
	holds shipmentports.ActiveHoldReader,
	comments shipmentports.NewestCommentReader,
) *getShipmentTool {
	t.batchHolds = holds
	t.batchComments = comments

	return t
}

func (t *getShipmentTool) several(
	ctx context.Context,
	params *agentports.QueryToolParams,
	ids []pulid.ID,
	detail shipmentDetail,
) (any, error) {
	if detail == shipmentDetailFull {
		return nil, fmt.Errorf("detail full reads one shipment; name one id for it")
	}

	entities, missing, err := loadShipments(ctx, t.repo, tenantOf(params), ids,
		shipmentports.ShipmentOptions{
			IncludeCustomer:    true,
			IncludeRoute:       true,
			IncludeCommodities: true,
		})
	if err != nil {
		return nil, err
	}

	readable := t.readableBatch(ctx, params)
	found := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		found = append(found, entity.ID)
	}
	holds, err := t.batchActiveHolds(ctx, readable, found)
	if err != nil {
		return nil, err
	}
	comments, err := t.batchNewestComments(ctx, readable, found)
	if err != nil {
		return nil, err
	}

	outcome := &shipmentReadsOutcome{
		Count:            len(entities),
		SearchedFor:      []string{"shipments by id"},
		Columns:          shipmentReadColumns,
		Items:            make([]shipmentReadRow, 0, len(entities)),
		Note:             missingShipmentsNote(missing),
		LeftOutForAccess: readable.leftOutNote(params),
	}
	for _, entity := range entities {
		row := shipmentReadRowOf(entity)
		row.ActiveHolds = holds[entity.ID]
		if comment, ok := comments[entity.ID]; ok {
			commentRow, outside := commentRowOf(comment)
			row.NewestComment = &commentRow
			if outside {
				outcome.outside = append(outcome.outside, agent.RecordRef{
					EntityType: agent.TaintEntityShipmentComment,
					ID:         comment.ID.String(),
				})
			}
		}
		outcome.Items = append(outcome.Items, row)
	}
	if len(outcome.outside) > 0 {
		outcome.CommentsNote = outsideCommentsNote
	}

	return outcome, nil
}

func shipmentReadRowOf(entity *shipment.Shipment) shipmentReadRow {
	listed := toShipmentRow(entity)

	return shipmentReadRow{
		ID:            listed.ID,
		ProNumber:     listed.ProNumber,
		BOL:           listed.BOL,
		Status:        listed.Status,
		Customer:      listed.Customer,
		Pickup:        listed.Pickup,
		Delivery:      listed.Delivery,
		Drivers:       listed.Drivers,
		Carrying:      listed.Carrying,
		TotalCharge:   listed.TotalCharge,
		BillingStatus: listed.BillingStatus,
	}
}

func (t *getShipmentTool) readableBatch(
	ctx context.Context,
	params *agentports.QueryToolParams,
) shipmentActivityAccess {
	var access shipmentActivityAccess
	if t.batchComments != nil {
		access.comments = t.access.mayRead(ctx, params, permission.ResourceShipmentComment)
		access.commentsWithheld = !access.comments
	}
	if t.batchHolds != nil {
		access.holds = t.access.mayRead(ctx, params, permission.ResourceShipmentHold)
		access.holdsWithheld = !access.holds
	}

	return access
}

func (t *getShipmentTool) batchActiveHolds(
	ctx context.Context,
	readable shipmentActivityAccess,
	ids []pulid.ID,
) (map[pulid.ID][]shipmentHoldRow, error) {
	out := make(map[pulid.ID][]shipmentHoldRow, len(ids))
	if !readable.holds || len(ids) == 0 {
		return out, nil
	}

	holds, err := t.batchHolds.ActiveHolds(ctx, &shipmentports.ActiveHoldsRequest{ShipmentIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("read the holds on these shipments: %w", err)
	}
	for _, hold := range holds {
		if hold != nil {
			out[hold.ShipmentID] = append(out[hold.ShipmentID], holdRowOf(hold))
		}
	}

	return out, nil
}

func (t *getShipmentTool) batchNewestComments(
	ctx context.Context,
	readable shipmentActivityAccess,
	ids []pulid.ID,
) (map[pulid.ID]*shipment.ShipmentComment, error) {
	out := make(map[pulid.ID]*shipment.ShipmentComment, len(ids))
	if !readable.comments || len(ids) == 0 {
		return out, nil
	}

	comments, err := t.batchComments.NewestComments(ctx,
		&shipmentports.NewestCommentsRequest{ShipmentIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("read the newest comments on these shipments: %w", err)
	}
	for _, comment := range comments {
		if comment != nil && !comment.IsDeleted() {
			out[comment.ShipmentID] = comment
		}
	}

	return out, nil
}

var trackingColumns = []string{"proNumber", "status", "customer", "summary"}

type trackingRow struct {
	ID        string                             `json:"id"`
	ProNumber string                             `json:"proNumber"`
	Status    string                             `json:"status"`
	Customer  string                             `json:"customer,omitempty"`
	Summary   string                             `json:"summary"`
	NextStop  *shipmenttracking.StopSnapshot     `json:"nextStop,omitempty"`
	Estimate  *shipmenttracking.ArrivalEstimate  `json:"estimate,omitempty"`
	Driver    *shipmenttracking.DriverSnapshot   `json:"driver,omitempty"`
	Position  *shipmenttracking.PositionSnapshot `json:"position,omitempty"`
	Flags     []string                           `json:"flags"`
	Canceled  *shipmenttracking.CanceledSnapshot `json:"canceled,omitempty"`
}

type trackingOutcome struct {
	Count       int           `json:"count"`
	SearchedFor []string      `json:"searchedFor"`
	Columns     []string      `json:"columns"`
	Items       []trackingRow `json:"items"`
	Note        string        `json:"note,omitempty"`
}

func trackingTable(snapshots []*shipmenttracking.Snapshot, missing []string) *trackingOutcome {
	outcome := &trackingOutcome{
		Count:       len(snapshots),
		SearchedFor: []string{"shipments by id"},
		Columns:     trackingColumns,
		Items:       make([]trackingRow, 0, len(snapshots)),
		Note:        missingShipmentsNote(missing),
	}
	for _, snap := range snapshots {
		outcome.Items = append(outcome.Items, trackingRow{
			ID:        snap.ShipmentID,
			ProNumber: snap.ProNumber,
			Status:    snap.Status,
			Customer:  snap.Customer,
			Summary:   snap.Summary,
			NextStop:  snap.NextStop,
			Estimate:  snap.Estimate,
			Driver:    snap.Driver,
			Position:  snap.Position,
			Flags:     snap.Flags,
			Canceled:  snap.Canceled,
		})
	}

	return outcome
}
