/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { queries } from "@/lib/queries";
import {
  type WorkflowDefinitionSchema,
  type WorkflowEdgeSchema,
  type WorkflowNodeSchema,
} from "@/lib/schemas/workflow-schema";
import { api } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Background,
  Controls,
  MiniMap,
  Panel,
  ReactFlow,
  addEdge,
  useEdgesState,
  useNodesState,
  type Connection,
  type Edge,
  type Node,
  type OnEdgesChange,
  type OnNodesChange,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Play, Save } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { NodeConfigPanel } from "./node-config-panel";
import { NodePalette } from "./node-palette";
import { WorkflowNode } from "./workflow-nodes";

const nodeTypes = {
  trigger: WorkflowNode,
  action: WorkflowNode,
  condition: WorkflowNode,
  loop: WorkflowNode,
  delay: WorkflowNode,
  end: WorkflowNode,
};

interface NodeData {
  label: string;
  nodeType: string;
  config: Record<string, any>;
  actionType?: string;
}

// Convert backend workflow node to React Flow node
function toReactFlowNode(node: WorkflowNodeSchema): Node<NodeData> {
  return {
    id: node.id,
    type: node.type,
    position: node.position,
    data: {
      label: node.label,
      nodeType: node.type,
      config: node.config || {},
      actionType: node.actionType,
    },
  };
}

// Convert React Flow node to backend workflow node
function toWorkflowNode(node: Node<NodeData>): WorkflowNodeSchema {
  return {
    id: node.id,
    type: node.type as any,
    label: node.data.label,
    config: node.data.config || {},
    position: node.position,
    actionType: node.data.actionType as any,
    data: node.data.config,
  };
}

// Convert backend workflow edge to React Flow edge
function toReactFlowEdge(edge: WorkflowEdgeSchema): Edge {
  return {
    id: edge.id,
    source: edge.source,
    target: edge.target,
    sourceHandle: edge.sourceHandle || undefined,
    targetHandle: edge.targetHandle || undefined,
    label: edge.label || undefined,
  };
}

// Convert React Flow edge to backend workflow edge
function toWorkflowEdge(edge: Edge): WorkflowEdgeSchema {
  return {
    id: edge.id,
    source: edge.source,
    target: edge.target,
    sourceHandle: edge.sourceHandle || null,
    targetHandle: edge.targetHandle || null,
    label: (edge.label as string) || null,
    condition: {},
  };
}

export function WorkflowBuilder({
  workflowId,
  versionId,
}: {
  workflowId?: string;
  versionId?: string;
}) {
  const queryClient = useQueryClient();
  const [nodes, setNodes, onNodesChange] = useNodesState<NodeData>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedNode, setSelectedNode] = useState<Node<NodeData> | null>(null);

  // Load workflow version if exists
  const { data: version, isLoading } = useQuery({
    ...queries.workflow.getVersion(workflowId!, versionId!),
    enabled: !!workflowId && !!versionId,
  });

  // Load nodes and edges from version
  useEffect(() => {
    if (version?.definition) {
      const def = version.definition as WorkflowDefinitionSchema;

      if (def.nodes && Array.isArray(def.nodes)) {
        setNodes(def.nodes.map(toReactFlowNode));
      }

      if (def.edges && Array.isArray(def.edges)) {
        setEdges(def.edges.map(toReactFlowEdge));
      }
    }
  }, [version, setNodes, setEdges]);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  );

  const onNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node<NodeData>) => {
      setSelectedNode(node);
    },
    [],
  );

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  const addNode = useCallback(
    (type: string) => {
      const timestamp = Date.now();
      const newNode: Node<NodeData> = {
        id: `node-${type}-${timestamp}`,
        type,
        position: {
          x: Math.random() * 400 + 100,
          y: Math.random() * 400 + 100,
        },
        data: {
          label: type.charAt(0).toUpperCase() + type.slice(1),
          nodeType: type,
          config: {},
        },
      };
      setNodes((nds) => [...nds, newNode]);
      toast.success(
        `${type.charAt(0).toUpperCase() + type.slice(1)} node added`,
      );
    },
    [setNodes],
  );

  const updateNodeData = useCallback(
    (nodeId: string, updates: Partial<NodeData>) => {
      setNodes((nds) =>
        nds.map((node) =>
          node.id === nodeId
            ? { ...node, data: { ...node.data, ...updates } }
            : node,
        ),
      );
    },
    [setNodes],
  );

  const deleteNode = useCallback(() => {
    if (selectedNode) {
      setNodes((nds) => nds.filter((node) => node.id !== selectedNode.id));
      setEdges((eds) =>
        eds.filter(
          (edge) =>
            edge.source !== selectedNode.id && edge.target !== selectedNode.id,
        ),
      );
      setSelectedNode(null);
      toast.success("Node deleted");
    }
  }, [selectedNode, setNodes, setEdges]);

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!workflowId || !versionId) {
        throw new Error("Workflow ID and Version ID are required");
      }

      const definition: WorkflowDefinitionSchema = {
        nodes: nodes.map(toWorkflowNode),
        edges: edges.map(toWorkflowEdge),
      };

      return api.workflows.saveDefinition(workflowId, versionId, {
        definition,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflow", workflowId] });
      toast.success("Workflow saved successfully");
    },
    onError: (error: Error) => {
      toast.error("Failed to save workflow", {
        description: error.message,
      });
    },
  });

  const publishMutation = useMutation({
    mutationFn: async () => {
      if (!workflowId || !versionId) {
        throw new Error("Workflow ID and Version ID are required");
      }
      // Save first, then publish
      await saveMutation.mutateAsync();
      return api.workflows.publishVersion(workflowId, versionId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workflow", workflowId] });
      toast.success("Workflow published successfully");
    },
    onError: (error: Error) => {
      toast.error("Failed to publish workflow", {
        description: error.message,
      });
    },
  });

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">Loading workflow...</div>
      </div>
    );
  }

  if (!workflowId) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-muted-foreground">
          Select or create a workflow to start building
        </div>
      </div>
    );
  }

  return (
    <div style={{ height: "100vh", width: "100vw" }}>
      <ReactFlow<NodeData>
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange as OnNodesChange<Node<NodeData>>}
        onEdgesChange={onEdgesChange as OnEdgesChange<Edge>}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        nodeTypes={nodeTypes}
        fitView
        className="bg-background"
      >
        <Background />
        <Controls />
        <MiniMap />

        <Panel position="top-left" className="space-y-2">
          <NodePalette onAddNode={addNode} />
        </Panel>

        <Panel position="top-right" className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => saveMutation.mutate()}
            disabled={saveMutation.isPending || !versionId}
          >
            <Save className="mr-2 size-4" />
            Save Draft
          </Button>
          <Button
            size="sm"
            onClick={() => publishMutation.mutate()}
            disabled={publishMutation.isPending || !versionId}
          >
            <Play className="mr-2 size-4" />
            Publish
          </Button>
        </Panel>
      </ReactFlow>

      <NodeConfigPanel
        node={selectedNode}
        onUpdate={updateNodeData}
        onDelete={deleteNode}
        onClose={() => setSelectedNode(null)}
      />
    </div>
  );
}
