package factoringrepository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcol"
	"github.com/emoss08/trenova/pkg/buncolgen/factoringcompanycol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

const (
	remittanceEntity        = "FactoringRemittance"
	maxRemittanceCandidates = 5000
	maxOpenRemittances      = 500
	minCandidateTolerance   = 100
)

type remittanceRepository struct {
	db kernelports.DBConnection
	l  *zap.Logger
}

func NewRemittanceRepository(p Params) factoringports.RemittanceRepository {
	return &remittanceRepository{
		db: p.DB,
		l:  p.Logger.Named("repository.factoring-remittance"),
	}
}

func (r *remittanceRepository) Create(
	ctx context.Context,
	entity *factoring.Remittance,
) (*factoring.Remittance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("create factoring remittance: %w", err)
		}
		return entity, nil
	})
}

func (r *remittanceRepository) Update(
	ctx context.Context,
	entity *factoring.Remittance,
) (*factoring.Remittance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
		cols := factoringcol.RemittanceColumns
		previousVersion := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			ExcludeColumn(
				cols.CreatedAt.Bare(),
				cols.FactoringCompanyID.Bare(),
				cols.Source.Bare(),
				cols.InboundMessageID.Bare(),
				cols.CreatedByID.Bare(),
			).
			WherePK().
			Where(cols.Version.Eq(), previousVersion).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = previousVersion
			return nil, fmt.Errorf("update factoring remittance: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, remittanceEntity, entity.ID.String()); err != nil {
			entity.Version = previousVersion
			return nil, err
		}
		return entity, nil
	})
}

func (r *remittanceRepository) Get(
	ctx context.Context,
	req factoringports.GetRemittanceRequest,
) (*factoring.Remittance, error) {
	read := dbtx.Read[*factoring.Remittance]
	if req.ForUpdate {
		read = dbtx.Write[*factoring.Remittance]
	}
	return read(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
		entity := new(factoring.Remittance)
		q := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			Relation(factoringcol.RemittanceRelations.FactoringCompany).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.RemittanceScopeTenant(sq, req.TenantInfo).
					Where(factoringcol.RemittanceColumns.ID.Eq(), req.ID)
			})
		if req.ForUpdate {
			q = q.For("UPDATE OF " + factoringcol.RemittanceTable.Alias)
		}
		if err := q.Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, remittanceEntity)
		}
		if !req.IncludeLines {
			return entity, nil
		}
		lines, err := r.lines(ctx, req.TenantInfo, entity.ID, req.ForUpdate)
		if err != nil {
			return nil, err
		}
		entity.Lines = lines
		return entity, nil
	})
}

func (r *remittanceRepository) lines(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	remittanceID pulid.ID,
	forUpdate bool,
) ([]*factoring.RemittanceLine, error) {
	cols := factoringcol.RemittanceLineColumns
	lines := make([]*factoring.RemittanceLine, 0, 32)
	q := r.db.DBForContext(ctx).NewSelect().
		Model(&lines).
		ColumnExpr(factoringcol.RemittanceLineTable.All()).
		Apply(withMatchedItem).
		Apply(factoringcol.RemittanceLineApplyTenant(tenantInfo)).
		Where(cols.RemittanceID.Eq(), remittanceID).
		Order(cols.RowNumber.OrderAsc())
	if forUpdate {
		q = q.For("UPDATE OF " + factoringcol.RemittanceLineTable.Alias)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("load factoring remittance lines: %w", err)
	}
	return lines, nil
}

func (r *remittanceRepository) FindByInboundDocument(
	ctx context.Context,
	req *factoringports.FindRemittanceByDocumentRequest,
) (*factoring.Remittance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
		cols := factoringcol.RemittanceColumns
		entity := new(factoring.Remittance)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.RemittanceScopeTenant(sq, req.TenantInfo).
					Where(cols.InboundMessageID.Eq(), req.InboundMessageID).
					Where(cols.DocumentID.Eq(), req.DocumentID)
			}).
			Limit(1).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, remittanceEntity)
		}
		return entity, nil
	})
}

