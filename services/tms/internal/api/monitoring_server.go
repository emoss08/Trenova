package api

import (
	"context"
	"encoding/json" //nolint:depguard // this is fine
	"fmt"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MonitoringServerParams struct {
	fx.In

	Config   *config.Config
	Logger   *zap.Logger
	LC       fx.Lifecycle
	Metrics  *metrics.Registry
	Database ports.DBConnection
}

type MonitoringServer struct {
	server   *http.Server
	cfg      *config.Config
	logger   *zap.Logger
	metrics  *metrics.Registry
	database ports.DBConnection
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
	addr := fmt.Sprintf("%s:%d", p.Config.Server.Host, p.Config.Monitoring.Metrics.Port)

	s := &MonitoringServer{
		cfg:      p.Config,
		logger:   p.Logger.Named("monitoring-server"),
		metrics:  p.Metrics,
		database: p.Database,
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
	if s.database == nil {
		return monitoringCheck{
			Status:  "down",
			Message: "database connection is not configured",
		}
	}

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
