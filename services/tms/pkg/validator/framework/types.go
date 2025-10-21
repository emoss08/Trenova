package framework

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
)

type ValidationRule interface {
	Stage() ValidationStage
	Priority() ValidationPriority
	Validate(ctx context.Context, multiErr *errortypes.MultiError) error
}

type ValidationRuleFunc struct {
	StageFunc    func() ValidationStage
	PriorityFunc func() ValidationPriority
	ValidateFunc func(ctx context.Context, multiErr *errortypes.MultiError) error
}

func (v ValidationRuleFunc) Stage() ValidationStage {
	return v.StageFunc()
}

func (v ValidationRuleFunc) Priority() ValidationPriority {
	return v.PriorityFunc()
}

func (v ValidationRuleFunc) Validate(ctx context.Context, multiErr *errortypes.MultiError) error {
	return v.ValidateFunc(ctx, multiErr)
}

func NewValidationRule(
	stage ValidationStage,
	priority ValidationPriority,
	validateFunc func(ctx context.Context, multiErr *errortypes.MultiError) error,
) ValidationRule {
	return ValidationRuleFunc{
		StageFunc: func() ValidationStage {
			return stage
		},
		PriorityFunc: func() ValidationPriority {
			return priority
		},
		ValidateFunc: validateFunc,
	}
}

type ValidationContext struct {
	Field      string
	Index      int
	Parent     *errortypes.MultiError
	IsIndexed  bool
	IndexedErr *errortypes.MultiError
}

type ValidationStage int

const (
	ValidationStageBasic ValidationStage = iota
	ValidationStageDataIntegrity
	ValidationStageBusinessRules
	ValidationStageCompliance
)

type ValidationPriority int

const (
	ValidationPriorityHigh ValidationPriority = iota
	ValidationPriorityMedium
	ValidationPriorityLow
)
