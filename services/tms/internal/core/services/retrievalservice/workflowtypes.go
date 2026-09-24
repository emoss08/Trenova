package retrievalservice

import (
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

const (
	IndexOrganizationWorkflowName = "IndexOrganizationWorkflow"
	ReindexSourceWorkflowName     = "ReindexRetrievalSourceWorkflow"
	SweepWorkflowName             = "RetrievalIndexSweepWorkflow"
	IndexSignalName               = "retrieval-index-stale"
	WorkflowPriority              = 4

	indexWorkflowIDPrefix   = "ai-index:"
	reindexWorkflowIDPrefix = "ai-reindex:"
)

type IndexOrganizationInput struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Batches        int      `json:"batches,omitempty"`
}

func (i IndexOrganizationInput) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: i.OrganizationID, BuID: i.BusinessUnitID}
}

type IndexSignal struct {
	SourceType airetrieval.SourceType `json:"sourceType,omitempty"`
	Count      int                    `json:"count,omitempty"`
}

type ReindexSourceInput struct {
	OrganizationID pulid.ID               `json:"organizationId"`
	BusinessUnitID pulid.ID               `json:"businessUnitId"`
	SourceType     airetrieval.SourceType `json:"sourceType"`
	AfterID        pulid.ID               `json:"afterId,omitempty"`
	Marked         int                    `json:"marked,omitempty"`
}

func (i ReindexSourceInput) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: i.OrganizationID, BuID: i.BusinessUnitID}
}

func IndexWorkflowID(orgID pulid.ID) string {
	return indexWorkflowIDPrefix + orgID.String()
}

func ReindexWorkflowID(orgID pulid.ID, sourceType airetrieval.SourceType) string {
	return reindexWorkflowIDPrefix + orgID.String() + ":" + sourceType.String()
}

func WorkflowPriorityFor(orgID pulid.ID) temporal.Priority {
	return temporal.Priority{PriorityKey: WorkflowPriority, FairnessKey: orgID.String()}
}

func indexStartOptions(tenant pagination.TenantInfo) client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		ID:        IndexWorkflowID(tenant.OrgID),
		TaskQueue: temporaltype.TaskQueueSystem.String(),
		Priority:  WorkflowPriorityFor(tenant.OrgID),
		StaticSummary: "Index retrieval sources for organization " +
			tenant.OrgID.String(),
	}
}
