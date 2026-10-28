package masterdatatools

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
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

type masterRecord[T any] struct {
	kind     string
	resource permission.Resource
	entity   string
	idParam  string
	idsParam string
	lister   string
	label    func(*T) string
	id       func(*T) pulid.ID
	version  func(*T) int64
	detach   func(*T)
	stateIDs func(*T) []pulid.ID
	view     func(entity *T, states map[pulid.ID]string) any
	options  []toolpreview.Option
	// linkEntity and linkID point the reported record at another page when
	// the record has none of its own, such as a carrier's capacity posting
	// shown on the carrier.
	linkEntity string
	linkID     func(*T) pulid.ID
}

func (r *masterRecord[T]) artifact() string {
	if r.linkEntity != "" {
		return r.linkEntity
	}
	return r.entity
}

func (r *masterRecord[T]) artifactID(entity *T) pulid.ID {
	if r.linkID != nil {
		return r.linkID(entity)
	}
	return r.id(entity)
}

type StateLookup interface {
	stateReader
	GetByIDs(ctx context.Context, ids []pulid.ID) ([]*usstate.UsState, error)
}

func (r *masterRecord[T]) stateNames(
	ctx context.Context,
	states StateLookup,
	entities ...*T,
) map[pulid.ID]string {
	names := make(map[pulid.ID]string)
	if r.stateIDs == nil || states == nil {
		return names
	}
	ids := make([]pulid.ID, 0, len(entities)*2)
	for _, entity := range entities {
		for _, id := range r.stateIDs(entity) {
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

func stateName(names map[pulid.ID]string, id pulid.ID) string {
	if id.IsNil() {
		return ""
	}
	if name, ok := names[id]; ok {
		return name
	}

	return id.String()
}

func stateNamePointer(names map[pulid.ID]string, id *pulid.ID) string {
	if id == nil {
		return ""
	}

	return stateName(names, *id)
}

type masterPlanned[T any] struct {
	entity *T
	states map[pulid.ID]string
}

type masterChanged[T any] struct {
	change *kernelports.RecordChange[T]
	states map[pulid.ID]string
}

type masterStatusPlanned[T any] struct {
	changes []kernelports.RecordChange[T]
	states  map[pulid.ID]string
}

func (r *masterRecord[T]) record(entity *T) toolpreview.Record {
	rec := toolpreview.Record{Resource: r.resource, Label: r.label(entity)}
	if id := r.id(entity); id.IsNotNil() {
		rec.ID = id
		rec.Version = common.PinnedVersion(r.version(entity))
	}

	return rec
}

func (r *masterRecord[T]) viewOf(entity *T, states map[pulid.ID]string) *any {
	view := r.view(entity, states)

	return &view
}

func (r *masterRecord[T]) result(action string, entity *T) *agent.ToolExecutionResult {
	id := r.id(entity)

	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   r.kind,
		Name:   r.label(entity),
		IDs:    map[string]string{r.idParam: id.String()},
		Record: common.RecordOf(r.artifact(), r.artifactID(entity)),
	}
}

func (r *masterRecord[T]) target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, r.idParam, r.resource)
}

type masterInput struct {
	values map[string]any
	states StateLookup
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

type masterField[T any] struct {
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
		return common.KeepWhenLeftOut + masterClearsWhenEmpty
	case update:
		return common.KeepWhenLeftOut
	default:
		return ""
	}
}

func masterText[T any](
	key, description string,
	limit int,
	required bool,
	at func(*T) *string,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				!required,
				common.StringProperty(description+keepSuffix(update, !required), limit),
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

func masterUpperText[T any](
	key, description string,
	limit int,
	at func(*T) *string,
) masterField[T] {
	field := masterText(key, description, limit, false, at)
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

func masterEnum[T any, E ~string](
	key, description string,
	source agenttoolschema.EnumSource[E],
	at func(*T) *E,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return agenttoolschema.Enum(description+keepSuffix(update, false), source)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := common.RequireEnum(in.values, key, source.Values)
			if err != nil {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Use one of {0}", strings.Join(source.Names(), ", "))
			}
			*at(entity) = value

			return nil
		},
	}
}

