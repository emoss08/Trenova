package graph

import (
	"context"
)

// PathFinder defines the interface for pathfinding algorithms
type PathFinder interface {
	FindPath(ctx context.Context, startID, endID int64, opts PathOptions) (*PathResult, error)
}

// PathOptions contains options for pathfinding
type PathOptions struct {
	MaxHeight      float64
	MaxWeight      float64
	TruckOnly      bool
	MaxSearchNodes int           // Limit search space
	PreferHighways bool          // Prefer highways over local roads
	AvoidTolls     bool          // Avoid toll roads
	Algorithm      AlgorithmType // Which algorithm to use
}

// AlgorithmType specifies which pathfinding algorithm to use
type AlgorithmType int

const (
	AlgorithmAStar AlgorithmType = iota
	AlgorithmBidirectionalAStar
	AlgorithmDijkstra
)

// PathResult contains the result of a pathfinding operation
type PathResult struct {
	Path        []*Node    `json:"-"`            // Exclude from JSON
	PathNodes   []PathNode `json:"path"`         // Simplified nodes for JSON
	Distance    float64    `json:"distance"`     // Total distance in meters
	TravelTime  float64    `json:"travel_time"`  // Total time in seconds
	Algorithm   string     `json:"algorithm"`    // Algorithm used
	SearchNodes int        `json:"search_nodes"` // Nodes explored
	ComputeTime int64      `json:"compute_time"` // Computation time in ms
}

// PathNode is a simplified node representation for JSON serialization
type PathNode struct {
	ID       int64     `json:"id"`
	Location []float64 `json:"location"` // [lon, lat]
}

// RouteGeometry represents the geographic path for visualization
type RouteGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"` // [[lon, lat], ...]
}

// RouteVisualization contains route data for visualization
type RouteVisualization struct {
	*PathResult
	Geometry     RouteGeometry `json:"geometry"`
	Bounds       BoundingBox   `json:"bounds"`
	Instructions []Instruction `json:"instructions,omitempty"`
}

// BoundingBox represents the geographic bounds of a route
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
}

// Instruction represents a turn-by-turn instruction
type Instruction struct {
	Text     string  `json:"text"`
	Distance float64 `json:"distance"`
	Time     float64 `json:"time"`
	Type     string  `json:"type"`
}
