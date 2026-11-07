package auditintegrity

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Entries   auditports.AuditRepository
	Seals     auditports.SealRepository
	Retention tenantports.DataRetentionRepository
	Config    *aiauditconfig.Config
	Metrics   *metrics.Registry
	Logger    *zap.Logger
}

type Components struct {
	fx.Out

	Keyring  *Keyring
	Sealer   *Sealer
	Verifier *Verifier
	Status   *Status
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) Components {
	keyring := NewKeyring(p.Config)
	return Components{
		Keyring: keyring,
		Sealer: NewSealer(&SealerParams{
			Entries: p.Entries,
			Seals:   p.Seals,
			Keyring: keyring,
			Metrics: p.Metrics,
			Logger:  p.Logger,
		}),
		Verifier: NewVerifier(&VerifierParams{
			Seals:     p.Seals,
			Retention: p.Retention,
			Keyring:   keyring,
			Metrics:   p.Metrics,
			Logger:    p.Logger,
		}),
		Status: NewStatus(p.Seals, keyring),
	}
}
