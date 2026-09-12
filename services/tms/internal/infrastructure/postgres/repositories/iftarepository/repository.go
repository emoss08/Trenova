package iftarepository

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                *postgres.Connection
	Logger            *zap.Logger
	JurisdictionCache repositories.IFTAJurisdictionCacheRepository
}

type repository struct {
	db                *postgres.Connection
	l                 *zap.Logger
	jurisdictionCache repositories.IFTAJurisdictionCacheRepository
}

func New(p Params) repositories.IFTARepository {
	return &repository{
		db:                p.DB,
		l:                 p.Logger.Named("postgres.ifta-repository"),
		jurisdictionCache: p.JurisdictionCache,
	}
}
