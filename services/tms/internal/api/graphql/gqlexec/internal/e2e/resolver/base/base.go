package base

import (
	"sync"

	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/model"
)

type Params struct {
	Trucks  []*model.Truck
	Drivers []*model.Driver
}

type Resolver struct {
	Trucks  []*model.Truck
	Drivers []*model.Driver

	mu    sync.Mutex
	calls int
}

func New(p Params) *Resolver {
	return &Resolver{Trucks: p.Trucks, Drivers: p.Drivers}
}

func (r *Resolver) Truck(id string) *model.Truck {
	for _, t := range r.Trucks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (r *Resolver) NextCall() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.calls
}
