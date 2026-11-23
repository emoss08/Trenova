package workeremploymentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
)

type Preview struct {
	Kind        employment.EmploymentEventKind
	EffectiveAt int64
	FromValues  map[string]string
	ToValues    map[string]string
	Effects     []*employment.Effect
}

type Options struct {
	RecordableKinds     []employment.EmploymentEventKind
	ReasonRequiredKinds []employment.EmploymentEventKind
}

func (s *Service) Preview(ctx context.Context, req *RecordRequest) (*Preview, error) {
	plan, err := s.Plan(ctx, req)
	if err != nil {
		return nil, err
	}
	return &Preview{
		Kind:        plan.Event.Kind,
		EffectiveAt: plan.Event.EffectiveAt,
		FromValues:  plan.Event.FromValues,
		ToValues:    plan.Event.ToValues,
		Effects:     plan.Effects(),
	}, nil
}
