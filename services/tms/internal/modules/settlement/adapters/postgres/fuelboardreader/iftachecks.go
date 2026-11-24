package fuelboardreader

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelboard"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta/iftadb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	maxCheckRefs = 8
	hoursInDay   = 24
)

type IFTAChecks struct{ db *postgres.Connection }

func NewIFTAChecks(p Params) *IFTAChecks { return &IFTAChecks{db: p.DB} }

func (*IFTAChecks) Metric() string { return fuelboard.MetricIFTAReturnChecks }

type namedLine struct {
	ID             pulid.ID
	Jurisdiction   string
	LineTotalMinor int64
}

type returnFacts struct {
	ret     *ifta.Return
	missing []namedLine
	nMissed int64
	credit  *namedLine
}

func (r *IFTAChecks) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	if req.For.IsNil() {
		return nil, nil //nolint:nilnil // a metric with no return reads as nothing
	}
	facts, err := r.readFacts(ctx, req.For)
	if err != nil || facts == nil {
		return nil, err
	}

	return checksRead(ctx, req, facts), nil
}

func (r *IFTAChecks) readFacts(ctx context.Context, id pulid.ID) (*returnFacts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*returnFacts, error) {
		db := r.db.Keel(ctx)
		ifr := iftadb.Returns
		rl := iftadb.ReturnLines
		ifj := iftadb.Jurisdictions

		var rets []*ifta.Return
		var missing, credits []namedLine
		var missed int64
		err := keel.Batch(ctx, db,
			keel.ListInto(ifr.Where(ifr.ID.Eq(id)).Limit(1), &rets),
			keel.IntoSlice(rl.Join(ifj, ifj.ID.EqCol(rl.JurisdictionID)).
				Pick(rl.ID, text(ifj.Name), rl.LineTotalMinor).
				Where(rl.ReturnID.Eq(id), rl.RateMissing.IsTrue()).
				Order(rl.SortOrder.Asc()).
				Limit(maxCheckRefs), &missing),
			keel.CountInto(rl.Where(rl.ReturnID.Eq(id), rl.RateMissing.IsTrue()), &missed),
			keel.IntoSlice(rl.Join(ifj, ifj.ID.EqCol(rl.JurisdictionID)).
				Pick(rl.ID, text(ifj.Name), rl.LineTotalMinor).
				Where(rl.ReturnID.Eq(id), rl.LineTotalMinor.Lt(0)).
				Order(rl.LineTotalMinor.Asc(), rl.SortOrder.Asc()).
				Limit(1), &credits),
		)
		if err != nil {
			return nil, err
		}
		if len(rets) == 0 {
			return nil, nil
		}
		facts := &returnFacts{ret: rets[0], missing: missing, nMissed: missed}
		if len(credits) > 0 {
			facts.credit = &credits[0]
		}

		return facts, nil
	})
}

func returnZone(ret *ifta.Return, fallback *time.Location) *time.Location {
	if ret.Timezone != "" {
		if loc, err := time.LoadLocation(ret.Timezone); err == nil {
			return loc
		}
	}
	if fallback != nil {
		return fallback
	}

	return time.UTC
}

func checksRead(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
	facts *returnFacts,
) *boardcatalog.MetricRead {
	ret := facts.ret
	loc := returnZone(ret, req.Location)
	locale := i18n.FromContext(ctx)
	period := ret.Period()
	due := time.Unix(period.DueDate(loc), 0).In(loc)
	dueDay := i18n.Day(locale, due, time.Unix(req.Now, 0).In(loc))

	blocked, warned, cleared := problemChecks(ctx, facts)
	warned = append(warned, timingChecks(ctx, req.Now, ret, loc)...)
	if check, ok := statusCheck(ctx, req.Now, ret, loc); ok {
		if check.Tone == boardcatalog.ToneOK {
			cleared = append([]boardcatalog.Check{check}, cleared...)
		} else {
			warned = append(warned, check)
		}
	}

	checks := slices.Concat(blocked, warned, cleared)
	blocking := boardcatalog.Count(int64(len(blocked)))
	net := boardcatalog.Money(ret.NetDueMinor, ret.CurrencyCode)
	missing := boardcatalog.Count(facts.nMissed)
	credit := boardcatalog.Money(0, ret.CurrencyCode)
	dueText := boardcatalog.Value{Kind: boardcatalog.ValueText, Text: dueDay}
	figures := map[string]*boardcatalog.Value{
		fuelboard.FigureFilingBlockers:   &blocking,
		fuelboard.FigureNetTaxDue:        &net,
		fuelboard.FigureRatesMissing:     &missing,
		fuelboard.FigureLargestCredit:    &credit,
		fuelboard.FigureLargestCreditFor: nil,
		fuelboard.FigureFilingDue:        &dueText,
	}
	if facts.credit != nil {
		credit = boardcatalog.Money(-facts.credit.LineTotalMinor, ret.CurrencyCode)
		where := boardcatalog.Value{Kind: boardcatalog.ValueText, Text: facts.credit.Jurisdiction}
		figures[fuelboard.FigureLargestCreditFor] = &where
	}

	return &boardcatalog.MetricRead{
		Title:   i18n.T(ctx, "{0} IFTA return", period.Label()),
		Source:  i18n.T(ctx, "The {0} return and its checks", period.Label()),
		Figures: figures,
		Checks:  checks,
	}
}

