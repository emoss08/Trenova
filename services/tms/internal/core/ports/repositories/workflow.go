package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
)

type TemplateOptions struct {
	IncludeVersions *bool `form:"includeVersions" json:"includeVersions"`
}

type ListTemplateRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
	TemplateOptions
}

type GetTemplateByIDRequest struct {
	ID              pulid.ID `json:"id"              form:"id"`
	OrgID           pulid.ID `json:"orgId"           form:"orgId"`
	BuID            pulid.ID `json:"buId"            form:"buId"`
	IncludeVersions bool     `json:"includeVersions" form:"includeVersions"`
}

type DeleteTemplateRequest struct {
	ID    pulid.ID `json:"id"`
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
}

type DuplicateTemplateRequest struct {
	TemplateID pulid.ID `json:"templateId"`
	OrgID      pulid.ID `json:"orgId"`
	BuID       pulid.ID `json:"buId"`
	UserID     pulid.ID `json:"userId"`
	NewName    string   `json:"newName"`
}

type ExportTemplateRequest struct {
	TemplateID         pulid.ID  `json:"templateId"`
	OrgID              pulid.ID  `json:"orgId"`
	BuID               pulid.ID  `json:"buId"`
	VersionID          *pulid.ID `json:"versionId"`          // If nil, export published version
	IncludeAllVersions bool      `json:"includeAllVersions"` // If true, export all versions
}

type ImportTemplateRequest struct {
	TemplateData map[string]any `json:"templateData"`
	OrgID        pulid.ID       `json:"orgId"`
	BuID         pulid.ID       `json:"buId"`
	UserID       pulid.ID       `json:"userId"`
}

type TemplateRepository interface {
	List(
		ctx context.Context,
		req *ListTemplateRequest,
	) (*pagination.ListResult[*workflow.Template], error)
	GetByID(ctx context.Context, req *GetTemplateByIDRequest) (*workflow.Template, error)
	Create(
		ctx context.Context,
		entity *workflow.Template,
		userID pulid.ID,
	) (*workflow.Template, error)
	Update(
		ctx context.Context,
		entity *workflow.Template,
		userID pulid.ID,
	) (*workflow.Template, error)
	Delete(ctx context.Context, req *DeleteTemplateRequest) error
	Duplicate(ctx context.Context, req *DuplicateTemplateRequest) (*workflow.Template, error)
	ExportToJSON(ctx context.Context, req *ExportTemplateRequest) (map[string]any, error)
	ImportFromJSON(ctx context.Context, req *ImportTemplateRequest) (*workflow.Template, error)
}

type ListVersionRequest struct {
	Filter     *pagination.QueryOptions `json:"filter"     form:"filter"`
	TemplateID pulid.ID                 `json:"templateId" form:"templateId"`
	OrgID      pulid.ID                 `json:"orgId"      form:"orgId"`
	BuID       pulid.ID                 `json:"buId"       form:"buId"`
	VersionOptions
}

type VersionOptions struct {
	Status             string `form:"status"             json:"status"`
	IncludeNodes       *bool  `form:"includeNodes"       json:"includeNodes"`
	IncludeConnections *bool  `form:"includeConnections" json:"includeConnections"`
}

type GetVersionByIDRequest struct {
	ID                 pulid.ID `json:"id"                 form:"id"`
	OrgID              pulid.ID `json:"orgId"              form:"orgId"`
	BuID               pulid.ID `json:"buId"               form:"buId"`
	IncludeNodes       bool     `json:"includeNodes"       form:"includeNodes"`
	IncludeConnections bool     `json:"includeConnections" form:"includeConnections"`
}

type CreateVersionRequest struct {
	TemplateID         pulid.ID  `json:"templateId"`
	OrgID              pulid.ID  `json:"orgId"`
	BuID               pulid.ID  `json:"buId"`
	UserID             pulid.ID  `json:"userId"`
	CloneFromVersionID *pulid.ID `json:"cloneFromVersionId"` // If nil, create empty version
	ChangeDescription  string    `json:"changeDescription"`
}

type UpdateVersionRequest struct {
	Version *workflow.Version `json:"version"`
	UserID  pulid.ID          `json:"userId"`
}

type DeleteVersionRequest struct {
	ID    pulid.ID `json:"id"`
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
}

