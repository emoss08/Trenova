package seqgen

import (
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/internal/pkg/seqgen/adapters"
	"go.uber.org/fx"
)

type GeneratorParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

var Module = fx.Module("seqgen",
	fx.Provide(
		// * Provide the sequence store
		seqgen.NewSequenceStore,

		// * Provide the format provider - using the adapter for pro numbers
		adapters.NewProNumberFormatProvider,

		// * Provide the main generator that can be used for all sequence types
		seqgen.NewGenerator,
	),
)
