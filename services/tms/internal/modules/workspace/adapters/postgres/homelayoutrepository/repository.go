package homelayoutrepository

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/homelayoutports"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) homelayoutports.HomeLayoutRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.home-layout-repository"),
	}
}
