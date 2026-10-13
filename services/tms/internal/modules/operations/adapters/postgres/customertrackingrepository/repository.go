package customertrackingrepository

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultLinkPurgeLimit = 500
	maxShipmentLinks      = 100
	maxShipmentLedger     = 200
	defaultDueLimit       = 100
	maxDueLimit           = 500
	defaultCandidateLimit = 2000
	maxCandidateLimit     = 5000
	maxCustomerBatch      = 500
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}
