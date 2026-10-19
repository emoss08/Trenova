package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/equipmentmanufacturerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tablechangealertports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxCaptureBatches     = 10
	defaultCaptureBatches = 5
	paramCaptureBatchID   = "captureBatchId"
	listFieldName         = "name"
	listFieldDescription  = "description"
)

func masterDataReadProviders() []any {
	return []any{
		newListEquipmentManufacturersTool,
		newListTableChangeAlertsTool,
		provideListCaptureBatchesTool,
	}
}

func newListEquipmentManufacturersTool(
	repo equipmentmanufacturerports.EquipmentManufacturerRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "equipment_list_manufacturers",
		entityPlural: "equipment manufacturers",
		summary: "List equipment manufacturers — who built the tractors and trailers in " +
			"the fleet, such as Freightliner or Utility. Use it to turn a make a person " +
			"named into the id equipment_create_tractor and equipment_create_trailer need.",
		resource: permission.ResourceEquipmentManufacturer,
		config: querybuilder.GetFieldConfiguration(
			(*equipmentmanufacturer.EquipmentManufacturer)(nil),
		),
		fields: []listField{
			{Name: paramStatus, Kind: filterEnum, Values: statusValues},
			{Name: listFieldName, Kind: filterText, Sortable: true},
			{Name: listFieldDescription, Kind: filterText},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(ctx,
				&equipmentmanufacturerports.ListEquipmentManufacturersRequest{Filter: opts})
			if err != nil {
				return nil, err
			}

			return listRows(
				result.Items,
				func(item *equipmentmanufacturer.EquipmentManufacturer) any {
					return codedRow{
						ID:          item.ID.String(),
						Name:        item.Name,
						Description: item.Description,
						Status:      string(item.Status),
					}
				},
			), nil
		},
	})
}

type tableChangeAlertRow struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	TableName      string   `json:"tableName"`
	EventTypes     []string `json:"eventTypes"`
	WatchedColumns []string `json:"watchedColumns,omitempty"`
	ConditionCount int      `json:"conditionCount"`
	ConditionMatch string   `json:"conditionMatch"`
	Status         string   `json:"status"`
}

func newListTableChangeAlertsTool(
	repo tablechangealertports.TCASubscriptionRepository,
) agentports.AgentQueryTool {
	return newListTool(listSpec{
		name:         "report_list_table_change_alerts",
		entityPlural: "table change alerts",
		summary: "List the change alerts the person you are working for has set up: what " +
			"each watches, for which changes, and whether it is Active or Paused. Only " +
			"their own alerts are returned. Use it to find one to change, pause, resume or " +
			"delete.",
		resource: permission.ResourceTableChangeAlert,
		config:   querybuilder.GetFieldConfiguration((*tablechangealert.TCASubscription)(nil)),
		fields: []listField{
			{
				Name:   paramStatus,
				Kind:   filterEnum,
				Values: tableChangeAlertStatusValues(),
			},
			{Name: listFieldName, Kind: filterText, Sortable: true},
			{Name: "tableName", Kind: filterText},
		},
		fetch: func(ctx context.Context, opts *pagination.QueryOptions) ([]any, error) {
			result, err := repo.List(
				ctx,
				&tablechangealertports.ListTCASubscriptionsRequest{Filter: opts},
			)
			if err != nil {
				return nil, err
			}

			return listRows(result.Items, func(item *tablechangealert.TCASubscription) any {
				return tableChangeAlertRow{
					ID:             item.ID.String(),
					Name:           item.Name,
					TableName:      item.TableName,
					EventTypes:     item.EventTypes,
					WatchedColumns: item.WatchedColumns,
					ConditionCount: len(item.Conditions),
					ConditionMatch: item.ConditionMatch,
					Status:         string(item.Status),
				}
			}), nil
		},
	})
}

func tableChangeAlertStatusValues() []string {
	statuses := tablechangealert.SubscriptionStatusValues()
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}

	return values
}

type captureBatchReader interface {
	ListBatches(
		ctx context.Context,
		in *captureservice.ListBatchesInput,
	) (*pagination.CursorListResult[*capture.CaptureBatch], error)
	GetBatch(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		batchID pulid.ID,
	) (*capture.CaptureBatch, error)
}

type captureItemRow struct {
	ID                   string   `json:"id"`
	Version              int64    `json:"version"`
	Position             int      `json:"position"`
	Status               string   `json:"status"`
	PageCount            int      `json:"pageCount"`
	SuggestedType        string   `json:"suggestedType,omitempty"`
	SuggestedID          string   `json:"suggestedId,omitempty"`
	SuggestedDocTypeID   string   `json:"suggestedDocumentTypeId,omitempty"`
	SuggestionConfidence *float64 `json:"suggestionConfidence,omitempty"`
	SuggestionReason     string   `json:"suggestionReason,omitempty"`
	DetectedKind         string   `json:"detectedKind,omitempty"`
	FiledType            string   `json:"filedType,omitempty"`
	FiledID              string   `json:"filedId,omitempty"`
	FailureMessage       string   `json:"failureMessage,omitempty"`
}

