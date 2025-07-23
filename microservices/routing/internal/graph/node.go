// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package graph

import "github.com/paulmach/orb"

// Node represents an intersection or waypoint in the road network
type Node struct {
	ID       int64
	Location orb.Point // [lon, lat]
	Edges    []*Edge   // Outgoing edges from this node
}

// Edge represents a road segment between two nodes
type Edge struct {
	ID           int64
	From         *Node
	To           *Node
	Distance     float64 // Distance in meters
	TravelTime   float64 // Time in seconds
	MaxHeight    float64 // Maximum height in meters (0 = no restriction)
	MaxWeight    float64 // Maximum weight in kg (0 = no restriction)
	TruckAllowed bool    // Whether trucks are allowed on this road
}

// Graph represents the road network
type Graph struct {
	Nodes map[int64]*Node
}

// Bounds represents a geographic bounding box
type Bounds struct {
	MinLat float64
	MinLon float64
	MaxLat float64
	MaxLon float64
}

// SimpleNode represents a basic node for storage operations
type SimpleNode struct {
	ID  int64
	Lat float64
	Lon float64
}

// SimpleEdge represents a basic edge for storage operations
type SimpleEdge struct {
	FromNodeID   int64
	ToNodeID     int64
	Distance     float64
	TravelTime   float64
	TruckAllowed bool
}

// NewGraph creates a new graph instance
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[int64]*Node),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) {
	g.Nodes[node.ID] = node
}

// AddEdge adds an edge between two nodes
func (g *Graph) AddEdge(edge *Edge) {
	if from, exists := g.Nodes[edge.From.ID]; exists {
		from.Edges = append(from.Edges, edge)
	}
}
