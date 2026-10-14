package aianalyticsservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	readConcurrency    = 4
	recentFailureLimit = 8
	maxToolNameLength  = 200
	maxSeriesKeyLength = 200
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Repo        aianalyticsports.AnalyticsRepository
	Cache       aianalyticsports.AnalyticsCache `optional:"true"`
	Providers   agentports.AIProviderRepository
	Definitions agentdefinitionports.AgentDefinitionRepository
}

type Service struct {
	l           *zap.Logger
	repo        aianalyticsports.AnalyticsRepository
	cache       aianalyticsports.AnalyticsCache
	providers   agentports.AIProviderRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	clock       func() time.Time
}

var _ aianalyticsports.AnalyticsService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.ai-analytics"),
		repo:        p.Repo,
		cache:       p.Cache,
		providers:   p.Providers,
		definitions: p.Definitions,
		clock:       time.Now,
	}
}

func validateRequest(req *aianalyticsports.AnalyticsRequest) (*time.Location, error) {
	multiErr := errortypes.NewMultiError()
	if !req.Window.IsValid() {
		multiErr.Add("window", errortypes.ErrInvalid, "Unknown window")
	}
	if !req.CompareBy.IsValid() {
		multiErr.Add("compareBy", errortypes.ErrInvalid, "Unknown comparison")
	}
	if req.Task != "" && !req.Task.IsValid() {
		multiErr.Add("task", errortypes.ErrInvalid, "Unknown task")
	}
	location := time.UTC
	if req.Timezone != "" {
		loaded, err := time.LoadLocation(req.Timezone)
		if err != nil {
			multiErr.Add("timezone", errortypes.ErrInvalid, "Unknown timezone")
		} else {
			location = loaded
		}
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return location, nil
}

func (s *Service) filter(
	req *aianalyticsports.AnalyticsRequest,
	now int64,
) aianalyticsports.Filter {
	return aianalyticsports.Filter{
		TenantInfo:        req.TenantInfo,
		Since:             now - req.Window.Seconds(),
		CompareBy:         req.CompareBy,
		AgentDefinitionID: req.AgentDefinitionID,
		Task:              req.Task,
	}
}

type readings struct {
	state               stateReading
	totals              *aianalyticsports.TotalsRow
	deployments         []aianalyticsports.DeploymentRow
	durations           []aianalyticsports.OutcomeDurationRow
	deploymentDurations []aianalyticsports.OutcomeDurationRow
	trend               []aianalyticsports.TrendRow
	toolCells           []aianalyticsports.ToolCellRow
	taskCells           []aianalyticsports.TaskCellRow
	decisions           []aianalyticsports.DecisionRow
	ratings             []aianalyticsports.RatingRow
	providerCalls       []aianalyticsports.ProviderCallRow
	histograms          []aianalyticsports.HistogramRow
	providerHours       []aianalyticsports.ProviderHourRow
	versions            []aianalyticsports.ProviderVersionRow
	palette             []aianalyticsports.DeploymentKey
	providers           []*aiprovider.Provider
}

type stateReading struct {
	refreshedAt *int64
}

func (s *Service) read(
	ctx context.Context,
	req *aianalyticsports.AnalyticsRequest,
	filter aianalyticsports.Filter,
	starts []int64,
	until int64,
) (*readings, error) {
	out := new(readings)

	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(readConcurrency)
	run := func(fn func(context.Context) error) {
		group.Go(func() error { return fn(gctx) })
	}

	run(func(ctx context.Context) error {
		state, err := s.repo.RefreshState(ctx, req.TenantInfo)
		if err == nil && state != nil {
			refreshed := state.RefreshedAt
			out.state.refreshedAt = &refreshed
		}
		return err
	})
	run(
		func(ctx context.Context) (err error) { out.totals, err = s.repo.Totals(ctx, filter); return },
	)
	run(func(ctx context.Context) (err error) {
		out.deployments, err = s.repo.Deployments(ctx, filter)
		return
	})
	run(func(ctx context.Context) (err error) {
		out.durations, err = s.repo.OutcomeDurations(ctx, filter, req.CompareBy.Grouping())
		return
	})
	run(func(ctx context.Context) (err error) {
		out.deploymentDurations, err = s.repo.OutcomeDurations(
			ctx,
			filter,
			aianalytics.GroupingDeployment,
		)
		return
	})
	run(func(ctx context.Context) (err error) {
		out.trend, err = s.repo.Trend(ctx, aianalyticsports.TrendRequest{
			Filter:       filter,
			Grouping:     req.CompareBy.Grouping(),
			BucketStarts: starts,
		})
		return
	})
	run(
		func(ctx context.Context) (err error) { out.toolCells, err = s.repo.ToolCells(ctx, filter); return },
	)
	run(
		func(ctx context.Context) (err error) { out.taskCells, err = s.repo.TaskCells(ctx, filter); return },
	)
	run(
		func(ctx context.Context) (err error) { out.decisions, err = s.repo.Decisions(ctx, filter); return },
	)
	run(
		func(ctx context.Context) (err error) { out.ratings, err = s.repo.Ratings(ctx, filter); return },
	)
	run(func(ctx context.Context) (err error) {
		out.providerCalls, err = s.repo.ProviderCalls(ctx, filter)
		return
	})
	run(func(ctx context.Context) (err error) {
		out.histograms, err = s.repo.FirstTokenHistograms(ctx, filter)
		return
	})
	run(func(ctx context.Context) (err error) {
		out.providerHours, err = s.repo.ProviderHours(ctx, filter)
		return
	})
	run(func(ctx context.Context) (err error) {
		out.versions, err = s.repo.ProviderVersions(ctx, filter)
		return
	})
	run(func(ctx context.Context) (err error) {
		out.palette, err = s.repo.Palette(
			ctx,
			req.TenantInfo,
			until-aianalytics.WindowMonth.Seconds(),
		)
		return
	})
	run(func(ctx context.Context) (err error) {
		out.providers, err = s.providers.ListOrdered(ctx, req.TenantInfo)
		return
	})

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("read ai analytics: %w", err)
	}

	return out, nil
}

