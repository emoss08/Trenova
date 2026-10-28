package ladingports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetLadingProjectRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
	PartnerID  pulid.ID              `json:"partnerId"`
}

type ListLadingProjectsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type CreateLadingProjectRequest struct {
	Project *ladingproject.LadingProject
	Draft   *ladingproject.LadingVersion
	Samples []*ladingproject.LadingSample
}

type GetLadingVersionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
}

type ListLadingVersionsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ProjectID  pulid.ID              `json:"projectId"`
	Limit      int                   `json:"limit"`
}

type GetLadingSamplesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ProjectID  pulid.ID              `json:"projectId"`
	Digests    []string              `json:"digests"`
}

type ActivateLadingVersionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ProjectID  pulid.ID              `json:"projectId"`
	VersionID  pulid.ID              `json:"versionId"`
	ActorID    pulid.ID              `json:"actorId"`
}

type ActivatedLadingVersion struct {
	Version    *ladingproject.LadingVersion
	Superseded *ladingproject.LadingVersion
}

type GetRunnableLadingVersionRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	PartnerID  pulid.ID              `json:"partnerId"`
}

type RunnableLadingVersions struct {
	Project *ladingproject.LadingProject
	Active  *ladingproject.LadingVersion
	Shadow  *ladingproject.LadingVersion
}

type LadingProjectRepository interface {
	GetProject(
		ctx context.Context,
		req GetLadingProjectRequest,
	) (*ladingproject.LadingProject, error)
	ListProjects(
		ctx context.Context,
		req *ListLadingProjectsRequest,
	) (*pagination.ListResult[*ladingproject.LadingProject], error)
	CreateProject(
		ctx context.Context,
		req *CreateLadingProjectRequest,
	) (*ladingproject.LadingProject, error)
	UpdateProjectSettings(
		ctx context.Context,
		project *ladingproject.LadingProject,
	) (*ladingproject.LadingProject, error)
	SetShadowVersion(
		ctx context.Context,
		project *ladingproject.LadingProject,
	) (*ladingproject.LadingProject, error)
	GetVersion(
		ctx context.Context,
		req GetLadingVersionRequest,
	) (*ladingproject.LadingVersion, error)
	ListVersions(
		ctx context.Context,
		req ListLadingVersionsRequest,
	) ([]*ladingproject.LadingVersion, error)
	CreateVersion(
		ctx context.Context,
		version *ladingproject.LadingVersion,
		samples []*ladingproject.LadingSample,
	) (*ladingproject.LadingVersion, error)
	SaveDraft(
		ctx context.Context,
		version *ladingproject.LadingVersion,
		samples []*ladingproject.LadingSample,
	) (*ladingproject.LadingVersion, error)
	UpdateVersionStatus(
		ctx context.Context,
		version *ladingproject.LadingVersion,
	) (*ladingproject.LadingVersion, error)
	ActivateVersion(
		ctx context.Context,
		req ActivateLadingVersionRequest,
	) (*ActivatedLadingVersion, error)
	GetSamples(
		ctx context.Context,
		req GetLadingSamplesRequest,
	) ([]*ladingproject.LadingSample, error)
	GetRunnableVersions(
		ctx context.Context,
		req GetRunnableLadingVersionRequest,
	) (*RunnableLadingVersions, error)
}

type GetFlowInstanceRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
}

type GetFlowInstanceByKeyRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	PartnerID  pulid.ID              `json:"partnerId"`
	Flow       string                `json:"flow"`
	Key        string                `json:"key"`
}

type ListFlowInstancesRequest struct {
	Filter     *pagination.QueryOptions           `json:"filter"`
	Cursor     pagination.CursorInfo              `json:"-"`
	PartnerID  pulid.ID                           `json:"partnerId"`
	ShipmentID pulid.ID                           `json:"shipmentId"`
	TransferID pulid.ID                           `json:"transferId"`
	Statuses   []ladingproject.FlowInstanceStatus `json:"statuses"`
}

type ListShipmentFlowInstancesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"shipmentId"`
	TransferID pulid.ID              `json:"transferId"`
}

type ListFlowEventsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InstanceID pulid.ID              `json:"instanceId"`
	Limit      int                   `json:"limit"`
}

type GetFlowSendMessageRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	InstanceID pulid.ID              `json:"instanceId"`
	SendID     string                `json:"sendId"`
}

type FlowRepository interface {
	GetInstance(
		ctx context.Context,
		req GetFlowInstanceRequest,
	) (*ladingproject.FlowInstance, error)
	GetInstanceByKey(
		ctx context.Context,
		req GetFlowInstanceByKeyRequest,
	) (*ladingproject.FlowInstance, error)
	OpenInstance(
		ctx context.Context,
		instance *ladingproject.FlowInstance,
	) (*ladingproject.FlowInstance, error)
	SaveInstance(
		ctx context.Context,
		instance *ladingproject.FlowInstance,
	) (*ladingproject.FlowInstance, error)
	AppendEvent(ctx context.Context, event *ladingproject.FlowEvent) error
	ListOpenInstances(
		ctx context.Context,
		req *ListShipmentFlowInstancesRequest,
	) ([]*ladingproject.FlowInstance, error)
	ListInstancesCursor(
		ctx context.Context,
		req *ListFlowInstancesRequest,
	) (*pagination.CursorListResult[*ladingproject.FlowInstance], error)
	ListInstances(
		ctx context.Context,
		req *ListFlowInstancesRequest,
	) (*pagination.ListResult[*ladingproject.FlowInstance], error)
	ListEvents(ctx context.Context, req ListFlowEventsRequest) ([]*ladingproject.FlowEvent, error)
	GetSendMessage(
		ctx context.Context,
		req GetFlowSendMessageRequest,
	) (*edimessage.EDIMessage, error)
}

type GetLadingReplayRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ID         pulid.ID              `json:"id"`
}

type ListLadingReplaysRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	VersionID  pulid.ID              `json:"versionId"`
	Limit      int                   `json:"limit"`
}

type ListLadingShadowResultsRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	VersionID     pulid.ID              `json:"versionId"`
	OnlyDiffering bool                  `json:"onlyDiffering"`
	Limit         int                   `json:"limit"`
}

type LadingShadowSummary struct {
	Total  int64 `json:"total"`
	Same   int64 `json:"same"`
	Failed int64 `json:"failed"`
}

type LadingReplayRepository interface {
	CreateReplay(
		ctx context.Context,
		replay *ladingproject.LadingReplay,
	) (*ladingproject.LadingReplay, error)
	GetReplay(ctx context.Context, req GetLadingReplayRequest) (*ladingproject.LadingReplay, error)
	ListReplays(
		ctx context.Context,
		req ListLadingReplaysRequest,
	) ([]*ladingproject.LadingReplay, error)
	FinishReplay(
		ctx context.Context,
		replay *ladingproject.LadingReplay,
	) (*ladingproject.LadingReplay, error)
	RecordShadowResult(ctx context.Context, result *ladingproject.LadingShadowResult) error
	ListShadowResults(
		ctx context.Context,
		req ListLadingShadowResultsRequest,
	) ([]*ladingproject.LadingShadowResult, error)
	SummarizeShadow(
		ctx context.Context,
		req ListLadingShadowResultsRequest,
	) (*LadingShadowSummary, error)
}

type ScheduleFlowTimerRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	InstanceID pulid.ID              `json:"instanceId"`
	WakeAt     int64                 `json:"wakeAt"`
}

type FlowTimerScheduler interface {
	Schedule(ctx context.Context, req *ScheduleFlowTimerRequest) error
}
