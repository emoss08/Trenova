import { type Node } from "@xyflow/react";
// Define node data type
export type NodeData = {
  label: string;
  nodeType: string;
  config: Record<string, any>;
  actionType?: string;
};

// Define custom node type
export type WorkflowNodeType = Node<NodeData, string>;