func (s *Service) Analytics(
	ctx context.Context,
	req *aianalyticsports.AnalyticsRequest,
) (*aianalyticsports.Analytics, error) {
	location, err := validateRequest(req)
	if err != nil {
		return nil, err
	}
	req.Timezone = location.String()
	locale := i18n.FromContext(ctx)

	key, cached := s.cached(ctx, req, locale)
	if cached != nil {
		return cached, nil
	}

	now := s.clock().Unix()
	filter := s.filter(req, now)
	starts := aianalytics.BucketStarts(req.Window, location, now)
	data, err := s.read(ctx, req, filter, starts, now)
	if err != nil {
		return nil, err
	}

	out := assemble(&assembly{
		locale:  locale,
		request: req,
		since:   filter.Since,
		until:   now,
		buckets: starts,
		data:    data,
	})
	if key != nil {
		if setErr := s.cache.Set(ctx, key, out); setErr != nil {
			s.l.Warn("failed to cache ai analytics", zap.Error(setErr))
		}
	}

	return out, nil
}

func (s *Service) cached(
	ctx context.Context,
	req *aianalyticsports.AnalyticsRequest,
	locale i18n.Locale,
) (*aianalyticsports.CacheKey, *aianalyticsports.Analytics) {
	if s.cache == nil {
		return nil, nil
	}
	epoch, err := s.cache.Epoch(ctx, req.TenantInfo)
	if err != nil {
		s.l.Warn("failed to read the ai analytics epoch", zap.Error(err))
		return nil, nil
	}
	key := &aianalyticsports.CacheKey{Request: req, Epoch: epoch, Locale: locale.String()}
	out := new(aianalyticsports.Analytics)
	found, err := s.cache.Get(ctx, key, out)
	if err != nil {
		s.l.Warn("failed to read cached ai analytics", zap.Error(err))
		return key, nil
	}
	if !found {
		return key, nil
	}

	return key, out
}

func (s *Service) ToolFailures(
	ctx context.Context,
	req *aianalyticsports.ToolFailureDetailRequest,
) (*aianalyticsports.ToolFailureDetail, error) {
	if _, err := validateRequest(&req.AnalyticsRequest); err != nil {
		return nil, err
	}
	multiErr := errortypes.NewMultiError()
	if req.ToolName == "" || len(req.ToolName) > maxToolNameLength {
		multiErr.Add("toolName", errortypes.ErrInvalid, "Name a tool")
	}
	if len(req.SeriesKey) > maxSeriesKeyLength {
		multiErr.Add("seriesKey", errortypes.ErrInvalid, "Unknown series")
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	filter := s.filter(&req.AnalyticsRequest, s.clock().Unix())
	rows, err := s.repo.ToolFailures(ctx, aianalyticsports.ToolFailureRequest{
		Filter:    filter,
		SeriesKey: req.SeriesKey,
		ToolName:  req.ToolName,
		Limit:     recentFailureLimit,
	})
	if err != nil {
		return nil, err
	}

	agents, err := s.agentNames(ctx, req.TenantInfo, rows.Examples)
	if err != nil {
		return nil, err
	}
	providers, err := s.providers.ListOrdered(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	deployments, err := s.repo.Deployments(ctx, filter)
	if err != nil {
		return nil, err
	}

	return failureDetail(failureDetailInput{
		request:     req,
		rows:        rows,
		agents:      agents,
		providers:   providerIndex(i18n.FromContext(ctx), providers),
		deployments: deployments,
	}), nil
}

func (s *Service) agentNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	examples []aianalyticsports.ToolFailureExample,
) (map[pulid.ID]string, error) {
	ids := make([]pulid.ID, 0, len(examples))
	seen := make(map[pulid.ID]struct{}, len(examples))
	for idx := range examples {
		id := examples[idx].AgentDefinitionID
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	names := make(map[pulid.ID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}

	definitions, err := s.definitions.ListByIDs(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByIDsRequest{
			IDs:        ids,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, definition := range definitions {
		names[definition.ID] = definition.Name
	}

	return names, nil
}
