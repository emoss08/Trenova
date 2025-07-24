/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package graph

import (
	"container/heap"
	"math"
	"sync"
)

// BidirectionalAStar implements bidirectional A* for improved performance on long routes
func (g *Graph) BidirectionalAStar(startID, endID int64, opts PathOptions) (*PathResult, error) {
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

	// _ Initialize forward and backward search data
	forward := &searchState{
		openSet:   &priorityQueue{},
		gScore:    make(map[int64]float64, 1000),
		fScore:    make(map[int64]float64, 1000),
		cameFrom:  make(map[int64]int64, 1000),
		timeScore: make(map[int64]float64, 1000),
		closedSet: make(map[int64]bool, 1000),
	}

	backward := &searchState{
		openSet:   &priorityQueue{},
		gScore:    make(map[int64]float64, 1000),
		fScore:    make(map[int64]float64, 1000),
		cameFrom:  make(map[int64]int64, 1000),
		timeScore: make(map[int64]float64, 1000),
		closedSet: make(map[int64]bool, 1000),
	}

	heap.Init(forward.openSet)
	heap.Init(backward.openSet)

	// _ Initialize start and end nodes
	forward.gScore[startID] = 0
	forward.fScore[startID] = heuristic(start, end)
	forward.timeScore[startID] = 0

	backward.gScore[endID] = 0
	backward.fScore[endID] = heuristic(end, start)
	backward.timeScore[endID] = 0

	heap.Push(forward.openSet, &item{node: start, priority: forward.fScore[startID]})
	heap.Push(backward.openSet, &item{node: end, priority: backward.fScore[endID]})

	var (
		meetingNode  int64
		bestPathCost = math.Inf(1)
		mu           sync.RWMutex
	)

	// _ Process both directions
	for forward.openSet.Len() > 0 || backward.openSet.Len() > 0 {
		// _ Expand forward search
		if forward.openSet.Len() > 0 {
			if node := expandSearch(g, forward, backward, end, opts, true); node != -1 {
				mu.Lock()
				cost := forward.gScore[node] + backward.gScore[node]
				if cost < bestPathCost {
					bestPathCost = cost
					meetingNode = node
				}
				mu.Unlock()
			}
		}

		// _ Expand backward search
		if backward.openSet.Len() > 0 {
			if node := expandSearch(g, backward, forward, start, opts, false); node != -1 {
				mu.Lock()
				cost := forward.gScore[node] + backward.gScore[node]
				if cost < bestPathCost {
					bestPathCost = cost
					meetingNode = node
				}
				mu.Unlock()
			}
		}

		// _ Check termination condition
		mu.RLock()
		terminate := bestPathCost < math.Inf(1) &&
			(forward.openSet.Len() == 0 || backward.openSet.Len() == 0 ||
				forward.fScore[forward.openSet.peek().node.ID]+backward.fScore[backward.openSet.peek().node.ID] >= bestPathCost)
		mu.RUnlock()

		if terminate {
			break
		}
	}

	if meetingNode == 0 {
		return nil, ErrNoPathFound
	}

	// _ Reconstruct the complete path
	path := reconstructBidirectionalPath(forward.cameFrom, backward.cameFrom, meetingNode, g.Nodes)

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
		Distance:    bestPathCost,
		TravelTime:  forward.timeScore[meetingNode] + backward.timeScore[meetingNode],
		SearchNodes: len(forward.closedSet) + len(backward.closedSet),
	}, nil
}

type searchState struct {
	openSet   *priorityQueue
	gScore    map[int64]float64
	fScore    map[int64]float64
	cameFrom  map[int64]int64
	timeScore map[int64]float64
	closedSet map[int64]bool
}