type PublishVersionRequest struct {
	VersionID pulid.ID `json:"versionId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
	UserID    pulid.ID `json:"userId"`
}

type ArchiveVersionRequest struct {
	VersionID pulid.ID `json:"versionId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
	UserID    pulid.ID `json:"userId"`
}

type RollbackVersionRequest struct {
	TemplateID      pulid.ID `json:"templateId"`
	TargetVersionID pulid.ID `json:"targetVersionId"`
	OrgID           pulid.ID `json:"orgId"`
	BuID            pulid.ID `json:"buId"`
	UserID          pulid.ID `json:"userId"`
}

type GetPublishedVersionRequest struct {
	TemplateID         pulid.ID `json:"templateId"`
	OrgID              pulid.ID `json:"orgId"`
	BuID               pulid.ID `json:"buId"`
	IncludeNodes       bool     `json:"includeNodes"`
	IncludeConnections bool     `json:"includeConnections"`
}

type CloneConnectionRequest struct {
	SourceConnections []*workflow.Connection `json:"sourceConnections"`
	VersionID         pulid.ID               `json:"versionId"`
	OrgID             pulid.ID               `json:"orgId"`
	BuID              pulid.ID               `json:"buId"`
	NodeIDMap         map[pulid.ID]pulid.ID  `json:"nodeIdMap"`
}

type ImportVersionRequest struct {
	VersionData map[string]any `json:"versionData"`
	TemplateID  pulid.ID       `json:"templateId"`
	OrgID       pulid.ID       `json:"orgId"`
	BuID        pulid.ID       `json:"buId"`
	UserID      pulid.ID       `json:"userId"`
}

type VersionRepository interface {
	List(
		ctx context.Context,
		req *ListVersionRequest,
	) (*pagination.ListResult[*workflow.Version], error)
	GetByID(ctx context.Context, req *GetVersionByIDRequest) (*workflow.Version, error)
	Create(ctx context.Context, req *CreateVersionRequest) (*workflow.Version, error)
	CreateEntity(ctx context.Context, entity *workflow.Version) (*workflow.Version, error)

	Update(ctx context.Context, entity *workflow.Version) (*workflow.Version, error)
	Delete(ctx context.Context, req *DeleteVersionRequest) error
	Publish(ctx context.Context, req *PublishVersionRequest) (*workflow.Version, error)
	Archive(ctx context.Context, req *ArchiveVersionRequest) (*workflow.Version, error)
	Rollback(ctx context.Context, req *RollbackVersionRequest) (*workflow.Version, error)
	GetPublished(ctx context.Context, req *GetPublishedVersionRequest) (*workflow.Version, error)
	ImportVersion(ctx context.Context, req *ImportVersionRequest) error
}

type ListWorkflowInstanceRequest struct {
	Filter *pagination.QueryOptions `json:"filter" form:"filter"`
	WorkflowInstanceOptions
}

type WorkflowInstanceOptions struct {
	WorkflowTemplateID *pulid.ID `form:"workflowTemplateId" json:"workflowTemplateId"`
	WorkflowVersionID  *pulid.ID `form:"workflowVersionId"  json:"workflowVersionId"`
	Status             string    `form:"status"             json:"status"`
	ExecutionMode      string    `form:"executionMode"      json:"executionMode"`
}

type GetWorkflowInstanceByIDRequest struct {
	ID    pulid.ID `json:"id"    form:"id"`
	OrgID pulid.ID `json:"orgId" form:"orgId"`
	BuID  pulid.ID `json:"buId"  form:"buId"`
	WorkflowInstanceOptions
}

type StartWorkflowExecutionRequest struct {
	WorkflowTemplateID pulid.ID       `json:"workflowTemplateId"`
	WorkflowVersionID  *pulid.ID      `json:"workflowVersionId"` // If nil, use published version
	OrgID              pulid.ID       `json:"orgId"`
	BuID               pulid.ID       `json:"buId"`
	UserID             *pulid.ID      `json:"userId"`
	TriggerPayload     map[string]any `json:"triggerPayload"`
	ExecutionMode      string         `json:"executionMode"`
}

type CancelWorkflowInstanceRequest struct {
	InstanceID pulid.ID `json:"instanceId"`
	OrgID      pulid.ID `json:"orgId"`
	BuID       pulid.ID `json:"buId"`
}

