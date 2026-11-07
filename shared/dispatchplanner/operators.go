package dispatchplanner

type operator int

const (
	opInsert operator = iota
	opRemove
	opRelocate
	opExchange
	opReplace
	opTailExchange
	opReorder
	operatorCount
)

func (s *search) propose(op operator) {
	s.choice.reset()

	switch op {
	case opInsert:
		if task := s.randomFree(); task != Unassigned {
			s.bestInsertion(task)
		}
	case opReplace:
		if task := s.randomFree(); task != Unassigned {
			s.bestReplacement(task)
		}
	case opRemove:
		if task := s.randomAssigned(); task != Unassigned {
			s.removal(task)
		}
	case opRelocate:
		if task := s.randomAssigned(); task != Unassigned {
			s.bestRelocation(task)
		}
	case opExchange:
		if task := s.randomAssigned(); task != Unassigned {
			s.bestExchange(task)
		}
	case opTailExchange:
		if task := s.randomAssigned(); task != Unassigned {
			s.bestTailExchange(task)
		}
	case opReorder:
		if task := s.randomAssigned(); task != Unassigned {
			s.reorder(task)
		}
	case operatorCount:
	}
}

func (s *search) randomAssigned() int {
	if s.cur.assigned == 0 {
		return Unassigned
	}

	return s.cur.order[s.rng.IntN(s.cur.assigned)]
}

func (s *search) randomFree() int {
	free := s.tasks - s.cur.assigned
	if free == 0 {
		return Unassigned
	}

	return s.cur.order[s.cur.assigned+s.rng.IntN(free)]
}

func (s *search) bestInsertion(task int) {
	for _, resource := range s.candidates(task) {
		chain := s.cur.chains[resource]
		if s.full(chain) {
			continue
		}
		for _, at := range s.slots(task, chain) {
			s.trial.reset()
			s.trial.add(resource, inserted(s.trial.buffer(), chain, task, at))
			s.consider()
		}
	}
}

func (s *search) removal(task int) {
	resource := s.cur.owner[task]
	s.trial.reset()
	s.trial.add(resource, removed(s.trial.buffer(), s.cur.chains[resource], s.cur.pos[task]))
	s.consider()
}

func (s *search) bestRelocation(task int) {
	from, at := s.cur.owner[task], s.cur.pos[task]
	source := s.cur.chains[from]

	for _, resource := range s.candidates(task) {
		target := s.cur.chains[resource]
		if resource == from || s.full(target) {
			continue
		}
		for _, slot := range s.slots(task, target) {
			s.trial.reset()
			s.trial.add(from, removed(s.trial.buffer(), source, at))
			s.trial.add(resource, inserted(s.trial.buffer(), target, task, slot))
			s.consider()
		}
	}
}

func (s *search) bestExchange(task int) {
	from, at := s.cur.owner[task], s.cur.pos[task]
	source := s.cur.chains[from]

	for _, resource := range s.candidates(task) {
		target := s.cur.chains[resource]
		if resource == from || len(target) == 0 {
			continue
		}
		other := s.nearestIndex(target, s.opens(task))
		if !s.reaches(target[other], from) {
			continue
		}
		s.trial.reset()
		s.trial.add(from, replaced(s.trial.buffer(), source, at, target[other]))
		s.trial.add(resource, replaced(s.trial.buffer(), target, other, task))
		s.consider()
	}
}

func (s *search) bestReplacement(task int) {
	for _, resource := range s.candidates(task) {
		chain := s.cur.chains[resource]
		if len(chain) == 0 {
			continue
		}
		s.work = removed(s.work[:0], chain, s.nearestIndex(chain, s.opens(task)))
		for _, at := range s.slots(task, s.work) {
			s.trial.reset()
			s.trial.add(resource, inserted(s.trial.buffer(), s.work, task, at))
			s.consider()
		}
	}
}

func (s *search) bestTailExchange(task int) {
	second, at := s.cur.owner[task], s.cur.pos[task]
	right := s.cur.chains[second]

	for _, resource := range s.candidates(task) {
		if resource == second {
			continue
		}
		left := s.cur.chains[resource]
		slots := s.slots(task, left)
		cut := slots[len(slots)-1]
		s.trial.reset()
		s.trial.add(resource, joined(s.trial.buffer(), left[:cut], right[at:]))
		s.trial.add(second, joined(s.trial.buffer(), right[:at], left[cut:]))
		s.consider()
	}
}

func (s *search) reorder(task int) {
	resource, at := s.cur.owner[task], s.cur.pos[task]
	chain := s.cur.chains[resource]
	other := at + 1
	if other >= len(chain) {
		other = at - 1
	}
	if other < 0 {
		return
	}

	s.trial.reset()
	swapped := append(s.trial.buffer(), chain...)
	swapped[at], swapped[other] = swapped[other], swapped[at]
	s.trial.add(resource, swapped)
	s.consider()
}
