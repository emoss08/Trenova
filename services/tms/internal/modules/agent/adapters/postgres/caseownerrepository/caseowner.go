package caseownerrepository

import (
	"context"
	"errors"
	"fmt"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/shared/pulid"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner/caseownerdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxAway = 50

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) caseownerports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.case-owner-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req caseownerports.RecordRequest,
) (*caseowner.Owner, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*caseowner.Owner, error) {
		o := caseownerdb.Owners
		owner, err := o.Where(o.RecordID.Eq(req.RecordID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // an unowned case has no owner row
			}

			return nil, fmt.Errorf("get case owner: %w", err)
		}

		return owner, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req caseownerports.RecordsRequest,
) ([]*caseowner.Owner, error) {
	if len(req.RecordIDs) == 0 {
		return []*caseowner.Owner{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*caseowner.Owner, error) {
		o := caseownerdb.Owners
		owners, err := o.Where(o.RecordID.In(req.RecordIDs...)).
			Limit(len(req.RecordIDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list case owners: %w", err)
		}

		return owners, nil
	})
}

func ownerOf(req *caseownerports.SetOwnerRequest) *caseowner.Owner {
	return &caseowner.Owner{
		RecordKind:   req.Kind,
		RecordID:     req.RecordID,
		OwnerID:      req.OwnerID,
		Source:       req.Source,
		AssignedByID: req.ByID,
		AssignedAt:   req.Now,
	}
}

func (r *repository) Set(
	ctx context.Context,
	req *caseownerports.SetOwnerRequest,
) (*caseowner.Owner, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*caseowner.Owner, bool, error) {
		o := caseownerdb.Owners
		if req.Expected == nil {
			owner := ownerOf(req)
			created, err := o.CreateOrSkip(ctx, r.db.Keel(ctx), []*caseowner.Owner{owner},
				o.OrganizationID, o.BusinessUnitID, o.RecordID)
			if err != nil {
				return nil, false, r.setFailed(req, fmt.Errorf("give a case its owner: %w", err))
			}

			return owner, created > 0, nil
		}

		owner := *req.Expected
		owner.OwnerID = req.OwnerID
		owner.Source = req.Source
		owner.AssignedByID = req.ByID
		owner.AssignedAt = req.Now
		err := o.Save(ctx, r.db.Keel(ctx), &owner,
			o.OwnerID, o.Source, o.AssignedByID, o.AssignedAt)
		switch {
		case errors.Is(err, keel.ErrConflict), errors.Is(err, keel.ErrNotFound):
			return nil, false, nil
		case err != nil:
			return nil, false, r.setFailed(
				req,
				fmt.Errorf("give a case to another owner: %w", err),
			)
		}

		return &owner, true, nil
	})
}

func (r *repository) setFailed(req *caseownerports.SetOwnerRequest, err error) error {
	r.l.Error("failed to set a case's owner",
		zap.String("recordId", req.RecordID.String()), zap.Error(err))

	return err
}

func (r *repository) SetIfUnowned(
	ctx context.Context,
	req *caseownerports.SetOwnerRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		o := caseownerdb.Owners
		if _, err := o.CreateOrSkip(ctx, r.db.Keel(ctx), []*caseowner.Owner{ownerOf(req)},
			o.OrganizationID, o.BusinessUnitID, o.RecordID); err != nil {
			return fmt.Errorf("route a case to its owner: %w", err)
		}

		return nil
	})
}

func (r *repository) Release(
	ctx context.Context,
	req caseownerports.RecordRequest,
	expected *caseowner.Owner,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		o := caseownerdb.Owners
		n, err := o.Where(
			o.RecordID.Eq(req.RecordID),
			o.OwnerID.Eq(expected.OwnerID),
			o.Version.Eq(expected.Version),
		).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("release a case: %w", err)
		}

		return n > 0, nil
	})
}

func awayNow(now int64) keel.Pred {
	u := tenantdb.Users

	return keel.And(
		u.AvailabilityStatus.Eq(tenant.AvailabilityAway),
		keel.Or(u.AwayBackOn.IsNull(), u.AwayBackOn.Gt(now)),
	)
}

func (r *repository) AwayFor(
	ctx context.Context,
	req caseownerports.AwayRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		u := tenantdb.Users
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), u.Pick(u.ID).Where(
			u.AwayDelegateUserID.Eq(req.CoverID),
			awayNow(req.Now),
		).Limit(maxAway))
		if err != nil {
			return nil, fmt.Errorf("read who a person covers for: %w", err)
		}

		return ids, nil
	})
}

func (r *repository) Uncovered(
	ctx context.Context,
	req caseownerports.AwayRequest,
) ([]pulid.ID, error) {
	if len(req.UserIDs) == 0 {
		return []pulid.ID{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		u := tenantdb.Users
		ids, err := keel.Into[pulid.ID](ctx, r.db.Keel(ctx), u.Pick(u.ID).Where(
			u.ID.In(req.UserIDs...),
			u.AwayDelegateUserID.IsNull(),
			awayNow(req.Now),
		).Limit(len(req.UserIDs)))
		if err != nil {
			return nil, fmt.Errorf("read who is away with no cover: %w", err)
		}

		return ids, nil
	})
}
