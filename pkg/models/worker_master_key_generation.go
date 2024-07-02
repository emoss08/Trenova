package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WorkerMasterKeyGenerationPermission string

const (
	// PermissionWorkerMasterKeygenerationView is the permission to view worker master key generation details
	PermissionWorkerMasterKeygenerationView = WorkerMasterKeyGenerationPermission("workermasterkeygeneration.view")

	// PermissionWorkerMasterKeygenerationEdit is the permission to edit worker master key generation details
	PermissionWorkerMasterKeygenerationEdit = WorkerMasterKeyGenerationPermission("workermasterkeygeneration.edit")

	// PermissionWorkerMasterKeygenerationAdd is the permission to add a new worker master key generation
	PermissionWorkerMasterKeygenerationAdd = WorkerMasterKeyGenerationPermission("workermasterkeygeneration.add")

	// PermissionWorkerMasterKeygenerationDelete is the permission to delete an worker master key generation
	PermissionWorkerMasterKeygenerationDelete = WorkerMasterKeyGenerationPermission("workermasterkeygeneration.delete")
)

// String returns the string representation of the WorkerMasterKeyGenerationPermission
func (p WorkerMasterKeyGenerationPermission) String() string {
	return string(p)
}

type WorkerMasterKeyGeneration struct {
	bun.BaseModel `bun:"table:worker_master_key_generations,alias:wmkg" json:"-"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Pattern       string     `bun:"type:VARCHAR(255),notnull" json:"pattern"`
	MasterKeyID   *uuid.UUID `bun:"type:uuid" json:"masterKeyGenerationId"`

	MasterKey *MasterKeyGeneration `bun:"rel:belongs-to,join:master_key_id=id" json:"masterKeyGeneration"`
}

func QueryWorkerMasterKeyGenerationByOrgID(ctx context.Context, db *bun.DB, orgID uuid.UUID) (*WorkerMasterKeyGeneration, error) {
	var workerMasterKeyGeneration WorkerMasterKeyGeneration
	err := db.NewSelect().Model(&workerMasterKeyGeneration).Relation("MasterKey").Where("master_key.organization_id = ?", orgID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &workerMasterKeyGeneration, nil
}

var _ bun.BeforeAppendModelHook = (*WorkerMasterKeyGeneration)(nil)

func (c *WorkerMasterKeyGeneration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
