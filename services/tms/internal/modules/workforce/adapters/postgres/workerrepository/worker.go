package workerrepository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/app/quotaservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driverpay/driverpaydb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment/payassignmentdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential/workercredentialdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersyncdrift"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersyncdrift/workersyncdriftdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	workerBusyMessage    = "Worker is busy. Retry the request."
	syncDriftBusyMessage = "Worker sync state is busy. Retry the request."
)

var workerErrors = dberror.Mapping{Entity: "Worker"}

var workerProfileErrors = dberror.Mapping{Entity: "WorkerProfile"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
	Quota  platformplanports.QuotaGuard `optional:"true"`
}

type repository struct {
	db    *postgres.Connection
	l     *zap.Logger
	quota platformplanports.QuotaGuard
}

func New(p Params) workerports.WorkerRepository {
	return &repository{
		db:    p.DB,
		l:     p.Logger.Named("postgres.worker-repository"),
		quota: quotaservice.OrUnlimited(p.Quota),
	}
}

func ownerOperatorFilter(now int64) keel.Pred {
	w := workerdb.Workers

	wpa := payassignmentdb.WorkerPayAssignments.As("wpa")
	dpp := driverpaydb.PayProfiles.As("dpp")

	return keel.Or(
		w.Type.Eq(worker.WorkerTypeContractor),
		keel.Exists(wpa.
			Join(
				dpp,
				dpp.ID.EqCol(wpa.PayProfileID),
				dpp.OrganizationID.EqCol(wpa.OrganizationID),
				dpp.BusinessUnitID.EqCol(wpa.BusinessUnitID),
			).
			Where(
				wpa.WorkerID.EqCol(w.ID),
				wpa.OrganizationID.EqCol(w.OrganizationID),
				wpa.BusinessUnitID.EqCol(w.BusinessUnitID),
				wpa.EffectiveFrom.Lte(now),
				keel.Or(wpa.EffectiveTo.IsNull(), wpa.EffectiveTo.Gt(now)),
				dpp.Classification.Eq(driverpay.PayeeClassificationOwnerOperator),
			)),
	)
}

func selectOptionsQuery(ownerOperatorsOnly bool, now int64) *keel.Query[*worker.Worker] {
	w := workerdb.Workers

	return w.Pick(w.ID, w.CreatedAt, w.FirstName, w.LastName, w.FleetCodeID, w.Status).
		Where(
			w.Status.Eq(domaintypes.StatusActive),
			keel.When(ownerOperatorsOnly, ownerOperatorFilter(now)),
		).
		Load(w.FleetCode())
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *workerports.WorkerSelectOptionsRequest,
) (*pagination.ListResult[*worker.Worker], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*worker.Worker], error) {
			w := workerdb.Workers

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				selectOptionsQuery(req.OwnerOperatorsOnly, timeutils.NowUnix()),
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(w.FirstName.ContainsFold(s), w.LastName.ContainsFold(s))
				},
			)
		},
	)
}

