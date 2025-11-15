package workflowutils

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/pkg/pulid"
)

// NodeKeyMapper handles the translation between React Flow node keys and database IDs.
// This allows the frontend to use its own stable identifiers (nodeKeys) while maintaining
// proper foreign key relationships in the database.
type NodeKeyMapper struct {
	keyToID map[string]*pulid.ID
	idToKey map[string]string
}

// NewNodeKeyMapper creates a new mapper from a list of nodes.
// It builds bidirectional maps between nodeKey and database ID.
func NewNodeKeyMapper(nodes []*workflow.WorkflowNode) *NodeKeyMapper {
	mapper := &NodeKeyMapper{
		keyToID: make(map[string]*pulid.ID, len(nodes)),
		idToKey: make(map[string]string, len(nodes)),
	}

	for _, node := range nodes {
		mapper.keyToID[node.NodeKey] = &node.ID
		mapper.idToKey[node.ID.String()] = node.NodeKey
	}

	return mapper
}

// GetDatabaseID returns the database ID for a given nodeKey.
// Returns an error if the nodeKey doesn't exist in the mapping.
func (m *NodeKeyMapper) GetDatabaseID(nodeKey string) (*pulid.ID, error) {
	idPtr, exists := m.keyToID[nodeKey]
	if !exists {
		return nil, fmt.Errorf("node key not found: %s", nodeKey)
	}
	return idPtr, nil
}

// GetNodeKey returns the nodeKey for a given database ID.
// Returns an error if the database ID doesn't exist in the mapping.
func (m *NodeKeyMapper) GetNodeKey(databaseID pulid.ID) (string, error) {
	key, exists := m.idToKey[databaseID.String()]
	if !exists {
		return "", fmt.Errorf("database ID not found: %s", databaseID.String())
	}
	return key, nil
}

// TranslateEdgesToDatabaseIDs converts edge references from nodeKeys to database IDs.
// This is required before inserting edges into the database to satisfy foreign key constraints.
func (m *NodeKeyMapper) TranslateEdgesToDatabaseIDs(edges []*workflow.WorkflowEdge) error {
	for _, edge := range edges {
		// Translate source node reference
		sourceIDPtr, err := m.GetDatabaseID(edge.SourceNodeID.String())
		if err != nil {
			return fmt.Errorf("invalid source node reference: %w", err)
		}

		// Translate target node reference
		targetIDPtr, err := m.GetDatabaseID(edge.TargetNodeID.String())
		if err != nil {
			return fmt.Errorf("invalid target node reference: %w", err)
		}

		// Update edge with database IDs
		edge.SourceNodeID = *sourceIDPtr
		edge.TargetNodeID = *targetIDPtr
	}

	return nil
}

// TranslateEdgesToNodeKeys converts edge references from database IDs to nodeKeys.
// This is required when loading edges from the database to send to the frontend.
func (m *NodeKeyMapper) TranslateEdgesToNodeKeys(edges []*workflow.WorkflowEdge) error {
	for _, edge := range edges {
		// Translate source node reference
		sourceKey, err := m.GetNodeKey(edge.SourceNodeID)
		if err != nil {
			return fmt.Errorf("invalid source node ID: %w", err)
		}

		// Translate target node reference
		targetKey, err := m.GetNodeKey(edge.TargetNodeID)
		if err != nil {
			return fmt.Errorf("invalid target node ID: %w", err)
		}

		// Update edge with node keys (convert strings back to pulid for consistency)
		edge.SourceNodeID = pulid.ID(sourceKey)
		edge.TargetNodeID = pulid.ID(targetKey)
	}

	return nil
}
