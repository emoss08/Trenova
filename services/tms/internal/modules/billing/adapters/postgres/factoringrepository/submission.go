package factoringrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	submissionEntity          = "FactoringSubmission"
	liveInvoiceIndex          = "uq_factoring_submission_items_live_invoice"
	submissionIdempotencyIdx  = "uq_factoring_submissions_idempotency"
	submissionNumberIndex     = "uq_factoring_submissions_number"
	maxSubmissionIDsPerLookup = 1000
)

var submissionErrors = dberror.Mapping{Entity: submissionEntity}

type submissionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSubmissionRepository(p Params) factoringsubmissionports.SubmissionRepository {
	return &submissionRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-submission"),
	}
}

func (r *submissionRepository) Create(
	ctx context.Context,
	entity *factoringsubmission.Submission,
) (*factoringsubmission.Submission, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringsubmission.Submission, error) {
			db := r.db.Keel(ctx)
			if err := factoringsubmissiondb.Submissions.Create(ctx, db, entity); err != nil {
				return nil, translateSubmissionError(err, entity.ID)
			}
			if len(entity.Items) == 0 {
				return entity, nil
			}
			for idx, item := range entity.Items {
				item.SubmissionID = entity.ID
				item.Position = idx + 1
			}
			if err := factoringsubmissiondb.SubmissionItems.CreateMany(ctx, db, entity.Items); err != nil {
				return nil, translateSubmissionError(err, entity.ID)
			}

			return entity, nil
		},
	)
}

func (r *submissionRepository) Update(
	ctx context.Context,
	entity *factoringsubmission.Submission,
) (*factoringsubmission.Submission, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*factoringsubmission.Submission, error) {
			s := factoringsubmissiondb.Submissions
			err := s.SaveExcept(ctx, r.db.Keel(ctx), entity,
				s.Number,
				s.FactoringCompanyID,
				s.AgreementID,
				s.IdempotencyKey,
				s.InvoiceCount,
				s.FaceMinor,
				s.ExpectedAdvanceMinor,
				s.ExpectedFeeMinor,
				s.ExpectedReserveMinor,
				s.SubmittedByID,
				s.AgentRunID,
			)
			if err != nil {
				return nil, translateSubmissionError(err, entity.ID)
			}
			if err = r.updateItemStatuses(ctx, entity); err != nil {
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *submissionRepository) updateItemStatuses(
	ctx context.Context,
	entity *factoringsubmission.Submission,
) error {
	db := r.db.Keel(ctx)
	item := factoringsubmissiondb.SubmissionItems
	for _, row := range entity.Items {
		if row == nil || row.ID.IsNil() {
			continue
		}
		err := item.Save(ctx, db, row,
			item.Status,
			item.ReleasedAt,
			item.FundedOn,
			item.AdvanceMinor,
			item.FeeMinor,
			item.ReserveMinor,
			item.RejectionReason,
			item.SettledOn,
			item.ChargedBackOn,
			item.ChargebackReason,
		)
		if err != nil {
			return translateSubmissionError(err, row.ID)
		}
	}

	return nil
}

func (r *submissionRepository) SetDocuments(
	ctx context.Context,
	req *factoringsubmissionports.SetSubmissionDocumentsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		s := factoringsubmissiondb.Submissions
		n, err := s.Where(s.ID.Eq(req.SubmissionID)).Update(ctx, db,
			s.SchedulePDFDocumentID.ToExpr(keel.Val(req.SchedulePDFDocumentID)),
			s.ScheduleCSVDocumentID.ToExpr(keel.Val(req.ScheduleCSVDocumentID)),
			s.Version.ToExpr(s.Version),
			s.UpdatedAt.ToExpr(s.UpdatedAt),
		)
		if err != nil {
			return submissionErrors.Map(err, req.SubmissionID.String())
		}
		if err = dberror.CheckAffected(n, submissionEntity, req.SubmissionID.String()); err != nil {
			return err
		}
		if len(req.Packets) == 0 {
			return nil
		}

		itemIDs := make([]pulid.ID, 0, len(req.Packets))
		documentIDs := make([]pulid.ID, 0, len(req.Packets))
		for itemID, documentID := range req.Packets {
			itemIDs = append(itemIDs, itemID)
			documentIDs = append(documentIDs, documentID)
		}

		item := factoringsubmissiondb.SubmissionItems
		packets := keel.Unnest("packets", []string{"id", "document_id"}, itemIDs, documentIDs)
		_, err = item.With(packets).
			Join(packets, packets.Col("id").EqCol(item.ID)).
			Where(item.SubmissionID.Eq(req.SubmissionID)).
			Update(ctx, db,
				item.PacketDocumentID.ToExpr(keel.StringOf(packets.Col("document_id")).NullIf("")),
				item.UpdatedAt.ToExpr(item.UpdatedAt),
			)

		return submissionErrors.Map(err, req.SubmissionID.String())
	})
}

