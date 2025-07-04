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