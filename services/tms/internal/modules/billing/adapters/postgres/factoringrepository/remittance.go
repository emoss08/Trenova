package factoringrepository

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring/factoringdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	remittanceEntity        = "FactoringRemittance"
	maxRemittanceCandidates = 5000
	maxOpenRemittances      = 500
	minCandidateTolerance   = 100
)

var (
	remittanceErrors     = dberror.Mapping{Entity: remittanceEntity}
	remittanceLineErrors = dberror.Mapping{Entity: "FactoringRemittanceLine"}
	profileErrors        = dberror.Mapping{Entity: "FactoringRemittanceProfile"}
)

type remittanceRepository struct {
	db *postgres.Connection
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
		if err := factoringdb.Remittances.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, remittanceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *remittanceRepository) Update(
	ctx context.Context,
	entity *factoring.Remittance,
) (*factoring.Remittance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
		rem := factoringdb.Remittances
		err := rem.SaveExcept(ctx, r.db.Keel(ctx), entity,
			rem.FactoringCompanyID,
			rem.Source,
			rem.InboundMessageID,
			rem.CreatedByID,
		)
		if err != nil {
			return nil, remittanceErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *remittanceRepository) Get(
	ctx context.Context,
	req factoringports.GetRemittanceRequest,
) (*factoring.Remittance, error) {
	if req.ForUpdate {
		return dbtx.Write(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
			return r.get(ctx, req)
		})
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
		return r.get(ctx, req)
	})
}

func (r *remittanceRepository) get(
	ctx context.Context,
	req factoringports.GetRemittanceRequest,
) (*factoring.Remittance, error) {
	db := r.db.Keel(ctx)
	rem := factoringdb.Remittances
	q := rem.Where(rem.ID.Eq(req.ID)).Load(rem.FactoringCompany())
	if req.ForUpdate {
		q = q.Lock(keel.ForUpdate)
	}
	entity, err := q.One(ctx, db)
	if err != nil {
		return nil, remittanceErrors.Map(err, req.ID.String())
	}
	if !req.IncludeLines {
		return entity, nil
	}

	lines := withMatchedItem(factoringdb.RemittanceLines.Query()).
		Where(factoringdb.RemittanceLines.RemittanceID.Eq(entity.ID)).
		Order(factoringdb.RemittanceLines.RowNumber.Asc())
	if req.ForUpdate {
		lines = lines.Lock(keel.ForUpdate)
	}
	entity.Lines, err = lines.All(ctx, db)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func withMatchedItem(
	q *keel.Query[*factoring.RemittanceLine],
) *keel.Query[*factoring.RemittanceLine] {
	line := factoringdb.RemittanceLines
	item := line.SubmissionItem()
	return q.
		Compute(line.MatchedInvoice, item.InvoiceNumber.Coalesce("")).
		Compute(line.MatchedSubmission, item.SubmissionID.Coalesce(""))
}

func (r *remittanceRepository) FindByInboundDocument(
	ctx context.Context,
	req *factoringports.FindRemittanceByDocumentRequest,
) (*factoring.Remittance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.Remittance, error) {
		rem := factoringdb.Remittances
		entity, err := rem.Where(
			rem.InboundMessageID.Eq(req.InboundMessageID),
			rem.DocumentID.Eq(req.DocumentID),
		).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, remittanceErrors.Map(err, "")
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
			filter, term := withoutQuery(req.Filter)
			rem := factoringdb.Remittances
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				rem.Where(
					keel.When(term != "", rem.FileName.ContainsFold(term)),
					keel.When(
						req.FactoringCompanyID.IsNotNil(),
						rem.FactoringCompanyID.Eq(req.FactoringCompanyID),
					),
					keel.When(
						req.InboundMessageID.IsNotNil(),
						rem.InboundMessageID.Eq(req.InboundMessageID),
					),
					keel.When(len(req.Statuses) > 0, rem.Status.In(req.Statuses...)),
				),
				&listquery.Request[factoring.Remittance]{Filter: filter, Cursor: req.Cursor},
			)
			if err != nil {
				r.l.Error("failed to list factoring remittances", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *remittanceRepository) ListLineConnection(
	ctx context.Context,
	req *factoringports.ListRemittanceLineConnectionRequest,
) (*pagination.CursorListResult[*factoring.RemittanceLine], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*factoring.RemittanceLine], error) {
			line := factoringdb.RemittanceLines
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				line.Where(
					line.RemittanceID.Eq(req.RemittanceID),
					keel.When(
						len(req.MatchStatuses) > 0,
						line.MatchStatus.In(req.MatchStatuses...),
					),
				),
				&listquery.Request[factoring.RemittanceLine]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withMatchedItem,
				},
			)
			if err != nil {
				r.l.Error("failed to list factoring remittance lines", zap.Error(err))
				return nil, err
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
		db := r.db.Keel(ctx)
		line := factoringdb.RemittanceLines
		if _, err := line.Where(line.RemittanceID.Eq(remittance.ID)).Delete(ctx, db); err != nil {
			return remittanceLineErrors.Map(err, remittance.ID.String())
		}
		if len(remittance.Lines) == 0 {
			return nil
		}
		for _, l := range remittance.Lines {
			l.RemittanceID = remittance.ID
		}

		return remittanceLineErrors.Map(
			line.CreateMany(ctx, db, remittance.Lines),
			remittance.ID.String(),
		)
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
		q, sets, err := lineOutcomes(lines)
		if err != nil {
			return err
		}

		_, err = q.Update(ctx, r.db.Keel(ctx), sets...)
		return remittanceLineErrors.Map(err, "")
	})
}

