/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package graph

import (
	"context"
)

// PathFinder defines the interface for pathfinding algorithms
type PathFinder interface {
	FindPath(ctx context.Context, startID, endID int64, opts PathOptions) (*PathResult, error)
}

// OptimizationType specifies what to optimize for in routing
type OptimizationType int

const (
	OptimizeShortest  OptimizationType = iota // Minimize distance
	OptimizeFastest                           // Minimize travel time
	OptimizePractical                         // Balance time and distance
)

// PathOptions contains options for pathfinding
type PathOptions struct {
	MaxHeight        float64
	MaxWeight        float64
	MaxLength        float64 // Maximum vehicle length in meters
	MaxAxleLoad      float64 // Maximum weight per axle in kg
	TruckOnly        bool
	HazmatAllowed    bool             // Allow hazmat routes
	MaxSearchNodes   int              // Limit search space
	PreferHighways   bool             // Prefer highways over local roads
	AvoidTolls       bool             // Avoid toll roads
	Algorithm        AlgorithmType    // Which algorithm to use
	OptimizationType OptimizationType // What to optimize for
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
	Path             []*Node          `json:"-"`                   // Exclude from JSON
	PathNodes        []PathNode       `json:"path"`                // Simplified nodes for JSON
	Distance         float64          `json:"distance"`            // Total distance in meters
	TravelTime       float64          `json:"travel_time"`         // Total time in seconds
	Algorithm        string           `json:"algorithm"`           // Algorithm used
	SearchNodes      int              `json:"search_nodes"`        // Nodes explored
	ComputeTime      float64          `json:"compute_time"`        // Computation time in seconds
	OptimizationType OptimizationType `json:"optimization_type"`   // What was optimized
	TollCost         float64          `json:"toll_cost,omitempty"` // Estimated toll costs
	FuelCost         float64          `json:"fuel_cost,omitempty"` // Estimated fuel costs
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