func masterEnumPointer[T any, E ~string](
	key, description string,
	source agenttoolschema.EnumSource[E],
	at func(*T) **E,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return agenttoolschema.Enum(description+keepSuffix(update, false), source)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := common.RequireEnum(in.values, key, source.Values)
			if err != nil {
				return errortypes.NewValidationError(key, errortypes.ErrInvalid,
					"Use one of {0}", strings.Join(source.Names(), ", "))
			}
			*at(entity) = &value

			return nil
		},
	}
}

func masterBool[T any](key, description string, at func(*T) *bool) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return common.BooleanProperty(description + keepSuffix(update, false))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, err := common.OptionalBoolPointer(in.values, key)
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
	value, err := common.RequireIntInRange(in.values, key, minimum, maximum)
	if err != nil {
		return nil, errortypes.NewValidationError(key, errortypes.ErrInvalid,
			"Send a whole number from {0} to {1}", minimum, maximum)
	}

	return &value, nil
}

func masterInt[T any](
	key, description string,
	minimum, maximum int,
	at func(*T) *int,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return common.IntegerProperty(description+keepSuffix(update, false), minimum, maximum)
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

func masterOptionalInt[T any](
	key, description string,
	minimum, maximum int,
	at func(*T) **int,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return common.IntegerProperty(description+keepSuffix(update, false), minimum, maximum)
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

func masterDecimal[T any](key, description string, at func(*T) **float64) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				true,
				common.StringProperty(description+" A decimal such as 42.5."+
					keepSuffix(update, true), 0),
			)
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			value, present, err := common.OptionalDecimal(in.values, key)
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

func masterID[T any](
	key string,
	resource permission.Resource,
	description string,
	at func(*T) *pulid.ID,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return agenttoolschema.RecordIDText(resource, description+keepSuffix(update, false))
		},
		apply: func(_ context.Context, in *masterInput, entity *T) error {
			id, err := common.RequirePulid(in.values, key)
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

func masterOptionalID[T any](
	key string,
	resource permission.Resource,
	description string,
	at func(*T) *pulid.ID,
) masterField[T] {
	return masterField[T]{
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
	state, err := in.states.GetByAbbreviation(ctx, code)
	if err != nil || state == nil {
		return pulid.Nil, errortypes.NewValidationError(key, errortypes.ErrInvalid,
			"{0} is not a US state abbreviation", code)
	}

	return state.ID, nil
}

func masterState[T any](
	key, description string,
	required bool,
	at func(*T) *pulid.ID,
) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(
				update,
				!required,
				common.StringProperty(
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

func masterStatePointer[T any](key, description string, at func(*T) **pulid.ID) masterField[T] {
	return masterField[T]{
		key: key,
		property: func(update bool) map[string]any {
			return clearable(update, true,
				common.StringProperty(description+" "+stateCodeNote+keepSuffix(update, true), 2))
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

func masterDay[T any](key, description string, at func(*T) **int64) masterField[T] {
	return masterField[T]{
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
				return common.DateRefusal(key, text, toolschema.FormatDate)
			}
			seconds := day.Unix()
			*at(entity) = &seconds

			return nil
		},
	}
}

func masterProperties[T any](fields []masterField[T], update bool) map[string]any {
	properties := make(map[string]any, len(fields)+1)
	for idx := range fields {
		properties[fields[idx].key] = fields[idx].property(update)
	}

	return properties
}

func applyMasterFields[T any](
	ctx context.Context,
	fields []masterField[T],
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

type masterPolicy struct {
	defaultTier agent.AutonomyTier
	maxTier     agent.AutonomyTier
	taintHold   string
	moneyKeys   []string
	outsideKeys []string
}

func (p *masterPolicy) classify(
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // ToolPolicy.Classify passes params by value
) agenttoolports.CallPolicy {
	switch {
	case common.SendsAny(params.Params, p.moneyKeys...):
		return agenttoolports.CallPolicy{Egress: agent.EgressMoney, MaxTier: agent.TierPropose}
	case common.SendsAny(params.Params, p.outsideKeys...):
		return agenttoolports.CallPolicy{
			Egress:  agent.EgressExternalRecipient,
			MaxTier: agent.TierPropose,
		}
	default:
		return agenttoolports.CallPolicy{Egress: agent.EgressInternal}
	}
}

func (p *masterPolicy) apply(spec *common.ReceivableSpec) *common.ReceivableSpec {
	spec.Egress = agent.EgressInternal
	spec.DefaultTier = p.defaultTier
	spec.MaxTier = p.maxTier
	spec.TaintHold = p.taintHold
	if len(p.moneyKeys) > 0 {
		spec.AlsoEgress = append(spec.AlsoEgress, agent.EgressMoney)
	}
	if len(p.outsideKeys) > 0 {
		spec.AlsoEgress = append(spec.AlsoEgress, agent.EgressExternalRecipient)
	}
	if len(spec.AlsoEgress) > 0 {
		spec.Classify = p.classify
	}

	return spec
}

func (p *masterPolicy) guard(name string, params *agenttoolports.ToolExecuteParams) error {
	if common.SendsAny(params.Params, p.moneyKeys...) && !params.ApprovedFromProposal() {
		return fmt.Errorf("%s: %w", name, common.ErrNeedsAPersonsApproval)
	}

	return nil
}

type masterCreateSpec[T any] struct {
	record      *masterRecord[T]
	name        string
	description string
	rationale   string
	fields      []masterField[T]
	required    []string
	searchTerms []string
	policy      masterPolicy
	states      StateLookup
	fresh       func(tenant pagination.TenantInfo) *T
	plan        func(ctx context.Context, entity *T) (*T, error)
	create      func(ctx context.Context, entity *T, actor *permissionports.RequestActor) (*T, error)
}

func (s *masterCreateSpec[T]) Build(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*T, error) {
	entity := s.fresh(common.TenantFrom(*params))
	for _, key := range s.required {
		if _, given := params.Params[key]; !given {
			return nil, errortypes.NewValidationError(key, errortypes.ErrRequired,
				"This is required")
		}
	}
	in := &masterInput{values: params.Params, states: s.states}
	if err := applyMasterFields(ctx, s.fields, in, entity); err != nil {
		return nil, err
	}

	return entity, nil
}

func newMasterCreateTool[T any](spec *masterCreateSpec[T]) agenttoolports.AgentTool {
	record := spec.record

	return common.NewReportingReceivableTool(spec.policy.apply(&common.ReceivableSpec{
		Name:        spec.name,
		Description: spec.description,
		Resource:    record.resource,
		Operation:   permission.OpCreate,
		Artifact:    record.artifact(),
		Reversible:  true,
		Idempotent:  true,
		Rationale:   spec.rationale,
		Properties:  masterProperties(spec.fields, false),
		Required:    spec.required,
		SearchTerms: spec.searchTerms,
	}), common.ReceivablePlan[*agenttoolports.ToolExecuteParams, *masterPlanned[T]]{
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
			planned, err := spec.plan(ctx, entity)
			if err != nil {
				return nil, err
			}

			return &masterPlanned[T]{
				entity: planned,
				states: record.stateNames(ctx, spec.states, planned),
			}, nil
		},
		Refused: func(*agenttoolports.ToolExecuteParams) string {
			return fmt.Sprintf("Would create a %s.", record.kind)
		},
		Render: func(
			_ *agenttoolports.ToolExecuteParams,
			planned *masterPlanned[T],
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(record.record(planned.entity),
				record.viewOf(planned.entity, planned.states), record.options...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would create the %s %s.",
				record.kind, record.label(planned.entity)), change), nil
		},
		Run: func(
			ctx context.Context,
			_ *agenttoolports.ToolExecuteParams,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if err := spec.policy.guard(spec.name, params); err != nil {
				return nil, err
			}
			entity, err := spec.Build(ctx, params)
			if err != nil {
				return nil, err
			}
			created, err := spec.create(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}

			return record.result("created", created), nil
		},
	})
}

type masterUpdateSpec[T any] struct {
	record      *masterRecord[T]
	name        string
	description string
	rationale   string
	fields      []masterField[T]
	searchTerms []string
	policy      masterPolicy
	states      StateLookup
	get         func(ctx context.Context, tenant pagination.TenantInfo, id pulid.ID) (*T, error)
	plan        func(ctx context.Context, entity *T) (*kernelports.RecordChange[T], error)
	update      func(ctx context.Context, entity *T, actor *permissionports.RequestActor) (*T, error)
}

type masterEdit struct {
	id     pulid.ID
	values map[string]any
}

func (s *masterUpdateSpec[T]) edit(params *agenttoolports.ToolExecuteParams) (*masterEdit, error) {
	id, err := common.RequirePulid(params.Params, s.record.idParam)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(params.Params))
	for key, value := range params.Params {
		if key != s.record.idParam {
			values[key] = value
		}
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("name at least one field of the %s to change", s.record.kind)
	}

	return &masterEdit{id: id, values: values}, nil
}

func (s *masterUpdateSpec[T]) Build(
	ctx context.Context,
	edit *masterEdit,
	params *agenttoolports.ToolExecuteParams,
) (*T, error) {
	stored, err := s.get(ctx, common.TenantFrom(*params), edit.id)
	if err != nil {
		return nil, err
	}
	changed := *stored
	if s.record.detach != nil {
		s.record.detach(&changed)
	}
	in := &masterInput{values: edit.values, states: s.states}
	if err = applyMasterFields(ctx, s.fields, in, &changed); err != nil {
		return nil, err
	}

	return &changed, nil
}

func newMasterUpdateTool[T any](spec *masterUpdateSpec[T]) agenttoolports.AgentTool {
	record := spec.record
	properties := masterProperties(spec.fields, true)
	properties[record.idParam] = agenttoolschema.RecordIDText(record.resource,
		fmt.Sprintf("The %s to change, from %s. Never guess one.", record.kind, record.lister))

	return common.NewReportingReceivableTool(spec.policy.apply(&common.ReceivableSpec{
		Name:          spec.name,
		Description:   spec.description,
		Resource:      record.resource,
		Operation:     permission.OpUpdate,
		Artifact:      record.artifact(),
		Reversible:    true,
		Rationale:     spec.rationale,
		Properties:    properties,
		Required:      []string{record.idParam},
		SearchTerms:   spec.searchTerms,
		Prerequisites: []string{record.lister},
		Target:        record.target,
	}), common.ReceivablePlan[*masterEdit, *masterChanged[T]]{
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
			change, err := spec.plan(ctx, entity)
			if err != nil {
				return nil, err
			}

			return &masterChanged[T]{
				change: change,
				states: record.stateNames(ctx, spec.states, change.Before, change.After),
			}, nil
		},
		Refused: func(*masterEdit) string {
			return fmt.Sprintf("Would change a %s.", record.kind)
		},
		Render: func(_ *masterEdit, plan *masterChanged[T]) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(record.record(plan.change.Before),
				record.viewOf(plan.change.Before, plan.states),
				record.viewOf(plan.change.After, plan.states), record.options...)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would change the %s %s.",
				record.kind, record.label(plan.change.Before)), change), nil
		},
		Run: func(
			ctx context.Context,
			edit *masterEdit,
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if err := spec.policy.guard(spec.name, params); err != nil {
				return nil, err
			}
			entity, err := spec.Build(ctx, edit, params)
			if err != nil {
				return nil, err
			}
			updated, err := spec.update(ctx, entity, params.Actor)
			if err != nil {
				return nil, err
			}

			return record.result(ActionUpdated, updated), nil
		},
	})
}

