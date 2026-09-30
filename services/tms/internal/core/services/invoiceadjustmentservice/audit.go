package invoiceadjustmentservice

import (
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func (s *Service) logAudit(
	entity *invoiceadjustment.InvoiceAdjustment,
	actor *servicesports.RequestActor,
	op permission.Operation,
	comment string,
) {
	if entity == nil {
		return
	}
	if err := s.auditService.LogAction(
		&servicesports.LogActionParams{
			Resource:       permission.ResourceInvoice,
			ResourceID:     entity.ID.String(),
			Operation:      op,
			UserID:         actor.UserID,
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			APIKeyID:       actor.APIKeyID,
			CurrentState:   jsonutils.MustToJSON(entity),
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		auditservice.WithComment(comment),
	); err != nil {
		s.l.Warn("failed to log invoice adjustment audit action", zap.Error(err))
	}
}

func (s *Service) logAdjustmentEvent(
	message string,
	entity *invoiceadjustment.InvoiceAdjustment,
	level zapcore.Level,
) {
	if entity == nil {
		return
	}

	fields := []zap.Field{
		zap.String("adjustmentId", entity.ID.String()),
		zap.String("invoiceId", entity.OriginalInvoiceID.String()),
		zap.String("correctionGroupId", entity.CorrectionGroupID.String()),
		zap.String("idempotencyKey", entity.IdempotencyKey),
		zap.String("status", string(entity.Status)),
		zap.String("approvalStatus", string(entity.ApprovalStatus)),
	}
	if entity.ExecutionError != "" {
		fields = append(fields, zap.String("executionError", entity.ExecutionError))
	}

	//nolint:exhaustive // only actionable enum states require explicit handling here
	switch level {
	case zap.WarnLevel:
		s.l.Warn(message, fields...)
	case zap.ErrorLevel:
		s.l.Error(message, fields...)
	default:
		s.l.Info(message, fields...)
	}
}
