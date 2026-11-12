package recordcase

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MinOptions = 2
	MaxOptions = 3

	maxOptionKey         = 40
	maxOptionLabel       = 120
	maxOptionOutcome     = 40
	maxOptionOutcomeNote = 60
	maxOptionCost        = 80
	maxOptionDescription = 400
	maxStepLabel         = 120
	maxStepWhen          = 40
	maxWindowText        = 80
)

type ProjectedStep struct {
	Label     string   `json:"label"`
	When      string   `json:"when,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

type Projection struct {
	Step   *ProjectedStep `json:"step,omitempty"`
	Window string         `json:"window,omitempty"`
}

func (p *Projection) normalize() *Projection {
	if p == nil {
		return nil
	}
	out := &Projection{Window: stringutils.Ellipsize(strings.TrimSpace(p.Window), maxWindowText)}
	if p.Step != nil && strings.TrimSpace(p.Step.Label) != "" {
		out.Step = &ProjectedStep{
			Label:     stringutils.Ellipsize(strings.TrimSpace(p.Step.Label), maxStepLabel),
			When:      stringutils.Ellipsize(strings.TrimSpace(p.Step.When), maxStepWhen),
			Latitude:  p.Step.Latitude,
			Longitude: p.Step.Longitude,
		}
	}
	if out.Step == nil && out.Window == "" {
		return nil
	}

	return out
}

type Option struct {
	keel.Table `keel:"record_case_options"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rco_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RecordKind recordkind.Kind `json:"recordKind" keel:"record_kind"`
	RecordID   pulid.ID        `json:"recordId"   keel:"record_id"`

	Key         string      `json:"key"         keel:"key"`
	Position    int         `json:"position"    keel:"position"`
	Kind        OptionKind  `json:"kind"        keel:"kind"`
	Label       string      `json:"label"       keel:"label"`
	Outcome     string      `json:"outcome"     keel:"outcome,nullzero"`
	OutcomeNote string      `json:"outcomeNote" keel:"outcome_note,nullzero"`
	Tone        Tone        `json:"tone"        keel:"tone"`
	Cost        string      `json:"cost"        keel:"cost,nullzero"`
	Description string      `json:"description" keel:"description,nullzero"`
	Recommended bool        `json:"recommended" keel:"recommended"`
	Chosen      bool        `json:"chosen"      keel:"chosen"`
	Projection  *Projection `json:"projection"  keel:"projection,jsonb"`

	AgentDefinitionID pulid.ID `json:"agentDefinitionId" keel:"agent_definition_id,nullzero"`
	RunID             pulid.ID `json:"runId"             keel:"run_id,nullzero"`
	DraftRunID        pulid.ID `json:"draftRunId"        keel:"draft_run_id,nullzero"`
	DraftRequestedAt  *int64   `json:"draftRequestedAt"  keel:"draft_requested_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (o *Option) GetID() pulid.ID { return o.ID }

func (o *Option) GetTableName() string { return "record_case_options" }

func (o *Option) Normalize() {
	o.Key = strings.ToLower(strings.TrimSpace(o.Key))
	o.Label = stringutils.Ellipsize(strings.TrimSpace(o.Label), maxOptionLabel)
	o.Outcome = stringutils.Ellipsize(strings.TrimSpace(o.Outcome), maxOptionOutcome)
	o.OutcomeNote = stringutils.Ellipsize(strings.TrimSpace(o.OutcomeNote), maxOptionOutcomeNote)
	o.Cost = stringutils.Ellipsize(strings.TrimSpace(o.Cost), maxOptionCost)
	o.Description = stringutils.Ellipsize(strings.TrimSpace(o.Description), maxOptionDescription)
	o.Projection = o.Projection.normalize()
	if o.Kind == "" {
		o.Kind = OptionKindPlan
	}
	if o.Tone == "" {
		o.Tone = ToneNeutral
	}
}

func (o *Option) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(o,
		validation.Field(&o.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&o.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&o.RecordID, validation.Required.Error("Record is required")),
		validation.Field(&o.Key,
			validation.Required.Error("Key is required"),
			validation.Length(1, maxOptionKey).Error("Key is too long"),
		),
		validation.Field(&o.Label, validation.Required.Error("Label is required")),
		validation.Field(&o.Kind, domainvalidation.ValidEnum[OptionKind]("Kind is invalid")),
		validation.Field(&o.Tone, domainvalidation.ValidEnum[Tone]("Tone is invalid")),
	))
	if !o.RecordKind.Known() {
		multiErr.Add("recordKind", errortypes.ErrInvalid, "Record kind is invalid")
	}
	if o.Kind == OptionKindLeave && o.Recommended {
		multiErr.Add(
			"recommended",
			errortypes.ErrInvalid,
			"Leaving it as it is cannot be the recommended option",
		)
	}
}

func (o *Option) Drafting() bool {
	return o.Kind == OptionKindPlan && o.DraftRequestedAt != nil
}

func (o *Option) DraftNote() string {
	var builder strings.Builder
	builder.WriteString("A person opened this record's case and chose the option \"")
	builder.WriteString(o.Label)
	builder.WriteString("\"")
	if o.Description != "" {
		builder.WriteString(": ")
		builder.WriteString(o.Description)
	}
	builder.WriteString(". Read the record as it is now and draft the plan that carries out ")
	builder.WriteString("exactly this option, as one plan they can approve. Do not offer the ")
	builder.WriteString("options again and do not draft any other option.")

	return builder.String()
}

func Chosen(options []*Option) *Option {
	var recommended *Option
	for _, option := range options {
		if option.Chosen {
			return option
		}
		if option.Recommended && recommended == nil {
			recommended = option
		}
	}

	return recommended
}

func ValidateSet(options []*Option, multiErr *errortypes.MultiError) {
	if len(options) < MinOptions || len(options) > MaxOptions {
		multiErr.Add("options", errortypes.ErrInvalid, "Offer two or three options")

		return
	}
	keys := make(map[string]struct{}, len(options))
	recommended, leave := 0, 0
	for _, option := range options {
		if _, taken := keys[option.Key]; taken {
			multiErr.Add("options", errortypes.ErrDuplicate, "Each option needs its own key")
		}
		keys[option.Key] = struct{}{}
		if option.Recommended {
			recommended++
		}
		if option.Kind == OptionKindLeave {
			leave++
		}
	}
	if recommended != 1 {
		multiErr.Add("options", errortypes.ErrInvalid, "Recommend exactly one option")
	}
	if leave > 1 {
		multiErr.Add(
			"options",
			errortypes.ErrInvalid,
			"Only one option can leave things as they are",
		)
	}
}
