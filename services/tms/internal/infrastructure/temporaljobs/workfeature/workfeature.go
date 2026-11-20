package workfeature

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

const HeaderKey = "trenova-work-feature"

type (
	startingKey struct{}
	runningKey  struct{}
)

func Starting(ctx context.Context, feature platformcatalog.FeatureKey) context.Context {
	return context.WithValue(ctx, startingKey{}, feature)
}

func From(ctx context.Context) (platformcatalog.FeatureKey, bool) {
	need, ok := entitlement.WorkFrom(ctx)
	if !ok {
		return "", false
	}
	return need.Feature()
}

func Running(ctx context.Context, feature platformcatalog.FeatureKey) context.Context {
	return entitlement.WithWork(ctx, entitlement.For(feature))
}

func FromWorkflow(ctx workflow.Context) (platformcatalog.FeatureKey, bool) {
	feature, ok := ctx.Value(runningKey{}).(platformcatalog.FeatureKey)
	return feature, ok && feature != ""
}

type Propagator struct {
	converter converter.DataConverter
}

var _ workflow.ContextPropagator = (*Propagator)(nil)

func NewPropagator() *Propagator {
	return &Propagator{converter: converter.GetDefaultDataConverter()}
}

func (p *Propagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
	feature, ok := ctx.Value(startingKey{}).(platformcatalog.FeatureKey)
	if !ok || feature == "" {
		return nil
	}
	return p.write(feature, writer)
}

func (p *Propagator) Extract(
	ctx context.Context,
	reader workflow.HeaderReader,
) (context.Context, error) {
	feature, err := p.read(reader)
	if err != nil || feature == "" {
		return ctx, err
	}
	return Running(ctx, feature), nil
}

func (p *Propagator) InjectFromWorkflow(ctx workflow.Context, writer workflow.HeaderWriter) error {
	feature, ok := FromWorkflow(ctx)
	if !ok {
		return nil
	}
	return p.write(feature, writer)
}

func (p *Propagator) ExtractToWorkflow(
	ctx workflow.Context,
	reader workflow.HeaderReader,
) (workflow.Context, error) {
	feature, err := p.read(reader)
	if err != nil || feature == "" {
		return ctx, err
	}
	return workflow.WithValue(ctx, runningKey{}, feature), nil
}

func (p *Propagator) write(
	feature platformcatalog.FeatureKey,
	writer workflow.HeaderWriter,
) error {
	payload, err := p.converter.ToPayload(string(feature))
	if err != nil {
		return fmt.Errorf("encode the work feature header: %w", err)
	}
	writer.Set(HeaderKey, payload)
	return nil
}

func (p *Propagator) read(reader workflow.HeaderReader) (platformcatalog.FeatureKey, error) {
	payload, ok := reader.Get(HeaderKey)
	if !ok {
		return "", nil
	}
	var feature string
	if err := p.converter.FromPayload(payload, &feature); err != nil {
		return "", fmt.Errorf("decode the work feature header: %w", err)
	}
	return platformcatalog.FeatureKey(feature), nil
}
