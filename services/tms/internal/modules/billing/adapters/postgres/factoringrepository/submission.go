package factoringrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcompanycol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	submissionEntity          = "FactoringSubmission"
	liveInvoiceIndex          = "uq_factoring_submission_items_live_invoice"
	submissionIdempotencyIdx  = "uq_factoring_submissions_idempotency"
	submissionNumberIndex     = "uq_factoring_submissions_number"
	maxSubmissionIDsPerLookup = 1000
)

type submissionRepository struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewSubmissionRepository(p Params) factoringports.SubmissionRepository {
	return &submissionRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-submission"),
	}
}

func (r *submissionRepository) Create(
	ctx context.Context,
	entity *factoring.Submission,
) (*factoring.Submission, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Submission, error) {
		dba := r.db.DBForContext(ctx)
		if _, err := dba.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, translateSubmissionError(err, "create factoring submission")
		}
		if len(entity.Items) == 0 {
			return entity, nil
		}
		for idx, item := range entity.Items {
			item.OrganizationID = entity.OrganizationID
			item.BusinessUnitID = entity.BusinessUnitID
			item.SubmissionID = entity.ID
			item.Position = idx + 1
		}
		if _, err := dba.NewInsert().
			Model(&entity.Items).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, translateSubmissionError(err, "create factoring submission items")
		}

		return entity, nil
	})
}

func (r *submissionRepository) Update(
	ctx context.Context,
	entity *factoring.Submission,
) (*factoring.Submission, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Submission, error) {
		cols := factoringcol.SubmissionColumns
		previousVersion := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			ExcludeColumn(
				cols.CreatedAt.Bare(),
				cols.Number.Bare(),
				cols.FactoringCompanyID.Bare(),
				cols.AgreementID.Bare(),
				cols.IdempotencyKey.Bare(),
				cols.InvoiceCount.Bare(),
				cols.FaceMinor.Bare(),
				cols.ExpectedAdvanceMinor.Bare(),
				cols.ExpectedFeeMinor.Bare(),
				cols.ExpectedReserveMinor.Bare(),
				cols.SubmittedByID.Bare(),
				cols.AgentRunID.Bare(),
			).
			WherePK().
			Where(cols.Version.Eq(), previousVersion).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = previousVersion
			return nil, translateSubmissionError(err, "update factoring submission")
		}
		if err = dberror.CheckRowsAffected(res, submissionEntity, entity.ID.String()); err != nil {
			entity.Version = previousVersion
			return nil, err
		}
		if err = r.updateItemStatuses(ctx, entity); err != nil {
			entity.Version = previousVersion
			return nil, err
		}

		return entity, nil
	})
}

func (r *submissionRepository) updateItemStatuses(
	ctx context.Context,
	entity *factoring.Submission,
) error {
	cols := factoringcol.SubmissionItemColumns
	dba := r.db.DBForContext(ctx)
	for _, item := range entity.Items {
		if item == nil || item.ID.IsNil() {
			continue
		}
		if _, err := dba.NewUpdate().
			Model(item).
			Column(
				cols.Status.Bare(),
				cols.ReleasedAt.Bare(),
				cols.FundedOn.Bare(),
				cols.AdvanceMinor.Bare(),
				cols.FeeMinor.Bare(),
				cols.ReserveMinor.Bare(),
				cols.RejectionReason.Bare(),
				cols.SettledOn.Bare(),
				cols.ChargedBackOn.Bare(),
				cols.ChargebackReason.Bare(),
				cols.UpdatedAt.Bare(),
			).
			WherePK().
			Where(cols.SubmissionID.Eq(), entity.ID).
			Exec(ctx); err != nil {
			return translateSubmissionError(err, "update factoring submission item")
		}
	}
	return nil
}

func (r *submissionRepository) SetDocuments(
	ctx context.Context,
	req *factoringports.SetSubmissionDocumentsRequest,
) error {
	_, err := dbtx.Write(ctx, r.db, func(ctx context.Context) (struct{}, error) {
		dba := r.db.DBForContext(ctx)
		cols := factoringcol.SubmissionColumns
		res, err := dba.NewUpdate().
			Model((*factoring.Submission)(nil)).
			Set(cols.SchedulePDFDocumentID.Set(), nullableID(req.SchedulePDFDocumentID)).
			Set(cols.ScheduleCSVDocumentID.Set(), nullableID(req.ScheduleCSVDocumentID)).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return factoringcol.SubmissionScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.SubmissionID)
			}).
			Exec(ctx)
		if err != nil {
			return struct{}{}, fmt.Errorf("store factoring schedule documents: %w", err)
		}
		if err = dberror.CheckRowsAffected(
			res,
			submissionEntity,
			req.SubmissionID.String(),
		); err != nil {
			return struct{}{}, err
		}

		items := factoringcol.SubmissionItemColumns
		for itemID, documentID := range req.Packets {
			if _, err = dba.NewUpdate().
				Model((*factoring.SubmissionItem)(nil)).
				Set(items.PacketDocumentID.Set(), nullableID(documentID)).
				Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return factoringcol.SubmissionItemScopeTenantUpdate(uq, req.TenantInfo).
						Where(items.ID.Eq(), itemID).
						Where(items.SubmissionID.Eq(), req.SubmissionID)
				}).
				Exec(ctx); err != nil {
				return struct{}{}, fmt.Errorf("store factoring packet: %w", err)
			}
		}
		return struct{}{}, nil
	})
	return err
}

