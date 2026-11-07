package dispatchplanner

type prefixCache struct {
	timeline Timeline
	limit    int
	roots    map[prefixKey]int32
	children map[prefixKey]int32
	cost     []int64
	feasible []bool
	hits     int64
	misses   int64
	flushes  int
}

func newPrefixCache(timeline Timeline, limit int) *prefixCache {
	return &prefixCache{
		timeline: timeline,
		limit:    limit,
		roots:    make(map[prefixKey]int32),
		children: make(map[prefixKey]int32),
	}
}

func (c *prefixCache) chain(resource int, tasks []int) (int64, bool) {
	c.reserve(len(tasks))

	parent := Root
	total := int64(0)
	for _, task := range tasks {
		node := c.child(resource, parent, task)
		if !c.feasible[node] {
			return 0, false
		}
		total += c.cost[node]
		parent = node
	}

	return total, true
}

func (c *prefixCache) steps(resource int, tasks []int, dst []int64) ([]int64, bool) {
	c.reserve(len(tasks))

	parent := Root
	for _, task := range tasks {
		node := c.child(resource, parent, task)
		if !c.feasible[node] {
			return dst, false
		}
		dst = append(dst, c.cost[node])
		parent = node
	}

	return dst, true
}

func (c *prefixCache) reserve(steps int) {
	if len(c.cost)+steps > c.limit {
		c.flush()
	}
}

func (c *prefixCache) child(resource int, parent int32, task int) int32 {
	table, key := c.children, prefixKey{owner: int(parent), task: task}
	if parent == Root {
		table, key = c.roots, prefixKey{owner: resource, task: task}
	}
	if node, ok := table[key]; ok {
		c.hits++
		return node
	}

	c.misses++
	//nolint:gosec // G115: reserve flushes before len(c.cost) passes c.limit, at most 1<<24.
	node := int32(len(c.cost))
	step := c.timeline.Extend(Extension{
		Resource: resource,
		Task:     task,
		Parent:   parent,
		Node:     node,
	})
	c.cost = append(c.cost, step.Cost)
	c.feasible = append(c.feasible, step.Feasible)
	table[key] = node

	return node
}

func (c *prefixCache) flush() {
	clear(c.roots)
	clear(c.children)
	c.cost = c.cost[:0]
	c.feasible = c.feasible[:0]
	c.flushes++
	c.timeline.Reset()
}

type prefixKey struct {
	owner int
	task  int
}
