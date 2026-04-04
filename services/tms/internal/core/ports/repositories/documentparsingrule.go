package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentparsingrule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDocumentParsingRuleSetsRequest struct {
	TenantInfo   pagination.TenantInfo
	DocumentKind string
}

type GetDocumentParsingRuleSetRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListDocumentParsingRuleVersionsRequest struct {
	RuleSetID  pulid.ID
	TenantInfo pagination.TenantInfo
	IncludeAll bool
}

type GetDocumentParsingRuleVersionRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListDocumentParsingRuleFixturesRequest struct {
	RuleSetID  pulid.ID
	TenantInfo pagination.TenantInfo
}

type GetDocumentParsingRuleFixtureRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
}

type PublishedDocumentParsingRuleVersion struct {
	RuleSet *documentparsingrule.RuleSet
	Version *documentparsingrule.RuleVersion
}

type DocumentParsingRuleRepository interface {
	ListRuleSets(ctx context.Context, req ListDocumentParsingRuleSetsRequest) ([]*documentparsingrule.RuleSet, error)
	GetRuleSet(ctx context.Context, req GetDocumentParsingRuleSetRequest) (*documentparsingrule.RuleSet, error)
	CreateRuleSet(ctx context.Context, entity *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error)
	UpdateRuleSet(ctx context.Context, entity *documentparsingrule.RuleSet) (*documentparsingrule.RuleSet, error)
	DeleteRuleSet(ctx context.Context, req GetDocumentParsingRuleSetRequest) error

	ListVersions(ctx context.Context, req ListDocumentParsingRuleVersionsRequest) ([]*documentparsingrule.RuleVersion, error)
	GetVersion(ctx context.Context, req GetDocumentParsingRuleVersionRequest) (*documentparsingrule.RuleVersion, error)
	GetVersionWithRuleSet(ctx context.Context, req GetDocumentParsingRuleVersionRequest) (*documentparsingrule.RuleVersion, *documentparsingrule.RuleSet, error)
	CreateVersion(ctx context.Context, entity *documentparsingrule.RuleVersion) (*documentparsingrule.RuleVersion, error)
	UpdateVersion(ctx context.Context, entity *documentparsingrule.RuleVersion) (*documentparsingrule.RuleVersion, error)
	ArchivePublishedVersions(ctx context.Context, ruleSetID, orgID, buID pulid.ID) error
	SetPublishedVersion(ctx context.Context, ruleSetID, versionID, orgID, buID pulid.ID) error
	NextVersionNumber(ctx context.Context, ruleSetID, orgID, buID pulid.ID) (int, error)
	ListPublishedVersionsByDocumentKind(ctx context.Context, tenantInfo pagination.TenantInfo, documentKind string) ([]*PublishedDocumentParsingRuleVersion, error)

	ListFixtures(ctx context.Context, req ListDocumentParsingRuleFixturesRequest) ([]*documentparsingrule.Fixture, error)
	GetFixture(ctx context.Context, req GetDocumentParsingRuleFixtureRequest) (*documentparsingrule.Fixture, error)
	CreateFixture(ctx context.Context, entity *documentparsingrule.Fixture) (*documentparsingrule.Fixture, error)
	UpdateFixture(ctx context.Context, entity *documentparsingrule.Fixture) (*documentparsingrule.Fixture, error)
	DeleteFixture(ctx context.Context, req GetDocumentParsingRuleFixtureRequest) error
}
