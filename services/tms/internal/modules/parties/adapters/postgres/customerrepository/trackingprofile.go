package customerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"
	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile/trackingprofiledb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type trackingProfileReader struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewTrackingProfileReader(p Params) customerports.TrackingProfileReader {
	return &trackingProfileReader{
		db: p.DB,
		l:  p.Logger.Named("postgres.customer-tracking-profile-reader"),
	}
}

func (r *trackingProfileReader) GetByCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID pulid.ID,
) (*trackingprofile.Profile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*trackingprofile.Profile, error) {
		p := trackingprofiledb.Profiles
		entity, err := p.Where(p.CustomerID.Eq(customerID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return trackingprofile.NewDefault(tenantInfo.OrgID, tenantInfo.BuID, customerID), nil
			}
			r.l.Error("failed to get tracking profile", zap.Error(err))
			return nil, fmt.Errorf("get customer tracking profile: %w", err)
		}
		return entity, nil
	})
}
