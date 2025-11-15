import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { queries } from "@/lib/queries";
import {
  type WorkflowDefinitionSchema,
  type WorkflowEdgeSchema,
  type WorkflowNodeSchema,
} from "@/lib/schemas/workflow-schema";
import { api } from "@/services/api";
import { WorkflowNodeType } from "@/types/workflow";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  addEdge,
  Background,
  Controls,
  MiniMap,
  Node,
  NodeMouseHandler,
  Panel,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Edge,
  type OnConnect,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { Play, Save } from "lucide-react";
import React, { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { WorkflowNode } from "./workflow-nodes/workflow-nodes";

const nodeTypes = {
  trigger: WorkflowNode,
  action: WorkflowNode,
  condition: WorkflowNode,
  loop: WorkflowNode,
  delay: WorkflowNode,
  end: WorkflowNode,
};

function toReactFlowNode(node: WorkflowNodeSchema): WorkflowNodeType {
  return {
    id: node.nodeKey, // Use nodeKey as React Flow ID
    type: node.nodeType, // Use nodeType as React Flow type
    position: {
      x: node.positionX,
      y: node.positionY,
    },
    data: {
      label: node.label,
      nodeType: node.nodeType,
      config: node.config || {},
      actionType: node.actionType,
    },
  };
}

function toWorkflowNode(node: WorkflowNodeType): WorkflowNodeSchema {
  return {
    nodeKey: node.id, // Use React Flow ID as nodeKey
    nodeType: node.type as any, // Use React Flow type as nodeType
    label: node.data.label,
    description: node.data.config?.description || undefined,
    config: node.data.config || {},
    positionX: node.position.x,
    positionY: node.position.y,
    actionType: node.data.actionType as any,
  };
}

function toReactFlowEdge(edge: WorkflowEdgeSchema): Edge {
  return {
    id: edge.id || `${edge.sourceNodeId}-${edge.targetNodeId}`,
    source: edge.sourceNodeId, // Backend node ID becomes React Flow source
    target: edge.targetNodeId, // Backend node ID becomes React Flow target
    sourceHandle: edge.sourceHandle || undefined,
    targetHandle: edge.targetHandle || undefined,
    label: edge.label,
  };
}

function toWorkflowEdge(edge: Edge): WorkflowEdgeSchema {
  return {
    sourceNodeId: edge.source, // React Flow source becomes backend sourceNodeId
    targetNodeId: edge.target, // React Flow target becomes backend targetNodeId
    sourceHandle: edge.sourceHandle || undefined,
    targetHandle: edge.targetHandle || undefined,
    label: (edge.label as string) || undefined,
    condition: {},
  };
}
export default function WorkflowContent({
  workflowId,
  versionId,
}: {
  workflowId: string;
  versionId: string | undefined;
}) {
  const { theme } = useTheme();
  const queryClient = useQueryClient();
  const [nodes, setNodes, onNodesChange] = useNodesState([] as Node[]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([] as Edge[]);
  const [selectedNode, setSelectedNode] = useState<WorkflowNodeType | null>(
    null,
  );
  const [activeVersionId, setActiveVersionId] = useState<string | undefined>(
    versionId,
  );

  // Create initial version mutation
  const createInitialVersionMutation = useMutation({
    mutationFn: async () => {
      if (!workflowId) {
        throw new Error("Workflow ID is required");
      }

      return api.workflows.createVersion(workflowId, {
        versionName: "v1",
        changelog: "Initial version",
        workflowDefinition: { nodes: [], edges: [] },
      });
    },
    onSuccess: (newVersion) => {
      setActiveVersionId(newVersion.id);
      queryClient.invalidateQueries({ queryKey: ["workflow", workflowId] });
      toast.success("Initial version created");
    },
    onError: (error: Error) => {
      toast.error("Failed to create initial version", {
        description: error.message,
      });
    },
  });

  // Load workflow version with definition (only if versionId exists)
  const { data: version, isLoading } = useQuery({
    ...queries.workflow.getVersion(workflowId, activeVersionId!),
    enabled: !!activeVersionId && !!workflowId,
  });

  // Load nodes and edges from version (nodes/edges come from backend relationships)
  useEffect(() => {
    if (version) {
      // Backend returns nodes and edges as top-level arrays via relationships
      if (version.nodes && Array.isArray(version.nodes)) {
        const reactFlowNodes = version.nodes.map(toReactFlowNode);
        setNodes(reactFlowNodes);
      }

      if (version.edges && Array.isArray(version.edges)) {
        const reactFlowEdges = version.edges.map(toReactFlowEdge);
        setEdges(reactFlowEdges);
      }
    }
  }, [version, setNodes, setEdges]);

  const onConnect: OnConnect = useCallback(
    (params) => setEdges((eds) => addEdge({ ...params, animated: true }, eds)),
    [setEdges],
  );

  const onNodeClick = useCallback<NodeMouseHandler>(
    (_, node) => {
      setSelectedNode(node as WorkflowNodeType);
    },
    [setSelectedNode],
  );

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!workflowId || !activeVersionId) {
        throw new Error("Workflow ID and Version ID are required");
      }

      const definition: WorkflowDefinitionSchema = {
        nodes: nodes.map((node) => toWorkflowNode(node as WorkflowNodeType)),
        edges: edges.map((edge) => toWorkflowEdge(edge as Edge)),
      };

      return api.workflows.saveDefinition(workflowId, activeVersionId, {
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
      if (!workflowId || !activeVersionId) {
        throw new Error("Workflow ID and Version ID are required");
      }
      await saveMutation.mutateAsync();
      return api.workflows.publishVersion(workflowId, activeVersionId);
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

  if (isLoading || createInitialVersionMutation.isPending) {
    return (
      <FlowContainer>
        <div className="flex h-full items-center justify-center">
          <div className="text-muted-foreground">
            {createInitialVersionMutation.isPending
              ? "Creating initial version..."
              : "Loading workflow..."}
          </div>
        </div>
      </FlowContainer>
    );
  }

  if (!workflowId) {
    return (
      <FlowContainer>
        <div className="flex h-full items-center justify-center">
          <div className="text-muted-foreground">
            Select or create a workflow to start building
          </div>
        </div>
      </FlowContainer>
    );
  }

  // Show create version button if no version exists
  if (!activeVersionId) {
    return (
      <FlowContainer>
        <div className="flex h-full flex-col items-center justify-center gap-4">
          <div className="text-center">
            <h3 className="text-lg font-semibold">No Workflow Version</h3>
            <p className="text-sm text-muted-foreground">
              Create an initial version to start building your workflow
            </p>
          </div>
          <Button
            onClick={() => createInitialVersionMutation.mutate()}
            disabled={createInitialVersionMutation.isPending}
          >
            <Play className="mr-2 size-4" />
            Create Initial Version
          </Button>
        </div>
      </FlowContainer>
    );
  }

  return (
    <FlowContainer>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onConnect={onConnect}
        onNodeClick={onNodeClick}
        onPaneClick={onPaneClick}
        nodeTypes={nodeTypes}
        fitView
        colorMode={theme === "dark" ? "dark" : "light"}
        className="bg-background"
      >
        <Background />
        <Controls />
        <MiniMap />
        <Panel position="top-right" className="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={() => saveMutation.mutate()}
            disabled={saveMutation.isPending || !activeVersionId}
          >
            <Save className="mr-2 size-4" />
            Save Draft
          </Button>
          <Button
            size="sm"
            onClick={() => publishMutation.mutate()}
            disabled={publishMutation.isPending || !activeVersionId}
          >
            <Play className="mr-2 size-4" />
            Publish
          </Button>
        </Panel>
      </ReactFlow>
    </FlowContainer>
  );
}

function FlowContainer({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative h-[80vh] w-full overflow-hidden rounded-lg border border-border">
      {children}
    </div>
  );
}
