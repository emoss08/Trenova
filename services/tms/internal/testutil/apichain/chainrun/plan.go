package chainrun

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/gin-gonic/gin"
)

type Call struct {
	Step      *apichain.Step
	Operation apichain.Operation
	Route     string
}

func (c Call) Label() string {
	if c.Step.IsGraphQL() {
		return "graphql " + c.Operation.Name
	}

	return c.Step.Method() + " " + c.Route
}

func (c Call) IsRead() bool {
	if c.Step.IsGraphQL() {
		return c.Operation.IsQuery()
	}

	return c.Step.Method() == http.MethodGet
}

type Plan struct {
	Chain *apichain.Chain
	Calls []Call
}

func (p *Plan) Call(id string) (Call, bool) {
	for _, call := range p.Calls {
		if call.Step.ID == id {
			return call, true
		}
	}

	return Call{}, false
}

func Prepare(
	chain *apichain.Chain,
	manifest *apichain.Manifest,
	routes gin.RoutesInfo,
) (*Plan, error) {
	plan := &Plan{Chain: chain, Calls: make([]Call, 0, len(chain.Steps))}
	var errs []error
	for i := range chain.Steps {
		step := &chain.Steps[i]
		call, err := prepareCall(step, manifest, routes)
		if err != nil {
			errs = append(errs, fmt.Errorf("step %s: %w", step.ID, err))
			continue
		}
		plan.Calls = append(plan.Calls, call)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s: %w", chain.Path, errors.Join(errs...))
	}

	for _, call := range plan.Calls {
		for _, effect := range call.Step.Effects {
			read, _ := plan.Call(effect.Read)
			if !read.IsRead() {
				errs = append(errs, fmt.Errorf(
					"step %s: effect reads %s (%s), which is not a GraphQL query or a GET",
					call.Step.ID, effect.Read, read.Label(),
				))
			}
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("%s: %w", chain.Path, errors.Join(errs...))
	}

	return plan, nil
}

func prepareCall(
	step *apichain.Step,
	manifest *apichain.Manifest,
	routes gin.RoutesInfo,
) (Call, error) {
	if step.IsGraphQL() {
		if step.Document != "" {
			operation, err := apichain.ParseOperation(step.GraphQL, step.Document)
			if err != nil {
				return Call{}, err
			}
			if _, inManifest := manifest.Operation(step.GraphQL); inManifest {
				return Call{}, fmt.Errorf(
					"%s is in the persisted manifest: drop document and send the client's",
					step.GraphQL,
				)
			}
			return Call{Step: step, Operation: operation}, nil
		}
		operation, ok := manifest.Operation(step.GraphQL)
		if !ok {
			return Call{}, fmt.Errorf(
				"%s is not in the persisted manifest, so no client sends it: "+
					"name a client operation, or give document for an API-only caller",
				step.GraphQL,
			)
		}
		return Call{Step: step, Operation: operation}, nil
	}

	route, ok := matchRoute(step.Method(), step.Route(), routes)
	if !ok {
		return Call{}, fmt.Errorf("%s matches no registered route", step.REST)
	}

	return Call{Step: step, Route: route}, nil
}

func matchRoute(method, path string, routes gin.RoutesInfo) (string, bool) {
	want := strings.Split(path, "/")
	best, bestLiterals := "", -1
	for _, route := range routes {
		if route.Method != method {
			continue
		}
		literals, ok := routeLiterals(strings.Split(route.Path, "/"), want)
		if ok && literals > bestLiterals {
			best, bestLiterals = route.Path, literals
		}
	}

	return best, bestLiterals >= 0
}

func routeLiterals(have, want []string) (int, bool) {
	if len(have) != len(want) {
		return 0, false
	}
	literals := 0
	for i := range have {
		if strings.HasPrefix(have[i], ":") || strings.HasPrefix(have[i], "*") {
			continue
		}
		if strings.Contains(want[i], "${") || have[i] != want[i] {
			return 0, false
		}
		literals++
	}

	return literals, true
}
