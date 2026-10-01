package base

import (
	"sync"

	"github.com/emoss08/trenova/internal/api/graphql/gqlexec/internal/e2e/model"
)

// Core is what every resolver package shares.
type Core struct {
	mu    sync.Mutex
	calls int
}

func (c *Core) NextCall() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	return c.calls
}

func TruckByID(trucks []*model.Truck, id string) *model.Truck {
	for _, t := range trucks {
		if t.ID == id {
			return t
		}
	}
	return nil
}