func submissionWithItems() *keel.Query[*factoringsubmission.Submission] {
	s := factoringsubmissiondb.Submissions
	items := s.Items()
	return s.Load(items.Order(items.Position.Asc()))
}

func (r *submissionRepository) GetByID(
	ctx context.Context,
	req factoringsubmissionports.GetSubmissionRequest,
) (*factoringsubmission.Submission, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringsubmission.Submission, error) {
		s := factoringsubmissiondb.Submissions
		q := s.Query()
		if req.IncludeItems {
			q = submissionWithItems()
		}

		entity, err := q.Where(s.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, submissionErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *submissionRepository) GetByIdempotencyKey(
	ctx context.Context,
	req factoringsubmissionports.GetSubmissionByKeyRequest,
) (*factoringsubmission.Submission, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoringsubmission.Submission, error) {
		entity, err := submissionWithItems().
			Where(factoringsubmissiondb.Submissions.IdempotencyKey.Eq(req.IdempotencyKey)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, submissionErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *submissionRepository) ListConnection(
	ctx context.Context,
	req *factoringsubmissionports.ListSubmissionConnectionRequest,
) (*pagination.CursorListResult[*factoringsubmission.Submission], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoringsubmission.Submission], error) {
			filter, term := withoutQuery(req.Filter)
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				submissionScope(req, term),
				&listquery.Request[factoringsubmission.Submission]{
					Filter: filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list factoring submissions", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func submissionScope(
	req *factoringsubmissionports.ListSubmissionConnectionRequest,
	term string,
) *keel.Query[*factoringsubmission.Submission] {
	s := factoringsubmissiondb.Submissions
	items := s.Items()
	return s.Where(
		keel.When(term != "", keel.Or(
			s.Number.ContainsFold(term),
			s.FactoringCompany().Name.ContainsFold(term),
			items.Any(items.InvoiceNumber.ContainsFold(term)),
		)),
		keel.When(
			req.FactoringCompanyID.IsNotNil(),
			s.FactoringCompanyID.Eq(req.FactoringCompanyID),
		),
		keel.When(len(req.Statuses) > 0, s.Status.In(req.Statuses...)),
	)
}

func (r *submissionRepository) ListLiveItems(
	ctx context.Context,
	req *factoringsubmissionports.ListLiveItemsRequest,
) (map[pulid.ID]*factoringsubmissionports.LiveItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoringsubmissionports.LiveItem, error) {
			result := make(map[pulid.ID]*factoringsubmissionports.LiveItem, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}

			rows, err := keel.Into[factoringsubmissionports.LiveItem](
				ctx,
				r.db.Keel(ctx),
				liveItems(req),
			)
			if err != nil {
				return nil, err
			}
			for i := range rows {
				row := &rows[i]
				if existing, ok := result[row.InvoiceID]; ok && existing.IsLive() {
					continue
				}
				result[row.InvoiceID] = row
			}

			return result, nil
		},
	)
}

