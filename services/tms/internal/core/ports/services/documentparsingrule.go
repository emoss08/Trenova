package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DocumentParsingPage struct {
	PageNumber int    `json:"pageNumber"`
	Text       string `json:"text"`
}

type DocumentParsingField struct {
	Key               string   `json:"key"`
	Label             string   `json:"label"`
	Value             string   `json:"value"`
	Confidence        float64  `json:"confidence"`
	PageNumber        int      `json:"pageNumber"`
	ReviewRequired    bool     `json:"reviewRequired"`
	EvidenceExcerpt   string   `json:"evidenceExcerpt"`
	Source            string   `json:"source"`
	AlternativeValues []string `json:"alternativeValues,omitempty"`
}

type DocumentParsingStop struct {
	Sequence            int     `json:"sequence"`
	Role                string  `json:"role"`
	Name                string  `json:"name"`
	AddressLine1        string  `json:"addressLine1"`
	AddressLine2        string  `json:"addressLine2"`
	City                string  `json:"city"`
	State               string  `json:"state"`
	PostalCode          string  `json:"postalCode"`
	Date                string  `json:"date"`
	TimeWindow          string  `json:"timeWindow"`
	AppointmentRequired bool    `json:"appointmentRequired"`
	PageNumber          int     `json:"pageNumber"`
	EvidenceExcerpt     string  `json:"evidenceExcerpt"`
	Confidence          float64 `json:"confidence"`
	ReviewRequired      bool    `json:"reviewRequired"`
	Source              string  `json:"source"`
}

type DocumentParsingConflict struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Values          []string `json:"values"`
	PageNumbers     []int    `json:"pageNumbers"`
	EvidenceExcerpt string   `json:"evidenceExcerpt"`
	Source          string   `json:"source"`
}

type DocumentParsingRuleMetadata struct {
	RuleSetID        pulid.ID `json:"ruleSetId"`
	RuleSetName      string   `json:"ruleSetName"`
	RuleVersionID    pulid.ID `json:"ruleVersionId"`
	VersionNumber    int      `json:"versionNumber"`
	ParserMode       string   `json:"parserMode"`
	ProviderMatched  string   `json:"providerMatched"`
	MatchSpecificity int      `json:"matchSpecificity"`
}

type DocumentParsingAnalysis struct {
	Fields            map[string]DocumentParsingField `json:"fields"`
	Stops             []DocumentParsingStop           `json:"stops"`
	Conflicts         []DocumentParsingConflict       `json:"conflicts"`
	MissingFields     []string                        `json:"missingFields"`
	Signals           []string                        `json:"signals"`
	ReviewStatus      string                          `json:"reviewStatus"`
	OverallConfidence float64                         `json:"overallConfidence"`
	Metadata          *DocumentParsingRuleMetadata    `json:"metadata,omitempty"`
}

type DocumentParsingRuntimeInput struct {
	TenantInfo          pagination.TenantInfo
	DocumentKind        string
	FileName            string
	Text                string
	ProviderFingerprint string
	Pages               []DocumentParsingPage
}

type DocumentParsingSimulationRequest struct {
	TenantInfo          pagination.TenantInfo
	VersionID           pulid.ID
	FileName            string
	Text                string
	Pages               []DocumentParsingPage
	ProviderFingerprint string
	Baseline            *DocumentParsingAnalysis
}

type DocumentParsingSimulationDiff struct {
	AddedFields      []string `json:"addedFields"`
	ChangedFields    []string `json:"changedFields"`
	AddedStopRoles   []string `json:"addedStopRoles"`
	ChangedStopRoles []string `json:"changedStopRoles"`
}

type DocumentParsingSimulationResult struct {
	Matched          bool                          `json:"matched"`
	ValidationPassed bool                          `json:"validationPassed"`
	ValidationErrors []string                      `json:"validationErrors"`
	Metadata         *DocumentParsingRuleMetadata  `json:"metadata,omitempty"`
	Baseline         *DocumentParsingAnalysis      `json:"baseline,omitempty"`
	Candidate        *DocumentParsingAnalysis      `json:"candidate,omitempty"`
	Diff             DocumentParsingSimulationDiff `json:"diff"`
}

type DocumentParsingRuleRuntime interface {
	ApplyPublished(
		ctx context.Context,
		input *DocumentParsingRuntimeInput,
		baseline *DocumentParsingAnalysis,
	) (*DocumentParsingAnalysis, error)
	SimulateVersion(
		ctx context.Context,
		req *DocumentParsingSimulationRequest,
	) (*DocumentParsingSimulationResult, error)
}

type DocumentParsingRuleAdminService interface {
	ListRuleSets(ctx context.Context, tenantInfo pagination.TenantInfo, documentKind string) ([]*documentparsingrule.RuleSet, error)
	GetRuleSet(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo) (*documentparsingrule.RuleSet, error)
	CreateRuleSet(ctx context.Context, entity *documentparsingrule.RuleSet, userID pulid.ID) (*documentparsingrule.RuleSet, error)
	UpdateRuleSet(ctx context.Context, entity *documentparsingrule.RuleSet, userID pulid.ID) (*documentparsingrule.RuleSet, error)
	DeleteRuleSet(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo, userID pulid.ID) error

	ListVersions(ctx context.Context, ruleSetID pulid.ID, tenantInfo pagination.TenantInfo) ([]*documentparsingrule.RuleVersion, error)
	GetVersion(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo) (*documentparsingrule.RuleVersion, error)
	CreateVersion(ctx context.Context, entity *documentparsingrule.RuleVersion, userID pulid.ID) (*documentparsingrule.RuleVersion, error)
	UpdateVersion(ctx context.Context, entity *documentparsingrule.RuleVersion, userID pulid.ID) (*documentparsingrule.RuleVersion, error)
	PublishVersion(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo, userID pulid.ID) (*documentparsingrule.RuleVersion, error)

	ListFixtures(ctx context.Context, ruleSetID pulid.ID, tenantInfo pagination.TenantInfo) ([]*documentparsingrule.Fixture, error)
	GetFixture(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo) (*documentparsingrule.Fixture, error)
	SaveFixture(ctx context.Context, entity *documentparsingrule.Fixture, userID pulid.ID) (*documentparsingrule.Fixture, error)
	DeleteFixture(ctx context.Context, id pulid.ID, tenantInfo pagination.TenantInfo, userID pulid.ID) error
}
