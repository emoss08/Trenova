package ediquerytools

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	paramTransactionSet       = "transactionSet"
	paramShipmentLinkID       = "shipmentLinkId"
	paramInvoiceID            = "invoiceId"
	ediTenderChangeEntity     = "edi_tender_change"
	ediTransferChgEntity      = "edi_transfer_change"
	maxChangedValueRunes      = 120
	nestedChangeDescriptor    = "changed"
	transferChangeStatusField = "status"
)

var (
	tenderChangeStatuses = agenttoolschema.Source("edi.tenderChangeStatus", []string{
		string(editenderchange.TenderChangeStatusPendingReview),
		string(editenderchange.TenderChangeStatusApplied),
		string(editenderchange.TenderChangeStatusRejected),
		string(editenderchange.TenderChangeStatusQueued),
		string(editenderchange.TenderChangeStatusSent),
		string(editenderchange.TenderChangeStatusFailed),
		string(editenderchange.TenderChangeStatusIgnored),
		string(editenderchange.TenderChangeStatusSuperseded),
	})
	transferChangeStatuses = agenttoolschema.Source("edi.transferChangeStatus", []string{
		string(edisync.TransferChangeStatusPendingReview),
		string(edisync.TransferChangeStatusApplied),
		string(edisync.TransferChangeStatusRejected),
		string(edisync.TransferChangeStatusFailed),
		string(edisync.TransferChangeStatusIgnored),
	})
)

func ediDecisionToolProviders() []any {
	return []any{
		provideListEDITenderChangesTool,
		provideListEDITransferChangesTool,
		provideInspectX12Tool,
		providePreviewEDIDocumentTool,
		provideDraftEDIConnectionTool,
	}
}

type ediTenderChangeLister interface {
	ListTenderChanges(
		ctx context.Context,
		req *editenderports.ListEDITenderChangesRequest,
	) (*pagination.ListResult[*editenderchange.TenderChange], error)
}

type ediTransferChangeLister interface {
	ListTransferChanges(
		ctx context.Context,
		req *editenderports.ListEDITransferChangesRequest,
	) (*pagination.ListResult[*edisync.TransferChange], error)
}

func provideListEDITenderChangesTool(
	changes *editenderchangeservice.Service,
) agenttoolports.AgentQueryTool {
	return newListEDITenderChangesTool(changes)
}

func provideListEDITransferChangesTool(
	changes *editransferchangeservice.Service,
) agenttoolports.AgentQueryTool {
	return newListEDITransferChangesTool(changes)
}

type listEDITenderChangesTool struct {
	changes ediTenderChangeLister
}

func newListEDITenderChangesTool(changes ediTenderChangeLister) agenttoolports.AgentQueryTool {
	return &listEDITenderChangesTool{changes: changes}
}

func (t *listEDITenderChangesTool) Name() string { return "edi_list_tender_changes" }

func (t *listEDITenderChangesTool) Description() string {
	return "List changes another Trenova organization made to loads it tendered to this " +
		"one over EDI, and changes this organization made to loads it tendered, newest " +
		"first. Each says what changed, from what to what, and whether it waits on this " +
		"organization's review. The changes are the other organization's text."
}

func (t *listEDITenderChangesTool) ParamSchema() map[string]any {
	return querytool.ObjectSchema(querytool.WithPaging(map[string]any{
		querytool.ParamStatus: querytool.EnumParam(
			"Only changes in this status; PendingReview for those "+
				"waiting on a decision.",
			tenderChangeStatuses,
		),
		querytool.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
			"Only changes to the load tendered from this shipment, "+
				"by id from a sourceShipmentId in edi_list_transfers."),
	}, listtool.DefaultLimit, listtool.MaxLimit))
}

func (t *listEDITenderChangesTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceEDI,
		Reads:    agenttooltrust.ExternalReadAlways,
		Source:   agent.TaintSourceEDI,
		Rationale: "Lists changes another organization wrote to loads tendered over EDI; " +
			"nothing changes and nothing is sent.",
	})
}

