package retrievalservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/airetrieval"
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
	indexSweepIDPrefix      = "ai-index-sweep:"
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

type SweepRequest struct {
	Organization *pagination.TenantInfo `json:"organization,omitempty"`
}

func SweepWorkflowIDFor(orgID pulid.ID) string {
	return indexSweepIDPrefix + orgID.String()
}

func IndexWorkflowID(orgID pulid.ID) string {
	return indexWorkflowIDPrefix + orgID.String()
}

func ReindexWorkflowID(orgID pulid.ID, sourceType airetrieval.SourceType) string {
	return reindexWorkflowIDPrefix + orgID.String() + ":" + sourceType.String()
}

func IndexPriority() temporal.Priority {
	return temporal.Priority{PriorityKey: WorkflowPriority}
}

func indexStartOptions(tenant pagination.TenantInfo) client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		ID:        IndexWorkflowID(tenant.OrgID),
		TaskQueue: temporaltype.TaskQueueSystem.String(),
		Priority:  IndexPriority(),
		StaticSummary: "Index retrieval sources for organization " +
			tenant.OrgID.String(),
	}
}