func (r *repository) List(
	ctx context.Context,
	req *workerports.ListWorkersRequest,
) (*pagination.CursorListResult[*worker.Worker], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*worker.Worker], error) {
			w := workerdb.Workers
			q := w.Query()
			if req.WithoutCredential != "" {
				typeID, err := r.credentialTypeID(ctx, req.WithoutCredential)
				if err != nil {
					return nil, err
				}
				q = q.Where(keel.NotExists(validCredential(typeID, timeutils.NowUnix())))
			}
			result, err := listquery.Connection(ctx, r.db.Keel(ctx), q,
				&listquery.Request[worker.Worker]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(q *keel.Query[*worker.Worker]) *keel.Query[*worker.Worker] {
						return q.Load(w.FleetCode(), w.State(), w.Profile())
					},
				})
			if err != nil {
				r.l.Error("failed to list workers", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

// credentialTypeID is the id of the credential type a code names, in any
// case. A code that names none is refused rather than read as every worker
// lacking it.
func (r *repository) credentialTypeID(ctx context.Context, code string) (pulid.ID, error) {
	t := workercredentialdb.WorkerCredentialTypes
	kind, err := t.Where(t.Code.Lower().Eq(strings.ToLower(code))).One(ctx, r.db.Keel(ctx))
	if errors.Is(err, keel.ErrNotFound) {
		return pulid.Nil, errortypes.NewValidationError(
			"withoutCredential",
			errortypes.ErrInvalid,
			fmt.Sprintf("No credential type has the code %s", code),
		)
	}
	if err != nil {
		r.l.Error("failed to read the credential type a worker list narrows by", zap.Error(err))
		return pulid.Nil, err
	}

	return kind.ID, nil
}

// validCredential is a worker's active credential of a type that has not
// expired, through the index on the worker's active credential of each type.
func validCredential(typeID pulid.ID, now int64) *keel.Query[*workercredential.WorkerCredential] {
	c := workercredentialdb.WorkerCredentials

	return c.Where(
		c.WorkerID.EqCol(workerdb.Workers.ID),
		c.CredentialTypeID.Eq(typeID),
		c.Status.Eq(workercredential.CredentialStatusActive),
		keel.Or(c.ExpiresAt.IsNull(), c.ExpiresAt.Gte(now)),
	)
}

func workerQuery(req *workerports.GetWorkerByIDRequest) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	profile := w.Profile()

	loads := make([]keel.Loader, 0, 5)
	switch {
	case req.IncludeProfile && req.IncludeLicenseState:
		loads = append(loads, profile.LicenseState())
	case req.IncludeProfile:
		loads = append(loads, profile)
	}
	loads = append(loads, listquery.When(req.IncludeState, w.State())...)
	loads = append(loads, listquery.When(req.IncludeFleetCode, w.FleetCode())...)
	loads = append(loads, listquery.When(req.IncludeManager, w.Manager())...)

	return w.Where(w.ID.Eq(req.ID)).Load(loads...)
}

func (r *repository) GetByID(
	ctx context.Context,
	req workerports.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*worker.Worker, error) {
		entity, err := workerQuery(&req).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, workerErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *worker.Worker,
) (*worker.Worker, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.Worker, error) {
		if err := r.create(ctx, entity); err != nil {
			r.l.Error(
				"failed to create worker",
				zap.String("firstName", entity.FirstName),
				zap.String("lastName", entity.LastName),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(err, workerBusyMessage)
		}

		return entity, nil
	})
}

func (r *repository) create(ctx context.Context, entity *worker.Worker) error {
	err := quotaservice.EnforceAll(ctx, r.quota, platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		Meter:    platformcatalog.MeterWorkersTotal,
		Quantity: 1,
	})
	if err != nil {
		return err
	}

	db := r.db.Keel(ctx)
	if err = workerdb.Workers.Create(ctx, db, entity); err != nil {
		return workerErrors.Map(err, entity.ID.String())
	}

	if entity.Profile == nil {
		return nil
	}

	entity.Profile.WorkerID = entity.ID
	entity.Profile.OrganizationID = entity.OrganizationID
	entity.Profile.BusinessUnitID = entity.BusinessUnitID
	if err = workerdb.WorkerProfiles.Create(ctx, db, entity.Profile); err != nil {
		return workerProfileErrors.Map(err, entity.Profile.ID.String())
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *worker.Worker,
) (*worker.Worker, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*worker.Worker, error) {
		if err := r.update(ctx, entity); err != nil {
			r.l.Error(
				"failed to update worker",
				zap.String("id", entity.ID.String()),
				zap.Error(err),
			)
			return nil, dberror.MapRetryableTransactionError(err, workerBusyMessage)
		}

		return entity, nil
	})
}

