package agentredteam

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentmemoryservice"
	"github.com/emoss08/trenova/internal/core/services/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/core/services/agenttoolservice"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/reflectutils"
	"go.uber.org/zap"
)

var passthroughTools = []string{agentdefinition.CoreToolRemember}

type toolDeps struct {
	desk        *inboundDesk
	inbox       *inboundmessageservice.Service
	memories    serviceports.AgentMemoryService
	shipments   *shipmentDesk
	comments    *commentDesk
	permissions *permissionDesk
}

func newToolDeps(rec *recorder, responses map[string]map[string]any) toolDeps {
	desk := &inboundDesk{rec: rec}

	return toolDeps{
		desk: desk,
		inbox: inboundmessageservice.New(inboundmessageservice.Params{
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
		permissions: &permissionDesk{rec: rec},
	}
}

func (d toolDeps) fakes() []reflect.Value {
	return []reflect.Value{
		reflect.ValueOf(d.desk),
		reflect.ValueOf(d.inbox),
		reflect.ValueOf(d.memories),
		reflect.ValueOf(d.shipments),
		reflect.ValueOf(d.comments),
		reflect.ValueOf(d.permissions),
	}
}

type builtTools struct {
	queries []serviceports.AgentQueryTool
	actions []serviceports.AgentTool
}

func buildTools(rec *recorder, responses map[string]map[string]any) (builtTools, error) {
	fakes := newToolDeps(rec, responses).fakes()
	providers := append(
		agentquerytoolservice.ToolProviders(),
		agenttoolservice.ToolProviders()...,
	)

	built := builtTools{
		queries: make([]serviceports.AgentQueryTool, 0, len(providers)),
		actions: make([]serviceports.AgentTool, 0, len(providers)),
	}
	for _, provider := range providers {
		tool, err := reflectutils.Construct(provider, suppliedFor(provider, fakes))
		if err != nil {
			return builtTools{}, err
		}

		switch typed := tool.(type) {
		case serviceports.AgentQueryTool:
			built.queries = append(built.queries, &recordingQuery{
				inner:    typed,
				rec:      rec,
				response: responses[typed.Name()],
			})
		case serviceports.AgentTool:
			built.actions = append(built.actions, &recordingAction{inner: typed, rec: rec})
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
	inner    serviceports.AgentQueryTool
	rec      *recorder
	response map[string]any
}

func (q *recordingQuery) Name() string                    { return q.inner.Name() }
func (q *recordingQuery) Description() string             { return q.inner.Description() }
func (q *recordingQuery) ParamSchema() map[string]any     { return q.inner.ParamSchema() }
func (q *recordingQuery) Policy() serviceports.ToolPolicy { return q.inner.Policy() }

func (q *recordingQuery) SearchTerms() []string { return searchTermsOf(q.inner) }

func (q *recordingQuery) Prerequisites() []string { return prerequisitesOf(q.inner) }

func (q *recordingQuery) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
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
	inner serviceports.AgentTool
	rec   *recorder
}

func (a *recordingAction) Name() string                    { return a.inner.Name() }
func (a *recordingAction) Description() string             { return a.inner.Description() }
func (a *recordingAction) ParamSchema() map[string]any     { return a.inner.ParamSchema() }
func (a *recordingAction) Policy() serviceports.ToolPolicy { return a.inner.Policy() }

func (a *recordingAction) SearchTerms() []string { return searchTermsOf(a.inner) }

func (a *recordingAction) Prerequisites() []string { return prerequisitesOf(a.inner) }

func (a *recordingAction) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
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
	if searchable, ok := tool.(serviceports.SearchableTool); ok {
		return searchable.SearchTerms()
	}

	return nil
}

func prerequisitesOf(tool any) []string {
	if dependent, ok := tool.(serviceports.PrerequisiteTool); ok {
		return dependent.Prerequisites()
	}

	return nil
}
