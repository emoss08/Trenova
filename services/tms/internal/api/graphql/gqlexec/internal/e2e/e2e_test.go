package e2e_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/model"
	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/resolver"
	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/resolver/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServer(t *testing.T) *handler.Server {
	t.Helper()

	capacity := 12
	root := resolver.New(base.Params{
		Trucks: []*model.Truck{
			{
				ID:       "t1",
				Name:     "Kenworth",
				Status:   model.StatusActive,
				Owner:    &model.Owner{ID: "o1", Name: "Acme"},
				Tags:     []string{"reefer", "team"},
				Capacity: &capacity,
			},
			{ID: "t2", Name: "Peterbilt", Status: model.StatusActive, Tags: []string{}},
			{ID: "t3", Name: "Mack", Status: model.StatusInactive, Tags: []string{}},
		},
		Drivers: []*model.Driver{
			{ID: "d1", Name: "Ada", Rating: 4.5, TruckID: "t1"},
			{ID: "d2", Name: "Grace", Rating: 5, TruckID: "missing"},
		},
	})

	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: root}))
	srv.AddTransport(transport.POST{})
	srv.Use(extension.Introspection{})
	return srv
}

func post(t *testing.T, srv http.Handler, body string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	out, err := io.ReadAll(rec.Body)
	require.NoError(t, err)

	var resp map[string]any
	require.NoError(t, sonic.Unmarshal(out, &resp))
	if errs, ok := resp["errors"].([]any); ok {
		for _, e := range errs {
			delete(e.(map[string]any), "locations")
		}
	}
	normalized, err := sonic.MarshalString(resp)
	require.NoError(t, err)
	return normalized
}

