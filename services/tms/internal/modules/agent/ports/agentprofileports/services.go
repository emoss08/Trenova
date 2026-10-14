package agentprofileports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// RealtimeResource is the resource a change to a profile or its guidance is
// announced on, so AI Control and the Desk refresh what they show.
const RealtimeResource = "agent_profile"

// SkipReason says why a pass wrote no new version.
type SkipReason string

const (
	// SkipUnchanged: the facts and guidance give the version already current.
	SkipUnchanged = SkipReason("Unchanged")
	// SkipTooSoon: a scheduled pass came before the profile is due again.
	SkipTooSoon = SkipReason("TooSoon")
	// SkipLearningOff: the organization turned learning off, or agents are in
	// shadow mode.
	SkipLearningOff = SkipReason("LearningOff")
	// SkipNotAllowed: the organization's plan does not include agent automation.
	SkipNotAllowed = SkipReason("NotAllowed")
	// SkipNoActivity: a person's records show nothing to write about yet.
	SkipNoActivity = SkipReason("NoActivity")
)

type GenerateRequest struct {
	Ref           SubjectRef           `json:"ref"`
	Trigger       agentprofile.Trigger `json:"trigger"`
	RequestedByID pulid.ID             `json:"requestedById,omitempty"`
}

// GenerationPlan is what Prepare worked out: the facts, the guidance and, when
// a new version is due, the request that writes it. A worker runs the request
// in its own activity and hands the reply to Finish.
type GenerationPlan struct {
	Ref           SubjectRef           `json:"ref"`
	Trigger       agentprofile.Trigger `json:"trigger"`
	RequestedByID pulid.ID             `json:"requestedById,omitempty"`
	Skip          SkipReason           `json:"skip,omitempty"`
	Facts         agentprofile.Facts   `json:"facts"`
	GuidanceIDs   []pulid.ID           `json:"guidanceIds"`
	Digest        string               `json:"digest"`
	CurrentID     pulid.ID             `json:"currentId,omitempty"`
	// VersionID is the id the version will be kept under, chosen when the
	// plan is made, so a Finish run twice keeps one version.
	VersionID        pulid.ID                                `json:"versionId,omitempty"`
	OrganizationName string                                  `json:"organizationName,omitempty"`
	Request          *agentports.StructuredCompletionRequest `json:"request,omitempty"`
}

func (p *GenerationPlan) Ready() bool {
	return p != nil && p.Skip == "" && p.Request != nil
}

type FinishGenerationRequest struct {
	Plan   *GenerationPlan                        `json:"plan"`
	Result *agentports.StructuredCompletionResult `json:"result,omitempty"`
	Error  string                                 `json:"error,omitempty"`
}

type GenerationOutcome struct {
	ProfileID pulid.ID            `json:"profileId,omitempty"`
	Status    agentprofile.Status `json:"status,omitempty"`
	Skip      SkipReason          `json:"skip,omitempty"`
}

type AddGuidanceRequest struct {
	Ref      SubjectRef
	Content  string
	Source   agentprofile.GuidanceSource
	ThreadID pulid.ID
	Actor    *permissionports.RequestActor
}

type RetireGuidanceRequest struct {
	Ref   SubjectRef
	ID    pulid.ID
	Actor *permissionports.RequestActor
}

// DecideGuidanceRequest approves or dismisses a line offered about the
// organization; the actor is the administrator deciding.
type DecideGuidanceRequest struct {
	Ref   SubjectRef
	ID    pulid.ID
	Actor *permissionports.RequestActor
}

type RegenerateRequest struct {
	Ref   SubjectRef
	Actor *permissionports.RequestActor
}

// PromptProfiles is what a turn is told: the organization's profile and the
// profile of the person it is for. Either may be empty.
type PromptProfiles struct {
	Organization          string
	OrganizationCheckedAt int64
	User                  string
	UserCheckedAt         int64
}

type PromptProfilesRequest struct {
	TenantInfo   pagination.TenantInfo
	UserID       pulid.ID
	Organization bool
}

// ProfileView is a profile as AI Control and the Desk show it: the current
// version, if any, with the guidance it honours and the newest attempt that
// did not become current.
type ProfileView struct {
	Ref           SubjectRef
	Current       *agentprofile.Profile
	LatestFailure *agentprofile.Profile
	Guidance      []*agentprofile.Guidance
}

type Service interface {
	View(ctx context.Context, ref SubjectRef) (*ProfileView, error)
	History(ctx context.Context, req ListProfileHistoryRequest) ([]*agentprofile.Profile, error)
	ListGuidance(ctx context.Context, req *ListGuidanceRequest) ([]*agentprofile.Guidance, error)
	// AddGuidance keeps a line, or offers it: a line about the organization
	// from a person who does not manage shared memory, or written down by an
	// agent for such a person, waits as Suggested for an administrator.
	AddGuidance(ctx context.Context, req *AddGuidanceRequest) (*agentprofile.Guidance, error)
	// PlanGuidance is the line AddGuidance would create, with the status it
	// would be given, after the same checks and without writing it.
	PlanGuidance(ctx context.Context, req *AddGuidanceRequest) (*agentprofile.Guidance, error)
	RetireGuidance(ctx context.Context, req *RetireGuidanceRequest) (*agentprofile.Guidance, error)
	ApproveGuidance(ctx context.Context, req *DecideGuidanceRequest) (*agentprofile.Guidance, error)
	DismissGuidance(ctx context.Context, req *DecideGuidanceRequest) (*agentprofile.Guidance, error)
	Regenerate(ctx context.Context, req *RegenerateRequest) error
	ForPrompt(ctx context.Context, req PromptProfilesRequest) (PromptProfiles, error)
}

// Generator is the worker's side: Prepare works out the facts and decides
// whether a version is due; Finish keeps what the model wrote.
type Generator interface {
	Prepare(ctx context.Context, req *GenerateRequest) (*GenerationPlan, error)
	Finish(ctx context.Context, req *FinishGenerationRequest) (*GenerationOutcome, error)
	Prune(ctx context.Context, ref SubjectRef) error
}

// Scheduler asks a worker to write a profile again: once a person asks, or a
// little after guidance changes so several lines given together cost one
// write.
type Scheduler interface {
	Request(ctx context.Context, req *GenerateRequest) error
}

// Reader is the narrow view a turn's context builder needs.
type Reader interface {
	ForPrompt(ctx context.Context, req PromptProfilesRequest) (PromptProfiles, error)
}
