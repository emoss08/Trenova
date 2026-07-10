//revive:disable-next-line:var-naming
package api

import (
	"context"
	"encoding/json" //nolint:depguard // this is fine
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MonitoringServerParams struct {
	fx.In

	Config         *config.Config
	Logger         *zap.Logger
	LC             fx.Lifecycle
	Metrics        *metrics.Registry
	Database       ports.DBConnection
	EDIMessageRepo repositories.EDIMessageRepository
	EDIInboundRepo repositories.EDIInboundFileRepository
	EDIProfileRepo repositories.EDICommunicationProfileRepository
}

type MonitoringServer struct {
	server         *http.Server
	cfg            *config.Config
	logger         *zap.Logger
	metrics        *metrics.Registry
	database       ports.DBConnection
	ediMessageRepo repositories.EDIMessageRepository
	ediInboundRepo repositories.EDIInboundFileRepository
	ediProfileRepo repositories.EDICommunicationProfileRepository
}

type monitoringCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type monitoringResponse struct {
	Status    string                     `json:"status"`
	Timestamp time.Time                  `json:"timestamp"`
	Checks    map[string]monitoringCheck `json:"checks,omitempty"`
}

func NewMonitoringServer(p MonitoringServerParams) *MonitoringServer {
	addr := fmt.Sprintf(
		"%s:%d",
		p.Config.Monitoring.Metrics.GetHost(),
		p.Config.Monitoring.Metrics.Port,
	)

	s := &MonitoringServer{
		cfg:            p.Config,
		logger:         p.Logger.Named("monitoring-server"),
		metrics:        p.Metrics,
		database:       p.Database,
		ediMessageRepo: p.EDIMessageRepo,
		ediInboundRepo: p.EDIInboundRepo,
		ediProfileRepo: p.EDIProfileRepo,
	}

	mux := http.NewServeMux()
	mux.Handle(p.Config.Monitoring.Metrics.Path, p.Metrics.Handler())
	mux.HandleFunc(p.Config.Monitoring.Health.Path, s.handleHealth)
	mux.HandleFunc(p.Config.Monitoring.Health.ReadinessPath, s.handleReadiness)
	mux.HandleFunc(p.Config.Monitoring.Health.LivenessPath, s.handleLiveness)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	p.LC.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return s.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return s.Stop(ctx)
		},
	})

	return s
}

func (s *MonitoringServer) Start(_ context.Context) error {
	s.logger.Info("Starting monitoring server",
		zap.String("address", fmt.Sprintf("http://%s", s.server.Addr)),
		zap.String("metrics_path", s.cfg.Monitoring.Metrics.Path),
		zap.String("health_path", s.cfg.Monitoring.Health.Path),
		zap.String("readiness_path", s.cfg.Monitoring.Health.ReadinessPath),
		zap.String("liveness_path", s.cfg.Monitoring.Health.LivenessPath),
		zap.Bool("metrics_enabled", s.metrics.IsEnabled()),
	)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("failed to start monitoring server", zap.Error(err))
		}
	}()

	return nil
}

func (s *MonitoringServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping monitoring server")
	return s.server.Shutdown(ctx)
}

func (s *MonitoringServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := map[string]monitoringCheck{
		"liveness": {
			Status: "up",
		},
		"database": s.databaseCheck(r.Context()),
		"edi":      s.ediCheck(r.Context()),
	}

	statusCode := http.StatusOK
	status := "up"
	if checks["database"].Status != "up" {
		statusCode = http.StatusServiceUnavailable
		status = "down"
	}

	s.writeJSON(w, statusCode, monitoringResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Checks:    checks,
	})
}

func (s *MonitoringServer) handleReadiness(w http.ResponseWriter, r *http.Request) {
	dbCheck := s.databaseCheck(r.Context())
	statusCode := http.StatusOK
	status := "ready"
	if dbCheck.Status != "up" {
		statusCode = http.StatusServiceUnavailable
		status = "not_ready"
	}

	s.writeJSON(w, statusCode, monitoringResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
		Checks: map[string]monitoringCheck{
			"database": dbCheck,
		},
	})
}

func (s *MonitoringServer) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, monitoringResponse{
		Status:    "alive",
		Timestamp: time.Now().UTC(),
		Checks: map[string]monitoringCheck{
			"process": {
				Status: "up",
			},
		},
	})
}

func (s *MonitoringServer) databaseCheck(ctx context.Context) monitoringCheck {
	if err := s.database.HealthCheck(ctx); err != nil {
		return monitoringCheck{
			Status:  "down",
			Message: err.Error(),
		}
	}

	return monitoringCheck{
		Status: "up",
	}
}

const (
	ediHealthLookback       = 24 * time.Hour
	ediHealthPollStaleAfter = 30 * time.Minute
)

func (s *MonitoringServer) ediCheck(ctx context.Context) monitoringCheck {
	now := time.Now()
	issues := make([]string, 0, 3)

	deadLettered, err := s.ediMessageRepo.CountDeadLetteredSince(
		ctx,
		now.Add(-ediHealthLookback).Unix(),
	)
	switch {
	case err != nil:
		issues = append(issues, "dead-letter backlog could not be read: "+err.Error())
	case deadLettered > 0:
		issues = append(issues, fmt.Sprintf(
			"%d message(s) dead-lettered in the last 24h",
			deadLettered,
		))
	}

	quarantined, err := s.ediInboundRepo.CountQuarantinedSince(
		ctx,
		now.Add(-ediHealthLookback).Unix(),
	)
	switch {
	case err != nil:
		issues = append(issues, "quarantine backlog could not be read: "+err.Error())
	case quarantined > 0:
		issues = append(issues, fmt.Sprintf(
			"%d inbound file(s) quarantined in the last 24h",
			quarantined,
		))
	}

	staleProfiles, err := s.ediProfileRepo.CountStaleInboundPollingProfiles(
		ctx,
		now.Add(-ediHealthPollStaleAfter).Unix(),
	)
	switch {
	case err != nil:
		issues = append(issues, "mailbox poll status could not be read: "+err.Error())
	case staleProfiles > 0:
		issues = append(issues, fmt.Sprintf(
			"%d polling profile(s) have not reached their mailbox in over %s",
			staleProfiles,
			ediHealthPollStaleAfter,
		))
	}

	if len(issues) > 0 {
		return monitoringCheck{
			Status:  "degraded",
			Message: strings.Join(issues, "; "),
		}
	}

	return monitoringCheck{Status: "up"}
}

func (s *MonitoringServer) writeJSON(
	w http.ResponseWriter,
	statusCode int,
	payload monitoringResponse,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Warn("failed to write monitoring response", zap.Error(err))
	}
}
