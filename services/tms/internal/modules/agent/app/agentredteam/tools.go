package agentredteam

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentmemoryservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundqueue"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/reflectutils"
	"go.uber.org/zap"
)

var passthroughTools = []string{agentdefinition.CoreToolRemember}

type toolDeps struct {
	desk        *inboundDesk
	inbox       *inboundqueue.Service
	memories    agentmemoryports.AgentMemoryService
	shipments   *shipmentDesk
	comments    *commentDesk
	watchtower  *watchtowerDesk
	permissions *permissionDesk
	scripts     *scriptDesk
}

func newToolDeps(rec *recorder, responses map[string]map[string]any) toolDeps {
	desk := &inboundDesk{rec: rec}

	return toolDeps{
		desk: desk,
		inbox: inboundqueue.New(inboundqueue.Params{
			Logger:      zap.NewNop(),
			MessageRepo: &inboundMessageRepository{desk: desk},
		}),
		memories: agentmemoryservice.New(agentmemoryservice.Params{
			Logger: zap.NewNop(),
			Repo:   &memoryRepository{rec: rec},
			Runs:   &runRepository{rec: rec},
		}),
		shipments:   &shipmentDesk{rec: rec, response: responses[getShipmentTool]},
		comments:    &commentDesk{rec: rec, response: responses[getShipmentTool]},
		watchtower:  &watchtowerDesk{rec: rec, response: responses[listWatchtowerTool]},
		permissions: &permissionDesk{rec: rec},
		scripts:     &scriptDesk{rec: rec},
	}
}

func (d toolDeps) fakes() []reflect.Value {
	return []reflect.Value{
		reflect.ValueOf(d.desk),
		reflect.ValueOf(d.inbox),
		reflect.ValueOf(d.memories),
		reflect.ValueOf(d.shipments),
		reflect.ValueOf(d.comments),
		reflect.ValueOf(d.watchtower),
		reflect.ValueOf(d.permissions),
		reflect.ValueOf(d.scripts),
	}
}

type builtTools struct {
	queries []agenttoolports.AgentQueryTool
	actions []agenttoolports.AgentTool
}

func buildTools(rec *recorder, responses map[string]map[string]any) (builtTools, error) {
	fakes := newToolDeps(rec, responses).fakes()
	providers := append(
		agentquerytoolservice.ToolProviders(),
		agenttoolservice.ToolProviders()...,
	)

	built := builtTools{
		queries: make([]agenttoolports.AgentQueryTool, 0, len(providers)),
		actions: make([]agenttoolports.AgentTool, 0, len(providers)),
	}
	for _, provider := range providers {
		tool, err := reflectutils.Construct(provider, suppliedFor(provider, fakes))
		if err != nil {
			return builtTools{}, err
		}

		switch typed := tool.(type) {
		case agenttoolports.AgentQueryTool:
			built.queries = append(built.queries, &recordingQuery{
				inner:    typed,
				rec:      rec,
				response: responses[typed.Name()],
			})
		case agenttoolports.AgentTool:
			built.actions = append(built.actions, recordedAction(typed, rec))
		default:
			return builtTools{}, fmt.Errorf("%T builds neither a query nor an action tool",
				provider)
		}
	}

	return built, nil
}

func suppliedFor(provider any, fakes []reflect.Value) map[reflect.Type]reflect.Value {
	supplied := map[reflect.Type]reflect.Value{
		reflect.TypeFor[*filtercatalog.Catalog](): reflect.ValueOf(
			agentquerytoolservice.FilterCatalog(),
		),
	}

	fnType := reflect.TypeOf(provider)
	if fnType.Kind() != reflect.Func {
		return supplied
	}
	for idx := range fnType.NumIn() {
		in := fnType.In(idx)
		if _, ok := supplied[in]; ok {
			continue
		}
		if fake, ok := fakeFor(in, fakes); ok {
			supplied[in] = fake
			continue
		}
		if in.Kind() == reflect.Struct {
			if params, ok := paramsFor(in, fakes); ok {
				supplied[in] = params
			}
		}
	}

	return supplied
}

func fakeFor(in reflect.Type, fakes []reflect.Value) (reflect.Value, bool) {
	for _, fake := range fakes {
		if fake.Type() == in {
			return fake, true
		}
		if in.Kind() == reflect.Interface && in.NumMethod() > 0 && fake.Type().Implements(in) {
			return fake, true
		}
	}

	return reflect.Value{}, false
}

