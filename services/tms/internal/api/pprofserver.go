//revive:disable-next-line:var-naming
package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type PprofServerParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
	LC     fx.Lifecycle
}

type PprofServer struct {
	server *http.Server
	cfg    *config.Config
	logger *zap.Logger
}

func NewPprofServer(p PprofServerParams) *PprofServer {
	s := &PprofServer{
		cfg:    p.Config,
		logger: p.Logger.Named("pprof-server"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))

	s.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", p.Config.Monitoring.Pprof.GetHost(), p.Config.Monitoring.Pprof.GetPort()),
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

func (s *PprofServer) Start(_ context.Context) error {
	if !s.cfg.Monitoring.Pprof.Enabled {
		s.logger.Info("pprof server disabled")
		return nil
	}

	s.logger.Info("Starting pprof server", zap.String("address", fmt.Sprintf("http://%s", s.server.Addr)))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("failed to start pprof server", zap.Error(err))
		}
	}()

	return nil
}

func (s *PprofServer) Stop(ctx context.Context) error {
	if !s.cfg.Monitoring.Pprof.Enabled {
		return nil
	}

	s.logger.Info("Stopping pprof server")
	return s.server.Shutdown(ctx)
}
