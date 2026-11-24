package boardcardservice

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/planwording"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
)

type readScope struct {
	actor    *permissionports.RequestActor
	tenant   pagination.TenantInfo
	now      int64
	location *time.Location
	reach    *fieldReach
}

func (r *readScope) records(ids []pulid.ID) *boardreaderports.RecordRequest {
	return &boardreaderports.RecordRequest{
		Tenant:   r.tenant,
		Actor:    r.actor,
		IDs:      distinct(ids),
		Now:      r.now,
		Location: r.location,
		Visible:  r.reach.groups,
	}
}

func distinct(ids []pulid.ID) []pulid.ID {
	out := make([]pulid.ID, 0, min(len(ids), board.MaxBoardRecords))
	for _, id := range ids {
		if id.IsNotNil() && !slices.Contains(out, id) {
			out = append(out, id)
		}
		if len(out) == board.MaxBoardRecords {
			break
		}
	}

	return out
}

func pluralOf(kind recordkind.Kind) string {
	if spec, ok := boardregistry.Catalog().KindOf(kind); ok {
		return spec.Plural()
	}

	return inflect.PluralNoun(kind.Noun())
}

func (s *Service) allAllowed(
	ctx context.Context,
	actor *permissionports.RequestActor,
	resources []permission.Resource,
) (bool, error) {
	for _, resource := range resources {
		allowed, err := s.mayRead(ctx, actor, resource)
		if err != nil || !allowed {
			return false, err
		}
	}

	return true, nil
}