type codeGroup struct {
	code     ifta.ProblemCode
	problems []ifta.Problem
}

func groupProblems(problems []ifta.Problem) []codeGroup {
	groups := make([]codeGroup, 0, len(problems))
	for _, problem := range problems {
		at := slices.IndexFunc(groups, func(g codeGroup) bool { return g.code == problem.Code })
		if at < 0 {
			groups = append(groups, codeGroup{code: problem.Code})
			at = len(groups) - 1
		}
		groups[at].problems = append(groups[at].problems, problem)
	}

	return groups
}

func allProblemCodes() []ifta.ProblemCode {
	return []ifta.ProblemCode{
		ifta.ProblemMissingRate, ifta.ProblemNoFuelForType, ifta.ProblemUnattributedMiles,
		ifta.ProblemNoTractorMiles, ifta.ProblemMileageMismatch, ifta.ProblemNonMemberActivity,
		ifta.ProblemNonQualifiedActivity,
	}
}

func problemChecks(
	ctx context.Context,
	facts *returnFacts,
) (blocked, warned, cleared []boardcatalog.Check) {
	groups := groupProblems(facts.ret.Problems)
	for _, group := range groups {
		check := boardcatalog.Check{
			Title:  i18n.T(ctx, group.code.Label()),
			Tone:   boardcatalog.ToneWarn,
			Detail: problemDetail(ctx, group.problems),
			Count:  problemCount(ctx, facts.ret, group),
		}
		if group.code.Blocks() {
			check.Tone = boardcatalog.ToneBad
		}
		if group.code == ifta.ProblemMissingRate {
			check.Refs, check.Hidden = missingRefs(facts)
		}
		if check.Tone == boardcatalog.ToneBad {
			blocked = append(blocked, check)
		} else {
			warned = append(warned, check)
		}
	}
	for _, code := range allProblemCodes() {
		if slices.ContainsFunc(groups, func(g codeGroup) bool { return g.code == code }) {
			continue
		}
		if title := clearedTitle(ctx, code); title != "" {
			cleared = append(cleared, boardcatalog.Check{Title: title, Tone: boardcatalog.ToneOK})
		}
	}

	return blocked, warned, cleared
}

func missingRefs(facts *returnFacts) (refs []boardcatalog.Ref, hidden int) {
	refs = make([]boardcatalog.Ref, 0, len(facts.missing))
	for _, line := range facts.missing {
		refs = append(refs, boardcatalog.Ref{Kind: recordkind.IFTAReturnLine, ID: line.ID})
	}

	return refs, max(int(facts.nMissed)-len(refs), 0)
}

func problemDetail(ctx context.Context, problems []ifta.Problem) string {
	if len(problems) == 0 {
		return ""
	}
	if len(problems) == 1 {
		return problems[0].Message
	}

	return i18n.T(ctx, "{0}, and {1} more like it", problems[0].Message,
		strconv.Itoa(len(problems)-1))
}

func milesText(ctx context.Context, miles decimal.Decimal) *boardcatalog.Value {
	return &boardcatalog.Value{
		Kind: boardcatalog.ValueText,
		Text: i18n.T(ctx, "{0} mi",
			i18n.FormatNumber(i18n.FromContext(ctx), miles.Round(0).InexactFloat64())),
	}
}

func problemCount(ctx context.Context, ret *ifta.Return, group codeGroup) *boardcatalog.Value {
	switch group.code {
	case ifta.ProblemUnattributedMiles:
		return milesText(ctx, ret.UnattributedMiles)
	case ifta.ProblemNoTractorMiles:
		return milesText(ctx, ret.NoTractorMiles)
	case ifta.ProblemMileageMismatch:
		if moves, err := strconv.ParseInt(group.problems[0].Amount, 10, 64); err == nil {
			count := boardcatalog.Count(moves)
			return &count
		}
		return nil
	case ifta.ProblemMissingRate, ifta.ProblemNoFuelForType, ifta.ProblemNonMemberActivity,
		ifta.ProblemNonQualifiedActivity:
	}
	count := boardcatalog.Count(int64(len(group.problems)))

	return &count
}

