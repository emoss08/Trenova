package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListModeProfilesRequest struct {
	Filter *pagination.QueryOptions
	Status string
}

type GetModeProfileByIDRequest struct {
	ModeProfileID pulid.ID
	TenantInfo    pagination.TenantInfo
	IncludeRules  bool
}

type ModeProfileCacheRepository interface {
	GetActiveProfiles(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*modeprofile.Profile, error)

	SetActiveProfiles(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		profiles []*modeprofile.Profile,
	) error

	Invalidate(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) error
}

type ModeProfileRepository interface {
	List(
		ctx context.Context,
		req *ListModeProfilesRequest,
	) (*pagination.ListResult[*modeprofile.Profile], error)

	GetActiveProfiles(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*modeprofile.Profile, error)

	GetByID(
		ctx context.Context,
		req *GetModeProfileByIDRequest,
	) (*modeprofile.Profile, error)

	Create(
		ctx context.Context,
		entity *modeprofile.Profile,
	) (*modeprofile.Profile, error)

	Update(
		ctx context.Context,
		entity *modeprofile.Profile,
	) (*modeprofile.Profile, error)
}

type ListDeviationsRequest struct {
	Filter       *pagination.QueryOptions
	ResourceType string
	ResourceID   pulid.ID
	RuleKey      string
	State        string
}

type GetDeviationByIDRequest struct {
	DeviationID pulid.ID
	TenantInfo  pagination.TenantInfo
}

type AcknowledgeDeviationRequest struct {
	DeviationID      pulid.ID
	TenantInfo       pagination.TenantInfo
	AcknowledgedByID pulid.ID
	Reason           string
}

type SupersedeDeviationsRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceID   pulid.ID
	KeepRuleKeys []string
}

// LiveDeviationKeysRequest asks which rules already have a deviation on record
// for a resource that has not been resolved.
//
// Only the keys are returned rather than whole rows: the caller uses them to
// decide what not to insert again, and reading the rows would invite carrying
// stale copies of state that only the database should own.
type LiveDeviationKeysRequest struct {
	TenantInfo   pagination.TenantInfo
	ResourceType string
	ResourceID   pulid.ID
}

type DeviationLedgerEntry struct {
	RuleKey        string `json:"ruleKey"`
	Capability     string `json:"capability"`
	Label          string `json:"label"`
	Occurrences    int64  `json:"occurrences"`
	OpenCount      int64  `json:"openCount"`
	LastOccurredAt int64  `json:"lastOccurredAt"`
}

type DeviationLedgerRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

type DeviationRepository interface {
	RecordMany(
		ctx context.Context,
		deviations []*modeprofile.Deviation,
	) error

	List(
		ctx context.Context,
		req *ListDeviationsRequest,
	) (*pagination.ListResult[*modeprofile.Deviation], error)

	GetByID(
		ctx context.Context,
		req *GetDeviationByIDRequest,
	) (*modeprofile.Deviation, error)

	Acknowledge(
		ctx context.Context,
		req *AcknowledgeDeviationRequest,
	) (*modeprofile.Deviation, error)

	SupersedeOpenForResource(
		ctx context.Context,
		req *SupersedeDeviationsRequest,
	) error

	// LiveRuleKeysForResource covers Open and Acknowledged alike. An
	// acknowledged deviation is still on record, so re-inserting its rule would
	// leave the resource carrying the same finding twice, once accepted and
	// once not.
	LiveRuleKeysForResource(
		ctx context.Context,
		req *LiveDeviationKeysRequest,
	) ([]string, error)

	Ledger(
		ctx context.Context,
		req *DeviationLedgerRequest,
	) ([]*DeviationLedgerEntry, error)
}
