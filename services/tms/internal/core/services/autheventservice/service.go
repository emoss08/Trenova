package autheventservice

import (
	"context"
	"net/netip"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	writeTimeout      = 3 * time.Second
	securityEventKind = "auth_event"
	maxErrorCodeLen   = 120
	maxProviderLen    = 120
	maxMFAStateLen    = 80
)

type Params struct {
	fx.In

	Repository repositories.AuthEventRepository
	Metrics    *metrics.Registry
	Logger     *zap.Logger
}

type Service struct {
	repo    repositories.AuthEventRepository
	metrics *metrics.Registry
	l       *zap.Logger
}

func New(p Params) services.AuthEventRecorder {
	return &Service{
		repo:    p.Repository,
		metrics: p.Metrics,
		l:       p.Logger.Named("service.auth-event"),
	}
}

func (s *Service) Record(ctx context.Context, rec *services.AuthEventRecord) {
	event := newAuthEvent(ctx, rec)

	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), writeTimeout)
	defer cancel()

	if event.OrganizationID.IsNotNil() {
		writeCtx = dbscope.WithTenant(writeCtx, dbscope.Tenant{
			OrganizationID: event.OrganizationID,
			BusinessUnitID: event.BusinessUnitID,
			UserID:         event.UserID,
		})
	}

	err := s.repo.Create(writeCtx, event)
	s.recordMetric(err == nil)
	if err == nil {
		return
	}

	s.l.Error("failed to record authentication event",
		zap.Error(err),
		zap.String("provider", event.Provider),
		zap.String("outcome", string(event.Outcome)),
		zap.String("errorCode", event.ErrorCode),
		zap.String("userID", event.UserID.String()),
		zap.String("organizationID", event.OrganizationID.String()),
		zap.String("businessUnitID", event.BusinessUnitID.String()),
		zap.String("ipAddress", event.IPAddress),
		zap.String("userAgent", event.UserAgent),
		zap.Int64("occurredAt", event.OccurredAt),
	)
}

func (s *Service) recordMetric(success bool) {
	if s.metrics == nil || s.metrics.Audit == nil {
		return
	}
	s.metrics.Audit.RecordSecurityEvent(securityEventKind, success)
}

func newAuthEvent(ctx context.Context, rec *services.AuthEventRecord) *iam.AuthEvent {
	event := &iam.AuthEvent{
		UserID:           rec.UserID,
		Provider:         stringutils.TruncateRunes(rec.Provider, maxProviderLen),
		Outcome:          rec.Outcome,
		AuthenticatorAAL: max(rec.AuthenticatorAAL, 1),
		FederationFAL:    max(rec.FederationFAL, 1),
		MFAState:         stringutils.TruncateRunes(rec.MFAState, maxMFAStateLen),
		RiskOutcome:      rec.RiskOutcome,
		RiskSignals:      rec.RiskSignals,
		ErrorCode:        stringutils.TruncateRunes(rec.ErrorCode, maxErrorCodeLen),
		OccurredAt:       time.Now().Unix(),
	}

	if !event.Outcome.IsValid() {
		event.Outcome = iam.AuthEventOutcomeFailed
	}
	if !event.RiskOutcome.IsValid() {
		event.RiskOutcome = iam.RiskOutcomeAllow
	}
	if rec.OrganizationID.IsNotNil() && rec.BusinessUnitID.IsNotNil() {
		event.OrganizationID = rec.OrganizationID
		event.BusinessUnitID = rec.BusinessUnitID
	}

	if meta, ok := requestmeta.From(ctx); ok {
		if addr, err := netip.ParseAddr(meta.ClientIP); err == nil {
			event.IPAddress = addr.String()
		}
		event.UserAgent = meta.UserAgent
	}

	return event
}
