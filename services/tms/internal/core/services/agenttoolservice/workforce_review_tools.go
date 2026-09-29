package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/performancereviewservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramReviewID       = "reviewId"
	paramReviewTemplate = "templateId"
	paramRatings        = "ratings"
	paramRatingKey      = "key"
	paramRatingScore    = "score"
	paramRatingComment  = "comment"
	paramReviewSummary  = "summary"
	paramStrengths      = "strengths"
	paramImprovements   = "improvements"
	paramGoals          = "goals"
	paramGoalTitle      = "title"
	paramGoalDueAt      = "dueAt"
	paramGoalStatus     = "status"
	kindReview          = "performance review"
	maxReviewRatings    = 30
	maxReviewGoals      = 10
	minReviewScore      = 1
	maxReviewScore      = 5
)

var (
	reviewGoalStatuses = agenttoolschema.Source(
		"worker.reviewGoalStatus",
		worker.ReviewGoalStatusValues(),
	)
	reviewFields = []string{
		wfFieldWorkerID, paramReviewTemplate, "reviewerId", fieldStatus, paramTitle,
		fieldPeriodStart, fieldPeriodEnd, paramRatings, "overallScore", paramReviewSummary,
		paramStrengths, paramImprovements, paramGoals,
	}
)

type reviewKeeper interface {
	GetReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.PerformanceReview, error)
	PlanCreateReview(
		ctx context.Context,
		req *performancereviewservice.CreateReviewRequest,
	) (*worker.PerformanceReview, error)
	CreateReview(
		ctx context.Context,
		req *performancereviewservice.CreateReviewRequest,
	) (*worker.PerformanceReview, error)
	PlanUpdateReview(
		ctx context.Context,
		req *performancereviewservice.UpdateReviewRequest,
	) (*performancereviewservice.ReviewChange, error)
	UpdateReview(
		ctx context.Context,
		req *performancereviewservice.UpdateReviewRequest,
	) (*worker.PerformanceReview, error)
	PlanDeleteReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*worker.PerformanceReview, error)
	DeleteReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

var _ reviewKeeper = (*performancereviewservice.Service)(nil)

func reviewToolProviders() []any {
	return []any{
		provideStartPerformanceReviewTool,
		provideDraftPerformanceReviewTool,
		provideDeletePerformanceReviewTool,
	}
}

func reviewIDProperty() map[string]any {
	return idProperty("The review, from list_performance_reviews. Never guess one.")
}

func reviewRecord(review *worker.PerformanceReview) toolpreview.Record {
	return wfRecord(permission.ResourcePerformanceReview, review.ID, review.Title, review.Version)
}