func (r *repository) update(ctx context.Context, entity *worker.Worker) error {
	db := r.db.Keel(ctx)
	if err := workerdb.Workers.Save(ctx, db, entity); err != nil {
		return workerErrors.Map(err, entity.ID.String())
	}

	if entity.Profile == nil {
		return nil
	}

	entity.Profile.WorkerID = entity.ID
	entity.Profile.OrganizationID = entity.OrganizationID
	entity.Profile.BusinessUnitID = entity.BusinessUnitID
	if err := workerdb.WorkerProfiles.Save(ctx, db, entity.Profile); err != nil {
		return workerProfileErrors.Map(err, entity.Profile.ID.String())
	}

	return nil
}

func (r *repository) GetWorkerSyncReadinessCounts(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*workerports.WorkerSyncReadinessCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*workerports.WorkerSyncReadinessCounts, error) {
			w := workerdb.Workers
			active := w.Status.Eq(domaintypes.StatusActive)
			counts, err := keel.IntoOne[workerports.WorkerSyncReadinessCounts](
				ctx,
				r.db.Keel(ctx),
				w.Pick(
					keel.Count(),
					keel.Count().Filter(active),
					keel.Count().Filter(keel.And(
						active,
						w.ExternalID.Trim().NullIf("").IsNotNull(),
					)),
				),
			)
			if err != nil {
				r.l.Error("failed to count worker sync readiness", zap.Error(err))
				return nil, err
			}

			return &counts, nil
		},
	)
}

func (r *repository) ReplaceWorkerSyncDrifts(
	ctx context.Context,
	_ pagination.TenantInfo,
	drifts []workerports.WorkerSyncDriftRecord,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		d := workersyncdriftdb.WorkerSyncDrifts
		if _, err := d.Where(keel.True()).Delete(ctx, db); err != nil {
			return dberror.MapRetryableTransactionError(err, syncDriftBusyMessage)
		}

		if len(drifts) == 0 {
			return nil
		}

		models := make([]*workersyncdrift.WorkerSyncDrift, 0, len(drifts))
		for idx := range drifts {
			drift := drifts[idx]
			models = append(models, &workersyncdrift.WorkerSyncDrift{
				WorkerID:        drift.WorkerID,
				DriftType:       drift.DriftType,
				WorkerName:      drift.WorkerName,
				Message:         drift.Message,
				LocalExternalID: drift.LocalExternalID,
				RemoteDriverID:  drift.RemoteDriverID,
				DetectedAt:      drift.DetectedAt,
			})
		}

		if err := d.CreateMany(ctx, db, models); err != nil {
			return dberror.MapRetryableTransactionError(err, syncDriftBusyMessage)
		}

		return nil
	})
}

func (r *repository) ListWorkerSyncDrifts(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]workerports.WorkerSyncDriftRecord, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]workerports.WorkerSyncDriftRecord, error) {
			d := workersyncdriftdb.WorkerSyncDrifts
			models, err := d.Query().
				Order(d.DetectedAt.Desc(), d.WorkerName.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list worker sync drifts", zap.Error(err))
				return nil, err
			}

			records := make([]workerports.WorkerSyncDriftRecord, 0, len(models))
			for _, model := range models {
				records = append(records, workerports.WorkerSyncDriftRecord{
					WorkerID:        model.WorkerID,
					WorkerName:      model.WorkerName,
					DriftType:       model.DriftType,
					Message:         model.Message,
					LocalExternalID: model.LocalExternalID,
					RemoteDriverID:  model.RemoteDriverID,
					DetectedAt:      model.DetectedAt,
				})
			}

			return records, nil
		},
	)
}

