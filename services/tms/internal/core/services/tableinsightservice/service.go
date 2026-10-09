package tableinsightservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger  *zap.Logger
	Sources []repositories.TableInsightSource `group:"table_insight_sources"`
}

type Service struct {
	l       *zap.Logger
	sources map[permission.Resource]repositories.TableInsightSource
}

func New(p Params) *Service {
	sources := make(map[permission.Resource]repositories.TableInsightSource, len(p.Sources))
	for _, source := range p.Sources {
		if source == nil {
			continue
		}
		sources[source.Resource()] = source
	}

	return &Service{
		l:       p.Logger.Named("service.tableinsight"),
		sources: sources,
	}
}

func (s *Service) Resolve(resource string) (permission.Resource, error) {
	key := permission.Resource(resource)
	if _, ok := s.sources[key]; !ok {
		return "", errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			fmt.Sprintf("Table %q has no counts or totals", resource),
		)
	}
	return key, nil
}

func (s *Service) Fields(resource permission.Resource) []repositories.TableInsightField {
	source, ok := s.sources[resource]
	if !ok {
		return nil
	}
	return source.InsightFields()
}

func (s *Service) Facet(
	ctx context.Context,
	resource permission.Resource,
	req *repositories.TableFacetRequest,
) (*repositories.TableFacetResult, error) {
	source, ok := s.sources[resource]
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"This table has no counts",
		)
	}

	result, err := source.Facet(ctx, req)
	if err != nil {
		s.l.Debug("table facet failed",
			zap.String("resource", string(resource)),
			zap.String("field", req.Field),
			zap.Error(err),
		)
		return nil, err
	}
	return result, nil
}

func (s *Service) Aggregate(
	ctx context.Context,
	resource permission.Resource,
	req *repositories.TableAggregateRequest,
) (*repositories.TableAggregateResult, error) {
	source, ok := s.sources[resource]
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"This table has no totals",
		)
	}

	if len(req.Fields) == 0 {
		return &repositories.TableAggregateResult{}, nil
	}

	result, err := source.Aggregate(ctx, req)
	if err != nil {
		s.l.Debug("table aggregate failed",
			zap.String("resource", string(resource)),
			zap.Strings("fields", req.Fields),
			zap.Error(err),
		)
		return nil, err
	}
	return result, nil
}

func (s *Service) MatchingIDs(
	ctx context.Context,
	resource permission.Resource,
	req *repositories.TableMatchRequest,
) (*repositories.TableMatchResult, error) {
	source, ok := s.sources[resource]
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"This table cannot find rows by its filters",
		)
	}
	return source.MatchingIDs(ctx, req)
}

type SeriesInterval string

const (
	SeriesWeek  SeriesInterval = "week"
	SeriesMonth SeriesInterval = "month"
)

type SeriesRequest struct {
	Scope       repositories.TableInsightScope
	GroupField  string
	GroupValues []string
	DateField   string
	ValueField  string
	Interval    SeriesInterval
	Periods     int
	Timezone    string
	Now         time.Time
}

type SeriesResult struct {
	PeriodStarts []int64
	Groups       []repositories.TableSeriesGroup
}

func (s *Service) Series(
	ctx context.Context,
	resource permission.Resource,
	req *SeriesRequest,
) (*SeriesResult, error) {
	source, ok := s.sources[resource]
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"This table cannot be charted",
		)
	}

	bounds, err := SeriesBounds(req.Now, req.Timezone, req.Interval, req.Periods)
	if err != nil {
		return nil, err
	}

	result, err := source.Series(ctx, &repositories.TableSeriesRequest{
		Scope:       req.Scope,
		GroupField:  req.GroupField,
		GroupValues: req.GroupValues,
		DateField:   req.DateField,
		ValueField:  req.ValueField,
		Bounds:      bounds,
	})
	if err != nil {
		s.l.Debug("table series failed",
			zap.String("resource", string(resource)),
			zap.String("groupField", req.GroupField),
			zap.String("dateField", req.DateField),
			zap.Error(err),
		)
		return nil, err
	}

	return &SeriesResult{PeriodStarts: bounds[:len(bounds)-1], Groups: result.Groups}, nil
}

func SeriesBounds(
	now time.Time,
	timezone string,
	interval SeriesInterval,
	periods int,
) ([]int64, error) {
	if periods < 1 || periods > dbhelper.MaxSeriesPeriods {
		return nil, errortypes.NewValidationError(
			"periods",
			errortypes.ErrInvalid,
			fmt.Sprintf("A chart covers between 1 and %d periods", dbhelper.MaxSeriesPeriods),
		)
	}

	loc := time.UTC
	if timezone != "" {
		parsed, err := time.LoadLocation(timezone)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"timezone",
				errortypes.ErrInvalid,
				fmt.Sprintf("%q is not a time zone", timezone),
			)
		}
		loc = parsed
	}

	local := now.In(loc)
	var start time.Time
	var step func(time.Time, int) time.Time
	switch interval {
	case SeriesWeek:
		midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		offset := (int(midnight.Weekday()) + 6) % 7
		current := midnight.AddDate(0, 0, -offset)
		start = current.AddDate(0, 0, -7*(periods-1))
		step = func(t time.Time, n int) time.Time { return t.AddDate(0, 0, 7*n) }
	case SeriesMonth:
		current := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
		start = current.AddDate(0, -(periods - 1), 0)
		step = func(t time.Time, n int) time.Time { return t.AddDate(0, n, 0) }
	default:
		return nil, errortypes.NewValidationError(
			"interval",
			errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a chart interval", interval),
		)
	}

	bounds := make([]int64, periods+1)
	for idx := range bounds {
		bounds[idx] = step(start, idx).Unix()
	}
	return bounds, nil
}
