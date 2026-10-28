package workforcetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramReviewID = "reviewId"

const paramReviewTemplate = "templateId"

const paramRatings = "ratings"

const paramRatingKey = "key"

const paramRatingScore = "score"

const paramRatingComment = "comment"

const paramReviewSummary = "summary"

const paramStrengths = "strengths"

const paramImprovements = "improvements"

const paramGoals = "goals"

const paramGoalTitle = "title"

const paramGoalDueAt = "dueAt"

const paramGoalStatus = "status"

const kindReview = "performance review"

const maxReviewRatings = 30

const maxReviewGoals = 10

const minReviewScore = 1

const maxReviewScore = 5

var reviewGoalStatuses = agenttoolschema.Source(
	"worker.reviewGoalStatus",
	workerreview.ReviewGoalStatusValues(),
)

var reviewFields = []string{
	wfFieldWorkerID,
	paramReviewTemplate,
	"reviewerId",
	changepreview.FieldStatus,
	paramTitle,
	changepreview.FieldPeriodStart,
	changepreview.FieldPeriodEnd,
	paramRatings,
	"overallScore",
	paramReviewSummary,
	paramStrengths,
	paramImprovements,
	paramGoals,
}

type ReviewKeeper interface {
	GetReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerreview.PerformanceReview, error)
	PlanCreateReview(
		ctx context.Context,
		req *performancereviewservice.CreateReviewRequest,
	) (*workerreview.PerformanceReview, error)
	CreateReview(
		ctx context.Context,
		req *performancereviewservice.CreateReviewRequest,
	) (*workerreview.PerformanceReview, error)
	PlanUpdateReview(
		ctx context.Context,
		req *performancereviewservice.UpdateReviewRequest,
	) (*performancereviewservice.ReviewChange, error)
	UpdateReview(
		ctx context.Context,
		req *performancereviewservice.UpdateReviewRequest,
	) (*workerreview.PerformanceReview, error)
	PlanDeleteReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*workerreview.PerformanceReview, error)
	DeleteReview(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		userID pulid.ID,
	) error
}

func ReviewToolProviders() []any {
	return []any{
		provideStartPerformanceReviewTool,
		provideDraftPerformanceReviewTool,
		provideDeletePerformanceReviewTool,
	}
}

func reviewIDProperty() map[string]any {
	return agenttoolschema.RecordID(permission.ResourcePerformanceReview, "The review",
		"worker_list_performance_reviews")
}

func reviewRecord(review *workerreview.PerformanceReview) toolpreview.Record {
	return wfRecord(permission.ResourcePerformanceReview, review.ID, review.Title, review.Version)
}