func credentialFieldColumn(field workercredential.CredentialProfileField) (keel.Int64Col, error) {
	wp := workerdb.WorkerProfiles
	switch field {
	case workercredential.CredentialProfileFieldLicenseExpiry:
		return wp.LicenseExpiry, nil
	case workercredential.CredentialProfileFieldHazmatExpiry:
		return wp.HazmatExpiry, nil
	case workercredential.CredentialProfileFieldMedicalCardExpiry:
		return wp.MedicalCardExpiry, nil
	case workercredential.CredentialProfileFieldTWICExpiry:
		return wp.TWICExpiry, nil
	case workercredential.CredentialProfileFieldPhysicalDueDate:
		return wp.PhysicalDueDate, nil
	case workercredential.CredentialProfileFieldMVRDueDate:
		return wp.MVRDueDate, nil
	case workercredential.CredentialProfileFieldNone:
	}

	return keel.Int64Col{}, fmt.Errorf(
		"%w: %s",
		workercredential.ErrInvalidCredentialProfileField,
		field,
	)
}

func credentialFieldAssignments(
	req *workerports.PatchProfileCredentialFieldRequest,
	column keel.Int64Col,
) []keel.Assignment {
	wp := workerdb.WorkerProfiles
	sets := []keel.Assignment{wp.Version.Increment(0)}
	hasExpiry := req.ExpiresAt != nil && *req.ExpiresAt > 0

	if req.Field == workercredential.CredentialProfileFieldLicenseExpiry {
		if hasExpiry {
			sets = append(sets, column.To(*req.ExpiresAt))
		}
		if req.Number != "" {
			sets = append(sets, wp.LicenseNumber.To(req.Number))
		}
		return sets
	}

	if hasExpiry {
		return append(sets, column.To(*req.ExpiresAt))
	}

	return append(sets, column.ToNull())
}

func (r *repository) PatchProfileCredentialField(
	ctx context.Context,
	req *workerports.PatchProfileCredentialFieldRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req.Field == workercredential.CredentialProfileFieldNone {
			return nil
		}

		column, err := credentialFieldColumn(req.Field)
		if err != nil {
			return err
		}

		wp := workerdb.WorkerProfiles
		_, err = wp.Where(wp.WorkerID.Eq(req.WorkerID)).
			Update(ctx, r.db.Keel(ctx), credentialFieldAssignments(req, column)...)
		if err != nil {
			r.l.Error("failed to patch worker profile credential field",
				zap.String("workerId", req.WorkerID.String()),
				zap.String("field", req.Field.String()),
				zap.Error(err))
			return fmt.Errorf("patch worker profile credential field: %w", err)
		}

		return nil
	})
}

func setNullable(col keel.Int64Col, value *int64) keel.Assignment {
	if value == nil {
		return col.ToNull()
	}

	return col.To(*value)
}

func distinctFrom(col keel.Int64Col, value *int64) keel.Pred {
	if value == nil {
		return col.IsNotNull()
	}

	return col.IsDistinctFrom(*value)
}

func (r *repository) UpdateProfileComplianceStatus(
	ctx context.Context,
	req *workerports.UpdateProfileComplianceStatusRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		wp := workerdb.WorkerProfiles
		_, err := wp.Where(
			wp.WorkerID.Eq(req.WorkerID),
			keel.Or(
				wp.ComplianceStatus.Ne(req.Status),
				distinctFrom(wp.NextCredentialExpiry, req.NextExpiry),
			),
		).Update(
			ctx,
			r.db.Keel(ctx),
			wp.ComplianceStatus.To(req.Status),
			setNullable(wp.NextCredentialExpiry, req.NextExpiry),
			wp.Version.Increment(0),
		)
		if err != nil {
			r.l.Error("failed to update worker compliance status",
				zap.String("workerId", req.WorkerID.String()),
				zap.Error(err))
			return fmt.Errorf("update worker compliance status: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateProfileTrainingRollup(
	ctx context.Context,
	req *workerports.UpdateProfileTrainingRollupRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		wp := workerdb.WorkerProfiles
		_, err := wp.Where(
			wp.WorkerID.Eq(req.WorkerID),
			keel.Or(
				wp.TrainingHealth.Ne(req.Health),
				distinctFrom(wp.NextTrainingDue, req.NextDue),
			),
		).Update(
			ctx,
			r.db.Keel(ctx),
			wp.TrainingHealth.To(req.Health),
			setNullable(wp.NextTrainingDue, req.NextDue),
			wp.Version.Increment(0),
		)
		if err != nil {
			r.l.Error("failed to update worker training rollup",
				zap.String("workerId", req.WorkerID.String()),
				zap.Error(err))
			return fmt.Errorf("update worker training rollup: %w", err)
		}

		return nil
	})
}

