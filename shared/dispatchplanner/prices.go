package dispatchplanner

import (
	"cmp"
	"context"
	"slices"
)

func (s *search) price(ctx context.Context) {
	for resource, chain := range s.cur.chains {
		if len(chain) == 0 {
			continue
		}
		if ctx.Err() != nil {
			return
		}

		value := -s.cur.cost[resource]
		for at, task := range chain {
			value += s.priceTask(task, resource, at)
		}
		s.values[resource] = max(value, 0)
	}
}

func (s *search) priceTask(task, owner, at int) int64 {
	held := s.term(task, owner)
	cheapest := s.term(task, Unassigned)
	if s.opt.Alternatives == 0 {
		return cheapest - held
	}

	s.work = removed(s.work[:0], s.cur.chains[owner], at)
	shrunk, removable := s.chainCost(owner, s.work)
	release := shrunk - s.cur.cost[owner]

	alternatives := make([]Alternative, 0, s.opt.Alternatives)
	for _, resource := range s.candidates(task) {
		if len(alternatives) == s.opt.Alternatives {
			break
		}
		if resource == owner {
			continue
		}
		insertion, ok := s.insertionCost(task, resource)
		if !ok {
			continue
		}
		cover := insertion + s.term(task, resource)
		cheapest = min(cheapest, cover)
		if removable {
			alternatives = append(alternatives, Alternative{
				Resource:  resource,
				ExtraCost: release + cover - held,
			})
		}
	}

	slices.SortStableFunc(alternatives, func(a, b Alternative) int {
		if c := cmp.Compare(a.ExtraCost, b.ExtraCost); c != 0 {
			return c
		}
		return cmp.Compare(a.Resource, b.Resource)
	})
	s.alternatives[task] = alternatives

	return cheapest - held
}

func (s *search) insertionCost(task, resource int) (int64, bool) {
	target := s.cur.chains[resource]
	if s.full(target) {
		return 0, false
	}

	best, found := int64(0), false
	for _, at := range s.slots(task, target) {
		s.trial.reset()
		chain := inserted(s.trial.buffer(), target, task, at)
		s.trial.add(resource, chain)
		cost, ok := s.chainCost(resource, chain)
		if !ok {
			continue
		}
		if extra := cost - s.cur.cost[resource]; !found || extra < best {
			best, found = extra, true
		}
	}

	return best, found
}