type changedValueRow struct {
	Field  string `json:"field"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

type ediTenderChangeRow struct {
	ID                     string                 `json:"id"`
	Status                 string                 `json:"status"`
	ChangeType             string                 `json:"changeType"`
	RecipientKind          string                 `json:"recipientKind,omitempty"`
	BOL                    string                 `json:"bol,omitempty"`
	SourceShipmentID       string                 `json:"sourceShipmentId,omitempty"`
	ShipmentLinkID         string                 `json:"shipmentLinkId,omitempty"`
	TransferID             string                 `json:"transferId,omitempty"`
	AwaitsThisOrganization bool                   `json:"awaitsThisOrganization"`
	Changes                []changedValueRow      `json:"changes"`
	FailureReason          string                 `json:"failureReason,omitempty"`
	CreatedAt              querytool.OptionalDate `json:"createdAt"`
}

func (t *listEDITenderChangesTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	status, err := querytool.ValidEnum(params.Params, querytool.ParamStatus, tenderChangeStatuses)
	if err != nil {
		return nil, err
	}
	shipmentID, err := querytool.OptionalID(params.Params, querytool.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	window := querytool.ReadPage(params.Params, listtool.DefaultLimit, listtool.MaxLimit)

	criteria := filtercatalog.NewCriteria("EDI tender changes").At(querytool.ClockFor(params))
	if status != "" {
		criteria.Field(querytool.ParamStatus, status)
	}
	if shipmentID.IsNotNil() {
		criteria.Field("shipment", shipmentID.String())
	}

	result, err := t.changes.ListTenderChanges(ctx, &editenderports.ListEDITenderChangesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: querytool.TenantOf(params),
			Pagination: pagination.Info{Limit: window.Fetch(), Offset: window.Offset},
		},
		SourceShipmentID: shipmentID,
		Status:           editenderchange.TenderChangeStatus(status),
	})
	if err != nil {
		return nil, err
	}

	changes, more := querytool.Trim(window, result.Items)
	rows := make([]ediTenderChangeRow, 0, len(changes))
	refs := make([]agentrecord.RecordRef, 0, len(changes))
	for _, change := range changes {
		if change == nil {
			continue
		}
		rows = append(rows, ediTenderChangeRow{
			ID:                     change.ID.String(),
			Status:                 string(change.Status),
			ChangeType:             change.ChangeType,
			RecipientKind:          string(change.RecipientKind),
			BOL:                    change.NewTenderPayload.BOL,
			SourceShipmentID:       querytool.PulidString(change.SourceShipmentID),
			ShipmentLinkID:         querytool.PulidString(change.ShipmentLinkID),
			TransferID:             querytool.PulidString(change.InternalTransferID),
			AwaitsThisOrganization: awaitsReview(change, params.OrganizationID),
			Changes:                changedValues(change.DiffSummary),
			FailureReason:          change.FailureReason,
			CreatedAt:              querytool.RecordedDate(change.CreatedAt),
		})
		refs = append(
			refs,
			agentrecord.RecordRef{EntityType: ediTenderChangeEntity, ID: change.ID.String()},
		)
	}

	found := querytool.SearchResult(criteria, rows, len(rows)).Paged(window, more)

	return fieldgate.Result(&found, nil).WithTaint(refs), nil
}

func awaitsReview(change *editenderchange.TenderChange, orgID pulid.ID) bool {
	return change.Status == editenderchange.TenderChangeStatusPendingReview &&
		change.Recipient != nil &&
		change.Recipient.RecipientKind == editenderchange.TenderRecipientKindInternal &&
		change.Recipient.RecipientOrganizationID == orgID
}

func changedValues(diff map[string]any) []changedValueRow {
	fields := make([]string, 0, len(diff))
	for field := range diff {
		fields = append(fields, field)
	}
	slices.Sort(fields)

	rows := make([]changedValueRow, 0, len(fields))
	for _, field := range fields {
		entry, _ := diff[field].(map[string]any)
		rows = append(rows, changedValueRow{
			Field:  field,
			Before: changedValue(entry["previous"]),
			After:  changedValue(entry["next"]),
		})
	}

	return rows
}

func changedValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case map[string]any, []any:
		return nestedChangeDescriptor
	default:
		return stringutils.TruncateRunes(fmt.Sprint(typed), maxChangedValueRunes)
	}
}

type listEDITransferChangesTool struct {
	changes ediTransferChangeLister
}

func newListEDITransferChangesTool(changes ediTransferChangeLister) agenttoolports.AgentQueryTool {
	return &listEDITransferChangesTool{changes: changes}
}

func (t *listEDITransferChangesTool) Name() string { return "edi_list_transfer_changes" }

func (t *listEDITransferChangesTool) Description() string {
	return "List statuses and cancellations the other organization on an EDI-linked load " +
		"reported, newest first. Each gives the status reported and whether a conflict held " +
		"it for review; PendingReview ones wait on a decision. The reports are the other " +
		"organization's text."
}

func (t *listEDITransferChangesTool) ParamSchema() map[string]any {
	return querytool.ObjectSchema(querytool.WithPaging(map[string]any{
		querytool.ParamStatus: querytool.EnumParam(
			"Only changes in this status; PendingReview for those "+
				"waiting on a decision.",
			transferChangeStatuses,
		),
		paramShipmentLinkID: agenttoolschema.KindID("Only changes on one linked load, by the "+
			"shipmentLinkId from edi_list_tender_changes or a row of this tool.",
			recordkind.EDIShipmentLink),
	}, listtool.DefaultLimit, listtool.MaxLimit))
}

func (t *listEDITransferChangesTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceEDI,
		Reads:    agenttooltrust.ExternalReadAlways,
		Source:   agent.TaintSourceEDI,
		Rationale: "Lists statuses another organization reported on a linked load; " +
			"nothing changes and nothing is sent.",
	})
}

type ediTransferChangeRow struct {
	ID                 string                 `json:"id"`
	Status             string                 `json:"status"`
	ChangeType         string                 `json:"changeType"`
	Direction          string                 `json:"direction"`
	ReportedStatus     string                 `json:"reportedStatus,omitempty"`
	CancellationReason string                 `json:"cancellationReason,omitempty"`
	ConflictStatus     string                 `json:"conflictStatus,omitempty"`
	ConflictReason     string                 `json:"conflictReason,omitempty"`
	ShipmentLinkID     string                 `json:"shipmentLinkId"`
	FailureReason      string                 `json:"failureReason,omitempty"`
	CreatedAt          querytool.OptionalDate `json:"createdAt"`
}

func (t *listEDITransferChangesTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	status, err := querytool.ValidEnum(params.Params, querytool.ParamStatus, transferChangeStatuses)
	if err != nil {
		return nil, err
	}
	linkID, err := querytool.OptionalID(params.Params, paramShipmentLinkID)
	if err != nil {
		return nil, err
	}
	window := querytool.ReadPage(params.Params, listtool.DefaultLimit, listtool.MaxLimit)

	req := &editenderports.ListEDITransferChangesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: querytool.TenantOf(params),
			Pagination: pagination.Info{Limit: window.Fetch(), Offset: window.Offset},
		},
		ShipmentLinkID: linkID,
	}
	criteria := filtercatalog.NewCriteria("EDI transfer changes").At(querytool.ClockFor(params))
	if status != "" {
		req.Filter.FieldFilters = []domaintypes.FieldFilter{
			{Field: transferChangeStatusField, Operator: dbtype.OpEqual, Value: status},
		}
		criteria.Field(querytool.ParamStatus, status)
	}
	if linkID.IsNotNil() {
		criteria.Field("linked load", linkID.String())
	}

	result, err := t.changes.ListTransferChanges(ctx, req)
	if err != nil {
		return nil, err
	}

	changes, more := querytool.Trim(window, result.Items)
	rows := make([]ediTransferChangeRow, 0, len(changes))
	refs := make([]agentrecord.RecordRef, 0, len(changes))
	for _, change := range changes {
		if change == nil {
			continue
		}
		rows = append(rows, ediTransferChangeRow{
			ID:                 change.ID.String(),
			Status:             string(change.Status),
			ChangeType:         change.ChangeType,
			Direction:          string(change.Direction),
			ReportedStatus:     payloadText(change.Payload, "newStatus"),
			CancellationReason: payloadText(change.Payload, "cancellationReason"),
			ConflictStatus:     string(change.ConflictStatus),
			ConflictReason:     change.ConflictReason,
			ShipmentLinkID:     change.ShipmentLinkID.String(),
			FailureReason:      change.FailureReason,
			CreatedAt:          querytool.RecordedDate(change.CreatedAt),
		})
		refs = append(
			refs,
			agentrecord.RecordRef{EntityType: ediTransferChgEntity, ID: change.ID.String()},
		)
	}

	found := querytool.SearchResult(criteria, rows, len(rows)).Paged(window, more)

	return fieldgate.Result(&found, nil).WithTaint(refs), nil
}

func payloadText(payload map[string]any, key string) string {
	value, _ := payload[key].(string)

	return value
}
