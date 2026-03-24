package seeder

type Graph struct {
	nodes map[string]Seed
	edges map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		nodes: make(map[string]Seed),
		edges: make(map[string][]string),
	}
}

func (g *Graph) AddNode(seed Seed) {
	g.nodes[seed.Name()] = seed
	if _, exists := g.edges[seed.Name()]; !exists {
		g.edges[seed.Name()] = []string{}
	}
}

func (g *Graph) AddEdge(from, to string) {
	g.edges[from] = append(g.edges[from], to)
}

func (g *Graph) BuildFromSeeds(seeds []Seed) {
	for _, seed := range seeds {
		g.AddNode(seed)
	}

	for _, seed := range seeds {
		for _, dep := range seed.Dependencies() {
			g.AddEdge(seed.Name(), dep)
		}
	}
}

func (g *Graph) Validate() error {
	for name, deps := range g.edges {
		var missing []string
		for _, dep := range deps {
			if _, exists := g.nodes[dep]; !exists {
				missing = append(missing, dep)
			}
		}
		if len(missing) > 0 {
			return NewMissingDependencyError(name, missing)
		}
	}

	if cycle := g.DetectCycle(); len(cycle) > 0 {
		return NewCircularDependencyError(cycle)
	}

	return nil
}

func (g *Graph) DetectCycle() []string {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(node string) []string
	dfs = func(node string) []string {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, dep := range g.edges[node] {
			if !visited[dep] {
				if cycle := dfs(dep); cycle != nil {
					return cycle
				}
			} else if recStack[dep] {
				cycleStart := -1
				for i, n := range path {
					if n == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]string, len(path)-cycleStart+1)
					copy(cycle, path[cycleStart:])
					cycle[len(cycle)-1] = dep
					return cycle
				}
				return []string{dep}
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
		return nil
	}

	for node := range g.nodes {
		if !visited[node] {
			if cycle := dfs(node); cycle != nil {
				return cycle
			}
		}
	}

	return nil
}

func (g *Graph) TopologicalSort() ([]string, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}

	inDegree := make(map[string]int)
	for name := range g.nodes {
		inDegree[name] = 0
	}

	for _, deps := range g.edges {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	queue := make([]string, 0)
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	var result []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		result = append(result, name)

		for _, dep := range g.edges[name] {
			inDegree[dep]--
			if inDegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, NewCircularDependencyError(nil)
	}

	reversed := make([]string, len(result))
	for i, name := range result {
		reversed[len(result)-1-i] = name
	}

	return reversed, nil
}

func (g *Graph) GetDependenciesFor(seedName string) ([]string, error) {
	if _, exists := g.nodes[seedName]; !exists {
		return nil, ErrSeedNotFound
	}

	visited := make(map[string]bool)
	var result []string

	var collectDeps func(name string)
	collectDeps = func(name string) {
		for _, dep := range g.edges[name] {
			if !visited[dep] {
				visited[dep] = true
				collectDeps(dep)
				result = append(result, dep)
			}
		}
	}

	collectDeps(seedName)
	return result, nil
}

func (g *Graph) GetSeed(name string) (Seed, bool) {
	seed, exists := g.nodes[name]
	return seed, exists
}

func (g *Graph) Size() int {
	return len(g.nodes)
}
