"use no memo";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { WorkflowNodeType } from "@/types/workflow";
import { faNetworkWired } from "@fortawesome/pro-regular-svg-icons";
import { useReactFlow } from "@xyflow/react";
import {
  Clock,
  GitBranch,
  Play,
  Repeat,
  Square,
  Trash2,
  Zap,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

const nodeIcons = {
  trigger: Zap,
  action: Play,
  condition: GitBranch,
  loop: Repeat,
  delay: Clock,
  end: Square,
};

const nodeColors = {
  trigger: "text-green-600",
  action: "text-blue-600",
  condition: "text-yellow-600",
  loop: "text-purple-600",
  delay: "text-orange-600",
  end: "text-red-600",
};

export default function NodesInUse() {
  const { getNodes, setNodes, deleteElements } = useReactFlow();
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);

  const nodes = getNodes() as WorkflowNodeType[];
  const selectedNode = nodes.find((n) => n.id === selectedNodeId);

  const handleUpdateNode = (
    nodeId: string,
    updates: Partial<WorkflowNodeType["data"]>,
  ) => {
    setNodes((nds) =>
      nds.map((node) =>
        node.id === nodeId
          ? { ...node, data: { ...node.data, ...updates } }
          : node,
      ),
    );
    toast.success("Node updated");
  };

  const handleDeleteNode = (nodeId: string) => {
    deleteElements({ nodes: [{ id: nodeId }] });
    setSelectedNodeId(null);
    toast.success("Node deleted");
  };

  if (nodes.length === 0) {
    return (
      <div className="flex h-[80vh] w-80 flex-col">
        <div className="flex flex-col items-center justify-center gap-1 p-4">
          <div className="flex size-12 items-center justify-center rounded-full bg-primary">
            <Icon
              icon={faNetworkWired}
              className="size-6 shrink-0 text-background"
            />
          </div>
          <span className="text-xl font-medium">Nodes in Use</span>
          <span className="text-sm text-muted-foreground">
            No nodes in this workflow yet
          </span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-[80vh] w-80 flex-col">
      <div className="flex flex-col items-center justify-center gap-1 p-4">
        <div className="flex size-12 items-center justify-center rounded-full bg-primary">
          <Icon
            icon={faNetworkWired}
            className="size-6 shrink-0 text-background"
          />
        </div>
        <span className="text-xl font-medium">Nodes in Use</span>
        <span className="text-sm text-muted-foreground">
          {nodes.length} {nodes.length === 1 ? "node" : "nodes"} in this
          workflow
        </span>
      </div>

      <ScrollArea className="flex-1 px-4">
        <div className="flex flex-col gap-2">
          {nodes.map((node) => {
            const IconComponent =
              nodeIcons[node.data.nodeType as keyof typeof nodeIcons] || Play;
            const colorClass =
              nodeColors[node.data.nodeType as keyof typeof nodeColors] ||
              "text-blue-600";
            const isSelected = selectedNodeId === node.id;

            return (
              <div key={node.id}>
                <button
                  className={`flex w-full flex-row items-center gap-2 rounded-md border px-2 py-2 text-left transition-colors ${
                    isSelected
                      ? "border-primary bg-primary/10"
                      : "border-border hover:bg-accent"
                  }`}
                  onClick={() => setSelectedNodeId(isSelected ? null : node.id)}
                >
                  <IconComponent className={`size-4 shrink-0 ${colorClass}`} />
                  <div className="flex flex-1 flex-col leading-tight">
                    <span className="text-sm font-medium">
                      {node.data.label}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {node.data.nodeType}
                    </span>
                  </div>
                </button>

                {isSelected && selectedNode && (
                  <NodeProperties
                    node={selectedNode}
                    onUpdate={(updates) => handleUpdateNode(node.id, updates)}
                    onDelete={() => handleDeleteNode(node.id)}
                  />
                )}
              </div>
            );
          })}
        </div>
      </ScrollArea>
    </div>
  );
}

function NodeProperties({
  node,
  onUpdate,
  onDelete,
}: {
  node: WorkflowNodeType;
  onUpdate: (updates: Partial<WorkflowNodeType["data"]>) => void;
  onDelete: () => void;
}) {
  const [label, setLabel] = useState(node.data.label);
  const [description, setDescription] = useState(
    node.data.config?.description || "",
  );
  const [delaySeconds, setDelaySeconds] = useState(
    node.data.config?.delaySeconds || 1,
  );

  const handleSave = () => {
    const baseConfig = {
      ...node.data.config,
      description,
    };

    // Add type-specific config
    if (node.data.nodeType === "delay") {
      baseConfig.delaySeconds = delaySeconds;
    }

    onUpdate({
      label,
      config: baseConfig,
    });
  };

  return (
    <div className="mt-2 flex flex-col gap-3 rounded-md border border-border bg-muted/50 p-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold">Node Properties</span>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 w-7 p-0 text-destructive hover:text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="size-4" />
        </Button>
      </div>

      <Separator />

      <div className="space-y-2">
        <Label htmlFor="node-label" className="text-xs">
          Label
        </Label>
        <Input
          id="node-label"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          onBlur={handleSave}
          className="h-8"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="node-description" className="text-xs">
          Description
        </Label>
        <Textarea
          id="node-description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          onBlur={handleSave}
          className="min-h-[60px] resize-none text-xs"
          placeholder="Add a description for this node..."
        />
      </div>

      {/* Delay node specific config */}
      {node.data.nodeType === "delay" && (
        <div className="space-y-2">
          <Label htmlFor="delay-seconds" className="text-xs">
            Delay (seconds)
          </Label>
          <Input
            id="delay-seconds"
            type="number"
            min="1"
            value={delaySeconds}
            onChange={(e) => setDelaySeconds(Number(e.target.value))}
            onBlur={handleSave}
            className="h-8"
          />
        </div>
      )}

      {/* Action node - show action type selector */}
      {node.data.nodeType === "action" && (
        <div className="space-y-2">
          <Label className="text-xs">Action Type</Label>
          <div className="rounded-md bg-background px-2 py-1.5 text-xs text-muted-foreground">
            {node.data.actionType || "Not configured"}
          </div>
          <p className="text-xs text-muted-foreground">
            Configure action type via the action configuration dialog
          </p>
        </div>
      )}

      {/* Condition node - show condition info */}
      {node.data.nodeType === "condition" && (
        <div className="space-y-2">
          <Label className="text-xs">Condition</Label>
          <div className="rounded-md bg-background px-2 py-1.5 text-xs text-muted-foreground">
            {node.data.config?.field ? (
              <div>
                {node.data.config.field} {node.data.config.operator}{" "}
                {node.data.config.value}
              </div>
            ) : (
              "Not configured"
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            Configure condition via the condition configuration dialog
          </p>
        </div>
      )}

      <Separator />

      <div className="space-y-2">
        <Label className="text-xs">Node Type</Label>
        <div className="rounded-md bg-background px-2 py-1.5 text-xs capitalize">
          {node.data.nodeType}
        </div>
      </div>

      <div className="space-y-2">
        <Label className="text-xs">Node ID</Label>
        <div className="rounded-md bg-background px-2 py-1.5 font-mono text-xs">
          {node.id}
        </div>
      </div>
    </div>
  );
}
