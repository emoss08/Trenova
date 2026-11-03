package benchscenario

import (
	"slices"
)

const (
	delegationTag = "delegation"
)

type Route string

const (
	RouteChat       = Route("chat")
	RouteUnattended = Route("unattended")
	RouteDelegated  = Route("delegated")
)

func AllRoutes() []Route {
	return []Route{RouteChat, RouteUnattended, RouteDelegated}
}

func (r Route) IsValid() bool {
	return slices.Contains(AllRoutes(), r)
}

func RouteOf(scenario *Scenario) Route {
	switch {
	case scenario.Unattended():
		return RouteUnattended
	case slices.Contains(scenario.Tags, delegationTag) ||
		Labelled("delegate_task", scenario.ExpectedTools):
		return RouteDelegated
	default:
		return RouteChat
	}
}
