package modeprofilerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile/modeprofiledb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/modeprofileports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var deviationErrors = dberror.Mapping{Entity: "Deviation"}

type ledgerRow struct {
	RuleKey        modeprofile.RuleKey
	Capability     modeprofile.Capability
	Occurrences    int64
	OpenCount      int64
	LastOccurredAt int64
}

type DeviationParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type deviationRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDeviationRepository(p DeviationParams) modeprofileports.DeviationRepository {
	return &deviationRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.mode-profile-deviation-repository"),
	}
}

func (r *deviationRepository) RecordMany(
	ctx context.Context,
	deviations []*modeprofile.Deviation,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(deviations) == 0 {
			return nil
		}

		if err := modeprofiledb.Deviations.CreateMany(ctx, r.db.Keel(ctx), deviations); err != nil {
			r.l.Error(
				"failed to record deviations",
				zap.Int("count", len(deviations)),
				zap.Error(err),
			)
			return deviationErrors.Map(err, "")
		}

		return nil
	})
}

func (r *deviationRepository) List(
	ctx context.Context,
	req *modeprofileports.ListDeviationsRequest,
) (*pagination.ListResult[*modeprofile.Deviation], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*modeprofile.Deviation], error) {
			d := modeprofiledb.Deviations
			q := d.Where(
				keel.When(req.ResourceType != "", d.ResourceType.Eq(req.ResourceType)),
				keel.When(req.ResourceID.IsNotNil(), d.ResourceID.Eq(req.ResourceID)),
				keel.When(req.RuleKey != "", d.RuleKey.Eq(modeprofile.RuleKey(req.RuleKey))),
				keel.When(req.State != "", d.State.Eq(modeprofile.DeviationState(req.State))),
			)

			result, err := listquery.List(ctx, r.db.Keel(ctx), q,
				&listquery.Request[modeprofile.Deviation]{
					Filter: req.Filter,
					Shape: func(q *keel.Query[modeprofile.Deviation]) *keel.Query[modeprofile.Deviation] {
						return q.Load(d.Profile(), d.AcknowledgedBy())
					},
					Then: []keel.Order{d.OccurredAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list deviations", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *deviationRepository) GetByID(
	ctx context.Context,
	req *modeprofileports.GetDeviationByIDRequest,
) (*modeprofile.Deviation, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*modeprofile.Deviation, error) {
		d := modeprofiledb.Deviations
		entity, err := d.Where(d.ID.Eq(req.DeviationID)).
			Load(d.Profile()).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, deviationErrors.Map(err, req.DeviationID.String())
		}

		return entity, nil
	})
}

func (r *deviationRepository) Acknowledge(
	ctx context.Context,
	req *modeprofileports.AcknowledgeDeviationRequest,
) (*modeprofile.Deviation, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*modeprofile.Deviation, error) {
		entity, err := r.GetByID(ctx, &modeprofileports.GetDeviationByIDRequest{
			DeviationID: req.DeviationID,
			TenantInfo:  req.TenantInfo,
		})
		if err != nil {
			return nil, err
		}

		now := timeutils.NowUnix()
		acknowledgedBy := req.AcknowledgedByID

		entity.State = modeprofile.DeviationStateAcknowledged
		entity.AcknowledgedByID = &acknowledgedBy
		entity.AcknowledgedAt = &now
		entity.AcknowledgementReason = req.Reason

		d := modeprofiledb.Deviations
		err = d.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			d.State,
			d.AcknowledgedByID,
			d.AcknowledgedAt,
			d.AcknowledgementReason,
		)
		if err != nil {
			r.l.Error(
				"failed to acknowledge deviation",
				zap.String("id", req.DeviationID.String()),
				zap.Error(err),
			)
			return nil, deviationErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *deviationRepository) SupersedeOpenForResource(
	ctx context.Context,
	req *modeprofileports.SupersedeDeviationsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := modeprofiledb.Deviations
		keep := make([]modeprofile.RuleKey, 0, len(req.KeepRuleKeys))
		for _, key := range req.KeepRuleKeys {
			keep = append(keep, modeprofile.RuleKey(key))
		}

		_, err := d.Where(
			d.ResourceType.Eq(req.ResourceType),
			d.ResourceID.Eq(req.ResourceID),
			d.State.Eq(modeprofile.DeviationStateOpen),
			keel.When(len(keep) > 0, d.RuleKey.NotIn(keep...)),
		).Update(ctx, r.db.Keel(ctx), d.State.To(modeprofile.DeviationStateResolved))
		if err != nil {
			r.l.Error(
				"failed to supersede open deviations",
				zap.String("resourceId", req.ResourceID.String()),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}

func (r *deviationRepository) LiveRuleKeysForResource(
	ctx context.Context,
	req *modeprofileports.LiveDeviationKeysRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		d := modeprofiledb.Deviations
		ruleKeys, err := keel.Into[modeprofile.RuleKey](ctx, r.db.Keel(ctx), d.
			Group(d.RuleKey).
			Pick(d.RuleKey).
			Where(
				d.ResourceType.Eq(req.ResourceType),
				d.ResourceID.Eq(req.ResourceID),
				d.State.In(modeprofile.DeviationStateOpen, modeprofile.DeviationStateAcknowledged),
			))
		if err != nil {
			r.l.Error(
				"failed to read live deviation rule keys",
				zap.String("resourceId", req.ResourceID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		keys := make([]string, 0, len(ruleKeys))
		for _, key := range ruleKeys {
			keys = append(keys, string(key))
		}

		return keys, nil
	})
}

func (r *deviationRepository) Ledger(
	ctx context.Context,
	req *modeprofileports.DeviationLedgerRequest,
) ([]*modeprofileports.DeviationLedgerEntry, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*modeprofileports.DeviationLedgerEntry, error) {
			d := modeprofiledb.Deviations
			rows, err := keel.Into[ledgerRow](ctx, r.db.Keel(ctx), d.
				Group(d.RuleKey, d.Capability).
				Pick(
					d.RuleKey,
					d.Capability,
					keel.Count(),
					keel.Count().Filter(d.State.Eq(modeprofile.DeviationStateOpen)),
					d.OccurredAt.Max(),
				).
				Where(keel.When(req.Since > 0, d.OccurredAt.Gte(req.Since))).
				Order(keel.Count().Desc()))
			if err != nil {
				r.l.Error("failed to build deviation ledger", zap.Error(err))
				return nil, err
			}

			entries := make([]*modeprofileports.DeviationLedgerEntry, 0, len(rows))
			for _, row := range rows {
				label := string(row.RuleKey)
				if def, defErr := modeprofile.RuleDefinitionFor(row.RuleKey); defErr == nil {
					label = def.Label
				}
				entries = append(entries, &modeprofileports.DeviationLedgerEntry{
					RuleKey:        string(row.RuleKey),
					Capability:     string(row.Capability),
					Label:          label,
					Occurrences:    row.Occurrences,
					OpenCount:      row.OpenCount,
					LastOccurredAt: row.LastOccurredAt,
				})
			}

			return entries, nil
		},
	)
}
