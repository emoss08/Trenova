package common

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/gen"

	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuditableService struct {
	DB            *bun.DB
	AuditService  *audit.Service
	CodeGenerator *gen.CodeGenerator
}

func (as *AuditableService) GetAuditUser(ctx context.Context, userID uuid.UUID) (audit.AuditUser, error) {
	user := new(models.User)
	err := as.DB.NewSelect().Model(user).Where("id = ?", userID).Scan(ctx)
	if err != nil {
		return audit.AuditUser{}, fmt.Errorf("failed to fetch user: %w", err)
	}
	return audit.AuditUser{
		ID:       user.ID,
		Username: user.Username,
	}, nil
}

type Auditable interface {
	Insert(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error
	UpdateOne(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error
}

type CodeGeneratableAuditable interface {
	Auditable
	InsertWithCodeGen(ctx context.Context, tx bun.Tx, codeGen *gen.CodeGenerator, pattern string, auditService *audit.Service, user audit.AuditUser) error
}

func (as *AuditableService) GetByID(ctx context.Context, id, orgID, buID uuid.UUID, entity any) error {
	return as.DB.NewSelect().
		Model(entity).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Where("id = ?", id).
		Scan(ctx)
}

func (as *AuditableService) CreateWithAuditAndCodeGen(ctx context.Context, entity CodeGeneratableAuditable, userID uuid.UUID, pattern string) error {
	auditUser, err := as.GetAuditUser(ctx, userID)
	if err != nil {
		return err
	}

	return as.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return entity.InsertWithCodeGen(ctx, tx, as.CodeGenerator, pattern, as.AuditService, auditUser)
	})
}

func (as *AuditableService) CreateWithAudit(ctx context.Context, entity Auditable, userID uuid.UUID) (audit.AuditUser, error) {
	auditUser, err := as.GetAuditUser(ctx, userID)
	if err != nil {
		return audit.AuditUser{}, err
	}

	return auditUser, as.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return entity.Insert(ctx, tx, as.AuditService, auditUser)
	})
}

func (as *AuditableService) UpdateWithAudit(ctx context.Context, entity Auditable, userID uuid.UUID) error {
	auditUser, err := as.GetAuditUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get audit user %s", err)
	}

	return as.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return entity.UpdateOne(ctx, tx, as.AuditService, auditUser)
	})
}