func (s *Service) featureOff(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (denial permissionports.PlanDenial, off bool, err error) {
	if feature == "" || s.plan == nil {
		return denial, false, nil
	}
	denial, off, err = s.plan.Feature(ctx, feature)
	if err != nil {
		return denial, false, fmt.Errorf("read whether the plan includes %s: %w", feature, err)
	}

	return denial, off, nil
}

func (s *Service) metricAllowed(
	ctx context.Context,
	actor *permissionports.RequestActor,
	reach *fieldReach,
	name string,
) (bool, *permissionports.PlanDenial, error) {
	metric, known := boardregistry.Catalog().MetricNamed(name)
	if !known {
		return false, nil, nil
	}
	denial, off, err := s.featureOff(ctx, metric.Feature)
	if err != nil {
		return false, nil, err
	}
	if off {
		return false, &denial, nil
	}
	allowed, err := s.allAllowed(ctx, actor, metric.Requires)
	if err != nil || !allowed || !reach.shows(ctx, metricFields(&metric)) {
		return false, nil, err
	}
	if allowed, err = wholeReadable(ctx, &metric); err != nil || !allowed {
		return false, nil, err
	}
	if metric.For != "" {
		allowed, err = s.mayRead(ctx, actor, metric.For.Resource())
	}

	return allowed, nil, err
}

func (s *Service) checkFigures(
	ctx context.Context,
	req *assistantports.BoardFillRequest,
	reach *fieldReach,
) error {
	check := func(metric, figure string) error {
		spec, known := boardregistry.Catalog().MetricNamed(metric)
		if !known {
			return nil
		}
		named, declared := spec.Figure(figure)
		if !declared {
			return nil
		}
		denial, off, err := s.featureOff(ctx, named.Feature)
		if err != nil {
			return err
		}
		if off {
			return mistake("the %s figure is not available: %s, so leave it out and say so "+
				"if the person asked for it", named.Name, planwording.Restriction(&denial))
		}
		if !reach.shows(ctx, named.Reads) {
			return mistake("the person may not read the %s figure, so leave it out", named.Name)
		}
		return nil
	}
	for idx := range req.Board.Blocks {
		block := &req.Board.Blocks[idx]
		if err := check(block.Metric, block.Figure); err != nil {
			return err
		}
		for figure := range block.Figures {
			if err := check(
				block.Figures[figure].Metric,
				block.Figures[figure].Figure,
			); err != nil {
				return err
			}
		}
		for item := range block.Checks {
			if err := check(block.Checks[item].Metric, block.Checks[item].Figure); err != nil {
				return err
			}
		}
	}
	for idx := range req.Board.Readings {
		if err := check(
			req.Board.Readings[idx].Metric,
			req.Board.Readings[idx].Figure,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) readMetrics(
	ctx context.Context,
	scope *readScope,
	asks []board.MetricAsk,
	reads *boardassemble.Reads,
) error {
	for _, ask := range asks {
		metric, known := boardregistry.Catalog().MetricNamed(ask.Metric)
		if !known {
			continue
		}
		allowed, _, err := s.metricAllowed(ctx, scope.actor, scope.reach, metric.Name)
		if err != nil {
			return err
		}
		reader := s.readers.Metrics[metric.Name]
		if !allowed || reader == nil {
			continue
		}
		seen, err := s.forReadable(ctx, scope, &metric, ask.For)
		if err != nil {
			return err
		}
		if !seen {
			continue
		}
		read, err := reader.Read(ctx, &boardreaderports.MetricRequest{
			Tenant:   scope.tenant,
			Actor:    scope.actor,
			For:      ask.For,
			Period:   boardcatalog.Period(ask.Period),
			Now:      scope.now,
			Location: scope.location,
			Visible:  scope.reach.groups,
		})
		if err != nil {
			return fmt.Errorf("read the %s metric: %w", metric.Name, err)
		}
		if read == nil {
			continue
		}
		if err = s.withholdFigures(ctx, scope.reach, &metric, read); err != nil {
			return err
		}
		reads.Metrics[ask] = read
	}

	return nil
}

func (s *Service) withholdFigures(
	ctx context.Context,
	reach *fieldReach,
	metric *boardcatalog.Metric,
	read *boardcatalog.MetricRead,
) error {
	for _, figure := range metric.Figures {
		_, off, err := s.featureOff(ctx, figure.Feature)
		if err != nil {
			return err
		}
		if off || !reach.shows(ctx, figure.Reads) {
			read.Figures[figure.Name] = nil
			delete(read.Changes, figure.Name)
		}
	}

	return nil
}

func (s *Service) readRecords(
	ctx context.Context,
	scope *readScope,
	refs map[recordkind.Kind][]pulid.ID,
	reads *boardassemble.Reads,
) error {
	for kind, ids := range refs {
		allowed, err := s.mayRead(ctx, scope.actor, kind.Resource())
		if err != nil {
			return err
		}
		reader := s.readers.Records[kind]
		reads.Readable[kind] = allowed && reader != nil
		if !reads.Readable[kind] {
			continue
		}
		summaries, err := reader.Summaries(ctx, scope.records(ids))
		if err != nil {
			return fmt.Errorf("read %s: %w", pluralOf(kind), err)
		}
		reads.Summaries[kind] = summaries
	}

	return nil
}

func (s *Service) readMeasures(
	ctx context.Context,
	scope *readScope,
	measures map[recordkind.Kind][]board.Measure,
	refs map[recordkind.Kind][]pulid.ID,
	reads *boardassemble.Reads,
) error {
	for kind, names := range measures {
		if !reads.Readable[kind] {
			continue
		}
		for _, name := range names {
			spec, known := name.Spec(kind)
			if !known {
				continue
			}
			key := board.MeasureKey{Kind: kind, Measure: name}
			allowed, err := s.allAllowed(ctx, scope.actor, spec.Requires)
			if err != nil {
				return err
			}
			if !allowed || !scope.reach.shows(ctx, measureFields(kind, &spec)) {
				reads.Withheld[key] = true
				continue
			}
			reader := s.readers.Measures[boardreaderports.MeasureKey{
				Kind:    kind,
				Measure: spec.Name,
			}]
			if reader == nil {
				reads.Withheld[key] = true
				continue
			}
			place, err := s.placeAllowed(ctx, scope, spec.Place)
			if err != nil {
				return err
			}
			req := scope.records(refs[kind])
			req.Place = place
			readings, err := reader.Readings(ctx, req)
			if err != nil {
				return fmt.Errorf("read the %s of %s: %w", spec.Name, pluralOf(kind), err)
			}
			if !place {
				withoutPlace(readings)
			}
			reads.Readings[key] = readings
		}
	}

	return nil
}

func (s *Service) placeAllowed(
	ctx context.Context,
	scope *readScope,
	part *boardcatalog.PlacePart,
) (bool, error) {
	if part == nil {
		return true, nil
	}
	allowed, err := s.allAllowed(ctx, scope.actor, part.Requires)
	if err != nil || !allowed {
		return false, err
	}

	return scope.reach.shows(ctx, part.Reads), nil
}

func withoutPlace(readings map[pulid.ID]boardreading.Reading) {
	for id := range readings {
		reading := readings[id]
		if reading.Fact == nil || reading.Fact.Place == "" {
			continue
		}
		fact := *reading.Fact
		fact.Place = ""
		reading.Fact = &fact
		readings[id] = reading
	}
}

func (s *Service) forReadable(
	ctx context.Context,
	scope *readScope,
	metric *boardcatalog.Metric,
	id pulid.ID,
) (bool, error) {
	if metric.For == "" || id.IsNil() {
		return true, nil
	}
	reader := s.readers.Records[metric.For]
	if reader == nil {
		return true, nil
	}
	summaries, err := reader.Summaries(ctx, scope.records([]pulid.ID{id}))
	if err != nil {
		return false, fmt.Errorf("read the %s a metric reads: %w", metric.For, err)
	}
	_, seen := summaries[id]

	return seen, nil
}