func expandSearch(
	g *Graph,
	current, opposite *searchState,
	target *Node,
	opts PathOptions,
	isForward bool,
) int64 {
	if current.openSet.Len() == 0 {
		return -1
	}

	poppedItem := heap.Pop(current.openSet)
	currentItem, ok := poppedItem.(*item)
	if !ok {
		return -1
	}
	node := currentItem.node

	if current.closedSet[node.ID] {
		return -1
	}

	current.closedSet[node.ID] = true

	// _ Check if this node has been reached from the opposite direction
	if _, exists := opposite.gScore[node.ID]; exists {
		return node.ID
	}

	for _, edge := range node.Edges {
		var neighbor *Node
		var edgeToUse *Edge

		if isForward {
			if !isEdgeTraversable(edge, opts) {
				continue
			}
			neighbor = edge.To
			edgeToUse = edge
		} else {
			// _ For backward search, we need to find reverse edges
			reverseEdge := findReverseEdge(g, edge)
			if reverseEdge == nil || !isEdgeTraversable(reverseEdge, opts) {
				continue
			}
			neighbor = edge.From
			edgeToUse = reverseEdge
		}

		if current.closedSet[neighbor.ID] {
			continue
		}

		tentativeGScore := current.gScore[node.ID] + edgeToUse.Distance
		tentativeTimeScore := current.timeScore[node.ID] + edgeToUse.TravelTime

		if currentGScore, exists := current.gScore[neighbor.ID]; !exists ||
			tentativeGScore < currentGScore {
			current.cameFrom[neighbor.ID] = node.ID
			current.gScore[neighbor.ID] = tentativeGScore
			current.timeScore[neighbor.ID] = tentativeTimeScore

			// _ Calculate heuristic based on optimization type
			var h float64
			switch opts.OptimizationType {
			case OptimizeShortest:
				h = heuristic(neighbor, target)
				current.fScore[neighbor.ID] = tentativeGScore + h
			case OptimizeFastest:
				// _ Estimate time based on straight-line distance at highway speed
				dist := heuristic(neighbor, target)
				h = (dist / 1609.34) / 65 * 3600 // _ Convert to seconds at 65 mph
				current.fScore[neighbor.ID] = tentativeTimeScore + h
			case OptimizePractical:
				// _ Balanced heuristic
				dist := heuristic(neighbor, target)
				timeH := (dist / 1609.34) / 55 * 3600 // _ 55 mph average
				h = dist*0.3 + timeH*0.7
				current.fScore[neighbor.ID] = tentativeGScore*0.3 + tentativeTimeScore*0.7 + h
			default: // OptimizeShortest
				h = heuristic(neighbor, target)
				current.fScore[neighbor.ID] = tentativeGScore + h
			}

			heap.Push(current.openSet, &item{
				node:     neighbor,
				priority: current.fScore[neighbor.ID],
			})
		}
	}

	return -1
}

func findReverseEdge(g *Graph, edge *Edge) *Edge {
	if toNode, exists := g.Nodes[edge.To.ID]; exists {
		for _, e := range toNode.Edges {
			if e.To.ID == edge.From.ID {
				return e
			}
		}
	}
	return nil
}

func reconstructBidirectionalPath(
	forwardCameFrom, backwardCameFrom map[int64]int64,
	meetingNode int64,
	nodes map[int64]*Node,
) []*Node {
	// _ Reconstruct forward path
	forwardPath := []*Node{}
	current := meetingNode
	for {
		forwardPath = append([]*Node{nodes[current]}, forwardPath...)
		if prev, exists := forwardCameFrom[current]; exists {
			current = prev
		} else {
			break
		}
	}

	// _ Reconstruct backward path
	backwardPath := []*Node{}
	current = meetingNode
	for {
		if prev, exists := backwardCameFrom[current]; exists {
			current = prev
			backwardPath = append(backwardPath, nodes[current])
		} else {
			break
		}
	}

	// _ Combine paths
	return append(forwardPath, backwardPath...)
}

// peek returns the top item without removing it
func (pq *priorityQueue) peek() *item {
	if pq.Len() > 0 {
		return (*pq)[0]
	}
	return nil
}
