package grpcmod

import (
	grpcserver "github.com/emoss08/trenova/internal/infrastructure/grpc/server"
	"go.uber.org/fx"
)

var Module = fx.Module("grpc",
	fx.Provide(grpcserver.New),
)
