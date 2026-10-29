package masterdatakit

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const maxMasterRecordsPerStatusChange = 25

const masterClearsWhenEmpty = " Send an empty value to clear it."

type Record[T any] struct {
	Kind     string
	Resource permission.Resource
	Entity   string
	IDParam  string
	IDsParam string
	Lister   string
	Label    func(*T) string
	ID       func(*T) pulid.ID
	Version  func(*T) int64
	Detach   func(*T)
	StateIDs func(*T) []pulid.ID
	View     func(entity *T, states map[pulid.ID]string) any
	Options  []toolpreview.Option
	// linkEntity and linkID point the reported record at another page when
	// the record has none of its own, such as a carrier's capacity posting
	// shown on the carrier.
	LinkEntity string
	LinkID     func(*T) pulid.ID
}

func (r *Record[T]) artifact() string {
	if r.LinkEntity != "" {
		return r.LinkEntity
	}
	return r.Entity
}

func (r *Record[T]) artifactID(entity *T) pulid.ID {
	if r.LinkID != nil {
		return r.LinkID(entity)
	}
	return r.ID(entity)
}

type StateLookup interface {
	StateReader
	GetByIDs(ctx context.Context, ids []pulid.ID) ([]*usstate.UsState, error)
}

func (r *Record[T]) stateNames(
	ctx context.Context,
	states StateLookup,
	entities ...*T,
) map[pulid.ID]string {
	names := make(map[pulid.ID]string)
	if r.StateIDs == nil || states == nil {
		return names
	}
	ids := make([]pulid.ID, 0, len(entities)*2)
	for _, entity := range entities {
		for _, id := range r.StateIDs(entity) {
			if id.IsNotNil() {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return names
	}
	found, err := states.GetByIDs(ctx, ids)
	if err != nil {
		return names
	}
	for _, state := range found {
		names[state.ID] = state.Abbreviation
	}

	return names
}

func StateName(names map[pulid.ID]string, id pulid.ID) string {
	if id.IsNil() {
		return ""
	}
	if name, ok := names[id]; ok {
		return name
	}

	return id.String()
}

func StateNamePointer(names map[pulid.ID]string, id *pulid.ID) string {
	if id == nil {
		return ""
	}

	return StateName(names, *id)
}

type masterPlanned[T any] struct {
	Entity *T
	States map[pulid.ID]string
}

type masterChanged[T any] struct {
	change *kernelports.RecordChange[T]
	States map[pulid.ID]string
}

type masterStatusPlanned[T any] struct {
	changes []kernelports.RecordChange[T]
	States  map[pulid.ID]string
}

func (r *Record[T]) PreviewRecord(entity *T) toolpreview.Record {
	rec := toolpreview.Record{Resource: r.Resource, Label: r.Label(entity)}
	if id := r.ID(entity); id.IsNotNil() {
		rec.ID = id
		rec.Version = changepreview.PinnedVersion(r.Version(entity))
	}

	return rec
}

func (r *Record[T]) ViewOf(entity *T, states map[pulid.ID]string) *any {
	view := r.View(entity, states)

	return &view
}

func (r *Record[T]) result(action string, entity *T) *agentrecord.ToolExecutionResult {
	id := r.ID(entity)

	return &agentrecord.ToolExecutionResult{
		Action: action,
		Kind:   r.Kind,
		Name:   r.Label(entity),
		IDs:    map[string]string{r.IDParam: id.String()},
		Record: changepreview.RecordOf(r.artifact(), r.artifactID(entity)),
	}
}

func (r *Record[T]) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, r.IDParam, r.Resource)
}

type masterInput struct {
	values map[string]any
	States StateLookup
}

func (in *masterInput) given(key string) bool {
	_, ok := in.values[key]

	return ok
}

func (in *masterInput) text(key string) (string, error) {
	raw := in.values[key]
	if raw == nil {
		return "", nil
	}
	text, ok := raw.(string)
	if !ok {
		return "", errortypes.NewValidationError(key, errortypes.ErrInvalid, "Send text")
	}

	return strings.TrimSpace(text), nil
}

type Field[T any] struct {
	key      string
	property func(update bool) map[string]any
	apply    func(ctx context.Context, in *masterInput, entity *T) error
}

// clearable marks a field an update clears when it is sent empty, so the
// runtime reads its empty value as the request it is rather than as a
// parameter left unfilled (toolschema.KeepEmpty). A create has nothing to
// clear, and its empty values are read as not sent.
func clearable(update, clears bool, property map[string]any) map[string]any {
	if update && clears {
		return toolschema.KeepEmpty(property)
	}

	return property
}