func (r *remittanceRepository) ListConnection(
	ctx context.Context,
	req *factoringports.ListRemittanceConnectionRequest,
) (*pagination.CursorListResult[*factoring.Remittance], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.Remittance], error) {
			dba := r.db.DBForContext(ctx)
			alias := factoringcol.RemittanceTable.Alias
			filter, term := withoutQuery(req.Filter)

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*factoring.Remittance)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.scope(
							querybuilder.ApplyFiltersWithoutSort(
								sq,
								alias,
								filter,
								(*factoring.Remittance)(nil),
							),
							req,
							term,
						)
					}).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count factoring remittances: %w", err)
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*factoring.Remittance]{
					Filter:     filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*factoring.Remittance) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(factoringcol.RemittanceTable.All())
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						q, applyErr := querybuilder.ApplyCursorFilters(
							sq,
							alias,
							filter,
							req.Cursor,
							(*factoring.Remittance)(nil),
						)
						if applyErr != nil {
							return nil, applyErr
						}
						return r.scope(q, req, term), nil
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list factoring remittances: %w", err)
			}
			return result, nil
		},
	)
}

func (r *remittanceRepository) scope(
	q *bun.SelectQuery,
	req *factoringports.ListRemittanceConnectionRequest,
	term string,
) *bun.SelectQuery {
	cols := factoringcol.RemittanceColumns
	if term != "" {
		q = q.Where(
			cols.FileName.Qualified()+" ILIKE ?",
			"%"+stringutils.EscapeLikePattern(term)+"%",
		)
	}
	if !req.FactoringCompanyID.IsNil() {
		q = q.Where(cols.FactoringCompanyID.Eq(), req.FactoringCompanyID)
	}
	if len(req.Statuses) > 0 {
		q = q.Where(cols.Status.In(), bun.List(req.Statuses))
	}
	return q
}

func (r *remittanceRepository) ListLineConnection(
	ctx context.Context,
	req *factoringports.ListRemittanceLineConnectionRequest,
) (*pagination.CursorListResult[*factoring.RemittanceLine], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.RemittanceLine], error) {
			dba := r.db.DBForContext(ctx)
			alias := factoringcol.RemittanceLineTable.Alias
			filter := req.Filter
			if filter == nil {
				filter = &pagination.QueryOptions{TenantInfo: req.TenantInfo}
			}

			scope := func(q *bun.SelectQuery) *bun.SelectQuery {
				cols := factoringcol.RemittanceLineColumns
				q = q.Where(cols.RemittanceID.Eq(), req.RemittanceID)
				if len(req.MatchStatuses) > 0 {
					q = q.Where(cols.MatchStatus.In(), bun.List(req.MatchStatuses))
				}
				return q
			}

			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.NewSelect().
					Model((*factoring.RemittanceLine)(nil)).
					Apply(factoringcol.RemittanceLineApplyTenant(req.TenantInfo)).
					Apply(scope).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count factoring remittance lines: %w", err)
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*factoring.RemittanceLine]{
					Filter:     filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*factoring.RemittanceLine) *bun.SelectQuery {
						return dba.NewSelect().
							Model(entities).
							ColumnExpr(factoringcol.RemittanceLineTable.All()).
							Apply(withMatchedItem)
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						q, applyErr := querybuilder.ApplyCursorFilters(
							sq,
							alias,
							filter,
							req.Cursor,
							(*factoring.RemittanceLine)(nil),
						)
						if applyErr != nil {
							return nil, applyErr
						}
						return scope(
							q.Apply(factoringcol.RemittanceLineApplyTenant(req.TenantInfo)),
						), nil
					},
				},
			)
			if err != nil {
				return nil, fmt.Errorf("list factoring remittance lines: %w", err)
			}
			return result, nil
		},
	)
}

