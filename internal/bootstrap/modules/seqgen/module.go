/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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

		// * Provide the unified format provider for all sequence types
		fx.Annotate(
			adapters.NewUnifiedFormatProvider,
			fx.As(new(seqgen.FormatProvider)),
		),

		// * Provide the main generator that can be used for all sequence types
		seqgen.NewGenerator,
	),
)