func keepSuffix(update, clears bool) string {
	switch {
	case update && clears:
		return toolparams.KeepWhenLeftOut + masterClearsWhenEmpty
	case update:
		return toolparams.KeepWhenLeftOut
	default:
		return ""
	}
}

func Text[T any](
	key, description string,
	limit int,
	required bool,
	at func(*T) *string,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				!required,
				toolparams.StringProperty(description+keepSuffix(update, !required), limit),
			)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			text, err := in.text(key)
			if err != nil {
				return err
			}
			switch {
			case text == "" && required:
				return errortypes.NewValidationError(key, errortypes.ErrRequired,
					"This is required")
			case limit > 0 && utf8.RuneCountInString(text) > limit:
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"At most {0} characters are kept", limit)
			}
			*at(entity) = text

			return nil
		},
	}
}

func UpperText[T any](
	key, description string,
	limit int,
	at func(*T) *string,
) Field[T] {
	field := Text(key, description, limit, false, at)
	apply := field.apply
	field.apply = func(ctx context.Context, in *masterInput, entity *T) error {
		if err := apply(ctx, in, entity); err != nil {
			return err
		}
		*at(entity) = strings.ToUpper(*at(entity))

		return nil
	}

	return field
}

func Enum[T any, E ~string](
	key, description string,
	source agenttoolschema.EnumSource[E],
	at func(*T) *E,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return agenttoolschema.Enum(description+keepSuffix(update, false), source)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := toolparams.RequireEnum(in.values, key, source.Values)
			if err != nil {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Use one of {0}", strings.Join(source.Names(), ", "))
			}
			*at(entity) = value

			return nil
		},
	}
}

func EnumPointer[T any, E ~string](
	key, description string,
	source agenttoolschema.EnumSource[E],
	at func(*T) **E,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return agenttoolschema.Enum(description+keepSuffix(update, false), source)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := toolparams.RequireEnum(in.values, key, source.Values)
			if err != nil {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Use one of {0}", strings.Join(source.Names(), ", "))
			}
			*at(entity) = &value

			return nil
		},
	}
}

func Bool[T any](key, description string, at func(*T) *bool) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return toolparams.BooleanProperty(description + keepSuffix(update, false))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := toolparams.OptionalBoolPointer(in.values, key)
			if err != nil || value == nil {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Send true or false")
			}
			*at(entity) = *value

			return nil
		},
	}
}

func wholeNumber(in *masterInput, key string, minimum, maximum int) (*int, error) {
	raw, given := in.values[key]
	if !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent number is no number and no error
	}
	if text, isText := raw.(string); isText && strings.TrimSpace(text) == "" {
		return nil, nil //nolint:nilnil // an empty number clears the value
	}
	value, err := toolparams.RequireIntInRange(in.values, key, minimum, maximum)
	if err != nil {
		return nil, errortypes.NewValidationError(key, errortypes.ErrInvalid,
			"Send a whole number from {0} to {1}", minimum, maximum)
	}

	return &value, nil
}

func Int[T any](
	key, description string,
	minimum, maximum int,
	at func(*T) *int,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return toolparams.IntegerProperty(
				description+keepSuffix(update, false),
				minimum,
				maximum,
			)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := wholeNumber(in, key, minimum, maximum)
			if err != nil {
				return err
			}
			if value == nil {
				return errortypes.NewValidationError(key, errortypes.ErrRequired,
					"Send a whole number from {0} to {1}", minimum, maximum)
			}
			*at(entity) = *value

			return nil
		},
	}
}

func OptionalInt[T any](
	key, description string,
	minimum, maximum int,
	at func(*T) **int,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return toolparams.IntegerProperty(
				description+keepSuffix(update, false),
				minimum,
				maximum,
			)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := wholeNumber(in, key, minimum, maximum)
			if err != nil {
				return err
			}
			*at(entity) = value

			return nil
		},
	}
}

func Decimal[T any](key, description string, at func(*T) **float64) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				true,
				toolparams.StringProperty(description+" A decimal such as 42.5."+
					keepSuffix(update, true), 0),
			)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, present, err := toolparams.OptionalDecimal(in.values, key)
			if err != nil || (present && value.IsNegative()) {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Send a decimal that is zero or more, such as 42.5")
			}
			if !present {
				*at(entity) = nil
				return nil
			}
			number := value.Round(2).InexactFloat64()
			*at(entity) = &number

			return nil
		},
	}
}

