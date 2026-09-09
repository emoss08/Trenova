package fuelpurchaserepository

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultActiveCardLimit = 50
	maxActiveCardLimit     = 200
	rowInsertBatch         = 500
	purchaseInsertBatch    = 500
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.FuelPurchaseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.fuel-purchase-repository"),
	}
}

func limitOr(requested, fallback, ceiling int) int {
	if requested > 0 && requested <= ceiling {
		return requested
	}
	return fallback
}

func duplicateCard() error {
	return errortypes.NewValidationError(
		"lastFour",
		errortypes.ErrDuplicate,
		"A card from this provider with these last four digits already exists",
	)
}

func duplicateReference() error {
	return errortypes.NewValidationError(
		"transactionReference",
		errortypes.ErrDuplicate,
		"A purchase with this transaction reference already exists",
	)
}
