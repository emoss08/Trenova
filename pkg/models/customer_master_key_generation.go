package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CustomerMasterKeyGenerationPermission string

const (
	// PermissionCustomerMasterKeygenerationView is the permission to view customer master key generation details
	PermissionCustomerMasterKeygenerationView = CustomerMasterKeyGenerationPermission("customermasterkeygeneration.view")

	// PermissionCustomerMasterKeygenerationEdit is the permission to edit customer master key generation details
	PermissionCustomerMasterKeygenerationEdit = CustomerMasterKeyGenerationPermission("customermasterkeygeneration.edit")

	// PermissionCustomerMasterKeygenerationAdd is the permission to add a new customer master key generation
	PermissionCustomerMasterKeygenerationAdd = CustomerMasterKeyGenerationPermission("customermasterkeygeneration.add")

	// PermissionCustomerMasterKeygenerationDelete is the permission to delete an customer master key generation
	PermissionCustomerMasterKeygenerationDelete = CustomerMasterKeyGenerationPermission("customermasterkeygeneration.delete")
)

// String returns the string representation of the CustomerMasterKeyGenerationPermission
func (p CustomerMasterKeyGenerationPermission) String() string {
	return string(p)
}

type CustomerMasterKeyGeneration struct {
	bun.BaseModel `bun:"table:worker_master_key_generations,alias:wmkg" json:"-"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Pattern       string     `bun:"type:VARCHAR(255),notnull" json:"pattern"`
	MasterKeyID   *uuid.UUID `bun:"type:uuid" json:"masterKeyGenerationId"`

	MasterKey *MasterKeyGeneration `bun:"rel:belongs-to,join:master_key_id=id" json:"masterKeyGeneration"`
}

func QueryCustomerMasterKeyGenerationByOrgID(ctx context.Context, db *bun.DB, orgID uuid.UUID) (*CustomerMasterKeyGeneration, error) {
	var customerMasterKeyGeneration CustomerMasterKeyGeneration
	err := db.NewSelect().Model(&customerMasterKeyGeneration).Relation("MasterKey").Where("master_key.organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &customerMasterKeyGeneration, nil
}

var _ bun.BeforeAppendModelHook = (*CustomerMasterKeyGeneration)(nil)

func (c *CustomerMasterKeyGeneration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