func ID[T any](
	key string,
	resource permission.Resource,
	description string,
	at func(*T) *pulid.ID,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return agenttoolschema.RecordIDText(resource, description+keepSuffix(update, false))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			id, err := toolparams.RequirePulid(in.values, key)
			if err != nil {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Send an id from the read tool named here; never guess one")
			}
			*at(entity) = id

			return nil
		},
	}
}

func optionalMasterID(in *masterInput, key string) (pulid.ID, error) {
	text, err := in.text(key)
	if err != nil || text == "" {
		return pulid.Nil, err
	}
	id, err := pulid.Parse(text)
	if err != nil {
		return pulid.Nil, errortypes.NewValidationError(key, errortypes.ErrInvalid,
			"Send an id from the read tool named here; never guess one")
	}

	return id, nil
}

func OptionalID[T any](
	key string,
	resource permission.Resource,
	description string,
	at func(*T) *pulid.ID,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(update, true,
				agenttoolschema.RecordIDText(resource, description+keepSuffix(update, true)))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			id, err := optionalMasterID(in, key)
			if err != nil {
				return err
			}
			*at(entity) = id

			return nil
		},
	}
}

const stateCodeNote = "The two-letter US state abbreviation, such as TX."

func resolveState(ctx context.Context, in *masterInput, key string) (pulid.ID, error) {
	code, err := in.text(key)
	if err != nil || code == "" {
		return pulid.Nil, err
	}
	code = strings.ToUpper(code)
	if utf8.RuneCountInString(code) != 2 {
		return pulid.Nil, errortypes.NewValidationError(key, errortypes.ErrInvalid,
			"Use the two-letter state abbreviation")
	}
	state, err := in.States.GetByAbbreviation(ctx, code)
	if err != nil || state == nil {
		return pulid.Nil, errortypes.NewValidationError(key, errortypes.ErrInvalid,
			"{0} is not a US state abbreviation", code)
	}

	return state.ID, nil
}

func State[T any](
	key, description string,
	required bool,
	at func(*T) *pulid.ID,
) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				!required,
				toolparams.StringProperty(
					description+" "+stateCodeNote+keepSuffix(update, !required),
					2,
				),
			)
		},
		apply: func(ctx context.Context, in *masterInput, entity *T) error {
			id, err := resolveState(ctx, in, key)
			if err != nil {
				return err
			}
			if id.IsNil() && required {
				return errortypes.NewValidationError(key, errortypes.ErrRequired,
					"Name the state")
			}
			*at(entity) = id

			return nil
		},
	}
}

func StatePointer[T any](key, description string, at func(*T) **pulid.ID) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				true,
				toolparams.StringProperty(
					description+" "+stateCodeNote+keepSuffix(update, true),
					2,
				),
			)
		},
		apply: func(ctx context.Context, in *masterInput, entity *T) error {
			id, err := resolveState(ctx, in, key)
			if err != nil {
				return err
			}
			if id.IsNil() {
				*at(entity) = nil
				return nil
			}
			*at(entity) = &id

			return nil
		},
	}
}

func Day[T any](key, description string, at func(*T) **int64) Field[T] {
	return Field[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(update, true,
				agenttoolschema.Date(description+keepSuffix(update, true)))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			text, err := in.text(key)
			if err != nil {
				return err
			}
			if text == "" {
				*at(entity) = nil
				return nil
			}
			day, err := time.Parse(time.DateOnly, text)
			if err != nil {
				return toolparams.DateRefusal(key, text, toolschema.FormatDate)
			}
			seconds := day.Unix()
			*at(entity) = &seconds

			return nil
		},
	}
}

func masterProperties[T any](fields []Field[T], update bool) map[string]any {
	properties := make(map[string]any, len(fields)+1)
	for idx := range fields {
		properties[fields[idx].key] = fields[idx].property(update)
	}

	return properties
}