func (r *remittanceRepository) ReplaceLines(
	ctx context.Context,
	remittance *factoring.Remittance,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		dba := r.db.DBForContext(ctx)
		cols := factoringcol.RemittanceLineColumns
		if _, err := dba.NewDelete().
			Model((*factoring.RemittanceLine)(nil)).
			Where(cols.OrganizationID.Eq(), remittance.OrganizationID).
			Where(cols.BusinessUnitID.Eq(), remittance.BusinessUnitID).
			Where(cols.RemittanceID.Eq(), remittance.ID).
			Exec(ctx); err != nil {
			return fmt.Errorf("clear factoring remittance lines: %w", err)
		}
		if len(remittance.Lines) == 0 {
			return nil
		}
		for _, line := range remittance.Lines {
			line.OrganizationID = remittance.OrganizationID
			line.BusinessUnitID = remittance.BusinessUnitID
			line.RemittanceID = remittance.ID
		}
		if _, err := dba.NewInsert().
			Model(&remittance.Lines).
			Returning("*").
			Exec(ctx); err != nil {
			return fmt.Errorf("store factoring remittance lines: %w", err)
		}
		return nil
	})
}

func (r *remittanceRepository) UpdateLines(
	ctx context.Context,
	lines []*factoring.RemittanceLine,
) error {
	if len(lines) == 0 {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := factoringcol.RemittanceLineColumns
		for _, line := range lines {
			if line.Suggestions == nil {
				line.Suggestions = []*factoring.RemittanceSuggestion{}
			}
		}
		if _, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(&lines).
			Column(
				cols.Kind.Bare(),
				cols.MatchStatus.Bare(),
				cols.MatchedBy.Bare(),
				cols.SubmissionItemID.Bare(),
				cols.Score.Bare(),
				cols.Reason.Bare(),
				cols.Suggestions.Bare(),
				cols.IgnoredReason.Bare(),
				cols.ReserveEntryID.Bare(),
				cols.UpdatedAt.Bare(),
			).
			Bulk().
			Exec(ctx); err != nil {
			return fmt.Errorf("update factoring remittance lines: %w", err)
		}
		return nil
	})
}

func (r *remittanceRepository) ListCandidates(
	ctx context.Context,
	req *factoringports.ListRemittanceCandidatesRequest,
) ([]*factoring.RemittanceCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoring.RemittanceCandidate, error) {
			item := factoringcol.SubmissionItemColumns
			sub := factoringcol.SubmissionColumns
			subTable := factoringcol.SubmissionTable
			agr := factoringcol.AgreementColumns
			agrTable := factoringcol.AgreementTable

			type row struct {
				factoring.SubmissionItem `bun:",extend"`

				CandidateSubmissionNumber string             `bun:"candidate_submission_number"`
				CandidateSentAt           *int64             `bun:"candidate_sent_at"`
				CandidateRecourse         factoring.Recourse `bun:"candidate_recourse"`
			}

			invoiceMatch, amountMatch := candidateMatchSQL()

			limit := boundedLimit(req.Limit, maxRemittanceCandidates)
			rows := make([]*row, 0, min(limit, 64))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model(&rows).
				ColumnExpr(factoringcol.SubmissionItemTable.All()).
				ColumnExpr(sub.Number.As("candidate_submission_number")).
				ColumnExpr(sub.SentAt.As("candidate_sent_at")).
				ColumnExpr(agr.Recourse.As("candidate_recourse")).
				Join("JOIN "+subTable.As(subTable.Alias)).
				JoinOn(sub.ID.EqColumn(item.SubmissionID)).
				JoinOn(sub.OrganizationID.EqColumn(item.OrganizationID)).
				JoinOn(sub.BusinessUnitID.EqColumn(item.BusinessUnitID)).
				Join("JOIN "+agrTable.As(agrTable.Alias)).
				JoinOn(agr.ID.EqColumn(sub.AgreementID)).
				JoinOn(agr.OrganizationID.EqColumn(sub.OrganizationID)).
				JoinOn(agr.BusinessUnitID.EqColumn(sub.BusinessUnitID)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.SubmissionItemScopeTenant(sq, req.TenantInfo).
						Where(sub.FactoringCompanyID.Eq(), req.FactoringCompanyID).
						Where(sub.Status.In(), bun.List([]factoring.SubmissionStatus{
							factoring.SubmissionStatusSent,
							factoring.SubmissionStatusPartially,
							factoring.SubmissionStatusFunded,
						})).
						Where(item.Status.In(), bun.List([]factoring.SubmissionItemStatus{
							factoring.SubmissionItemStatusPending,
							factoring.SubmissionItemStatusFunded,
						}))
				}).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = sq.Where(invoiceMatch, pgdialect.Array(nonNilTexts(req.Invoices))).
						WhereOr(amountMatch, pgdialect.Array(nonNilAmounts(req.Amounts)))
					if len(req.ItemIDs) > 0 {
						sq = sq.WhereOr(item.ID.In(), bun.List(req.ItemIDs))
					}
					return sq
				}).
				OrderExpr(sub.SentAt.Qualified() + " DESC NULLS LAST").
				OrderExpr(item.ID.Qualified()).
				Limit(int64(limit)).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list factoring remittance candidates: %w", err)
			}

			out := make([]*factoring.RemittanceCandidate, 0, len(rows))
			for _, rw := range rows {
				it := rw.SubmissionItem
				out = append(out, &factoring.RemittanceCandidate{
					Item:             &it,
					SubmissionNumber: rw.CandidateSubmissionNumber,
					SentAt:           rw.CandidateSentAt,
					Recourse:         rw.CandidateRecourse == factoring.RecourseFull,
				})
			}
			return out, nil
		},
	)
}

