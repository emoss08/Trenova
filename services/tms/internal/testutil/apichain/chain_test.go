package apichain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeChain(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name+".yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	return path
}

func TestLoad_ReadsAMixedGraphQLAndRESTChain(t *testing.T) {
	t.Parallel()

	path := writeChain(t, "mixed", `
name: mixed
actor: admin@trenova.app
fixtures:
  customer: SELECT id FROM customers WHERE code = 'ACME' AND organization_id = $1 AND business_unit_id = $2
steps:
  - id: create
    graphql: CreateShipment
    vars: { input: { customerId: "${fixture.customer}" } }
    export: { move: data.createShipment.moves.0.id }
  - id: assign
    rest: POST /api/v1/shipment-moves/${create.move}/assignment/
    body: { primaryWorkerId: wrk_1 }
  - id: read
    graphql: Shipment
    vars: { id: "${create.response.data.createShipment.id}" }
probes: [unauthenticated, other-tenant, no-permission]
`)

	chain, err := apichain.Load(path)

	require.NoError(t, err)
	assert.Len(t, chain.Steps, 3)
	assert.True(t, chain.Steps[0].IsGraphQL())
	assert.Equal(t, "POST", chain.Steps[1].Method())
	assert.Equal(t, "/api/v1/shipment-moves/${create.move}/assignment/", chain.Steps[1].Route())
	assert.Equal(t, filepath.Join(filepath.Dir(path), "mixed.golden.json"), chain.GoldenPath())
}

func TestLoad_RefusesWhatCannotRun(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		body string
		want string
	}{
		"unknown key": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: s, graphql: X, varz: {}}]\n",
			want: "field varz not found",
		},
		"name differs from the file": {
			body: "name: other\nactor: a@b.c\nsteps: [{id: s, graphql: X}]\n",
			want: `must match the file name "bad"`,
		},
		"both transports": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: s, graphql: X, rest: GET /x}]\n",
			want: "graphql or rest, not both",
		},
		"body on a graphql step": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: s, graphql: X, body: {a: 1}}]\n",
			want: "a graphql step sends vars",
		},
		"bad method": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: s, rest: FETCH /x}]\n",
			want: "must start with one of",
		},
		"forward reference": {
			body: "name: bad\nactor: a@b.c\nsteps:\n  - {id: a, graphql: X, vars: {id: '${b.id}'}}\n  - {id: b, graphql: Y}\n",
			want: `step "b", which has not run yet`,
		},
		"undeclared fixture": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: a, graphql: X, vars: {id: '${fixture.nope}'}}]\n",
			want: "names a fixture the chain does not declare",
		},
		"effect on a missing read": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: a, graphql: X, effects: [{read: z, path: p, unchanged: true}]}]\n",
			want: `reads step "z", which the chain does not have`,
		},
		"effect with two rules": {
			body: "name: bad\nactor: a@b.c\nsteps:\n  - {id: r, graphql: R}\n  - {id: a, graphql: X, effects: [{read: r, path: p, unchanged: true, equals: 1}]}\n",
			want: "exactly one of equals, increase, decrease or unchanged",
		},
		"duplicate step": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: a, graphql: X}, {id: a, graphql: Y}]\n",
			want: `id "a" is used twice`,
		},
		"openTo a probe the chain does not run": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: a, rest: GET /x, openTo: [no-permission]}]\n",
			want: `openTo names "no-permission", which the chain does not probe`,
		},
		"generated value that is not one reference": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: a, graphql: X}]\ngenerated: {pro: 'PRO-${a.response.pro}'}\n",
			want: "generated pro must be one reference",
		},
		"unknown probe": {
			body: "name: bad\nactor: a@b.c\nsteps: [{id: a, graphql: X}]\nprobes: [sideways]\n",
			want: `unknown probe "sideways"`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := apichain.Load(writeChain(t, "bad", tc.body))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}