func applyMasterFields[T any](
	ctx context.Context,
	fields []Field[T],
	in *masterInput,
	entity *T,
) error {
	multiErr := errortypes.NewMultiError()
	for idx := range fields {
		field := &fields[idx]
		if !in.given(field.key) {
			continue
		}
		if err := field.apply(ctx, in, entity); err != nil {
			addFieldError(multiErr, field.key, err)
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func addFieldError(multiErr *errortypes.MultiError, key string, err error) {
	if fieldErr, ok := err.(*errortypes.Error); ok { //nolint:errorlint // the field helpers return the concrete error
		multiErr.AddError(fieldErr)
		return
	}
	multiErr.Add(key, errortypes.ErrInvalid, err.Error())
}

type Policy struct {
	DefaultTier agent.AutonomyTier
	MaxTier     agent.AutonomyTier
	TaintHold   string
	MoneyKeys   []string
	OutsideKeys []string
}

func (p *Policy) classify(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agenttoolports.CallPolicy {
	switch {
	case toolparams.SendsAny(params.Params, p.MoneyKeys...):
		return agenttoolports.CallPolicy{Egress: agent.EgressMoney, MaxTier: agent.TierPropose}
	case toolparams.SendsAny(params.Params, p.OutsideKeys...):
		return agenttoolports.CallPolicy{
			Egress:  agent.EgressExternalRecipient,
			MaxTier: agent.TierPropose,
		}
	default:
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}
}

func (p *Policy) apply(spec *toolspec.ReceivableSpec) *toolspec.ReceivableSpec {
	spec.Egress = agent.EgressInternal
	spec.DefaultTier = p.DefaultTier
	spec.MaxTier = p.MaxTier
	spec.TaintHold = p.TaintHold
	if len(p.MoneyKeys) > 0 {
		spec.AlsoEgress = append(spec.AlsoEgress, agent.EgressMoney)
	}
	if len(p.OutsideKeys) > 0 {
		spec.AlsoEgress = append(spec.AlsoEgress, agent.EgressExternalRecipient)
	}
	if len(spec.AlsoEgress) > 0 {
		spec.Classify = p.classify
	}

	return spec
}

func (p *Policy) guard(name string, params *agenttoolports.ToolExecuteParams) error {
	if toolparams.SendsAny(params.Params, p.MoneyKeys...) && !params.ApprovedFromProposal() {
		return fmt.Errorf("%s: %w", name, toolspec.ErrNeedsAPersonsApproval)
	}

	return nil
}

type CreateSpec[T any] struct {
	Record      *Record[T]
	Name        string
	Description string
	Rationale   string
	Fields      []Field[T]
	Required    []string
	SearchTerms []string
	Policy      Policy
	States      StateLookup
	Fresh       func(tenant pagination.TenantInfo) *T
	Plan        func(ctx context.Context, entity *T) (*T, error)
	Create      func(ctx context.Context, entity *T, actor *permissionports.RequestActor) (*T, error)
}

func (s *CreateSpec[T]) Build(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*T, error) {
	entity := s.Fresh(toolparams.TenantFrom(params))
	for _, key := range s.Required {
		if _, given := params.Params[key]; !given {
			return nil, errortypes.NewValidationError(key, errortypes.ErrRequired,
				"This is required")
		}
	}
	in := &masterInput{values: params.Params, States: s.States}
	if err := applyMasterFields(ctx, s.Fields, in, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func NewCreateTool[T any](spec *CreateSpec[T]) agenttoolports.AgentTool {
	record := spec.Record

	return toolspec.NewReportingReceivableTool(spec.Policy.apply(&toolspec.ReceivableSpec{
		Name:        spec.Name,
		Description: spec.Description,
		Resource:    record.Resource,
		Operation:   permission.OpCreate,
		Artifact:    record.artifact(),
		Reversible:  true,
		Idempotent:  true,
		Rationale:   spec.Rationale,
		Properties:  masterProperties(spec.Fields, false),
		Required:    spec.Required,
		SearchTerms: spec.SearchTerms,
	}), toolspec.ReceivablePlan[*agenttoolports.ToolExecuteParams, *masterPlanned[T]]{
		Request: func(params *agenttoolports.ToolExecuteParams) (*agenttoolports.ToolExecuteParams, error) {
			return params, nil
		},
		Plan: func(
			ctx context.Context,
			params *agenttoolports.ToolExecuteParams,
			_ *agenttoolports.ToolExecuteParams,
		) (*masterPlanned[T], error) {
			entity, err := spec.Build(ctx, params)
			if err != nil {
				return nil, err
			}
			planned, err := spec.Plan(ctx, entity)
			if err != nil {
				return nil, err
			}

			return &masterPlanned[T]{
				Entity: planned,
				States: record.stateNames(ctx, spec.States, planned),
			}, nil
		},
		Refused: func(*agenttoolports.ToolExecuteParams) string {
			return fmt.Sprintf("Would create a %s.", record.Kind)
		},
		Render: func(
			_ *agenttoolports.ToolExecuteParams,
			planned *masterPlanned[T],
		) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Create(record.PreviewRecord(planned.Entity),
				record.ViewOf(planned.Entity, planned.States), record.Options...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would create the %s %s.",
				record.Kind, record.Label(planned.Entity)), change), nil
		},
		Run: func(
			ctx context.Context,
			_ *agenttoolports.ToolExecuteParams,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if err := spec.Policy.guard(spec.Name, params); err != nil {
				return nil, err
			}
			entity, err := spec.Build(ctx, params)
			if err != nil {
				return nil, err
			}
			created, err := spec.Create(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}

			return record.result("created", created), nil
		},
	})
}

type UpdateSpec[T any] struct {
	Record      *Record[T]
	Name        string
	Description string
	Rationale   string
	Fields      []Field[T]
	SearchTerms []string
	Policy      Policy
	States      StateLookup
	Get         func(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID) (*T, error)
	Plan        func(ctx context.Context, entity *T) (*kernelports.RecordChange[T], error)
	Update      func(ctx context.Context, entity *T, actor *permissionports.RequestActor) (*T, error)
}

type masterEdit struct {
	ID     pulid.ID
	values map[string]any
}

func (s *UpdateSpec[T]) edit(params *agenttoolports.ToolExecuteParams) (*masterEdit, error) {
	id, err := toolparams.RequirePulid(params.Params, s.Record.IDParam)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(params.Params))
	for key, value := range params.Params {
		if key != s.Record.IDParam {
			values[key] = value
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("name at least one field of the %s to change", s.Record.Kind)
	}

	return &masterEdit{ID: id, values: values}, nil
}

func (s *UpdateSpec[T]) Build(
	ctx context.Context,
	edit *masterEdit,
	params *agenttoolports.ToolExecuteParams,
) (*T, error) {
	stored, err := s.Get(ctx, toolparams.TenantFrom(params), edit.ID)
	if err != nil {
		return nil, err
	}
	changed := *stored
	if s.Record.Detach != nil {
		s.Record.Detach(&changed)
	}
	in := &masterInput{values: edit.values, States: s.States}
	if err = applyMasterFields(ctx, s.Fields, in, &changed); err != nil {
		return nil, err
	}

	return &changed, nil
}

func NewUpdateTool[T any](spec *UpdateSpec[T]) agenttoolports.AgentTool {
	record := spec.Record
	properties := masterProperties(spec.Fields, true)
	properties[record.IDParam] = agenttoolschema.RecordIDText(record.Resource,
		fmt.Sprintf("The %s to change, from %s. Never guess one.", record.Kind, record.Lister))

	return toolspec.NewReportingReceivableTool(spec.Policy.apply(&toolspec.ReceivableSpec{
		Name:          spec.Name,
		Description:   spec.Description,
		Resource:      record.Resource,
		Operation:     permission.OpUpdate,
		Artifact:      record.artifact(),
		Reversible:    true,
		Rationale:     spec.Rationale,
		Properties:    properties,
		Required:      []string{record.IDParam},
		SearchTerms:   spec.SearchTerms,
		Prerequisites: []string{record.Lister},
		Target:        record.Target,
	}), toolspec.ReceivablePlan[*masterEdit, *masterChanged[T]]{
		Request: spec.edit,
		Plan: func(
			ctx context.Context,
			edit *masterEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*masterChanged[T], error) {
			entity, err := spec.Build(ctx, edit, params)
			if err != nil {
				return nil, err
			}
			change, err := spec.Plan(ctx, entity)
			if err != nil {
				return nil, err
			}

			return &masterChanged[T]{
				change: change,
				States: record.stateNames(ctx, spec.States, change.Before, change.After),
			}, nil
		},
		Refused: func(*masterEdit) string {
			return fmt.Sprintf("Would change a %s.", record.Kind)
		},
		Render: func(_ *masterEdit, plan *masterChanged[T]) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(record.PreviewRecord(plan.change.Before),
				record.ViewOf(plan.change.Before, plan.States),
				record.ViewOf(plan.change.After, plan.States), record.Options...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would change the %s %s.",
				record.Kind, record.Label(plan.change.Before)), change), nil
		},
		Run: func(
			ctx context.Context,
			edit *masterEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if err := spec.Policy.guard(spec.Name, params); err != nil {
				return nil, err
			}
			entity, err := spec.Build(ctx, edit, params)
			if err != nil {
				return nil, err
			}
			updated, err := spec.Update(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}

			return record.result(ActionUpdated, updated), nil
		},
	})
}

type StatusSpec[T any, S ~string] struct {
	Record      *Record[T]
	Name        string
	Description string
	Rationale   string
	SearchTerms []string
	Statuses    agenttoolschema.EnumSource[S]
	StatusNote  string
	Policy      Policy
	States      StateLookup
	Plan        func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
		status S,
	) ([]kernelports.RecordChange[T], error)
	Run func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
		status S,
	) ([]*T, error)
}

