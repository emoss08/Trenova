package databasesessionservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/system"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.DatabaseSessionRepository
	AuditService services.AuditService
	Metrics      *metrics.Registry
}

type Service struct {
	l            *zap.Logger
	repo         repositories.DatabaseSessionRepository
	auditService services.AuditService
	metrics      *metrics.Registry
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.databasesession"),
		repo:         p.Repo,
		auditService: p.AuditService,
		metrics:      p.Metrics,
	}
}

func (s *Service) ListBlocked(ctx context.Context) ([]*system.DatabaseSessionChain, error) {
	rows, err := s.repo.ListBlocked(ctx)
	if err != nil {
		s.recordAction("diagnostics_fetch", "error")
		return nil, err
	}

	s.recordAction("diagnostics_fetch", "success")
	return rows, nil
}

func (s *Service) TerminateSession(
	ctx context.Context,
	pid int64,
	userID, orgID, buID pulid.ID,
) (*system.TerminateDatabaseSessionResult, error) {
	result, err := s.repo.Terminate(ctx, pid)
	if err != nil {
		s.recordAction("terminate", "error")
		return nil, err
	}

	s.recordAction("terminate", "success")
	if err := s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAuditLog,
		ResourceID:     strconv.FormatInt(pid, 10),
		Operation:      permission.OpUpdate,
		CurrentState:   map[string]any{"pid": pid, "terminated": result.Terminated, "message": result.Message},
		PreviousState:  nil,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Critical:       true,
	}); err != nil {
		s.l.Warn(
			"failed to audit database session termination",
			zap.Error(err),
			zap.Int64("pid", pid),
		)
	}

	return result, nil
}

func (s *Service) recordAction(action, outcome string) {
	if s.metrics == nil || !s.metrics.IsEnabled() {
		return
	}

	s.metrics.Database.RecordOperatorAction(action, outcome)
}
