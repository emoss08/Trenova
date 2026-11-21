package play

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/emoss08/trenova/edi-partner-sim/internal/provision"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
)

var errNoResources = errors.New("the tenant has no available tractor with a driver that is " +
	"not already on a move in progress")

type fleet struct {
	mu      sync.Mutex
	next    uint64
	driving map[string]bool
}

func (f *fleet) turn() uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	turn := f.next
	f.next++
	return turn
}

func (f *fleet) drive(shipmentID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.driving == nil {
		f.driving = map[string]bool{}
	}
	f.driving[shipmentID] = true
}

func (f *fleet) park(shipmentID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.driving, shipmentID)
}

func (f *fleet) isDriving(shipmentID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.driving[shipmentID]
}

func (e *Env) pair(ctx context.Context) (string, string, error) {
	free, held, err := e.tractors(ctx)
	if err != nil {
		return "", "", err
	}
	if len(free) == 0 && len(held) > 0 {
		if err = e.bringHome(ctx, held); err != nil {
			return "", "", err
		}
		if free, _, err = e.tractors(ctx); err != nil {
			return "", "", err
		}
	}
	if len(free) == 0 {
		return "", "", errNoResources
	}
	tractor := free[e.fleet.turn()%uint64(len(free))]
	return tractor.PrimaryWorkerID, tractor.ID, nil
}

func (e *Env) tractors(ctx context.Context) ([]trenova.Tractor, map[string]bool, error) {
	crewed, err := e.Trenova.AvailableTractors(ctx, resourceLimit)
	if err != nil {
		return nil, nil, err
	}
	free := make([]trenova.Tractor, 0, len(crewed))
	held := map[string]bool{}
	for _, tractor := range crewed {
		if tractor.InProgressMoveID == "" {
			free = append(free, tractor)
		} else {
			held[tractor.InProgressMoveID] = true
		}
	}
	return free, held, nil
}

func (e *Env) bringHome(ctx context.Context, held map[string]bool) error {
	partners, err := e.Trenova.Partners(ctx, provision.CodePrefix)
	if err != nil {
		return fmt.Errorf("list the simulator's partners: %w", err)
	}
	seen := map[string]bool{}
	for index := range partners {
		if len(held) == 0 {
			return nil
		}
		if !strings.HasPrefix(strings.ToUpper(partners[index].Code), provision.CodePrefix) {
			continue
		}
		instances, listErr := e.Trenova.FlowInstances(ctx, partners[index].ID, "")
		if listErr != nil {
			return fmt.Errorf("list %s's flows: %w", partners[index].Code, listErr)
		}
		for _, instance := range instances {
			id := instance.ShipmentID
			if id == "" || seen[id] || e.fleet.isDriving(id) {
				continue
			}
			seen[id] = true
			brought, homeErr := e.bringShipmentHome(ctx, id, held)
			if homeErr != nil {
				return fmt.Errorf("bring home the load for %s: %w", instance.Key, homeErr)
			}
			if brought {
				e.logf("  brought home the load for %s, left on the road by an earlier "+
					"scenario, to free its tractor", instance.Key)
			}
			if len(held) == 0 {
				return nil
			}
		}
	}
	return nil
}

func (e *Env) bringShipmentHome(
	ctx context.Context,
	shipmentID string,
	held map[string]bool,
) (bool, error) {
	shipment, err := e.Trenova.Shipment(ctx, shipmentID)
	if errors.Is(err, trenova.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	holding := false
	for _, move := range shipment.Moves {
		if held[move.ID] {
			holding = true
			delete(held, move.ID)
		}
	}
	if !holding {
		return false, nil
	}
	return true, e.finishStops(ctx, shipment)
}

func (e *Env) finishStops(ctx context.Context, shipment *trenova.Shipment) error {
	for index, stop := range shipment.Stops() {
		if stop.ActualArrival == nil {
			if err := e.Trenova.RecordStop(ctx, stop.MoveID, stop.ID,
				trenova.ActionArrive); err != nil {
				return fmt.Errorf("arrive at stop %d: %w", index+1, err)
			}
		}
		if stop.ActualDeparture == nil {
			if err := e.Trenova.RecordStop(ctx, stop.MoveID, stop.ID,
				trenova.ActionDepart); err != nil {
				return fmt.Errorf("depart stop %d: %w", index+1, err)
			}
		}
	}
	return nil
}