func (r *submissionRepository) GetByID(
	ctx context.Context,
	req factoringports.GetSubmissionRequest,
) (*factoring.Submission, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Submission, error) {
		entity := new(factoring.Submission)
		q := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.SubmissionScopeTenant(sq, req.TenantInfo).
					Where(factoringcol.SubmissionColumns.ID.Eq(), req.ID)
			})
		if req.IncludeItems {
			q = q.Relation(
				factoringcol.SubmissionRelations.Items,
				func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(factoringcol.SubmissionItemColumns.Position.OrderAsc())
				},
			)
		}
		if err := q.Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, submissionEntity)
		}

		return entity, nil
	})
}

func (r *submissionRepository) GetByIdempotencyKey(
	ctx context.Context,
	req factoringports.GetSubmissionByKeyRequest,
) (*factoring.Submission, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Submission, error) {
		entity := new(factoring.Submission)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			Relation(
				factoringcol.SubmissionRelations.Items,
				func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Order(factoringcol.SubmissionItemColumns.Position.OrderAsc())
				},
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.SubmissionScopeTenant(sq, req.TenantInfo).
					Where(factoringcol.SubmissionColumns.IdempotencyKey.Eq(), req.IdempotencyKey)
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, submissionEntity)
		}

		return entity, nil
	})
}

func (r *submissionRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListSubmissionConnectionRequest,
) (*pagination.CursorListResult[*factoring.Submission], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.Submission], error) {
			dba := r.db.DBForContext(ctx)
			alias := factoringcol.SubmissionTable.Alias
			filter, term := withoutQuery(req.Filter)

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*factoring.Submission)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.scope(
							dba,
							querybuilder.ApplyFiltersWithoutSort(
								sq,
								alias,
								filter,
								(*factoring.Submission)(nil),
							),
							req,
							term,
						)
					}).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count factoring submissions: %w", err)
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*factoring.Submission]{
					Filter:     filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*factoring.Submission) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(factoringcol.SubmissionTable.All())
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						q, applyErr := querybuilder.ApplyCursorFilters(
							sq,
							alias,
							filter,
							req.Cursor,
							(*factoring.Submission)(nil),
						)
						if applyErr != nil {
							return nil, applyErr
						}
						return r.scope(dba, q, req, term), nil
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list factoring submissions: %w", err)
			}

			return result, nil
		},
	)
}

func (r *submissionRepository) scope(
	dba bun.IDB,
	q *bun.SelectQuery,
	req *factoringports.ListSubmissionConnectionRequest,
	term string,
) *bun.SelectQuery {
	cols := factoringcol.SubmissionColumns
	if term != "" {
		q = searchSubmissions(dba, q, term)
	}
	if !req.FactoringCompanyID.IsNil() {
		q = q.Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID)
	}
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	return q
}

func searchSubmissions(dba bun.IDB, q *bun.SelectQuery, term string) *bun.SelectQuery {
	pattern := "%" + stringutils.EscapeLikePattern(term) + "%"
	sub := factoringcol.SubmissionColumns
	item := factoringcol.SubmissionItemColumns
	fac := factoringcompanycol.FactoringCompanyColumns

	factors := dba.NewSelect().
		Model((*factoringcompany.FactoringCompany)(nil)).
		ColumnExpr("1").
		Where(fac.ID.Qualified()+" = "+sub.FactoringCompanyID.Qualified()).
		Where(fac.OrganizationID.Qualified()+" = "+sub.OrganizationID.Qualified()).
		Where(fac.BusinessUnitID.Qualified()+" = "+sub.BusinessUnitID.Qualified()).
		Where(fac.Name.ILike(), pattern)
	invoices := dba.NewSelect().
		Model((*factoring.SubmissionItem)(nil)).
		ColumnExpr("1").
		Where(item.SubmissionID.Qualified()+" = "+sub.ID.Qualified()).
		Where(item.OrganizationID.Qualified()+" = "+sub.OrganizationID.Qualified()).
		Where(item.BusinessUnitID.Qualified()+" = "+sub.BusinessUnitID.Qualified()).
		Where(item.InvoiceNumber.ILike(), pattern)

	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(sub.Number.ILike(), pattern).
			WhereOr("EXISTS (?)", factors).
			WhereOr("EXISTS (?)", invoices)
	})
}

