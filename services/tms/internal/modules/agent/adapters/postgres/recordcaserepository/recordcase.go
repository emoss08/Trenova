package recordcaserepository

import (
	"context"
	"fmt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision/agentdecisiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan/agentplandb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase/recordcasedb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxDecisions       = 20
	decisionsPerRecord = 3
)

var (
	livePlans = []agentplan.PlanStatus{
		agentplan.PlanStatusPending,
		agentplan.PlanStatusApproving,
		agentplan.PlanStatusApproved,
	}
	liveProposals = []agent.ProposalStatus{
		agent.ProposalStatusPending,
		agent.ProposalStatusApproving,
		agent.ProposalStatusAccepted,
		agent.ProposalStatusModified,
	}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) recordcaseports.RecordCaseRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.record-case-repository"),
	}
}

func (r *repository) Options(
	ctx context.Context,
	req recordcaseports.RecordRequest,
) ([]*recordcase.Option, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*recordcase.Option, error) {
		o := recordcasedb.Options
		options, err := o.Where(o.RecordID.Eq(req.RecordID)).
			Order(o.Position.Asc()).
			Limit(recordcase.MaxOptions).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list case options: %w", err)
		}

		return options, nil
	})
}

func (r *repository) GetOption(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*recordcase.Option, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*recordcase.Option, error) {
		o := recordcasedb.Options
		option, err := o.Where(o.ID.Eq(id)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewNotFoundError("That option is no longer offered")
			}

			return nil, fmt.Errorf("get case option: %w", err)
		}

		return option, nil
	})
}

func (r *repository) ReplaceOptions(
	ctx context.Context,
	req *recordcaseports.ReplaceOptionsRequest,
) ([]*recordcase.Option, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*recordcase.Option, error) {
		o := recordcasedb.Options
		if _, err := o.Where(o.RecordID.Eq(req.RecordID)).Delete(ctx, r.db.Keel(ctx)); err != nil {
			return nil, fmt.Errorf("clear case options: %w", err)
		}
		if err := o.CreateMany(ctx, r.db.Keel(ctx), req.Options); err != nil {
			return nil, fmt.Errorf("create case options: %w", err)
		}

		return req.Options, nil
	})
}

func (r *repository) Choose(
	ctx context.Context,
	req recordcaseports.ChooseRequest,
) (*recordcase.Option, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*recordcase.Option, error) {
		o := recordcasedb.Options
		if _, err := o.Where(o.RecordID.Eq(req.RecordID), o.ID.Ne(req.OptionID)).
			Update(ctx, r.db.Keel(ctx), o.Chosen.To(false)); err != nil {
			return nil, fmt.Errorf("clear the chosen case option: %w", err)
		}
		rows, err := o.Where(o.RecordID.Eq(req.RecordID), o.ID.Eq(req.OptionID)).
			UpdateReturning(ctx, r.db.Keel(ctx), o.Chosen.To(true))
		if err != nil {
			return nil, fmt.Errorf("choose case option: %w", err)
		}
		if len(rows) == 0 {
			return nil, errortypes.NewNotFoundError("That option is no longer offered")
		}

		return rows[0], nil
	})
}

func (r *repository) SetDraft(ctx context.Context, req recordcaseports.SetDraftRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		o := recordcasedb.Options
		if _, err := o.Where(o.ID.Eq(req.OptionID)).Update(
			ctx,
			r.db.Keel(ctx),
			o.DraftRunID.To(req.RunID),
			o.DraftRequestedAt.To(req.RequestedAt),
		); err != nil {
			return fmt.Errorf("record the case option's draft run: %w", err)
		}

		return nil
	})
}

func (r *repository) Reading(
	ctx context.Context,
	req recordcaseports.ReadingRequest,
) (*recordcase.Reading, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*recordcase.Reading, error) {
		rd := recordcasedb.Readings
		reading, err := rd.Where(
			rd.RecordID.Eq(req.RecordID),
			rd.Fingerprint.Eq(req.Fingerprint),
		).One(ctx, r.db.Keel(ctx))
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil //nolint:nilnil // a case nobody has read yet has no reading
			}

			return nil, fmt.Errorf("get case reading: %w", err)
		}

		return reading, nil
	})
}

func (r *repository) ClaimReading(
	ctx context.Context,
	req *recordcaseports.ClaimReadingRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		rd := recordcasedb.Readings
		if _, err := rd.Where(
			rd.RecordID.Eq(req.RecordID),
			rd.Fingerprint.Ne(req.Fingerprint),
			rd.RequestedAt.Lt(req.ForgetBefore),
		).Delete(ctx, r.db.Keel(ctx)); err != nil {
			return false, fmt.Errorf("forget old case readings: %w", err)
		}

		current, err := rd.Where(
			rd.RecordID.Eq(req.RecordID),
			rd.Fingerprint.Eq(req.Fingerprint),
		).
			Lock(keel.ForUpdate).
			One(ctx, r.db.Keel(ctx))
		switch {
		case err != nil && !dberror.IsNotFoundError(err):
			return false, fmt.Errorf("lock case reading: %w", err)
		case err != nil:
			created, cErr := rd.CreateOrSkip(ctx, r.db.Keel(ctx), []*recordcase.Reading{{
				RecordKind:  req.RecordKind,
				RecordID:    req.RecordID,
				Fingerprint: req.Fingerprint,
				Status:      recordcase.ReadingStatusWriting,
				Segments:    []recordcase.Segment{},
				RequestedAt: req.Now,
			}}, rd.OrganizationID, rd.BusinessUnitID, rd.RecordID, rd.Fingerprint)
			if cErr != nil {
				return false, fmt.Errorf("claim case reading: %w", cErr)
			}

			return created > 0, nil
		}

		if current.Status != recordcase.ReadingStatusWriting ||
			current.RequestedAt >= req.StaleBefore {
			return false, nil
		}
		if _, err = rd.Where(rd.ID.Eq(current.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			rd.Status.To(recordcase.ReadingStatusWriting),
			rd.RequestedAt.To(req.Now),
		); err != nil {
			return false, fmt.Errorf("reclaim case reading: %w", err)
		}

		return true, nil
	})
}