func TestExecution(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body string
		want string
	}{
		"fields, methods, resolvers and defaults": {
			body: `{"query":"{ truck(id: \"t1\") { __typename id name status displayName tags capacity owner { name } loadCount twice: loadCount(since: 3) } }"}`,
			want: `{"data":{"truck":{"__typename":"Truck","id":"t1","name":"Kenworth","status":"ACTIVE","displayName":"t1 Kenworth","tags":["reefer","team"],"capacity":12,"owner":{"name":"Acme"},"loadCount":14,"twice":6}}}`,
		},
		"a nullable field that fails is null alone": {
			body: `{"query":"{ truck(id: \"t1\") { id nullableFailing } }"}`,
			want: `{"errors":[{"message":"nullable failing field","path":["truck","nullableFailing"]}],"data":{"truck":{"id":"t1","nullableFailing":null}}}`,
		},
		"a non-null field that fails nulls its nullable parent": {
			body: `{"query":"{ truck(id: \"t1\") { id failing } }"}`,
			want: `{"errors":[{"message":"failing field","path":["truck","failing"]}],"data":{"truck":null}}`,
		},
		"a failure inside a non-null list reaches the root": {
			body: `{"query":"{ trucks(filter: {nameContains: \"Ken\"}) { id failing } }"}`,
			want: `{"errors":[{"message":"failing field","path":["trucks",0,"failing"]}],"data":null}`,
		},
		"a resolver error nulls the field it resolves": {
			body: `{"query":"{ truck(id: \"boom\") { id } }"}`,
			want: `{"errors":[{"message":"truck not found","path":["truck"]}],"data":{"truck":null}}`,
		},
		"a panic becomes an internal error": {
			body: `{"query":"{ panics }"}`,
			want: `{"errors":[{"message":"internal system error","path":["panics"]}],"data":{"panics":null}}`,
		},
		"nullable list elements stay null": {
			body: `{"query":"{ mixed { id } }"}`,
			want: `{"data":{"mixed":[{"id":"t1"},null,{"id":"t2"}]}}`,
		},
		"interfaces and unions dispatch on the Go type": {
			body: `{"query":"{ truck: node(id: \"t2\") { __typename id } driver: node(id: \"d1\") { __typename id ... on Driver { rating } } none: node(id: \"x\") { id } search(text: \"a\") { __typename ... on Truck { name } ... on Driver { name } } }"}`,
			want: `{"data":{"truck":{"__typename":"Truck","id":"t2"},"driver":{"__typename":"Driver","id":"d1","rating":4.5},"none":null,"search":[{"__typename":"Truck","name":"Kenworth"},{"__typename":"Driver","name":"Ada"}]}}`,
		},
		"object resolvers from another schema file": {
			body: `{"query":"{ drivers(page: {first: 5}) { name truck { name } } }"}`,
			want: `{"data":{"drivers":[{"name":"Ada","truck":{"name":"Kenworth"}},{"name":"Grace","truck":null}]}}`,
		},
		"input defaults apply across schema files": {
			body: `{"query":"query ($f: TruckFilter!) { echoFilter(filter: $f) }","variables":{"f":{"nameContains":"a","page":{"after":"c"}}}}`,
			want: `{"data":{"echoFilter":"status=ACTIVE name=a first=10 after=c"}}`,
		},
		"an omitted argument takes no default": {
			body: `{"query":"{ trucks { id } inactive: trucks(filter: {status: INACTIVE}) { id } }"}`,
			want: `{"data":{"trucks":[{"id":"t1"},{"id":"t2"}],"inactive":[{"id":"t3"}]}}`,
		},
		"omittable input fields tell absent from null": {
			body: `{"query":"mutation { absent: renameTruck(input: {id: \"t1\", name: \"A\"}) { name tags } null: renameTruck(input: {id: \"t1\", name: \"B\", note: null}) { tags } set: renameTruck(input: {id: \"t1\", name: \"C\", note: \"hi\"}) { tags } }"}`,
			want: `{"data":{"absent":{"name":"A","tags":["note:absent"]},"null":{"tags":["note:null"]},"set":{"tags":["note:hi"]}}}`,
		},
		"invalid input is rejected with its path": {
			body: `{"query":"query ($f: TruckFilter!) { echoFilter(filter: $f) }","variables":{"f":{"status":"PARKED"}}}`,
			want: `{"errors":[{"message":"PARKED is not a valid Status","path":["variable","f","status"],"extensions":{"code":"GRAPHQL_VALIDATION_FAILED"}}],"data":null}`,
		},
		"introspection": {
			body: `{"query":"{ __type(name: \"SearchResult\") { kind possibleTypes { name } } }"}`,
			want: `{"data":{"__type":{"kind":"UNION","possibleTypes":[{"name":"Truck"},{"name":"Driver"}]}}}`,
		},
	}

	srv := newServer(t)
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.JSONEq(t, tc.want, post(t, srv, tc.body))
		})
	}
}

func TestMutationFieldsRunInOrder(t *testing.T) {
	t.Parallel()

	got := post(t, newServer(t), `{"query":"mutation { a: touch(ids: [\"x\", \"y\"]) b: touch(ids: [\"z\"]) c: touch(ids: []) }"}`)

	assert.JSONEq(t, `{"data":{"a":["1:x","1:y"],"b":["2:z"],"c":[]}}`, got)
}

func TestFieldContextMatchesGqlgen(t *testing.T) {
	t.Parallel()

	type seen struct {
		isMethod, isResolver bool
		args                 map[string]any
	}
	var (
		mu     sync.Mutex
		fields = map[string]seen{}
	)
	srv := newServer(t)
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (any, error) {
		fc := graphql.GetFieldContext(ctx)
		mu.Lock()
		fields[fc.Object+"."+fc.Field.Name] = seen{fc.IsMethod, fc.IsResolver, fc.Args}
		mu.Unlock()
		return next(ctx)
	})

	post(t, srv, `{"query":"{ truck(id: \"t1\") { name displayName loadCount(since: 2) } }"}`)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, seen{true, true, map[string]any{"id": "t1"}}, fields["Query.truck"])
	assert.Equal(t, seen{false, false, nil}, fields["Truck.name"])
	assert.Equal(t, seen{true, false, nil}, fields["Truck.displayName"])
	since := 2
	assert.Equal(t, seen{true, true, map[string]any{"since": &since}}, fields["Truck.loadCount"])
}