func (r *remittanceRepository) ListOpen(
	ctx context.Context,
	req *factoringports.ListOpenRemittancesRequest,
) ([]*factoringports.OpenRemittance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.OpenRemittance, error) {
			rem := factoringcol.RemittanceColumns
			fac := factoringcompanycol.FactoringCompanyColumns
			facTable := factoringcompanycol.FactoringCompanyTable
			limit := boundedLimit(req.Limit, maxOpenRemittances)
			rows := make([]*factoringports.OpenRemittance, 0, min(limit, 16))
			if err := r.db.DBForContext(ctx).NewSelect().
				Model((*factoring.Remittance)(nil)).
				ColumnExpr(rem.ID.As("remittance_id")).
				ColumnExpr(rem.FactoringCompanyID.As("factoring_company_id")).
				ColumnExpr(fac.Name.Expr(coalesceBlank)+" AS factor_name").
				ColumnExpr(rem.FileName.Expr(coalesceBlank)+" AS file_name").
				ColumnExpr(rem.Status.As("status")).
				ColumnExpr(rem.ExceptionCount.As("exception_count")).
				ColumnExpr(rem.MatchedCount.As("matched_count")).
				ColumnExpr(rem.CreatedAt.As("created_at")).
				Join("LEFT JOIN "+facTable.As(facTable.Alias)).
				JoinOn(fac.ID.EqColumn(rem.FactoringCompanyID)).
				JoinOn(fac.OrganizationID.EqColumn(rem.OrganizationID)).
				JoinOn(fac.BusinessUnitID.EqColumn(rem.BusinessUnitID)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return factoringcol.RemittanceScopeTenant(sq, req.TenantInfo).
						Where(rem.Status.In(), bun.List([]factoring.RemittanceStatus{
							factoring.RemittanceStatusNeedsMapping,
							factoring.RemittanceStatusParsed,
							factoring.RemittanceStatusPartiallyApplied,
						})).
						WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
							return inner.Where(rem.ExceptionCount.Qualified()+" > 0").
								WhereOr(rem.Status.Eq(), factoring.RemittanceStatusNeedsMapping)
						})
				}).
				Order(rem.CreatedAt.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx, &rows); err != nil {
				return nil, fmt.Errorf("list open factoring remittances: %w", err)
			}
			return rows, nil
		},
	)
}