type masterStatusChange[S ~string] struct {
	ids    []pulid.ID
	status S
}

func (s *StatusSpec[T, S]) change(
	params *agenttoolports.ToolExecuteParams,
) (*masterStatusChange[S], error) {
	ids, err := toolparams.RequirePulidSlice(params.Params, s.Record.IDsParam,
		maxMasterRecordsPerStatusChange)
	if err != nil {
		return nil, err
	}
	status, err := toolparams.RequireEnum(
		params.Params,
		changepreview.FieldStatus,
		s.Statuses.Values,
	)
	if err != nil {
		return nil, err
	}

	return &masterStatusChange[S]{ids: ids, status: status}, nil
}

func NewStatusTool[T any, S ~string](spec *StatusSpec[T, S]) agenttoolports.AgentTool {
	record := spec.Record

	return toolspec.NewReportingReceivableTool(spec.Policy.apply(&toolspec.ReceivableSpec{
		Name:        spec.Name,
		Description: spec.Description,
		Resource:    record.Resource,
		Operation:   permission.OpUpdate,
		Artifact:    record.Entity,
		Reversible:  true,
		Rationale:   spec.Rationale,
		Properties: map[string]any{
			record.IDsParam: agenttoolschema.RecordIDs(record.Resource, fmt.Sprintf(
				"The %s records to change, by id from %s. One id is the normal case.",
				record.Kind, record.Lister), maxMasterRecordsPerStatusChange),
			changepreview.FieldStatus: agenttoolschema.Enum("The status to set. "+spec.StatusNote,
				spec.Statuses),
		},
		Required:      []string{record.IDsParam, changepreview.FieldStatus},
		SearchTerms:   spec.SearchTerms,
		Prerequisites: []string{record.Lister},
	}), toolspec.ReceivablePlan[*masterStatusChange[S], *masterStatusPlanned[T]]{
		Request: spec.change,
		Plan: func(
			ctx context.Context,
			change *masterStatusChange[S],
			params *agenttoolports.ToolExecuteParams,
		) (*masterStatusPlanned[T], error) {
			changes, err := spec.Plan(
				ctx,
				toolparams.TenantFrom(params),
				change.ids,
				change.status,
			)
			if err != nil {
				return nil, err
			}
			befores := kernelports.Befores(changes)

			return &masterStatusPlanned[T]{
				changes: changes,
				States:  record.stateNames(ctx, spec.States, befores...),
			}, nil
		},
		Refused: func(change *masterStatusChange[S]) string {
			return fmt.Sprintf("Would set %s to %s.",
				changepreview.CountOf(len(change.ids), record.Kind), change.status)
		},
		Render: func(
			change *masterStatusChange[S],
			plan *masterStatusPlanned[T],
		) (*agentpreview.ToolPreview, error) {
			changes := make([]*agentpreview.RecordChange, 0, len(plan.changes))
			for idx := range plan.changes {
				built, err := toolpreview.Changed(record.PreviewRecord(plan.changes[idx].Before),
					record.ViewOf(plan.changes[idx].Before, plan.States),
					record.ViewOf(plan.changes[idx].After, plan.States), record.Options...)
				if err != nil {
					return nil, err
				}
				changes = append(changes, built)
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would set %s to %s.",
				changepreview.CountOf(
					len(plan.changes),
					record.Kind,
				),
				change.status,
			), changes...), nil
		},
		Run: func(
			ctx context.Context,
			change *masterStatusChange[S],
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			updated, err := spec.Run(ctx, toolparams.TenantFrom(params), change.ids, change.status)
			if err != nil {
				return nil, err
			}
			ids := make(map[string]string, len(updated))
			for idx, entity := range updated {
				ids[fmt.Sprintf("%s[%d]", record.IDsParam, idx)] = record.ID(entity).String()
			}

			return &agentrecord.ToolExecutionResult{
				Action: ActionUpdated,
				Kind:   record.Kind,
				Name:   changepreview.CountOf(len(updated), record.Kind),
				IDs:    ids,
			}, nil
		},
	})
}

func DecimalText(value *float64) string {
	if value == nil {
		return ""
	}

	return decimal.NewFromFloat(*value).String()
}
