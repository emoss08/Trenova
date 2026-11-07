package dispatchplanner

type plan struct {
	chains    [][]int
	cost      []int64
	owner     []int
	pos       []int
	order     []int
	slot      []int
	assigned  int
	objective int64
}

func newPlan(tasks, resources int) *plan {
	p := &plan{
		chains: make([][]int, resources),
		cost:   make([]int64, resources),
		owner:  make([]int, tasks),
		pos:    make([]int, tasks),
		order:  make([]int, tasks),
		slot:   make([]int, tasks),
	}
	for task := range tasks {
		p.owner[task] = Unassigned
		p.pos[task] = Unassigned
		p.order[task] = task
		p.slot[task] = task
	}

	return p
}

func (p *plan) release(task int) {
	if p.owner[task] == Unassigned {
		return
	}
	p.owner[task] = Unassigned
	p.pos[task] = Unassigned
	p.assigned--
	p.swapOrder(p.slot[task], p.assigned)
}

func (p *plan) take(task, resource, at int) {
	if p.owner[task] == Unassigned {
		p.swapOrder(p.slot[task], p.assigned)
		p.assigned++
	}
	p.owner[task] = resource
	p.pos[task] = at
}

func (p *plan) swapOrder(i, j int) {
	a, b := p.order[i], p.order[j]
	p.order[i], p.order[j] = b, a
	p.slot[b], p.slot[a] = i, j
}

type change struct {
	resource int
	chain    []int
	cost     int64
}

type proposal struct {
	changes [2]change
	bufs    [2][]int
	n       int
	delta   int64
	valid   bool
}

func (pr *proposal) reset() {
	pr.n = 0
	pr.valid = false
}

func (pr *proposal) buffer() []int {
	return pr.bufs[pr.n][:0]
}

func (pr *proposal) add(resource int, chain []int) {
	pr.bufs[pr.n] = chain
	pr.changes[pr.n] = change{resource: resource, chain: chain}
	pr.n++
}

func (pr *proposal) copyFrom(src *proposal) {
	pr.n = src.n
	pr.delta = src.delta
	pr.valid = src.valid
	for i := range src.n {
		pr.bufs[i] = append(pr.bufs[i][:0], src.changes[i].chain...)
		pr.changes[i] = change{
			resource: src.changes[i].resource,
			chain:    pr.bufs[i],
			cost:     src.changes[i].cost,
		}
	}
}

func inserted(dst, chain []int, task, at int) []int {
	dst = append(dst, chain[:at]...)
	dst = append(dst, task)

	return append(dst, chain[at:]...)
}

func removed(dst, chain []int, at int) []int {
	dst = append(dst, chain[:at]...)

	return append(dst, chain[at+1:]...)
}

func replaced(dst, chain []int, at, task int) []int {
	start := len(dst)
	dst = append(dst, chain...)
	dst[start+at] = task

	return dst
}

func joined(dst, head, tail []int) []int {
	dst = append(dst, head...)

	return append(dst, tail...)
}
