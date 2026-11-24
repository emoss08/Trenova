package boardreaderports

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	RecordReaders  = `group:"board.recordreaders"`
	MeasureReaders = `group:"board.measurereaders"`
	MetricReaders  = `group:"board.metricreaders"`
)

type RecordRequest struct {
	Tenant   pagination.TenantInfo
	Actor    *permissionports.RequestActor
	IDs      []pulid.ID
	Now      int64
	Location *time.Location
	Visible  protectedfield.GroupSet
	Place    bool
}

type RecordReader interface {
	Kind() recordkind.Kind
	Summaries(
		ctx context.Context,
		req *RecordRequest,
	) (map[pulid.ID]boardreading.Summary, error)
}

type MeasureReader interface {
	Kind() recordkind.Kind
	Measure() string
	Readings(
		ctx context.Context,
		req *RecordRequest,
	) (map[pulid.ID]boardreading.Reading, error)
}

type MetricRequest struct {
	Tenant   pagination.TenantInfo
	Actor    *permissionports.RequestActor
	For      pulid.ID
	Period   boardcatalog.Period
	Now      int64
	Location *time.Location
	Visible  protectedfield.GroupSet
}

type MetricReader interface {
	Metric() string
	Read(ctx context.Context, req *MetricRequest) (*boardcatalog.MetricRead, error)
}

type Readers struct {
	fx.In

	Catalog  *boardcatalog.Catalog
	Records  []RecordReader  `group:"board.recordreaders"`
	Measures []MeasureReader `group:"board.measurereaders"`
	Metrics  []MetricReader  `group:"board.metricreaders"`
}

type Index struct {
	Records  map[recordkind.Kind]RecordReader
	Measures map[MeasureKey]MeasureReader
	Metrics  map[string]MetricReader
}

type MeasureKey struct {
	Kind    recordkind.Kind
	Measure string
}

var ErrIncomplete = errors.New("the board catalog and its readers do not match")

func NewIndex(
	readers Readers, //nolint:gocritic // fx requires an fx.In parameter struct by value
) (*Index, error) {
	index := &Index{
		Records:  make(map[recordkind.Kind]RecordReader, len(readers.Records)),
		Measures: make(map[MeasureKey]MeasureReader, len(readers.Measures)),
		Metrics:  make(map[string]MetricReader, len(readers.Metrics)),
	}
	catalog := readers.Catalog
	problems := make([]string, 0)
	for _, reader := range readers.Records {
		if _, dup := index.Records[reader.Kind()]; dup {
			problems = append(problems, fmt.Sprintf("two record readers for %s", reader.Kind()))
		}
		if !catalog.Supports(reader.Kind()) {
			problems = append(problems, fmt.Sprintf("record reader for %s, a kind the "+
				"catalog does not list", reader.Kind()))
		}
		index.Records[reader.Kind()] = reader
	}
	for _, reader := range readers.Measures {
		key := MeasureKey{Kind: reader.Kind(), Measure: reader.Measure()}
		if _, dup := index.Measures[key]; dup {
			problems = append(problems, fmt.Sprintf("two readers for measure %s of %s",
				key.Measure, key.Kind))
		}
		if measure, ok := catalog.MeasureOf(key.Kind, key.Measure); !ok ||
			measure.Name != key.Measure {
			problems = append(problems, fmt.Sprintf("measure reader %s of %s, a measure the "+
				"catalog does not list", key.Measure, key.Kind))
		}
		index.Measures[key] = reader
	}
	for _, reader := range readers.Metrics {
		if _, dup := index.Metrics[reader.Metric()]; dup {
			problems = append(problems, fmt.Sprintf("two readers for metric %s", reader.Metric()))
		}
		if metric, ok := catalog.MetricNamed(reader.Metric()); !ok ||
			metric.Name != reader.Metric() {
			problems = append(problems, fmt.Sprintf("metric reader %s, a metric the catalog "+
				"does not list", reader.Metric()))
		}
		index.Metrics[reader.Metric()] = reader
	}
	for _, kind := range catalog.Kinds() {
		if _, ok := index.Records[kind.Kind]; !ok {
			problems = append(problems, fmt.Sprintf("no record reader for %s", kind.Kind))
		}
	}
	measures := catalog.Measures()
	for idx := range measures {
		key := MeasureKey{Kind: measures[idx].Kind, Measure: measures[idx].Name}
		if _, ok := index.Measures[key]; !ok {
			problems = append(problems, fmt.Sprintf("no reader for measure %s of %s",
				key.Measure, key.Kind))
		}
	}
	metrics := catalog.Metrics()
	for idx := range metrics {
		if _, ok := index.Metrics[metrics[idx].Name]; !ok {
			problems = append(problems, fmt.Sprintf("no reader for metric %s", metrics[idx].Name))
		}
	}
	if len(problems) > 0 {
		slices.Sort(problems)
		return nil, fmt.Errorf("%w: %v", ErrIncomplete, problems)
	}

	return index, nil
}
