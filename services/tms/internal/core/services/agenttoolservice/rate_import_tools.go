package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rateimport"
	"github.com/emoss08/trenova/internal/core/domain/ratesimulation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/rateimportservice"
	"github.com/emoss08/trenova/internal/core/services/ratesimulationservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramRateSimulationID = "rateSimulationId"
	maxSimulationName     = 150
	maxSimulationText     = 500
)

type rateImportKeeper interface {
	PlanCommit(
		ctx context.Context,
		req *rateimportservice.CommitRequest,
	) (*rateimportservice.CommitPlan, error)
	Commit(
		ctx context.Context,
		req *rateimportservice.CommitRequest,
	) (*rateimport.RateImportBatch, error)
	PlanDiscard(
		ctx context.Context,
		req *rateimportservice.CommitRequest,
	) (*rateimportservice.BatchChange, error)
	Discard(
		ctx context.Context,
		req *rateimportservice.CommitRequest,
	) (*rateimport.RateImportBatch, error)
}

type rateSimulationKeeper interface {
	PlanCreate(
		ctx context.Context,
		entity *ratesimulation.RateSimulation,
	) (*ratesimulationservice.CreatePlan, error)
	Create(
		ctx context.Context,
		entity *ratesimulation.RateSimulation,
		userID pulid.ID,
	) (*ratesimulation.RateSimulation, error)
}

var (
	_ rateImportKeeper     = (*rateimportservice.Service)(nil)
	_ rateSimulationKeeper = (*ratesimulationservice.Service)(nil)
)

func rateImportRequestFrom(
	params *serviceports.ToolExecuteParams,
) (*rateimportservice.CommitRequest, error) {
	id, err := requirePulid(params.Params, paramRateImportID)
	if err != nil {
		return nil, err
	}

	return &rateimportservice.CommitRequest{
		TenantInfo:        tenantFrom(*params),
		RateImportBatchID: id,
	}, nil
}

func rateImportRecord(batch *rateimport.RateImportBatch) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceRateAgreement,
		ID:       batch.ID,
		Label:    batch.FileName,
		Version:  pinnedVersion(batch.Version),
	}
}

func targetRateImport(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramRateImportID, permission.ResourceRateAgreement)
}

func newCommitRateImportTool(imports rateImportKeeper) serviceports.AgentTool {
	return newReceivableTool(rateMoneySpec(&receivableSpec{
		name:   "commit_rate_import",
		recipe: []string{"list_rate_imports", "commit_rate_import"},
		description: "Propose applying a reviewed rate sheet to its agreement from the " +
			"import's day. The lanes it changes and drops are closed out, its new rates take " +
			"effect, and the old rates stay in history. Read it with list_rate_imports first. It " +
			"changes what shipments are charged, so a person always decides.",
		operation: permission.OpUpdate,
		rationale: "Changes the rates an agreement prices shipments at; only a person applies " +
			"a rate sheet.",
		properties: map[string]any{
			paramRateImportID: agenttoolschema.KindID(rateImportSupplier, permission.KindRateImport),
		},
		required:    []string{paramRateImportID},
		target:      targetRateImport,
		searchTerms: []string{"rate sheet", "apply", "uploaded", "import"},
	}), receivablePlan[*rateimportservice.CommitRequest, *rateimportservice.CommitPlan]{
		request: rateImportRequestFrom,
		plan: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *serviceports.ToolExecuteParams,
		) (*rateimportservice.CommitPlan, error) {
			return imports.PlanCommit(ctx, req)
		},
		refused: func(*rateimportservice.CommitRequest) string {
			return "Would apply a rate sheet to its agreement."
		},
		render: func(
			_ *rateimportservice.CommitRequest,
			plan *rateimportservice.CommitPlan,
		) (*agent.ToolPreview, error) {
			lines := make([]agent.MoneyLine, 0, len(plan.Rules))
			for _, rule := range plan.Rules {
				lines = append(lines, agent.MoneyLine{
					Label: laneRateLabel(rule),
					After: rule.Rate,
				})
			}
			change := toolpreview.Money(
				rateImportRecord(plan.Batch),
				toolpreview.MoneyBlock("", lines...),
			)

			return toolpreview.Build(fmt.Sprintf(
				"Would apply the rate sheet %s from %s: close out %s and add %s. Shipments "+
					"from that day are charged at the new rates.",
				plan.Batch.FileName, dayLabel(plan.Batch.EffectiveFrom),
				countOf(len(plan.SupersededIDs), "lane"), countOf(len(plan.Rules), "lane"),
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := imports.Commit(ctx, req)

			return nil, err
		},
	})
}

type rateImportView struct {
	Status string `json:"status"`
}

func newDiscardRateImportTool(imports rateImportKeeper) serviceports.AgentTool {
	return newReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "discard_rate_import",
		description: "Discard a rate sheet that was read and should not be applied, such as " +
			"last month's file uploaded again. Its agreement is not changed.",
		resource:  permission.ResourceRateAgreement,
		operation: permission.OpUpdate,
		rationale: "Closes a staged rate sheet inside Trenova without changing any rate; the " +
			"sheet can be uploaded again.",
		reversible: true,
		properties: map[string]any{
			paramRateImportID: agenttoolschema.KindID(rateImportSupplier, permission.KindRateImport),
		},
		required: []string{paramRateImportID},
		target:   targetRateImport,
	}), receivablePlan[*rateimportservice.CommitRequest, *rateimportservice.BatchChange]{
		request: rateImportRequestFrom,
		plan: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *serviceports.ToolExecuteParams,
		) (*rateimportservice.BatchChange, error) {
			return imports.PlanDiscard(ctx, req)
		},
		refused: func(*rateimportservice.CommitRequest) string {
			return "Would discard a rate sheet."
		},
		render: func(
			_ *rateimportservice.CommitRequest,
			plan *rateimportservice.BatchChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				rateImportRecord(plan.Before),
				&rateImportView{Status: string(plan.Before.Status)},
				&rateImportView{Status: string(plan.After.Status)},
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would discard the rate sheet %s; its agreement is not changed.",
				plan.Before.FileName,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			_, err := imports.Discard(ctx, req)

			return nil, err
		},
	})
}

