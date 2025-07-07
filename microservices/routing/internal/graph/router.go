package graph

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Router provides high-level routing functionality
type Router struct {
	graph      *Graph
	algorithms map[AlgorithmType]PathFinder
}

// NewRouter creates a new router with the given graph
func NewRouter(graph *Graph) *Router {
	r := &Router{
		graph:      graph,
		algorithms: make(map[AlgorithmType]PathFinder),
	}

	// _ Register available algorithms
	r.algorithms[AlgorithmAStar] = &aStarPathFinder{graph: graph}
	r.algorithms[AlgorithmBidirectionalAStar] = &bidirectionalAStarPathFinder{graph: graph}

	return r
}

// FindRoute finds a route between two nodes with the given options
func (r *Router) FindRoute(ctx context.Context, startID, endID int64, opts PathOptions) (*PathResult, error) {
	// _ Set defaults
	if opts.MaxSearchNodes == 0 {
		opts.MaxSearchNodes = 100000 // Default limit
	}

	// _ Select algorithm based on distance heuristic
	if opts.Algorithm == 0 {
		// _ Auto-select algorithm
		start, ok1 := r.graph.Nodes[startID]
		end, ok2 := r.graph.Nodes[endID]
		if ok1 && ok2 {
			distance := heuristic(start, end)
			if distance > 100000 { // > 100km, use bidirectional
				opts.Algorithm = AlgorithmBidirectionalAStar
			} else {
				opts.Algorithm = AlgorithmAStar
			}
		}
	}

	// _ Get the pathfinder
	pathfinder, ok := r.algorithms[opts.Algorithm]
	if !ok {
		return nil, fmt.Errorf("unknown algorithm: %v", opts.Algorithm)
	}

	// _ Find the path with timeout
	return pathfinder.FindPath(ctx, startID, endID, opts)
}

// GetRouteVisualization returns a route with visualization data
func (r *Router) GetRouteVisualization(ctx context.Context, startID, endID int64, opts PathOptions) (*RouteVisualization, error) {
	// _ Find the route
	result, err := r.FindRoute(ctx, startID, endID, opts)
	if err != nil {
		return nil, err
	}

	// _ Build visualization
	viz := &RouteVisualization{
		PathResult: result,
		Geometry: RouteGeometry{
			Type:        "LineString",
			Coordinates: make([][]float64, 0, len(result.Path)),
		},
	}

	// _ Initialize bounds
	viz.Bounds = BoundingBox{
		MinLat: 90,
		MinLon: 180,
		MaxLat: -90,
		MaxLon: -180,
	}

	// _ Build coordinates and update bounds
	for _, node := range result.Path {
		lon, lat := node.Location[0], node.Location[1]
		viz.Geometry.Coordinates = append(viz.Geometry.Coordinates, []float64{lon, lat})

		// _ Update bounds
		viz.Bounds.MinLat = math.Min(viz.Bounds.MinLat, lat)
		viz.Bounds.MinLon = math.Min(viz.Bounds.MinLon, lon)
		viz.Bounds.MaxLat = math.Max(viz.Bounds.MaxLat, lat)
		viz.Bounds.MaxLon = math.Max(viz.Bounds.MaxLon, lon)
	}

	// _ Add some padding to bounds
	latPadding := (viz.Bounds.MaxLat - viz.Bounds.MinLat) * 0.1
	lonPadding := (viz.Bounds.MaxLon - viz.Bounds.MinLon) * 0.1
	viz.Bounds.MinLat -= latPadding
	viz.Bounds.MaxLat += latPadding
	viz.Bounds.MinLon -= lonPadding
	viz.Bounds.MaxLon += lonPadding

	return viz, nil
}

// aStarPathFinder implements PathFinder using A* algorithm
type aStarPathFinder struct {
	graph *Graph
}

func (a *aStarPathFinder) FindPath(ctx context.Context, startID, endID int64, opts PathOptions) (*PathResult, error) {
	start := time.Now()

	// _ Create a context-aware version of A*
	type result struct {
		path *PathResult
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		path, err := a.graph.AStar(startID, endID, opts)
		resultChan <- result{path, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ErrTimeout
	case res := <-resultChan:
		if res.err != nil {
			return nil, res.err
		}
		res.path.Algorithm = "A*"
		res.path.ComputeTime = time.Since(start).Seconds()
		return res.path, nil
	}
}

// bidirectionalAStarPathFinder implements PathFinder using Bidirectional A*
type bidirectionalAStarPathFinder struct {
	graph *Graph
}

func (b *bidirectionalAStarPathFinder) FindPath(ctx context.Context, startID, endID int64, opts PathOptions) (*PathResult, error) {
	start := time.Now()

	// _ Create a context-aware version
	type result struct {
		path *PathResult
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		path, err := b.graph.BidirectionalAStar(startID, endID, opts)
		resultChan <- result{path, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ErrTimeout
	case res := <-resultChan:
		if res.err != nil {
			return nil, res.err
		}
		res.path.Algorithm = "Bidirectional A*"
		res.path.ComputeTime = time.Since(start).Seconds()
		return res.path, nil
	}
}
