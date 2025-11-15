import { Button } from "@/components/ui/button";
import { queries } from "@/lib/queries";
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
  type Node,
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

export function WorkflowBuilder({
  workflowId,
  versionId,
}: {
  workflowId?: string;
  versionId?: string;
}) {
  const queryClient = useQueryClient();
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);

  // Load workflow version if exists
  const { data: version, isLoading } = useQuery({
    ...queries.workflow.getVersion(workflowId!, versionId!),
    enabled: !!workflowId && !!versionId,
  });

  // Load nodes and edges from version
  useEffect(() => {
    if (version?.definition) {
      const def = version.definition;
      if (def.nodes && Array.isArray(def.nodes)) {
        setNodes(
          def.nodes.map((node: any) => ({
            id: node.id,
            type: node.type,
            position: node.position || { x: 0, y: 0 },
            data: {
              label: node.data?.label || node.type,
              nodeType: node.type,
              config: node.data?.config || {},
            },
          })),
        );
      }
      if (def.edges && Array.isArray(def.edges)) {
        setEdges(
          def.edges.map((edge: any) => ({
            id: edge.id,
            source: edge.source,
            target: edge.target,
            sourceHandle: edge.sourceHandle,
            targetHandle: edge.targetHandle,
          })),
        );
      }
    }
  }, [version, setNodes, setEdges]);

  const onConnect = useCallback(
    (params: Connection) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  );

  const onNodeClick = useCallback((_event: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedNode(null);
  }, []);

  const addNode = useCallback(
    (type: string) => {
      const newNode: Node = {
        id: `${type}-${Date.now()}`,
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
    },
    [setNodes],
  );

  const updateNodeData = useCallback(
    (nodeId: string, data: any) => {
      setNodes((nds) =>
        nds.map((node) =>
          node.id === nodeId
            ? { ...node, data: { ...node.data, ...data } }
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
    }
  }, [selectedNode, setNodes, setEdges]);

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!workflowId || !versionId) {
        throw new Error("Workflow ID and Version ID are required");
      }

      const definition = {
        nodes: nodes.map((node) => ({
          id: node.id,
          type: node.type,
          position: node.position,
          data: node.data,
        })),
        edges: edges.map((edge) => ({
          id: edge.id,
          source: edge.source,
          target: edge.target,
          sourceHandle: edge.sourceHandle,
          targetHandle: edge.targetHandle,
        })),
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
    <div className="relative h-full w-full">
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
