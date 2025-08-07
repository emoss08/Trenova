/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package framework

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

// ValidationEngineFactory is a factory for creating validation engines
type ValidationEngineFactory interface {
	// CreateEngine creates a new validation engine
	CreateEngine() *ValidationEngine
}

// defaultValidationEngineFactory is the default implementation of ValidationEngineFactory
type defaultValidationEngineFactory struct{}

// CreateEngine creates a new validation engine
func (f *defaultValidationEngineFactory) CreateEngine() *ValidationEngine {
	return NewValidationEngine()
}

// ProvideValidationEngineFactory provides a ValidationEngineFactory
func ProvideValidationEngineFactory() ValidationEngineFactory {
	return &defaultValidationEngineFactory{}
}

// EngineLifecycle represents the lifecycle of the ValidationEngine
type EngineLifecycle struct{}

// OnStart logs that the validation engine is starting
func (l *EngineLifecycle) OnStart(_ context.Context) error {
	log.Info().Msg("🚀 Starting validation framework engine")
	return nil
}

// OnStop logs that the validation engine is stopping
func (l *EngineLifecycle) OnStop(_ context.Context) error {
	log.Info().Msg("🔴 Stopping validation framework engine")
	return nil
}

// ProvideLifecycle provides a lifecycle for the validation engine
func ProvideLifecycle() *EngineLifecycle {
	return &EngineLifecycle{}
}

// Module provides the validation framework module
var Module = fx.Module("validation-framework",
	fx.Provide(
		ProvideValidationEngineFactory,
		ProvideLifecycle,
	),
	fx.Invoke(
		func(lc fx.Lifecycle, lifecycle *EngineLifecycle) {
			lc.Append(fx.Hook{
				OnStart: lifecycle.OnStart,
				OnStop:  lifecycle.OnStop,
			})
		},
	),
)