type WorkflowInstanceRepository interface {
	List(
		ctx context.Context,
		req *ListWorkflowInstanceRequest,
	) (*pagination.ListResult[*workflow.Instance], error)
	GetByID(ctx context.Context, req *GetWorkflowInstanceByIDRequest) (*workflow.Instance, error)
	Create(ctx context.Context, entity *workflow.Instance) (*workflow.Instance, error)
	Update(ctx context.Context, entity *workflow.Instance) (*workflow.Instance, error)
	GetNodeExecutions(
		ctx context.Context,
		instanceID, orgID, buID pulid.ID,
	) ([]*workflow.NodeExecution, error)
}

type UpdateNodeExecutionStatusRequest struct {
	ExecutionID  pulid.ID       `json:"executionId"`
	OrgID        pulid.ID       `json:"orgId"`
	BuID         pulid.ID       `json:"buId"`
	Status       string         `json:"status"`
	OutputData   map[string]any `json:"outputData"`
	ErrorDetails map[string]any `json:"errorDetails"`
}

type WorkflowNodeExecutionRepository interface {
	Create(ctx context.Context, entity *workflow.NodeExecution) (*workflow.NodeExecution, error)
	Update(ctx context.Context, entity *workflow.NodeExecution) (*workflow.NodeExecution, error)
	GetByInstanceID(
		ctx context.Context,
		instanceID, orgID, buID pulid.ID,
	) ([]*workflow.NodeExecution, error)
}

type DeleteNodeRequest struct {
	NodeID pulid.ID `json:"nodeId"`
	OrgID  pulid.ID `json:"orgId"`
	BuID   pulid.ID `json:"buId"`
}

type GetNodesByVersionIDRequest struct {
	VersionID pulid.ID `json:"versionId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
}

type CloneNodesRequest struct {
	SourceNodes []*workflow.Node `json:"sourceNodes"`
	VersionID   pulid.ID         `json:"versionId"`
	OrgID       pulid.ID         `json:"orgId"`
	BuID        pulid.ID         `json:"buId"`
}

type CloneVersionNodesRequest struct {
	SourceVersion *workflow.Version `json:"sourceVersion"`
	NewVersionID  pulid.ID          `json:"newVersionId"`
	OrgID         pulid.ID          `json:"orgId"`
	BuID          pulid.ID          `json:"buId"`
}

type ImportNodesRequest struct {
	Nodes     []any    `json:"nodes"`
	VersionID pulid.ID `json:"versionId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
}
type WorkflowNodeRepository interface {
	ImportNodes(ctx context.Context, req *ImportNodesRequest) ([]pulid.ID, error)
	GetByVersionID(
		ctx context.Context,
		req *GetNodesByVersionIDRequest,
	) ([]*workflow.Node, error)
	Create(ctx context.Context, entity *workflow.Node) (*workflow.Node, error)
	Update(ctx context.Context, entity *workflow.Node) (*workflow.Node, error)
	Delete(ctx context.Context, req *DeleteNodeRequest) error
	Clone(
		ctx context.Context,
		tx bun.IDB,
		req *CloneNodesRequest,
	) (map[pulid.ID]pulid.ID, error)
	CloneVersionNodes(ctx context.Context, tx bun.IDB, req *CloneVersionNodesRequest) error
}

type GetConnectionsByVersionIDRequest struct {
	VersionID pulid.ID `json:"versionId"`
	OrgID     pulid.ID `json:"orgId"`
	BuID      pulid.ID `json:"buId"`
}

type DeleteConnectionRequest struct {
	ConnectionID pulid.ID `json:"connectionId"`
	OrgID        pulid.ID `json:"orgId"`
	BuID         pulid.ID `json:"buId"`
}

type ImportConnectionsRequest struct {
	Connections []any      `json:"connections"`
	VersionID   pulid.ID   `json:"versionId"`
	OrgID       pulid.ID   `json:"orgId"`
	BuID        pulid.ID   `json:"buId"`
	NodeIDs     []pulid.ID `json:"nodeIds"`
}

type WorkflowConnectionRepository interface {
	ImportConnections(ctx context.Context, req *ImportConnectionsRequest) error
	GetByVersionID(
		ctx context.Context,
		req *GetConnectionsByVersionIDRequest,
	) ([]*workflow.Connection, error)
	Create(ctx context.Context, entity *workflow.Connection) (*workflow.Connection, error)
	Update(ctx context.Context, entity *workflow.Connection) (*workflow.Connection, error)
	Delete(ctx context.Context, req *DeleteConnectionRequest) error
	Clone(ctx context.Context, tx bun.IDB, req *CloneConnectionRequest) error
}
