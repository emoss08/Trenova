package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/db"
	portrepo "github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/grpc/edicfg"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	configpb "github.com/emoss08/trenova/shared/edi/proto/config/v1"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Params bundles dependencies via Fx.
type Params struct {
	fx.In

	Lc     fx.Lifecycle
	Cfg    *config.Config
	Logger *logger.Logger
	Conn   db.Connection
	Repo   portrepo.EDIPartnerConfigRepository
}

type GRPCServer struct {
	srv    *grpc.Server
	lis    net.Listener
	cfg    *config.GRPCServerConfig
	logger *zerolog.Logger
}

// New creates and manages the lifecycle of a gRPC server.
func New(p Params) (*GRPCServer, error) {
	l := p.Logger.With().
		Str("component", "grpc_server").
		Logger()
	s := &GRPCServer{cfg: &p.Cfg.GRPC, logger: &l}

	p.Lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if !s.cfg.Enabled {
				l.Info().Msg("gRPC server disabled; skipping startup")
				return nil
			}
			ln, err := net.Listen("tcp", s.cfg.ListenAddress)
			if err != nil {
				return fmt.Errorf("grpc listen: %w", err)
			}
			s.lis = ln

			// Build server options
			var opts []grpc.ServerOption
			if s.cfg.MaxRecvMsgSize > 0 {
				opts = append(opts, grpc.MaxRecvMsgSize(s.cfg.MaxRecvMsgSize))
			}
			if s.cfg.MaxSendMsgSize > 0 {
				opts = append(opts, grpc.MaxSendMsgSize(s.cfg.MaxSendMsgSize))
			}

			// Interceptors: otel, auth, logging, recovery
			opts = append(
				opts,
				grpc.ChainUnaryInterceptor(
					otelgrpc.UnaryServerInterceptor(),
					s.authUnaryInterceptor(),
					s.loggingUnaryInterceptor(),
					s.recoveryUnaryInterceptor(),
				),
				grpc.ChainStreamInterceptor(
					otelgrpc.StreamServerInterceptor(),
					s.authStreamInterceptor(),
					s.loggingStreamInterceptor(),
					s.recoveryStreamInterceptor(),
				),
			)

			// TLS if configured
			if s.cfg.TLS.Enabled {
				creds, err := loadServerTLS(s.cfg)
				if err != nil {
					return fmt.Errorf("grpc tls: %w", err)
				}
				opts = append(opts, grpc.Creds(creds))
			}

			s.srv = grpc.NewServer(opts...)

			// Health service
			hs := health.NewServer()
			healthpb.RegisterHealthServer(s.srv, hs)

			// Register services
			cfgsvc := edicfg.NewServer(p.Conn, &l, p.Repo)
			configpb.RegisterEDIConfigServiceServer(s.srv, cfgsvc)
			hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

			if s.cfg.Reflection {
				reflection.Register(s.srv)
			}

			go func() {
				l.Info().Str("listen", s.cfg.ListenAddress).Msg("gRPC server started")
				if err := s.srv.Serve(ln); err != nil {
					l.Error().Err(err).Msg("gRPC server exited")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if s.srv == nil {
				return nil
			}
			done := make(chan struct{})
			go func() {
				s.srv.GracefulStop()
				close(done)
			}()
			select {
			case <-done:
				s.logger.Info().Msg("gRPC server stopped")
			case <-time.After(5 * time.Second):
				s.logger.Warn().Msg("gRPC graceful stop timeout; forcing stop")
				s.srv.Stop()
			}
			if s.lis != nil {
				_ = s.lis.Close()
			}
			return nil
		},
	})

	return s, nil
}

func loadServerTLS(cfg *config.GRPCServerConfig) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		return nil, err
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	if cfg.TLS.ClientCAFile != "" {
		caPool := x509.NewCertPool()
		if caBytes, err := os.ReadFile(cfg.TLS.ClientCAFile); err == nil {
			_ = caPool.AppendCertsFromPEM(caBytes)
			tlsCfg.ClientCAs = caPool
		}
	}
	if cfg.TLS.RequireClientCert {
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return credentials.NewTLS(tlsCfg), nil
}

// Basic logging interceptor with request metadata.
func (s *GRPCServer) loggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		md, _ := metadata.FromIncomingContext(ctx)
		s.logger.Debug().
			Str("method", info.FullMethod).
			Interface("md", md).
			Msg("grpc unary request")
		resp, err := handler(ctx, req)
		dur := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			s.logger.Error().
				Str("method", info.FullMethod).
				Dur("dur", dur).
				Str("code", st.Code().String()).
				Err(err).
				Msg("grpc unary error")
			return resp, err
		}
		s.logger.Debug().Str("method", info.FullMethod).Dur("dur", dur).Msg("grpc unary success")
		return resp, nil
	}
}

func (s *GRPCServer) recoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Str("method", info.FullMethod).
					Interface("panic", r).
					Msg("grpc unary panic recovered")
				err = status.Errorf(13, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

func (s *GRPCServer) loggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		s.logger.Debug().Str("method", info.FullMethod).Msg("grpc stream request")
		return handler(srv, stream)
	}
}

func (s *GRPCServer) recoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Str("method", info.FullMethod).
					Interface("panic", r).
					Msg("grpc stream panic recovered")
				err = status.Errorf(13, "internal server error")
			}
		}()
		return handler(srv, stream)
	}
}

// authUnaryInterceptor enforces token or API key auth when enabled.
func (s *GRPCServer) authUnaryInterceptor() grpc.UnaryServerInterceptor {
	enabled := s.cfg.Auth.Enabled
	tokens := make(map[string]struct{})
	keys := make(map[string]struct{})
	for _, t := range s.cfg.Auth.BearerTokens {
		tokens[t] = struct{}{}
	}
	for _, k := range s.cfg.Auth.APIKeys {
		keys[k] = struct{}{}
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !enabled {
			return handler(ctx, req)
		}
		if s.authorize(ctx, tokens, keys) {
			return handler(ctx, req)
		}
		return nil, status.Errorf(16, "unauthenticated")
	}
}

func (s *GRPCServer) authStreamInterceptor() grpc.StreamServerInterceptor {
	enabled := s.cfg.Auth.Enabled
	tokens := make(map[string]struct{})
	keys := make(map[string]struct{})
	for _, t := range s.cfg.Auth.BearerTokens {
		tokens[t] = struct{}{}
	}
	for _, k := range s.cfg.Auth.APIKeys {
		keys[k] = struct{}{}
	}
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !enabled {
			return handler(srv, stream)
		}
		if s.authorize(stream.Context(), tokens, keys) {
			return handler(srv, stream)
		}
		return status.Errorf(16, "unauthenticated")
	}
}

func (s *GRPCServer) authorize(ctx context.Context, tokens, keys map[string]struct{}) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	if auths := md.Get("authorization"); len(auths) > 0 {
		for _, a := range auths {
			// Expect Bearer <token>
			if len(a) > 7 && (a[:7] == "Bearer " || a[:7] == "bearer ") {
				if _, ok := tokens[a[7:]]; ok {
					return true
				}
			}
		}
	}
	if apis := md.Get("x-api-key"); len(apis) > 0 {
		for _, k := range apis {
			if _, ok := keys[k]; ok {
				return true
			}
		}
	}
	return false
}