func newStartPerformanceReviewTool(reviews reviewKeeper) serviceports.AgentTool {
	spec := withSchema(wfSpec(
		"start_performance_review",
		"Open a draft performance review for a worker from a review template. It covers "+
			"a period, with the person who approves it as the reviewer. "+
			"The template's items are copied on unrated; draft_performance_review fills them "+
			"in. The worker sees nothing until the reviewer submits it.",
		"Opens a draft review inside Trenova; the worker sees nothing, and "+
			"delete_performance_review removes the draft.",
		permission.ResourcePerformanceReview,
		permission.OpCreate,
	), map[string]any{
		paramWorkerID: workerProperty(),
		paramReviewTemplate: idProperty("The review template, from list_performance_reviews. " +
			"Never guess one."),
		paramTitle:       stringProperty("A title; defaults to the template and period.", 120),
		fieldPeriodStart: dayProperty("The first day the review covers."),
		fieldPeriodEnd:   dayProperty("The last day the review covers."),
	}, paramWorkerID, paramReviewTemplate, fieldPeriodStart, fieldPeriodEnd)

	return newReportingReceivableTool(spec, receivablePlan[
		*performancereviewservice.CreateReviewRequest, *worker.PerformanceReview,
	]{
		request: func(
			params *serviceports.ToolExecuteParams,
		) (*performancereviewservice.CreateReviewRequest, error) {
			workerID, err := requirePulid(params.Params, paramWorkerID)
			if err != nil {
				return nil, err
			}
			templateID, err := requirePulid(params.Params, paramReviewTemplate)
			if err != nil {
				return nil, err
			}
			start, err := requireScheduleDay(params.Params, fieldPeriodStart)
			if err != nil {
				return nil, err
			}
			end, err := requireScheduleDay(params.Params, fieldPeriodEnd)
			if err != nil {
				return nil, err
			}
			title, err := boundedText(params.Params, paramTitle, 120)
			if err != nil {
				return nil, err
			}
			return &performancereviewservice.CreateReviewRequest{
				TenantInfo:  tenantFrom(*params),
				WorkerID:    workerID,
				TemplateID:  templateID,
				Title:       title,
				PeriodStart: start,
				PeriodEnd:   end,
				UserID:      params.Actor.UserID,
			}, nil
		},
		plan: func(
			ctx context.Context,
			req *performancereviewservice.CreateReviewRequest,
			_ *serviceports.ToolExecuteParams,
		) (*worker.PerformanceReview, error) {
			return reviews.PlanCreateReview(ctx, req)
		},
		refused: func(*performancereviewservice.CreateReviewRequest) string {
			return "Would open a draft performance review."
		},
		render: func(
			_ *performancereviewservice.CreateReviewRequest,
			planned *worker.PerformanceReview,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				wfRecord(permission.ResourcePerformanceReview, pulid.Nil, planned.Title, 0),
				planned, wfOptions(reviewFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf(
				"Would open %q as a draft with %d item(s) to rate.", planned.Title,
				len(planned.Ratings)), change), nil
		},
		run: func(
			ctx context.Context,
			req *performancereviewservice.CreateReviewRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := reviews.CreateReview(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("drafted", kindReview, paramReviewID, created.ID,
				created.WorkerID), nil
		},
	})
}

type reviewRatingParam struct {
	Key     string `json:"key"`
	Score   *int32 `json:"score"`
	Comment string `json:"comment"`
}

type reviewGoalParam struct {
	Title  string `json:"title"`
	DueAt  string `json:"dueAt"`
	Status string `json:"status"`
}

type reviewDraft struct {
	id     pulid.ID
	params *serviceports.ToolExecuteParams
}

func (d *reviewDraft) request(
	ctx context.Context,
	reviews reviewKeeper,
) (*performancereviewservice.UpdateReviewRequest, error) {
	current, err := reviews.GetReview(ctx, tenantFrom(*d.params), d.id)
	if err != nil {
		return nil, err
	}
	req := &performancereviewservice.UpdateReviewRequest{
		TenantInfo:   tenantFrom(*d.params),
		ID:           d.id,
		Ratings:      append([]worker.ReviewRating(nil), current.Ratings...),
		Summary:      current.Summary,
		Strengths:    current.Strengths,
		Improvements: current.Improvements,
		Goals:        append([]worker.ReviewGoal(nil), current.Goals...),
		Version:      current.Version,
		UserID:       d.params.Actor.UserID,
	}
	if req.Title, err = boundedText(d.params.Params, paramTitle, 120); err != nil {
		return nil, err
	}
	if err = applyReviewText(req, d.params.Params); err != nil {
		return nil, err
	}
	if err = applyReviewRatings(req, d.params.Params); err != nil {
		return nil, err
	}
	return req, applyReviewGoals(req, d.params.Params)
}

func applyReviewText(
	req *performancereviewservice.UpdateReviewRequest,
	params map[string]any,
) error {
	for _, field := range []struct {
		key  string
		dest *string
	}{
		{paramReviewSummary, &req.Summary},
		{paramStrengths, &req.Strengths},
		{paramImprovements, &req.Improvements},
	} {
		text, err := optionalBoundedText(params, field.key, wfNoteChars)
		if err != nil {
			return err
		}
		if text != nil {
			*field.dest = *text
		}
	}
	return nil
}

func applyReviewRatings(
	req *performancereviewservice.UpdateReviewRequest,
	params map[string]any,
) error {
	if _, given := params[paramRatings]; !given {
		return nil
	}
	var ratings []reviewRatingParam
	if err := decodeParam(params, paramRatings, &ratings); err != nil {
		return err
	}
	byKey := make(map[string]int, len(req.Ratings))
	for i := range req.Ratings {
		byKey[req.Ratings[i].Key] = i
	}
	for i, rating := range ratings {
		index, known := byKey[rating.Key]
		if !known {
			return errortypes.NewValidationError(
				fmt.Sprintf("%s[%d].%s", paramRatings, i, paramRatingKey),
				errortypes.ErrInvalid,
				"This review has no item {0}", rating.Key,
			)
		}
		if rating.Score != nil &&
			(*rating.Score < minReviewScore || *rating.Score > maxReviewScore) {
			return fmt.Errorf("parameter %s[%d].%s must be %d to %d", paramRatings, i,
				paramRatingScore, minReviewScore, maxReviewScore)
		}
		req.Ratings[index].Score = rating.Score
		req.Ratings[index].Comment = rating.Comment
	}
	return nil
}

func applyReviewGoals(
	req *performancereviewservice.UpdateReviewRequest,
	params map[string]any,
) error {
	if _, given := params[paramGoals]; !given {
		return nil
	}
	var goals []reviewGoalParam
	if err := decodeParam(params, paramGoals, &goals); err != nil {
		return err
	}
	req.Goals = make([]worker.ReviewGoal, 0, len(goals))
	for i, goal := range goals {
		entry := worker.ReviewGoal{Title: goal.Title, Status: worker.ReviewGoalStatusOpen}
		if goal.Status != "" {
			status, err := requireEnum(map[string]any{paramGoalStatus: goal.Status},
				paramGoalStatus, reviewGoalStatuses.Values)
			if err != nil {
				return fmt.Errorf("%s[%d]: %w", paramGoals, i, err)
			}
			entry.Status = status
		}
		if goal.DueAt != "" {
			due, err := requireScheduleDay(map[string]any{paramGoalDueAt: goal.DueAt},
				paramGoalDueAt)
			if err != nil {
				return fmt.Errorf("%s[%d]: %w", paramGoals, i, err)
			}
			entry.DueAt = &due
		}
		req.Goals = append(req.Goals, entry)
	}
	return nil
}

func reviewDraftProperties() map[string]any {
	return map[string]any{
		paramReviewID: reviewIDProperty(),
		paramTitle:    stringProperty("A new title.", 120),
		paramRatings: map[string]any{
			toolschema.KeyType: toolschema.TypeArray,
			toolschema.KeyDescription: "Scores for the review's items, each by the key " +
				"list_performance_reviews shows. Items left out keep what they have.",
			toolschema.KeyMaxItems: maxReviewRatings,
			toolschema.KeyItems: map[string]any{
				toolschema.KeyType: toolschema.TypeObject,
				toolschema.KeyProperties: map[string]any{
					paramRatingKey: stringProperty("The item's key.", wfShortChars),
					paramRatingScore: integerProperty("The score, 1 to 5.", minReviewScore,
						maxReviewScore),
					paramRatingComment: stringProperty("Why that score, from the record.",
						wfNoteChars),
				},
				toolschema.KeyRequired:             []string{paramRatingKey},
				toolschema.KeyAdditionalProperties: false,
			},
		},
		paramReviewSummary: wfNoteProperty("The overall summary."),
		paramStrengths:     wfNoteProperty("What went well."),
		paramImprovements:  wfNoteProperty("What to work on."),
		paramGoals: map[string]any{
			toolschema.KeyType: toolschema.TypeArray,
			toolschema.KeyDescription: "The goals for the next period. Given, they replace " +
				"the review's goals; left out, the goals stay.",
			toolschema.KeyMaxItems: maxReviewGoals,
			toolschema.KeyItems: map[string]any{
				toolschema.KeyType: toolschema.TypeObject,
				toolschema.KeyProperties: map[string]any{
					paramGoalTitle: stringProperty("The goal.", wfShortChars),
					paramGoalDueAt: dayProperty("When it is due."),
					paramGoalStatus: agenttoolschema.Enum("Open, Done or Dropped. Defaults "+
						"to Open.", reviewGoalStatuses),
				},
				toolschema.KeyRequired:             []string{paramGoalTitle},
				toolschema.KeyAdditionalProperties: false,
			},
		},
	}
}

func newDraftPerformanceReviewTool(reviews reviewKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"draft_performance_review",
		"Fill in a draft performance review: score its items 1 to 5 with a comment each, "+
			"and write the summary, strengths, improvements and goals. Only a draft is edited, "+
			"and only what you give changes. Draw every score from the record, such as the "+
			"safety scorecard and training, and say so in the comment; the reviewer submits "+
			"it, not you.",
		"Saves the reviewer's draft inside Trenova; the worker sees nothing until the reviewer "+
			"submits it, and the draft is edited again the same way.",
		permission.ResourcePerformanceReview,
		permission.OpUpdate,
	), reviewDraftProperties(), paramReviewID), paramReviewID, permission.ResourcePerformanceReview)

	spec.searchTerms = []string{"review scores", "score the review", "fill in review"}

	return newReportingReceivableTool(spec, receivablePlan[
		*reviewDraft, *performancereviewservice.ReviewChange,
	]{
		request: func(params *serviceports.ToolExecuteParams) (*reviewDraft, error) {
			id, err := requirePulid(params.Params, paramReviewID)
			if err != nil {
				return nil, err
			}
			return &reviewDraft{id: id, params: params}, nil
		},
		plan: func(
			ctx context.Context,
			draft *reviewDraft,
			_ *serviceports.ToolExecuteParams,
		) (*performancereviewservice.ReviewChange, error) {
			req, err := draft.request(ctx, reviews)
			if err != nil {
				return nil, err
			}
			return reviews.PlanUpdateReview(ctx, req)
		},
		refused: func(*reviewDraft) string { return "Would save the review draft." },
		render: func(
			_ *reviewDraft,
			change *performancereviewservice.ReviewChange,
		) (*agent.ToolPreview, error) {
			recorded, err := toolpreview.Changed(reviewRecord(change.Before), change.Before,
				change.After, wfOptions(reviewFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would save the draft of %q.",
				change.Before.Title), recorded), nil
		},
		run: func(
			ctx context.Context,
			draft *reviewDraft,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := draft.request(ctx, reviews)
			if err != nil {
				return nil, err
			}
			saved, err := reviews.UpdateReview(ctx, req)
			if err != nil {
				return nil, err
			}
			return wfResult("saved", kindReview, paramReviewID, saved.ID, saved.WorkerID), nil
		},
	})
}

