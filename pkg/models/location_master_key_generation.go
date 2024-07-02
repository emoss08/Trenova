package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type LocationMasterKeyGenerationPermission string

const (
	// PermissionLocationMasterKeygenerationView is the permission to view location master key generation details
	PermissionLocationMasterKeygenerationView = LocationMasterKeyGenerationPermission("locationmasterkeygeneration.view")

	// PermissionLocationMasterKeygenerationEdit is the permission to edit location master key generation details
	PermissionLocationMasterKeygenerationEdit = LocationMasterKeyGenerationPermission("locationmasterkeygeneration.edit")

	// PermissionLocationMasterKeygenerationAdd is the permission to add a new location master key generation
	PermissionLocationMasterKeygenerationAdd = LocationMasterKeyGenerationPermission("locationmasterkeygeneration.add")

	// PermissionLocationMasterKeygenerationDelete is the permission to delete an location master key generation
	PermissionLocationMasterKeygenerationDelete = LocationMasterKeyGenerationPermission("locationmasterkeygeneration.delete")
)

// String returns the string representation of the LocationMasterKeyGenerationPermission
func (p LocationMasterKeyGenerationPermission) String() string {
	return string(p)
}

type LocationMasterKeyGeneration struct {
	bun.BaseModel `bun:"table:location_master_key_generations,alias:lmkg" json:"-"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Pattern       string     `bun:"type:VARCHAR(255),notnull" json:"pattern"`
	MasterKeyID   *uuid.UUID `bun:"type:uuid" json:"masterKeyGenerationId"`

	MasterKey *MasterKeyGeneration `bun:"rel:belongs-to,join:master_key_id=id" json:"masterKeyGeneration"`
}

func QueryLocationMasterKeyGenerationByOrgID(ctx context.Context, db *bun.DB, orgID uuid.UUID) (*LocationMasterKeyGeneration, error) {
	var locationMasterKeyGeneration LocationMasterKeyGeneration
	err := db.NewSelect().Model(&locationMasterKeyGeneration).Relation("MasterKey").Where("master_key.organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &locationMasterKeyGeneration, nil
}

var _ bun.BeforeAppendModelHook = (*LocationMasterKeyGeneration)(nil)

func (m *LocationMasterKeyGeneration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		m.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		m.UpdatedAt = time.Now()
	}
	return nil
}
