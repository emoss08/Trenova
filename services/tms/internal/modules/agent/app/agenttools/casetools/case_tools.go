package casetools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	paramRecordID    = "recordId"
	paramOptions     = "options"
	paramKey         = "key"
	paramKind        = "kind"
	paramLabel       = "label"
	paramOutcome     = "outcome"
	paramOutcomeNote = "outcomeNote"
	paramTone        = "tone"
	paramCost        = "cost"
	paramRecommended = "recommended"
	paramWindow      = "window"
	paramStep        = "step"
	paramWhen        = "when"
	paramLatitude    = "latitude"
	paramLongitude   = "longitude"

	maxKeyLength    = 40
	maxLabelLength  = 120
	maxShortLength  = 60
	maxCostLength   = 80
	maxDetailLength = 400
)

var (
	optionKinds = agenttoolschema.Source("case_option_kinds", []recordcase.OptionKind{
		recordcase.OptionKindPlan,
		recordcase.OptionKindLeave,
	})
	optionTones = agenttoolschema.Source("case_option_tones", []recordcase.Tone{
		recordcase.ToneGood,
		recordcase.ToneWarning,
		recordcase.ToneBad,
		recordcase.ToneNeutral,
	})

	errNoCases = errors.New("cases are not available here")
)

type offerOptionsTool struct {
	cases recordcaseports.OptionOfferer
}

func NewOfferOptionsTool(cases recordcaseports.OptionOfferer) agenttoolports.AgentTool {
	return &offerOptionsTool{cases: cases}
}

func (t *offerOptionsTool) Name() string { return "agent_offer_options" }

func (t *offerOptionsTool) SearchTerms() []string {
	return []string{"alternatives", "what could happen", "recovery options", "ways to fix it"}
}

func (t *offerOptionsTool) Description() string {
	return "Offer the two or three ways a record's open problem can go, on its case in the " +
		"Desk, when you propose a plan for one of them. Use it when there is a real choice " +
		"(recover a late load by a relay, by moving the appointment, or not at all; settle a " +
		"dispute by a credit, a rebill, or upholding the invoice), in the same turn as the " +
		"plan for the option you recommend, and never for a problem with one sensible fix. " +
		"Say what each leads to and what it costs from what you read, never from a guess, " +
		"and include leaving it as it is when that is a choice a person could make. The " +
		"person picks one; if it is not the one you drafted, you are started again to draft " +
		"that plan. Offering again replaces what you offered before."
}

func (t *offerOptionsTool) ParamSchema() map[string]any {
	option := map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramKey: toolparams.StringProperty(
				"A short name for the option that stays the same if you offer again: relay, "+
					"reschedule, leave.",
				maxKeyLength,
			),
			paramKind: agenttoolschema.Enum(
				"Plan for an option you would draft a plan for; Leave for leaving the record "+
					"as it is.",
				optionKinds,
			),
			paramLabel: toolparams.StringProperty(
				"The option in a few words, as a choice: \"Relay at the Little Rock yard\".",
				maxLabelLength,
			),
			paramOutcome: toolparams.StringProperty(
				"What it leads to, as the figure a person compares: an arrival time, an "+
					"amount. \"1:40 PM\".",
				maxShortLength,
			),
			paramOutcomeNote: toolparams.StringProperty(
				"What that figure means: \"on time\", \"4h late\", \"rescheduled\".",
				maxShortLength,
			),
			paramTone: agenttoolschema.Enum(
				"How the outcome reads: Good when it fixes the problem, Warning when it "+
					"trades it for another, Bad when the problem stands.",
				optionTones,
			),
			paramCost: toolparams.StringProperty(
				"What it costs, in a few words: \"+$85 deadhead\", \"No fee if they confirm\".",
				maxCostLength,
			),
			toolparams.ParamDescription: toolparams.StringProperty(
				"One or two sentences on what would happen and who does what.",
				maxDetailLength,
			),
			paramRecommended: toolparams.BooleanProperty(
				"True for the one option you recommend and drafted the plan for.",
			),
			paramWindow: toolparams.StringProperty(
				"For an option that moves an appointment or a due date: the new one as a "+
					"person reads it, \"Tue 7:00–8:00 AM\".",
				maxShortLength,
			),
			paramStep: map[string]any{
				toolschema.KeyType: toolschema.TypeObject,
				toolschema.KeyDescription: "For an option that adds a step to the record's " +
					"timeline, such as a relay stop: what it is, when, and where.",
				toolschema.KeyProperties: map[string]any{
					paramLabel: toolparams.StringProperty(
						"The step: \"Relay · Little Rock yard\".",
						maxLabelLength,
					),
					paramWhen: toolparams.StringProperty(
						"When it would happen, as a person reads it: \"10:30 AM\".",
						maxShortLength,
					),
					paramLatitude: map[string]any{
						toolschema.KeyType:        "number",
						toolschema.KeyDescription: "Where it is, from the location you read.",
					},
					paramLongitude: map[string]any{
						toolschema.KeyType:        "number",
						toolschema.KeyDescription: "Where it is, from the location you read.",
					},
				},
				toolschema.KeyRequired:             []string{paramLabel},
				toolschema.KeyAdditionalProperties: false,
			},
		},
		toolschema.KeyRequired: []string{
			paramKey, paramKind, paramLabel, paramTone, toolparams.ParamDescription,
			paramRecommended,
		},
		toolschema.KeyAdditionalProperties: false,
	}

	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramRecordID: agenttoolschema.IDText(
				"The record the options are about: the run subject, or the id shipment_get, " +
					"invoice_get or another tool returned.",
			),
			paramOptions: map[string]any{
				toolschema.KeyType:     toolschema.TypeArray,
				toolschema.KeyMinItems: recordcase.MinOptions,
				toolschema.KeyMaxItems: recordcase.MaxOptions,
				toolschema.KeyItems:    option,
				toolschema.KeyDescription: "Two or three options, the recommended one among " +
					"them, in the order a person should read them.",
			},
		},
		toolschema.KeyRequired:             []string{paramRecordID, paramOptions},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *offerOptionsTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceAgentRun,
		Operation:     permission.OpRead,
		Scope:         agenttooltrust.ToolScopeRun,
		DefaultTier:   agent.TierAutoExecute,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		OwnWork:       true,
		Artifact:      "agent_run",
		Rationale: "Writes the choices an agent lays out on a record's case; it changes no " +
			"record and reaches nobody, and every option still needs its plan approved.",
	}
}

