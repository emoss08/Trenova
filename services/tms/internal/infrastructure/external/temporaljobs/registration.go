/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/domains/compliance"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/domains/email"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/domains/patterns"
	"github.com/emoss08/trenova/internal/infrastructure/external/temporaljobs/domains/shipment"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
)

// RegistrationParams defines dependencies for registration
type RegistrationParams struct {
	fx.In

	Logger   *logger.Logger
	Registry *Registry
	Worker   *Worker
}

// RegisterAllWorkflowsAndActivities registers all domain workflows and activities
func RegisterAllWorkflowsAndActivities(p RegistrationParams) error {
	log := p.Logger.With().
		Str("component", "temporal-registration").
		Logger()

	// Register email domain workflows and activities
	for _, def := range email.RegisterWorkflows() {
		wfDef := &WorkflowDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			TaskQueue:   def.TaskQueue,
			Description: def.Description,
		}
		if err := p.Registry.RegisterWorkflow(wfDef); err != nil {
			log.Error().Err(err).Str("workflow", def.Name).Msg("failed to register workflow")
			return err
		}
	}

	for _, def := range email.RegisterActivities() {
		actDef := &ActivityDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			Description: def.Description,
		}
		if err := p.Registry.RegisterActivity(actDef); err != nil {
			log.Error().Err(err).Str("activity", def.Name).Msg("failed to register activity")
			return err
		}
	}

	// Register pattern domain workflows and activities
	for _, def := range patterns.RegisterWorkflows() {
		wfDef := &WorkflowDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			TaskQueue:   def.TaskQueue,
			Description: def.Description,
		}
		if err := p.Registry.RegisterWorkflow(wfDef); err != nil {
			log.Error().Err(err).Str("workflow", def.Name).Msg("failed to register workflow")
			return err
		}
	}

	for _, def := range patterns.RegisterActivities() {
		actDef := &ActivityDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			Description: def.Description,
		}
		if err := p.Registry.RegisterActivity(actDef); err != nil {
			log.Error().Err(err).Str("activity", def.Name).Msg("failed to register activity")
			return err
		}
	}

	// Register compliance domain workflows and activities
	for _, def := range compliance.RegisterWorkflows() {
		wfDef := &WorkflowDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			TaskQueue:   def.TaskQueue,
			Description: def.Description,
		}
		if err := p.Registry.RegisterWorkflow(wfDef); err != nil {
			log.Error().Err(err).Str("workflow", def.Name).Msg("failed to register workflow")
			return err
		}
	}

	for _, def := range compliance.RegisterActivities() {
		actDef := &ActivityDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			Description: def.Description,
		}
		if err := p.Registry.RegisterActivity(actDef); err != nil {
			log.Error().Err(err).Str("activity", def.Name).Msg("failed to register activity")
			return err
		}
	}

	// Register shipment domain workflows and activities
	for _, def := range shipment.RegisterWorkflows() {
		wfDef := &WorkflowDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			TaskQueue:   def.TaskQueue,
			Description: def.Description,
		}
		if err := p.Registry.RegisterWorkflow(wfDef); err != nil {
			log.Error().Err(err).Str("workflow", def.Name).Msg("failed to register workflow")
			return err
		}
	}

	for _, def := range shipment.RegisterActivities() {
		actDef := &ActivityDefinition{
			Name:        def.Name,
			Fn:          def.Fn,
			Description: def.Description,
		}
		if err := p.Registry.RegisterActivity(actDef); err != nil {
			log.Error().Err(err).Str("activity", def.Name).Msg("failed to register activity")
			return err
		}
	}

	// Apply all registered workflows and activities to the worker
	if err := p.Registry.ApplyToWorker(p.Worker); err != nil {
		log.Error().Err(err).Msg("failed to apply registrations to worker")
		return err
	}

	log.Info().
		Int("workflows", len(p.Registry.GetAllWorkflows())).
		Int("activities", len(p.Registry.GetAllActivities())).
		Msg("registered all workflows and activities")

	return nil
}

// RegisterWorkflowsAndActivities is a backwards-compatible function that registers workflows directly
// This is called by the worker during startup
func RegisterWorkflowsAndActivities(worker *Worker) {
	// Register workflows directly on the worker for backwards compatibility
	// These are the actual workflow functions that execute
	for _, def := range email.RegisterWorkflows() {
		worker.RegisterWorkflowWithOptions(def.Fn, workflow.RegisterOptions{
			Name: def.Name,
		})
	}

	for _, def := range patterns.RegisterWorkflows() {
		worker.RegisterWorkflowWithOptions(def.Fn, workflow.RegisterOptions{
			Name: def.Name,
		})
	}

	for _, def := range compliance.RegisterWorkflows() {
		worker.RegisterWorkflowWithOptions(def.Fn, workflow.RegisterOptions{
			Name: def.Name,
		})
	}

	for _, def := range shipment.RegisterWorkflows() {
		worker.RegisterWorkflowWithOptions(def.Fn, workflow.RegisterOptions{
			Name: def.Name,
		})
	}

	// Register activities
	for _, def := range email.RegisterActivities() {
		worker.RegisterActivity(def.Fn)
	}

	for _, def := range patterns.RegisterActivities() {
		worker.RegisterActivity(def.Fn)
	}

	for _, def := range compliance.RegisterActivities() {
		worker.RegisterActivity(def.Fn)
	}

	for _, def := range shipment.RegisterActivities() {
		worker.RegisterActivity(def.Fn)
	}
}