// UpdateProfileSafetyRollup caches the worker's safety standing on the profile
// for the same reason.
func (r *repository) UpdateProfileSafetyRollup(
	ctx context.Context,
	req *workerports.UpdateProfileSafetyRollupRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		wp := workerdb.WorkerProfiles
		_, err := wp.Where(
			wp.WorkerID.Eq(req.WorkerID),
			keel.Or(wp.SafetyRating.Ne(req.Rating), wp.SafetyScore.Ne(req.Score)),
		).Update(
			ctx,
			r.db.Keel(ctx),
			wp.SafetyRating.To(req.Rating),
			wp.SafetyScore.To(req.Score),
			wp.Version.Increment(0),
		)
		if err != nil {
			r.l.Error("failed to update worker safety rollup",
				zap.String("workerId", req.WorkerID.String()),
				zap.Error(err))
			return fmt.Errorf("update worker safety rollup: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateProfileQualification(
	ctx context.Context,
	req *workerports.UpdateProfileQualificationRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		wp := workerdb.WorkerProfiles
		_, err := wp.Where(
			wp.WorkerID.Eq(req.WorkerID),
			wp.IsQualified.Eq(!req.Qualified),
		).Update(ctx, r.db.Keel(ctx), wp.IsQualified.To(req.Qualified), wp.Version.Increment(0))
		if err != nil {
			r.l.Error("failed to update worker qualification",
				zap.String("workerId", req.WorkerID.String()),
				zap.Error(err))
			return fmt.Errorf("update worker qualification: %w", err)
		}

		return nil
	})
}

// CountRosterAttention answers the home screen's HR tile in one pass over the
// roll-up columns. Counting these by replaying credentials, training records
// and safety events per worker is what those columns were denormalised to
// avoid.
func (r *repository) CountRosterAttention(
	ctx context.Context,
	req *workerports.CountRosterAttentionRequest,
) (*workerports.RosterAttention, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*workerports.RosterAttention, error) {
		counts, err := keel.IntoOne[workerports.RosterAttention](
			ctx,
			r.db.Keel(ctx),
			rosterAttentionQuery(req.ExpiryHorizon),
		)
		if err != nil {
			r.l.Error("failed to count roster attention", zap.Error(err))
			return nil, fmt.Errorf("count roster attention: %w", err)
		}

		return &counts, nil
	})
}

func rosterAttentionQuery(expiryHorizon int64) *keel.Query[*worker.Worker] {
	w := workerdb.Workers
	wp := workerdb.WorkerProfiles

	return w.Join(wp, wp.WorkerID.EqCol(w.ID)).
		Where(w.Status.Eq(domaintypes.StatusActive)).
		Pick(
			keel.Count(),
			keel.Count().Filter(wp.ComplianceStatus.Eq(worker.ComplianceStatusNonCompliant)),
			keel.Count().Filter(wp.TrainingHealth.In(
				worker.TrainingHealthOverdue,
				worker.TrainingHealthExpired,
				worker.TrainingHealthFailed,
				worker.TrainingHealthMissing,
			)),
			keel.Count().Filter(wp.SafetyRating.Eq(worker.SafetyRatingAtRisk)),
			keel.Count().Filter(keel.And(
				wp.NextCredentialExpiry.IsNotNull(),
				wp.NextCredentialExpiry.Lte(expiryHorizon),
			)),
		)
}