func clearedTitle(ctx context.Context, code ifta.ProblemCode) string {
	switch code {
	case ifta.ProblemMissingRate:
		return i18n.T(ctx, "Every line has a tax rate")
	case ifta.ProblemNoFuelForType:
		return i18n.T(ctx, "Fuel bought for every fuel type run")
	case ifta.ProblemUnattributedMiles:
		return i18n.T(ctx, "Every move's miles are placed in a jurisdiction")
	case ifta.ProblemNoTractorMiles:
		return i18n.T(ctx, "Every move has a tractor")
	case ifta.ProblemMileageMismatch:
		return i18n.T(ctx, "Jurisdiction miles match the moves")
	case ifta.ProblemNonMemberActivity:
		return i18n.T(ctx, "No miles outside IFTA jurisdictions")
	case ifta.ProblemNonQualifiedActivity:
		return i18n.T(ctx, "No miles on tractors that are not IFTA qualified")
	default:
		return ""
	}
}

func lastDay(end int64, loc *time.Location) time.Time {
	return time.Unix(end-1, 0).In(loc)
}

func dayOf(at time.Time) time.Time {
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
}

func daysBetween(from, to time.Time) int64 {
	return int64(dayOf(to).Sub(dayOf(from)).Hours() / hoursInDay)
}

func timingChecks(
	ctx context.Context,
	now int64,
	ret *ifta.Return,
	loc *time.Location,
) []boardcatalog.Check {
	locale := i18n.FromContext(ctx)
	today := time.Unix(now, 0).In(loc)
	checks := make([]boardcatalog.Check, 0, 2)
	if now < ret.PeriodEnd {
		ends := lastDay(ret.PeriodEnd, loc)
		left := boardcatalog.Days(daysBetween(today, ends))
		checks = append(checks, boardcatalog.Check{
			Title: i18n.T(ctx, "The quarter is still running"),
			Tone:  boardcatalog.ToneWarn,
			Detail: i18n.T(ctx,
				"It ends {0}; miles driven and fuel bought until then still go on the return.",
				i18n.Day(locale, ends, today)),
			Count: &left,
		})
	}
	if ret.Status == ifta.ReturnStatusFiled {
		return checks
	}
	due := time.Unix(ret.Period().DueDate(loc), 0).In(loc)
	if late := daysBetween(due, today); late > 0 {
		days := boardcatalog.Days(late)
		checks = append(checks, boardcatalog.Check{
			Title: i18n.T(ctx, "Past the filing date"),
			Tone:  boardcatalog.ToneWarn,
			Detail: i18n.T(ctx, "The return is not filed; the quarter's filing date was {0}.",
				i18n.Day(locale, due, today)),
			Count: &days,
		})
	}

	return checks
}

func statusCheck(
	ctx context.Context,
	now int64,
	ret *ifta.Return,
	loc *time.Location,
) (boardcatalog.Check, bool) {
	locale := i18n.FromContext(ctx)
	today := time.Unix(now, 0).In(loc)
	due := i18n.Day(locale, time.Unix(ret.Period().DueDate(loc), 0).In(loc), today)
	switch ret.Status {
	case ifta.ReturnStatusDraft:
		return boardcatalog.Check{
			Title: i18n.T(ctx, "Not finalized yet"),
			Tone:  boardcatalog.ToneWarn,
			Detail: i18n.T(
				ctx,
				"A draft is finalized once nothing blocks it, and only then marked filed. It is due {0}.",
				due,
			),
		}, true
	case ifta.ReturnStatusFinalized:
		return boardcatalog.Check{
			Title:  i18n.T(ctx, "Finalized"),
			Tone:   boardcatalog.ToneOK,
			Detail: i18n.T(ctx, "Ready to mark filed once it is sent. It is due {0}.", due),
		}, true
	case ifta.ReturnStatusFiled:
		check := boardcatalog.Check{Title: i18n.T(ctx, "Filed"), Tone: boardcatalog.ToneOK}
		if ret.FiledAt != nil {
			check.Detail = i18n.T(ctx, "Filed {0}.",
				i18n.Day(locale, time.Unix(*ret.FiledAt, 0).In(loc), today))
		}
		return check, true
	default:
		return boardcatalog.Check{}, false
	}
}