func liveItems(
	req *factoringsubmissionports.ListLiveItemsRequest,
) *keel.Query[*factoringsubmission.SubmissionItem] {
	item := factoringsubmissiondb.SubmissionItems
	sub := factoringsubmissiondb.Submissions
	return item.Pick(item.InvoiceID, item.SubmissionID, sub.Number, sub.Status, item.ID, item.Status).
		Join(sub, sub.ID.EqCol(item.SubmissionID)).
		Where(
			item.InvoiceID.In(req.InvoiceIDs...),
			item.Status.In(liveItemStatuses(req.IncludeChargedBack)...),
		)
}

func (r *submissionRepository) ListItemsBySubmissionIDs(
	ctx context.Context,
	req *factoringsubmissionports.ListItemsBySubmissionIDsRequest,
) (map[pulid.ID][]*factoringsubmission.SubmissionItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*factoringsubmission.SubmissionItem, error) {
			result := make(
				map[pulid.ID][]*factoringsubmission.SubmissionItem,
				len(req.SubmissionIDs),
			)
			if len(req.SubmissionIDs) == 0 {
				return result, nil
			}
			if len(req.SubmissionIDs) > maxSubmissionIDsPerLookup {
				return nil, errortypes.NewValidationError(
					"submissionIds",
					errortypes.ErrInvalid,
					"Too many schedules requested at once",
				)
			}

			item := factoringsubmissiondb.SubmissionItems
			rows, err := item.Where(item.SubmissionID.In(req.SubmissionIDs...)).
				Order(item.SubmissionID.Asc(), item.Position.Asc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}
			for _, row := range rows {
				result[row.SubmissionID] = append(result[row.SubmissionID], row)
			}

			return result, nil
		},
	)
}

func translateSubmissionError(err error, id pulid.ID) error {
	if dberror.IsUniqueConstraintViolation(err) {
		switch dberror.ExtractConstraintName(err) {
		case liveInvoiceIndex:
			return errortypes.NewConflictError(
				"One of these invoices is already on a schedule with the factor; reload the queue and try again",
			)
		case submissionIdempotencyIdx:
			return errortypes.NewConflictError(
				"These invoices were just submitted; reload to see the schedule",
			)
		case submissionNumberIndex:
			return errortypes.NewConflictError(
				"That schedule number is already in use; check the factoring schedule sequence",
			)
		}
	}

	return submissionErrors.Map(err, id.String())
}

func liveItemStatuses(includeChargedBack bool) []factoringsubmission.SubmissionItemStatus {
	statuses := factoringsubmission.LiveSubmissionItemStatuses()
	if includeChargedBack {
		statuses = append(statuses, factoringsubmission.SubmissionItemStatusChargedBack)
	}
	return statuses
}

func (r *submissionRepository) ClaimChargebackWarning(
	ctx context.Context,
	req factoringsubmissionports.ClaimChargebackWarningRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		item := factoringsubmissiondb.SubmissionItems
		n, err := item.Where(
			item.ID.Eq(req.ItemID),
			item.Status.Eq(factoringsubmission.SubmissionItemStatusFunded),
			item.ChargebackWarnedAt.IsNull(),
		).Update(ctx, r.db.Keel(ctx),
			item.ChargebackWarnedAt.To(req.At),
			item.UpdatedAt.ToExpr(item.UpdatedAt),
		)
		if err != nil {
			return false, submissionErrors.Map(err, req.ItemID.String())
		}

		return n > 0, nil
	})
}

func (r *submissionRepository) GetByIDs(
	ctx context.Context,
	req factoringsubmissionports.GetSubmissionsByIDsRequest,
) ([]*factoringsubmission.Submission, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringsubmission.Submission, error) {
			if len(req.IDs) == 0 {
				return []*factoringsubmission.Submission{}, nil
			}
			if len(req.IDs) > maxSubmissionIDsPerLookup {
				return nil, errortypes.NewValidationError(
					"ids",
					errortypes.ErrInvalid,
					"Too many schedules were asked for at once",
				)
			}

			s := factoringsubmissiondb.Submissions
			rows, err := s.Where(s.ID.In(req.IDs...)).Limit(len(req.IDs)).List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			return rows, nil
		},
	)
}