func newDeletePerformanceReviewTool(reviews reviewKeeper) serviceports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"delete_performance_review",
		"Delete a draft performance review started in error. A submitted review is part "+
			"of the record and is refused.",
		"Removes a draft review nobody has seen; the audit trail keeps what was removed.",
		permission.ResourcePerformanceReview,
		permission.OpDelete,
	), map[string]any{paramReviewID: reviewIDProperty()}, paramReviewID), paramReviewID,
		permission.ResourcePerformanceReview)
	spec.maxTier = agent.TierPropose
	spec.reversible = false

	return newReceivableTool(spec, receivablePlan[*recordDelete, *worker.PerformanceReview]{
		request: recordDeleteFrom(paramReviewID),
		plan: func(
			ctx context.Context,
			req *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*worker.PerformanceReview, error) {
			return reviews.PlanDeleteReview(ctx, req.tenant, req.id)
		},
		refused: func(*recordDelete) string { return "Would delete a draft review." },
		render: func(_ *recordDelete, review *worker.PerformanceReview) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(reviewRecord(review), review,
				wfOptions(reviewFields...)...)
			if err != nil {
				return nil, err
			}
			return toolpreview.Build(fmt.Sprintf("Would delete the draft %q.", review.Title),
				change), nil
		},
		run: func(
			ctx context.Context,
			req *recordDelete,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			return nil, reviews.DeleteReview(ctx, req.tenant, req.id, req.userID)
		},
	})
}

func provideStartPerformanceReviewTool(
	s *performancereviewservice.Service,
) serviceports.AgentTool {
	return newStartPerformanceReviewTool(s)
}

func provideDraftPerformanceReviewTool(
	s *performancereviewservice.Service,
) serviceports.AgentTool {
	return newDraftPerformanceReviewTool(s)
}

func provideDeletePerformanceReviewTool(
	s *performancereviewservice.Service,
) serviceports.AgentTool {
	return newDeletePerformanceReviewTool(s)
}
