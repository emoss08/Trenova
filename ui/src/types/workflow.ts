import { type Node } from "@xyflow/react";

export type ActionConfigFormProps = {
  actionType: string;
  initialConfig: Record<string, any>;
  onSave: (config: Record<string, any>) => void;
  onCancel: () => void;
};
export type NodeData = {
  label: string;
  nodeType: string;
  config: Record<string, any>;
  actionType?: string;
};

export type WorkflowNodeType = Node<NodeData, string>;

export type VariableCategory = {
  label: string;
  description: string;
  variables: Variable[];
};

export type Variable = {
  value: string;
  label: string;
  description: string;
};
