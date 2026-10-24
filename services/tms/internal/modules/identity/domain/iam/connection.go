package iam

import (
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func activityUserRelationship() *domaintypes.RelationshipDefintion {
	return &domaintypes.RelationshipDefintion{Field: "user", Queryable: true}
}

func (e *AuthEvent) GetID() pulid.ID {
	return e.ID
}

func (e *AuthEvent) GetCreatedAt() int64 {
	return e.CreatedAt
}

func (e *AuthEvent) GetTableName() string {
	return "auth_events"
}

func (e *AuthEvent) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ae",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "provider", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "error_code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (d *RiskDecision) GetID() pulid.ID {
	return d.ID
}

func (d *RiskDecision) GetCreatedAt() int64 {
	return d.CreatedAt
}

func (d *RiskDecision) GetTableName() string {
	return "risk_decisions"
}

func (d *RiskDecision) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "rd",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "reason", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (e *ExternalIdentity) GetID() pulid.ID {
	return e.ID
}

func (e *ExternalIdentity) GetCreatedAt() int64 {
	return e.CreatedAt
}

func (e *ExternalIdentity) GetTableName() string {
	return "external_identities"
}

func (e *ExternalIdentity) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "extid",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "external_email",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "external_username",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "external_subject",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (a *MFAAuthenticator) GetID() pulid.ID {
	return a.ID
}

func (a *MFAAuthenticator) GetCreatedAt() int64 {
	return a.CreatedAt
}

func (a *MFAAuthenticator) GetTableName() string {
	return "mfa_authenticators"
}

func (a *MFAAuthenticator) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "mfa",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}

func (d *SCIMDirectory) GetID() pulid.ID {
	return d.ID
}

func (d *SCIMDirectory) GetCreatedAt() int64 {
	return d.CreatedAt
}

func (d *SCIMDirectory) GetTableName() string {
	return "scim_directories"
}

func (d *SCIMDirectory) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "sd",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "tenant_slug", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
	}
}
