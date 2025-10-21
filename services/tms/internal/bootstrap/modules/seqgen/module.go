package seqgen

import (
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
)

var Module = fx.Module("seqgen", fx.Provide(
	seqgen.NewFormatProvider,
	seqgen.NewSequenceStore,
	seqgen.NewGenerator,
))