func paramsFor(in reflect.Type, fakes []reflect.Value) (reflect.Value, bool) {
	params := reflect.New(in).Elem()
	filled := false
	for idx := range in.NumField() {
		field := in.Field(idx)
		if field.Anonymous || !field.IsExported() {
			continue
		}
		if fake, ok := fakeFor(field.Type, fakes); ok {
			params.Field(idx).Set(fake)
			filled = true
		}
	}

	return params, filled
}

type recordingQuery struct {
	inner    agenttoolports.AgentQueryTool
	rec      *recorder
	response map[string]any
}

func (q *recordingQuery) Name() string                      { return q.inner.Name() }
func (q *recordingQuery) Description() string               { return q.inner.Description() }
func (q *recordingQuery) ParamSchema() map[string]any       { return q.inner.ParamSchema() }
func (q *recordingQuery) Policy() agenttoolports.ToolPolicy { return q.inner.Policy() }

func (q *recordingQuery) SearchTerms() []string { return searchTermsOf(q.inner) }

func (q *recordingQuery) Prerequisites() []string { return prerequisitesOf(q.inner) }

func (q *recordingQuery) Companions() []string { return companionsOf(q.inner) }

func (q *recordingQuery) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	q.rec.read(ReadTool, q.Name(), pagination.TenantInfo{
		OrgID: params.OrganizationID,
		BuID:  params.BusinessUnitID,
	})

	if slices.Contains(liveQueries, q.Name()) {
		return q.inner.Query(ctx, params)
	}
	if q.response != nil {
		return q.response, nil
	}

	return map[string]any{"records": []any{}, "note": "Nothing else is on record."}, nil
}

type recordingAction struct {
	inner agenttoolports.AgentTool
	rec   *recorder
}

func (a *recordingAction) Name() string                      { return a.inner.Name() }
func (a *recordingAction) Description() string               { return a.inner.Description() }
func (a *recordingAction) ParamSchema() map[string]any       { return a.inner.ParamSchema() }
func (a *recordingAction) Policy() agenttoolports.ToolPolicy { return a.inner.Policy() }

func (a *recordingAction) SearchTerms() []string { return searchTermsOf(a.inner) }

func (a *recordingAction) Prerequisites() []string { return prerequisitesOf(a.inner) }

func (a *recordingAction) Companions() []string { return companionsOf(a.inner) }

//nolint:gocritic // agenttoolports tools take their execute params by value
func (a *recordingAction) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	a.rec.executed(&ExecutionRecord{
		Tool:   a.Name(),
		Params: params,
		Egress: a.inner.Policy().Classified(params).Egress,
	})

	if slices.Contains(passthroughTools, a.Name()) {
		return a.inner.Execute(ctx, params)
	}

	return nil
}

func searchTermsOf(tool any) []string {
	if searchable, ok := tool.(agenttoolports.SearchableTool); ok {
		return searchable.SearchTerms()
	}

	return nil
}

func prerequisitesOf(tool any) []string {
	if dependent, ok := tool.(agenttoolports.PrerequisiteTool); ok {
		return dependent.Prerequisites()
	}

	return nil
}

func companionsOf(tool any) []string {
	if companion, ok := tool.(agenttoolports.CompanionTool); ok {
		return companion.Companions()
	}

	return nil
}

// recordedAction records a write's runs. A write a script may prepare keeps
// the record it names, its preview and, for a tool over a set of records, the
// split into one call per record, so the broker sees the tool as it is.
func recordedAction(tool agenttoolports.AgentTool, rec *recorder) agenttoolports.AgentTool {
	recording := &recordingAction{inner: tool, rec: rec}
	if !agenttoolports.PreparableFromScript(tool) {
		return recording
	}
	preparable := &recordingPreparable{recordingAction: recording}
	if splitter, splits := tool.(agenttoolports.ToolRecordSplitter); splits {
		return &recordingSplitter{recordingPreparable: preparable, splitter: splitter}
	}

	return preparable
}

type recordingPreparable struct {
	*recordingAction
}

func (a *recordingPreparable) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	targeted, _ := a.inner.(agenttoolports.TargetedTool)

	return targeted.Target(params)
}

func (a *recordingPreparable) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	previewer, _ := a.inner.(agenttoolports.ToolPreviewer)

	return previewer.Preview(ctx, params)
}

type recordingSplitter struct {
	*recordingPreparable
	splitter agenttoolports.ToolRecordSplitter
}

func (a *recordingSplitter) SplitRecords(params map[string]any) ([]map[string]any, error) {
	return a.splitter.SplitRecords(params)
}
