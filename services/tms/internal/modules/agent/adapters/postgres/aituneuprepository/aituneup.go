package aituneuprepository

import (
	"context"
	"fmt"
	"github.com/emoss08/trenova/pkg/listquery"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aituneup"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aituneup/aituneupdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aituneupports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const entityName = "AITuneUp"

var tuneUpErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB *postgres.Connection
}

type repository struct {
	db *postgres.Connection
}

func New(p Params) aituneupports.AITuneUpRepository {
	return &repository{db: p.DB}
}

func (r *repository) List(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*aituneup.TuneUp, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aituneup.TuneUp, error) {
		t := aituneupdb.TuneUps
		tuneUps, err := t.Order(t.CreatedAt.Asc(), t.ID.Asc()).All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list ai tune-ups: %w", err)
		}
		if tuneUps == nil {
			tuneUps = []*aituneup.TuneUp{}
		}
		return tuneUps, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req aituneupports.GetAITuneUpRequest,
) (*aituneup.TuneUp, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aituneup.TuneUp, error) {
		entity, err := aituneupdb.TuneUps.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, tuneUpErrors.Map(err, req.ID.String())
		}
		return entity, nil
	})
}

func (r *repository) Save(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	plan *aituneup.Plan,
) error {
	if plan.Empty() {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		if len(plan.Delete) > 0 {
			if err := deleteTuneUps(ctx, db, plan.Delete); err != nil {
				return err
			}
		}
		for _, tuneUp := range plan.Update {
			if err := refresh(ctx, db, tuneUp); err != nil {
				return err
			}
		}
		if len(plan.Insert) > 0 {
			for _, tuneUp := range plan.Insert {
				tuneUp.OrganizationID = tenantInfo.OrgID
				tuneUp.BusinessUnitID = tenantInfo.BuID
			}
			t := aituneupdb.TuneUps
			if _, err := t.CreateOrSkip(ctx, db, plan.Insert,
				t.OrganizationID, t.BusinessUnitID, t.Fingerprint); err != nil {
				return fmt.Errorf("insert ai tune-ups: %w", err)
			}
		}
		return nil
	})
}

func deleteTuneUps(ctx context.Context, db *keel.DB, ids []pulid.ID) error {
	t := aituneupdb.TuneUps
	if _, err := t.Where(t.ID.In(ids...)).Delete(ctx, db); err != nil {
		return fmt.Errorf("delete ai tune-ups: %w", err)
	}
	return nil
}

func refresh(ctx context.Context, db *keel.DB, tuneUp *aituneup.TuneUp) error {
	t := aituneupdb.TuneUps
	_, err := t.Where(t.ID.Eq(tuneUp.ID), t.Version.Eq(tuneUp.Version)).Update(ctx, db,
		t.AgentDefinitionID.To(tuneUp.AgentDefinitionID),
		t.ProviderID.To(tuneUp.ProviderID),
		t.OtherProviderID.To(tuneUp.OtherProviderID),
		t.ToolName.To(tuneUp.ToolName),
		t.Task.To(tuneUp.Task),
		t.Evidence.To(tuneUp.Evidence),
		t.Status.To(tuneUp.Status),
		listquery.OptionalInt64(t.DismissedUntil, tuneUp.DismissedUntil),
		t.DecidedByID.To(tuneUp.DecidedByID),
		listquery.OptionalInt64(t.DecidedAt, tuneUp.DecidedAt),
		t.ComputedAt.To(tuneUp.ComputedAt),
	)
	if err != nil {
		return fmt.Errorf("refresh ai tune-up: %w", err)
	}
	return nil
}

func (r *repository) Decide(
	ctx context.Context,
	req *aituneupports.DecideAITuneUpRequest,
) (*aituneup.TuneUp, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aituneup.TuneUp, error) {
		t := aituneupdb.TuneUps
		updated, err := t.Where(t.ID.Eq(req.ID), t.Version.Eq(req.Version)).
			UpdateReturning(ctx, r.db.Keel(ctx),
				t.Status.To(req.Status),
				listquery.OptionalInt64(t.DismissedUntil, req.DismissedUntil),
				t.DecidedByID.To(req.DecidedByID),
				t.DecidedAt.To(req.DecidedAt),
			)
		if err != nil {
			return nil, fmt.Errorf("decide ai tune-up: %w", err)
		}
		if len(updated) == 0 {
			return nil, dberror.CreateVersionMismatchError(entityName, req.ID.String())
		}
		return updated[0], nil
	})
}

func (r *repository) Reopen(
	ctx context.Context,
	req *aituneupports.ReopenAITuneUpRequest,
) (*aituneup.TuneUp, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aituneup.TuneUp, error) {
		t := aituneupdb.TuneUps
		updated, err := t.Where(
			t.ID.Eq(req.ID),
			t.Version.Eq(req.Version),
			t.Status.Eq(aituneup.StatusDismissed),
		).UpdateReturning(ctx, r.db.Keel(ctx),
			t.Status.To(aituneup.StatusOpen),
			t.DismissedUntil.ToNull(),
			t.DecidedByID.ToNull(),
			t.DecidedAt.ToNull(),
		)
		if err != nil {
			return nil, fmt.Errorf("reopen ai tune-up: %w", err)
		}
		if len(updated) == 0 {
			return nil, dberror.CreateVersionMismatchError(entityName, req.ID.String())
		}
		return updated[0], nil
	})
}
