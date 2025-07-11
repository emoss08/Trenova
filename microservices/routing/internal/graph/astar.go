package graph

import (
	"container/heap"
	"errors"
	"sync"

	"github.com/paulmach/orb/geo"
)

// DefaultMaxSearchNodes is the default limit for search nodes
const DefaultMaxSearchNodes = 100000

// nodePool is a sync.Pool for reusing node-related data structures
var nodePool = sync.Pool{
	New: func() any {
		return &nodeData{
			gScore:    make(map[int64]float64, 1000),
			fScore:    make(map[int64]float64, 1000),
			cameFrom:  make(map[int64]int64, 1000),
			timeScore: make(map[int64]float64, 1000),
			closedSet: make(map[int64]bool, 1000),
		}
	},
}

type nodeData struct {
	gScore      map[int64]float64
	fScore      map[int64]float64
	cameFrom    map[int64]int64
	timeScore   map[int64]float64
	closedSet   map[int64]bool
	searchNodes int
}

// AStar implements the A* pathfinding algorithm with optimizations
func (g *Graph) AStar(startID, endID int64, opts PathOptions) (*PathResult, error) {
	start, ok := g.Nodes[startID]
	if !ok {
		return nil, ErrNodeNotFound
	}

	end, ok := g.Nodes[endID]
	if !ok {
		return nil, ErrNodeNotFound
	}

	// _ Early exit if start and end are the same
	if startID == endID {
		return &PathResult{
			Path: []*Node{start},
			PathNodes: []PathNode{
				{ID: start.ID, Location: []float64{start.Location[0], start.Location[1]}},
			},
			Distance:   0,
			TravelTime: 0,
		}, nil
	}

	// _ Get pooled data structures
	data, ok := nodePool.Get().(*nodeData)
	if !ok {
		return nil, errors.New("failed to get nodeData from pool")
	}
	defer func() {
		// _ Clear maps for reuse
		clear(data.gScore)
		clear(data.fScore)
		clear(data.cameFrom)
		clear(data.timeScore)
		clear(data.closedSet)
		data.searchNodes = 0
		nodePool.Put(data)
	}()

	// ! Priority queue for open set
	openSet := &priorityQueue{}
	heap.Init(openSet)

	// _ Initialize start node
	data.gScore[startID] = 0
	data.fScore[startID] = heuristic(start, end)
	data.timeScore[startID] = 0

	heap.Push(openSet, &item{
		node:     start,
		priority: data.fScore[startID],
	})

	for openSet.Len() > 0 {
		currentItem := heap.Pop(openSet)
		poppedItem, validItem := currentItem.(*item)
		if !validItem {
			continue
		}
		current := poppedItem.node

		// _ Skip if already processed
		if data.closedSet[current.ID] {
			continue
		}

		if current.ID == endID {
			// _ Reconstruct path
			path := reconstructPath(data.cameFrom, current.ID, g.Nodes)
			// _ Convert to PathNodes for JSON serialization
			pathNodes := make([]PathNode, len(path))
			for i, node := range path {
				pathNodes[i] = PathNode{
					ID:       node.ID,
					Location: []float64{node.Location[0], node.Location[1]},
				}
			}
			return &PathResult{
				Path:        path,
				PathNodes:   pathNodes,
				Distance:    data.gScore[endID],
				TravelTime:  data.timeScore[endID],
				SearchNodes: data.searchNodes,
			}, nil
		}

		data.closedSet[current.ID] = true
		data.searchNodes++

		// _ Check search space limit
		if data.searchNodes > 100000 {
			return nil, ErrSearchSpaceLimit
		}

		for _, edge := range current.Edges {
			// _ Skip if neighbor already processed
			if data.closedSet[edge.To.ID] {
				continue
			}

			// _ Check truck restrictions
			if !isEdgeTraversable(edge, opts) {
				continue
			}

			neighbor := edge.To
			tentativeGScore := data.gScore[current.ID] + edge.Distance

			if currentGScore, exists := data.gScore[neighbor.ID]; !exists ||
				tentativeGScore < currentGScore {
				// _ This path to neighbor is better
				data.cameFrom[neighbor.ID] = current.ID
				data.gScore[neighbor.ID] = tentativeGScore
				tentativeTimeScore := data.timeScore[current.ID] + edge.TravelTime
				data.timeScore[neighbor.ID] = tentativeTimeScore

				// _ Calculate priority based on optimization type
				var priority float64
				switch opts.OptimizationType {
				case OptimizeShortest:
					data.fScore[neighbor.ID] = tentativeGScore + heuristic(neighbor, end)
					priority = data.fScore[neighbor.ID]
				case OptimizeFastest:
					// _ Optimize for time: use time score + time heuristic
					dist := heuristic(neighbor, end)
					timeHeuristic := (dist / 1609.34) / 65 * 3600 // _ Estimate time at 65 mph
					data.fScore[neighbor.ID] = tentativeTimeScore + timeHeuristic
					priority = data.fScore[neighbor.ID]
				case OptimizePractical:
					// _ Balance distance and time
					dist := heuristic(neighbor, end)
					timeHeuristic := (dist / 1609.34) / 55 * 3600 // _ 55 mph average
					practicalScore := tentativeGScore*0.3 + tentativeTimeScore*0.7
					practicalHeuristic := dist*0.3 + timeHeuristic*0.7
					data.fScore[neighbor.ID] = practicalScore + practicalHeuristic
					priority = data.fScore[neighbor.ID]
				default: // OptimizeShortest
					data.fScore[neighbor.ID] = tentativeGScore + heuristic(neighbor, end)
					priority = data.fScore[neighbor.ID]
				}

				// _ Add to open set
				heap.Push(openSet, &item{
					node:     neighbor,
					priority: priority,
				})
			}
		}
	}

	return nil, ErrNoPathFound
}

// heuristic calculates the heuristic distance between two nodes using haversine formula
func heuristic(a, b *Node) float64 {
	return geo.Distance(a.Location, b.Location)
}

// isEdgeTraversable checks if an edge can be traversed given the constraints
func isEdgeTraversable(edge *Edge, opts PathOptions) bool {
	if opts.TruckOnly && !edge.TruckAllowed {
		return false
	}

	if opts.MaxHeight > 0 && edge.MaxHeight > 0 && opts.MaxHeight > edge.MaxHeight {
		return false
	}

	if opts.MaxWeight > 0 && edge.MaxWeight > 0 && opts.MaxWeight > edge.MaxWeight {
		return false
	}

	return true
}

// reconstructPath rebuilds the path from the cameFrom map
func reconstructPath(cameFrom map[int64]int64, currentID int64, nodes map[int64]*Node) []*Node {
	// _ Count path length first to pre-allocate
	pathLen := 1
	tempID := currentID
	for {
		if prevID, exists := cameFrom[tempID]; exists {
			pathLen++
			tempID = prevID
		} else {
			break
		}
	}

	// _ Pre-allocate path slice
	path := make([]*Node, pathLen)
	idx := pathLen - 1
	path[idx] = nodes[currentID]

	for idx > 0 {
		if prevID, exists := cameFrom[currentID]; exists {
			idx--
			path[idx] = nodes[prevID]
			currentID = prevID
		} else {
			break
		}
	}

	return path
}

// Priority queue implementation for A*
type item struct {
	node     *Node
	priority float64
	index    int
}

type priorityQueue []*item

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x any) {
	n := len(*pq)
	it, ok := x.(*item)
	if !ok {
		return
	}
	it.index = n
	*pq = append(*pq, it)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	it.index = -1
	*pq = old[0 : n-1]
	return it
}
