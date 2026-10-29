package ratetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateimportservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratesimulationservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateimport"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratesimulation"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramRateSimulationID = "rateSimulationId"

const maxSimulationName = 150

const maxSimulationText = 500

type RateImportKeeper interface {
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

type RateSimulationKeeper interface {
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

func rateImportRequestFrom(
	params *agenttoolports.ToolExecuteParams,
) (*rateimportservice.CommitRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, paramRateImportID)
	if err != nil {
		return nil, err
	}

	return &rateimportservice.CommitRequest{
		TenantInfo:        toolparams.TenantFrom(*params),
		RateImportBatchID: id,
	}, nil
}

func rateImportRecord(batch *rateimport.RateImportBatch) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceRateAgreement,
		ID:       batch.ID,
		Label:    batch.FileName,
		Version:  changepreview.PinnedVersion(batch.Version),
	}
}

func targetRateImport(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramRateImportID, permission.ResourceRateAgreement)
}

func NewCommitRateImportTool(imports RateImportKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(rateMoneySpec(&toolspec.ReceivableSpec{
		Name:   "rate_commit_import",
		Recipe: []string{"rate_list_imports", "rate_commit_import"},
		Description: "Propose applying a reviewed rate sheet to its agreement from the " +
			"import's day. The lanes it changes and drops are closed out, its new rates take " +
			"effect, and the old rates stay in history. Read it with rate_list_imports first. It " +
			"changes what shipments are charged, so a person always decides.",
		Operation: permission.OpUpdate,
		Rationale: "Changes the rates an agreement prices shipments at; only a person applies " +
			"a rate sheet.",
		Properties: map[string]any{
			paramRateImportID: agenttoolschema.KindID(
				rateImportSupplier,
				permission.KindRateImport,
			),
		},
		Required:    []string{paramRateImportID},
		Target:      targetRateImport,
		SearchTerms: []string{"rate sheet", "apply", "uploaded", "import"},
	}), toolspec.ReceivablePlan[*rateimportservice.CommitRequest, *rateimportservice.CommitPlan]{
		Request: rateImportRequestFrom,
		Plan: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*rateimportservice.CommitPlan, error) {
			return imports.PlanCommit(ctx, req)
		},
		Refused: func(*rateimportservice.CommitRequest) string {
			return "Would apply a rate sheet to its agreement."
		},
		Render: func(
			_ *rateimportservice.CommitRequest,
			plan *rateimportservice.CommitPlan,
		) (*agentpreview.ToolPreview, error) {
			lines := make([]agentpreview.MoneyLine, 0, len(plan.Rules))
			for _, rule := range plan.Rules {
				lines = append(lines, agentpreview.MoneyLine{
					Label: laneRateLabel(rule),
					After: headlineRate(rule.Inputs),
				})
			}
			change := toolpreview.Money(
				rateImportRecord(plan.Batch),
				toolpreview.MoneyBlock("", lines...),
			)

			return toolpreview.Build(fmt.Sprintf(
				"Would apply the rate sheet %s from %s: close out %s and add %s. Shipments "+
					"from that day are charged at the new rates.",
				plan.Batch.FileName,
				changepreview.DayLabel(plan.Batch.EffectiveFrom),
				changepreview.CountOf(
					len(plan.SupersededIDs),
					"lane",
				),
				changepreview.CountOf(len(plan.Rules), "lane"),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			_, err := imports.Commit(ctx, req)

			return nil, err
		},
	})
}

type rateImportView struct {
	Status string `json:"status"`
}

func NewDiscardRateImportTool(imports RateImportKeeper) agenttoolports.AgentTool {
	return toolspec.NewReceivableTool(toolspec.FuelInternalSpec(&toolspec.ReceivableSpec{
		Name: "rate_discard_import",
		Description: "Discard a rate sheet that was read and should not be applied, such as " +
			"last month's file uploaded again. Its agreement is not changed.",
		Resource:  permission.ResourceRateAgreement,
		Operation: permission.OpUpdate,
		Rationale: "Closes a staged rate sheet inside Trenova without changing any rate; the " +
			"sheet can be uploaded again.",
		Reversible: true,
		Properties: map[string]any{
			paramRateImportID: agenttoolschema.KindID(
				rateImportSupplier,
				permission.KindRateImport,
			),
		},
		Required: []string{paramRateImportID},
		Target:   targetRateImport,
	}), toolspec.ReceivablePlan[*rateimportservice.CommitRequest, *rateimportservice.BatchChange]{
		Request: rateImportRequestFrom,
		Plan: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*rateimportservice.BatchChange, error) {
			return imports.PlanDiscard(ctx, req)
		},
		Refused: func(*rateimportservice.CommitRequest) string {
			return "Would discard a rate sheet."
		},
		Render: func(
			_ *rateimportservice.CommitRequest,
			plan *rateimportservice.BatchChange,
		) (*agentpreview.ToolPreview, error) {
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
		Run: func(
			ctx context.Context,
			req *rateimportservice.CommitRequest,
			_ *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
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
	params *agenttoolports.ToolExecuteParams,
) (*ratesimulation.RateSimulation, error) {
	agreementID, err := toolparams.RequirePulid(params.Params, paramRateAgreementID)
	if err != nil {
		return nil, err
	}
	name, err := toolparams.RequireBoundedText(
		params.Params,
		paramSimulationName,
		maxSimulationName,
	)
	if err != nil {
		return nil, err
	}
	description, err := toolparams.BoundedText(params.Params, paramDescription, maxSimulationText)
	if err != nil {
		return nil, err
	}
	from, err := toolparams.RequireDay(params.Params, paramSampleFrom)
	if err != nil {
		return nil, err
	}
	through, err := toolparams.RequireDay(params.Params, paramSampleTo)
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
		SampleTo:        through + toolparams.SecondsPerDay,
	}
	if party, given, partyErr := toolparams.OptionalEnum(
		params.Params, paramPartyType, simulationParties.Values,
	); partyErr != nil {
		return nil, partyErr
	} else if given {
		simulation.PartyType = party
	}
	if _, given := params.Params[paramSampleLimit]; given {
		limit, limitErr := toolparams.RequireIntInRange(
			params.Params,
			paramSampleLimit,
			1,
			maxSampleShipments,
		)
		if limitErr != nil {
			return nil, limitErr
		}
		simulation.SampleLimit = limit
	}

	return simulation, nil
}

func NewRunRateSimulationTool(simulations RateSimulationKeeper) agenttoolports.AgentTool {
	return toolspec.NewReportingReceivableTool(toolspec.FuelInternalSpec(&toolspec.ReceivableSpec{
		Name: "rate_run_simulation",
		Description: "Replay past shipments against a rate agreement, usually a draft, to see " +
			"what it would have charged or paid next to what they actually were. It runs in " +
			"the background and changes no shipment and no rate.",
		Resource:    permission.ResourceRateSimulation,
		Operation:   permission.OpCreate,
		DefaultTier: agent.TierActWithApproval,
		MaxTier:     agent.TierAutoExecute,
		Reversible:  true,
		Artifact:    rateAgreementRecordEntity,
		Rationale: "Saves a simulation inside Trenova and replays past shipments in the " +
			"background; no shipment, rate or invoice changes.",
		Properties: map[string]any{
			paramRateAgreementID: rateAgreementIDProperty(),
			paramSimulationName: toolparams.StringProperty(
				"What the simulation is called, such as "+
					"Acme renewal against Q3.",
				maxSimulationName,
			),
			paramDescription: toolparams.StringProperty(
				"What it is meant to answer.",
				maxSimulationText,
			),
			paramPartyType: agenttoolschema.Enum("Which side to replay: what customers are "+
				"charged or what carriers are paid. Defaults to Customer.", simulationParties),
			paramSampleFrom: agenttoolschema.Date("The first ship day to replay."),
			paramSampleTo:   agenttoolschema.Date("The last ship day to replay."),
			paramSampleLimit: toolschema.KeepEmpty(
				toolparams.IntegerProperty("The most shipments to "+
					"replay. Leave it out for every shipment in the window.", 1, maxSampleShipments),
			),
		},
		Required: []string{
			paramRateAgreementID, paramSimulationName, paramSampleFrom, paramSampleTo,
		},
		Target:      targetRateAgreement,
		SearchTerms: []string{"simulate", "what if", "replay", "backtest", "renewal"},
	}), toolspec.ReceivablePlan[*ratesimulation.RateSimulation, *ratesimulationservice.CreatePlan]{
		Request: rateSimulationFrom,
		Plan: func(
			ctx context.Context,
			simulation *ratesimulation.RateSimulation,
			_ *agenttoolports.ToolExecuteParams,
		) (*ratesimulationservice.CreatePlan, error) {
			return simulations.PlanCreate(ctx, simulation)
		},
		Refused: func(*ratesimulation.RateSimulation) string {
			return "Would replay past shipments against a rate agreement."
		},
		Render: func(
			_ *ratesimulation.RateSimulation,
			plan *ratesimulationservice.CreatePlan,
		) (*agentpreview.ToolPreview, error) {
			return toolpreview.Build(fmt.Sprintf(
				"Would replay %s shipments shipped %s to %s against the %s rate agreement %s, "+
					"in the background; nothing is re-rated.",
				string(plan.Simulation.PartyType),
				changepreview.DayLabel(plan.Simulation.SampleFrom),
				changepreview.DayLabel(plan.Simulation.SampleTo-toolparams.SecondsPerDay),
				plan.Agreement.Status,
				plan.Agreement.Code,
			)), nil
		},
		Run: func(
			ctx context.Context,
			simulation *ratesimulation.RateSimulation,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			created, err := simulations.Create(ctx, simulation, params.Actor.UserID)
			if err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "started",
				Kind:   "rate simulation",
				Name:   created.Name,
				IDs: map[string]string{
					paramRateSimulationID: created.ID.String(),
					paramRateAgreementID:  created.RateAgreementID.String(),
				},
				Record: changepreview.RecordOf(rateAgreementRecordEntity, created.RateAgreementID),
			}, nil
		},
	})
}
