package seqgen

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig/sequenceconfigdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/resultcache"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FormatProviderParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type formatProvider struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFormatProvider(p FormatProviderParams) FormatProvider {
	return &formatProvider{
		db: p.DB,
		l:  p.Logger.Named("seq-format-provider"),
	}
}

func (p *formatProvider) GetFormat(
	ctx context.Context,
	sequenceType sequenceconfig.SequenceType,
	orgID, buID pulid.ID,
) (*sequenceconfig.SequenceFormat, error) {
	var cfg *sequenceconfig.SequenceConfig
	err := p.db.RunDetached(ctx, true, func(ctx context.Context) error {
		sc := sequenceconfigdb.SequenceConfigs
		var err error
		cfg, err = sc.Where(
			sc.SequenceType.Eq(sequenceType),
			sc.OrganizationID.Eq(orgID),
			sc.BusinessUnitID.Eq(buID),
		).Cache(resultcache.Settings()...).First(ctx, p.db.Keel(ctx))
		return err
	})
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return sequenceconfig.DefaultSequenceFormat(sequenceType)
		}
		return nil, fmt.Errorf("find sequence config: %w", err)
	}

	return cfg.Format(), nil
}