var simulationParties = agenttoolschema.Source(
	"rateSimulation.partyType",
	rateagreement.PartyTypeValues(),
)

func rateSimulationFrom(
	params *serviceports.ToolExecuteParams,
) (*ratesimulation.RateSimulation, error) {
	agreementID, err := requirePulid(params.Params, paramRateAgreementID)
	if err != nil {
		return nil, err
	}
	name, err := requireBoundedText(params.Params, paramSimulationName, maxSimulationName)
	if err != nil {
		return nil, err
	}
	description, err := boundedText(params.Params, paramDescription, maxSimulationText)
	if err != nil {
		return nil, err
	}
	from, err := requireDay(params.Params, paramSampleFrom)
	if err != nil {
		return nil, err
	}
	through, err := requireDay(params.Params, paramSampleTo)
	if err != nil {
		return nil, err
	}
	simulation := &ratesimulation.RateSimulation{
		OrganizationID:  params.OrganizationID,
		BusinessUnitID:  params.BusinessUnitID,
		RateAgreementID: agreementID,
		Name:            name,
		Description:     description,
		PartyType:       rateagreement.PartyTypeCustomer,
		SampleFrom:      from,
		SampleTo:        through + secondsPerDay,
	}
	if party, given, partyErr := optionalEnum(
		params.Params, paramPartyType, simulationParties.Values,
	); partyErr != nil {
		return nil, partyErr
	} else if given {
		simulation.PartyType = party
	}
	if _, given := params.Params[paramSampleLimit]; given {
		limit, limitErr := requireIntInRange(params.Params, paramSampleLimit, 1, maxSampleShipments)
		if limitErr != nil {
			return nil, limitErr
		}
		simulation.SampleLimit = limit
	}

	return simulation, nil
}

func newRunRateSimulationTool(simulations rateSimulationKeeper) serviceports.AgentTool {
	return newReportingReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "run_rate_simulation",
		description: "Replay past shipments against a rate agreement, usually a draft, to see " +
			"what it would have charged or paid next to what they actually were. It runs in " +
			"the background and changes no shipment and no rate.",
		resource:    permission.ResourceRateSimulation,
		operation:   permission.OpCreate,
		defaultTier: agent.TierActWithApproval,
		maxTier:     agent.TierAutoExecute,
		reversible:  true,
		artifact:    rateAgreementRecordEntity,
		rationale: "Saves a simulation inside Trenova and replays past shipments in the " +
			"background; no shipment, rate or invoice changes.",
		properties: map[string]any{
			paramRateAgreementID: rateAgreementIDProperty(),
			paramSimulationName: stringProperty("What the simulation is called, such as "+
				"Acme renewal against Q3.", maxSimulationName),
			paramDescription: stringProperty("What it is meant to answer.", maxSimulationText),
			paramPartyType: agenttoolschema.Enum("Which side to replay: what customers are "+
				"charged or what carriers are paid. Defaults to Customer.", simulationParties),
			paramSampleFrom: agenttoolschema.Date("The first ship day to replay."),
			paramSampleTo:   agenttoolschema.Date("The last ship day to replay."),
			paramSampleLimit: integerProperty("The most shipments to replay. Leave it out "+
				"for every shipment in the window.", 1, maxSampleShipments),
		},
		required: []string{
			paramRateAgreementID, paramSimulationName, paramSampleFrom, paramSampleTo,
		},
		target:      targetRateAgreement,
		searchTerms: []string{"simulate", "what if", "replay", "backtest", "renewal"},
	}), receivablePlan[*ratesimulation.RateSimulation, *ratesimulationservice.CreatePlan]{
		request: rateSimulationFrom,
		plan: func(
			ctx context.Context,
			simulation *ratesimulation.RateSimulation,
			_ *serviceports.ToolExecuteParams,
		) (*ratesimulationservice.CreatePlan, error) {
			return simulations.PlanCreate(ctx, simulation)
		},
		refused: func(*ratesimulation.RateSimulation) string {
			return "Would replay past shipments against a rate agreement."
		},
		render: func(
			_ *ratesimulation.RateSimulation,
			plan *ratesimulationservice.CreatePlan,
		) (*agent.ToolPreview, error) {
			return toolpreview.Build(fmt.Sprintf(
				"Would replay %s shipments shipped %s to %s against the %s rate agreement %s, "+
					"in the background; nothing is re-rated.",
				string(plan.Simulation.PartyType),
				dayLabel(plan.Simulation.SampleFrom),
				dayLabel(plan.Simulation.SampleTo-secondsPerDay),
				plan.Agreement.Status,
				plan.Agreement.Code,
			)), nil
		},
		run: func(
			ctx context.Context,
			simulation *ratesimulation.RateSimulation,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := simulations.Create(ctx, simulation, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "started",
				Kind:   "rate simulation",
				Name:   created.Name,
				IDs: map[string]string{
					paramRateSimulationID: created.ID.String(),
					paramRateAgreementID:  created.RateAgreementID.String(),
				},
				Record: recordOf(rateAgreementRecordEntity, created.RateAgreementID),
			}, nil
		},
	})
}
