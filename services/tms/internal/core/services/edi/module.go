package edi

import (
	"github.com/emoss08/trenova/internal/core/services/edi/partnerconfig"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"edi",
	partnerconfig.Module,
)
