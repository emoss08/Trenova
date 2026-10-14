package agentprofileports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultHistoryLimit = 20
	MaxHistoryLimit     = 100
	// KeptVersions is how many versions of one profile are kept; older
	// superseded and failed versions are pruned by the nightly pass.
	KeptVersions = 60
)

// SubjectRef names one profile: the organization's, or one person's in it.
// The person is PersonID, never UserID, so reading the tenant off a request
// never mistakes the person a profile is about for the person acting.
type SubjectRef struct {
	OrganizationID pulid.ID             `json:"organizationId"`
	BusinessUnitID pulid.ID             `json:"businessUnitId"`
	Subject        agentprofile.Subject `json:"subject"`
	PersonID       pulid.ID             `json:"personId,omitempty"`
}

func OrganizationSubject(tenant pagination.TenantInfo) SubjectRef {
	return SubjectRef{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Subject:        agentprofile.SubjectOrganization,
	}
}

func UserSubject(tenant pagination.TenantInfo, userID pulid.ID) SubjectRef {
	return SubjectRef{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Subject:        agentprofile.SubjectUser,
		PersonID:       userID,
	}
}

func (r SubjectRef) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: r.OrganizationID, BuID: r.BusinessUnitID}
}

func (r SubjectRef) Valid() bool {
	if r.OrganizationID.IsNil() || r.BusinessUnitID.IsNil() || !r.Subject.IsValid() {
		return false
	}

	return (r.Subject == agentprofile.SubjectUser) == r.PersonID.IsNotNil()
}

func (r SubjectRef) PersonPtr() *pulid.ID {
	if r.Subject != agentprofile.SubjectUser || r.PersonID.IsNil() {
		return nil
	}
	id := r.PersonID

	return &id
}

type ListProfileHistoryRequest struct {
	Ref   SubjectRef
	Limit int
}

type GetProfileRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type PruneProfilesRequest struct {
	Ref  SubjectRef
	Keep int
}

type ProfileRepository interface {
	GetCurrent(ctx context.Context, ref SubjectRef) (*agentprofile.Profile, error)
	GetByID(ctx context.Context, req GetProfileRequest) (*agentprofile.Profile, error)
	ListHistory(
		ctx context.Context,
		req ListProfileHistoryRequest,
	) ([]*agentprofile.Profile, error)
	// Publish makes entity the current version in one transaction, moving the
	// version it replaces to Superseded.
	Publish(ctx context.Context, entity *agentprofile.Profile) (*agentprofile.Profile, error)
	// RecordFailure keeps a version that could not be written; the current one
	// stays what agents are told.
	RecordFailure(
		ctx context.Context,
		entity *agentprofile.Profile,
	) (*agentprofile.Profile, error)
	// MarkChecked notes that the facts were worked out again and gave the same
	// version.
	MarkChecked(ctx context.Context, ref SubjectRef, profileID pulid.ID, at int64) error
	Prune(ctx context.Context, req PruneProfilesRequest) (int, error)
}

// ListGuidanceRequest lists a profile's lines: the active and the offered ones,
// and the closed ones too when IncludeRetired is set.
type ListGuidanceRequest struct {
	Ref            SubjectRef
	IncludeRetired bool
	Limit          int
}

type GetGuidanceRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type GuidanceRepository interface {
	ListActive(ctx context.Context, ref SubjectRef) ([]*agentprofile.Guidance, error)
	List(ctx context.Context, req *ListGuidanceRequest) ([]*agentprofile.Guidance, error)
	CountActive(ctx context.Context, ref SubjectRef) (int, error)
	// CountOpen counts the lines a profile holds open: active and offered.
	CountOpen(ctx context.Context, ref SubjectRef) (int, error)
	GetByID(ctx context.Context, req GetGuidanceRequest) (*agentprofile.Guidance, error)
	Create(ctx context.Context, entity *agentprofile.Guidance) (*agentprofile.Guidance, error)
	Update(ctx context.Context, entity *agentprofile.Guidance) (*agentprofile.Guidance, error)
}
