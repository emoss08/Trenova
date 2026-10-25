package apikey

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Key struct {
	keel.Table `keel:"api_keys"`

	ID                pulid.ID `json:"id"                keel:"id,pk,autoid:ak_"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    keel:"business_unit_id,tenant"`
	OrganizationID    pulid.ID `json:"organizationId"    keel:"organization_id,tenant"`
	Name              string   `json:"name"              keel:"name"`
	Description       string   `json:"description"       keel:"description,nullzero"`
	KeyPrefix         string   `json:"keyPrefix"         keel:"key_prefix"`
	SecretHash        string   `json:"-"                 keel:"secret_hash"`
	SecretSalt        string   `json:"-"                 keel:"secret_salt"`
	Status            Status   `json:"status"            keel:"status"`
	ExpiresAt         int64    `json:"expiresAt"         keel:"expires_at,nullzero"`
	LastUsedAt        int64    `json:"lastUsedAt"        keel:"last_used_at,nullzero"`
	LastUsedIP        string   `json:"lastUsedIp"        keel:"last_used_ip,nullzero"`
	LastUsedUserAgent string   `json:"lastUsedUserAgent" keel:"last_used_user_agent,nullzero"`
	CreatedByID       pulid.ID `json:"createdById"       keel:"created_by_id"`
	RevokedByID       pulid.ID `json:"revokedById"       keel:"revoked_by_id,nullzero"`
	RevokedAt         int64    `json:"revokedAt"         keel:"revoked_at,nullzero"`
	CreatedAt         int64    `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64    `json:"updatedAt"         keel:"updated_at,autoupdate"`

	Permissions []*Permission `json:"permissions" keel:"many:api_key_id"`
}

func (k *Key) BeforeCreate(_ context.Context) error {
	if k.Status == "" {
		k.Status = StatusActive
	}

	return nil
}

func (k *Key) GetID() pulid.ID {
	return k.ID
}

func (k *Key) GetCreatedAt() int64 {
	return k.CreatedAt
}

func (k *Key) GetOrganizationID() pulid.ID {
	return k.OrganizationID
}

func (k *Key) GetBusinessUnitID() pulid.ID {
	return k.BusinessUnitID
}

func (k *Key) GetTableName() string {
	return "api_keys"
}

func (k *Key) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ak",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
			{Name: "keyPrefix", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
			{Name: "lastUsedAt", Type: domaintypes.FieldTypeNumber},
			{Name: "expiresAt", Type: domaintypes.FieldTypeNumber},
			{Name: "createdAt", Type: domaintypes.FieldTypeNumber},
			{Name: "updatedAt", Type: domaintypes.FieldTypeNumber},
		},
	}
}

func (k *Key) IsExpired(now int64) bool {
	return k.ExpiresAt > 0 && k.ExpiresAt <= now
}

func (k *Key) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(k,
		validation.Field(&k.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&k.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&k.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxKeyNameLength).
				Error("Name cannot be longer than 255 characters"),
		),
		validation.Field(&k.KeyPrefix,
			validation.Required.Error("Key prefix is required"),
			validation.Length(1, maxKeyPrefixLength).
				Error("Key prefix cannot be longer than 32 characters"),
		),
		// The secret is never stored in the clear, so a key with no digest or no
		// salt would authenticate nobody and could not be repaired.
		validation.Field(&k.SecretHash, validation.Required.Error("Secret hash is required")),
		validation.Field(&k.SecretSalt, validation.Required.Error("Secret salt is required")),
		validation.Field(&k.Status,
			validation.Required.Error("Status is required"),
			validation.In(StatusActive, StatusRevoked).Error("Status is invalid"),
		),
		validation.Field(&k.CreatedByID, validation.Required.Error("Created by is required")),
		// A revocation is an authorization record: who revoked it and when are
		// both part of it or neither is.
		validation.Field(&k.RevokedByID,
			validation.When(k.RevokedAt != 0, validation.Required.Error(
				"A revoked key must record who revoked it",
			)),
		),
		validation.Field(&k.RevokedAt,
			validation.When(k.RevokedByID.IsNotNil(), validation.Required.Error(
				"A revoked key must record when it was revoked",
			)),
			validation.When(k.Status == StatusRevoked, validation.Required.Error(
				"A revoked key must record when it was revoked",
			)),
		),
		validation.Field(&k.LastUsedIP,
			validation.Length(0, maxIPLength).
				Error("Last used IP cannot be longer than 45 characters"),
		),
		validation.Field(&k.LastUsedUserAgent,
			validation.Length(0, maxUserAgentLength).
				Error("Last used user agent cannot be longer than 255 characters"),
		),
	))
}

const (
	maxKeyNameLength   = 255
	maxKeyPrefixLength = 32
	maxIPLength        = 45
	maxUserAgentLength = 255
)