func (t *offerOptionsTool) Validate(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	_, err := t.request(params)

	return err
}

func (t *offerOptionsTool) Preview(
	_ context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	request, err := t.request(params)
	if err != nil {
		return toolpreview.Build(
			"Would offer options on the record's case.",
		), nil //nolint:nilerr // a preview says what would happen; Validate refuses the call
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would offer %d options on the record's case. No record changes.",
		len(request.Options),
	)), nil
}

func (t *offerOptionsTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *offerOptionsTool) ExecuteWithResult(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentrecord.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	if t.cases == nil {
		return nil, errNoCases
	}
	request, err := t.request(params)
	if err != nil {
		return nil, err
	}
	if _, err = t.cases.OfferOptions(ctx, request); err != nil {
		return nil, err
	}

	return &agentrecord.ToolExecutionResult{
		Action: "offered",
		Kind:   "options",
		Name:   fmt.Sprintf("%d options", len(request.Options)),
		IDs:    map[string]string{paramRecordID: request.ID.String()},
	}, nil
}

func (t *offerOptionsTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*recordcaseports.OfferOptionsRequest, error) {
	recordID, err := toolparams.RequirePulid(params.Params, paramRecordID)
	if err != nil {
		return nil, err
	}
	kind, ok := recordkind.OfIDPrefix(recordID.Prefix())
	if !ok || kind.Resource() == "" {
		return nil, errortypes.NewValidationError(
			paramRecordID,
			errortypes.ErrInvalid,
			"recordId must be the id of a record a case can be about",
		)
	}
	raw, _ := params.Params[paramOptions].([]any)
	if len(raw) < recordcase.MinOptions || len(raw) > recordcase.MaxOptions {
		return nil, errortypes.NewValidationError(
			paramOptions, errortypes.ErrInvalid, "Offer two or three options",
		)
	}

	options := make([]*recordcase.Option, 0, len(raw))
	for _, entry := range raw {
		fields, isObject := entry.(map[string]any)
		if !isObject {
			return nil, errortypes.NewValidationError(
				paramOptions, errortypes.ErrInvalid, "Each option is an object",
			)
		}
		option, oErr := optionOf(fields)
		if oErr != nil {
			return nil, oErr
		}
		options = append(options, option)
	}

	return &recordcaseports.OfferOptionsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
		Kind:              kind,
		ID:                recordID,
		AgentDefinitionID: params.DefinitionID,
		RunID:             params.RunID,
		Options:           options,
	}, nil
}

func optionOf(fields map[string]any) (*recordcase.Option, error) {
	key, err := toolparams.RequireString(fields, paramKey)
	if err != nil {
		return nil, err
	}
	label, err := toolparams.RequireString(fields, paramLabel)
	if err != nil {
		return nil, err
	}
	kind := recordcase.OptionKind(toolparams.OptionalString(fields, paramKind))
	if !kind.IsValid() {
		return nil, errortypes.NewValidationError(
			paramKind, errortypes.ErrInvalid, "kind must be Plan or Leave",
		)
	}
	tone := recordcase.Tone(toolparams.OptionalString(fields, paramTone))
	if !tone.IsValid() {
		return nil, errortypes.NewValidationError(
			paramTone, errortypes.ErrInvalid, "tone must be Good, Warning, Bad or Neutral",
		)
	}

	option := &recordcase.Option{
		Key:         key,
		Kind:        kind,
		Label:       label,
		Outcome:     toolparams.OptionalString(fields, paramOutcome),
		OutcomeNote: toolparams.OptionalString(fields, paramOutcomeNote),
		Tone:        tone,
		Cost:        toolparams.OptionalString(fields, paramCost),
		Description: toolparams.OptionalString(fields, toolparams.ParamDescription),
		Recommended: toolparams.OptionalBool(fields, paramRecommended),
	}
	projection := &recordcase.Projection{Window: toolparams.OptionalString(fields, paramWindow)}
	if step, isObject := fields[paramStep].(map[string]any); isObject {
		projection.Step = &recordcase.ProjectedStep{
			Label:     toolparams.OptionalString(step, paramLabel),
			When:      toolparams.OptionalString(step, paramWhen),
			Latitude:  coordinate(step, paramLatitude, 90),
			Longitude: coordinate(step, paramLongitude, 180),
		}
	}
	option.Projection = projection

	return option, nil
}

func coordinate(fields map[string]any, key string, limit float64) *float64 {
	value, ok := fields[key].(float64)
	if !ok || value < -limit || value > limit {
		return nil
	}

	return &value
}