func lineOutcomes(
	lines []*factoring.RemittanceLine,
) (*keel.Query[*factoring.RemittanceLine], []keel.Assignment, error) {
	n := len(lines)
	ids := make([]pulid.ID, n)
	kinds := make([]string, n)
	statuses := make([]string, n)
	matchedBy := make([]string, n)
	items := make([]pulid.ID, n)
	scores := make([]int64, n)
	reasons := make([]string, n)
	suggestions := make([]string, n)
	ignored := make([]string, n)
	reserves := make([]pulid.ID, n)
	for i, l := range lines {
		if l.Suggestions == nil {
			l.Suggestions = []*factoring.RemittanceSuggestion{}
		}
		encoded, err := sonic.MarshalString(l.Suggestions)
		if err != nil {
			return nil, nil, err
		}
		ids[i] = l.ID
		kinds[i] = string(l.Kind)
		statuses[i] = string(l.MatchStatus)
		matchedBy[i] = string(l.MatchedBy)
		items[i] = l.SubmissionItemID
		scores[i] = int64(l.Score)
		reasons[i] = string(l.Reason)
		suggestions[i] = encoded
		ignored[i] = l.IgnoredReason
		reserves[i] = l.ReserveEntryID
	}

	line := factoringdb.RemittanceLines
	outcome := keel.Unnest("outcome", []string{
		"id", "kind", "match_status", "matched_by", "submission_item_id", "score",
		"reason", "suggestions", "ignored_reason", "reserve_entry_id",
	}, ids, kinds, statuses, matchedBy, items, scores, reasons, suggestions, ignored, reserves)
	orNull := func(column string) keel.Expr {
		return keel.StringOf(outcome.Col(column)).NullIf("")
	}

	q := line.With(outcome).
		Join(outcome, outcome.Col("id").EqCol(line.ID)).
		Where(keel.True())

	return q, []keel.Assignment{
		line.Kind.ToExpr(outcome.Col("kind")),
		line.MatchStatus.ToExpr(outcome.Col("match_status")),
		line.MatchedBy.ToExpr(orNull("matched_by")),
		line.SubmissionItemID.ToExpr(orNull("submission_item_id")),
		line.Score.ToExpr(outcome.Col("score")),
		line.Reason.ToExpr(orNull("reason")),
		line.Suggestions.ToExpr(keel.CastJSONB(outcome.Col("suggestions"))),
		line.IgnoredReason.ToExpr(orNull("ignored_reason")),
		line.ReserveEntryID.ToExpr(orNull("reserve_entry_id")),
	}, nil
}

func (r *remittanceRepository) ListCandidates(
	ctx context.Context,
	req *factoringports.ListRemittanceCandidatesRequest,
) ([]*factoring.RemittanceCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoring.RemittanceCandidate, error) {
			items, err := remittanceCandidates(req).List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, err
			}

			out := make([]*factoring.RemittanceCandidate, 0, len(items))
			for _, item := range items {
				candidate := &factoring.RemittanceCandidate{Item: item}
				if sub := item.Submission; sub != nil {
					candidate.SubmissionNumber = sub.Number
					candidate.SentAt = sub.SentAt
					candidate.Recourse = sub.Agreement != nil &&
						sub.Agreement.Recourse == factoring.RecourseFull
				}
				item.Submission = nil
				out = append(out, candidate)
			}

			return out, nil
		},
	)
}

func remittanceCandidates(
	req *factoringports.ListRemittanceCandidatesRequest,
) *keel.Query[*factoring.SubmissionItem] {
	item := factoringdb.SubmissionItems
	sub := item.Submission()
	agr := sub.Agreement()
	invoiceMatch, amountMatch := candidateMatch(
		nonNilTexts(req.Invoices),
		nonNilAmounts(req.Amounts),
	)

	return item.Where(
		sub.FactoringCompanyID.Eq(req.FactoringCompanyID),
		sub.Status.In(
			factoring.SubmissionStatusSent,
			factoring.SubmissionStatusPartially,
			factoring.SubmissionStatusFunded,
		),
		item.Status.In(factoring.SubmissionItemStatusPending, factoring.SubmissionItemStatusFunded),
		keel.Or(
			invoiceMatch,
			amountMatch,
			keel.When(len(req.ItemIDs) > 0, item.ID.In(req.ItemIDs...)),
		),
	).
		Load(sub.Pick(sub.Number, sub.SentAt, sub.AgreementID).Load(agr.Pick(agr.Recourse))).
		Order(sub.SentAt.Desc().NullsLast(), item.ID.Asc()).
		Limit(boundedLimit(req.Limit, maxRemittanceCandidates))
}

