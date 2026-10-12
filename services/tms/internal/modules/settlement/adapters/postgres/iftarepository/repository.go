package iftarepository

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/iftaports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                *postgres.Connection
	Logger            *zap.Logger
	JurisdictionCache iftaports.IFTAJurisdictionCacheRepository
}

type repository struct {
	db                *postgres.Connection
	l                 *zap.Logger
	jurisdictionCache iftaports.IFTAJurisdictionCacheRepository
}

func New(p Params) iftaports.IFTARepository {
	return &repository{
		db:                p.DB,
		l:                 p.Logger.Named("postgres.ifta-repository"),
		jurisdictionCache: p.JurisdictionCache,
	}
}
