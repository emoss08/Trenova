package apikey

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Key struct {
	bun.BaseModel `bun:"table:api_keys,alias:ak" json:"-"`

	ID                pulid.ID `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),notnull"`
	Name              string   `json:"name"              bun:"name,type:VARCHAR(255),notnull"`
	Description       string   `json:"description"       bun:"description,type:TEXT,nullzero"`
	KeyPrefix         string   `json:"keyPrefix"         bun:"key_prefix,type:VARCHAR(32),notnull"`
	SecretHash        string   `json:"-"                 bun:"secret_hash,type:VARCHAR(128),notnull"`
	SecretSalt        string   `json:"-"                 bun:"secret_salt,type:VARCHAR(64),notnull"`
	Status            Status   `json:"status"            bun:"status,type:api_key_status_enum,notnull,default:'active'"`
	ExpiresAt         int64    `json:"expiresAt"         bun:"expires_at,nullzero"`
	LastUsedAt        int64    `json:"lastUsedAt"        bun:"last_used_at,nullzero"`
	LastUsedIP        string   `json:"lastUsedIp"        bun:"last_used_ip,type:VARCHAR(45),nullzero"`
	LastUsedUserAgent string   `json:"lastUsedUserAgent" bun:"last_used_user_agent,type:VARCHAR(255),nullzero"`
	CreatedByID       pulid.ID `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),notnull"`
	RevokedByID       pulid.ID `json:"revokedById"       bun:"revoked_by_id,type:VARCHAR(100),nullzero"`
	RevokedAt         int64    `json:"revokedAt"         bun:"revoked_at,nullzero"`
	CreatedAt         int64    `json:"createdAt"         bun:"created_at,notnull"`
	UpdatedAt         int64    `json:"updatedAt"         bun:"updated_at,notnull"`

	Permissions []*Permission `json:"permissions" bun:"rel:has-many,join:id=api_key_id"`
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

func (k *Key) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if k.ID.IsNil() {
			k.ID = pulid.MustNew("ak_")
		}
		k.CreatedAt = now
		k.UpdatedAt = now
	case *bun.UpdateQuery:
		k.UpdatedAt = now
	}

	return nil
}

func (k *Key) IsExpired(now int64) bool {
	return k.ExpiresAt > 0 && k.ExpiresAt <= now
}
