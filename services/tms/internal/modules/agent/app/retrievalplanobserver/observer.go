package retrievalplanobserver

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/retrievalservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	SweepWorkflowID = "retrieval-index-sweep-plan"
	SweepDelay      = 45 * time.Second
)

type Params struct {
	fx.In

	Workflows kernelports.WorkflowStarter
	Logger    *zap.Logger
}

type Result struct {
	fx.Out

	Observer managedfeaturesports.PlanObserver `group:"managed_plan_observers"`
}

type Observer struct {
	workflows kernelports.WorkflowStarter
	l         *zap.Logger
}

func New(p Params) Result {
	return Result{Observer: &Observer{
		workflows: p.Workflows,
		l:         p.Logger.Named("service.retrieval-plan-observer"),
	}}
}

func (o *Observer) PlanChanged(
	ctx context.Context,
	replacement *managedfeatures.Replacement,
) error {
	if !TurnedOn(replacement, platformcatalog.FeatureAIDesk) {
		return nil
	}

	if _, err := o.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       SweepWorkflowID,
		TaskQueue:                temporaltype.TaskQueueSystem.String(),
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		StartDelay:               SweepDelay,
		StaticSummary:            "Start search by meaning for every organization after Trenova AI was added",
	}, retrievalservice.SweepWorkflowName); err != nil {
		return fmt.Errorf("start the retrieval sweep after Trenova AI was added: %w", err)
	}

	o.l.Info("Trenova AI was added; every organization's search index is checked shortly")

	return nil
}

func TurnedOn(replacement *managedfeatures.Replacement, feature platformcatalog.FeatureKey) bool {
	if !replacement.FeaturesChanged() || !slices.Contains(replacement.Current.Features, feature) {
		return false
	}

	return replacement.Previous == nil || !slices.Contains(replacement.Previous.Features, feature)
}
