package filestorage

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ClientParams struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

func NewClient(p ClientParams) (*minio.Client, error) {
	mc, err := minio.New(p.Config.Storage.Endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(
			p.Config.Storage.AccessKey,
			p.Config.Storage.SecretKey,
			p.Config.Storage.SessionToken,
		),
		Secure: p.Config.Storage.UseSSL,
	})
	if err != nil {
		p.Logger.Error("failed to create minio client", zap.Error(err))
		return nil, err
	}

	return mc, nil
}
