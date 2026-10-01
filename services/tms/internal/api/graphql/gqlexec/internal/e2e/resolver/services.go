package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/model"
	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/resolver/base"
)

type Params struct {
	Trucks  []*model.Truck
	Drivers []*model.Driver
}

// Services holds every dependency a resolver package may ask for. The
// generator hands each package only the fields its code names.
type Services struct {
	*base.Core
	Fleet  []*model.Truck
	Roster []*model.Driver
}

func newServices(p *Params) *Services {
	return &Services{Core: &base.Core{}, Fleet: p.Trucks, Roster: p.Drivers}
}
