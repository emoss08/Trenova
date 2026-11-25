package shipmentreadquerytools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/shipmentquerytools"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
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
	ids, err := querytool.OptionalPulidSlice(params, paramShipmentIDs, maxShipmentReads)
	if err != nil {
		return nil, "", err
	}
	pro := strings.TrimSpace(querytool.OptionalString(params, listtool.ParamProNumber))
	switch {
	case len(ids) > 0 && pro != "":
		return nil, "", fmt.Errorf(
			"give %s or %s, not both",
			paramShipmentIDs,
			listtool.ParamProNumber,
		)
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

const batchCommentsPerShipment = 5

type shipmentRecordsOutcome struct {
	Count            int    `json:"count"`
	Items            []any  `json:"items"`
	Note             string `json:"note,omitempty"`
	CommentsNote     string `json:"commentsNote,omitempty"`
	LeftOutForAccess string `json:"leftOutForAccess,omitempty"`

	outside []agentrecord.RecordRef
}

func (o *shipmentRecordsOutcome) TaintedRecords() []agentrecord.RecordRef { return o.outside }

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
	params *agenttoolports.QueryToolParams,
	ids []pulid.ID,
	detail shipmentquerytools.Detail,
) (any, error) {
	entities, missing, err := loadShipments(ctx, t.repo, querytool.TenantOf(params), ids,
		shipmentports.ShipmentOptions{ExpandShipmentDetails: true, IncludeCustomer: true})
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
	comments, err := t.newestComments(ctx, readable, found)
	if err != nil {
		return nil, err
	}

	people := t.access.Redactor(ctx, params)
	outcome := &shipmentRecordsOutcome{
		Count:            len(entities),
		Items:            make([]any, 0, len(entities)),
		Note:             missingShipmentsNote(missing),
		LeftOutForAccess: readable.leftOutNote(params),
	}
	for _, entity := range entities {
		var newest []*shipment.ShipmentComment
		total := 0
		if digest, ok := comments[entity.ID]; ok {
			newest, total = digest.Newest, digest.Total
		}
		activity := newShipmentActivity(newest, holds[entity.ID])
		activity.countComments(total)
		activity.noteEntry(entity)
		outcome.outside = append(outcome.outside, activity.outside...)
		if activity.CommentsNote != "" {
			outcome.CommentsNote = activity.CommentsNote
			activity.CommentsNote = ""
		}
		if detail == shipmentquerytools.DetailFull {
			outcome.Items = append(outcome.Items, newShipmentView(entity, activity))
			continue
		}
		outcome.Items = append(outcome.Items, summarizeShipment(&shipmentSummaryInput{
			entity:   entity,
			activity: activity,
			timezone: params.Timezone,
			people:   people,
		}))
	}

	if detail == shipmentquerytools.DetailFull {
		return people.Withhold(outcome)
	}

	return people.Annotate(outcome)
}

func (t *getShipmentTool) readableBatch(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) shipmentActivityAccess {
	var access shipmentActivityAccess
	if t.batchComments != nil {
		access.comments = t.access.MayRead(ctx, params, permission.ResourceShipmentComment)
		access.commentsWithheld = !access.comments
	}
	if t.batchHolds != nil {
		access.holds = t.access.MayRead(ctx, params, permission.ResourceShipmentHold)
		access.holdsWithheld = !access.holds
	}

	return access
}

func (t *getShipmentTool) batchActiveHolds(
	ctx context.Context,
	readable shipmentActivityAccess,
	ids []pulid.ID,
) (map[pulid.ID][]*shipmenthold.ShipmentHold, error) {
	out := make(map[pulid.ID][]*shipmenthold.ShipmentHold, len(ids))
	if !readable.holds || len(ids) == 0 {
		return out, nil
	}

	holds, err := t.batchHolds.ActiveHolds(ctx, &shipmentports.ActiveHoldsRequest{ShipmentIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("read the holds on these shipments: %w", err)
	}
	for _, hold := range holds {
		if hold != nil {
			out[hold.ShipmentID] = append(out[hold.ShipmentID], hold)
		}
	}

	return out, nil
}

func (t *getShipmentTool) newestComments(
	ctx context.Context,
	readable shipmentActivityAccess,
	ids []pulid.ID,
) (map[pulid.ID]*shipmentports.ShipmentCommentDigest, error) {
	if !readable.comments || len(ids) == 0 {
		return map[pulid.ID]*shipmentports.ShipmentCommentDigest{}, nil
	}

	digests, err := t.batchComments.NewestComments(ctx, &shipmentports.NewestCommentsRequest{
		ShipmentIDs: ids,
		PerShipment: batchCommentsPerShipment,
	})
	if err != nil {
		return nil, fmt.Errorf("read the newest comments on these shipments: %w", err)
	}

	return digests, nil
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
