package caseownerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecordRequest struct {
	TenantInfo pagination.TenantInfo
	RecordID   pulid.ID
}

type SetOwnerRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
	OwnerID    pulid.ID
	Source     caseowner.Source
	ByID       pulid.ID
	Now        int64
	Label      string
	// Expected is the owner the caller saw, nil when it saw none. The write is
	// refused when someone else changed it in between.
	Expected *caseowner.Owner
}

type CandidatesRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
}

type RecordsRequest struct {
	TenantInfo pagination.TenantInfo
	RecordIDs  []pulid.ID
}

type AwayRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
	// CoverID asks who is away and named this person to cover for them.
	CoverID pulid.ID
	// UserIDs asks which of these people are away with nobody covering.
	UserIDs []pulid.ID
}

type Repository interface {
	// AwayFor are the people away right now who named CoverID to cover.
	AwayFor(ctx context.Context, req AwayRequest) ([]pulid.ID, error)
	// Uncovered are those of UserIDs away right now with nobody named to
	// cover: a case routed to one of them would wait unseen.
	Uncovered(ctx context.Context, req AwayRequest) ([]pulid.ID, error)
	Get(ctx context.Context, req RecordRequest) (*caseowner.Owner, error)
	// List are the owners of those of the records that have one, in one read.
	List(ctx context.Context, req RecordsRequest) ([]*caseowner.Owner, error)
	// Set gives the case to its owner. It reports false when the case was
	// given to someone else since the caller looked.
	Set(ctx context.Context, req *SetOwnerRequest) (*caseowner.Owner, bool, error)
	// SetIfUnowned gives an unowned case its owner and leaves an owned one alone.
	SetIfUnowned(ctx context.Context, req *SetOwnerRequest) error
	Release(ctx context.Context, req RecordRequest, expected *caseowner.Owner) (bool, error)
	// Candidates are the people the record itself names for its case, most
	// specific first: its owner, the manager of its fleet, its biller or
	// collector.
	Candidates(ctx context.Context, req CandidatesRequest) ([]pulid.ID, error)
}

type AssignRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
	OwnerID    pulid.ID
	// Label is the record's name, for the notice its new owner gets.
	Label string
}

type ReleaseRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       recordkind.Kind
	RecordID   pulid.ID
}

// Standing is a case's owner as one reader stands to it.
type Standing struct {
	Owner *caseowner.Owner
	// Covering is set when the reader works the case in the owner's place.
	Covering bool
	CanWork  bool
	CanSee   bool
	// CanClaim: the case is unowned and the reader may take it.
	CanClaim bool
	// CanAssign: the reader may give the case to someone, or take it over.
	CanAssign bool
}

// Ownership is the standing as a case shows it to its reader.
func (s *Standing) Ownership() recordcase.Ownership {
	out := recordcase.Ownership{
		Covering:  s.Covering,
		CanWork:   s.CanWork,
		CanClaim:  s.CanClaim,
		CanAssign: s.CanAssign,
	}
	if s.Owner != nil {
		out.OwnerID = s.Owner.OwnerID
		out.Auto = s.Owner.Source == caseowner.SourceAuto
	}

	return out
}

type Service interface {
	// Sight is whose cases the actor works and whether they see everyone's.
	Sight(ctx context.Context, actor *permissionports.RequestActor) (caseowner.Sight, error)
	Standing(
		ctx context.Context,
		req RecordRequest,
		actor *permissionports.RequestActor,
	) (*Standing, error)
	// Standings is Standing for a page of records in one read, keyed by
	// record. Every record asked for has an entry, owned or not.
	Standings(
		ctx context.Context,
		req RecordsRequest,
		actor *permissionports.RequestActor,
	) (map[pulid.ID]*Standing, error)
	// AssertMayWork refuses a change to a record whose case is someone else's.
	AssertMayWork(
		ctx context.Context,
		req RecordRequest,
		actor *permissionports.RequestActor,
	) error
	Claim(
		ctx context.Context,
		req *AssignRequest,
		actor *permissionports.RequestActor,
	) (*caseowner.Owner, error)
	Assign(
		ctx context.Context,
		req *AssignRequest,
		actor *permissionports.RequestActor,
	) (*caseowner.Owner, error)
	Release(ctx context.Context, req *ReleaseRequest, actor *permissionports.RequestActor) error
	// Route gives an unowned case to the first of the people its record names
	// who may work cases, and leaves an owned one alone.
	Route(ctx context.Context, req CandidatesRequest, first ...pulid.ID)
}
