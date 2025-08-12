package grpc

import (
	"github.com/emoss08/trenova/internal/core/services/edi"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"grpc-server",
	edi.Module,
	fx.Invoke(New),
)