func (r *remittanceRepository) GetProfile(
	ctx context.Context,
	req factoringports.GetRemittanceProfileRequest,
) (*factoring.RemittanceProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.RemittanceProfile, error) {
		profile := new(factoring.RemittanceProfile)
		if err := r.db.DBForContext(ctx).NewSelect().
			Model(profile).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return factoringcol.RemittanceProfileScopeTenant(sq, req.TenantInfo).
					Where(factoringcol.RemittanceProfileColumns.FactoringCompanyID.Eq(), req.FactoringCompanyID)
			}).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "FactoringRemittanceProfile")
		}
		return profile, nil
	})
}

func (r *remittanceRepository) SaveProfile(
	ctx context.Context,
	profile *factoring.RemittanceProfile,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := factoringcol.RemittanceProfileColumns
		if _, err := r.db.DBForContext(ctx).NewInsert().
			Model(profile).
			On("CONFLICT (organization_id, business_unit_id, factoring_company_id) DO UPDATE").
			Set(cols.Mapping.Bare() + " = EXCLUDED." + cols.Mapping.Bare()).
			Set(cols.HeaderSignature.Bare() + " = EXCLUDED." + cols.HeaderSignature.Bare()).
			Set(cols.Version.Bare() + " = " + factoringcol.RemittanceProfileTable.Alias + "." + cols.Version.Bare() + " + 1").
			Set(cols.UpdatedAt.Bare() + " = EXCLUDED." + cols.UpdatedAt.Bare()).
			Returning("*").
			Exec(ctx); err != nil {
			return fmt.Errorf("save factoring remittance profile: %w", err)
		}
		return nil
	})
}

func nonNilTexts(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilAmounts(values []int64) []int64 {
	if values == nil {
		return []int64{}
	}
	return values
}

func candidateMatchSQL() (invoiceMatch, amountMatch string) {
	item := factoringcol.SubmissionItemColumns
	normalized := "upper(regexp_replace(" + item.InvoiceNumber.Qualified() +
		", '[^[:alnum:]]', '', 'g'))"
	tolerance := buncolgen.Expr(
		"greatest("+strconv.Itoa(minCandidateTolerance)+", {0})",
		item.ExpectedFeeMinor,
	)
	near := func(target string) string {
		return "abs(abs(amt.v) - " + target + ") <= " + tolerance
	}
	amountMatch = "EXISTS (SELECT 1 FROM unnest(?::bigint[]) AS amt(v) WHERE " +
		near(item.ExpectedAdvanceMinor.Qualified()) + " OR " +
		near(item.FaceMinor.Qualified()) + " OR " +
		near(buncolgen.Expr("coalesce({0}, -1)", item.ReserveMinor)) + " OR " +
		near(
			buncolgen.Expr(
				"coalesce({0}, 0) + coalesce({1}, 0)",
				item.AdvanceMinor,
				item.FeeMinor,
			),
		) +
		")"
	invoiceMatch = "EXISTS (SELECT 1 FROM unnest(?::text[]) AS ref(v) WHERE " +
		normalized + " = ref.v OR (length(ref.v) >= 4 AND length(" + normalized + ") >= 4 AND (" +
		normalized + " LIKE '%' || ref.v || '%' OR ref.v LIKE '%' || " + normalized + " || '%')))"
	return invoiceMatch, amountMatch
}

func withMatchedItem(q *bun.SelectQuery) *bun.SelectQuery {
	line := factoringcol.RemittanceLineColumns
	item := factoringcol.SubmissionItemColumns
	itemTable := factoringcol.SubmissionItemTable
	return q.
		ColumnExpr(item.InvoiceNumber.As("matched_invoice_number")).
		ColumnExpr(item.SubmissionID.As("matched_submission_id")).
		Join("LEFT JOIN " + itemTable.As(itemTable.Alias)).
		JoinOn(item.ID.EqColumn(line.SubmissionItemID)).
		JoinOn(item.OrganizationID.EqColumn(line.OrganizationID)).
		JoinOn(item.BusinessUnitID.EqColumn(line.BusinessUnitID))
}