func newStartPerformanceReviewTool(reviews ReviewKeeper) agenttoolports.AgentTool {
	spec := withSchema(wfSpec(
		"worker_start_performance_review",
		"Open a draft performance review for a worker from a review template. It covers "+
			"a period, with the person who approves it as the reviewer. "+
			"The template's items are copied on unrated; worker_update_performance_review fills them "+
			"in. The worker sees nothing until the reviewer submits it.",
		"Opens a draft review inside Trenova; the worker sees nothing, and "+
			"worker_delete_performance_review removes the draft.",
		permission.ResourcePerformanceReview,
		permission.OpCreate,
	), map[string]any{
		toolparams.ParamWorkerID: toolparams.WorkerProperty(),
		paramReviewTemplate: agenttoolschema.RecordIDText(
			permission.ResourcePerformanceReviewTemplate,
			"The review template, from worker_list_performance_reviews. "+
				"Never guess one.",
		),
		paramTitle: toolparams.StringProperty(
			"A title; defaults to the template and period.",
			120,
		),
		changepreview.FieldPeriodStart: agenttoolschema.Date("The first day the review covers."),
		changepreview.FieldPeriodEnd:   agenttoolschema.Date("The last day the review covers."),
	}, toolparams.ParamWorkerID, paramReviewTemplate, changepreview.FieldPeriodStart, changepreview.FieldPeriodEnd)

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*performancereviewservice.CreateReviewRequest, *workerreview.PerformanceReview,
	]{
		Request: func(
			params *agenttoolports.ToolExecuteParams,
		) (*performancereviewservice.CreateReviewRequest, error) {
			workerID, err := toolparams.RequirePulid(params.Params, toolparams.ParamWorkerID)
			if err != nil {
				return nil, err
			}
			templateID, err := toolparams.RequirePulid(params.Params, paramReviewTemplate)
			if err != nil {
				return nil, err
			}
			start, err := toolparams.RequireScheduleDay(
				params.Params,
				changepreview.FieldPeriodStart,
			)
			if err != nil {
				return nil, err
			}
			end, err := toolparams.RequireScheduleDay(params.Params, changepreview.FieldPeriodEnd)
			if err != nil {
				return nil, err
			}
			title, err := toolparams.BoundedText(params.Params, paramTitle, 120)
			if err != nil {
				return nil, err
			}
			return &performancereviewservice.CreateReviewRequest{
				TenantInfo:  toolparams.TenantFrom(*params),
				WorkerID:    workerID,
				TemplateID:  templateID,
				Title:       title,
				PeriodStart: start,
				PeriodEnd:   end,
				UserID:      params.Actor.UserID,
			}, nil
		},
		Plan: func(
			ctx context.Context,
			req *performancereviewservice.CreateReviewRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*workerreview.PerformanceReview, error) {
			return reviews.PlanCreateReview(ctx, req)
		},
		Refused: func(*performancereviewservice.CreateReviewRequest) string {
			return "Would open a draft performance review."
		},
		Render: func(
			_ *performancereviewservice.CreateReviewRequest,
			planned *workerreview.PerformanceReview,
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
		Run: func(
			ctx context.Context,
			req *performancereviewservice.CreateReviewRequest,
			_ *agenttoolports.ToolExecuteParams,
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
	params *agenttoolports.ToolExecuteParams
}

func (d *reviewDraft) request(
	ctx context.Context,
	reviews ReviewKeeper,
) (*performancereviewservice.UpdateReviewRequest, error) {
	current, err := reviews.GetReview(ctx, toolparams.TenantFrom(*d.params), d.id)
	if err != nil {
		return nil, err
	}
	req := &performancereviewservice.UpdateReviewRequest{
		TenantInfo:   toolparams.TenantFrom(*d.params),
		ID:           d.id,
		Ratings:      append([]workerreview.ReviewRating(nil), current.Ratings...),
		Summary:      current.Summary,
		Strengths:    current.Strengths,
		Improvements: current.Improvements,
		Goals:        append([]workerreview.ReviewGoal(nil), current.Goals...),
		Version:      current.Version,
		UserID:       d.params.Actor.UserID,
	}
	if req.Title, err = toolparams.BoundedText(d.params.Params, paramTitle, 120); err != nil {
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
		text, err := toolparams.OptionalBoundedText(params, field.key, wfNoteChars)
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
	if err := toolparams.DecodeParam(params, paramRatings, &ratings); err != nil {
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
	if err := toolparams.DecodeParam(params, paramGoals, &goals); err != nil {
		return err
	}
	req.Goals = make([]workerreview.ReviewGoal, 0, len(goals))
	for i, goal := range goals {
		entry := workerreview.ReviewGoal{
			Title:  goal.Title,
			Status: workerreview.ReviewGoalStatusOpen,
		}
		if goal.Status != "" {
			status, err := toolparams.RequireEnum(map[string]any{paramGoalStatus: goal.Status},
				paramGoalStatus, reviewGoalStatuses.Values)
			if err != nil {
				return fmt.Errorf("%s[%d]: %w", paramGoals, i, err)
			}
			entry.Status = status
		}
		if goal.DueAt != "" {
			due, err := toolparams.RequireScheduleDay(map[string]any{paramGoalDueAt: goal.DueAt},
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
		paramTitle:    toolparams.StringProperty("A new title.", 120),
		paramRatings: map[string]any{
			toolschema.KeyType: toolschema.TypeArray,
			toolschema.KeyDescription: "Scores for the review's items, each by the key " +
				"worker_list_performance_reviews shows. Items left out keep what they have.",
			toolschema.KeyMaxItems: maxReviewRatings,
			toolschema.KeyItems: map[string]any{
				toolschema.KeyType: toolschema.TypeObject,
				toolschema.KeyProperties: map[string]any{
					paramRatingKey: toolparams.StringProperty("The item's key.", wfShortChars),
					paramRatingScore: toolparams.IntegerProperty(
						"The score, 1 to 5.",
						minReviewScore,
						maxReviewScore,
					),
					paramRatingComment: toolparams.StringProperty(
						"Why that score, from the record.",
						wfNoteChars,
					),
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
					paramGoalTitle: toolparams.StringProperty("The goal.", wfShortChars),
					paramGoalDueAt: agenttoolschema.Date("When it is due."),
					paramGoalStatus: agenttoolschema.Enum("Open, Done or Dropped. Defaults "+
						"to Open.", reviewGoalStatuses),
				},
				toolschema.KeyRequired:             []string{paramGoalTitle},
				toolschema.KeyAdditionalProperties: false,
			},
		},
	}
}

func newDraftPerformanceReviewTool(reviews ReviewKeeper) agenttoolports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"worker_update_performance_review",
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

	spec.SearchTerms = []string{"review scores", "score the review", "fill in review"}
	spec.Recipe = []string{"worker_list_performance_reviews", "worker_update_performance_review"}

	return toolspec.NewReportingReceivableTool(spec, toolspec.ReceivablePlan[
		*reviewDraft, *performancereviewservice.ReviewChange,
	]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*reviewDraft, error) {
			id, err := toolparams.RequirePulid(params.Params, paramReviewID)
			if err != nil {
				return nil, err
			}
			return &reviewDraft{id: id, params: params}, nil
		},
		Plan: func(
			ctx context.Context,
			draft *reviewDraft,
			_ *agenttoolports.ToolExecuteParams,
		) (*performancereviewservice.ReviewChange, error) {
			req, err := draft.request(ctx, reviews)
			if err != nil {
				return nil, err
			}
			return reviews.PlanUpdateReview(ctx, req)
		},
		Refused: func(*reviewDraft) string { return "Would save the review draft." },
		Render: func(
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
		Run: func(
			ctx context.Context,
			draft *reviewDraft,
			_ *agenttoolports.ToolExecuteParams,
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

func newDeletePerformanceReviewTool(reviews ReviewKeeper) agenttoolports.AgentTool {
	spec := targeting(withSchema(wfSpec(
		"worker_delete_performance_review",
		"Delete a draft performance review started in error. A submitted review is part "+
			"of the record and is refused.",
		"Removes a draft review nobody has seen; the audit trail keeps what was removed.",
		permission.ResourcePerformanceReview,
		permission.OpDelete,
	), map[string]any{paramReviewID: reviewIDProperty()}, paramReviewID), paramReviewID,
		permission.ResourcePerformanceReview)
	spec.MaxTier = agent.TierPropose
	spec.Reversible = false

	return toolspec.NewReceivableTool(
		spec,
		toolspec.ReceivablePlan[*recordDelete, *workerreview.PerformanceReview]{
			Request: recordDeleteFrom(paramReviewID),
			Plan: func(
				ctx context.Context,
				req *recordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*workerreview.PerformanceReview, error) {
				return reviews.PlanDeleteReview(ctx, req.tenant, req.id)
			},
			Refused: func(*recordDelete) string { return "Would delete a draft review." },
			Render: func(_ *recordDelete, review *workerreview.PerformanceReview) (*agent.ToolPreview, error) {
				change, err := toolpreview.Delete(reviewRecord(review), review,
					wfOptions(reviewFields...)...)
				if err != nil {
					return nil, err
				}
				return toolpreview.Build(fmt.Sprintf("Would delete the draft %q.", review.Title),
					change), nil
			},
			Run: func(
				ctx context.Context,
				req *recordDelete,
				_ *agenttoolports.ToolExecuteParams,
			) (*agent.ToolExecutionResult, error) {
				return nil, reviews.DeleteReview(ctx, req.tenant, req.id, req.userID)
			},
		},
	)
}

func provideStartPerformanceReviewTool(
	s *performancereviewservice.Service,
) agenttoolports.AgentTool {
	return newStartPerformanceReviewTool(s)
}

func provideDraftPerformanceReviewTool(
	s *performancereviewservice.Service,
) agenttoolports.AgentTool {
	return newDraftPerformanceReviewTool(s)
}

func provideDeletePerformanceReviewTool(
	s *performancereviewservice.Service,
) agenttoolports.AgentTool {
	return newDeletePerformanceReviewTool(s)
}
