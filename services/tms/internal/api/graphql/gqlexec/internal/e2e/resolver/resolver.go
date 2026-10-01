package resolver

import (
	"sync"

	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/model"
)

type Resolver struct {
	trucks  []*model.Truck
	drivers []*model.Driver

	mu    sync.Mutex
	calls int
}

func New(trucks []*model.Truck, drivers []*model.Driver) *Resolver {
	return &Resolver{trucks: trucks, drivers: drivers}
}

func (r *Resolver) truckByID(id string) *model.Truck {
	for _, t := range r.trucks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

func (r *Resolver) nextCall() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.calls
}

func deref[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}