func candidateMatch(invoices []string, amounts []int64) (invoiceMatch, amountMatch keel.Pred) {
	item := factoringdb.SubmissionItems
	normalized := item.InvoiceNumber.RegexpReplace("[^[:alnum:]]", "", "g").Upper()
	tolerance := keel.Int64(minCandidateTolerance).Greatest(item.ExpectedFeeMinor)
	amt := keel.Unnest("amt", []string{"v"}, amounts)
	amount := keel.Int64Of(amt.Col("v")).Abs()
	near := func(target keel.Int64Value) keel.Pred {
		return amount.Sub(target).Abs().LteExpr(tolerance)
	}
	amountMatch = keel.Exists(amt.Query().With(amt).Where(keel.Or(
		near(item.ExpectedAdvanceMinor),
		near(item.FaceMinor),
		near(item.ReserveMinor.Coalesce(-1)),
		near(item.AdvanceMinor.Coalesce(0).Add(item.FeeMinor.Coalesce(0))),
	)))

	ref := keel.Unnest("ref", []string{"v"}, invoices)
	text := keel.StringOf(ref.Col("v"))
	contains := func(needle keel.StringValue) keel.StringExpr {
		return keel.ConstText("%").Cat(needle, keel.ConstText("%"))
	}
	invoiceMatch = keel.Exists(ref.Query().With(ref).Where(keel.Or(
		normalized.EqExpr(text),
		keel.And(
			text.Length().Gte(4),
			normalized.Length().Gte(4),
			keel.Or(
				keel.DynOf(normalized).Like(contains(text)),
				keel.DynOf(text).Like(contains(normalized)),
			),
		),
	)))

	return invoiceMatch, amountMatch
}

func (r *remittanceRepository) ListOpen(
	ctx context.Context,
	req *factoringports.ListOpenRemittancesRequest,
) ([]*factoringports.OpenRemittance, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*factoringports.OpenRemittance, error) {
			rows, err := keel.Into[factoringports.OpenRemittance](
				ctx,
				r.db.Keel(ctx),
				openRemittances(req.Limit),
			)
			if err != nil {
				return nil, err
			}

			out := make([]*factoringports.OpenRemittance, len(rows))
			for i := range rows {
				out[i] = &rows[i]
			}

			return out, nil
		},
	)
}

func openRemittances(limit int) *keel.Query[*factoring.Remittance] {
	rem := factoringdb.Remittances
	fac := rem.FactoringCompany()
	return rem.Pick(
		rem.ID,
		rem.FactoringCompanyID,
		fac.Name.Coalesce(""),
		rem.FileName.Coalesce(""),
		rem.Status,
		rem.ExceptionCount,
		rem.MatchedCount,
		rem.CreatedAt,
	).
		Where(
			rem.Status.In(
				factoring.RemittanceStatusNeedsMapping,
				factoring.RemittanceStatusParsed,
				factoring.RemittanceStatusPartiallyApplied,
			),
			keel.Or(
				rem.ExceptionCount.Gt(0),
				rem.Status.Eq(factoring.RemittanceStatusNeedsMapping),
			),
		).
		Order(rem.CreatedAt.Asc()).
		Limit(boundedLimit(limit, maxOpenRemittances))
}

func (r *remittanceRepository) GetProfile(
	ctx context.Context,
	req factoringports.GetRemittanceProfileRequest,
) (*factoring.RemittanceProfile, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*factoring.RemittanceProfile, error) {
		p := factoringdb.RemittanceProfiles
		profile, err := p.Where(p.FactoringCompanyID.Eq(req.FactoringCompanyID)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, profileErrors.Map(err, req.FactoringCompanyID.String())
		}

		return profile, nil
	})
}

func (r *remittanceRepository) SaveProfile(
	ctx context.Context,
	profile *factoring.RemittanceProfile,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		p := factoringdb.RemittanceProfiles
		_, err := p.Upsert(ctx, db, []*factoring.RemittanceProfile{profile},
			keel.OnConflict(p.OrganizationID, p.BusinessUnitID, p.FactoringCompanyID).
				Update(p.Mapping, p.HeaderSignature),
		)
		if err != nil {
			return profileErrors.Map(err, profile.FactoringCompanyID.String())
		}

		stored, err := p.Where(p.FactoringCompanyID.Eq(profile.FactoringCompanyID)).One(ctx, db)
		if err != nil {
			return profileErrors.Map(err, profile.FactoringCompanyID.String())
		}
		*profile = *stored

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