func (r *submissionRepository) ListLiveItems(
	ctx context.Context,
	req *factoringports.ListLiveItemsRequest,
) (map[pulid.ID]*factoringports.LiveItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoringports.LiveItem, error) {
			result := make(map[pulid.ID]*factoringports.LiveItem, len(req.InvoiceIDs))
			if len(req.InvoiceIDs) == 0 {
				return result, nil
			}
			item := factoringcol.SubmissionItemColumns
			sub := factoringcol.SubmissionColumns
			rows := make([]*factoringports.LiveItem, 0, len(req.InvoiceIDs))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model((*factoring.SubmissionItem)(nil)).
				ColumnExpr(item.InvoiceID.As("invoice_id")).
				ColumnExpr(item.SubmissionID.As("submission_id")).
				ColumnExpr(sub.Number.As("submission_number")).
				ColumnExpr(sub.Status.As("submission_status")).
				ColumnExpr(item.ID.As("item_id")).
				ColumnExpr(item.Status.As("item_status")).
				Join("JOIN "+factoringcol.SubmissionTable.As(factoringcol.SubmissionTable.Alias)).
				JoinOn(sub.ID.EqColumn(item.SubmissionID)).
				JoinOn(sub.OrganizationID.EqColumn(item.OrganizationID)).
				JoinOn(sub.BusinessUnitID.EqColumn(item.BusinessUnitID)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.SubmissionItemScopeTenant(sq, req.TenantInfo).
						Where(item.InvoiceID.In(), bun.List(req.InvoiceIDs)).
						Where(item.Status.In(), bun.List(liveItemStatuses(req.IncludeChargedBack)))
				}).
				Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("list live factoring submission items: %w", err)
			}
			for _, row := range rows {
				if existing, ok := result[row.InvoiceID]; ok && existing.IsLive() {
					continue
				}
				result[row.InvoiceID] = row
			}
			return result, nil
		},
	)
}

func (r *submissionRepository) ListItemsBySubmissionIDs(
	ctx context.Context,
	req *factoringports.ListItemsBySubmissionIDsRequest,
) (map[pulid.ID][]*factoring.SubmissionItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*factoring.SubmissionItem, error) {
			result := make(map[pulid.ID][]*factoring.SubmissionItem, len(req.SubmissionIDs))
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
			cols := factoringcol.SubmissionItemColumns
			rows := make([]*factoring.SubmissionItem, 0, len(req.SubmissionIDs)*4)
			if err := r.db.DBForContext(ctx).NewSelect().
				Model(&rows).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.SubmissionItemScopeTenant(sq, req.TenantInfo).
						Where(cols.SubmissionID.In(), bun.List(req.SubmissionIDs))
				}).
				Order(cols.SubmissionID.OrderAsc(), cols.Position.OrderAsc()).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list factoring submission items: %w", err)
			}
			for _, row := range rows {
				result[row.SubmissionID] = append(result[row.SubmissionID], row)
			}
			return result, nil
		},
	)
}

func nullableID(id pulid.ID) any {
	if id.IsNil() {
		return nil
	}
	return id
}

func translateSubmissionError(err error, action string) error {
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
	return fmt.Errorf("%s: %w", action, err)
}

func liveItemStatuses(includeChargedBack bool) []factoring.SubmissionItemStatus {
	statuses := factoring.LiveSubmissionItemStatuses()
	if includeChargedBack {
		statuses = append(statuses, factoring.SubmissionItemStatusChargedBack)
	}
	return statuses
}

func (r *submissionRepository) ClaimChargebackWarning(
	ctx context.Context,
	req factoringports.ClaimChargebackWarningRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := factoringcol.SubmissionItemColumns
		res, err := r.db.DBForContext(ctx).NewUpdate().
			Model((*factoring.SubmissionItem)(nil)).
			Set(cols.ChargebackWarnedAt.Set(), req.At).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return factoringcol.SubmissionItemScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ItemID).
					Where(cols.Status.Eq(), factoring.SubmissionItemStatusFunded).
					Where(cols.ChargebackWarnedAt.IsNull())
			}).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("claim chargeback warning: %w", err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("claim chargeback warning: %w", err)
		}
		return rows > 0, nil
	})
}

func (r *submissionRepository) GetByIDs(
	ctx context.Context,
	req factoringports.GetSubmissionsByIDsRequest,
) ([]*factoring.Submission, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*factoring.Submission, error) {
		rows := make([]*factoring.Submission, 0, len(req.IDs))
		if len(req.IDs) == 0 {
			return rows, nil
		}
		if len(req.IDs) > maxSubmissionIDsPerLookup {
			return nil, errortypes.NewValidationError(
				"ids",
				errortypes.ErrInvalid,
				"Too many schedules were asked for at once",
			)
		}
		cols := factoringcol.SubmissionColumns
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			Apply(factoringcol.SubmissionApplyTenant(req.TenantInfo)).
			Where(cols.ID.In(), bun.List(req.IDs)).
			Scan(ctx); err != nil {
			return nil, fmt.Errorf("get factoring submissions by ids: %w", err)
		}
		return rows, nil
	})
}
