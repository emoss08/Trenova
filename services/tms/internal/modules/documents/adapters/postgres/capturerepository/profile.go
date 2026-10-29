package capturerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

var profileErrors = dberror.Mapping{Entity: "Capture profile"}

type profileRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewProfileRepository(p Params) captureports.CaptureProfileRepository {
	return &profileRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-profile-repository"),
	}
}

func (r *profileRepository) List(
	ctx context.Context,
	req *captureports.ListCaptureProfilesRequest,
) (*pagination.ListResult[*captureprofile.CaptureProfile], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*captureprofile.CaptureProfile], error) {
			p := capturedb.CaptureProfiles
			return listquery.List(
				ctx,
				r.db.Keel(ctx),
				p.Where(keel.When(req.Status != "", p.Status.Eq(req.Status))),
				&listquery.Request[captureprofile.CaptureProfile]{
					Filter: req.Filter,
					Then:   []keel.Order{p.IsDefault.Desc(), p.Name.Asc()},
				},
			)
		},
	)
}

func (r *profileRepository) GetByID(
	ctx context.Context,
	req captureports.GetCaptureProfileByIDRequest,
) (*captureprofile.CaptureProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*captureprofile.CaptureProfile, error) {
		entity, err := capturedb.CaptureProfiles.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, profileErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *profileRepository) GetDefault(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*captureprofile.CaptureProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*captureprofile.CaptureProfile, error) {
		p := capturedb.CaptureProfiles
		entity, err := p.Where(p.IsDefault.IsTrue(), p.Status.Eq(captureprofile.ProfileActive)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *profileRepository) Create(
	ctx context.Context,
	entity *captureprofile.CaptureProfile,
) (*captureprofile.CaptureProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*captureprofile.CaptureProfile, error) {
		db := r.db.Keel(ctx)
		if entity.IsDefault {
			if err := clearDefault(ctx, db, entity); err != nil {
				return nil, err
			}
		}

		if err := capturedb.CaptureProfiles.Create(ctx, db, entity); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *profileRepository) Update(
	ctx context.Context,
	entity *captureprofile.CaptureProfile,
) (*captureprofile.CaptureProfile, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*captureprofile.CaptureProfile, error) {
		db := r.db.Keel(ctx)
		if entity.IsDefault {
			if err := clearDefault(ctx, db, entity); err != nil {
				return nil, err
			}
		}

		if err := capturedb.CaptureProfiles.Save(ctx, db, entity); err != nil {
			return nil, profileErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

// clearDefault unsets the flag on every other profile in the tenant. The
// partial unique index would refuse the write otherwise, and failing the
// save because another profile is the default would make "set as default"
// a two-step chore.
func clearDefault(ctx context.Context, db *keel.DB, entity *captureprofile.CaptureProfile) error {
	p := capturedb.CaptureProfiles
	_, err := p.Where(
		p.IsDefault.IsTrue(),
		keel.When(entity.ID.IsNotNil(), p.ID.Ne(entity.ID)),
	).Update(ctx, db, p.IsDefault.To(false))

	return err
}

func (r *profileRepository) Delete(
	ctx context.Context,
	req captureports.DeleteCaptureProfileRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := capturedb.CaptureProfiles
		n, err := p.Where(p.ID.Eq(req.ID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return err
		}

		return dberror.CheckAffected(n, "Capture profile", req.ID.String())
	})
}
