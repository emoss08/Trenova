package aianalyticsservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aianalyticsports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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
	Providers   aiproviderports.AIProviderRepository
	Definitions agentdefinitionports.AgentDefinitionRepository
}

type Service struct {
	l           *zap.Logger
	reader      aianalyticsports.AnalyticsReader
	refresher   aianalyticsports.AnalyticsRefresher
	cache       aianalyticsports.AnalyticsCache
	providers   aiproviderports.AIProviderRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	clock       func() time.Time
}

var _ aianalyticsports.AnalyticsService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.ai-analytics"),
		reader:      p.Repo,
		refresher:   p.Repo,
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
	location, err := timeutils.LookupLocation(req.Timezone)
	if err != nil {
		multiErr.Add("timezone", errortypes.ErrInvalid, "Unknown timezone")
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
	suggestions         []aianalyticsports.SuggestionRow
	suggestionsRead     bool
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

type readRequest struct {
	request *aianalyticsports.AnalyticsRequest
	filter  aianalyticsports.Filter
	starts  []int64
	until   int64
	cache   *aianalyticsports.CacheKey
}

type submitRead func(fn func(context.Context) error)

func (s *Service) read(ctx context.Context, in *readRequest) (*readings, error) {
	out := new(readings)
	group, gctx := errgroup.WithContext(ctx)
	group.SetLimit(readConcurrency)
	run := func(fn func(context.Context) error) {
		group.Go(func() error { return fn(gctx) })
	}

	s.readAggregates(run, out, in)
	s.readReferences(run, out, in)

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("read ai analytics: %w", err)
	}

	return out, nil
}

func (s *Service) readAggregates(run submitRead, out *readings, in *readRequest) {
	req := in.request
	filter := in.filter

	run(func(ctx context.Context) error {
		var err error
		out.trend, err = s.reader.Trend(ctx, aianalyticsports.TrendRequest{
			Filter:       filter,
			Grouping:     req.CompareBy.Grouping(),
			BucketStarts: in.starts,
		})
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.deploymentDurations, err = s.reader.OutcomeDurations(
			ctx,
			filter,
			aianalytics.GroupingDeployment,
		)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.histograms, err = s.reader.FirstTokenHistograms(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.durations, err = s.reader.OutcomeDurations(ctx, filter, req.CompareBy.Grouping())
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.totals, err = s.reader.Totals(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.toolCells, err = s.reader.ToolCells(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.deployments, err = s.reader.Deployments(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.providerHours, err = s.reader.ProviderHours(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.providerCalls, err = s.reader.ProviderCalls(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.taskCells, err = s.reader.TaskCells(ctx, filter)
		return err
	})
}

func (s *Service) readReferences(run submitRead, out *readings, in *readRequest) {
	req := in.request
	filter := in.filter

	run(func(ctx context.Context) error {
		var err error
		out.palette, err = s.palette(ctx, req.TenantInfo, in.cache, in.until)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.versions, err = s.reader.ProviderVersions(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.providers, err = s.providers.ListOrdered(ctx, req.TenantInfo)
		return err
	})
	run(func(ctx context.Context) error {
		state, err := s.reader.RefreshState(ctx, req.TenantInfo)
		if err == nil && state != nil {
			refreshed := state.RefreshedAt
			out.state.refreshedAt = &refreshed
		}
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.decisions, err = s.reader.Decisions(ctx, filter)
		return err
	})
	run(func(ctx context.Context) error {
		var err error
		out.ratings, err = s.reader.Ratings(ctx, filter)
		return err
	})
	if req.Task != "" {
		return
	}
	run(func(ctx context.Context) error {
		rows, err := s.reader.Suggestions(ctx, &aianalyticsports.SuggestionRequest{
			Filter:       filter,
			PendingSince: aianalytics.SuggestionPendingSince(in.until),
		})
		out.suggestions, out.suggestionsRead = rows, err == nil
		return err
	})
}

func (s *Service) palette(
	ctx context.Context,
	tenant pagination.TenantInfo,
	key *aianalyticsports.CacheKey,
	until int64,
) ([]aianalyticsports.DeploymentKey, error) {
	if key != nil {
		cached, found, err := s.cache.Palette(ctx, tenant, key.Epoch)
		if err != nil {
			s.l.Warn("failed to read the cached ai analytics palette", zap.Error(err))
		} else if found {
			return cached, nil
		}
	}
	keys, err := s.reader.Palette(ctx, tenant, until-aianalytics.WindowMonth.Seconds())
	if err != nil {
		return nil, err
	}
	if key != nil {
		if setErr := s.cache.SetPalette(ctx, tenant, key.Epoch, keys); setErr != nil {
			s.l.Warn("failed to cache the ai analytics palette", zap.Error(setErr))
		}
	}

	return keys, nil
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
	now := s.clock().Unix()
	starts := aianalytics.BucketStarts(req.Window, location, now)

	key, cached := s.cached(ctx, &aianalyticsports.CacheKey{
		Request: req,
		Locale:  locale.String(),
		Buckets: aianalytics.BucketsDigest(starts),
	})
	if cached != nil {
		return cached, nil
	}

	filter := s.filter(req, now)
	data, err := s.read(ctx, &readRequest{
		request: req,
		filter:  filter,
		starts:  starts,
		until:   now,
		cache:   key,
	})
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
	key *aianalyticsports.CacheKey,
) (*aianalyticsports.CacheKey, *aianalyticsports.Analytics) {
	if s.cache == nil {
		return nil, nil
	}
	epoch, err := s.cache.Epoch(ctx, key.Request.TenantInfo)
	if err != nil {
		s.l.Warn("failed to read the ai analytics epoch", zap.Error(err))
		return nil, nil
	}
	key.Epoch = epoch
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
	var rows *aianalyticsports.ToolFailureRows
	var providers []*aiprovider.Provider
	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() (err error) {
		rows, err = s.reader.ToolFailures(gctx, aianalyticsports.ToolFailureRequest{
			Filter:    filter,
			SeriesKey: req.SeriesKey,
			ToolName:  req.ToolName,
			Limit:     recentFailureLimit,
		})
		return err
	})
	group.Go(func() (err error) {
		providers, err = s.providers.ListOrdered(gctx, req.TenantInfo)
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("read ai analytics tool failures: %w", err)
	}

	agents, err := s.agentNames(ctx, req.TenantInfo, rows.Examples)
	if err != nil {
		return nil, err
	}

	return failureDetail(failureDetailInput{
		request:   req,
		rows:      rows,
		agents:    agents,
		providers: providerIndex(i18n.FromContext(ctx), providers),
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