type captureBatchRow struct {
	ID             string           `json:"id"`
	Version        int64            `json:"version"`
	Status         string           `json:"status"`
	Source         string           `json:"source"`
	SourceName     string           `json:"sourceName,omitempty"`
	JobName        string           `json:"jobName,omitempty"`
	CapturedOn     optionalDate     `json:"capturedOn"`
	PageCount      int              `json:"pageCount"`
	ItemCount      int              `json:"itemCount"`
	FiledItemCount int              `json:"filedItemCount"`
	Items          []captureItemRow `json:"items"`
}

func captureItemRowFrom(item *capture.CaptureItem) captureItemRow {
	row := captureItemRow{
		ID:                   item.ID.String(),
		Version:              item.Version,
		Position:             item.Position,
		Status:               string(item.Status),
		PageCount:            len(item.PageIDs),
		SuggestedType:        item.SuggestedType,
		SuggestionConfidence: item.SuggestionConfidence,
		SuggestionReason:     item.SuggestionReason,
		DetectedKind:         item.DetectedKind,
		FiledType:            item.FiledType,
		FailureMessage:       item.FailureMessage,
	}
	if item.SuggestedID != nil {
		row.SuggestedID = item.SuggestedID.String()
	}
	if item.SuggestedDocTypeID != nil {
		row.SuggestedDocTypeID = item.SuggestedDocTypeID.String()
	}
	if item.FiledID != nil {
		row.FiledID = item.FiledID.String()
	}

	return row
}

func captureBatchRowFrom(batch *capture.CaptureBatch) captureBatchRow {
	row := captureBatchRow{
		ID:             batch.ID.String(),
		Version:        batch.Version,
		Status:         string(batch.Status),
		Source:         string(batch.Source),
		SourceName:     batch.SourceName,
		JobName:        batch.JobName,
		CapturedOn:     recordedDate(batch.CreatedAt),
		PageCount:      len(batch.Pages),
		ItemCount:      batch.ItemCount,
		FiledItemCount: batch.FiledItemCount,
		Items:          make([]captureItemRow, 0, len(batch.Items)),
	}
	for _, item := range batch.Items {
		row.Items = append(row.Items, captureItemRowFrom(item))
	}

	return row
}

type listCaptureBatchesTool struct {
	batches captureBatchReader
}

func provideListCaptureBatchesTool(batches *captureservice.Service) agentports.AgentQueryTool {
	return newListCaptureBatchesTool(batches)
}

func newListCaptureBatchesTool(batches captureBatchReader) agentports.AgentQueryTool {
	return &listCaptureBatchesTool{batches: batches}
}

func (t *listCaptureBatchesTool) Name() string { return "inbox_list_capture_batches" }

func (t *listCaptureBatchesTool) SearchTerms() []string {
	return []string{"capture intake", "scanned paperwork waiting", "capture queue"}
}

func (t *listCaptureBatchesTool) Description() string {
	return "List scanned and printed paperwork waiting in capture intake, newest first. " +
		"Each stack comes with the documents it was split into, what each looks like, the " +
		"record it was matched to and how sure the match is. Use it before " +
		"inbox_file_capture_item or discarding a scan; it never shows page images."
}

func (t *listCaptureBatchesTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramCaptureBatchID: agenttoolschema.OfResource(map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "One stack to read, by an id a listing without it returned or " +
					"the stack on the page. Leave it out to list open stacks.",
			}, permission.ResourceCaptureBatch),
			paramLimit: map[string]any{
				toolschema.KeyType: toolschema.TypeInteger,
				toolschema.KeyDescription: fmt.Sprintf("How many stacks to return, at most %d.",
					maxCaptureBatches),
			},
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *listCaptureBatchesTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceCaptureBatch})
}

func (t *listCaptureBatchesTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}
	tenant := tenantOf(params)

	if raw := optionalString(params.Params, paramCaptureBatchID); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("%s %q is not an id", paramCaptureBatchID, raw)
		}
		batch, err := t.batches.GetBatch(ctx, tenant, id)
		if err != nil {
			return nil, err
		}

		return map[string]any{"batches": []captureBatchRow{captureBatchRowFrom(batch)}}, nil
	}

	limit := optionalInt(params.Params, paramLimit, defaultCaptureBatches)
	if limit <= 0 || limit > maxCaptureBatches {
		limit = maxCaptureBatches
	}
	listed, err := t.batches.ListBatches(ctx, &captureservice.ListBatchesInput{
		TenantInfo: tenant,
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: limit},
		},
		Cursor:   pagination.CursorInfo{Limit: limit},
		Statuses: capture.OpenBatchStatuses(),
	})
	if err != nil {
		return nil, err
	}

	rows := make([]captureBatchRow, 0, len(listed.Items))
	for _, listedBatch := range listed.Items {
		batch, getErr := t.batches.GetBatch(ctx, tenant, listedBatch.ID)
		if getErr != nil {
			return nil, getErr
		}
		rows = append(rows, captureBatchRowFrom(batch))
	}

	return map[string]any{"batches": rows}, nil
}