type masterStatusSpec[T any, S ~string] struct {
	record      *masterRecord[T]
	name        string
	description string
	rationale   string
	searchTerms []string
	statuses    agenttoolschema.EnumSource[S]
	statusNote  string
	policy      masterPolicy
	states      StateLookup
	plan        func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ids []pulid.ID,
		status S,
	) ([]kernelports.RecordChange[T], error)
	run func(
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

func (s *masterStatusSpec[T, S]) change(
	params *agenttoolports.ToolExecuteParams,
) (*masterStatusChange[S], error) {
	ids, err := common.RequirePulidSlice(params.Params, s.record.idsParam,
		maxMasterRecordsPerStatusChange)
	if err != nil {
		return nil, err
	}
	status, err := common.RequireEnum(params.Params, common.FieldStatus, s.statuses.Values)
	if err != nil {
		return nil, err
	}

	return &masterStatusChange[S]{ids: ids, status: status}, nil
}

func newMasterStatusTool[T any, S ~string](spec *masterStatusSpec[T, S]) agenttoolports.AgentTool {
	record := spec.record

	return common.NewReportingReceivableTool(spec.policy.apply(&common.ReceivableSpec{
		Name:        spec.name,
		Description: spec.description,
		Resource:    record.resource,
		Operation:   permission.OpUpdate,
		Artifact:    record.entity,
		Reversible:  true,
		Rationale:   spec.rationale,
		Properties: map[string]any{
			record.idsParam: agenttoolschema.RecordIDs(record.resource, fmt.Sprintf(
				"The %s records to change, by id from %s. One id is the normal case.",
				record.kind, record.lister), maxMasterRecordsPerStatusChange),
			common.FieldStatus: agenttoolschema.Enum("The status to set. "+spec.statusNote,
				spec.statuses),
		},
		Required:      []string{record.idsParam, common.FieldStatus},
		SearchTerms:   spec.searchTerms,
		Prerequisites: []string{record.lister},
	}), common.ReceivablePlan[*masterStatusChange[S], *masterStatusPlanned[T]]{
		Request: spec.change,
		Plan: func(
			ctx context.Context,
			change *masterStatusChange[S],
			params *agenttoolports.ToolExecuteParams,
		) (*masterStatusPlanned[T], error) {
			changes, err := spec.plan(ctx, common.TenantFrom(*params), change.ids, change.status)
			if err != nil {
				return nil, err
			}
			befores := kernelports.Befores(changes)

			return &masterStatusPlanned[T]{
				changes: changes,
				states:  record.stateNames(ctx, spec.states, befores...),
			}, nil
		},
		Refused: func(change *masterStatusChange[S]) string {
			return fmt.Sprintf("Would set %s to %s.",
				common.CountOf(len(change.ids), record.kind), change.status)
		},
		Render: func(
			change *masterStatusChange[S],
			plan *masterStatusPlanned[T],
		) (*agent.ToolPreview, error) {
			changes := make([]*agent.RecordChange, 0, len(plan.changes))
			for idx := range plan.changes {
				built, err := toolpreview.Changed(record.record(plan.changes[idx].Before),
					record.viewOf(plan.changes[idx].Before, plan.states),
					record.viewOf(plan.changes[idx].After, plan.states), record.options...)
				if err != nil {
					return nil, err
				}
				changes = append(changes, built)
			}

			return toolpreview.Build(fmt.Sprintf("Would set %s to %s.",
				common.CountOf(len(plan.changes), record.kind), change.status), changes...), nil
		},
		Run: func(
			ctx context.Context,
			change *masterStatusChange[S],
			params *agenttoolports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			updated, err := spec.run(ctx, common.TenantFrom(*params), change.ids, change.status)
			if err != nil {
				return nil, err
			}
			ids := make(map[string]string, len(updated))
			for idx, entity := range updated {
				ids[fmt.Sprintf("%s[%d]", record.idsParam, idx)] = record.id(entity).String()
			}

			return &agent.ToolExecutionResult{
				Action: ActionUpdated,
				Kind:   record.kind,
				Name:   common.CountOf(len(updated), record.kind),
				IDs:    ids,
			}, nil
		},
	})
}

func decimalText(value *float64) string {
	if value == nil {
		return ""
	}

	return decimal.NewFromFloat(*value).String()
}