func (r *repository) SaveReading(ctx context.Context, reading *recordcase.Reading) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		rd := recordcasedb.Readings
		if _, err := rd.Where(
			rd.RecordID.Eq(reading.RecordID),
			rd.Fingerprint.Eq(reading.Fingerprint),
		).Update(
			ctx,
			r.db.Keel(ctx),
			rd.Status.To(reading.Status),
			rd.Segments.To(reading.Segments),
			rd.WhyItMatters.To(reading.WhyItMatters),
			rd.IfNothingChanges.To(reading.IfNothingChanges),
			rd.ModelIdentifier.To(reading.ModelIdentifier),
			rd.ProviderID.To(reading.ProviderID),
			rd.WrittenAt.To(reading.WrittenAt),
		); err != nil {
			return fmt.Errorf("save case reading: %w", err)
		}

		return nil
	})
}

// askedOnTheCase is the runs of the reader's own conversations about the
// record. What an agent proposes there belongs to the record's case whatever
// record the change itself is made on: asked to chase a load's missing
// proof of delivery, the agent drafts a message to the driver, and that
// message is this load's plan.
func askedOnTheCase(req *recordcaseports.DecisionsRequest) keel.Query[*agent.AgentRun] {
	run, th := agentdb.AgentRuns, conversationdb.Threads

	return run.Pick(run.ID).Where(
		run.SubjectType.Eq(agent.SubjectAssistantThread),
		run.SubjectID.InQuery(th.Pick(th.ID).Where(
			th.UserID.Eq(req.TenantInfo.UserID),
			th.SubjectID.Eq(req.RecordID),
		)),
	)
}

func (r *repository) Decisions(
	ctx context.Context,
	req *recordcaseports.DecisionsRequest,
) (*recordcaseports.Decisions, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*recordcaseports.Decisions, error) {
		out := &recordcaseports.Decisions{Decided: map[pulid.ID]*agentdecision.AgentDecision{}}

		pl, pr := agentplandb.AgentPlans, agentdb.AgentProposals
		asked := askedOnTheCase(req)
		plans := pl.Where(
			keel.When(req.RunID.IsNotNil(), pl.RunID.Eq(req.RunID)),
			keel.When(
				req.RunID.IsNil(),
				keel.Or(
					pl.TargetID.Eq(req.RecordID),
					pl.RunID.InQuery(asked),
					keel.When(len(req.HandedRuns) > 0, pl.RunID.In(req.HandedRuns...)),
				),
			),
			keel.Or(pl.Status.In(livePlans...), pl.UpdatedAt.Gte(req.Since)),
		).Load(pl.Run()).Order(pl.CreatedAt.Desc()).Limit(maxDecisions)
		proposals := pr.Where(
			keel.When(req.RunID.IsNotNil(), pr.RunID.Eq(req.RunID)),
			keel.When(
				req.RunID.IsNil(),
				keel.Or(
					pr.TargetID.Eq(req.RecordID),
					// What an agent does for itself in a run (a wait it sets,
					// the options it offers) names no record and is not a
					// decision: only a change to a record is.
					keel.And(
						pr.TargetID.IsNotNull(),
						keel.Or(
							pr.RunID.InQuery(asked),
							keel.When(len(req.HandedRuns) > 0, pr.RunID.In(req.HandedRuns...)),
						),
					),
				),
			),
			pr.PlanID.IsNull(),
			keel.Or(pr.Status.In(liveProposals...), pr.UpdatedAt.Gte(req.Since)),
		).Load(pr.Run()).Order(pr.CreatedAt.Desc()).Limit(maxDecisions)

		if err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.ListInto(plans, &out.Plans),
			keel.ListInto(proposals, &out.Proposals),
		); err != nil {
			return nil, fmt.Errorf("list the record's decisions: %w", err)
		}
		if len(out.Proposals) == 0 {
			return out, nil
		}

		ids := make([]pulid.ID, 0, len(out.Proposals))
		for _, proposal := range out.Proposals {
			ids = append(ids, proposal.ID)
		}
		d := agentdecisiondb.AgentDecisions
		decided, err := d.Where(d.ProposalID.In(ids...)).
			Order(d.CreatedAt.Desc()).
			Limit(len(ids)*decisionsPerRecord).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list the proposals' decisions: %w", err)
		}
		for _, decision := range decided {
			if decision.ProposalID == nil {
				continue
			}
			if _, seen := out.Decided[*decision.ProposalID]; !seen {
				out.Decided[*decision.ProposalID] = decision
			}
		}

		return out, nil
	})
}
